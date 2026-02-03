package runner

import (
	"fmt"
	"slices"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/utils"
)

type Config struct {
	Global    GlobalConfig              `yaml:"global"`
	Analyzers map[string]AnalyzerConfig `yaml:"analyzers"`
}

type GlobalConfig struct {
	Enabled    bool              `yaml:"enabled"`
	Severity   analysis.Severity `yaml:"severity"`
	JSONOutput bool              `yaml:"jsonOutput"`
	GHAOutput  bool              `yaml:"ghaOutput"`
	ReportAll  bool              `yaml:"reportAll"`
}

type AnalyzerConfig struct {
	Enabled    *bool                 `yaml:"enabled"`
	Severity   *analysis.Severity    `yaml:"severity"`
	Rules      map[string]RuleConfig `yaml:"rules"`
	Exceptions []string              `yaml:"exceptions"`
}

type RuleConfig struct {
	Enabled    *bool              `yaml:"enabled"`
	Severity   *analysis.Severity `yaml:"severity"`
	Exceptions []string           `yaml:"exceptions"`
}

var defaultSeverity = analysis.Warning

func Check(
	analyzers []*analysis.Analyzer,
	params analysis.CheckParams,
	cfg Config,
	severityOverwrite analysis.Severity,
) (analysis.Diagnostics, error) {
	pluginId, err := utils.GetPluginId(params.ArchiveDir)
	if err != nil {
		// we only need the pluginId to check for exceptions
		// it might not be available at all
		logme.Debugln("Error getting plugin id")
	}

	initAnalyzers(analyzers, &cfg, pluginId, severityOverwrite)
	diagnostics := make(analysis.Diagnostics)

	pass := &analysis.Pass{
		RootDir:     params.ArchiveDir,
		CheckParams: params,
		ResultOf:    make(map[*analysis.Analyzer]any),
		Report: func(analyzerName string, d analysis.Diagnostic) {
			diagnostics[analyzerName] = append(diagnostics[analyzerName], d)
		},
	}

	seen := make(map[*analysis.Analyzer]bool)

	var runFn func(currentAnalyzer *analysis.Analyzer) error

	runFn = func(currentAnalyzer *analysis.Analyzer) error {
		// do not run the same analyzer twice
		if _, ok := seen[currentAnalyzer]; ok {
			return nil
		}

		seen[currentAnalyzer] = true

		logme.DebugFln("Running analyzer %s", currentAnalyzer.Name)

		// run all the dependencies of the analyzer
		for _, dep := range currentAnalyzer.Requires {
			// if dependency returned error. This analyzer should return error too
			if err := runFn(dep); err != nil {
				return fmt.Errorf("%s: %w", dep.Name, err)
			}
		}

		pass.AnalyzerName = currentAnalyzer.Name
		res, err := currentAnalyzer.Run(pass)
		if err != nil {
			return err
		}
		pass.ResultOf[currentAnalyzer] = res

		return nil
	}

	for _, a := range analyzers {
		if err := runFn(a); err != nil {
			// on an error we still return the diagnostics we have so far
			return diagnostics, err
		}
	}

	return diagnostics, nil
}

func initAnalyzers(
	analyzers []*analysis.Analyzer,
	cfg *Config,
	pluginId string,
	severityOverwrite analysis.Severity,
) {
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
				// Check for rule-level exceptions
				if slices.Contains(ruleConfig.Exceptions, pluginId) {
					logme.DebugFln(
						"Rule '%s' disabled for plugin '%s' due to a rule-level exception.",
						currentRule.Name,
						pluginId,
					)
					ruleEnabled = false
				}
			}

			if severityOverwrite != "" {
				ruleSeverity = severityOverwrite
			}

			currentRule.Disabled = !ruleEnabled
			currentRule.Severity = ruleSeverity
			currentRule.ReportAll = cfg.Global.ReportAll
		}
	}
}

func isExcepted(pluginId string, cfg *AnalyzerConfig) bool {
	if len(pluginId) > 0 && cfg != nil && len(cfg.Exceptions) > 0 {
		if slices.Contains(cfg.Exceptions, pluginId) {
			return true
		}
	}
	return false
}
