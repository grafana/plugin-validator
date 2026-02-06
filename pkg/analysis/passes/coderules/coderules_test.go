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
		ResultOf: map[*analysis.Analyzer]any{
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

	// Verify specific rule IDs are used
	ruleNames := []string{}
	for _, d := range interceptor.Diagnostics {
		ruleNames = append(ruleNames, d.Name)
	}
	require.Contains(t, ruleNames, "code-rules-access-forbidden-os-environment")
	require.Contains(t, ruleNames, "code-rules-access-os-environment")
}

func TestAccessAllowedEnvVariables(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
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

	// Verify specific rule IDs are used
	ruleNames := []string{}
	for _, d := range interceptor.Diagnostics {
		ruleNames = append(ruleNames, d.Name)
	}
	require.Contains(t, ruleNames, "code-rules-access-forbidden-os-environment")
	require.Contains(t, ruleNames, "code-rules-access-os-environment")
}

func TestAccessFS(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
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

	// Verify specific rule IDs are used
	ruleNames := []string{}
	for _, d := range interceptor.Diagnostics {
		ruleNames = append(ruleNames, d.Name)
	}
	require.Contains(t, ruleNames, "code-rules-access-file-system-with-fs")
	require.Contains(t, ruleNames, "code-rules-access-file-system-with-filepath")
	require.Contains(t, ruleNames, "code-rules-access-file-system-with-os")
}

func TestUseSyscall(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
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
	require.Equal(
		t,
		"code-rules-access-syscall",
		interceptor.Diagnostics[0].Name,
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
		ResultOf: map[*analysis.Analyzer]any{
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
	require.Equal(
		t,
		"code-rules-console-logging",
		interceptor.Diagnostics[0].Name,
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
		ResultOf: map[*analysis.Analyzer]any{
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
	require.Equal(
		t,
		"code-rules-topnav-toggle",
		interceptor.Diagnostics[0].Name,
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
		require.Equal(
			t,
			"code-rules-window-properties",
			interceptor.Diagnostics[index].Name,
		)
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
		ResultOf: map[*analysis.Analyzer]any{
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
	require.Equal(
		t,
		"code-rules-access-old-react-internals",
		interceptor.Diagnostics[0].Name,
	)
}

func TestOutdatedSqldsVersion(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "outdated-sqlds-bad"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	for i := range interceptor.Diagnostics {
		require.Equal(
			t,
			"Outdated sqlds version detected (v1 or v2). Use sqlds/v3 or sqlds/v4 which have updated signatures that allow passing context.Context for forward compatibility.",
			interceptor.Diagnostics[i].Title,
		)
		require.Equal(t, analysis.Warning, interceptor.Diagnostics[i].Severity)
		require.Equal(
			t,
			"code-rules-outdated-sqlds-version",
			interceptor.Diagnostics[i].Name,
		)
	}
}

func TestOutdatedSqldsVersionGood(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "outdated-sqlds-good"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestNativeBrowserDialogs(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "native-browser-dialogs-bad"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 4)
	for i := range interceptor.Diagnostics {
		require.Equal(
			t,
			"Native browser dialogs (alert, confirm, prompt) are not permitted. Use Grafana UI components (Modal, ConfirmModal) instead.",
			interceptor.Diagnostics[i].Title,
		)
		require.Equal(t, analysis.Error, interceptor.Diagnostics[i].Severity)
		require.Equal(
			t,
			"code-rules-native-browser-dialogs",
			interceptor.Diagnostics[i].Name,
		)
	}
}

func TestFmtPrintLogging(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "fmt-print-logging"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	for i := range interceptor.Diagnostics {
		require.Equal(
			t,
			"Use the logger provided by the Grafana plugin SDK (github.com/grafana/grafana-plugin-sdk-go/backend) instead of fmt.Println/fmt.Print/fmt.Printf for proper log management and integration with Grafana's logging system.",
			interceptor.Diagnostics[i].Title,
		)
		require.Equal(t, analysis.Error, interceptor.Diagnostics[i].Severity)
		require.Equal(
			t,
			"code-rules-fmt-print-logging",
			interceptor.Diagnostics[i].Name,
		)
	}
}

func TestWindowOpenWithoutNoopener(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "window-open-bad"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)
	for i := range interceptor.Diagnostics {
		require.Equal(
			t,
			"window.open() called without 'noopener,noreferrer' in the features parameter. This creates a tab nabbing vulnerability. Use window.open(url, target, 'noopener,noreferrer').",
			interceptor.Diagnostics[i].Title,
		)
		require.Equal(t, analysis.Error, interceptor.Diagnostics[i].Severity)
		require.Equal(
			t,
			"code-rules-window-open-without-noopener",
			interceptor.Diagnostics[i].Name,
		)
	}
}

func TestWindowOpenWithNoopenerGood(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "window-open-good"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestDeprecatedGfFormCSSClasses(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "deprecated-gf-form-bad"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Greater(t, len(interceptor.Diagnostics), 0)
	for i := range interceptor.Diagnostics {
		require.Equal(
			t,
			"Deprecated Grafana CSS class name detected (gf-form*). Use @grafana/ui components instead of legacy CSS classes.",
			interceptor.Diagnostics[i].Title,
		)
		require.Equal(t, analysis.Warning, interceptor.Diagnostics[i].Severity)
		require.Equal(
			t,
			"code-rules-deprecated-gf-form-css-classes",
			interceptor.Diagnostics[i].Name,
		)
	}
}

func TestDirectWindowLocationAccess(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "direct-window-location-bad"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 4)
	for i := range interceptor.Diagnostics {
		require.Equal(
			t,
			"Direct access to window.location is not permitted. Use locationService from @grafana/runtime instead.",
			interceptor.Diagnostics[i].Title,
		)
		require.Equal(t, analysis.Warning, interceptor.Diagnostics[i].Severity)
		require.Equal(
			t,
			"code-rules-direct-window-location-access",
			interceptor.Diagnostics[i].Name,
		)
	}
}

func TestDirectWindowLocationAccessGood(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "direct-window-location-good"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestTsIgnoreSuppress(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "ts-ignore-bad"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 2)
	for i := range interceptor.Diagnostics {
		require.Equal(
			t,
			"Avoid using @ts-ignore or @ts-expect-error to suppress TypeScript errors. Fix TypeScript errors properly so issues are caught during compilation rather than at runtime.",
			interceptor.Diagnostics[i].Title,
		)
		require.Equal(t, analysis.Warning, interceptor.Diagnostics[i].Severity)
		require.Equal(
			t,
			"code-rules-ts-ignore-suppress",
			interceptor.Diagnostics[i].Name,
		)
	}
}

func TestTsIgnoreSuppressGood(t *testing.T) {
	if !isSemgrepInstalled() {
		t.Skip("semgrep not installed, skipping test")
		return
	}
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: filepath.Join("testdata", "ts-ignore-good"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
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
