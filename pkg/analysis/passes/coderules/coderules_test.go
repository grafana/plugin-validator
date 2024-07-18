package coderules

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func isSemgrepInstalled() bool {
	semgrepPath, err := exec.LookPath("semgrep")
	if err != nil {
		return false
	}
	return semgrepPath != ""
}

func TestAccessEnvVariables(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "access-env"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	titles := []string{
		"It is not permitted to access environment variables from plugins. DO_NOT_INCLUDE is not an accessible variable.",
		"It is not permitted to access environment variables from plugins.",
	}
	require.ElementsMatch(t, titles, interceptor.GetTitles())
}

func TestAccessAllowedEnvVariables(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "access-env-allowed"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	titles := []string{
		"It is not permitted to access environment variables from plugins. MY_VARIABLE is not an accessible variable.",
		"It is not permitted to access environment variables from plugins. DO_NOT_INCLUDE is not an accessible variable.",
		"It is not permitted to access environment variables from plugins.",
	}
	require.ElementsMatch(t, titles, interceptor.GetTitles())
}

func TestAccessFS(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "access-fs"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	titles := []string{
		"It is not permitted to access the file system. Using fs.ReadFile is not permitted.",
		"It is not permitted to access the file system.",
		"It is not permitted to access the file system.",
	}
	require.ElementsMatch(t, titles, interceptor.GetTitles())
}

func TestUseSyscall(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "use-syscall"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"It is not permitted to use the syscall module. Using syscall.Getcwd is not permitted",
		interceptor.Diagnostics[0].Title,
	)
}

func TestJSConsoleLog(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "console-log"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Console logging detected. Plugins should not log to the console.",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"Code rule violation found in testdata/console-log/index.ts at line 2",
	)
}
