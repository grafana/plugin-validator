package sdkusage

import (
	"os"
	"path/filepath"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/githubutils"
	"github.com/grafana/plugin-validator/pkg/logme"
	"golang.org/x/mod/modfile"
)

var (
	goSdkNotUsed            = &analysis.Rule{Name: "go-sdk-not-used", Severity: analysis.Error}
	goModNotFound           = &analysis.Rule{Name: "go-mod-not-found", Severity: analysis.Error}
	goModError              = &analysis.Rule{Name: "go-mod-error", Severity: analysis.Error}
	goSdkOlderThanTwoMonths = &analysis.Rule{
		Name:     "go-sdk-older-than-2-months",
		Severity: analysis.Warning,
	}
	goSdkOlderThanFiveMonths = &analysis.Rule{
		Name:     "go-sdk-older-than-5-months",
		Severity: analysis.Error,
	}
)

var twoMonths = 30 * 2
var fiveMonths = 30 * 5

var Analyzer = &analysis.Analyzer{
	Name:     "sdkusage",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer, nestedmetadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{goSdkNotUsed, goModNotFound},
}

func run(pass *analysis.Pass) (interface{}, error) {

	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		// no source code found so we can't go.mod
		return nil, nil
	}

	metadatamap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		return nil, nil
	}

	hasBackend := false
	for _, data := range metadatamap {
		if data.Backend {
			hasBackend = true
			break
		}
	}

	// skip plugins with no backend
	if !hasBackend {
		return nil, nil
	}

	goModPath := filepath.Join(sourceCodeDir, "go.mod")
	// check if go.mod exists
	if _, err := os.Stat(goModPath); err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			goModNotFound,
			"go.mod can not be found in your source code",
			"You have indicated your plugin uses a backend (backend=true), but go.mod can not be found in your source code. If your plugin has a backend component you must use go (golang)",
		)
		// go.mod not found
		return nil, nil
	}

	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			goModNotFound,
			"go.mod can not be read from your source code",
			"You have indicated your plugin uses a backend (backend=true), but go.mod can not be read from your source code. If your plugin has a backend component you must use go (golang)",
		)
		return nil, nil
	}

	goModParsed, err := modfile.Parse("go.mod", goModContent, nil)

	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			goModNotFound,
			"go.mod can not be parsed from your source code",
			"You have indicated your plugin uses a backend (backend=true), but go.mod can not be parsed from your source code. If your plugin has a backend component you must use go (golang)",
		)
		return nil, nil
	}

	pluginHasGoSdk := false
	pluginGoSdkVersion := ""

	for _, req := range goModParsed.Require {
		if req.Mod.Path == "github.com/grafana/grafana-plugin-sdk-go" {
			pluginHasGoSdk = true
			pluginGoSdkVersion = req.Mod.Version
		}
	}

	if !pluginHasGoSdk {
		pass.ReportResult(
			pass.AnalyzerName,
			goSdkNotUsed,
			"Your plugin uses a backend (backend=true), but the Grafana go sdk is not used",
			"If your plugin has a backend component you must use Grafana go sdk (github.com/grafana/grafana-plugin-sdk-go)",
		)
		return nil, nil
	}

	if pluginGoSdkVersion == "" {
		pass.ReportResult(
			pass.AnalyzerName,
			goModError,
			"go.mod can not be parsed from your source code",
			"Your go.mod can not be parsed. Please make sure it is valid You can use `go mod tidy` to fix it.",
		)
		return nil, nil
	}

	latestRelease, err := githubutils.FetchLatestGrafanaSdkRelease()
	if err != nil {
		// it is most likely this failed because of github auth or rate limits
		logme.Debugln(err)
		return nil, nil
	}

	if latestRelease.TagName == pluginGoSdkVersion {
		// plugin is using the latest version no further checks
		return nil, nil
	}

	pluginGoSdkRelease, err := githubutils.FetchGrafanaSdkReleaseByTag(pluginGoSdkVersion)
	if err != nil {
		// it is most likely this failed because of github auth or rate limits
		logme.Debugln(err)
		return nil, nil
	}

	daysDiff, err := daysDifference(pluginGoSdkRelease.PublishedAt, latestRelease.PublishedAt)
	if err != nil {
		// error calculating the days difference could be a problem in github date format
		// ignoring it
		return nil, nil
	}

	if daysDiff > fiveMonths {
		pass.ReportResult(
			pass.AnalyzerName,
			goSdkOlderThanFiveMonths,
			"Your Grafana go sdk is older than 5 months",
			"Your Grafana go sdk is older than 5 months. Please upgrade to the latest version",
		)
		return nil, nil
	}

	if daysDiff > twoMonths {
		pass.ReportResult(
			pass.AnalyzerName,
			goSdkOlderThanTwoMonths,
			"Your Grafana go sdk is older than 2 months",
			"Your Grafana go sdk is older than 2 months. Please upgrade to the latest version",
		)
		return nil, nil
	}

	return nil, nil
}

// expecting dates in RFC3339 format. e.g.: 2024-04-18T09:53:47Z
func daysDifference(date1 string, date2 string) (int, error) {
	// Parse the dates using the time package
	t1, err := time.Parse(time.RFC3339, date1)
	if err != nil {
		return 0, err
	}
	t2, err := time.Parse(time.RFC3339, date2)
	if err != nil {
		return 0, err
	}

	// Calculate the difference in days
	diff := t2.Sub(t1)
	days := int(diff.Hours() / 24)

	return days, nil
}
