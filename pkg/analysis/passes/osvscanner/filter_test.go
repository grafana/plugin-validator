package osvscanner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/osv-scanner/v2/pkg/models"
	"github.com/stretchr/testify/require"
)

func TestFilterPackages(t *testing.T) {

	packages := models.VulnerabilityResults{
		Results: []models.PackageSource{
			{
				Source: models.SourceInfo{
					Path: "d3-color",
					Type: "lockfile",
				},
				Packages: []models.PackageVulns{},
			},
			{
				Source: models.SourceInfo{
					Path: "moment",
					Type: "lockfile",
				},
				Packages: []models.PackageVulns{},
			},
		},
	}

	filteredResults := FilterOSVResults(packages, filepath.Join("testdata", "node", "critical-yarn", "yarn.lock"))
	// should not have moment
	require.Len(t, filteredResults.Results, 1)
	require.Equal(t, "d3-color", filteredResults.Results[0].Source.Path)
}

func TestFilterGoModPackages(t *testing.T) {
	goMod := `module example.com/plugin

go 1.22

require (
	github.com/grafana/grafana-plugin-sdk-go v0.250.0
	github.com/getkin/kin-openapi v0.124.0 // indirect
	github.com/plugin/direct v1.0.0
	github.com/shared/direct v1.2.0
	github.com/unrelated/indirect v1.0.0 // indirect
	github.com/versioned/indirect v1.1.0 // indirect
)
`
	sdkGoMod := []byte(`module github.com/grafana/grafana-plugin-sdk-go

go 1.21

require (
	github.com/getkin/kin-openapi v0.124.0
	github.com/shared/direct v1.2.0 // indirect
	github.com/versioned/indirect v1.0.0 // indirect
)
`)

	goModPath := filepath.Join(t.TempDir(), "go.mod")
	require.NoError(t, os.WriteFile(goModPath, []byte(goMod), 0o600))

	actualFetch := fetchGrafanaSDKGoMod
	t.Cleanup(func() { fetchGrafanaSDKGoMod = actualFetch })
	fetchGrafanaSDKGoMod = func(version string) ([]byte, error) {
		require.Equal(t, "v0.250.0", version)
		return sdkGoMod, nil
	}

	source := vulnerabilityResults(
		"github.com/getkin/kin-openapi", "v0.124.0",
		"github.com/plugin/direct", "v1.0.0",
		"github.com/shared/direct", "v1.2.0",
		"github.com/unrelated/indirect", "v1.0.0",
		"github.com/versioned/indirect", "v1.1.0",
	)

	filtered := FilterOSVResults(source, goModPath)

	require.Equal(t, []string{
		"github.com/plugin/direct",
		"github.com/shared/direct",
		"github.com/unrelated/indirect",
		"github.com/versioned/indirect",
	}, packageNames(filtered))
}

func TestFilterGoModPackagesFailsOpen(t *testing.T) {
	goMod := `module example.com/plugin

go 1.22

require (
	github.com/grafana/grafana-plugin-sdk-go v0.250.0
	github.com/getkin/kin-openapi v0.124.0 // indirect
)
`
	goModPath := filepath.Join(t.TempDir(), "go.mod")
	require.NoError(t, os.WriteFile(goModPath, []byte(goMod), 0o600))

	actualFetch := fetchGrafanaSDKGoMod
	t.Cleanup(func() { fetchGrafanaSDKGoMod = actualFetch })
	fetchGrafanaSDKGoMod = func(string) ([]byte, error) {
		return nil, errors.New("module proxy unavailable")
	}

	source := vulnerabilityResults("github.com/getkin/kin-openapi", "v0.124.0")

	require.Equal(t, source, FilterOSVResults(source, goModPath))
}

func vulnerabilityResults(packages ...string) models.VulnerabilityResults {
	result := models.VulnerabilityResults{
		Results: []models.PackageSource{{
			Source: models.SourceInfo{Path: "go.mod", Type: "lockfile"},
		}},
	}
	for i := 0; i < len(packages); i += 2 {
		result.Results[0].Packages = append(result.Results[0].Packages, models.PackageVulns{
			Package: models.PackageInfo{Name: packages[i], Version: packages[i+1]},
		})
	}
	return result
}

func packageNames(result models.VulnerabilityResults) []string {
	var names []string
	for _, source := range result.Results {
		for _, pkg := range source.Packages {
			names = append(names, pkg.Package.Name)
		}
	}
	return names
}
