package coderules

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
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
		"Code rule violation found in testdata/console-log/src/index.ts at line 2",
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Severity,
		analysis.Warning,
	)
}

func TestTopnavToggle(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "topnav-toggle"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"The `topnav` toggle is deprecated and will be removed in a future version of Grafana. Plugins should default to using the code where the toggle is enabled.",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"Code rule violation found in testdata/topnav-toggle/src/index.ts at line 5",
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Severity,
		analysis.Error,
	)
}

func TestWindowAccessWindowObjects(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "access-window"),
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 4)
	// Define expected diagnostics
	expectedDiagnostics := []struct {
		title  string
		detail string
	}{
		{
			"Detected access to restricted window property: window.grafanaBootData. Accessing window.grafanaBootData is not permitted.",
			"Code rule violation found in testdata/access-window/src/index.ts at line 2",
		},
		{
			"Detected access to restricted window property: window.grafanaBootData. Accessing window.grafanaBootData is not permitted.",
			"Code rule violation found in testdata/access-window/src/index.ts at line 3",
		},
		{
			"Detected access to restricted window property: window.grafanaRuntime. Accessing window.grafanaRuntime is not permitted.",
			"Code rule violation found in testdata/access-window/src/index.ts at line 4",
		},
		{
			"Detected access to restricted window property: window.__grafanaSceneContext. Accessing window.__grafanaSceneContext is not permitted.",
			"Code rule violation found in testdata/access-window/src/index.ts at line 5",
		},
	}

	// Test all expectations in a loop
	for index, tc := range expectedDiagnostics {
		require.Equal(t, tc.title, interceptor.Diagnostics[index].Title)
		require.Equal(t, tc.detail, interceptor.Diagnostics[index].Detail)
	}
}

func TestOldReactInternals(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "old-react-internals"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"Detected usage of React internal API '__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED'. This API is internal to React and should not be used directly as it may break in future React versions.",
		interceptor.Diagnostics[0].Title,
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"Code rule violation found in testdata/old-react-internals/src/module.tsx at line 4",
	)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Severity,
		analysis.Warning,
	)
}

func TestNoDirectCSSImports(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "no-direct-css-imports"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	for i := range interceptor.Diagnostics {
		require.Equal(t, analysis.Error, interceptor.Diagnostics[i].Severity)
	}
}

func TestNoEmotionGlobalImport(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "no-emotion-global-import"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	for i := range interceptor.Diagnostics {
		require.Equal(t, analysis.Error, interceptor.Diagnostics[i].Severity)
	}
}

func TestNoDirectCSSImportsGood(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "no-direct-css-imports-good"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}
