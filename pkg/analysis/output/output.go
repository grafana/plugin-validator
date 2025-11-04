package output

import (
	"bytes"
	"encoding/json"

	"github.com/fatih/color"
	"github.com/grafana/plugin-validator/pkg/analysis"
)

// Marshaler is an interface for encoding the analysis results into bytes.
// Each implementation outputs the analysis results in a different format.
type Marshaler interface {
	// Marshal encodes the diagnostics data in the format implemented by the marshaler.
	Marshal(data analysis.Diagnostics) ([]byte, error)
}

// marshalerFunc is an adapter for using normal functions as Marshaler.
type marshalerFunc func(data analysis.Diagnostics) ([]byte, error)

// Marshal marshals the diagnostics data using the function.
func (f marshalerFunc) Marshal(data analysis.Diagnostics) ([]byte, error) {
	return f(data)
}

// jsonMarshaler is a Marshaler that outputs to JSON format.
type jsonMarshaler struct {
	// Additional fields used for JSON output

	// id is the plugin ID
	id string

	// version is the plugin version
	version string
}

// NewJSONMarshaler returns a new Marshaler that outputs the diagnostics data in JSON format.
// This marshaler requires additional plugin id and plugin version arguments.
func NewJSONMarshaler(id string, version string) Marshaler {
	return jsonMarshaler{id, version}
}

type jsonOutput struct {
	ID          string               `json:"id"`
	Version     string               `json:"version"`
	Diagnostics analysis.Diagnostics `json:"plugin-validator"`
}

// Marshal marshals the diagnostics data in JSON format.
// The additional id and version fields are taken from the marshaler itself.
func (j jsonMarshaler) Marshal(data analysis.Diagnostics) ([]byte, error) {
	return json.MarshalIndent(jsonOutput{
		ID:          j.id,
		Version:     j.version,
		Diagnostics: data,
	}, "", "  ")
}

// MarshalCLI is a Marshaler that returns the diagnostics data in a human-readable format, for CLI usage.
var MarshalCLI = marshalerFunc(func(data analysis.Diagnostics) ([]byte, error) {
	var buf bytes.Buffer
	for name := range data {
		for _, d := range data[name] {
			switch d.Severity {
			case analysis.Error:
				buf.WriteString(color.RedString("error: "))
			case analysis.Warning:
				buf.WriteString(color.YellowString("warning: "))
			case analysis.Recommendation:
				buf.WriteString(color.CyanString("recommendation: "))
			case analysis.OK:
				buf.WriteString(color.GreenString("ok: "))
			case analysis.SuspectedProblem:
				buf.WriteString(color.YellowString("suspected: "))
			}

			if d.Context != "" {
				buf.WriteString(d.Context + ": ")
			}

			buf.WriteString(d.Title)
			if len(d.Detail) > 0 {
				buf.WriteRune('\n')
				buf.WriteString(color.BlueString("detail: "))
				buf.WriteString(d.Detail)
			}
			buf.WriteRune('\n')
		}
	}
	return buf.Bytes(), nil
})

// ExitCode returns the exit code of the CLI program.
// It returns:
//   - 1 if there's an error;
//   - 1 if there's a warning AND strict is true;
//   - 0 in all other cases.
func ExitCode(strict bool, diags analysis.Diagnostics) int {
	for _, ds := range diags {
		for _, d := range ds {
			switch d.Severity {
			case analysis.Error:
				return 1
			case analysis.Warning:
				if strict {
					return 1
				}
			}
		}
	}
	return 0
}

// Static checks

var (
	_ = Marshaler(jsonMarshaler{})
	_ = Marshaler(MarshalCLI)
)
