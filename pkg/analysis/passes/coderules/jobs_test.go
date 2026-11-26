package coderules

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestSemgrepJobs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		jobs     string
		wantArgs []string
	}{
		{"unset", "", nil},
		{"set", "2", []string{"--jobs", "2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			directory := t.TempDir()
			argsPath := filepath.Join(directory, "args")
			writeArgsRecordingSemgrep(t, directory, argsPath)
			t.Setenv(semgrepJobsEnv, tc.jobs)

			var interceptor testpassinterceptor.TestPassInterceptor
			_, err := run(newSemgrepPass(directory, &interceptor))
			require.NoError(t, err)

			recorded, err := os.ReadFile(argsPath)
			require.NoError(t, err)
			args := strings.Fields(string(recorded))
			require.Equal(t, directory, args[len(args)-1])

			jobsIndex := slices.Index(args, "--jobs")
			if tc.wantArgs == nil {
				require.Equal(t, -1, jobsIndex)
				return
			}
			require.NotEqual(t, -1, jobsIndex)
			require.Equal(t, tc.wantArgs, args[jobsIndex:jobsIndex+2])
		})
	}
}

func TestSemgrepJobsInvalid(t *testing.T) {
	for _, jobs := range []string{"0", "two"} {
		t.Run(jobs, func(t *testing.T) {
			directory := t.TempDir()
			argsPath := filepath.Join(directory, "args")
			writeArgsRecordingSemgrep(t, directory, argsPath)
			t.Setenv(semgrepJobsEnv, jobs)

			var interceptor testpassinterceptor.TestPassInterceptor
			_, err := run(newSemgrepPass(directory, &interceptor))
			require.NoError(t, err)

			require.Len(t, interceptor.Diagnostics, 1)
			require.Equal(t, "scan-incomplete", interceptor.Diagnostics[0].Name)
			require.NoFileExists(t, argsPath)
		})
	}
}

func writeArgsRecordingSemgrep(t *testing.T, directory, argsPath string) {
	t.Helper()
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > '" + argsPath + "'\nprintf '%s\\n' '{\"results\":[],\"errors\":[]}'\n"
	require.NoError(t, os.WriteFile(filepath.Join(directory, "semgrep"), []byte(script), 0755))
	t.Setenv("PATH", directory)
}

func newSemgrepPass(directory string, interceptor *testpassinterceptor.TestPassInterceptor) *analysis.Pass {
	return &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf:     map[*analysis.Analyzer]any{sourcecode.Analyzer: directory},
		Report:       interceptor.ReportInterceptor(),
	}
}
