/**
This source code comes from here (it is not exported)
https://github.com/google/osv-scanner/pkg/lockfile/parse-yarn-lock.go
*/

package lockfile

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
)

func shouldSkipYarnLine(line string) bool {
	return line == "" || strings.HasPrefix(line, "#")
}

func groupYarnPackageLines(scanner *bufio.Scanner) [][]string {
	var groups [][]string
	var group []string

	for scanner.Scan() {
		line := scanner.Text()

		if shouldSkipYarnLine(line) {
			continue
		}

		// represents the start of a new dependency
		if !strings.HasPrefix(line, " ") {
			if len(group) > 0 {
				groups = append(groups, group)
			}
			group = make([]string, 0)
		}

		group = append(group, line)
	}

	if len(group) > 0 {
		groups = append(groups, group)
	}

	return groups
}

func extractYarnPackageName(str string) string {
	str = strings.TrimPrefix(str, "\"")

	isScoped := strings.HasPrefix(str, "@")

	if isScoped {
		str = strings.TrimPrefix(str, "@")
	}

	name := strings.SplitN(str, "@", 2)[0]

	if isScoped {
		name = "@" + name
	}

	return name
}

func extractYarnPackageDependencies(group []string) []Dependency {
	dependencies := make([]Dependency, 0)
	re := regexp.MustCompile(`^  dependencies:$`)
	offset := 0
	for i, s := range group {
		matched := re.FindStringSubmatch(s)
		if matched != nil {
			// found block of dependencies, check the next section
			offset = i + 1
			break
		}
	}
	// no match
	if offset == 0 {
		return dependencies
	}
	// starting from the index where the dependencies block was found, parse out the packages
	for i := offset; i < len(group); i++ {
		line := group[i]
		if strings.HasPrefix(line, "    ") {
			// this is a dependency
			line = strings.Trim(line, " ")
			array := strings.Split(line, " ")
			if len(array) == 2 {
				name := strings.TrimRight(array[0], ":\"")
				name = strings.TrimLeft(name, "\"")
				dependencies = append(dependencies, Dependency{
					Name:    name,
					Version: strings.Trim(array[1], "\""),
				})
			}
		} else {
			// end of dependencies
			break
		}
	}
	return dependencies
}

func determineYarnPackageVersion(group []string) string {
	re := regexp.MustCompile(`^ {2}version:? "?([\w-.]+)"?$`)

	for _, s := range group {
		matched := re.FindStringSubmatch(s)

		if matched != nil {
			return matched[1]
		}
	}

	// todo: decide what to do here - maybe panic...?
	return ""
}

func determineYarnPackageResolution(group []string) string {
	re := regexp.MustCompile(`^ {2}(?:resolution:|resolved) "([^ '"]+)"$`)

	for _, s := range group {
		matched := re.FindStringSubmatch(s)

		if matched != nil {
			return matched[1]
		}
	}

	// todo: decide what to do here - maybe panic...?
	return ""
}

func tryExtractCommit(resolution string) string {
	// language=GoRegExp
	matchers := []string{
		// ssh://...
		// git://...
		// git+ssh://...
		// git+https://...
		`(?:^|.+@)(?:git(?:\+(?:ssh|https))?|ssh)://.+#(\w+)$`,
		// https://....git/...
		`(?:^|.+@)https://.+\.git#(\w+)$`,
		`https://codeload\.github\.com(?:/[\w-.]+){2}/tar\.gz/(\w+)$`,
		`.+#commit[:=](\w+)$`,
		// github:...
		// gitlab:...
		// bitbucket:...
		`^(?:github|gitlab|bitbucket):.+#(\w+)$`,
	}

	for _, matcher := range matchers {
		re := regexp.MustCompile(matcher)
		matched := re.FindStringSubmatch(resolution)

		if matched != nil {
			return matched[1]
		}
	}

	u, err := url.Parse(resolution)

	if err == nil {
		gitRepoHosts := []string{
			"bitbucket.org",
			"github.com",
			"gitlab.com",
		}

		for _, host := range gitRepoHosts {
			if u.Host != host {
				continue
			}

			if u.RawQuery != "" {
				queries := u.Query()

				if queries.Has("ref") {
					return queries.Get("ref")
				}
			}

			return u.Fragment
		}
	}

	return ""
}

func parseYarnPackageGroup(group []string) PackageDetails {
	name := extractYarnPackageName(group[0])
	version := determineYarnPackageVersion(group)
	resolution := determineYarnPackageResolution(group)
	dependencies := extractYarnPackageDependencies(group)

	if version == "" {
		_, _ = fmt.Fprintf(
			os.Stderr,
			"Failed to determine version of %s while parsing a yarn.lock - please report this!\n",
			name,
		)
	}

	return PackageDetails{
		Name:         name,
		Version:      version,
		Ecosystem:    "npm",
		CompareAs:    "npm",
		Commit:       tryExtractCommit(resolution),
		Dependencies: dependencies,
	}
}

func ParseYarnLock(pathToLockfile string) ([]PackageDetails, error) {
	file, err := os.Open(pathToLockfile)
	if err != nil {
		return []PackageDetails{}, fmt.Errorf("could not open %s: %w", pathToLockfile, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	packageGroups := groupYarnPackageLines(scanner)

	if err := scanner.Err(); err != nil {
		return []PackageDetails{}, fmt.Errorf("error while scanning %s: %w", pathToLockfile, err)
	}

	packages := make([]PackageDetails, 0, len(packageGroups))

	for _, group := range packageGroups {
		if group[0] == "__metadata:" {
			continue
		}

		packages = append(packages, parseYarnPackageGroup(group))
	}

	return packages, nil
}
