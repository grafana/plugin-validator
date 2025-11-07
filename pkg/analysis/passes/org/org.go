package org

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/grafana"
)

var (
	missingGrafanaCloudAccount = &analysis.Rule{
		Name:     "missing-grafanacloud-account",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "org",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingGrafanaCloudAccount},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Organization (exists)",
		Description: "Verifies the org specified in the plugin ID exists.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := analysis.GetResult[[]byte](pass, metadata.Analyzer)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	idParts := strings.Split(data.ID, "-")

	if len(idParts) == 0 {
		return nil, nil
	}

	username := idParts[0]
	if username == "" {
		return nil, nil
	}

	client := grafana.NewClient()

	_, err := client.FindOrgBySlug(username)
	if err != nil {
		if errors.Is(err, grafana.ErrOrganizationNotFound) {
			pass.ReportResult(
				pass.AnalyzerName,
				missingGrafanaCloudAccount,
				fmt.Sprintf("unregistered Grafana Cloud account: %s", username),
				"The plugin's ID is prefixed with a Grafana Cloud account name, but that account does not exist. Please create the account or correct the name.",
			)
		} else if errors.Is(err, grafana.ErrPrivateOrganization) {
			return nil, nil
		}
		return nil, err
	}

	return nil, nil
}
