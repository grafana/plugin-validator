package llmreview

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/llmvalidate"
	"github.com/grafana/plugin-validator/pkg/logme"
)

var geminiKey = os.Getenv("GEMINI_API_KEY")

var (
	llmIssueFound = &analysis.Rule{Name: "llm-issue-found", Severity: analysis.SuspectedProblem}
)

var Analyzer = &analysis.Analyzer{
	Name:     "llmreview",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{llmIssueFound},
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
}

func run(pass *analysis.Pass) (any, error) {
	if os.Getenv("SKIP_LLM_REVIEW") != "" {
		return nil, nil
	}

	var err error
	// only run if sourcecode.Analyzer succeeded
	sourceCodeDir, ok := analysis.GetResult[string](pass, sourcecode.Analyzer)
	if !ok {
		return nil, nil
	}

	if geminiKey == "" {
		return nil, nil
	}

	logme.Debugln("Starting to run Gemini Validations. This might take a while...")

	llmClient, err := llmvalidate.New(context.Background(), geminiKey, "gemini-2.5-flash")

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
				detailParts = append(detailParts, fmt.Sprintf("**Code Snippet:**\n```\n%s\n```", answer.CodeSnippet))
			}

			if len(answer.Files) > 0 {
				detailParts = append(detailParts, fmt.Sprintf("**Files:** %s", strings.Join(answer.Files, ", ")))
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
