package jargon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
)

var cleanReadme = []byte(`
# Test

This is a README without jargon in it.
`)

var jargonReadme = []byte(`
# Test

This is a README with jargon on it

# Development instructions

yarn install

`)

func TestClean(t *testing.T) {
	var invoked bool

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: cleanReadme,
			Analyzer:        nil,
		},
		Report: func(_ string, _ analysis.Diagnostic) {
			invoked = true
		},
	}

	_, err := Analyzer.Run(pass)
	assert.NoError(t, err)

	// should not call the report function
	assert.False(t, invoked)

}

func TestWithJargon(t *testing.T) {

	var invoked bool

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: jargonReadme,
			Analyzer:        nil,
		},
		Report: func(_ string, d analysis.Diagnostic) {
			invoked = true
			assert.Equal(t, "README.md contains developer jargon: (yarn)", d.Title)
		},
	}

	_, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, invoked, "report should be called")
}
