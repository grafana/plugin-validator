/**
This source code comes from here (it is not exported)
https://github.com/google/osv-scanner/pkg/lockfile/parse-pnmp-lock.go
*/

package lockfile

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type PnpmLockPackageResolution struct {
	Tarball string `yaml:"tarball"`
	Commit  string `yaml:"commit"`
	Repo    string `yaml:"repo"`
	Type    string `yaml:"type"`
}

type PnpmLockPackage struct {
	Resolution   PnpmLockPackageResolution `yaml:"resolution"`
	Name         string                    `yaml:"name"`
	Version      string                    `yaml:"version"`
	Dependencies map[string]string         `yaml:"dependencies"`
}

type PnpmLockfile struct {
	Version  float64                    `yaml:"lockfileVersion"`
	Packages map[string]PnpmLockPackage `yaml:"packages,omitempty"`
}

type pnpmLockfileV6 struct {
	Version  string                     `yaml:"lockfileVersion"`
	Packages map[string]PnpmLockPackage `yaml:"packages,omitempty"`
}

func (l *PnpmLockfile) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var lockfileV6 pnpmLockfileV6

	if err := unmarshal(&lockfileV6); err != nil {
		return err
	}

	parsedVersion, err := strconv.ParseFloat(lockfileV6.Version, 64)

	if err != nil {
		return err
	}

	l.Version = parsedVersion
	l.Packages = lockfileV6.Packages

	return nil
}

const PnpmEcosystem = NpmEcosystem

func startsWithNumber(str string) bool {
	matcher := CachedMustCompile(`^\d`)

	return matcher.MatchString(str)
}

// extractPnpmPackageNameAndVersion parses a dependency path, attempting to
// extract the name and version of the package it represents
func extractPnpmPackageNameAndVersion(dependencyPath string) (string, string) {
	// file dependencies must always have a name property to be installed,
	// and their dependency path never has the version encoded, so we can
	// skip trying to extract either from their dependency path
	if strings.HasPrefix(dependencyPath, "file:") {
		return "", ""
	}

	parts := strings.Split(dependencyPath, "/")
	var name string

	parts = parts[1:]
	if len(parts) > 0 {
		if strings.HasPrefix(parts[0], "@") {
			name = strings.Join(parts[:2], "/")
			parts = parts[2:]
		} else {
			name = parts[0]
			parts = parts[1:]
		}
	} else {
		return "", ""
	}

	version := ""

	if len(parts) != 0 {
		version = parts[0]
	}

	if version == "" {
		name, version = parseNameAtVersion(name)
	}

	if version == "" || !startsWithNumber(version) {
		return "", ""
	}

	underscoreIndex := strings.Index(version, "_")

	if underscoreIndex != -1 {
		version = strings.Split(version, "_")[0]
	}

	return name, version
}

func parseNameAtVersion(value string) (name string, version string) {
	// look for pattern "name@version", where name is allowed to contain zero or more "@"
	matches := CachedMustCompile(`^(.+)@([\d.]+)$`).FindStringSubmatch(value)

	if len(matches) != 3 {
		return name, ""
	}

	return matches[1], matches[2]
}

func parsePnpmLock(lockfile PnpmLockfile) []PackageDetails {
	packages := make([]PackageDetails, 0, len(lockfile.Packages))

	for s, pkg := range lockfile.Packages {
		dependencies := make([]Dependency, 0)

		name, version := extractPnpmPackageNameAndVersion(s)

		// "name" is only present if it's not in the dependency path and takes
		// priority over whatever name we think we've extracted (if any)
		if pkg.Name != "" {
			name = pkg.Name
		}

		// "version" is only present if it's not in the dependency path and takes
		// priority over whatever version we think we've extracted (if any)
		if pkg.Version != "" {
			version = pkg.Version
		}

		if name == "" || version == "" {
			continue
		}

		commit := pkg.Resolution.Commit

		if strings.HasPrefix(pkg.Resolution.Tarball, "https://codeload.github.com") {
			re := CachedMustCompile(`https://codeload\.github\.com(?:/[\w-.]+){2}/tar\.gz/(\w+)$`)
			matched := re.FindStringSubmatch(pkg.Resolution.Tarball)

			if matched != nil {
				commit = matched[1]
			}
		}

		for aDependency, dependencyVersion := range pkg.Dependencies {
			dependencies = append(dependencies, Dependency{
				Name:    aDependency,
				Version: dependencyVersion,
			})
		}

		packages = append(packages, PackageDetails{
			Name:         name,
			Version:      version,
			Ecosystem:    PnpmEcosystem,
			CompareAs:    PnpmEcosystem,
			Commit:       commit,
			Dependencies: dependencies,
		})
	}

	// sort the packages for consistent indexing
	sort.SliceStable(packages, func(i, j int) bool {
		if packages[i].Name == packages[j].Name {
			return packages[i].Version < packages[j].Version
		}
		return packages[i].Name < packages[j].Name
	})
	return packages
}

func ParsePnpmLock(pathToLockfile string) ([]PackageDetails, error) {
	var parsedLockfile *PnpmLockfile

	lockfileContents, err := os.ReadFile(pathToLockfile)

	if err != nil {
		return []PackageDetails{}, fmt.Errorf("could not read %s: %w", pathToLockfile, err)
	}

	err = yaml.Unmarshal(lockfileContents, &parsedLockfile)

	if err != nil {
		return []PackageDetails{}, fmt.Errorf("could not parse %s: %w", pathToLockfile, err)
	}

	return parsePnpmLock(*parsedLockfile), nil
}
