package plugindocs

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestRuleForSeverity(t *testing.T) {
	require.Equal(t, pluginDocsError, ruleForSeverity("error"))
	require.Equal(t, pluginDocsWarning, ruleForSeverity("warning"))
	require.Equal(t, pluginDocsInfo, ruleForSeverity("info"))
	// unknown severities fall back to warning so unexpected CLI output is still surfaced
	require.Equal(t, pluginDocsWarning, ruleForSeverity("debug"))
	require.Equal(t, pluginDocsWarning, ruleForSeverity(""))
}

func TestFormatTitle(t *testing.T) {
	tests := []struct {
		name string
		in   cliDiagnostic
		want string
	}{
		{
			name: "file and line",
			in:   cliDiagnostic{Rule: "no-script-tags", Title: "Script tag found", File: "docs/index.md", Line: 12},
			want: "[no-script-tags] Script tag found (docs/index.md:12)",
		},
		{
			name: "file only",
			in:   cliDiagnostic{Rule: "has-markdown-files", Title: "No markdown files found", File: "docs/"},
			want: "[has-markdown-files] No markdown files found (docs/)",
		},
		{
			name: "no location",
			in:   cliDiagnostic{Rule: "manifest-valid", Title: "Manifest is invalid"},
			want: "[manifest-valid] Manifest is invalid",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, formatTitle(tc.in))
		})
	}
}

// TestSkipsWhenNoDocsPath is the end-to-end test for the hard gate. A plugin without a
// docsPath must produce zero diagnostics and must NOT invoke the CLI. We don't mock exec
// here: if the gate leaks, a spurious CLI invocation would be visible as diagnostics or
// test latency. The test passes even on machines without node/npx installed.
func TestSkipsWhenNoDocsPath(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer:     filepath.Join("testdata", "no-docspath"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{},
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestSkipsWhenDocsPathIsEmpty(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer:     filepath.Join("testdata", "empty-docspath"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{DocsPath: ""},
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestSkipsWhenDocsPathIsWhitespaceOnly(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer:     filepath.Join("testdata", "empty-docspath"),
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{DocsPath: "   "},
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestSkipsWhenNoSourceCode covers the case where the validator was invoked against a
// zip archive only (no source code reference). The analyzer should skip silently.
func TestSkipsWhenNoSourceCode(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			// sourcecode.Analyzer intentionally absent - matches how the runner records
			// "no source code provided" (sourcecode.run returns (nil, nil) in that case).
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{DocsPath: "docs"},
			},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

// TestSkipsWhenNoMetadata covers the case where nestedmetadata did not produce a result
// (e.g. archive was empty or invalid).
func TestSkipsWhenNoMetadata(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: "./",
		ResultOf: map[*analysis.Analyzer]interface{}{
			sourcecode.Analyzer: filepath.Join("testdata", "with-docspath"),
			// nestedmetadata.Analyzer intentionally absent
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}
