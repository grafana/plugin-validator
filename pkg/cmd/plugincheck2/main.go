package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/fatih/color"
	"gopkg.in/yaml.v2"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes"
	"github.com/grafana/plugin-validator/pkg/archivetool"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/repotool"
	"github.com/grafana/plugin-validator/pkg/runner"
)

type FormattedOutput struct {
	ID          string                           `json:"id"`
	Version     string                           `json:"version"`
	Diagnostics map[string][]analysis.Diagnostic `json:"plugin-validator"`
}

func main() {
	var (
		strictFlag = flag.Bool(
			"strict",
			false,
			"If set, plugincheck returns non-zero exit code for warnings",
		)
		configFlag    = flag.String("config", "", "Path to configuration file")
		sourceCodeUri = flag.String(
			"sourceCodeUri",
			"",
			"URL to the source code of the plugin. If set, the source code will be downloaded and analyzed. This can be a ZIP file or an URL to git repository",
		)
		checksum = flag.String(
			"checksum",
			"",
			"checksum of the plugin archive. MD5, SHA1 or a string with the the hash or an url to a file with the hash",
		)
		analyzer = flag.String(
			"analyzer",
			"",
			"Run a specific analyzer",
		)
		analyzerSeverity = flag.String(
			"analyzerSeverity",
			"",
			"Set severity of the analyzer. Only works in combination with -analyzer",
		)
	)

	flag.Parse()

	logme.Debugln("Initializing...")
	logme.Debugln("strict mode: ", *strictFlag)
	logme.Debugln("config file: ", *configFlag)
	logme.Debugln("source code: ", *sourceCodeUri)
	logme.Debugln("archive file: ", flag.Arg(0))
	logme.Debugln("checksum: ", *checksum)
	logme.Debugln("analyzer: ", *analyzer)
	logme.Debugln("analyzerSeverity: ", *analyzerSeverity)

	cfg, err := readConfigFile(*configFlag)
	if err != nil {
		logme.Errorln(fmt.Errorf("couldn't read configuration: %w", err))
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "missing plugin url")
		os.Exit(1)
	}

	pluginURL := flag.Args()[0]

	// read archive file into bytes
	b, err := archivetool.ReadArchive(pluginURL)
	if err != nil {
		logme.Errorln(fmt.Errorf("couldn't fetch plugin archive: %w", err))
		os.Exit(1)
	}

	// write archive to a temp file
	tmpZip, err := os.CreateTemp("", "plugin-archive")
	if err != nil {
		logme.Errorln(fmt.Errorf("couldn't create temporary file: %w", err))
		os.Exit(1)
	}
	defer os.Remove(tmpZip.Name())

	if _, err := tmpZip.Write(b); err != nil {
		logme.Errorln(fmt.Errorf("couldn't write temporary file: %w", err))
		os.Exit(1)
	}

	logme.Debugln(fmt.Sprintf("Archive copied to tmp file: %s", tmpZip.Name()))

	md5hasher := md5.New()
	md5hasher.Write(b)
	md5hash := md5hasher.Sum(nil)

	sha1hasher := sha1.New()
	sha1hasher.Write(b)
	sha1hash := sha1hasher.Sum(nil)

	logme.Debugln(fmt.Sprintf("ArchiveCalculatedMD5: %x", md5hash))
	logme.Debugln(fmt.Sprintf("ArchiveCalculatedSHA1: %x", sha1hash))

	// Extract the ZIP archive in a temporary directory.
	archiveDir, archiveCleanup, err := archivetool.ExtractPlugin(bytes.NewReader(b))
	if err != nil {
		logme.Errorln(fmt.Errorf("couldn't extract plugin archive: %w", err))
		os.Exit(1)
	}
	defer archiveCleanup()

	sourceCodeDir, sourceCodeDirCleanup, err := getSourceCodeDir(*sourceCodeUri)
	if err != nil {
		// if source code is not provided, we don't fail the validation
		logme.Errorln(fmt.Errorf("couldn't get source code: %w", err))
	}
	if sourceCodeDirCleanup != nil {
		defer sourceCodeDirCleanup()
	}

	analyzers := passes.Analyzers
	severity := analysis.Severity("")

	if *analyzer != "" {
		for _, a := range analyzers {
			if a.Name == *analyzer {
				analyzers = []*analysis.Analyzer{a}

				break
			}
		}
		if *analyzerSeverity != "" {
			severity = analysis.Severity(*analyzerSeverity)
		}
	}

	diags, err := runner.Check(
		analyzers,
		analysis.CheckParams{
			ArchiveFile:           tmpZip.Name(),
			ArchiveDir:            archiveDir,
			SourceCodeDir:         sourceCodeDir,
			SourceCodeReference:   *sourceCodeUri,
			Checksum:              *checksum,
			ArchiveCalculatedMD5:  fmt.Sprintf("%x", md5hash),
			ArchiveCalculatedSHA1: fmt.Sprintf("%x", sha1hash),
		},
		cfg,
		severity,
	)
	if err != nil {
		// we don't exit on error. we want to still report the diagnostics
		logme.DebugFln("check failed: %v", err)
	}

	var exitCode int

	// JSON output
	if cfg.Global.JSONOutput {
		pluginID, pluginVersion, err := GetIDAndVersion(archiveDir)
		if err != nil {
			pluginID, pluginVersion = GetIDAndVersionFallBack(archiveDir)
			archiveDiag := analysis.Diagnostic{
				Name:     "zip-invalid",
				Severity: analysis.Error,
				Title:    "Plugin archive is improperly structured",
				Detail:   "It is possible your plugin archive structure is incorrect. Please see https://grafana.com/developers/plugin-tools/publish-a-plugin/package-a-plugin for more information on how to package a plugin.",
			}
			diags["archive"] = append(diags["archive"], archiveDiag)
		}
		allData := FormattedOutput{
			ID:          pluginID,
			Version:     pluginVersion,
			Diagnostics: diags,
		}
		output, _ := json.MarshalIndent(allData, "", "  ")
		fmt.Fprintln(os.Stdout, string(output))
		for name := range diags {
			for _, d := range diags[name] {
				switch d.Severity {
				case analysis.Error:
					exitCode = 1
				case analysis.Warning:
					if *strictFlag {
						exitCode = 1
					}
				}
			}
		}
		os.Exit(exitCode)
	}

	// regular CLI output
	for name := range diags {
		for _, d := range diags[name] {
			var buf bytes.Buffer
			switch d.Severity {
			case analysis.Error:
				buf.WriteString(color.RedString("error: "))
				exitCode = 1
			case analysis.Warning:
				buf.WriteString(color.YellowString("warning: "))
				if *strictFlag {
					exitCode = 1
				}
			case analysis.Recommendation:
				buf.WriteString(color.CyanString("recommendation: "))
			case analysis.OK:
				buf.WriteString(color.GreenString("ok: "))
			case analysis.SuspectedProblem:
				buf.WriteString(color.YellowString("suspected: "))
			}

			if d.Context != "" {
				buf.WriteString(d.Context + ": ")
			}

			buf.WriteString(d.Title)
			if len(d.Detail) > 0 {
				buf.WriteString("\n" + color.BlueString("detail: "))
				buf.WriteString(d.Detail)
			}
			fmt.Fprintln(os.Stderr, buf.String())
		}
	}

	logme.DebugFln("exit code: %d", exitCode)
	os.Exit(exitCode)
}

