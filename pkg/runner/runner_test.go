package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes"
)

var tests = []struct {
	Dir      string
	Messages []string
}{
	{Dir: "EmptyArchive", Messages: []string{
		"Archive is empty",
		"Code diff skipped due to errors in archive",
	}},
	{Dir: "EmptyDirectory", Messages: []string{
		"missing plugin.json",
		"missing module.js",
		"missing README.md",
		"unsigned plugin",
		"LICENSE file not found",
		"plugin.json not found",
		"missing CHANGELOG.md",
		"Code diff skipped due to errors in metadata",
	}},
	{Dir: "AllFilesPresentButEmpty", Messages: []string{
		"empty manifest",
		"README.md is empty", "plugin.json: should include screenshots for the Plugin catalog",
		"plugin.json: (root): type is required",
		"plugin.json: (root): name is required",
		"plugin.json: (root): id is required",
		"plugin.json: (root): info is required",
		"plugin.json: (root): dependencies is required",
		"plugin.json: invalid empty small logo path for plugin.json",
		"plugin.json: invalid empty large logo path for plugin.json",
		"LICENSE file not found",
		"Plugin version \"\" is invalid.",
		"plugin.json: description is empty",
		"plugin.json: keywords are empty",
		"CHANGELOG.md is empty",
		"You can include a sponsorship link if you want users to support your work",
		"plugin.json: plugin id should follow the format org-name-type",
		"Code diff skipped due to errors in metadatavalid",
	}},
}

func TestRunner(t *testing.T) {
	// create empty archive
	emptyArchive := filepath.Join("testdata", "EmptyArchive")
	if _, err := os.Stat(emptyArchive); os.IsNotExist(err) {
		if err = os.Mkdir(emptyArchive, 0755); err != nil {
			t.Fatal(err)
		}
	}
	// create empty dir
	emptyDir := filepath.Join("testdata", "EmptyDirectory", "myorg-plugin-panel")
	if _, err := os.Stat(emptyDir); os.IsNotExist(err) {
		if err = os.MkdirAll(emptyDir, 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, tt := range tests {
		t.Run(tt.Dir, func(t *testing.T) {
			archiveDir := filepath.Join("testdata", tt.Dir)

			ds, err := Check(passes.Analyzers,
				analysis.CheckParams{
					ArchiveDir:    archiveDir,
					SourceCodeDir: "",
					Checksum:      "",
				},
				Config{Global: GlobalConfig{Enabled: true}},
				analysis.Severity(""),
			)
			if err != nil {
				t.Fatal(err)
			}

			var diagnostics []string

			for name := range ds {
				for _, d := range ds[name] {
					diagnostics = append(diagnostics, d.Title)
				}
			}

			for _, w := range tt.Messages {
				if !contains(diagnostics, w) {
					t.Errorf("unreported diagnostic: %q", w)
				}
			}

			for _, d := range diagnostics {
				if !contains(tt.Messages, d) {
					t.Errorf("unexpected diagnostic: %q", d)
				}
			}
		})
	}
}

func contains(strs []string, str string) bool {
	for _, s := range strs {
		if s == str {
			return true
		}
	}
	return false
}

func TestRuleExceptions(t *testing.T) {
	rule1 := &analysis.Rule{Name: "rule1"}
	rule2 := &analysis.Rule{Name: "rule2"}

	analyzer := &analysis.Analyzer{
		Name: "testanalyzer",
		Rules: []*analysis.Rule{
			rule1,
			rule2,
		},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			pass.ReportResult(pass.AnalyzerName, rule1, "diagnostic from rule1", "")
			pass.ReportResult(pass.AnalyzerName, rule2, "diagnostic from rule2", "")
			return nil, nil
		},
	}

	config := Config{
		Global: GlobalConfig{Enabled: true},
		Analyzers: map[string]AnalyzerConfig{
			"testanalyzer": {
				Rules: map[string]RuleConfig{
					"rule1": {
						Exceptions: []string{"myorg-plugin-panel"},
					},
				},
			},
		},
	}

	pluginDir := filepath.Join("testdata", "RuleExceptions", "myorg-plugin-panel")

	ds, err := Check([]*analysis.Analyzer{analyzer},
		analysis.CheckParams{
			ArchiveDir: pluginDir,
		},
		config,
		analysis.Severity(""),
	)

	if err != nil {
		t.Fatal(err)
	}

	var diagnostics []string
	for name := range ds {
		for _, d := range ds[name] {
			diagnostics = append(diagnostics, d.Title)
		}
	}

	// rule1 should be skipped
	assert.NotContains(t, diagnostics, "diagnostic from rule1")
	// rule2 should be reported
	assert.Contains(t, diagnostics, "diagnostic from rule2")
}

func TestLinearDependencies(t *testing.T) {
	res := make(map[string]bool)
	first := &analysis.Analyzer{
		Name: "first",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["first"] = true
			return true, nil
		},
	}
	second := &analysis.Analyzer{
		Name:     "second",
		Requires: []*analysis.Analyzer{first},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["second"] = true
			return true, nil
		},
	}
	third := &analysis.Analyzer{
		Name:     "third",
		Requires: []*analysis.Analyzer{second},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["third"] = true
			return true, nil
		},
	}
	fourth := &analysis.Analyzer{
		Name:     "fourth",
		Requires: []*analysis.Analyzer{third},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["fourth"] = true
			return nil, nil
		},
	}

	_, _ = Check(
		[]*analysis.Analyzer{fourth},
		analysis.CheckParams{
			ArchiveDir:    "",
			SourceCodeDir: "",
			Checksum:      "",
		},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	if len(res) != 4 {
		t.Fatal("unexpected results")
	}
}

