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
	dependsOnPatchReleases = &analysis.Rule{
		Name:     "depends-on-patch-releases",
		Severity: analysis.Warning,
	}
	dependsOnSingleRelease = &analysis.Rule{
		Name:     "depends-on-single-release",
		Severity: analysis.Warning,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "restrictivedep",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{dependsOnPatchReleases, dependsOnSingleRelease},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Restrictive Dependency",
		Description: "Specifies a valid range of Grafana versions that work with this version of the plugin.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	md, ok := analysis.GetResult[[]byte](pass, metadata.Analyzer)
	if !ok {
		return nil, nil
	}

	var data struct {
		Dependencies struct {
			GrafanaDependency string `json:"grafanaDependency"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(md, &data); err != nil {
		return nil, err
	}

	if data.Dependencies.GrafanaDependency == "" {
		return nil, nil
	}

	if regexp.MustCompile("^[0-9]+.[0-9]+.x$").Match([]byte(data.Dependencies.GrafanaDependency)) {
		version := strings.TrimSuffix(data.Dependencies.GrafanaDependency, ".x")
		pass.ReportResult(
			pass.AnalyzerName,
			dependsOnPatchReleases,
			fmt.Sprintf(
				"plugin.json: grafanaDependency only targets patch releases of Grafana %s",
				version,
			),
			"The plugin will only work in patch releases of the specified minor Grafana version.",
		)
		return nil, nil
	}

	if regexp.MustCompile("^[0-9]+.[0-9]+.[0-9]+$").
		Match([]byte(data.Dependencies.GrafanaDependency)) {
		pass.ReportResult(
			pass.AnalyzerName,
			dependsOnSingleRelease,
			fmt.Sprintf(
				"plugin.json: grafanaDependency only targets Grafana %s",
				data.Dependencies.GrafanaDependency,
			),
			"The plugin will only work in the specific version of Grafana down to patch version.",
		)
		return nil, nil
	}

	return nil, nil
}
