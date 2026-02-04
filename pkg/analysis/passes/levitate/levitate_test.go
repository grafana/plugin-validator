package levitate

import (
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMinVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "greater than or equal",
			input:    ">=9.0.0",
			expected: "9.0.0",
			wantErr:  false,
		},
		{
			name:     "range constraint",
			input:    ">=9.0.0 <11.0.0",
			expected: "9.0.0",
			wantErr:  false,
		},
		{
			name:     "x pattern",
			input:    "9.x",
			expected: "9.0.0",
			wantErr:  false,
		},
		{
			name:     "x pattern double digit",
			input:    "10.x",
			expected: "10.0.0",
			wantErr:  false,
		},
		{
			name:     "greater than or equal with spaces",
			input:    ">= 10.4.0",
			expected: "10.4.0",
			wantErr:  false,
		},
		{
			name:     "simple version",
			input:    "9.0.0",
			expected: "9.0.0",
			wantErr:  false,
		},
		{
			name:     "greater than",
			input:    ">9.0.0",
			expected: "9.0.1",
			wantErr:  false,
		},
		{
			name:     "tilde range",
			input:    "~9.5.0",
			expected: "9.5.0",
			wantErr:  false,
		},
		{
			name:     "caret range",
			input:    "^10.0.0",
			expected: "10.0.0",
			wantErr:  false,
		},
		{
			name:     "exact version with equals",
			input:    "=9.0.0",
			expected: "9.0.0",
			wantErr:  false,
		},
		{
			name:     "star pattern",
			input:    "9.*",
			expected: "9.0.0",
			wantErr:  false,
		},
		{
			name:     "star pattern double digit",
			input:    "11.*",
			expected: "11.0.0",
			wantErr:  false,
		},
		{
			name:    "invalid constraint",
			input:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMinVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, got)
			}
		})
	}
}

