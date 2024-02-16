package checksum

import (
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidMD5Checksum(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestCheckSumWithSpaceAndFileName(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "f9b1c42c45cbf4953d7da5c31b3d73d9 the-archive-name.zip",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidSHA1Checksum(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "187cd80948f240957bf8399745335b89f005c5f0",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}
	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestInvalidMD5Checksum(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d4",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	// should not call the report function
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"The provided checksum f9b1c42c45cbf4953d7da5c31b3d73d9 does not match the plugin archive",
		interceptor.Diagnostics[0].Title,
	)

}

func TestInvalidSHA1Checksum(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "187cd80948f240957bf8399745335b89f005c5f0",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f1",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	// should not call the report function
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		"The provided checksum 187cd80948f240957bf8399745335b89f005c5f0 does not match the plugin archive",
		interceptor.Diagnostics[0].Title,
	)
}

func TestSkipsCheckSumWhenNoProvided(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "", // no checksum
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	// should not call the report function
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidMD5ChecksumFromUrl(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "https://example.com/checksum",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://example.com/checksum",
		httpmock.NewStringResponder(200, "f9b1c42c45cbf4953d7da5c31b3d73d9 "))

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidMD5ChecksumFromUrlWithSpaces(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "https://example.com/checksum",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://example.com/checksum",
		httpmock.NewStringResponder(200, "f9b1c42c45cbf4953d7da5c31b3d73d9 archive-file-name.zip"))

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 0)
}

func TestInvalidMD5ChecksumFromUrl(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "https://example.com/checksum",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://example.com/checksum",
		httpmock.NewStringResponder(200, "x9b1c42c45cbf4953d7da5c31b3d73d3"))

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	assert.Equal(
		t,
		"The provided checksum x9b1c42c45cbf4953d7da5c31b3d73d3 does not match the plugin archive",
		interceptor.Diagnostics[0].Title,
	)
}

func TestWrongUrl404(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "https://example.com/checksum",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://example.com/checksum",
		httpmock.NewStringResponder(404, ""))

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	assert.Equal(
		t,
		"checksum file not found: https://example.com/checksum",
		interceptor.Diagnostics[0].Title,
	)
}

func TestUrlWithInvalidContent(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{},
		CheckParams: &analysis.CheckParams{
			Checksum:              "https://example.com/checksum",
			ArchiveCalculatedMD5:  "f9b1c42c45cbf4953d7da5c31b3d73d9",
			ArchiveCalculatedSHA1: "187cd80948f240957bf8399745335b89f005c5f0",
		},
		Report: interceptor.ReportInterceptor(),
	}

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://example.com/checksum",
		httpmock.NewStringResponder(200, "{\"test\": \"test\"}"))

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	assert.Equal(
		t,
		"Invalid checksum format: {\"test\":",
		interceptor.Diagnostics[0].Title,
	)
}
