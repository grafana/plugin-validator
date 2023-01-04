package restrictivedep

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	dependsOnPatchReleases = &analysis.Rule{Name: "depends-on-patch-releases", Severity: analysis.Warning}
	dependsOnSingleRelease = &analysis.Rule{Name: "depends-on-single-release", Severity: analysis.Warning}
)

var Analyzer = &analysis.Analyzer{
	Name:     "restrictivedep",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{dependsOnPatchReleases, dependsOnSingleRelease},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadata := pass.ResultOf[metadata.Analyzer].([]byte)

	var data struct {
		Dependencies struct {
			GrafanaDependency string `json:"grafanaDependency"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(metadata, &data); err != nil {
		return nil, err
	}

	if data.Dependencies.GrafanaDependency == "" {
		return nil, nil
	}

	if regexp.MustCompile("^[0-9]+.[0-9]+.x$").Match([]byte(data.Dependencies.GrafanaDependency)) {
		version := strings.TrimSuffix(data.Dependencies.GrafanaDependency, ".x")
		pass.ReportResult(pass.AnalyzerName, dependsOnPatchReleases, fmt.Sprintf("plugin.json: grafanaDependency only targets patch releases of Grafana %s", version), "The plugin will only work in patch releases of the specified minor grafana version.")
		return nil, nil
	} else {
		if dependsOnPatchReleases.ReportAll {
			dependsOnPatchReleases.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, dependsOnPatchReleases, "plugin.json: grafanaDependency correctly targets patch releases of Grafana", "")
		}
	}

	if regexp.MustCompile("^[0-9]+.[0-9]+.[0-9]+$").Match([]byte(data.Dependencies.GrafanaDependency)) {
		pass.ReportResult(pass.AnalyzerName, dependsOnSingleRelease, fmt.Sprintf("plugin.json: grafanaDependency only targets Grafana %s", data.Dependencies.GrafanaDependency), "The plugin will only work in the specific version of grafana down to patch version.")
		return nil, nil
	} else {
		if dependsOnSingleRelease.ReportAll {
			dependsOnSingleRelease.Severity = analysis.OK
			pass.ReportResult(pass.AnalyzerName, dependsOnSingleRelease, "plugin.json: grafanaDependency does not target single release of Grafana", "")
		}
	}

	return nil, nil
}
