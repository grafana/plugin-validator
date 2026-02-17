package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/grafana/plugin-validator/pkg/analysis/output"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/runner"
	"github.com/grafana/plugin-validator/pkg/service"
)

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
		outputToFile = flag.String(
			"output-to-file",
			"",
			"Write JSON output to specified file",
		)
		jsonOutputFlag = flag.Bool(
			"jsonOutput",
			false,
			"If set, outputs results in JSON format regardless of config file setting",
		)
		ghaOutputFlag = flag.Bool(
			"ghaOutput",
			false,
			"If set, outputs results in GitHub Actions format regardless of config file setting",
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
	logme.Debugln("outputToFile: ", *outputToFile)

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

	result, err := service.ValidatePlugin(service.Params{
		PluginURL:        pluginURL,
		SourceCodeUri:    *sourceCodeUri,
		Checksum:         *checksum,
		Analyzer:         *analyzer,
		AnalyzerSeverity: *analyzerSeverity,
		Config:           &cfg,
	})
	if err != nil {
		logme.Errorln(fmt.Errorf("couldn't validate plugin: %w", err))
		os.Exit(1)
	}
	diags := result.Diagnostics
	pluginID := result.PluginID
	pluginVersion := result.PluginVersion
	var outputMarshaler output.Marshaler

	// Additional JSON output to file
	if *outputToFile != "" {
		ob, err := output.NewJSONMarshaler(pluginID, pluginVersion).Marshal(diags)
		if err != nil {
			logme.Errorln(fmt.Errorf("couldn't marshal output: %w", err))
			os.Exit(1)
		}
		if err := os.WriteFile(*outputToFile, ob, 0644); err != nil {
			logme.Errorln(fmt.Errorf("couldn't write output to file: %w", err))
		}
	}

	// Stdout/Stderr output.

	// Check that the config and CLI flags are valid
	if (cfg.Global.JSONOutput && cfg.Global.GHAOutput) || (*jsonOutputFlag && *ghaOutputFlag) {
		logme.Errorln("can't have more than one output type set to true")
		os.Exit(1)
	}
	var jsonOutput, ghaOutput bool
	if *jsonOutputFlag || *ghaOutputFlag {
		// Prioritize CLI flags
		jsonOutput = *jsonOutputFlag
		ghaOutput = *ghaOutputFlag
	} else {
		// Fall-back to config file
		jsonOutput = cfg.Global.JSONOutput
		ghaOutput = cfg.Global.GHAOutput
	}

	// Determine the correct marshaler depending on the config
	if jsonOutput {
		outputMarshaler = output.NewJSONMarshaler(pluginID, pluginVersion)
	} else if ghaOutput {
		outputMarshaler = output.MarshalGHA
	} else {
		outputMarshaler = output.MarshalCLI
	}

	// Write to stdout or stderr, depending on config
	var outWriter io.Writer
	if jsonOutput {
		outWriter = os.Stdout
	} else {
		outWriter = os.Stderr
	}

	// Write output with the correct marshaler, depending on the config, then exit.
	// Nothing else should be printed from here on, or the output may become invalid.
	ob, err := outputMarshaler.Marshal(diags)
	if err != nil {
		logme.Errorln(fmt.Errorf("couldn't marshal output: %w", err))
		os.Exit(1)
	}
	if _, err = fmt.Fprintln(outWriter, string(ob)); err != nil {
		logme.Errorln(fmt.Errorf("couldn't write output: %w", err))
		os.Exit(1)
	}
	os.Exit(output.ExitCode(*strictFlag, diags))
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
