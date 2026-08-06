package osvscanner

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/osv-scanner/v2/pkg/models"
	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"

	"github.com/grafana/plugin-validator/pkg/logme"
)

const (
	grafanaSDKModulePath = "github.com/grafana/grafana-plugin-sdk-go"
	maxGoModSize         = 2 << 20
)

type goModuleRequirement struct {
	version  string
	indirect bool
}

var grafanaSDKGoModClient = &http.Client{Timeout: 10 * time.Second}

var fetchGrafanaSDKGoMod = func(version string) ([]byte, error) {
	escapedVersion, err := module.EscapeVersion(version)
	if err != nil {
		return nil, fmt.Errorf("escape Grafana SDK version: %w", err)
	}

	url := "https://proxy.golang.org/" + grafanaSDKModulePath + "/@v/" + escapedVersion + ".mod"
	response, err := grafanaSDKGoModClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch Grafana SDK go.mod: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch Grafana SDK go.mod: unexpected status %s", response.Status)
	}

	content, err := io.ReadAll(io.LimitReader(response.Body, maxGoModSize+1))
	if err != nil {
		return nil, fmt.Errorf("read Grafana SDK go.mod: %w", err)
	}
	if len(content) > maxGoModSize {
		return nil, fmt.Errorf("read Grafana SDK go.mod: response exceeds %d bytes", maxGoModSize)
	}
	return content, nil
}

func filterGoModResults(source models.VulnerabilityResults, goModPath string) models.VulnerabilityResults {
	pluginContent, err := os.ReadFile(goModPath)
	if err != nil {
		logme.DebugFln("could not read plugin go.mod for OSV filtering: %s", err)
		return source
	}
	pluginMod, err := modfile.Parse(goModPath, pluginContent, nil)
	if err != nil {
		logme.DebugFln("could not parse plugin go.mod for OSV filtering: %s", err)
		return source
	}

	sdkVersion, ok := grafanaSDKVersion(pluginMod)
	if !ok {
		return source
	}
	sdkContent, err := fetchGrafanaSDKGoMod(sdkVersion)
	if err != nil {
		logme.DebugFln("could not load Grafana SDK go.mod for OSV filtering: %s", err)
		return source
	}
	sdkMod, err := modfile.Parse("grafana-plugin-sdk-go.mod", sdkContent, nil)
	if err != nil {
		logme.DebugFln("could not parse Grafana SDK go.mod for OSV filtering: %s", err)
		return source
	}

	pluginRequirements := goModuleRequirements(pluginMod)
	sdkRequirements := goModuleRequirements(sdkMod)

	filtered := source
	filtered.Results = make([]models.PackageSource, len(source.Results))
	for resultIndex, result := range source.Results {
		filtered.Results[resultIndex] = result
		filtered.Results[resultIndex].Packages = nil

		for _, vulnerablePackage := range result.Packages {
			name := vulnerablePackage.Package.Name
			pluginRequirement, inPluginGoMod := pluginRequirements[name]
			sdkRequirement, inSDKGoMod := sdkRequirements[name]

			isSDKOwned := inPluginGoMod &&
				pluginRequirement.indirect &&
				inSDKGoMod &&
				sameGoModuleVersion(pluginRequirement.version, vulnerablePackage.Package.Version) &&
				sameGoModuleVersion(sdkRequirement.version, vulnerablePackage.Package.Version)
			if isSDKOwned {
				logme.DebugFln("excluded Grafana Go SDK dependency: %s@%s", name, vulnerablePackage.Package.Version)
				continue
			}
			filtered.Results[resultIndex].Packages = append(
				filtered.Results[resultIndex].Packages,
				vulnerablePackage,
			)
		}
	}
	return filtered
}

func grafanaSDKVersion(file *modfile.File) (string, bool) {
	for _, requirement := range file.Require {
		if requirement.Mod.Path != grafanaSDKModulePath {
			continue
		}
		version := requirement.Mod.Version
		for _, replacement := range file.Replace {
			if replacement.Old.Path != grafanaSDKModulePath ||
				(replacement.Old.Version != "" && replacement.Old.Version != version) {
				continue
			}
			if replacement.New.Path != grafanaSDKModulePath || replacement.New.Version == "" {
				return "", false
			}
			version = replacement.New.Version
		}
		return version, version != ""
	}
	return "", false
}

func goModuleRequirements(file *modfile.File) map[string]goModuleRequirement {
	requirements := make(map[string]goModuleRequirement, len(file.Require))
	for _, requirement := range file.Require {
		requirements[requirement.Mod.Path] = goModuleRequirement{
			version:  requirement.Mod.Version,
			indirect: requirement.Indirect,
		}
	}
	return requirements
}

func sameGoModuleVersion(left, right string) bool {
	return strings.TrimPrefix(left, "v") == strings.TrimPrefix(right, "v")
}
