package govulncheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

// Sample drawn from real `govulncheck -json` output. NDJSON: one Message per line.
// Two distinct findings: GO-2024-AAAA is "called" (has a frame with a Position
// in user code), GO-2024-BBBB is module-level only (no Position). Only the
// first should be counted.
const sampleNDJSON = `
{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck","scanner_version":"v1.1.4","db":"https://vuln.go.dev","go_version":"go1.26.3","scan_level":"symbol"}}
{"progress":{"message":"Scanning your code and 42 packages across 3 dependent modules for known vulnerabilities..."}}
{"osv":{"id":"GO-2024-AAAA","summary":"Some vuln in pkg/foo"}}
{"osv":{"id":"GO-2024-BBBB","summary":"Module-only finding"}}
{"finding":{"osv":"GO-2024-AAAA","fixed_version":"v1.2.3","trace":[{"module":"example.com/foo","version":"v1.2.0","package":"example.com/foo","function":"Vulnerable","position":{"filename":"/src/plugin/main.go","line":42}},{"module":"example.com/foo","version":"v1.2.0","package":"example.com/foo","function":"main"}]}}
{"finding":{"osv":"GO-2024-BBBB","fixed_version":"v2.0.0","trace":[{"module":"example.com/bar","version":"v0.1.0"}]}}
`

func TestParseCalledFindings_OnlyCounts_Reachable(t *testing.T) {
	got, err := parseCalledFindings(strings.NewReader(sampleNDJSON))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 called finding, got %d (%v)", len(got), got)
	}
	if _, ok := got["GO-2024-AAAA"]; !ok {
		t.Fatalf("expected GO-2024-AAAA in called set, got %v", got)
	}
	if _, ok := got["GO-2024-BBBB"]; ok {
		t.Fatalf("GO-2024-BBBB is module-only and should not be counted")
	}
}

func TestParseCalledFindings_EmptyStream(t *testing.T) {
	got, err := parseCalledFindings(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 findings on empty input, got %d", len(got))
	}
}

func TestParseCalledFindings_DedupesSameOSV(t *testing.T) {
	// Two findings for the same OSV (different call sites) should collapse
	// into a single entry in the result set.
	const dup = `
{"finding":{"osv":"GO-2024-XXXX","trace":[{"package":"p","function":"A","position":{"filename":"a.go","line":1}}]}}
{"finding":{"osv":"GO-2024-XXXX","trace":[{"package":"p","function":"B","position":{"filename":"b.go","line":2}}]}}
`
	got, err := parseCalledFindings(strings.NewReader(dup))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 deduped OSV, got %d", len(got))
	}
}

func TestParseCalledFindings_CountsPositionInAnyTraceFrame(t *testing.T) {
	const positionInSecondFrame = `
{"finding":{"osv":"GO-2024-ORDER","trace":[{"package":"p","function":"Vulnerable"},{"package":"p","function":"main","position":{"filename":"main.go","line":12}}]}}
`
	got, err := parseCalledFindings(strings.NewReader(positionInSecondFrame))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if _, ok := got["GO-2024-ORDER"]; !ok {
		t.Fatalf("expected GO-2024-ORDER in called set, got %v", got)
	}
}

func TestParseCalledFindings_CapturesSummaryAndFixedVersion(t *testing.T) {
	got, err := parseCalledFindings(strings.NewReader(sampleNDJSON))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	info, ok := got["GO-2024-AAAA"]
	if !ok {
		t.Fatalf("expected GO-2024-AAAA in result")
	}
	if info.summary != "Some vuln in pkg/foo" {
		t.Errorf("expected summary %q, got %q", "Some vuln in pkg/foo", info.summary)
	}
	if info.fixedVersion != "v1.2.3" {
		t.Errorf("expected fixedVersion %q, got %q", "v1.2.3", info.fixedVersion)
	}
	if info.module != "example.com/foo" {
		t.Errorf("expected module %q, got %q", "example.com/foo", info.module)
	}
}

