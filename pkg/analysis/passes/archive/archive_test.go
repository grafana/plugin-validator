package archive

import (
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
)

func TestDistDir(t *testing.T) {
	var invoked bool

	pass := &analysis.Pass{
		RootDir:  filepath.Join("testdata", "DistDir"),
		ResultOf: make(map[*analysis.Analyzer]interface{}),
		Report: func(analysis.Diagnostic) {
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

	result := res.(string)

	if filepath.Base(result) != "dist" {
		t.Fatalf("unexpected archive dir: %q", result)
	}
}

func TestIDDir(t *testing.T) {
	var invoked bool

	pass := &analysis.Pass{
		RootDir:  filepath.Join("testdata", "IDDir"),
		ResultOf: make(map[*analysis.Analyzer]interface{}),
		Report: func(analysis.Diagnostic) {
			invoked = true
		},
	}

	res, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	if invoked {
		t.Error("unexpected report")
	}

	result := res.(string)

	if filepath.Base(result) != "myorgid-plugin-panel" {
		t.Fatalf("unexpected archive dir: %q", result)
	}
}

func TestEmpty(t *testing.T) {
	var invoked bool

	pass := &analysis.Pass{
		RootDir:  filepath.Join("testdata", "Empty"),
		ResultOf: make(map[*analysis.Analyzer]interface{}),
		Report: func(d analysis.Diagnostic) {
			invoked = true

			if d.Message != "archive does not contain a identifying directory" {
				t.Errorf("unexpected diagnostic message: %q", d.Message)
			}
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
