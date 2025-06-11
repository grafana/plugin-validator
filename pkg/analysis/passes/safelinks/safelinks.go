package safelinks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

type LinkResult struct {
	Link    metadata.Link
	Threats []ThreatType
	Error   error
}

type WebRiskResponse struct {
	Threat *struct {
		ThreatTypes []string `json:"threatTypes"`
	} `json:"threat,omitempty"`
}

type ThreatType string

const (
	ThreatTypeMalware                           ThreatType = "MALWARE"
	ThreatTypeSocialEngineering                 ThreatType = "SOCIAL_ENGINEERING"
	ThreatTypeUnwantedSoftware                  ThreatType = "UNWANTED_SOFTWARE"
	ThreatTypeSocialEngineeringExtendedCoverage ThreatType = "SOCIAL_ENGINEERING_EXTENDED_COVERAGE"
)

var webriskApiKey = os.Getenv("WEBRISK_API_KEY")

var (
	webriskFlagged    = &analysis.Rule{Name: "webrisk-flagged", Severity: analysis.Error}
	webRiskAPIBaseURL = "https://webrisk.googleapis.com/v1/uris:search"
	requestTimeout    = 15 * time.Second
	httpClient        = &http.Client{}
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

	if webriskApiKey == "" {
		return nil, nil
	}

	ctx := context.Background()

	results := CheckURLs(ctx, data.Info.Links)

	for _, result := range results {
		if result.Error != nil || len(result.Threats) > 0 {
			pass.ReportResult(pass.AnalyzerName, webriskFlagged,
				"Webrisk flagged link",
				fmt.Sprintf("Link with name %s is not safe: can be a %s", result.Link.Name, getThreatTypeString(result.Threats)))
		}
	}
	return nil, nil
}

func CheckURLs(ctx context.Context, links []metadata.Link) []LinkResult {
	results := make([]LinkResult, len(links))

	for i, link := range links {
		result := LinkResult{Link: link}

		if link.URL == "" {
			results[i] = result
			continue
		}

		params := url.Values{}
		params.Set("uri", link.URL)
		params.Set("key", webriskApiKey)
		params.Add("threatTypes", string(ThreatTypeMalware))
		params.Add("threatTypes", string(ThreatTypeSocialEngineering))
		params.Add("threatTypes", string(ThreatTypeUnwantedSoftware))
		params.Add("threatTypes", string(ThreatTypeSocialEngineeringExtendedCoverage))

		apiURL := fmt.Sprintf("%s?%s", webRiskAPIBaseURL, params.Encode())

		reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		req, err := http.NewRequestWithContext(reqCtx, "GET", apiURL, nil)
		if err != nil {
			cancel()
			result.Error = fmt.Errorf("failed to create request: %w", err)
			results[i] = result
			continue
		}

		resp, err := httpClient.Do(req)
		cancel()

		if err != nil {
			result.Error = fmt.Errorf("API call failed: %w", err)
			results[i] = result
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			result.Error = fmt.Errorf("API returned status %d", resp.StatusCode)
			results[i] = result
			continue
		}

		// Parse response
		var webRiskResp WebRiskResponse
		if err := json.NewDecoder(resp.Body).Decode(&webRiskResp); err != nil {
			result.Error = fmt.Errorf("failed to decode response: %w", err)
			results[i] = result
			continue
		}

		if webRiskResp.Threat != nil && len(webRiskResp.Threat.ThreatTypes) > 0 {
			result.Threats = make([]ThreatType, len(webRiskResp.Threat.ThreatTypes))
			for j, threatStr := range webRiskResp.Threat.ThreatTypes {
				result.Threats[j] = ThreatType(threatStr)
			}
		}

		results[i] = result
	}

	return results
}

func getThreatTypeString(threatTypes []ThreatType) string {
	threatTypeStrings := make([]string, len(threatTypes))
	for i, threatType := range threatTypes {
		threatTypeStrings[i] = string(threatType)
	}
	return strings.Join(threatTypeStrings, ", ")
}
