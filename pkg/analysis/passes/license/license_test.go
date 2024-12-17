package license

import (
	"path/filepath"
	"testing"

	"github.com/go-enry/go-license-detector/v4/licensedb/filer"
	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func TestValidLicenseApache(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "apache-2"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidLicenseAgpl(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "agpl"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestNoLicenseFound(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "no-license"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "LICENSE file not found")
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"Could not find a license file inside the plugin archive. Please make sure to include a LICENSE file in your archive.",
	)
}

func TestValidMitLicense(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "mit"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestValidLicenseBSD(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "bsd"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestInvalidUnilicenseLicense(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "unilicense"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, interceptor.Diagnostics[0].Title, "Valid license not found")
	require.Equal(
		t,
		interceptor.Diagnostics[0].Detail,
		"The provided license is not compatible with Grafana plugins. Please refer to https://grafana.com/licensing/ for more information.",
	)
}

func TestInvalidGenericLicense(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "generic-text"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "License file contains generic text", interceptor.Diagnostics[0].Title)
	require.Equal(
		t,
		"Your current license file contains generic text from the license template. Please make sure to replace {name of copyright owner} and {yyyy} with the correct values in your LICENSE file.",
		interceptor.Diagnostics[0].Detail,
	)
}

func TestValidBackendExecutable(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: filepath.Join("testdata", "mime"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestMimeTypeFiler(t *testing.T) {
	t.Run("text", func(t *testing.T) {
		f, err := filer.FromDirectory(filepath.Join("testdata", "mime"))
		require.NoError(t, err)
		f = newMimeTypeFiler(f, "text/")
		files, err := f.ReadDir(".")
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Equal(t, "LICENSE", files[0].Name)
	})

	t.Run("binary", func(t *testing.T) {
		f, err := filer.FromDirectory(filepath.Join("testdata", "mime"))
		require.NoError(t, err)
		f = newMimeTypeFiler(f, "application/octet-stream")
		files, err := f.ReadDir(".")
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Equal(t, "executable", files[0].Name)
	})
}
