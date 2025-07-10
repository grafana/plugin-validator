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
	"github.com/grafana/plugin-validator/pkg/logme"
)

type linkResult struct {
	link    metadata.Link
	threats threatTypes
	err     error
}
type threatTypes []threatType

func (t threatTypes) String() string {
	threatTypeStrings := make([]string, len(t))
	for i, threatType := range t {
		threatTypeStrings[i] = string(threatType)
	}
	return strings.Join(threatTypeStrings, ", ")
}

type webRiskResponse struct {
	Threat *struct {
		ThreatTypes []threatType `json:"threatTypes"`
	} `json:"threat,omitempty"`
}

type threatType string

const (
	threatTypeMalware                           threatType = "MALWARE"
	threatTypeSocialEngineering                 threatType = "SOCIAL_ENGINEERING"
	threatTypeUnwantedSoftware                  threatType = "UNWANTED_SOFTWARE"
	threatTypeSocialEngineeringExtendedCoverage threatType = "SOCIAL_ENGINEERING_EXTENDED_COVERAGE"
)

var allThreatTypes = []threatType{
	threatTypeMalware,
	threatTypeSocialEngineering,
	threatTypeUnwantedSoftware,
	threatTypeSocialEngineeringExtendedCoverage,
}

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
	webriskApiKey := os.Getenv("WEBRISK_API_KEY")
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

	results := checkURLs(ctx, webriskApiKey, data.Info.Links)

	for _, result := range results {
		if result.err != nil {
			logme.ErrorF("Failed to check link via webrisk API: %q -> %v\n", result.link.Name, result.err)
			continue
		}
		if len(result.threats) > 0 {
			pass.ReportResult(pass.AnalyzerName, webriskFlagged,
				"Webrisk flagged link",
				fmt.Sprintf("Link with name %s is not safe: can be a %s", result.link.Name, result.threats.String()))
		}
	}
	return nil, nil
}

func checkURLs(ctx context.Context, apiKey string, links []metadata.Link) []linkResult {
	results := []linkResult{}

	for _, link := range links {
		result := linkResult{link: link}

		if link.URL == "" {
			continue
		}

		params := url.Values{}
		params.Set("uri", link.URL)
		for _, tt := range allThreatTypes {
			params.Add("threatTypes", string(tt))
		}

		apiURL := fmt.Sprintf("%s?%s", webRiskAPIBaseURL, params.Encode())

		reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
		defer cancel()

		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, apiURL, nil)
		if err != nil {
			result.err = fmt.Errorf("failed to create request: %w", err)
			results = append(results, result)
			continue
		}

		req.Header.Set("X-goog-api-key", apiKey)

		resp, err := httpClient.Do(req)

		if err != nil {
			result.err = fmt.Errorf("API call failed: %w", err)
			results = append(results, result)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			result.err = fmt.Errorf("API returned status %d", resp.StatusCode)
			results = append(results, result)
			continue
		}

		var webRiskResp webRiskResponse
		if err := json.NewDecoder(resp.Body).Decode(&webRiskResp); err != nil {
			result.err = fmt.Errorf("failed to decode response: %w", err)
			results = append(results, result)
			continue
		}

		if webRiskResp.Threat != nil && len(webRiskResp.Threat.ThreatTypes) > 0 {
			result.threats = webRiskResp.Threat.ThreatTypes
		}

		results = append(results, result)
	}

	return results
}