func TestSharedParent(t *testing.T) {
	res := make(map[string]bool)

	parent := &analysis.Analyzer{
		Name: "parent",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["parent"] = true
			return true, nil
		},
	}
	firstChild := &analysis.Analyzer{
		Name:     "firstChild",
		Requires: []*analysis.Analyzer{parent},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["firstChild"] = true
			return true, nil
		},
	}
	secondChild := &analysis.Analyzer{
		Name:     "secondChild",
		Requires: []*analysis.Analyzer{parent},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["secondChild"] = true
			return nil, nil
		},
	}

	_, _ = Check(
		[]*analysis.Analyzer{firstChild, secondChild},
		analysis.CheckParams{
			ArchiveDir:    "",
			SourceCodeDir: "",
			Checksum:      "",
		},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	if len(res) != 3 {
		t.Fatal("unexpected results")
	}
}

func TestCachedRun(t *testing.T) {
	res := make(map[string]bool)

	parent := &analysis.Analyzer{
		Name: "parent",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["parent"] = true
			return true, nil
		},
	}
	firstChild := &analysis.Analyzer{
		Name:     "firstChild",
		Requires: []*analysis.Analyzer{parent},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["firstChild"] = true
			return true, nil
		},
	}
	secondChild := &analysis.Analyzer{
		Name:     "secondChild",
		Requires: []*analysis.Analyzer{firstChild},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["secondChild"] = true
			return nil, nil
		},
	}

	_, _ = Check(
		[]*analysis.Analyzer{parent, firstChild, secondChild, firstChild, secondChild, parent},
		analysis.CheckParams{
			ArchiveDir:    "",
			SourceCodeDir: "",
			Checksum:      "",
		},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	if len(res) != 3 {
		t.Fatal("unexpected results", res)
	}
}

func TestDependencyReturnsNil(t *testing.T) {
	res := make(map[string]interface{})

	parent := &analysis.Analyzer{
		Name: "parent",
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["parent"] = nil
			return nil, nil
		},
	}
	firstChild := &analysis.Analyzer{
		Name:     "firstChild",
		Requires: []*analysis.Analyzer{parent},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			res["firstChild"] = true
			return true, nil
		},
	}
	_, _ = Check(
		[]*analysis.Analyzer{firstChild},
		analysis.CheckParams{
			ArchiveDir:    "",
			SourceCodeDir: "",
			Checksum:      "",
		},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	assert.Len(t, res, 2)
}

