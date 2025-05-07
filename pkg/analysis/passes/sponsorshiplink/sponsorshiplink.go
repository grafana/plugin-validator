package sponsorshiplink

import (
	"encoding/json"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	sponsorshiplink = &analysis.Rule{Name: "sponsorshiplink", Severity: analysis.Recommendation}
)

var Analyzer = &analysis.Analyzer{
	Name:     "sponsorshiplink",
	Run:      checkSponsorshiplink,
	Requires: []*analysis.Analyzer{metadata.Analyzer, archive.Analyzer},
	Rules:    []*analysis.Rule{sponsorshiplink},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Sponsorship Link",
		Description: "Checks if a sponsorship link is specified in `plugin.json` that will be shown in the Grafana plugin catalog for users to support the plugin developer.",
	},
}

func checkSponsorshiplink(pass *analysis.Pass) (interface{}, error) {
	metadataBody, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var data metadata.Metadata
	if err := json.Unmarshal(metadataBody, &data); err != nil {
		return nil, err
	}

	if len(data.Info.Links) == 0 {
		explanation := "Consider to add a sponsorship link in your plugin.json file (Info.Links section), which will be shown on the plugin details page to allow users to support your work if they wish."
		pass.ReportResult(pass.AnalyzerName, sponsorshiplink, "plugin.json: You can include a sponsorship link if you want users to support your work", explanation)
		return nil, nil
	}

	for _, link := range data.Info.Links {
		name := strings.ToLower(link.Name)
		if strings.Contains(name, "sponsor") || strings.Contains(name, "sponsorship") {
			return nil, nil
		}
	}
	explanation := "Consider to add a sponsorship link in your plugin.json file (Info.Links section), which will be shown on the plugin details page to allow users to support your work if they wish."
	pass.ReportResult(pass.AnalyzerName, sponsorshiplink, "plugin.json: You can include a sponsorship link if you want users to support your work", explanation)
	return nil, nil
}
