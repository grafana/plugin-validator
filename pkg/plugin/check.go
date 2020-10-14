package plugin

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/grafana"
)

// checkContext contains useful paths and data available to checker.
type checkContext struct {
	RootDir string
	DistDir string
	SrcDir  string

	MetadataPath string

	Readme   []byte
	Metadata []byte
}

type checkSeverity string

const (
	checkSeverityError   checkSeverity = "error"
	checkSeverityWarning checkSeverity = "warning"
)

type checker interface {
	check(ctx *checkContext) ([]ValidationComment, error)
}

// Ref describes a plugin version on GitHub.
type Ref struct {
	Username string `json:"username"`
	Repo     string `json:"repo"`
	Ref      string `json:"ref"`
}

// ValidationComment contains a comment returned by one of the checkers.
type ValidationComment struct {
	Severity checkSeverity `json:"level"`
	Message  string        `json:"message"`
	Details  string        `json:"details"`
}

// ErrPluginNotFound is returned whenever a plugin could be found for a given ref.
var ErrPluginNotFound = errors.New("plugin not found")

// Check executes a number of checks to validate a plugin.
func Check(url string, schemaPath string, client *grafana.Client) (json.RawMessage, []ValidationComment, error) {
	ref, err := parseRef(url)
	if err != nil {
		return nil, nil, err
	}

	archiveURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/zipball/%s", ref.Username, ref.Repo, ref.Ref)

	resp, err := http.Get(archiveURL)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return nil, nil, ErrPluginNotFound
		}

		return nil, nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	// Extract the ZIP archive in a temporary directory.
	rootDir, cleanup, err := extractPlugin(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	defer cleanup()

	var (
		distDir = filepath.Join(rootDir, "dist")
		srcDir  = filepath.Join(rootDir, "src")
	)

	// TODO: If there's no plugin.json or README, several checks will fail.
	// Ideally, each checker would declare checkers it depends on, and only run
	// if those checkers ran successfully.
	var fatalErrs []ValidationComment

	readmePath := filepath.Join(distDir, "README.md")
	exists, err := fileExists(readmePath)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		fatalErrs = append(fatalErrs, ValidationComment{
			Severity: "error",
			Message:  "Missing README",
			Details:  "Plugins require a `README.md` file, but we couldn't find one. The README should provide instructions to the users on how to use the plugin.",
		})
	}

	metadataPath := filepath.Join(distDir, "plugin.json")
	exists, err = fileExists(metadataPath)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		fatalErrs = append(fatalErrs, ValidationComment{
			Severity: "error",
			Message:  "Missing metadata",
			Details:  "Plugins require a `plugin.json` file, but we couldn't find one. For more information, refer to [plugin.json](https://grafana.com/docs/grafana/latest/developers/plugins/metadata/).",
		})
	}

	if len(fatalErrs) > 0 {
		return nil, fatalErrs, nil
	}

	metadata, err := ioutil.ReadFile(metadataPath)
	if err != nil {
		return nil, nil, err
	}

	readme, err := ioutil.ReadFile(readmePath)
	if err != nil {
		return nil, nil, err
	}

	ctx := &checkContext{
		RootDir: rootDir,
		DistDir: distDir,
		SrcDir:  srcDir,

		MetadataPath: metadataPath,

		Readme:   readme,
		Metadata: metadata,
	}

	username := usernameFromMetadata(metadata)

	checkers := []checker{
		&distExistsChecker{},
		&orgExistsChecker{username: username, client: client},
		&pluginIDFormatChecker{},
		&pluginNameChecker{},
		&pluginIDHasTypeSuffixChecker{},
		&jsonSchemaChecker{schema: schemaPath},
		&linkChecker{},
		&pluginPlatformChecker{},
		&screenshotChecker{},
		&logosExistChecker{},
		&largeFileChecker{},
		&developerJargonChecker{},
		&templateReadmeChecker{},
	}

	errs := []ValidationComment{}

	// Check and collect all errors.
	for _, checker := range checkers {
		newerrs, err := checker.check(ctx)
		if err != nil {
			return nil, nil, err
		}
		errs = append(errs, newerrs...)
	}

	return json.RawMessage(metadata), errs, nil
}

// usernameFromMetadata returns the first part of the plugin ID.
func usernameFromMetadata(metadata []byte) string {
	var meta struct {
		ID string `json:"id"`
	}
	json.Unmarshal(metadata, &meta)

	fields := strings.Split(meta.ID, "-")

	if len(fields) > 0 {
		return fields[0]
	}

	return ""
}

func extractPlugin(body io.Reader) (string, func(), error) {
	// Create a file for the zipball.
	zipball, err := ioutil.TempFile("", "")
	if err != nil {
		return "", nil, err
	}
	defer zipball.Close()
	defer os.Remove(zipball.Name())

	if _, err := io.Copy(zipball, body); err != nil {
		return "", nil, err
	}

	// Create a directory where we'll extract the archive.
	output, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, err
	}

	cleanup := func() {
		os.RemoveAll(output)
	}

	if _, err := unzip(zipball.Name(), output); err != nil {
		cleanup()
		return "", nil, err
	}

	infos, err := ioutil.ReadDir(output)
	if err != nil {
		cleanup()
		return "", nil, err
	}

	if len(infos) != 1 {
		cleanup()
		return "", nil, fmt.Errorf("unzip: expected 1 directory but got %d", len(infos))
	}

	pluginRoot := filepath.Join(output, infos[0].Name())

	return pluginRoot, cleanup, nil
}

func unzip(src string, dest string) ([]string, error) {
	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
