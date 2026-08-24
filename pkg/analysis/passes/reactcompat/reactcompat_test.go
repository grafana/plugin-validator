package reactcompat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestParseResultsRejectsJSONWithoutResultFields(t *testing.T) {
	for _, payload := range []string{
		`null`,
		`{}`,
		`{"error":"No source map files found"}`,
		`{"sourceCodeIssues":null,"dependencyIssues":[]}`,
		`{"sourceCodeIssues":{},"dependencyIssues":null}`,
	} {
		t.Run(payload, func(t *testing.T) {
			_, err := parseResults([]byte(payload))
			require.Error(t, err)
		})
	}
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

	count := reportIssues(pass, output, "")
	require.Equal(t, 1, count)
	require.Len(t, interceptor.Diagnostics, 1)

	d := interceptor.Diagnostics[0]
	require.Equal(t, "react-19-usePropTypes", d.Name)
	require.Equal(t, analysis.Warning, d.Severity)
	require.Equal(t, "React 19 compatibility: Uses deprecated propTypes", d.Title)
	require.Contains(t, d.Detail, "module.js")
	require.Contains(t, d.Detail, "10")
	require.Contains(t, d.Detail, "Remove propTypes.")
	require.Contains(t, d.Detail, "https://react.dev/upgrade")
	require.Contains(t, d.Detail, "this may be a false positive")
}

func TestReportIssuesKeepsFirstIssuePerFile(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, "/some/archive/dir")

	output := &reactDetectOutput{
		SourceCodeIssues: map[string][]sourceCodeIssue{
			"usePropTypes": {
				{
					Pattern:  "usePropTypes",
					Location: location{File: "module.js", Line: 10},
					Problem:  "Uses deprecated propTypes",
					Fix:      fix{Description: "Remove propTypes."},
					Link:     "https://react.dev/upgrade",
				},
				{
					Pattern:  "usePropTypes",
					Location: location{File: "module.js", Line: 20},
					Problem:  "Uses deprecated propTypes",
					Fix:      fix{Description: "Remove propTypes."},
					Link:     "https://react.dev/upgrade",
				},
				{
					Pattern:  "usePropTypes",
					Location: location{File: "other.js", Line: 30},
					Problem:  "Uses deprecated propTypes",
					Fix:      fix{Description: "Remove propTypes."},
					Link:     "https://react.dev/upgrade",
				},
			},
		},
	}

	count := reportIssues(pass, output, "")
	require.Equal(t, 2, count)
	require.Len(t, interceptor.Diagnostics, 2)
	require.Contains(t, interceptor.Diagnostics[0].Detail, "Detected in module.js at line 10.")
	require.NotContains(t, interceptor.Diagnostics[0].Detail, "line 20")
	require.Contains(t, interceptor.Diagnostics[1].Detail, "Detected in other.js at line 30.")
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

	count := reportIssues(pass, output, "")
	require.Equal(t, 1, count)
	require.Len(t, interceptor.Diagnostics, 1)

	d := interceptor.Diagnostics[0]
	require.Equal(t, "react-19-dep-oldReactDom", d.Name)
	require.Equal(t, analysis.Warning, d.Severity)
	require.Equal(t, "React 19 compatibility: Depends on old react-dom", d.Title)
	require.Contains(t, d.Detail, "react-dom, react")
	require.Contains(t, d.Detail, "https://example.com/fix")
	require.Contains(t, d.Detail, "this may be a false positive")
}

// TestReportIssuesNil verifies that a nil output produces no diagnostics.
func TestReportIssuesNil(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, "/some/archive/dir")

	count := reportIssues(pass, nil, "")
	require.Equal(t, 0, count)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestNpxNotAvailable verifies that the analyzer silently skips when
