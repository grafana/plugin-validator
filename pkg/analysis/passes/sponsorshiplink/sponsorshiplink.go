package sponsorshiplink

import (
	"encoding/json"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	sponsorshiplink = &analysis.Rule{Name: "sponsorshiplink", Severity: analysis.Recommendation}
	explanation     = "Consider to add a sponsorship link in your plugin.json file (Info.Links section: with Name: 'sponsor' or Name: 'sponsorship'), which will be shown on the plugin details page to allow users to support your work if they wish."
	recommendation  = "You can include a sponsorship link if you want users to support your work"
)

var Analyzer = &analysis.Analyzer{
	Name:     "sponsorshiplink",
	Run:      run,
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Rules:    []*analysis.Rule{sponsorshiplink},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Sponsorship Link",
		Description: "Checks if a sponsorship link is specified in `plugin.json` that will be shown in the Grafana plugin catalog for users to support the plugin developer.",
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

	if len(data.Info.Links) == 0 {
		pass.ReportResult(pass.AnalyzerName, sponsorshiplink, recommendation, explanation)
		return nil, nil
	}
	hasSponsorLink := false
	for _, link := range data.Info.Links {
		name := strings.ToLower(link.Name)
		if strings.Contains(name, "sponsor") || strings.Contains(name, "sponsorship") {
			hasSponsorLink = true
		}
	}
	if !hasSponsorLink {
		pass.ReportResult(pass.AnalyzerName, sponsorshiplink, recommendation, explanation)
	}

	return nil, nil
}
