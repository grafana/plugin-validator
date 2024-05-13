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
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/screenshots"
)

var (
	invalidPath            = &analysis.Rule{Name: "invalid-path", Severity: analysis.Error}
	pathRelativeToMetadata = &analysis.Rule{
		Name:     "path-relative-to-metadata",
		Severity: analysis.Error,
	}
	invalidRelativePath = &analysis.Rule{Name: "invalid-relative-path", Severity: analysis.Error}
	pathNotExists       = &analysis.Rule{Name: "path-not-exists", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name: "metadatapaths",
	Run:  checkMetadataPaths,
	Requires: []*analysis.Analyzer{
		screenshots.Analyzer,
		logos.Analyzer,
		nestedmetadata.Analyzer,
		archive.Analyzer,
	},
	Rules: []*analysis.Rule{
		invalidPath,
		pathRelativeToMetadata,
		invalidRelativePath,
		pathNotExists,
	},
}

type CheckPath struct {
	kind           string
	path           string
	relativeToPath string
}

func checkMetadataPaths(pass *analysis.Pass) (interface{}, error) {
	var paths []CheckPath

	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	metadatamap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	for currentPluginJson, meta := range metadatamap {
		relativeToPath := filepath.Join(archiveDir, filepath.Dir(currentPluginJson))
		for _, s := range meta.Info.Screenshots {
			paths = append(paths, CheckPath{
				kind:           "screenshot",
				path:           s.Path,
				relativeToPath: relativeToPath,
			})
		}

		// Logos
		paths = append(paths, CheckPath{
			kind:           "small logo",
			path:           meta.Info.Logos.Small,
			relativeToPath: relativeToPath,
		})

		paths = append(paths, CheckPath{
			kind:           "large logo",
			path:           meta.Info.Logos.Large,
			relativeToPath: relativeToPath,
		})
	}

	for _, path := range paths {
		u, err := url.Parse(path.path)
		if err != nil {
			pass.ReportResult(
				pass.AnalyzerName,
				invalidPath,
				fmt.Sprintf("plugin.json: invalid %s path: %s", path.kind, path.path),
				"The path doesn't exist",
			)
			continue
		} else {
			if invalidPath.ReportAll {
				invalidPath.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, invalidPath, fmt.Sprintf("plugin.json: valid %s path: %s", path.kind, path.path), "")
			}
		}

		if u.IsAbs() {
			pass.ReportResult(
				pass.AnalyzerName,
				pathRelativeToMetadata,
				fmt.Sprintf(
					"plugin.json: %s path should be relative to plugin.json: %s",
					path.kind,
					path.path,
				),
				"Don't use absolute paths inside plugin.json",
			)
			continue
		} else {
			if pathRelativeToMetadata.ReportAll {
				pathRelativeToMetadata.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, pathRelativeToMetadata, fmt.Sprintf("plugin.json: %s path is relative to plugin.json: %s", path.kind, path.path), "")
			}
		}

		if strings.HasPrefix(path.path, ".") || strings.HasPrefix(path.path, "/") {
			pass.ReportResult(
				pass.AnalyzerName,
				invalidRelativePath,
				fmt.Sprintf(
					"plugin.json: relative %s path should not start with '.' or '/': %s",
					path.kind,
					path.path,
				),
				"Write relative paths without leading '.' or '/'. e.g. Instead of './img/file.png' use 'img/file.png'",
			)
			continue
		} else {
			if invalidRelativePath.ReportAll {
				invalidRelativePath.Severity = analysis.OK
				pass.ReportResult(pass.AnalyzerName, invalidRelativePath, fmt.Sprintf("plugin.json: relative %s path does not start with '.' or '/': %s", path.kind, path.path), "")
			}
		}

		// validate path exists
		fullPath := filepath.Join(path.relativeToPath, path.path)
		_, err = os.Stat(fullPath)
		if err != nil {
			if os.IsNotExist(err) {
				relPathToDir, err := filepath.Rel(archiveDir, fullPath)
				if err != nil {
					return nil, err
				}
				pass.ReportResult(
					pass.AnalyzerName,
					pathNotExists,
					fmt.Sprintf(
						"plugin.json: %s path doesn't exists: %s",
						path.kind,
						relPathToDir,
					),
					"Refer only existing files. Make sure the files referred in plugin.json are included in the archive.",
				)
			}
		}
	}

	return nil, nil
}
