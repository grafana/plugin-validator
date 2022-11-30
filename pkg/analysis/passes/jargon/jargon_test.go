package jargon

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

func TestClean(t *testing.T) {
	var invoked bool

	b, err := ioutil.ReadFile(filepath.Join("testdata", "README.clean.md"))
	if err != nil {
		t.Fatal(err)
	}

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: b,
			Analyzer:        nil,
		},
		Report: func(_ string, _ analysis.Diagnostic) {
			invoked = true
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	if invoked {
		t.Error("unexpected report")
	}
}

func TestWithJargon(t *testing.T) {
	var invoked bool

	b, err := ioutil.ReadFile(filepath.Join("testdata", "README.jargon.md"))
	if err != nil {
		t.Fatal(err)
	}

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: b,
			Analyzer:        nil,
		},
		Report: func(_ string, d analysis.Diagnostic) {
			if d.Message != "README.md contains developer jargon: (yarn)" {
				t.Errorf("unexpected diagnostic message: %q", d.Message)
			}
			invoked = true
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	if !invoked {
		t.Error("unexpected report")
	}
}
