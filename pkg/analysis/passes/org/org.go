package org

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/grafana"
)

var (
	missingGrafanaCloudAccount = &analysis.Rule{Name: "missing-grafanacloud-account"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "org",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingGrafanaCloudAccount},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody := pass.ResultOf[metadata.Analyzer].([]byte)

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
		if err == grafana.ErrOrganizationNotFound {
			pass.Reportf(missingGrafanaCloudAccount, fmt.Sprintf("unregistered Grafana Cloud account: %s", username))
		} else if err == grafana.ErrPrivateOrganization {
			return nil, nil
		}
		return nil, err
	}

	return nil, nil
}
