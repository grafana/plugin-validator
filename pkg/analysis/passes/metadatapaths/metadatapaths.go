package metadatapaths

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/logos"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/screenshots"
)

var (
	invalidPath            = &analysis.Rule{Name: "invalid-path"}
	pathRelativeToMetadata = &analysis.Rule{Name: "path-relative-to-metadata"}
	invalidRelativePath    = &analysis.Rule{Name: "invalid-relative-path"}
	pathNotExists          = &analysis.Rule{Name: "path-not-exists"}
)

var Analyzer = &analysis.Analyzer{
	Name:     "metadatapaths",
	Run:      checkMetadataPaths,
	Requires: []*analysis.Analyzer{screenshots.Analyzer, logos.Analyzer},
	Rules:    []*analysis.Rule{invalidPath, pathRelativeToMetadata, invalidRelativePath, pathNotExists},
}

func checkMetadataPaths(pass *analysis.Pass) (interface{}, error) {
	var paths []struct {
		kind string
		path string
	}

	// Screenshots
	screenshots, ok := pass.ResultOf[screenshots.Analyzer].([]metadata.MetadataScreenshots)
	// Will be nil if no screenshots were found.
	if ok {
		for _, s := range screenshots {
			paths = append(paths, struct {
				kind string
				path string
			}{
				kind: "screenshot",
				path: s.Path,
			})
		}
	}

	// Logos
	logos := pass.ResultOf[logos.Analyzer].(metadata.MetadataLogos)
	paths = append(paths, struct {
		kind string
		path string
	}{
		kind: "small logo",
		path: logos.Small,
	})
	paths = append(paths, struct {
		kind string
		path string
	}{
		kind: "large logo",
		path: logos.Large,
	})

	archiveDir := ""
	archiveAnalyserResult := pass.ResultOf[archive.Analyzer]
	if archiveAnalyserResult != nil {
		archiveDir = archiveAnalyserResult.(string)
	}

	for _, path := range paths {
		u, err := url.Parse(path.path)
		if err != nil {
			pass.ReportResult(pass.AnalyzerName, invalidPath, fmt.Sprintf("plugin.json: invalid %s path: %s", path.kind, path.path), "The path doesn't exist")
			continue
		} else {
			if invalidPath.ReportAll {
				invalidPath.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, invalidPath, fmt.Sprintf("plugin.json: valid %s path: %s", path.kind, path.path), "")
			}
		}

		if u.IsAbs() {
			pass.ReportResult(pass.AnalyzerName, pathRelativeToMetadata, fmt.Sprintf("plugin.json: %s path should be relative to plugin.json: %s", path.kind, path.path), "Don't use absolute paths inside plugin.json")
			continue
		} else {
			if pathRelativeToMetadata.ReportAll {
				pathRelativeToMetadata.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, pathRelativeToMetadata, fmt.Sprintf("plugin.json: %s path is relative to plugin.json: %s", path.kind, path.path), "")
			}
		}

		if strings.HasPrefix(path.path, ".") || strings.HasPrefix(path.path, "/") {
			pass.ReportResult(pass.AnalyzerName, invalidRelativePath, fmt.Sprintf("plugin.json: relative %s path should not start with '.' or '/': %s", path.kind, path.path), "Write relative paths without leading '.' or '/'. e.g. Instead of './img/file.png' use 'img/file.png'")
			continue
		} else {
			if invalidRelativePath.ReportAll {
				invalidRelativePath.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, invalidRelativePath, fmt.Sprintf("plugin.json: relative %s path does not start with '.' or '/': %s", path.kind, path.path), "")
			}
		}

		if archiveDir != "" {
			// validate path exists
			fullPath := filepath.Join(archiveDir, path.path)
			_, err = os.Stat(fullPath)
			if err != nil {
				if os.IsNotExist(err) {
					pass.ReportResult(pass.AnalyzerName, pathNotExists, fmt.Sprintf("plugin.json: %s path doesn't exists: %s", path.kind, path.path), "Refer only existing files. Make sure the files refered in plugin.json are included in the archive.")
				}
			}
		}
	}

	return nil, nil
}
