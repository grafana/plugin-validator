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

var Analyzer = &analysis.Analyzer{
	Name:     "metadatapaths",
	Run:      checkMetadataPaths,
	Requires: []*analysis.Analyzer{screenshots.Analyzer, logos.Analyzer},
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
			pass.Report(analysis.Diagnostic{
				Severity: analysis.Error,
				Message:  fmt.Sprintf("invalid path: %s", path),
				Context:  "plugin.json",
			})
			continue
		}

		if u.IsAbs() {
			pass.Report(analysis.Diagnostic{
				Severity: analysis.Error,
				Message:  fmt.Sprintf("path should be relative to plugin.json: %s", path),
				Context:  "plugin.json",
			})
			continue
		}

		if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") {
			pass.Report(analysis.Diagnostic{
				Severity: analysis.Error,
				Message:  fmt.Sprintf("relative path should not start with '.' or '/': %s", path),
				Context:  "plugin.json",
			})
			continue
		}
	}

	return nil, nil
}
