package jargon

import (
	"testing"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/readme"
	"github.com/stretchr/testify/assert"
)

var cleanReadme = []byte(`
# Title

This is a README without any comment in it.
`)

var readmeWithComment = []byte(`
# Title

This is a README with comment in it.

<!-- hidden comment in markdown -->

`)

var readmeWithEmptyComment = []byte(`
# Title

This is a README with comment in it.

<!---->

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

func TestWithComment(t *testing.T) {

	var invoked bool

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: readmeWithComment,
			Analyzer:        nil,
		},
		Report: func(_ string, d analysis.Diagnostic) {
			invoked = true
			assert.Equal(t, "README.md contains comment(s).", d.Title)
		},
	}

	_, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, invoked, "report should be called")
}

func TestWithEmptyComment(t *testing.T) {

	var invoked bool

	pass := &analysis.Pass{
		ResultOf: map[*analysis.Analyzer]interface{}{
			readme.Analyzer: readmeWithEmptyComment,
			Analyzer:        nil,
		},
		Report: func(_ string, d analysis.Diagnostic) {
			invoked = true
			assert.Equal(t, "README.md contains comment(s).", d.Title)
		},
	}

	_, err := Analyzer.Run(pass)
	if err != nil {
		t.Fatal(err)
	}

	assert.True(t, invoked, "report should be called")
}
