package output

import (
	"bytes"
	"encoding/json"

	"github.com/fatih/color"
	"github.com/grafana/plugin-validator/pkg/analysis"
)

type Marshaler interface {
	Marshal(data analysis.Diagnostics) ([]byte, error)
}

type marshalerFunc func(data analysis.Diagnostics) ([]byte, error)

func (f marshalerFunc) Marshal(data analysis.Diagnostics) ([]byte, error) {
	return f(data)
}

type jsonMarshaler struct {
	id      string
	version string
}

func NewJSONMarshaler(id string, version string) Marshaler {
	return jsonMarshaler{id, version}
}

type jsonOutput struct {
	ID          string               `json:"id"`
	Version     string               `json:"version"`
	Diagnostics analysis.Diagnostics `json:"plugin-validator"`
}

func (j jsonMarshaler) Marshal(data analysis.Diagnostics) ([]byte, error) {
	return json.MarshalIndent(jsonOutput{
		ID:          j.id,
		Version:     j.version,
		Diagnostics: data,
	}, "", "  ")
}

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
