package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes"
	"github.com/grafana/plugin-validator/pkg/archivetool"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/repotool"
	"github.com/grafana/plugin-validator/pkg/runner"
	yaml "gopkg.in/yaml.v2"
)

type FormattedOutput struct {
	ID          string                           `json:"id"`
	Version     string                           `json:"version"`
	Diagnostics map[string][]analysis.Diagnostic `json:"plugin-validator"`
}

func main() {
	var (
		strictFlag    = flag.Bool("strict", false, "If set, plugincheck returns non-zero exit code for warnings")
		configFlag    = flag.String("config", "", "Path to configuration file")
		sourceCodeUri = flag.String("sourceCodeUri", "", "URL to the source code of the plugin. If set, the source code will be downloaded and analyzed. This can be a ZIP file or an URL to git repository")
	)

	flag.Parse()

	logme.Debugln("strict mode: ", *strictFlag)
	logme.Debugln("config file: ", *configFlag)
	logme.Debugln("source code: ", *sourceCodeUri)
	logme.Debugln("archive file: ", flag.Arg(0))

	if *configFlag == "" {
		logme.Errorln("no config file specified")
		flag.Usage()
		os.Exit(1)
	}

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

	b, err := archivetool.ReadArchive(pluginURL)
	if err != nil {
		logme.Errorln(fmt.Errorf("couldn't fetch plugin archive: %w", err))
		os.Exit(1)
	}

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

	diags, err := runner.Check(passes.Analyzers, archiveDir, sourceCodeDir, cfg)
	if err != nil {
		logme.Errorln(fmt.Errorf("check failed: %w", err))
		os.Exit(1)
	}

	var exitCode int

	if cfg.Global.JSONOutput {
		pluginID, pluginVersion, err := GetIDAndVersion(archiveDir)
		if err != nil {
			pluginID = "unknown"
			pluginVersion = "unknown"
			archiveDiag := analysis.Diagnostic{
				Name:     "zip-invalid",
				Severity: analysis.Error,
				Title:    "ZIP is improperly structured",
				Context:  "could not read plugin.json from archive to determine id and version",
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
			case analysis.OK:
				buf.WriteString(color.GreenString("ok: "))
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
		return "", sourceCodeCleanUp, fmt.Errorf("couldn't extract source code archive: %s. %w", sourceCodeUri, err)
	}
	return extractedDir, sourceCodeCleanUp, nil

}
