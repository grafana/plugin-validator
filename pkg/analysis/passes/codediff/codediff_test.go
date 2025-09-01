package codediff

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/llmclient"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"
)

func mockGrafanaClockPanelVersionsAPI() {
	mockResponse := `{
					  "items": [
						{
						  "id": 5942,
						  "pluginSlug": "grafana-clock-panel",
						  "version": "2.1.7"
						},
						{
						  "id": 5942,
						  "pluginSlug": "grafana-clock-panel",
						  "version": "2.1.6"
						}
					  ],
					  "orderBy": "version",
					  "direction": "desc",
					  "pluginSlugOrId": "grafana-clock-panel",
					  "links": [
						{
						  "rel": "self",
						  "href": "/plugins/grafana-clock-panel/versions/"
						}
					  ]
					}`

	httpmock.RegisterResponder(
		"GET",
		"https://grafana.com/api/plugins/grafana-clock-panel/versions",
		httpmock.NewStringResponder(http.StatusOK, mockResponse),
	)

	// Mock GitHub API releases
	githubReleasesResponse := `[
		{
			"tag_name": "v2.1.7",
			"target_commitish": "0618b305d0c9bfe9e229ce441a90c0eec03640ba",
			"html_url": "https://github.com/grafana/clock-panel/releases/tag/v2.1.7",
			"created_at": "2022-12-01T00:00:00Z"
		},
		{
			"tag_name": "v2.1.6",
			"target_commitish": "abb44ed5bb37b9feb5e6aa64fc3b8d4bfaaf9231",
			"html_url": "https://github.com/grafana/clock-panel/releases/tag/v2.1.6",
			"created_at": "2022-11-01T00:00:00Z"
		}
	]`

	httpmock.RegisterResponder(
		"GET",
		"https://api.github.com/repos/grafana/clock-panel/releases",
		httpmock.NewStringResponder(http.StatusOK, githubReleasesResponse),
	)
}

func TestValidDiffUrlGenerated(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Set up mock LLM client for this test
	mockClient := llmclient.NewMockLLMClient()
	SetLLMClient(mockClient)
	defer func() {
		// Restore original client after test
		SetLLMClient(llmclient.NewGeminiClient())
	}()

	pluginId := "grafana-clock-panel"
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.8"
		}
	}`)
	mockGrafanaClockPanelVersionsAPI()
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir:       "",
			SourceCodeReference: "https://github.com/grafana/clock-panel/",
		},
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, nil, result)
}

func TestLLMResponseFiltering_YesResponsesAreReported(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Create mock responses with "yes" answers that should be reported
	mockResponses := []llmclient.MockResponse{
		{
			Question:     "Are there any security vulnerabilities in the code changes?",
			Answer:       "Yes, there is a potential XSS vulnerability in the input handling.",
			RelatedFiles: []string{"src/ClockPanel.tsx"},
			CodeSnippet:  "dangerouslySetInnerHTML: {__html: userInput}",
			ShortAnswer:  "yes",
		},
		{
			Question:     "Are there any performance issues in the code changes?",
			Answer:       "Yes, the code contains an inefficient loop that could cause performance degradation.",
			RelatedFiles: []string{"src/migrations.ts"},
			CodeSnippet:  "for (let i = 0; i < largeArray.length; i++)",
			ShortAnswer:  "yes",
		},
	}

	mockClient := llmclient.NewMockLLMClient().WithResponses(mockResponses)
	SetLLMClient(mockClient)
	defer func() {
		SetLLMClient(llmclient.NewGeminiClient())
	}()

	pluginId := "grafana-clock-panel"
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.8"
		}
	}`)
	mockGrafanaClockPanelVersionsAPI()

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir:       "",
			SourceCodeReference: "https://github.com/grafana/clock-panel/",
		},
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Equal(t, nil, result)

	// Should have 3 reports: 1 for diff URL + 2 for the "yes" LLM responses
	require.Len(t, interceptor.Diagnostics, 3)

	// Verify the LLM analysis reports are present
	var analysisReports []*analysis.Diagnostic
	for _, diag := range interceptor.Diagnostics {
		if diag.Name == "code-diff-analysis" {
			analysisReports = append(analysisReports, diag)
		}
	}
	require.Len(t, analysisReports, 2, "Should have 2 analysis reports for 'yes' responses")

	// Verify the content of the reports
	require.Contains(t, analysisReports[0].Title, "security vulnerabilities")
	require.Contains(t, analysisReports[1].Title, "performance issues")
}

func TestLLMResponseFiltering_NoResponsesAreIgnored(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Create mock responses with "no" answers that should NOT be reported
	mockResponses := []llmclient.MockResponse{
		{
			Question:     "Are there any security vulnerabilities in the code changes?",
			Answer:       "No security vulnerabilities were found in the code changes.",
			RelatedFiles: []string{"src/ClockPanel.tsx"},
			CodeSnippet:  "// Safe code implementation",
			ShortAnswer:  "no",
		},
		{
			Question:     "Are there any performance issues in the code changes?",
			Answer:       "No performance issues were identified in the code changes.",
			RelatedFiles: []string{"src/migrations.ts"},
			CodeSnippet:  "// Optimized implementation",
			ShortAnswer:  "no",
		},
	}

	mockClient := llmclient.NewMockLLMClient().WithResponses(mockResponses)
	SetLLMClient(mockClient)
	defer func() {
		SetLLMClient(llmclient.NewGeminiClient())
	}()

	pluginId := "grafana-clock-panel"
	var interceptor testpassinterceptor.TestPassInterceptor
	pluginJsonContent := []byte(`{
		"id": "` + pluginId + `",
		"type": "panel",
		"info": {
			"version": "2.1.8"
		}
	}`)
	mockGrafanaClockPanelVersionsAPI()

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		CheckParams: analysis.CheckParams{
			SourceCodeDir:       "",
			SourceCodeReference: "https://github.com/grafana/clock-panel/",
		},
		ResultOf: map[*analysis.Analyzer]any{
			metadata.Analyzer: pluginJsonContent,
		},
		Report: interceptor.ReportInterceptor(),
	}

	result, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Equal(t, nil, result)

	// Should have only 1 report for the diff URL, no LLM analysis reports
	require.Len(t, interceptor.Diagnostics, 1)

	// Verify no LLM analysis reports are present
	var analysisReports []*analysis.Diagnostic
	for _, diag := range interceptor.Diagnostics {
		if diag.Name == "code-diff-analysis" {
			analysisReports = append(analysisReports, diag)
		}
	}
	require.Len(t, analysisReports, 0, "Should have 0 analysis reports for 'no' responses")

	// Verify the only report is the diff URL report
	require.Equal(t, "code-diff-versions", interceptor.Diagnostics[0].Name)
}
