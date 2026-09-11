package coderules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/stretchr/testify/require"
)

func TestIncompleteScan(t *testing.T) {
	for _, tc := range []struct{ name, output, command string }{
		{"exit failure", `{"results":[],"errors":[]}`, "exit 2"},
		{"killed", "", "kill -KILL $$"},
		{"invalid report", `not JSON`, ""},
		{"missing results", `{}`, ""},
		{"partial scan", `{"results":[],"errors":[{"type":"OutOfMemory","message":"memory limit reached"}]}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			script := "#!/bin/sh\nprintf '%s\\n' '" + tc.output + "'\n" + tc.command + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(directory, "semgrep"), []byte(script), 0755))
			t.Setenv("PATH", directory)
			var diagnostics []analysis.Diagnostic
			pass := &analysis.Pass{AnalyzerName: Analyzer.Name, ResultOf: map[*analysis.Analyzer]any{sourcecode.Analyzer: directory}, Report: func(_ string, d analysis.Diagnostic) { diagnostics = append(diagnostics, d) }}
			_, err := run(pass)
			require.NoError(t, err)
			require.Len(t, diagnostics, 1)
			require.Equal(t, "scan-incomplete", diagnostics[0].Name)
			require.Equal(t, analysis.Error, diagnostics[0].Severity)
		})
	}
}
