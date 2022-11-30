package jargon

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)

	// should not call the report function
	assert.False(t, invoked)

}

func TestWithJargon(t *testing.T) {
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
			assert.Equal(t, "README.md contains developer jargon: (yarn)", d.Message)
		},
	}

	_, err = Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}
}
