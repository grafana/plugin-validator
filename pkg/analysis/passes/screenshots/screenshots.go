package screenshots

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var (
	screenshots       = &analysis.Rule{Name: "screenshots", Severity: analysis.Warning}
	screenshotsType   = &analysis.Rule{Name: "screenshots-image-type", Severity: analysis.Error}
	screenshotsFormat = &analysis.Rule{Name: "screenshots-format", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "screenshots",
	Run:      checkScreenshots,
	Requires: []*analysis.Analyzer{metadata.Analyzer, archive.Analyzer},
	Rules:    []*analysis.Rule{screenshots, screenshotsType, screenshotsFormat},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Screenshots",
		Description: "Screenshots are specified in `plugin.json` that will be used in the Grafana plugin catalog.",
	},
}

var svgImage = "image/svg+xml"
var acceptedImageTypes = []string{"image/jpeg", "image/png", "image/gif", svgImage}

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
		// Check if the error is related to screenshots field format
		if strings.Contains(err.Error(), "cannot unmarshal") && strings.Contains(err.Error(), "screenshots") {
			pass.ReportResult(pass.AnalyzerName, screenshotsFormat,
				"plugin.json: invalid screenshots format",
				"The screenshots field must be an array of objects with 'name' and 'path' properties, not an array of strings. Example: [{\"name\": \"Screenshot 1\", \"path\": \"img/screenshot.png\"}]")
			return nil, nil
		}
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
		} else if err := validateImage(filepath.Join(archiveDir, screenshot.Path)); err != nil {
			reportCount++
			logme.Debugln(err)
			pass.ReportResult(pass.AnalyzerName, screenshotsType, err.Error(), "The screenshot image is of an unsupported format.")
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

// We can use mimetype but it does too much for our case
// https://github.com/gabriel-vasile/mimetype/blob/master/internal/magic/text.go#L298
func checkSVG(raw []byte) bool {
	return bytes.Contains(raw, []byte("<svg"))
}

func validateImage(imgPath string) error {
	file, err := os.Open(imgPath)
	if err != nil {
		logme.DebugFln("cannot open file: %v", err)
		return fmt.Errorf("invalid screenshot path: %q", imgPath)
	}
	defer file.Close()

	// 512 is enough for getting the content type
	// https://pkg.go.dev/net/http#DetectContentType
	buffer := make([]byte, 512)
	// files less than 512 it will read all the file
	// won't throw errors
	if _, err := file.Read(buffer); err != nil {
		logme.DebugFln("cannot read file: %v", err)
		return fmt.Errorf("cannot read file: %v", err)
	}

	// returns text/plain or text/xml for svg files
	mimeType := http.DetectContentType(buffer)
	// logo.svg returns text/plain, valid.svg returns text/xml
	if (strings.Contains(mimeType, "text/plain") || strings.Contains(mimeType, "text/xml")) && checkSVG(buffer) {
		mimeType = svgImage
	}

	for _, accepted := range acceptedImageTypes {
		if accepted == mimeType {
			return nil
		}
	}

	return fmt.Errorf("invalid screenshot image: %q. Accepted image types: %q", imgPath, acceptedImageTypes)
}
