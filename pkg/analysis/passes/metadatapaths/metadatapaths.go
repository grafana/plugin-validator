package metadatapaths

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/logos"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/screenshots"
)

var (
	invalidPath            = &analysis.Rule{Name: "invalid-path"}
	pathRelativeToMetadata = &analysis.Rule{Name: "path-relative-to-metadata"}
	invalidRelativePath    = &analysis.Rule{Name: "invalid-relative-path"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "metadatapaths",
	Run:      checkMetadataPaths,
	Requires: []*analysis.Analyzer{screenshots.Analyzer, logos.Analyzer},
	Rules:    []*analysis.Rule{invalidPath, pathRelativeToMetadata, invalidRelativePath},
}

func checkMetadataPaths(pass *analysis.Pass) (interface{}, error) {
	var paths []string

	// Screenshots
	screenshots, ok := pass.ResultOf[screenshots.Analyzer].([]metadata.MetadataScreenshots)
	// Will be nil if no screenshots were found.
	if ok {
		for _, s := range screenshots {
			paths = append(paths, s.Path)
		}
	}

	// Logos
	logos := pass.ResultOf[logos.Analyzer].(metadata.MetadataLogos)
	paths = append(paths, logos.Small)
	paths = append(paths, logos.Large)

	for _, path := range paths {
		u, err := url.Parse(path)
		if err != nil {
			pass.Reportf(pass.AnalyzerName, invalidPath, fmt.Sprintf("plugin.json: invalid path: %s", path))
			continue
		} else {
			if invalidPath.ReportAll {
				invalidPath.Severity = analysis.OK
				pass.Reportf(pass.AnalyzerName, invalidPath, fmt.Sprintf("plugin.json: valid path: %s", path))
			}
		}

		if u.IsAbs() {
			pass.Reportf(pass.AnalyzerName, pathRelativeToMetadata, fmt.Sprintf("plugin.json: path should be relative to plugin.json: %s", path))
			continue
		} else {
			if pathRelativeToMetadata.ReportAll {
				pathRelativeToMetadata.Severity = analysis.OK
				pass.Reportf(pass.AnalyzerName, pathRelativeToMetadata, fmt.Sprintf("plugin.json: path is relative to plugin.json: %s", path))
			}
		}

		if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
			pass.Reportf(pass.AnalyzerName, invalidRelativePath, fmt.Sprintf("plugin.json: relative path should not start with '.' or '/': %s", path))
			continue
		} else {
			if invalidRelativePath.ReportAll {
				invalidRelativePath.Severity = analysis.OK
				pass.Reportf(pass.AnalyzerName, invalidRelativePath, fmt.Sprintf("plugin.json: relative path does not start with '.' or '/': %s", path))
			}
		}
	}

	return nil, nil
}
