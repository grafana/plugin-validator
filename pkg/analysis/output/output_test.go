package output

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
)

func TestGHAOutput(t *testing.T) {
	for _, tc := range []struct {
		name  string
		diags analysis.Diagnostics
		exp   string
	}{
		{
			name: "error with title and details",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{
						Name:     "rule1",
						Severity: analysis.Error,
						Title:    "Test error",
						Detail:   "This is a test error detail",
					},
				},
			},
			exp: "::error title=plugin-validator: Error: Test error::This is a test error detail\n",
		},
		{
			name: "error with title and without details",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{
						Name:     "rule1",
						Severity: analysis.Error,
						Title:    "Test error",
					},
				},
			},
			exp: "::error title=plugin-validator: Error::Test error\n",
		},
		{
			name: "error without title and with details",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{
						Name:     "rule1",
						Severity: analysis.Error,
						Detail:   "This is a test error detail",
					},
				},
			},
			exp: "::error title=plugin-validator: Error::This is a test error detail\n",
		},
		{
			name: "warning",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{
						Name:     "rule1",
						Severity: analysis.Warning,
						Title:    "Test warning",
						Detail:   "This is a test warning detail",
					},
				},
			},
			exp: "::warning title=plugin-validator: Warning: Test warning::This is a test warning detail\n",
		},
		{
			name: "recommendation",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{
						Name:     "rule1",
						Severity: analysis.Recommendation,
						Title:    "Test recommendation",
						Detail:   "This is a test recommendation detail",
					},
				},
			},
			exp: "::notice title=plugin-validator: Recommendation: Test recommendation::This is a test recommendation detail\n",
		},
		{
			name: "ok debug",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{
						Name:     "rule1",
						Severity: analysis.OK,
						Title:    "Test ok",
						Detail:   "This is a test ok detail",
					},
				},
			},
			exp: "::debug::title=plugin-validator: OK: Test ok::This is a test ok detail\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := MarshalGHA(tc.diags)
			require.NoError(t, err)
			require.Equal(t, tc.exp, string(out))
		})
	}
}

func TestExitCode(t *testing.T) {
	for _, tc := range []struct {
		name   string
		diags  analysis.Diagnostics
		strict bool
		exp    int
	}{
		{name: "empty", diags: analysis.Diagnostics{}, exp: 0},
		{name: "empty strictr", diags: analysis.Diagnostics{}, strict: true, exp: 0},
		{
			name: "only ok",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{Severity: analysis.OK},
					{Severity: analysis.OK},
				},
			},
			exp: 0,
		},
		{
			name: "only ok strict",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{Severity: analysis.OK},
					{Severity: analysis.OK},
				},
			},
			strict: true,
			exp:    0,
		},
		{
			name: "only recommendation",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{Severity: analysis.Recommendation},
					{Severity: analysis.Recommendation},
				},
			},
			exp: 0,
		},
		{
			name: "only recommendation strict",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{Severity: analysis.Recommendation},
					{Severity: analysis.Recommendation},
				},
			},
			strict: true,
			exp:    0,
		},
		{
			name: "warning present not strict should exit with 0",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{Severity: analysis.OK},
					{Severity: analysis.Warning},
				},
			},
			exp: 0,
		},
		{
			name: "warning present strict should exit with 1",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{Severity: analysis.OK},
					{Severity: analysis.Warning},
				},
			},
			strict: true,
			exp:    1,
		},
		{
			name: "error present",
			diags: analysis.Diagnostics{
				"analyzer1": {
					{Severity: analysis.OK},
					{Severity: analysis.Error},
				},
			},
			exp: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			code := ExitCode(tc.strict, tc.diags)
			require.Equal(t, tc.exp, code)
		})
	}
}
