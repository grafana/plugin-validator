package runner

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/logme"
)

type Config struct {
	Global    GlobalConfig              `yaml:"global"`
	Analyzers map[string]AnalyzerConfig `yaml:"analyzers"`
}

type GlobalConfig struct {
	Enabled    bool              `yaml:"enabled"`
	Severity   analysis.Severity `yaml:"severity"`
	JSONOutput bool              `yaml:"jsonOutput"`
	ReportAll  bool              `yaml:"reportAll"`
}

type AnalyzerConfig struct {
	Enabled    *bool                 `yaml:"enabled"`
	Severity   *analysis.Severity    `yaml:"severity"`
	Rules      map[string]RuleConfig `yaml:"rules"`
	Exceptions []string              `yaml:"exceptions"`
}

type RuleConfig struct {
	Enabled  *bool              `yaml:"enabled"`
	Severity *analysis.Severity `yaml:"severity"`
}

var defaultSeverity = analysis.Warning

func Check(analyzers []*analysis.Analyzer, dir string, sourceCodeDir string, cfg Config) (map[string][]analysis.Diagnostic, error) {
	pluginId, err := getPluginId(dir)
	if err != nil {
		// we only need the pluginId to check for exceptions
		// it might not be available at all
		logme.Debugln("Error getting plugin id")
	}

	initAnalyzers(analyzers, &cfg, pluginId)
	diagnostics := make(map[string][]analysis.Diagnostic)

	pass := &analysis.Pass{
		RootDir:       dir,
		SourceCodeDir: sourceCodeDir,
		ResultOf:      make(map[*analysis.Analyzer]interface{}),
		Report: func(name string, d analysis.Diagnostic) {
			// Collect all diagnostics for presenting at the end.
			diagnostics[name] = append(diagnostics[name], d)
		},
	}

	seen := make(map[*analysis.Analyzer]bool)

	var runFn func(a *analysis.Analyzer) error

	runFn = func(a *analysis.Analyzer) error {
		if _, ok := seen[a]; ok {
			return nil
		}

		seen[a] = true

		// Recurse until all required analyzers have been run.
		for _, dep := range a.Requires {
			if _, ok := seen[dep]; !ok {
				if err := runFn(dep); err != nil {
					return fmt.Errorf("%s: %w", dep.Name, err)
				}
			}
		}

		// TODO: Is there a better way to skip downstream analyzers than based
		// on a nil result?
		for _, dep := range a.Requires {
			if pass.ResultOf[dep] == nil {
				return nil
			}
		}
		pass.AnalyzerName = a.Name
		res, err := a.Run(pass)
		if err != nil {
			return err
		}
		pass.ResultOf[a] = res

		return nil
	}

	for _, a := range analyzers {
		if err := runFn(a); err != nil {
			return nil, fmt.Errorf("%s: %w", a.Name, err)
		}
	}

	return diagnostics, nil
}

func initAnalyzers(analyzers []*analysis.Analyzer, cfg *Config, pluginId string) {
	for _, currentAnalyzer := range analyzers {
		// Inherit global config file
		analyzerEnabled := cfg.Global.Enabled
		analyzerSeverity := cfg.Global.Severity

		// default to hardcoded defaultSeverity if not set
		if analyzerSeverity == "" {
			analyzerSeverity = defaultSeverity
		}

		// Override via config file
		analyzerConfig, ok := cfg.Analyzers[currentAnalyzer.Name]
		if ok {
			if analyzerConfig.Enabled != nil {
				analyzerEnabled = *analyzerConfig.Enabled
			}
			if analyzerConfig.Severity != nil {
				analyzerSeverity = *analyzerConfig.Severity
			}
		}

		// Override via exceptions
		if isExcepted(pluginId, &analyzerConfig) {
			analyzerEnabled = false
		}

		for _, currentRule := range currentAnalyzer.Rules {
			// Inherit analyzer config
			ruleEnabled := analyzerEnabled

			// use own config if available
			ruleSeverity := currentRule.Severity
			if ruleSeverity == "" {
				ruleSeverity = analyzerSeverity
			}

			// overwrite via config file
			ruleConfig, ok := analyzerConfig.Rules[currentRule.Name]
			if ok {
				if ruleConfig.Enabled != nil {
					ruleEnabled = *ruleConfig.Enabled
				}
				if ruleConfig.Severity != nil {
					ruleSeverity = *ruleConfig.Severity
				}
			}

			currentRule.Disabled = !ruleEnabled
			currentRule.Severity = ruleSeverity
			currentRule.ReportAll = cfg.Global.ReportAll
		}
	}
}

type BarebonePluginJson struct {
	Id string `json:"id"`
}

/*
* getPuginId returns the plugin id from the plugin.json file
* in the archive directory
*
* The plugin.json file might not be in the root directory
* at this point in the validator there's no certainty that the
* plugin.json file even exists
 */
func getPluginId(archiveDir string) (string, error) {
	if len(archiveDir) == 0 || archiveDir == "/" {
		return "", fmt.Errorf("archiveDir is empty")
	}
	pluginJsonPath, err := doublestar.FilepathGlob(archiveDir + "/**/plugin.json")
	if err != nil || len(pluginJsonPath) == 0 {
		return "", fmt.Errorf("Error getting plugin.json path: %s", err)
	}

	pluginJsonContent, err := os.ReadFile(pluginJsonPath[0])
	if err != nil {
		return "", err
	}
	//unmarshal plugin.json
	var pluginJson BarebonePluginJson
	err = json.Unmarshal(pluginJsonContent, &pluginJson)
	if err != nil {
		return "", err
	}
	return pluginJson.Id, nil
}

func isExcepted(pluginId string, cfg *AnalyzerConfig) bool {
	if len(pluginId) > 0 && cfg != nil && len(cfg.Exceptions) > 0 {
		for _, exception := range cfg.Exceptions {
			if exception == pluginId {
				return true
			}
		}
	}
	return false
}
