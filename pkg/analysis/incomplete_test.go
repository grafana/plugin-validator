package analysis_test

import (
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/stretchr/testify/require"
)

func TestDisabledAnalyzerDoesNotReportIncomplete(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{Analyzer: &analysis.Analyzer{Rules: []*analysis.Rule{{Disabled: true}}}, Report: interceptor.ReportInterceptor()}
	pass.ReportIncomplete("failed")
	require.Empty(t, interceptor.Diagnostics)
}
