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
	}},
	{Dir: "EmptyDirectory", Messages: []string{
		"missing plugin.json",
		"missing module.js",
		"missing README.md",
		"unsigned plugin",
		"LICENSE file not found",
		"plugin.json not found",
		"missing CHANGELOG.md",
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
