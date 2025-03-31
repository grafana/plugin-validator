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

var questions = []string{
	"Does this code manipulate the file system? (explicit manipulation of the file system). Provide a code snippet if so.",
	"Does this code allow the execution or arbitrary javascript code from user input?. Provide a code snippet if so",
	"Does this code allow the execution or arbitrary code in go from user input?. Provide a code snippet if so.",
	"Does this code introduces analytics or tracking not part of Grafana APIs?. Provide a code snippet if so.",
}

func run(pass *analysis.Pass) (any, error) {
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

	llmClient, err := llmvalidate.New(context.Background(), geminiKey, "gemini-2.0-flash")

	if err != nil {
		logme.DebugFln("Error initializing llm client: %v", err)
		return nil, nil
	}

	retry := 3
	var answers []llmvalidate.LLMAnswer

	for i := 0; i < retry; i++ {
		answers, err = llmClient.AskLLMAboutCode(sourceCodeDir, questions, []string{"src", "pkg"})
		if err != nil {
			logme.DebugFln("Error getting answers from Gemini LLM: %v", err)
		} else {
			break
		}
	}

	if err != nil {
		logme.DebugFln("Error getting answers from Gemini LLM: %v", err)
		return nil, nil
	}

	for _, answer := range answers {
		if answer.ShortAnswer {

			detail := fmt.Sprintf("Question: %s\n. Answer: %s. ", answer.Question, answer.Answer)

			if answer.CodeSnippet != "" {
				detail += fmt.Sprintf("Code Snippet:\n```\n%s\n```\n", answer.CodeSnippet)
			}

			if len(answer.Files) > 0 {
				detail += fmt.Sprintf(". Files: %s", strings.Join(answer.Files, ", "))
			}

			pass.ReportResult(
				pass.AnalyzerName,
				llmIssueFound,
				fmt.Sprintf("LLM response: %s", answer.Answer),
				detail,
			)
		}
	}

	return nil, nil
}
