package grafanadependency

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGrafanaDependencyParse(t *testing.T) {
	for _, tc := range []struct {
		name       string
		dependency string
		expPre     string
	}{
		{"no pre-release", ">=12.4.0", ""},
		{"no pre-release with space", ">= 12.4.0", ""},
		{"zero pre-release", ">=12.4.0-0", "0"},
		{"zero pre-release with space", ">= 12.4.0-0", "0"},
		{"non-zero pre-release", ">=12.4.0-01189998819991197253", "01189998819991197253"},
		{"non-zero pre-release with space", ">= 12.4.0-01189998819991197253", "01189998819991197253"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pre := getPreRelease(">=12.4.0")
			require.Empty(t, pre)
		})
	}
}
