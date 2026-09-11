package gosec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/stretchr/testify/require"
)

func TestIncompleteScan(t *testing.T) {
	for _, script := range []string{"exit 2", "kill -KILL $$", "exit 0", "printf 'invalid JSON'"} {
		t.Run(script, func(t *testing.T) {
			directory := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(directory, "gosec"), []byte("#!/bin/sh\n"+script+"\n"), 0755))
			t.Setenv("PATH", directory)
			var diagnostics []analysis.Diagnostic
			pass := &analysis.Pass{AnalyzerName: Analyzer.Name, ResultOf: map[*analysis.Analyzer]any{sourcecode.Analyzer: directory}, Report: func(_ string, d analysis.Diagnostic) { diagnostics = append(diagnostics, d) }}
			_, err := run(pass)
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			require.Equal(t, "scan-incomplete", diagnostics[0].Name)
		})
	}
}
