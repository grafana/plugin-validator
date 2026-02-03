package llmreview

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/backendbinary"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/binarypermissions"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/coderules"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/gomanifest"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/manifest"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadatavalid"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/osvscanner"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/safelinks"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/trackingscripts"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/unsafesvg"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/virusscan"
	"github.com/grafana/plugin-validator/pkg/llmvalidate"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var geminiKey = os.Getenv("GEMINI_API_KEY")

var (
	llmIssueFound    = &analysis.Rule{Name: "llm-issue-found", Severity: analysis.SuspectedProblem}
	llmReviewSkipped = &analysis.Rule{
		Name:     "llm-review-skipped",
		Severity: analysis.SuspectedProblem,
	}
)

// blockingAnalyzers contains validators that, if they report errors, should cause
// the LLM review to be skipped to save costs. These are grouped into:
// - Tier 1 (Structure): archive, metadata, metadatavalid, modulejs
// - Tier 2 (Security): coderules, trackingscripts, virusscan, safelinks, unsafesvg, osvscanner
// - Tier 3 (Integrity): gomanifest, binarypermissions, backendbinary, manifest
var blockingAnalyzers = []*analysis.Analyzer{
	// Tier 1: Fundamental structure issues
	archive.Analyzer,
	metadata.Analyzer,
	metadatavalid.Analyzer,
	modulejs.Analyzer,
	// Tier 2: Security/Policy violations
	coderules.Analyzer,
	trackingscripts.Analyzer,
	virusscan.Analyzer,
	safelinks.Analyzer,
	unsafesvg.Analyzer,
	osvscanner.Analyzer,
	// Tier 3: Build/Integrity issues
	gomanifest.Analyzer,
	binarypermissions.Analyzer,
	backendbinary.Analyzer,
	manifest.Analyzer,
}

var Analyzer = &analysis.Analyzer{
	Name: "llmreview",
	Requires: []*analysis.Analyzer{
		sourcecode.Analyzer,
		// Blocking analyzers - if any report errors, LLM review is skipped
		archive.Analyzer,
		metadata.Analyzer,
		metadatavalid.Analyzer,
		modulejs.Analyzer,
		coderules.Analyzer,
		trackingscripts.Analyzer,
		virusscan.Analyzer,
		safelinks.Analyzer,
		unsafesvg.Analyzer,
		osvscanner.Analyzer,
		gomanifest.Analyzer,
		binarypermissions.Analyzer,
		backendbinary.Analyzer,
		manifest.Analyzer,
	},
	Run: run,
	Rules:    []*analysis.Rule{llmIssueFound, llmReviewSkipped},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:         "LLM Review",
		Description:  "Runs the code through Gemini LLM to check for security issues or disallowed usage.",
		Dependencies: "Gemini API key",
	},
}

