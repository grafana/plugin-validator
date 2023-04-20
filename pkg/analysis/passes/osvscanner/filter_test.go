package osvscanner

import (
	"path/filepath"
	"testing"

	"github.com/google/osv-scanner/pkg/models"
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
