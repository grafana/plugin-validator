package requiredlinks

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

type requiredLink struct {
	label string
	term  string
}

// The terms match how grafana.com picks links from info.links: a case-insensitive substring of the link name.
var requiredLinks = []requiredLink{
	{label: "License", term: "license"},
	{label: "Privacy Policy", term: "privacy"},
	{label: "Support", term: "support"},
}

const requiredScheme = "https"

var (
	missingRequiredLink = &analysis.Rule{
		Name:              "missing-required-link",
		Severity:          analysis.Error,
		DisabledByDefault: true,
	}
)

var Analyzer = &analysis.Analyzer{
	Name:     "requiredlinks",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{missingRequiredLink},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Required Links",
		Description: "Checks that `plugin.json` has License, Privacy Policy and Support links with valid https URLs. Disabled by default.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	for _, required := range requiredLinks {
		checkLink(pass, data.Info.Links, required)
	}

	return nil, nil
}

func checkLink(pass *analysis.Pass, links []metadata.Link, required requiredLink) {
	matches := findLinks(links, required.term)
	if len(matches) == 0 {
		pass.ReportResult(
			pass.AnalyzerName,
			missingRequiredLink,
			fmt.Sprintf("plugin.json: missing %s link", required.label),
			fmt.Sprintf("Add an entry to info.links with a name containing %q and a valid https URL.", required.term),
		)
		return
	}

	if slices.ContainsFunc(matches, func(link metadata.Link) bool { return isValidURL(link.URL) }) {
		return
	}

	pass.ReportResult(
		pass.AnalyzerName,
		missingRequiredLink,
		fmt.Sprintf("plugin.json: %s link has an invalid URL", required.label),
		fmt.Sprintf("The info.links entry for %s must have a valid https URL.", required.label),
	)
}

func findLinks(links []metadata.Link, term string) []metadata.Link {
	var matches []metadata.Link
	for _, link := range links {
		if !strings.Contains(strings.ToLower(link.Name), term) {
			continue
		}
		matches = append(matches, link)
	}
	return matches
}

func isValidURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == requiredScheme && parsed.Host != ""
}