func TestGetAnalyzerDiagnostics(t *testing.T) {
	parentRule := &analysis.Rule{Name: "parent-rule", Severity: analysis.Error}
	childRule := &analysis.Rule{Name: "child-rule", Severity: analysis.Warning}

	parentAnalyzer := &analysis.Analyzer{
		Name:  "parent-analyzer",
		Rules: []*analysis.Rule{parentRule},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			pass.ReportResult(pass.AnalyzerName, parentRule, "parent error", "parent detail")
			return nil, nil
		},
	}

	childAnalyzer := &analysis.Analyzer{
		Name:     "child-analyzer",
		Requires: []*analysis.Analyzer{parentAnalyzer},
		Rules:    []*analysis.Rule{childRule},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			// Child can read parent's diagnostics
			parentDiagnostics := pass.GetAnalyzerDiagnostics(parentAnalyzer)
			assert.Len(
				t,
				parentDiagnostics,
				1,
				"child should see exactly one diagnostic from parent",
			)
			assert.Equal(t, "parent error", parentDiagnostics[0].Title)
			assert.Equal(t, "parent detail", parentDiagnostics[0].Detail)
			assert.Equal(t, analysis.Error, parentDiagnostics[0].Severity)

			pass.ReportResult(pass.AnalyzerName, childRule, "child warning", "child detail")
			return nil, nil
		},
	}

	diagnostics, err := Check(
		[]*analysis.Analyzer{childAnalyzer},
		analysis.CheckParams{},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	assert.NoError(t, err)
	assert.Len(t, diagnostics, 2)
	assert.Contains(t, diagnostics, "parent-analyzer")
	assert.Contains(t, diagnostics, "child-analyzer")
}

func TestGetAnalyzerDiagnosticsEmpty(t *testing.T) {
	parentRule := &analysis.Rule{Name: "parent-rule", Severity: analysis.Error}

	parentAnalyzer := &analysis.Analyzer{
		Name:  "parent-analyzer",
		Rules: []*analysis.Rule{parentRule},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			// Parent reports nothing
			return nil, nil
		},
	}

	childAnalyzer := &analysis.Analyzer{
		Name:     "child-analyzer",
		Requires: []*analysis.Analyzer{parentAnalyzer},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			parentDiagnostics := pass.GetAnalyzerDiagnostics(parentAnalyzer)
			assert.Empty(
				t,
				parentDiagnostics,
				"should return empty slice when analyzer reported nothing",
			)
			return nil, nil
		},
	}

	_, err := Check(
		[]*analysis.Analyzer{childAnalyzer},
		analysis.CheckParams{},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	assert.NoError(t, err)
}

func TestGetAnalyzerDiagnosticsMultipleRules(t *testing.T) {
	rule1 := &analysis.Rule{Name: "rule1", Severity: analysis.Error}
	rule2 := &analysis.Rule{Name: "rule2", Severity: analysis.Warning}
	otherRule := &analysis.Rule{Name: "other-rule", Severity: analysis.Error}

	analyzer1 := &analysis.Analyzer{
		Name:  "analyzer1",
		Rules: []*analysis.Rule{rule1, rule2},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			pass.ReportResult(pass.AnalyzerName, rule1, "error from analyzer1", "")
			pass.ReportResult(pass.AnalyzerName, rule2, "warning from analyzer1", "")
			return nil, nil
		},
	}

	analyzer2 := &analysis.Analyzer{
		Name:  "analyzer2",
		Rules: []*analysis.Rule{otherRule},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			pass.ReportResult(pass.AnalyzerName, otherRule, "error from analyzer2", "")
			return nil, nil
		},
	}

	reader := &analysis.Analyzer{
		Name:     "reader",
		Requires: []*analysis.Analyzer{analyzer1, analyzer2},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			analyzer1Diagnostics := pass.GetAnalyzerDiagnostics(analyzer1)
			analyzer2Diagnostics := pass.GetAnalyzerDiagnostics(analyzer2)

			assert.Len(t, analyzer1Diagnostics, 2, "should get all diagnostics from analyzer1")
			assert.Len(t, analyzer2Diagnostics, 1, "should get only diagnostics from analyzer2")
			assert.Equal(t, "error from analyzer2", analyzer2Diagnostics[0].Title)
			return nil, nil
		},
	}

	_, err := Check(
		[]*analysis.Analyzer{reader},
		analysis.CheckParams{},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	assert.NoError(t, err)
}