func TestRun_BinaryDetailContainsSummaryAndFixHint(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte(`#!/bin/sh
printf '{"osv":{"id":"GO-2024-BIN","summary":"dangerous syscall usage"}}\n'
printf '{"finding":{"osv":"GO-2024-BIN","fixed_version":"v2.0.0","trace":[{"module":"example.com/mod","version":"v1.2.3"}]}}\n'
`), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	archiveDir := t.TempDir()
	writeCurrentTestBinary(t, filepath.Join(archiveDir, "test-plugin_linux_amd64"))

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			archive.Analyzer: archiveDir,
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{Executable: "test-plugin"},
			},
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diagnostics))
	}
	detail := diagnostics[0].Detail
	if !strings.Contains(detail, "GO-2024-BIN") {
		t.Errorf("expected OSV ID in detail, got %q", detail)
	}
	if !strings.Contains(detail, "example.com/mod") {
		t.Errorf("expected module path in detail, got %q", detail)
	}
	if !strings.Contains(detail, "v2.0.0") {
		t.Errorf("expected fixed version in detail, got %q", detail)
	}
	if !strings.Contains(detail, "Update the following dependencies") {
		t.Errorf("expected dependencies section in detail, got %q", detail)
	}
}

func TestRun_SkipsSilentlyWhenGovulncheckNotInstalled(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf:     map[*analysis.Analyzer]any{},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %d (%v)", len(diagnostics), diagnostics)
	}
}

func TestRun_ReportsWhenGovulncheckNotInstalledWithReportAll(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	govulncheckNotInstalled.ReportAll = true
	defer func() {
		govulncheckNotInstalled.ReportAll = false
	}()

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf:     map[*analysis.Analyzer]any{},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%v)", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Name != govulncheckNotInstalled.Name {
		t.Fatalf("expected %q diagnostic, got %q", govulncheckNotInstalled.Name, diagnostics[0].Name)
	}
	if diagnostics[0].Severity != analysis.Warning {
		t.Fatalf("expected severity %q, got %q", analysis.Warning, diagnostics[0].Severity)
	}
}

func TestRun_ReportsScanFailureOnNonVulnerabilityExit(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte(`#!/bin/sh
printf '{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck"}}\n'
printf 'loading packages failed\n' >&2
exit 1
`), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "go.mod"), []byte("module example.com/plugin\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: sourceDir,
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%v)", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Name != govulncheckScanFailed.Name {
		t.Fatalf("expected %q diagnostic, got %q", govulncheckScanFailed.Name, diagnostics[0].Name)
	}
	if !strings.Contains(diagnostics[0].Detail, "loading packages failed") {
		t.Fatalf("expected stderr in detail, got %q", diagnostics[0].Detail)
	}
}

func TestRun_ScansBackendBinaries(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte(`#!/bin/sh
printf '{"finding":{"osv":"GO-2024-BIN","trace":[{"module":"example.com/mod","version":"v1.2.3"}]}}\n'
`), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	archiveDir := t.TempDir()
	binaryName := "test-plugin_linux_amd64"
	binaryPath := filepath.Join(archiveDir, binaryName)
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("find test binary: %v", err)
	}
	testBinaryData, err := os.ReadFile(testBinary)
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	if err := os.WriteFile(binaryPath, testBinaryData, 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			archive.Analyzer: archiveDir,
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{Executable: "test-plugin"},
			},
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%v)", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Name != govulncheckIssueFound.Name {
		t.Fatalf("expected %q diagnostic, got %q", govulncheckIssueFound.Name, diagnostics[0].Name)
	}
	if !strings.Contains(diagnostics[0].Title, "binary scan reports 1") {
		t.Fatalf("expected binary scan title, got %q", diagnostics[0].Title)
	}
	if !strings.Contains(diagnostics[0].Detail, "GO-2024-BIN") {
		t.Fatalf("expected OSV ID in detail, got %q", diagnostics[0].Detail)
	}
}

func TestRun_ScansBackendBinaryWithoutPlatformSuffix(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte(`#!/bin/sh
printf '{"finding":{"osv":"GO-2024-EXACT","trace":[{"module":"example.com/mod","version":"v1.2.3"}]}}\n'
`), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	archiveDir := t.TempDir()
	binaryName := "test-plugin"
	writeCurrentTestBinary(t, filepath.Join(archiveDir, binaryName))

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			archive.Analyzer: archiveDir,
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{Executable: "test-plugin"},
			},
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%v)", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Name != govulncheckIssueFound.Name {
		t.Fatalf("expected %q diagnostic, got %q", govulncheckIssueFound.Name, diagnostics[0].Name)
	}
	if !strings.Contains(diagnostics[0].Detail, "GO-2024-EXACT") {
		t.Fatalf("expected OSV ID in detail, got %q", diagnostics[0].Detail)
	}
}

