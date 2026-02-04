package llmreview

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

func TestLLMReviewSkipsWhenBlockingAnalyzerHasErrors(t *testing.T) {
	// Simulate diagnostics from a blocking analyzer (archive)
	diagnostics := analysis.Diagnostics{
		archive.Analyzer.Name: []analysis.Diagnostic{
			{
				Name:     "empty-archive",
				Severity: analysis.Error,
				Title:    "Archive is empty",
			},
		},
	}

	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf:     map[*analysis.Analyzer]any{
			// sourcecode returns nil when archive fails
		},
		Diagnostics: &diagnostics,
		Report: func(analyzerName string, d analysis.Diagnostic) {
			diagnostics[analyzerName] = append(diagnostics[analyzerName], d)
		},
	}

	_, err := run(pass)
	assert.NoError(t, err)

	// Check that llmReviewSkipped diagnostic was reported
	llmDiags := diagnostics[Analyzer.Name]
	assert.Len(t, llmDiags, 1)
	assert.Equal(t, "llm-review-skipped", llmDiags[0].Name)
	assert.Contains(t, llmDiags[0].Title, archive.Analyzer.Name)
}

func TestLLMReviewDoesNotSkipOnWarnings(t *testing.T) {
	// Simulate diagnostics with only warnings (no errors)
	diagnostics := analysis.Diagnostics{
		metadata.Analyzer.Name: []analysis.Diagnostic{
			{
				Name:     "some-warning",
				Severity: analysis.Warning,
				Title:    "This is just a warning",
			},
		},
	}

	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf:     map[*analysis.Analyzer]any{},
		Diagnostics:  &diagnostics,
		Report: func(analyzerName string, d analysis.Diagnostic) {
			diagnostics[analyzerName] = append(diagnostics[analyzerName], d)
		},
	}

	_, err := run(pass)
	assert.NoError(t, err)

	// LLM review should NOT have reported a skip (it will exit early due to no source code)
	// But importantly, it should NOT have reported llmReviewSkipped
	llmDiags := diagnostics[Analyzer.Name]
	for _, d := range llmDiags {
		assert.NotEqual(t, "llm-review-skipped", d.Name, "should not skip on warnings only")
	}
}
