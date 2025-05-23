package safelinks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	webrisk "cloud.google.com/go/webrisk/apiv1"
	"cloud.google.com/go/webrisk/apiv1/webriskpb"
	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"google.golang.org/api/option"
)

type LinkResult struct {
	Link    metadata.Link
	Threats []webriskpb.ThreatType
	Error   error
}

var webriskApiKey = os.Getenv("WEBRISK_API_KEY")

var (
	webriskFlagged = &analysis.Rule{Name: "webrisk-flagged", Severity: analysis.Error}
)

var Analyzer = &analysis.Analyzer{
	Name:     "safelinks",
	Requires: []*analysis.Analyzer{metadata.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{webriskFlagged},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Safe Links",
		Description: "Checks that links from `plugin.json` are safe.",
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
		return nil, nil
	}

	ctx := context.Background()

	client, err := webrisk.NewClient(ctx, option.WithAPIKey(webriskApiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Web Risk client with API key: %w", err)
	}
	defer client.Close()
	results := CheckURLs(ctx, client, data.Info.Links)

	for _, result := range results {
		if result.Error != nil || len(result.Threats) > 0 {
			pass.ReportResult(pass.AnalyzerName, webriskFlagged,
				"Webrisk flagged link",
				fmt.Sprintf("Link with name %s is not safe: can be a %s", result.Link.Name, getThreatTypeString(result.Threats)))
		}

	}
	return nil, nil
}

func CheckURLs(ctx context.Context, client *webrisk.Client, links []metadata.Link) []LinkResult {
	results := make([]LinkResult, len(links))

	for i, link := range links {
		result := LinkResult{
			Link: link,
		}

		if link.URL == "" {
			continue
		}

		req := &webriskpb.SearchUrisRequest{
			Uri: link.URL,
			ThreatTypes: []webriskpb.ThreatType{
				webriskpb.ThreatType_MALWARE,
				webriskpb.ThreatType_SOCIAL_ENGINEERING,
				webriskpb.ThreatType_UNWANTED_SOFTWARE,
				webriskpb.ThreatType_SOCIAL_ENGINEERING_EXTENDED_COVERAGE,
			},
		}

		apiCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		resp, err := client.SearchUris(apiCtx, req)
		cancel()

		if err != nil {
			result.Error = fmt.Errorf("API call failed: %w", err)
		} else if resp.Threat != nil {
			result.Threats = resp.Threat.ThreatTypes
		}

		results[i] = result
	}

	return results
}

func getThreatTypeString(threatTypes []webriskpb.ThreatType) string {
	threatTypeStrings := make([]string, len(threatTypes))
	for i, threatType := range threatTypes {
		threatTypeStrings[i] = threatType.String()
	}
	return strings.Join(threatTypeStrings, ", ")
}