func TestRun_ScansValidBackendBinaryWhenNonGoSiblingMatchesPrefix(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte(`#!/bin/sh
printf '{"finding":{"osv":"GO-2024-DECOY","trace":[{"module":"example.com/mod","version":"v1.2.3"}]}}\n'
`), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	archiveDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(archiveDir, "test-plugin.sha256"), []byte("not a Go binary"), 0o644); err != nil {
		t.Fatalf("write decoy binary sibling: %v", err)
	}
	binaryName := "test-plugin_linux_amd64"
	writeCurrentTestBinary(t, filepath.Join(archiveDir, binaryName))

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			archive.Analyzer: archiveDir,
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{Executable: "test-plugin"},
			},
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%v)", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Name != govulncheckIssueFound.Name {
		t.Fatalf("expected %q diagnostic, got %q", govulncheckIssueFound.Name, diagnostics[0].Name)
	}
	if !strings.Contains(diagnostics[0].Detail, "GO-2024-DECOY") {
		t.Fatalf("expected OSV ID in detail, got %q", diagnostics[0].Detail)
	}
}

func TestRun_SilentlySkipsSourceScanWhenSourceNotProvided(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{Backend: true, Executable: "test-plugin"},
			},
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics when source not provided, got %d (%v)", len(diagnostics), diagnostics)
	}
}

func TestRun_ScansNestedGoModuleInSource(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte(`#!/bin/sh
printf '{"finding":{"osv":"GO-2024-NESTED","trace":[{"package":"p","function":"A","position":{"filename":"main.go","line":1}}]}}\n'
`), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	sourceDir := t.TempDir()
	moduleDir := filepath.Join(sourceDir, "backend")
	if err := os.Mkdir(moduleDir, 0o755); err != nil {
		t.Fatalf("create module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module example.com/plugin\n\ngo 1.22\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			sourcecode.Analyzer: sourceDir,
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%v)", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Name != govulncheckIssueFound.Name {
		t.Fatalf("expected %q diagnostic, got %q", govulncheckIssueFound.Name, diagnostics[0].Name)
	}
	if !strings.Contains(diagnostics[0].Detail, "GO-2024-NESTED") {
		t.Fatalf("expected OSV in detail, got %q", diagnostics[0].Detail)
	}
}

func TestRun_ReportsScanFailureForNonGoBackendBinary(t *testing.T) {
	binDir := t.TempDir()
	fakeGovulncheck := filepath.Join(binDir, "govulncheck")
	err := os.WriteFile(fakeGovulncheck, []byte("#!/bin/sh\nexit 0\n"), 0o755)
	if err != nil {
		t.Fatalf("write fake govulncheck: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	archiveDir := t.TempDir()
	binaryName := "test-plugin_linux_amd64"
	binaryPath := filepath.Join(archiveDir, binaryName)
	if err := os.WriteFile(binaryPath, []byte{0x7f, 'E', 'L', 'F', 0x00}, 0o755); err != nil {
		t.Fatalf("write non-Go binary: %v", err)
	}

	var diagnostics []analysis.Diagnostic
	pass := &analysis.Pass{
		AnalyzerName: Analyzer.Name,
		ResultOf: map[*analysis.Analyzer]any{
			archive.Analyzer: archiveDir,
			nestedmetadata.Analyzer: nestedmetadata.Metadatamap{
				nestedmetadata.MainPluginJson: metadata.Metadata{Executable: "test-plugin"},
			},
		},
		Report: func(_ string, d analysis.Diagnostic) {
			diagnostics = append(diagnostics, d)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if len(diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d (%v)", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Name != govulncheckScanFailed.Name {
		t.Fatalf("expected %q diagnostic, got %q", govulncheckScanFailed.Name, diagnostics[0].Name)
	}
	if !strings.Contains(diagnostics[0].Detail, "is not a Go binary") {
		t.Fatalf("expected non-Go binary detail, got %q", diagnostics[0].Detail)
	}
}

func TestPluralY(t *testing.T) {
	if got := pluralY(1); got != "y" {
		t.Errorf("pluralY(1) = %q, want %q", got, "y")
	}
	if got := pluralY(2); got != "ies" {
		t.Errorf("pluralY(2) = %q, want %q", got, "ies")
	}
}

func writeCurrentTestBinary(t *testing.T, dst string) {
	t.Helper()

	testBinary, err := os.Executable()
	if err != nil {
		t.Fatalf("find test binary: %v", err)
	}
	testBinaryData, err := os.ReadFile(testBinary)
	if err != nil {
		t.Fatalf("read test binary: %v", err)
	}
	if err := os.WriteFile(dst, testBinaryData, 0o755); err != nil {
		t.Fatalf("write fake binary: %v", err)
	}
}
