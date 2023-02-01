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

func Check(analyzers []interface{}, dir string, sourceCodeDir string, cfg Config) (map[string][]analysis.Diagnostic, error) {
	var staticAnalyzers []*analysis.StaticAnalyzer
	var dynamicAnalyzers []*analysis.Analyzer
	analyzersByName := make(map[string]interface{}, len(analyzers))
	for _, a := range analyzers {
		var dynamicAnalyzer *analysis.Analyzer
		if sa, ok := a.(analysis.StaticAnalyzer); ok {
			staticAnalyzers = append(staticAnalyzers, &sa)
			analyzersByName[dynamicAnalyzer.Name] = sa

			// Get the dynamic one for init
			dynamicAnalyzer = sa.GetAnalyzer()
		} else if da, ok := a.(*analysis.Analyzer); ok {
			dynamicAnalyzers = append(dynamicAnalyzers, da)
			analyzersByName[dynamicAnalyzer.Name] = da
			dynamicAnalyzer = da
		} else {
			return nil, fmt.Errorf("unknown analyzer type %T", a)
		}
		// Init all analyzers (static + dynamic, the static one embeds the dynamic one)
		initAnalyzer(dynamicAnalyzer, cfg)
	}

	diagnostics := make(map[string][]analysis.Diagnostic)
	//var diagnostics []analysis.Diagnostic

	pass := &analysis.Pass{
		RootDir:       dir,
		SourceCodeDir: sourceCodeDir,
		ResultOf:      make(map[*analysis.Analyzer]interface{}),
		Report: func(name string, d analysis.Diagnostic) {
			// Collect all diagnostics for presenting at the end.
			diagnostics[name] = append(diagnostics[name], d)
		},
	}

	seen := make(map[string]struct{}, len(analyzers))

	var runFn func(analyzer interface{}) error

	runFn = func(analyzer interface{}) error {
		var dynamicAnalyzer *analysis.Analyzer
		var staticAnalyzer analysis.StaticAnalyzer
		if sa, ok := analyzer.(analysis.StaticAnalyzer); ok {
			staticAnalyzer = sa
			dynamicAnalyzer = sa.GetAnalyzer()
		}

		if _, ok := seen[dynamicAnalyzer.Name]; ok {
			return nil
		}

		seen[dynamicAnalyzer.Name] = struct{}{}

		// Recurse until all required analyzers have been run.
		for _, dep := range dynamicAnalyzer.Requires {
			if _, ok := seen[dep.Name]; ok {
				continue
			}
			if err := runFn(dep); err != nil {
				return fmt.Errorf("%s: %w", dep.Name, err)
			}
		}
		// Do the same also for new dependencies (only set for static analyzers)
		for _, depName := range dynamicAnalyzer.NewRequires {
			if _, ok := seen[depName]; ok {
				continue
			}
			if err := runFn(analyzersByName[depName]); err != nil {
				return fmt.Errorf("%s: %w", depName, err)
			}
		}

		// TODO: Is there a better way to skip downstream analyzers than based
		// on a nil result?
		for _, dep := range dynamicAnalyzer.Requires {
			if pass.ResultOf[dep] == nil {
				return nil
			}
		}
		for _, depName := range dynamicAnalyzer.NewRequires {
			var innerAnalyzer *analysis.Analyzer
			dep := analyzersByName[depName]
			if sd, ok := dep.(analysis.StaticAnalyzer); ok {
				innerAnalyzer = sd.GetAnalyzer()
			} else if dd, ok := dep.(*analysis.Analyzer); ok {
				innerAnalyzer = dd
			} else {
				return fmt.Errorf("unknown dependency type %T", dep)
			}
			if pass.ResultOf[innerAnalyzer] == nil {
				return nil
			}
		}

		pass.AnalyzerName = dynamicAnalyzer.Name
		if staticAnalyzer != nil {
			if err := staticAnalyzer.Run(pass); err != nil {
				return err
			}
			// Only to allow older analyzers to access results in the old way, will go away
			// TODO: maybe replace with reflect?
			pass.ResultOf[staticAnalyzer.GetAnalyzer()] = staticAnalyzer.GetResult()
		} else {
			res, err := dynamicAnalyzer.Run(pass)
			if err != nil {
				return err
			}
			pass.ResultOf[dynamicAnalyzer] = res
		}
		return nil
	}

	for _, a := range analyzers {
		if err := runFn(a); err != nil {
			return nil, fmt.Errorf("%+v: %w", a, err)
		}
	}

	return diagnostics, nil
}

func initAnalyzer(a *analysis.Analyzer, cfg Config) {
	// Inherit global config file
	analyzerEnabled := cfg.Global.Enabled
	analyzerSeverity := cfg.Global.Severity

	// default to hardcoded defaultSeverity if not set
	if analyzerSeverity == "" {
		analyzerSeverity = defaultSeverity
	}

	// Override via config file
	analyzerConfig, ok := cfg.Analyzers[a.Name]
	if ok {
		if analyzerConfig.Enabled != nil {
			analyzerEnabled = *analyzerConfig.Enabled
		}
		if analyzerConfig.Severity != nil {
			analyzerSeverity = *analyzerConfig.Severity
		}
	}

	for _, r := range a.Rules {
		// Inherit analyzer config
		ruleEnabled := analyzerEnabled

		// use own config if available
		ruleSeverity := r.Severity
		if ruleSeverity == "" {
			ruleSeverity = analyzerSeverity
		}

		// overwrite via config file
		ruleConfig, ok := analyzerConfig.Rules[r.Name]
		if ok {
			if ruleConfig.Enabled != nil {
				ruleEnabled = *ruleConfig.Enabled
			}
			if ruleConfig.Severity != nil {
				ruleSeverity = *ruleConfig.Severity
			}
		}

		r.Disabled = !ruleEnabled
		r.Severity = ruleSeverity
		r.ReportAll = cfg.Global.ReportAll
	}
}
