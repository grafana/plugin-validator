package includesnested

import (
	"fmt"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
)

var (
	nestedPluginNotDeclared = &analysis.Rule{
		Name:     "nested-plugins-not-declared",
		Severity: analysis.Error,
	}
	nestedPluginMissingType = &analysis.Rule{
		Name:     "nested-plugin-missing-type",
		Severity: analysis.Error,
	}
	nestedPluginTypeMissmatch = &analysis.Rule{
		Name:     "nested-plugin-type-missmatch",
		Severity: analysis.Error,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "includesnested",
	Requires: []*analysis.Analyzer{archive.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		nestedPluginNotDeclared,
		nestedPluginMissingType,
		nestedPluginTypeMissmatch,
	},
}

func run(pass *analysis.Pass) (interface{}, error) {

	metadatamap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	if len(metadatamap) == 1 {
		// no nested plugins
		return nil, nil
	}

	// this should never happen and the nested validator should
	// have catched it before. Adding it to be safe
	if _, ok := metadatamap["plugin.json"]; !ok {
		return nil, nil
	}

	includes := metadatamap["plugin.json"].Includes

	for key := range metadatamap {
		if key == "plugin.json" {
			// skip main plugin.json
			continue
		}
		found := false
		for _, include := range includes {
			if include.Path == key {
				found = true

				if include.Type == "" {
					pass.ReportResult(
						pass.AnalyzerName,
						nestedPluginMissingType,
						fmt.Sprintf(
							"Nested plugin %s is missing type",
							key,
						),
						fmt.Sprintf(
							"Found a plugin %s declared in your main plugin.json without a type",
							key,
						),
					)
				} else if include.Type != metadatamap[key].Type {
					pass.ReportResult(
						pass.AnalyzerName,
						nestedPluginTypeMissmatch,
						fmt.Sprintf(
							"Nested plugin %s has a type missmatch",
							key,
						),
						fmt.Sprintf(
							"Plugin %s declared in your main plugin.json as %s but as %s in your main plugin.json",
							key,
							include.Type,
							metadatamap[key].Type,
						),
					)
				}
				continue
			}
		}
		if !found {
			pass.ReportResult(
				pass.AnalyzerName,
				nestedPluginNotDeclared,
				fmt.Sprintf(
					"Nested plugin %s is not declared in plugin main plugin.json",
					key,
				),
				fmt.Sprintf(
					"Found a plugin %s nested inside your archive but not declared in plugin.json",
					key,
				),
			)
		}
	}

	return nil, nil
}