func TestGetAnalyzerDiagnosticsCheckForErrors(t *testing.T) {
	errorRule := &analysis.Rule{Name: "error-rule", Severity: analysis.Error}
	warningRule := &analysis.Rule{Name: "warning-rule", Severity: analysis.Warning}

	parentAnalyzer := &analysis.Analyzer{
		Name:  "parent-analyzer",
		Rules: []*analysis.Rule{errorRule, warningRule},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			pass.ReportResult(pass.AnalyzerName, errorRule, "this is an error", "")
			pass.ReportResult(pass.AnalyzerName, warningRule, "this is a warning", "")
			return nil, nil
		},
	}

	childAnalyzer := &analysis.Analyzer{
		Name:     "child-analyzer",
		Requires: []*analysis.Analyzer{parentAnalyzer},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			// Check if parent reported any errors
			parentDiagnostics := pass.GetAnalyzerDiagnostics(parentAnalyzer)
			hasErrors := false
			for _, d := range parentDiagnostics {
				if d.Severity == analysis.Error {
					hasErrors = true
					break
				}
			}
			assert.True(t, hasErrors, "child should detect that parent reported errors")
			return nil, nil
		},
	}

	_, err := Check(
		[]*analysis.Analyzer{childAnalyzer},
		analysis.CheckParams{},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	assert.NoError(t, err)
}

func TestAnalyzerHasErrors(t *testing.T) {
	errorRule := &analysis.Rule{Name: "error-rule", Severity: analysis.Error}
	warningRule := &analysis.Rule{Name: "warning-rule", Severity: analysis.Warning}

	analyzerWithErrors := &analysis.Analyzer{
		Name:  "analyzer-with-errors",
		Rules: []*analysis.Rule{errorRule, warningRule},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			pass.ReportResult(pass.AnalyzerName, errorRule, "this is an error", "")
			pass.ReportResult(pass.AnalyzerName, warningRule, "this is a warning", "")
			return nil, nil
		},
	}

	analyzerWithWarningsOnly := &analysis.Analyzer{
		Name:  "analyzer-warnings-only",
		Rules: []*analysis.Rule{warningRule},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			pass.ReportResult(pass.AnalyzerName, warningRule, "just a warning", "")
			return nil, nil
		},
	}

	childAnalyzer := &analysis.Analyzer{
		Name:     "child-analyzer",
		Requires: []*analysis.Analyzer{analyzerWithErrors, analyzerWithWarningsOnly},
		Run: func(pass *analysis.Pass) (interface{}, error) {
			assert.True(t, pass.AnalyzerHasErrors(analyzerWithErrors), "should detect errors")
			assert.False(
				t,
				pass.AnalyzerHasErrors(analyzerWithWarningsOnly),
				"should not report errors for warnings-only",
			)
			return nil, nil
		},
	}

	_, err := Check(
		[]*analysis.Analyzer{childAnalyzer},
		analysis.CheckParams{},
		Config{Global: GlobalConfig{Enabled: true}},
		analysis.Severity(""),
	)

	assert.NoError(t, err)
}