func TestParseIncompatibilities(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []incompatibility
	}{
		{
			name: "multiple incompatibilities",
			output: `
🔬 Checking compatibility between tests/fixtures/modern-panel-plugin/src/module.tsx and @grafana/data@10.0.0...

  INCOMPATIBILITIES (3)

1) Removed ` + "`FieldType.nestedFrames`" + ` used in ` + "`tests/fixtures/modern-panel-plugin/src/module.tsx:48`" + `
2) Removed ` + "`createDataFrame`" + ` used in ` + "`tests/fixtures/modern-panel-plugin/src/module.tsx:2`" + `
3) Changed ` + "`PanelPlugin.setPanelOptions`" + ` used in ` + "`tests/fixtures/modern-panel-plugin/src/module.tsx:48`" + `
`,
			expected: []incompatibility{
				{
					changeType: "Removed",
					apiName:    "FieldType.nestedFrames",
					location:   "tests/fixtures/modern-panel-plugin/src/module.tsx:48",
				},
				{
					changeType: "Removed",
					apiName:    "createDataFrame",
					location:   "tests/fixtures/modern-panel-plugin/src/module.tsx:2",
				},
				{
					changeType: "Changed",
					apiName:    "PanelPlugin.setPanelOptions",
					location:   "tests/fixtures/modern-panel-plugin/src/module.tsx:48",
				},
			},
		},
		{
			name: "single incompatibility",
			output: `
  INCOMPATIBILITIES (1)

1) Added ` + "`NewFeature`" + ` used in ` + "`src/module.tsx:10`" + `
`,
			expected: []incompatibility{
				{
					changeType: "Added",
					apiName:    "NewFeature",
					location:   "src/module.tsx:10",
				},
			},
		},
		{
			name:     "no incompatibilities",
			output:   "✔ Successfully compared versions\n\nNo incompatibilities found",
			expected: nil,
		},
		{
			name:     "empty output",
			output:   "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseIncompatibilities(tt.output)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuildTargetPackages(t *testing.T) {
	packages := []string{"@grafana/data", "@grafana/ui", "@grafana/runtime"}
	version := "9.0.0"

	result := buildTargetPackages(packages, version)

	expected := []string{
		"@grafana/data@9.0.0",
		"@grafana/ui@9.0.0",
		"@grafana/runtime@9.0.0",
	}

	assert.Equal(t, expected, result)
}

func TestBuildTargetPackagesEmpty(t *testing.T) {
	packages := []string{}
	version := "10.0.0"

	result := buildTargetPackages(packages, version)

	assert.Empty(t, result)
}

func TestBuildIncompatibilityReport(t *testing.T) {
	tests := []struct {
		name                  string
		incompat              incompatibility
		minVersion            string
		hasVersionMismatch    bool
		maxPackageVersion     string
		expectedTitleContains string
		expectedDetailParts   []string
	}{
		{
			name: "removed API with version mismatch",
			incompat: incompatibility{
				changeType: "Removed",
				apiName:    "DataQueryRequest.filters",
				location:   "src/module.tsx:48",
			},
			minVersion:         "10.0.0",
			hasVersionMismatch: true,
			maxPackageVersion:  "11.0.0",
			expectedTitleContains: "src/module.tsx:48: DataQueryRequest.filters requires newer Grafana version",
			expectedDetailParts: []string{
				"was added in a version newer than Grafana 10.0.0",
				"**Version Mismatch:**",
				"grafanaDependency: >=10.0.0",
				"package.json: @grafana packages at 11.0.0",
				"**Recommendation:** Update grafanaDependency",
			},
		},
		{
			name: "changed API with version mismatch",
			incompat: incompatibility{
				changeType: "Changed",
				apiName:    "PanelPlugin.setPanelOptions",
				location:   "src/module.tsx:25",
			},
			minVersion:         "9.0.0",
			hasVersionMismatch: true,
			maxPackageVersion:  "10.0.0",
			expectedTitleContains: "src/module.tsx:25: PanelPlugin.setPanelOptions has incompatible changes",
			expectedDetailParts: []string{
				"has breaking changes between Grafana 9.0.0 and 10.0.0",
				"**Version Mismatch:**",
				"**Recommendation:**",
			},
		},
		{
			name: "removed API without version mismatch",
			incompat: incompatibility{
				changeType: "Removed",
				apiName:    "OldAPI",
				location:   "src/plugin.tsx:10",
			},
			minVersion:         "11.0.0",
			hasVersionMismatch: false,
			expectedTitleContains: "src/plugin.tsx:10: OldAPI requires newer Grafana version",
			expectedDetailParts: []string{
				"requires newer Grafana version in Grafana 11.0.0",
			},
		},
		{
			name: "added API",
			incompat: incompatibility{
				changeType: "Added",
				apiName:    "NewFeature",
				location:   "src/module.tsx:5",
			},
			minVersion:         "10.0.0",
			hasVersionMismatch: true,
			maxPackageVersion:  "11.0.0",
			expectedTitleContains: "src/module.tsx:5: NewFeature not available in minimum version",
			expectedDetailParts: []string{
				"was added in a version newer than Grafana 10.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build version info
			info := &versionInfo{
				hasVersionMismatch: tt.hasVersionMismatch,
			}
			if tt.maxPackageVersion != "" {
				v, _ := semver.NewVersion(tt.maxPackageVersion)
				info.maxPackageVersion = v
			}

			title, detail := buildIncompatibilityReport(tt.incompat, tt.minVersion, info)

			// Check title
			assert.Equal(t, tt.expectedTitleContains, title)

			// Check detail contains expected parts
			for _, expectedPart := range tt.expectedDetailParts {
				assert.Contains(t, detail, expectedPart,
					"Detail should contain: %s\nGot: %s", expectedPart, detail)
			}

			// Verify markdown formatting is present when there's a version mismatch
			if tt.hasVersionMismatch {
				assert.Contains(t, detail, "**", "Detail should contain markdown bold formatting")
			}
		})
	}
}