var Questions = []llmvalidate.LLMQuestion{
	{
		Question:       "Only for go/golang code: Does this code directly read from or write to the file system? (Look for uses of os.Open, os.Create, ioutil.ReadFile, ioutil.WriteFile, etc.). Provide the specific code snippet if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Does this code execute user input as code in a browser environment? (Look for eval(), new Function(), document.write() with unescaped content, innerHTML with script tags, etc.). Provide the specific code snippet if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Only for go/golang code: Does this code execute user input as commands or code in the backend? (Look for exec.Command, syscall.Exec, template.Execute with user data, etc.). Provide the specific code snippet if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Does this code introduce third-party analytics or tracking features? (Grafana's reportInteraction from @grafana/runtime is allowed, but external services like Google Analytics, Mixpanel, etc. are not). Provide the specific code snippet if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Does this code modify or create properties on the global window object? (Look for direct assignments like window.customVariable = x, window.functionName = function(){}, or adding undeclared variables in global scope). Exclude standard browser API usage. Provide the specific code snippet if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Does this code introduce global CSS not scoped to components? (Emotion CSS and CSS modules are allowed, but look for direct style tags, global class definitions, or modification of document.styleSheets). Provide the specific code snippet if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Does this code dynamically inject external third-party scripts? (Look for createElement('script'), setting src attributes to external domains, document.write with script tags, or dynamic import() from external sources). Provide the specific code snippet with the external URL if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Only for go/golang code: Are there any opened resources that are NOT properly closed? (Check for files, network connections, etc. that lack proper closure with defer, in finally blocks, or using 'with' statements). Identify any improperly closed resources with a code snippet. If there is no backend code reply negatively",
		ExpectedAnswer: false,
	},
	{
		Question:       "Does this code use global DOM selectors outside of component lifecycle methods? (Look for direct usage of document.querySelector(), document.getElementById(), document.getElementsByClassName(), etc. that aren't scoped to specific components or that bypass React refs). Component-scoped element access like useRef() or this.elementRef is acceptable. Provide the specific code snippet showing the global access if found.",
		ExpectedAnswer: false,
	},
	{
		Question:       "Only for go/golang code: Does this code create HTTP clients without using github.com/grafana/grafana-plugin-sdk-go/backend/httpclient? (Look for direct creation of http.Client{}, http.NewRequest, calls to third-party NewClient/NewHTTPClient functions that don't accept or use the SDK's httpclient, or any other HTTP client initialization that doesn't use github.com/grafana/grafana-plugin-sdk-go/backend/httpclient. The httpclient from github.com/grafana/grafana-plugin-sdk-go/backend/httpclient should be used directly or passed to the HTTP client being created. This includes cases where third-party libraries create HTTP clients internally - those libraries should accept the SDK's httpclient as a parameter). Provide the specific code snippet if found.",
		ExpectedAnswer: false,
	},
}

func run(pass *analysis.Pass) (any, error) {
	if os.Getenv("SKIP_LLM_REVIEW") != "" {
		return nil, nil
	}

	// Check if any blocking analyzers reported errors - skip LLM review to save costs
	// keep here before source code and key check for tests to work
	for _, analyzer := range blockingAnalyzers {
		if pass.AnalyzerHasErrors(analyzer) {
			pass.ReportResult(
				pass.AnalyzerName,
				llmReviewSkipped,
				fmt.Sprintf("LLM review skipped due to errors in %s", analyzer.Name),
				fmt.Sprintf(
					"Fix the errors reported by %s before LLM review can run.",
					analyzer.Name,
				),
			)
			return nil, nil
		}
	}

	var err error
	// only run if sourcecode.Analyzer succeeded
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	if geminiKey == "" {
		return nil, nil
	}

	logme.Debugln("Starting to run Gemini Validations. This might take a while...")

	llmClient, err := llmvalidate.New(context.Background(), geminiKey, "gemini-3-flash-preview")

	if err != nil {
		logme.DebugFln("Error initializing llm client: %v", err)
		return nil, nil
	}

	var answers []llmvalidate.LLMAnswer
	answers, err = llmClient.AskLLMAboutCode(sourceCodeDir, Questions, []string{"src", "pkg"})
	if err != nil {
		logme.DebugFln("Error getting answers from Gemini LLM: %v", err)
		return nil, nil
	}

	for _, answer := range answers {
		if answer.ShortAnswer != answer.ExpectedShortAnswer {

			var detailParts []string

			detailParts = append(detailParts, answer.Answer)

			if answer.CodeSnippet != "" {
				detailParts = append(
					detailParts,
					fmt.Sprintf("**Code Snippet:**\n```\n%s\n```", answer.CodeSnippet),
				)
			}

			if len(answer.Files) > 0 {
				detailParts = append(
					detailParts,
					fmt.Sprintf("**Files:** %s", strings.Join(answer.Files, ", ")),
				)
			}

			detail := strings.Join(detailParts, "\n\n")

			pass.ReportResult(
				pass.AnalyzerName,
				llmIssueFound,
				fmt.Sprintf("LLM flagged: %s", answer.Question),
				detail,
			)
		}
	}

	return nil, nil
}
