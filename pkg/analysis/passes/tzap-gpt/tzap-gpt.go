package tzapgpt

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/tailscale/hujson"
)

// embbed tzapinclude.txt and tzapignore.txt
//
//go:embed tzapinclude.txt tzapignore.txt
var embeddedFiles embed.FS

// max allowed amount to spend on gpt APIs
var maxAllowedPrice = "0.5"
var openaiApiKey = os.Getenv("OPENAI_API_KEY")

var (
	tzapInstallError  = &analysis.Rule{Name: "tzap-gpt-install-error", Severity: analysis.Warning}
	tzapGptIssueFound = &analysis.Rule{Name: "tzap-gpt-issue-found", Severity: analysis.SuspectedProblem}
)

type GptJsonResponse struct {
	ShortResponse string   `json:"shortResponse"`
	RelatedFiles  []string `json:"relatedFiles"`
	Answer        string   `json:"answer"`
}

var Analyzer = &analysis.Analyzer{
	Name:     "tzap-gpt",
	Requires: []*analysis.Analyzer{sourcecode.Analyzer},
	Run:      run,
	Rules:    []*analysis.Rule{tzapInstallError, tzapGptIssueFound},
}

var outputPrompt = "Strictly return the answer ONLY as a json (without any other comments and without format) The json response must have the following attributes: shortResponse: [yes or no], relatedFiles: [related files to answer if any, max 3], answer: [answer]"

var questions = []string{
	"Does this codebase uses golang apis to access the file system?. If so tell which files have the APIs accessing the file system",
	"Does this codebase uses nodejs apis to access the file system?. If so tell which files have the APIs accessing the file system",
	"Does this codebase allows the evaluation of user input code in javascript code using eval() or Function()?. Only reply yes if you find actual usage of those APIs. If so tell which files contain the code allowing the evaluation",
	"Does this codebase allows the evaluation of user input code inside golang code?. If so tell which files contain the code allowing the evaluation",
}

func run(pass *analysis.Pass) (interface{}, error) {
	var err error
	// only run if sourcecode.Analyzer succeeded
	sourceCodeDir, ok := pass.ResultOf[sourcecode.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	if openaiApiKey == "" {
		if tzapInstallError.ReportAll {
			pass.ReportResult(pass.AnalyzerName, tzapInstallError, "OPENAI_API_KEY not found in env", "Skipping tzap installation")
		}
		return nil, nil
	}

	logme.Debugln("Starting to run tzap. This might take a while...")

	err = initializeZTap(sourceCodeDir)
	if err != nil {
		if tzapInstallError.ReportAll {
			pass.ReportResult(pass.AnalyzerName, tzapInstallError, "could not setup tzap", err.Error())
		}
		return nil, err
	}

	for _, question := range questions {
		var err error
		var response GptJsonResponse

		// a retry logic is required because models have a tendency to ignore the output format instructions and
		// sometimes add comments or things that were not required
		retry := 3
		for i := 0; i < retry; i++ {
			response, err = askTzap(question, sourceCodeDir)
			if err == nil {
				break
			}
		}

		if err == nil && strings.TrimSpace(strings.ToLower(response.ShortResponse)) == "yes" {
			pass.ReportResult(pass.AnalyzerName, tzapGptIssueFound, fmt.Sprintf("LLM response: %s", response.Answer), fmt.Sprintf("Question: %s\n. Answer: %s. Files: %s", question, response.Answer, response.RelatedFiles))
			continue
		}

	}

	return nil, nil
}

func askTzap(question string, sourceCodeDir string) (GptJsonResponse, error) {
	npxPath, err := exec.LookPath("npx")
	if err != nil {
		return GptJsonResponse{}, err
	}
	questionToSend := fmt.Sprintf("%s. %s", question, outputPrompt)
	command := []string{"-y", "tzap", "prompt", "--yes", "--price", maxAllowedPrice, "--api", questionToSend}
	cmd := exec.Command(npxPath, command...)
	cmd.Dir = sourceCodeDir

	output, err := cmd.Output()
	if err != nil {
		logme.Debugln("Error running tzap: ", fmt.Sprintf("%v", err))
		return GptJsonResponse{}, err
	}

	// using hujson first to allow some tolerance in the package.json
	// such as comments and trailing commas that nodejs allows
	stdJsonResponse, err := hujson.Standardize(output)
	if err != nil {
		return GptJsonResponse{}, err
	}

	response := GptJsonResponse{}
	err = json.Unmarshal(stdJsonResponse, &response)
	if err != nil {
		return GptJsonResponse{}, err
	}
	return response, nil
}

func getNpxCommandPath() (string, error) {
	// get npx command
	npxCmd, err := exec.LookPath("npx")
	if err != nil {
		return "", err
	}
	if npxCmd == "" {
		return "", fmt.Errorf("npx command not found")
	}

	return npxCmd, nil
}

func initializeZTap(codePath string) error {

	npxPath, err := getNpxCommandPath()
	if err != nil {
		return err
	}

	// tzap include file
	tzapIncludePath := path.Join(codePath, ".tzapinclude")
	tzapIncludeContent, err := embeddedFiles.ReadFile("tzapinclude.txt")
	if err != nil {
		return err
	}
	err = os.WriteFile(tzapIncludePath, tzapIncludeContent, 0644)
	if err != nil {
		return err
	}

	// tzapignore file
	tzapignorePath := path.Join(codePath, ".tzapignore")
	tzapIgnoreContent, err := embeddedFiles.ReadFile("tzapignore.txt")
	if err != nil {
		return err
	}
	err = os.WriteFile(tzapignorePath, tzapIgnoreContent, 0644)
	if err != nil {
		return err
	}

	// initialize tzap (this also downloads it to npx cache)
	command := []string{"-y", "tzap", "--yes", "init"}
	cmd := exec.Command(npxPath, command...)
	cmd.Dir = codePath

	_, err = cmd.Output()
	if err != nil {
		return err
	}

	return nil
}
