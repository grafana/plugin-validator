package runner

import (
	"errors"
	"fmt"
	"sync"
	"time"

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
	Enabled  *bool              `yaml:"enabled"`
	Severity *analysis.Severity `yaml:"severity"`
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
	if !params.Parallel {
		logme.Debugln("running in sequential mode")
		return checkSequential(analyzers, params)
	}
	logme.DebugFln("running in parallel mode")
	return checkParallel(analyzers, params)
}

func checkParallel(analyzers []*analysis.Analyzer, params analysis.CheckParams) (analysis.Diagnostics, error) {
	// TODO: func because it's the same when sequential
	diagnostics := make(analysis.Diagnostics)
	var diagnosticsMux sync.Mutex
	pass := &analysis.Pass{
		RootDir:     params.ArchiveDir,
		CheckParams: params,
		ResultOf:    sync.Map{},
		Report: func(name string, d analysis.Diagnostic) {
			// Collect all diagnostics for presenting at the end.
			diagnosticsMux.Lock()
			defer diagnosticsMux.Unlock()
			diagnostics[name] = append(diagnostics[name], d)
		},
	}

	var seen sync.Map

	broker := newResultsBroker()
	ready := make(chan struct{}, len(analyzers))
	errs := make(chan error, len(analyzers))
	for _, currentAnalyzer := range analyzers {
		currentAnalyzer := currentAnalyzer
		// Subscribe for all dependencies
		depsChs := make([]<-chan any, 0, len(currentAnalyzer.Requires))
		for _, dep := range currentAnalyzer.Requires {
			depsChs = append(depsChs, broker.subscribe(dep))
		}

		// Start goroutine that will run the analyzer
		go func() {
			// Wait for all analyzers to be ready (all pubsub dependency requests done)
			<-ready

			// Wait for all dependencies to run
			tickerQuit := make(chan struct{})
			go func() {
				ticker := time.NewTicker(time.Second * 30)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						logme.DebugFln("analyzer %s: waiting for dependencies", currentAnalyzer.Name)
					case <-tickerQuit:
						return
					}
				}
			}()
			var wg sync.WaitGroup
			wg.Add(len(depsChs))
			for _, depCh := range depsChs {
				go func() {
					defer wg.Done()
					<-depCh
				}()
			}
			tickerQuit <- struct{}{}
			close(tickerQuit)
			logme.DebugFln("analyzer %s: all dependencies done", currentAnalyzer.Name)

			// Do not run the same analyzer twice
			if _, ok := seen.Load(currentAnalyzer); ok {
				// Always return nil error to the main goroutine
				logme.DebugFln("analyzer %s: analyzer already run", currentAnalyzer.Name)
				errs <- nil
				return
			}
			seen.Store(currentAnalyzer, true)
			logme.DebugFln("Running analyzer %s", currentAnalyzer.Name)

			// Run the analyzer
			// TODO: concurrent???
			pass.AnalyzerName = currentAnalyzer.Name
			// TODO: ensure no concurrent access in analyzers
			res, err := currentAnalyzer.Run(pass)
			// Publish the result to all subscribers (dependent analyzers)
			defer func() {
				logme.DebugFln("analyzer %s: publishing result", currentAnalyzer.Name)
				broker.publish(currentAnalyzer, res)
			}()
			if err != nil {
				errs <- err
				return
			}
			pass.ResultOf.Store(currentAnalyzer, res)
			errs <- nil
		}()
	}

	// Signal all goroutines that subscription is done and analyzers are ready to run
	for i := 0; i < len(analyzers); i++ {
		ready <- struct{}{}
	}

	// Await all errors from all goroutines and return combined error
	var finalErr error
	for i := 0; i < len(analyzers); i++ {
		errors.Join(finalErr, <-errs)
	}
	return diagnostics, finalErr
}

func checkSequential(analyzers []*analysis.Analyzer, params analysis.CheckParams) (analysis.Diagnostics, error) {
	diagnostics := make(analysis.Diagnostics)

	pass := &analysis.Pass{
		RootDir:     params.ArchiveDir,
		CheckParams: params,
		ResultOf:    sync.Map{},
		Report: func(name string, d analysis.Diagnostic) {
			// Collect all diagnostics for presenting at the end.
			diagnostics[name] = append(diagnostics[name], d)
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
		pass.ResultOf.Store(currentAnalyzer, res)

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
		for _, exception := range cfg.Exceptions {
			if exception == pluginId {
				return true
			}
		}
	}
	return false
}
