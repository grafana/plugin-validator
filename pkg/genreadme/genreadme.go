package genreadme

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes"
	"github.com/grafana/plugin-validator/pkg/logme"
)

const (
	magicStart = `<!-- analyzers-table-start -->`
	magicEnd   = `<!-- analyzers-table-end -->`
	header     = `<!--
THE FOLLOWING SECTION IS GENERATED, DO NOT EDIT.
Run "mage gen:readme" to regenerate this section.
-->
| Analyzer | Description | Dependencies |
|----------|-------------|--------------|
`
)

func Generate(readme io.Reader) (string, error) {
	var tableBuilder strings.Builder
	tableBuilder.WriteString(header)

	// Sort analyzers alphabetically
	slices.SortFunc(passes.Analyzers, func(a, b *analysis.Analyzer) int {
		return strings.Compare(strings.ToLower(a.ReadmeInfo.Name), strings.ToLower(b.ReadmeInfo.Name))
	})

	// Generate table content
	for _, analyzer := range passes.Analyzers {
		if analyzer.ReadmeInfo.Name == "" && analyzer.ReadmeInfo.Description == "" {
			logme.ErrorF("Warning: Analyzer %q does not have README data.\n", analyzer.Name)
			continue
		}
		dependencies := analyzer.ReadmeInfo.Dependencies
		if dependencies == "" {
			dependencies = "None"
		}
		tableBuilder.WriteString(fmt.Sprintf(
			"| %s | %s | %s |\n",
			fmt.Sprintf("%s / `%s`", analyzer.ReadmeInfo.Name, analyzer.Name),
			analyzer.ReadmeInfo.Description,
			dependencies,
		))
	}

	// Update the README
	var outBuilder strings.Builder
	var isBetweenMagicTags bool
	var done bool
	scanner := bufio.NewScanner(readme)
	for scanner.Scan() {
		line := scanner.Text()
		if !isBetweenMagicTags && strings.Contains(line, magicStart) {
			// Re-write the generated section
			isBetweenMagicTags = true
			outBuilder.WriteString(magicStart)
			outBuilder.WriteRune('\n')
			outBuilder.WriteString(tableBuilder.String())
			continue
		}

		if strings.Contains(line, magicEnd) {
			// Write magic end tag
			outBuilder.WriteString(line)
			outBuilder.WriteString("\n")
			isBetweenMagicTags = false
			done = true
			continue
		}

		// Copy the rest of the readme,
		// but don't copy the generated section from the old file
		if !isBetweenMagicTags {
			outBuilder.WriteString(line)
			outBuilder.WriteRune('\n')
		}
	}
	if !done {
		return "", errors.New("failed to find magic tags in readme")
	}
	return outBuilder.String(), nil
}
