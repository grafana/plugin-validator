package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/fatih/color"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes"
	"github.com/grafana/plugin-validator/pkg/runner"
	"gopkg.in/yaml.v2"
)

type FormattedOutput struct {
	ID          string                           `json:"id"`
	Version     string                           `json:"version"`
	Diagnostics map[string][]analysis.Diagnostic `json:"plugin-validator"`
}

func main() {
	var (
		strictFlag = flag.Bool("strict", false, "If set, plugincheck returns non-zero exit code for warnings")
		configFlag = flag.String("config", "", "Path to configuration file")
	)

	flag.Parse()

	if *configFlag == "" {
		fmt.Fprintln(os.Stderr, "missing config")
		os.Exit(1)
	}

	cfg, err := readConfigFile(*configFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("couldn't read configuration: %w", err))
		os.Exit(1)
	}

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "missing plugin url")
		os.Exit(1)
	}

	pluginURL := flag.Args()[0]

	b, err := readArchive(pluginURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("couldn't fetch plugin archive: %w", err))
		os.Exit(1)
	}

	// Extract the ZIP archive in a temporary directory.
	archiveDir, cleanup, err := extractPlugin(bytes.NewReader(b))
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("couldn't extract plugin archive: %w", err))
		os.Exit(1)
	}
	defer cleanup()

	diags, err := runner.Check(passes.Analyzers, archiveDir, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("check failed: %w", err))
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
				Message:  "ZIP is improperly structured",
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

			buf.WriteString(d.Message)
			if len(d.Detail) > 0 {
				buf.WriteString("\n" + color.BlueString("detail: "))
				buf.WriteString(d.Detail)
			}
			fmt.Fprintln(os.Stderr, buf.String())
		}
	}

	os.Exit(exitCode)
}

func readConfigFile(path string) (runner.Config, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return runner.Config{}, err
	}

	var config runner.Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return runner.Config{}, err
	}

	return config, nil
}
