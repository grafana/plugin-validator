package runner

import (
	"errors"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/stretchr/testify/require"
)

func TestAnalyzerErrorProducesIncompleteDiagnostic(t *testing.T) {
	failure := errors.New("scanner terminated")
	analyzer := &analysis.Analyzer{Name: "scanner", Run: func(*analysis.Pass) (any, error) { return nil, failure }}
	diagnostics, err := Check([]*analysis.Analyzer{analyzer}, analysis.CheckParams{ArchiveDir: t.TempDir()}, Config{Global: GlobalConfig{Enabled: true}}, analysis.Warning)
	require.ErrorIs(t, err, failure)
	require.Len(t, diagnostics["scanner"], 1)
	require.Equal(t, "scan-incomplete", diagnostics["scanner"][0].Name)
	require.Equal(t, analysis.Error, diagnostics["scanner"][0].Severity)
}
