package runner

import (
	"fmt"

	"github.com/grafana/plugin-validator/pkg/analysis"
)

func Check(analyzers []*analysis.Analyzer, dir string) ([]analysis.Diagnostic, error) {
	var diagnostics []analysis.Diagnostic

	pass := &analysis.Pass{
		RootDir:  dir,
		ResultOf: make(map[*analysis.Analyzer]interface{}),
		Report: func(d analysis.Diagnostic) {
			// Collect all diagnostics for presenting at the end.
			diagnostics = append(diagnostics, d)
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
