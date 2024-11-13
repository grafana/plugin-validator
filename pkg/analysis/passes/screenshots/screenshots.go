package screenshots

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	screenshots     = &analysis.Rule{Name: "screenshots", Severity: analysis.Warning}
	screenshotsType = &analysis.Rule{Name: "screenshots-image-type", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "screenshots",
	Run:      checkScreenshots,
	Requires: []*analysis.Analyzer{metadata.Analyzer, archive.Analyzer},
	Rules:    []*analysis.Rule{screenshots, screenshotsType},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Screenshots",
		Description: "Screenshots are specified in `plugin.json` that will be used in the Grafana plugin catalog.",
	},
}
var acceptedImageTypes = []string{"image/jpeg", "image/png", "image/svg+xml", "image/gif"}

func checkScreenshots(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if len(data.Info.Screenshots) == 0 {
		explanation := "Screenshots are displayed in the Plugin catalog. Please add at least one screenshot to your plugin.json."
		pass.ReportResult(pass.AnalyzerName, screenshots, "plugin.json: should include screenshots for the Plugin catalog", explanation)
		return data.Info.Screenshots, nil
	}

	reportCount := 0
	for _, screenshot := range data.Info.Screenshots {
		if strings.TrimSpace(screenshot.Path) == "" {
			reportCount++
			pass.ReportResult(pass.AnalyzerName, screenshots, fmt.Sprintf("plugin.json: invalid empty screenshot path: %q", screenshot.Name), "The screenshot path must not be empty.")
		} else if !validImageType(filepath.Join(archiveDir, screenshot.Path)) {
			reportCount++
			pass.ReportResult(pass.AnalyzerName, screenshotsType, fmt.Sprintf("invalid screenshot image type: %q. Accepted image types: %q", screenshot.Path, acceptedImageTypes), "The screenshot image type invalid.")
		}
	}

	if reportCount > 0 {
		return nil, nil
	}

	if screenshots.ReportAll {
		screenshots.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, screenshots, "plugin.json: includes screenshots for the Plugin catalog", "")
	}

	if screenshotsType.ReportAll {
		screenshotsType.Severity = analysis.OK
		pass.ReportResult(pass.AnalyzerName, screenshotsType, "screenshots are valid image type", "")
	}

	return data.Info.Screenshots, nil
}

func validImageType(imgPath string) bool {
	file, err := os.Open(imgPath)
	if err != nil {
		fmt.Printf("cannot open file: %v\n", err)
		return false
	}
	defer file.Close()

	// 512 is enough for getting the content type
	// https://pkg.go.dev/net/http#DetectContentType
	buffer := make([]byte, 512)
	if _, err := file.Read(buffer); err != nil {
		fmt.Printf("cannot read file: %v\n", err)
		return false
	}

	mimeType := http.DetectContentType(buffer)

	for _, accepted := range acceptedImageTypes {
		if accepted == mimeType {
			return true
		}
	}

	return false
}
