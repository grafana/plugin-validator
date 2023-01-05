package archivetool

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func ArchiveToLocalPath(uri string) (string, func(), error) {
	b, err := ReadArchive(uri)
	if err != nil {
		return "", nil, err
	}

	// Extract the ZIP archive in a temporary directory.
	archiveDir, cleanup, err := ExtractPlugin(bytes.NewReader(b))
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return "", nil, err
	}
	return archiveDir, cleanup, nil
}

// PluginArchiveToTempDir takes a uri to a plugin archive, downloads the archive and extract it to a temporary directory.
// Extract it and return the path to the extracted directory where the plugin dist is located.
// A cleanup function is returned that should be called when the plugin is no longer needed.
func PluginArchiveToTempDir(uri string) (string, func(), error) {
	archivePath, archiveCleanup, err := ArchiveToLocalPath(uri)
	if err != nil {
		return "", nil, err
	}

	defer func() {
		if err != nil && archiveCleanup != nil {
			archiveCleanup()
		}
	}()

	// get first folder inside archivepath
	files, err := os.ReadDir(archivePath)
	if err != nil {
		return "", nil, err
	}
	if len(files) == 0 {
		err = errors.New("no files in archive")
		return "", nil, err
	}
	archivePath = filepath.Join(archivePath, files[0].Name())

	// validate is a dir
	fileInfo, err := os.Stat(archivePath)
	if err != nil {
		return "", nil, err
	}

	if !fileInfo.IsDir() {
		err = errors.New("no files in archive")
		return "", nil, err
	}

	return archivePath, archiveCleanup, nil
}

// ReadArchive reads an archive from a URL or a local file
func ReadArchive(archiveURL string) ([]byte, error) {
	if strings.HasPrefix(archiveURL, "https://") || strings.HasPrefix(archiveURL, "http://") {
		resp, err := http.Get(archiveURL)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			if resp.StatusCode == http.StatusNotFound {
				return nil, errors.New("plugin not found")
			}
			return nil, fmt.Errorf("unexpected status: %s", resp.Status)
		}

		return ioutil.ReadAll(resp.Body)
	}

	return ioutil.ReadFile(archiveURL)
}

func ExtractPlugin(body io.Reader) (string, func(), error) {
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

	if _, err := Unzip(zipball.Name(), output); err != nil {
		cleanup()
		return "", nil, err
	}

	return output, cleanup, nil
}

func Unzip(src string, dest string) ([]string, error) {
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
			if err = os.MkdirAll(fpath, os.ModePerm); err != nil {
				return nil, err
			}
			continue
		}

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