func readConfigFile(path string) (runner.Config, error) {

	// provide a default config if no config file is provided
	if path == "" {
		return runner.Config{
			Global: runner.GlobalConfig{
				Enabled: true,
			},
		}, nil
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return runner.Config{}, err
	}

	var config runner.Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return runner.Config{}, err
	}

	return config, nil
}

func getSourceCodeDirSubDir(sourceCodePath string) string {
	// check if there's a package.json in the source code directory
	// if so return the source code directory as is
	if _, err := os.Stat(filepath.Join(sourceCodePath, "package.json")); err == nil {
		return sourceCodePath
	}

	// use double start to find the first ocurrance of package.json
	possiblePath, err := doublestar.FilepathGlob(sourceCodePath + "/**/package.json")
	if err != nil {
		return sourceCodePath
	}
	if len(possiblePath) == 0 {
		return sourceCodePath
	}
	logme.DebugFln(
		"Detected sourcecode inside a subdir: %v. Returning %s",
		possiblePath,
		filepath.Dir(possiblePath[0]),
	)
	// possiblePath points to a file, return the dir
	return filepath.Dir(possiblePath[0])
}

func getSourceCodeDir(sourceCodeUri string) (string, func(), error) {
	// If source code URI is not provided, return immediately with an empty string
	// otherwise we will get an error when trying to extract the source code archive
	if sourceCodeUri == "" {
		return "", func() {}, nil
	}

	// file:// protocol for local directories
	if strings.HasPrefix(sourceCodeUri, "file://") {
		sourceCodeDir := strings.TrimPrefix(sourceCodeUri, "file://")
		if _, err := os.Stat(sourceCodeDir); err != nil {
			return "", nil, err
		}
		return sourceCodeDir, func() {}, nil
	}

	if repotool.IsSupportedGitUrl(sourceCodeUri) {
		extractedGitRepo, sourceCodeCleanUp, err := repotool.GitUrlToLocalPath(sourceCodeUri)
		if err != nil {
			return "", sourceCodeCleanUp, err
		}
		return extractedGitRepo, sourceCodeCleanUp, nil
	}

	// assume is an archive url
	extractedDir, sourceCodeCleanUp, err := archivetool.ArchiveToLocalPath(sourceCodeUri)
	if err != nil {
		return "", sourceCodeCleanUp, fmt.Errorf(
			"couldn't extract source code archive: %s. %w",
			sourceCodeUri,
			err,
		)
	}
	// some submissions from zip have their source code in a subdirectory
	// of the extracted archive
	extractedDir = getSourceCodeDirSubDir(extractedDir)
	return extractedDir, sourceCodeCleanUp, nil

}
