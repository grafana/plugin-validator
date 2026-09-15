package gosec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestIncompleteScan(t *testing.T) {
	for _, script := range []string{"exit 2", "kill -KILL $$", "exit 1", "printf 'invalid JSON'"} {
		t.Run(script, func(t *testing.T) {
			directory := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(directory, "gosec"), []byte("#!/bin/sh\n"+script+"\n"), 0755))
			t.Setenv("PATH", directory)
			var interceptor testpassinterceptor.TestPassInterceptor
			pass := &analysis.Pass{AnalyzerName: Analyzer.Name, ResultOf: map[*analysis.Analyzer]any{sourcecode.Analyzer: directory}, Report: interceptor.ReportInterceptor()}
			_, err := run(pass)
			require.NoError(t, err)
			require.Len(t, interceptor.Diagnostics, 1)
			require.Equal(t, "scan-incomplete", interceptor.Diagnostics[0].Name)
		})
	}
}

func TestQuietSuccessfulScan(t *testing.T) {
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "gosec"), []byte("#!/bin/sh\nexit 0\n"), 0755))
	t.Setenv("PATH", directory)
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf:     map[*analysis.Analyzer]any{sourcecode.Analyzer: directory},
		Report:       interceptor.ReportInterceptor(),
	}
	_, err := run(pass)
	require.NoError(t, err)
	require.Empty(t, interceptor.Diagnostics)
}
