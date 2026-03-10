package metadata

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsGrafanaLabs(t *testing.T) {
	tests := []struct {
		name     string
		metadata Metadata
		expected bool
	}{
		{
			name:     "grafana org slug",
			metadata: Metadata{ID: "grafana-test-panel"},
			expected: true,
		},
		{
			name:     "grafana author name",
			metadata: Metadata{Info: Info{Author: Author{Name: "Grafana Labs"}}},
			expected: true,
		},
		{
			name:     "grafana author name case insensitive",
			metadata: Metadata{Info: Info{Author: Author{Name: "GRAFANA LABS"}}},
			expected: true,
		},
		{
			name:     "grafana org slug case insensitive",
			metadata: Metadata{ID: "Grafana-test-panel"},
			expected: true,
		},
		{
			name:     "non-grafana plugin",
			metadata: Metadata{ID: "myorg-test-panel", Info: Info{Author: Author{Name: "My Org"}}},
			expected: false,
		},
		{
			name:     "empty metadata",
			metadata: Metadata{},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, tc.metadata.IsGrafanaLabs())
		})
	}
}
