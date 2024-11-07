package circulardependencies

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
)

var (
	circularDependency = &analysis.Rule{Name: "circular-dependency", Severity: analysis.Error}
)

// Analyzer checks for circular dependencies between plugins.
// It returns errors if a plugin has a dependency on itself or if there is a circular dependency between plugins.
// It supports dependencies on nested plugins, as well as with external (non-nested) plugins,
// whose dependencies are fetched from GCOM.
var Analyzer = &analysis.Analyzer{
	Name:     "circulardependencies",
	Requires: []*analysis.Analyzer{nestedmetadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{circularDependency},
}

func run(pass *analysis.Pass) (interface{}, error) {
	ctx, canc := context.WithTimeout(context.Background(), time.Minute*1)
	defer canc()

	rawMetadataMaps, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	// Convert from plugin_json_path -> metadata.Metadata
	// to metadata.ID -> metadata.Metadata
	metadataMaps := make(map[string]metadata.Metadata, len(rawMetadataMaps))
	for _, v := range rawMetadataMaps {
		metadataMaps[v.ID] = v
	}

	for _, md := range metadataMaps {
		for _, dep := range md.Dependencies.Plugins {
			// Short path if the plugin has a dependency on itself
			if dep.ID == md.ID {
				pass.ReportResult(
					pass.AnalyzerName,
					circularDependency,
					fmt.Sprintf("Circular dependency detected: %s", formatDependency(md.ID, md.ID)),
					"The plugin specifies itself as a dependency. This is not allowed.",
				)
				continue
			}

			var dependantDependencies []metadata.MetadataPluginDependency
			if nestedMd, ok := metadataMaps[dep.ID]; ok {
				// The dependency is a nested plugin.
				// Check if the nested plugin has a dependency on the current plugin.
				dependantDependencies = nestedMd.Dependencies.Plugins
			} else {
				// The dependency is not a nested plugin, get the dependencies from GCOM
				var err error
				version := dep.Version
				if version == "" {
					version = "latest"
				}
				dependantDependencies, err = getGCOMPluginDependencies(ctx, dep.ID, version)
				if err != nil {
					return nil, fmt.Errorf("get plugin version dependency from gcom: id=%q version=%q: %w", dep.ID, version, err)
				}
			}

			// Check for circular dependencies
			for _, dd := range dependantDependencies {
				if dd.ID != md.ID {
					continue
				}
				pass.ReportResult(
					pass.AnalyzerName,
					circularDependency,
					fmt.Sprintf("Circular dependency detected: %s", formatDependency(md.ID, dep.ID)),
					"Plugins cannot have circular dependencies.",
				)
			}
		}
	}
	return nil, nil
}

// getGCOMPluginDependencies fetches the dependencies of a plugin from GCOM.
func getGCOMPluginDependencies(ctx context.Context, pluginID string, version string) ([]metadata.MetadataPluginDependency, error) {
	url := fmt.Sprintf("https://grafana.com/api/plugins/%s/versions/%s", pluginID, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	var r struct {
		JSON metadata.Metadata `json:"json"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	return r.JSON.Dependencies.Plugins, nil
}

// formatDependency returns a formatted string representing a dependency between plugins.
func formatDependency(parent, dependency string) string {
	return fmt.Sprintf("%s -> %s", parent, dependency)
}
