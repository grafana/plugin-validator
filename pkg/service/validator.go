package service

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes"
	"github.com/grafana/plugin-validator/pkg/archivetool"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/repotool"
	"github.com/grafana/plugin-validator/pkg/runner"
	"github.com/grafana/plugin-validator/pkg/utils"
)

type Params struct {
	PluginURL        string
	SourceCodeUri    string
	Checksum         string
	Analyzer         string
	AnalyzerSeverity string
	Config           *runner.Config
}

type Result struct {
	Diagnostics   analysis.Diagnostics
	PluginID      string
	PluginVersion string
}

func ValidatePlugin(params Params) (Result, error) {
	// read archive file into bytes
	b, err := archivetool.ReadArchive(params.PluginURL)
	if err != nil {
		err = fmt.Errorf("couldn't read plugin archive: %w", err)
		logme.Errorln(err)
		return Result{}, err
	}

	// write archive to a temp file
	tmpZip, err := os.CreateTemp("", "plugin-archive")
	if err != nil {
		err = fmt.Errorf("couldn't create temporary file: %w", err)
		logme.Errorln(err)
		return Result{}, err
	}
	defer os.Remove(tmpZip.Name())

	if _, err := tmpZip.Write(b); err != nil {
		err = fmt.Errorf("couldn't write temporary file: %w", err)
		logme.Errorln(err)
		return Result{}, err
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
		err = fmt.Errorf("couldn't extract plugin archive: %w", err)
		logme.Errorln(err)
		return Result{}, err
	}
	defer archiveCleanup()

	sourceCodeDir, sourceCodeDirCleanup, err := getSourceCodeDir(params.SourceCodeUri)
	if err != nil {
		// if source code is not provided, we don't fail the validation
		logme.Errorln(fmt.Errorf("couldn't get source code: %w", err))
	}
	if sourceCodeDirCleanup != nil {
		defer sourceCodeDirCleanup()
	}

	analyzers := passes.Analyzers
	severity := analysis.Severity("")

	if params.Analyzer != "" {
		for _, a := range analyzers {
			if a.Name == params.Analyzer {
				analyzers = []*analysis.Analyzer{a}

				break
			}
		}
		if params.AnalyzerSeverity != "" {
			severity = analysis.Severity(params.AnalyzerSeverity)
		}
	}

	if params.Config == nil {
		params.Config = &runner.Config{
			Global: runner.GlobalConfig{
				Enabled: true,
			},
		}
	}

	diags, err := runner.Check(
		analyzers,
		analysis.CheckParams{
			ArchiveFile:           tmpZip.Name(),
			ArchiveDir:            archiveDir,
			SourceCodeDir:         sourceCodeDir,
			SourceCodeReference:   params.SourceCodeUri,
			Checksum:              params.Checksum,
			ArchiveCalculatedMD5:  fmt.Sprintf("%x", md5hash),
			ArchiveCalculatedSHA1: fmt.Sprintf("%x", sha1hash),
		},
		*params.Config,
		severity,
	)
	if err != nil {
		// we don't exit on error. we want to still report the diagnostics
		logme.DebugFln("check failed: %v", err)
	}

	metadata, err := utils.GetPluginMetadata(archiveDir)
	if err != nil {
		archiveDiag := analysis.Diagnostic{
			Name:     "zip-invalid",
			Severity: analysis.Error,
			Title:    "Plugin archive is improperly structured",
			Detail:   "It is possible your plugin archive structure is incorrect. Please see https://grafana.com/developers/plugin-tools/publish-a-plugin/package-a-plugin for more information on how to package a plugin.",
		}
		diags["archive"] = append(diags["archive"], archiveDiag)
	}

	return Result{
		Diagnostics:   diags,
		PluginID:      metadata.ID,
		PluginVersion: metadata.Info.Version,
	}, nil
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
