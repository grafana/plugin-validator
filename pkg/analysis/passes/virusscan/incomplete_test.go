package virusscan

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/stretchr/testify/require"
)

func TestScanExecutionFailures(t *testing.T) {
	for _, tc := range []struct {
		name, output, command string
		incomplete            bool
	}{
		{"clean", "----------- SCAN SUMMARY -----------\nInfected files: 0", "exit 0", false},
		{"infected", "----------- SCAN SUMMARY -----------\nInfected files: 1", "exit 1", false},
		{"error with summary", "----------- SCAN SUMMARY -----------\nInfected files: 0", "exit 2", true},
		{"killed", "", "kill -KILL $$", true},
		{"missing summary", "ClamAV could not load database", "exit 0", true},
		{"truncated summary", "----------- SCAN SUMMARY -----------", "exit 0", true},
		{"inconsistent findings", "----------- SCAN SUMMARY -----------\nInfected files: 0", "exit 1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			script := "#!/bin/sh\nprintf '%s\\n' '" + tc.output + "'\n" + tc.command + "\n"
			require.NoError(t, os.WriteFile(filepath.Join(directory, "clamscan"), []byte(script), 0755))
			t.Setenv("PATH", directory)
			t.Setenv("SKIP_CLAMAV", "")
			var diagnostics []analysis.Diagnostic
			pass := &analysis.Pass{AnalyzerName: Analyzer.Name, ResultOf: map[*analysis.Analyzer]any{archive.Analyzer: directory}, Report: func(_ string, d analysis.Diagnostic) { diagnostics = append(diagnostics, d) }}
			_, err := run(pass)
			require.NoError(t, err)
			if tc.incomplete {
				require.Len(t, diagnostics, 1)
				require.Equal(t, "scan-incomplete", diagnostics[0].Name)
			} else {
				for _, diagnostic := range diagnostics {
					require.NotEqual(t, "scan-incomplete", diagnostic.Name)
				}
			}
		})
	}
}
