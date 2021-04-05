package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes"
	"github.com/grafana/plugin-validator/pkg/runner"
)

func main() {
	var (
		strictFlag = flag.Bool("strict", false, "If set, plugincheck returns non-zero exit code for warnings")
	)

	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "missing plugin url")
		os.Exit(1)
	}

	pluginURL := os.Args[1]

	b, err := readArchive(pluginURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Extract the ZIP archive in a temporary directory.
	archiveDir, cleanup, err := extractPlugin(bytes.NewReader(b))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer cleanup()

	diags, err := runner.Check(passes.Analyzers, archiveDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var exitCode int

	for _, d := range diags {
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
		}

		if d.Context != "" {
			buf.WriteString(d.Context + ": ")
		}

		buf.WriteString(d.Message)

		fmt.Fprintln(os.Stderr, buf.String())
	}

	os.Exit(exitCode)
}
