package virusscan

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func isClamAvInstalled() bool {
	clamavBin, err := exec.LookPath("clamscan")
	if err != nil {
		return false
	}
	if os.Getenv("SKIP_CLAMAV") != "" {
		return false
	}
	return clamavBin != ""
}
func TestValidArchiveAndSource(t *testing.T) {

	if !isClamAvInstalled() {
		t.Skip("clamav not installed, skipping test")
		return
	}

	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "valid"),
			sourcecode.Analyzer: filepath.Join("testdata", "valid"),
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestInvalidArchive(t *testing.T) {
	if !isClamAvInstalled() {
		t.Skip("clamav not installed, skipping test")
		return
	}

	var interceptor testpassinterceptor.TestPassInterceptor

	var invalidLocation = filepath.Join("testdata", "invalid")

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer: invalidLocation,
		},
		Report: interceptor.ReportInterceptor(),
	}

	cleanup, err := writeTestEicarFile(invalidLocation)
	require.NoError(t, err)
	defer cleanup()

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Title,
		"ClamAV found 1 infected file(s) inside your archive",
	)
	require.Contains(t, interceptor.Diagnostics[0].Detail, "EICAR-test-file")
	require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Error)

}

func TestInvalidSource(t *testing.T) {
	if !isClamAvInstalled() {
		t.Skip("clamav not installed, skipping test")
		return
	}

	var interceptor testpassinterceptor.TestPassInterceptor

	var invalidLocation = filepath.Join("testdata", "invalid")

	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			archive.Analyzer:    filepath.Join("testdata", "valid"),
			sourcecode.Analyzer: invalidLocation,
		},
		Report: interceptor.ReportInterceptor(),
	}

	cleanup, err := writeTestEicarFile(invalidLocation)
	require.NoError(t, err)
	defer cleanup()

	_, err = Analyzer.Run(pass)
	require.NoError(t, err)

	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(
		t,
		interceptor.Diagnostics[0].Title,
		"ClamAV found 1 infected file(s) inside your source code",
	)
	require.Contains(t, interceptor.Diagnostics[0].Detail, "EICAR-test-file")
	require.Equal(t, interceptor.Diagnostics[0].Severity, analysis.Error)

}

// we don't want to store malicious files (even if test ones)
// in our repository so we are building an EICAR test file manually
// nor we want this file to be flagged as virus
func writeTestEicarFile(location string) (func(), error) {
	content := "X5O!P%@AP[4\\PZX54(P^)7CC)7}$EICAR"
	content = content + "-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*"
	file, err := os.Create(filepath.Join(location, "EICAR-test-file"))
	if err != nil {
		return nil, err
	}
	_, err = file.WriteString(content)
	if err != nil {
		return nil, err
	}
	return func() { os.Remove(filepath.Join(location, "EICAR-test-file")) }, nil
}
