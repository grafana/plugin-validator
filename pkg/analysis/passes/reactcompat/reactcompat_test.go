package reactcompat

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func newPass(interceptor *testpassinterceptor.TestPassInterceptor, archiveDir string) *analysis.Pass {
	return &analysis.Pass{
		AnalyzerName: "reactcompat",
		RootDir:      filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: archiveDir,
		},
		Report: interceptor.ReportInterceptor(),
	}
}

// TestParseResults verifies that a valid JSON payload is correctly decoded and
// mapped to the expected diagnostics.
func TestParseResults(t *testing.T) {
	jsonPayload := []byte(`{
		"sourceCodeIssues": {
			"usePropTypes": [
				{
					"pattern": "usePropTypes",
					"severity": "critical",
					"location": {"type": "source-map", "file": "module.js", "line": 42, "column": 10},
					"problem": "Uses deprecated propTypes",
					"fix": {"description": "Remove propTypes usage."},
					"link": "https://react.dev/blog/2024/04/25/react-19-upgrade-guide"
				}
			]
		},
		"dependencyIssues": [
			{
				"pattern": "oldReactDom",
				"severity": "critical",
				"problem": "Depends on old react-dom",
				"link": "https://example.com",
				"packageNames": ["react-dom", "react"]
			}
		]
	}`)

	output, err := parseResults(jsonPayload)
	require.NoError(t, err)
	require.Len(t, output.SourceCodeIssues, 1)
	require.Len(t, output.SourceCodeIssues["usePropTypes"], 1)
	require.Len(t, output.DependencyIssues, 1)

	sc := output.SourceCodeIssues["usePropTypes"][0]
	require.Equal(t, "usePropTypes", sc.Pattern)
	require.Equal(t, "module.js", sc.Location.File)
	require.Equal(t, 42, sc.Location.Line)
	require.Equal(t, "Uses deprecated propTypes", sc.Problem)
	require.Equal(t, "Remove propTypes usage.", sc.Fix.Description)
	require.Equal(t, "https://react.dev/blog/2024/04/25/react-19-upgrade-guide", sc.Link)

	dep := output.DependencyIssues[0]
	require.Equal(t, "oldReactDom", dep.Pattern)
	require.Equal(t, []string{"react-dom", "react"}, dep.PackageNames)
}

// TestParseResultsEmpty verifies that a payload with no issues produces an
// empty but non-nil result.
func TestParseResultsEmpty(t *testing.T) {
	jsonPayload := []byte(`{"sourceCodeIssues": {}, "dependencyIssues": []}`)

	output, err := parseResults(jsonPayload)
	require.NoError(t, err)
	require.NotNil(t, output)
	require.Len(t, output.SourceCodeIssues, 0)
	require.Len(t, output.DependencyIssues, 0)
}

// TestParseResultsMalformed verifies that garbage input returns an error rather
// than a panic or silent zero value.
func TestParseResultsMalformed(t *testing.T) {
	_, err := parseResults([]byte(`not valid json {{{`))
	require.Error(t, err)
}

// TestReportIssuesSourceCode verifies correct diagnostic generation for source
// code issues.
func TestReportIssuesSourceCode(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, "/some/archive/dir")

	output := &reactDetectOutput{
		SourceCodeIssues: map[string][]sourceCodeIssue{
			"usePropTypes": {
				{
					Pattern:  "usePropTypes",
					Severity: "critical",
					Location: location{File: "module.js", Line: 10, Column: 5},
					Problem:  "Uses deprecated propTypes",
					Fix:      fix{Description: "Remove propTypes."},
					Link:     "https://react.dev/upgrade",
				},
			},
		},
	}

	count := reportIssues(pass, output)
	require.Equal(t, 1, count)
	require.Len(t, interceptor.Diagnostics, 1)

	d := interceptor.Diagnostics[0]
	require.Equal(t, "react-19-usePropTypes", d.Name)
	require.Equal(t, analysis.Warning, d.Severity)
	require.Equal(t, "Uses deprecated propTypes", d.Title)
	require.Contains(t, d.Detail, "module.js")
	require.Contains(t, d.Detail, "10")
	require.Contains(t, d.Detail, "Remove propTypes.")
	require.Contains(t, d.Detail, "https://react.dev/upgrade")
}