// npx is not found in PATH, producing no diagnostics.
func TestNpxNotAvailable(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")

	archiveDir := t.TempDir()
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, archiveDir)

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Nil(t, result)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestRunBoundsReactDetectOutput(t *testing.T) {
	issues := make([]sourceCodeIssue, 500)
	for i := range issues {
		issues[i] = sourceCodeIssue{
			Pattern:  "usePropTypes",
			Severity: "critical",
			Location: location{File: fmt.Sprintf("module-%d.js", i), Line: i + 1, Column: 1},
			Problem:  "Uses deprecated propTypes " + strings.Repeat("detail ", 20),
			Fix:      fix{Description: "Remove propTypes usage."},
			Link:     "https://react.dev/upgrade",
		}
	}
	largeValidOutput, err := json.Marshal(reactDetectOutput{
		SourceCodeIssues: map[string][]sourceCodeIssue{"usePropTypes": issues},
		DependencyIssues: []dependencyIssue{},
	})
	require.NoError(t, err)
	smallValidOutput, err := json.Marshal(reactDetectOutput{
		SourceCodeIssues: map[string][]sourceCodeIssue{"usePropTypes": issues[:2]},
		DependencyIssues: []dependencyIssue{},
	})
	require.NoError(t, err)

	tests := []struct {
		name        string
		stdout      []byte
		stderr      []byte
		killed      bool
		valid       bool
		totalIssues int
		truncated   bool
	}{
		{name: "small valid output and zero exit", stdout: smallValidOutput, valid: true, totalIssues: 2},
		{name: "small valid output and killed", stdout: smallValidOutput, killed: true, valid: true, totalIssues: 2},
		{name: "small invalid output and killed", stdout: []byte("complete invalid output"), killed: true},
		{name: "large valid output and zero exit", stdout: largeValidOutput, valid: true, totalIssues: len(issues), truncated: true},
		{name: "large valid output and killed", stdout: largeValidOutput, killed: true, valid: true, totalIssues: len(issues), truncated: true},
		{
			name:      "large invalid output and killed",
			stdout:    []byte(strings.Repeat("invalid output\n", 5000)),
			stderr:    []byte(strings.Repeat("stderr context\n", 5000)),
			killed:    true,
			truncated: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			binDir := t.TempDir()
			stdoutFile := filepath.Join(t.TempDir(), "stdout")
			stderrFile := filepath.Join(t.TempDir(), "stderr")
			require.NoError(t, os.WriteFile(stdoutFile, tc.stdout, 0o600))
			require.NoError(t, os.WriteFile(stderrFile, tc.stderr, 0o600))

			termination := "exit 0\n"
			if tc.killed {
				termination = "kill -KILL $$\n"
			}
			fakeNpx := filepath.Join(binDir, "npx")
			script := `#!/bin/sh
/bin/cat "$REACT_DETECT_TEST_STDOUT"
/bin/cat "$REACT_DETECT_TEST_STDERR" >&2
` + termination
			require.NoError(t, os.WriteFile(fakeNpx, []byte(script), 0o755))
			t.Setenv("REACT_DETECT_TEST_STDOUT", stdoutFile)
			t.Setenv("REACT_DETECT_TEST_STDERR", stderrFile)
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			var interceptor testpassinterceptor.TestPassInterceptor
			result, runErr := Analyzer.Run(newPass(&interceptor, t.TempDir()))
			require.NoError(t, runErr)
			require.Nil(t, result)

			serialized, marshalErr := json.Marshal(interceptor.Diagnostics)
			require.NoError(t, marshalErr)
			require.LessOrEqual(t, len(serialized), maxReportBytes)

			var details strings.Builder
			issueReports := 0
			for _, diagnostic := range interceptor.Diagnostics {
				details.WriteString(diagnostic.Detail)
				if strings.HasPrefix(diagnostic.Title, "React 19 compatibility: Uses deprecated propTypes") {
					issueReports++
				}
			}

			if tc.valid {
				require.NotContains(t, details.String(), "signal: killed")
				if tc.truncated {
					require.Positive(t, issueReports)
					require.Less(t, issueReports, tc.totalIssues)
					require.Contains(t, details.String(), "issues were omitted")
				} else {
					require.Equal(t, tc.totalIssues, issueReports)
					require.NotContains(t, details.String(), "issues were omitted")
				}
				return
			}

			require.Zero(t, issueReports)
			require.Contains(t, details.String(), "signal: killed")
			require.Contains(t, details.String(), "invalid output")
			if len(tc.stderr) > 0 {
				require.Contains(t, details.String(), "stderr context")
			}
			if tc.truncated {
				require.Contains(t, details.String(), "truncated")
			} else {
				require.Contains(t, details.String(), string(tc.stdout))
				require.NotContains(t, details.String(), "truncated")
			}
		})
	}
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

	count := reportIssues(pass, output, "")
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

// TestRelativeToArchiveSymlinkedDir verifies that paths reported with the
// archive dir's symlinks resolved (e.g. macOS /var/folders -> /private/var)
// are still made relative.
func TestRelativeToArchiveSymlinkedDir(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	require.NoError(t, os.Mkdir(target, 0o755))
	link := filepath.Join(base, "link")
	require.NoError(t, os.Symlink(target, link))

	resolved, err := filepath.EvalSymlinks(link)
	require.NoError(t, err)

	require.Equal(t, "src/module.ts", relativeToArchive(resolved+"/src/module.ts", link))
	require.Equal(t, "src/module.ts", relativeToArchive(link+"/src/module.ts", link))
	require.Equal(t, "/elsewhere/src/module.ts", relativeToArchive("/elsewhere/src/module.ts", link))
	require.Equal(t, "/abs/path.ts", relativeToArchive("/abs/path.ts", ""))
}
