package htmlreadme

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
		},
		Report: func(d analysis.Diagnostic) {
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

func TestHTML(t *testing.T) {
	var invoked bool

	b, err := ioutil.ReadFile(filepath.Join("testdata", "README.html.md"))
	if err != nil {
		t.Fatal(err)
	}

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: b,
		},
		Report: func(d analysis.Diagnostic) {
			if d.Message != "README.md: html is not supported and will not render correctly" {
				t.Errorf("unexpected diagnostic message: %q", d.Message)
			}
			invoked = true
		},
	}

	res, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	if !invoked {
		t.Error("expected report, but got none")
	}

	if res != nil {
		t.Fatalf("unexpected result: %v", res)
	}
}