// TestReportIssuesDependency verifies correct diagnostic generation for
// dependency issues.
func TestReportIssuesDependency(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, "/some/archive/dir")

	output := &reactDetectOutput{
		DependencyIssues: []dependencyIssue{
			{
				Pattern:      "oldReactDom",
				Severity:     "critical",
				Problem:      "Depends on old react-dom",
				Link:         "https://example.com/fix",
				PackageNames: []string{"react-dom", "react"},
			},
		},
	}

	count := reportIssues(pass, output)
	require.Equal(t, 1, count)
	require.Len(t, interceptor.Diagnostics, 1)

	d := interceptor.Diagnostics[0]
	require.Equal(t, "react-19-dep-oldReactDom", d.Name)
	require.Equal(t, analysis.Warning, d.Severity)
	require.Equal(t, "Depends on old react-dom", d.Title)
	require.Contains(t, d.Detail, "react-dom, react")
	require.Contains(t, d.Detail, "https://example.com/fix")
}

// TestReportIssuesNil verifies that a nil output produces no diagnostics.
func TestReportIssuesNil(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, "/some/archive/dir")

	count := reportIssues(pass, nil)
	require.Equal(t, 0, count)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestPrepareTmpDir verifies that prepareTmpDir creates the expected symlink
// and that the cleanup function removes the directory.
func TestPrepareTmpDir(t *testing.T) {
	archiveDir := t.TempDir()

	tmpDir, cleanup, err := prepareTmpDir(archiveDir)
	require.NoError(t, err)
	require.NotEmpty(t, tmpDir)

	distLink := filepath.Join(tmpDir, "dist")
	target, err := os.Readlink(distLink)
	require.NoError(t, err)
	require.Equal(t, archiveDir, target)

	cleanup()

	_, statErr := os.Stat(tmpDir)
	require.True(t, os.IsNotExist(statErr), "temp dir should be removed after cleanup")
}

// TestNpxNotAvailable verifies that the analyzer silently skips (nil, nil) when
// npx is not found in PATH, producing no diagnostics.
func TestNpxNotAvailable(t *testing.T) {
	if _, err := exec.LookPath("npx"); err == nil {
		t.Skip("npx is available in this environment; skipping npx-not-found test")
	}

	archiveDir := t.TempDir()
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, archiveDir)

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Nil(t, result)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestReportIssuesCombined verifies that multiple source code issue groups and a
// dependency issue are all counted and reported correctly in a single call.
func TestReportIssuesCombined(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, "/some/archive/dir")

	output := &reactDetectOutput{
		SourceCodeIssues: map[string][]sourceCodeIssue{
			"usePropTypes": {
				{
					Pattern:  "usePropTypes",
					Severity: "critical",
					Location: location{File: "module.js", Line: 10, Column: 5},
					Problem:  "Uses deprecated propTypes",
					Fix:      fix{Description: "Remove propTypes."},
					Link:     "https://react.dev/upgrade",
				},
			},
			"findDOMNode": {
				{
					Pattern:  "findDOMNode",
					Severity: "critical",
					Location: location{File: "other.js", Line: 20, Column: 3},
					Problem:  "Uses removed findDOMNode",
					Fix:      fix{Description: "Use a ref instead."},
					Link:     "https://react.dev/upgrade#finddomnode",
				},
			},
		},
		DependencyIssues: []dependencyIssue{
			{
				Pattern:      "oldReactDom",
				Severity:     "critical",
				Problem:      "Depends on old react-dom",
				Link:         "https://example.com/fix",
				PackageNames: []string{"react-dom"},
			},
		},
	}

	count := reportIssues(pass, output)
	require.Equal(t, 3, count)
	require.Len(t, interceptor.Diagnostics, 3)

	ruleNames := make([]string, 0, 3)
	for _, d := range interceptor.Diagnostics {
		ruleNames = append(ruleNames, d.Name)
	}
	require.Contains(t, ruleNames, "react-19-usePropTypes")
	require.Contains(t, ruleNames, "react-19-findDOMNode")
	require.Contains(t, ruleNames, "react-19-dep-oldReactDom")
}

// TestNoArchiveDir verifies that a missing archive result produces no diagnostics.
func TestNoArchiveDir(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: nil,
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Nil(t, result)
	require.Len(t, interceptor.Diagnostics, 0)
}
