package semvercheck

import (
	"testing"

	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/require"
)

func TestDetermineVersionBumpType(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		new      string
		expected string
	}{
		{
			name:     "major bump",
			current:  "1.2.3",
			new:      "2.0.0",
			expected: "major",
		},
		{
			name:     "minor bump",
			current:  "1.2.3",
			new:      "1.3.0",
			expected: "minor",
		},
		{
			name:     "patch bump",
			current:  "1.2.3",
			new:      "1.2.4",
			expected: "patch",
		},
		{
			name:     "major bump from 0.x",
			current:  "0.9.9",
			new:      "1.0.0",
			expected: "major",
		},
		{
			name:     "no change",
			current:  "1.2.3",
			new:      "1.2.3",
			expected: "none",
		},
		{
			name:     "short version numbers",
			current:  "1.2",
			new:      "1.3",
			expected: "minor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current, err := version.NewVersion(tt.current)
			require.NoError(t, err)
			new, err := version.NewVersion(tt.new)
			require.NoError(t, err)

			result := determineVersionBumpType(current, new)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBreakingChanges(t *testing.T) {
	tests := []struct {
		name     string
		changes  []string
		expected string
	}{
		{
			name:     "empty list",
			changes:  []string{},
			expected: "No specific breaking changes identified.",
		},
		{
			name:     "single change",
			changes:  []string{"Removed deprecated API"},
			expected: "- Removed deprecated API",
		},
		{
			name:     "multiple changes",
			changes:  []string{"Removed deprecated API", "Changed function signature"},
			expected: "- Removed deprecated API\n- Changed function signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBreakingChanges(tt.changes)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatFeaturesList(t *testing.T) {
	tests := []struct {
		name     string
		features []string
		expected string
	}{
		{
			name:     "empty list",
			features: []string{},
			expected: "No specific new features identified.",
		},
		{
			name:     "single feature",
			features: []string{"Added new panel option"},
			expected: "- Added new panel option",
		},
		{
			name:     "multiple features",
			features: []string{"Added new panel option", "Added query caching"},
			expected: "- Added new panel option\n- Added query caching",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFeaturesList(tt.features)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePrompt(t *testing.T) {
	tests := []struct {
		name           string
		newVersion     string
		newCommit      string
		currentVersion string
		currentCommit  string
		expectError    bool
	}{
		{
			name:           "valid inputs",
			newVersion:     "2.0.0",
			newCommit:      "abc123",
			currentVersion: "1.0.0",
			currentCommit:  "def456",
			expectError:    false,
		},
		{
			name:           "empty new version",
			newVersion:     "",
			newCommit:      "abc123",
			currentVersion: "1.0.0",
			currentCommit:  "def456",
			expectError:    true,
		},
		{
			name:           "empty commit",
			newVersion:     "2.0.0",
			newCommit:      "",
			currentVersion: "1.0.0",
			currentCommit:  "def456",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := generatePrompt(tt.newVersion, tt.newCommit, tt.currentVersion, tt.currentCommit)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Contains(t, result, tt.newVersion)
				require.Contains(t, result, tt.currentVersion)
			}
		})
	}
}
