package osvscanner

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestFilterGrafanaToolkit
func TestFilterGrafanaPackages(t *testing.T) {
	data := []byte(`
	{
  "results": [
		{
      "packages": [
        {
          "package": {
            "name": "d3-color"
          }
				},
        {
          "package": {
            "name": "moment"
          }
				}
			]
		}
		]
	}`)
	var packages OSVJsonOutput
	err := json.Unmarshal(data, &packages)
	require.NoError(t, err)

	filteredResults := FilterOSVResults(packages)
	require.Len(t, filteredResults.Results, 1) // should not have moment
	require.Equal(t, "d3-color", filteredResults.Results[0].Packages[0].Package.Name)
}
