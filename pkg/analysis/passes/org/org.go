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
	missingGrafanaCloudAccount = &analysis.Rule{Name: "missing-grafanacloud-account", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "org",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingGrafanaCloudAccount},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
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
		if err == grafana.ErrOrganizationNotFound {
			pass.ReportResult(pass.AnalyzerName, missingGrafanaCloudAccount, fmt.Sprintf("unregistered Grafana Cloud account: %s", username), "The plugin's ID is prefixed with a Grafana Cloud account name, but that account does not exist. Please create the account or correct the name.")
		} else if err == grafana.ErrPrivateOrganization {
			return nil, nil
		}
		return nil, err
	} else {
		if missingGrafanaCloudAccount.ReportAll {
			missingGrafanaCloudAccount.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, missingGrafanaCloudAccount, fmt.Sprintf("found Grafana Cloud account: %s", username), "")
		}
	}

	return nil, nil
}
