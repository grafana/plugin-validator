package runner

import (
	"fmt"

	"github.com/grafana/plugin-validator/pkg/analysis"
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
	Enabled  *bool                 `yaml:"enabled"`
	Severity *analysis.Severity    `yaml:"severity"`
	Rules    map[string]RuleConfig `yaml:"rules"`
}

type RuleConfig struct {
	Enabled  *bool              `yaml:"enabled"`
	Severity *analysis.Severity `yaml:"severity"`
}

var defaultSeverity = analysis.Warning

func Check(analyzers []*analysis.Analyzer, dir string, cfg Config) (map[string][]analysis.Diagnostic, error) {
	initAnalyzers(analyzers, cfg)
	diagnostics := make(map[string][]analysis.Diagnostic)
	//var diagnostics []analysis.Diagnostic

	pass := &analysis.Pass{
		RootDir:  dir,
		ResultOf: make(map[*analysis.Analyzer]interface{}),
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

func initAnalyzers(analyzers []*analysis.Analyzer, cfg Config) {
	for _, a := range analyzers {
		// Inherit global config
		analyzerEnabled := cfg.Global.Enabled
		analyzerSeverity := cfg.Global.Severity

		if analyzerSeverity == "" {
			analyzerSeverity = defaultSeverity
		}

		acfg, ok := cfg.Analyzers[a.Name]
		if ok {
			if acfg.Enabled != nil {
				analyzerEnabled = *acfg.Enabled
			}
			if acfg.Severity != nil {
				analyzerSeverity = *acfg.Severity
			}
		}

		for _, r := range a.Rules {
			// Inherit analyzer config
			ruleEnabled := analyzerEnabled
			ruleSeverity := analyzerSeverity

			rcfg, ok := acfg.Rules[r.Name]
			if ok {
				if rcfg.Enabled != nil {
					ruleEnabled = *rcfg.Enabled
				}
				if rcfg.Severity != nil {
					ruleSeverity = *rcfg.Severity
				}
			}

			r.Disabled = !ruleEnabled
			r.Severity = ruleSeverity
			r.ReportAll = cfg.Global.ReportAll
		}
	}
}
