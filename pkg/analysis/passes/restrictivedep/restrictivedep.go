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
	dependsOnPatchReleases = &analysis.Rule{Name: "depends-on-patch-releases"}
	dependsOnSingleRelease = &analysis.Rule{Name: "depends-on-single-release"}
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
		pass.Reportf(pass.AnalyzerName, dependsOnPatchReleases, fmt.Sprintf("plugin.json: plugin only targets patch releases of Grafana %s", version))
		return nil, nil
	}

	if regexp.MustCompile("^[0-9]+.[0-9]+.[0-9]+$").Match([]byte(data.Dependencies.GrafanaDependency)) {
		pass.Reportf(pass.AnalyzerName, dependsOnSingleRelease, fmt.Sprintf("plugin.json: plugin only targets Grafana %s", data.Dependencies.GrafanaDependency))
		return nil, nil
	}

	return nil, nil
}
