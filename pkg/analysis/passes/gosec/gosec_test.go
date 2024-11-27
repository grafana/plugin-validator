package gosec

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func isGoSecInstalled() bool {
	goSecPath, _ := exec.LookPath("gosec")
	return goSecPath != ""
}

func TestGoSecNoWarnings(t *testing.T) {
	if !isGoSecInstalled() {
		t.Skip("gosec not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "no-warnings"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	fmt.Println(err)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestWithGoSecWarnings(t *testing.T) {
	if !isGoSecInstalled() {
		t.Skip("gosec not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "with-warnings"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "gosec analysis reports 2 issues with HIGH severity", interceptor.Diagnostics[0].Title)
}
