package llmvalidate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/danwakefield/fnmatch"
	"github.com/google/generative-ai-go/genai"
	"github.com/grafana/plugin-validator/pkg/logme"
	"google.golang.org/api/option"
)

var ignoreList = []string{
	".**",
	"node_modules/**",
	"dist/**",
	"*.lock",
	"**/external/**",
	"tests/**",
	"**/__mocks__/**",
	"**.test.*",
	"test-**",
	"jest.config.*",
	"babel.config.*",
	"jest-setup.*",
	"playwright.config.*",
	"vite.config.*",
	"tsconfig.*",
	"package-lock.json",
	"Gruntfile.*",
	"webpack.config.*",
	"rollup.config.*",
	"*.min.js",
	"server-**",
	"test/**",
}

var allowExtensions = []string{".js", ".jsx", ".ts", ".tsx", ".cjs", ".mjs", ".go"}

type LLMAnswer struct {
	Question    string   `json:"question"`
	Answer      string   `json:"answer"`
	Files       []string `json:"files"`
	ShortAnswer string   `json:"short_answer"`
	CodeSnippet string   `json:"code_snippet"`
}

type LLMValidateClient struct {
	genaiClient *genai.Client
	apiKey      string
	modelName   string
	ctx         context.Context
}

func New(ctx context.Context, apiKey string, modelName string) (*LLMValidateClient, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if modelName == "" {
		modelName = "gemini-1.5-flash-latest"
	}

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	return &LLMValidateClient{
		genaiClient: genaiClient,
		modelName:   modelName,
		apiKey:      apiKey,
		ctx:         ctx,
	}, nil
}

func (c *LLMValidateClient) AskLLMAboutCode(
	codePath string,
	questions []string,
) ([]LLMAnswer, error) {

	if len(questions) == 0 {
		return nil, fmt.Errorf("No questions provided")
	}

	// check that codepath exists and it is a directory
	stat, err := os.Stat(codePath)
	if err != nil {
		return nil, err
	}

	if !stat.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", codePath)
	}

	absCodePath, err := filepath.Abs(codePath)
	if err != nil {
		return nil, err
	}

	codePrompt, err := getPromptContentForCode(absCodePath)
	if err != nil {
		return nil, fmt.Errorf("Error walking files inside %s: %v", codePath, err)
	}

	model := c.genaiClient.GenerativeModel(c.modelName)
	// ensure it outputs json
	model.GenerationConfig.ResponseMIMEType = "application/json"

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(
				`You are source code reviewer. You are provided with a source code repository information and files. You will answer questions only based on the context of the files provided

The output should be a valid plain JSON array. Each element with an answer containing fields:

* question: The original question
* answer: The answer
* files: An array of related files if applicable.
* short_answer: Yes/No/NA
* code_snippet: The code snippet relevant to the question. Empty if not applicable
        `,
			),
		},
	}

	formattedQuestions := ""
	for _, question := range questions {
		formattedQuestions += fmt.Sprintf("- %s\n", question)
	}

	mainPrompt := fmt.Sprintf(`
The files in the repository are:
### START OF FILES ###

%s

### END OF FILES ###

Answer the following questions in the context of the code above. be brief in your answers.

%s

`, strings.Join(codePrompt, "\n"), formattedQuestions)

	modelResponse, err := model.GenerateContent(c.ctx, genai.Text(mainPrompt))
	if err != nil {
		return nil, err
	}

	content := getTextContentFromModelContentResponse(modelResponse)

	//unmarshall content into []LLMAnswer
	var answers []LLMAnswer
	err = json.Unmarshal([]byte(content), &answers)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal content: %v", err)
	}

	return answers, nil

}

func getTextContentFromModelContentResponse(modelResponse *genai.GenerateContentResponse) string {
	if len(modelResponse.Candidates) == 0 {
		return ""
	}
	content := modelResponse.Candidates[0].Content
	finalContent := ""
	for _, part := range content.Parts {
		finalContent += fmt.Sprint(part)
	}
	return finalContent
}

func getPromptContentForCode(codePath string) ([]string, error) {
	var prompts []string

	err := filepath.Walk(codePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relFilePath, err := filepath.Rel(codePath, path)
		if err != nil {
			return err
		}

		extension := filepath.Ext(relFilePath)
		if !isAllowedExtension(extension) {
			return nil
		}

		if isIgnoredFile(relFilePath) {
			return nil
		}

		prompt := getPromptContentForFile(codePath, relFilePath)
		if prompt != "" {
			prompts = append(prompts, prompt)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return prompts, nil
}

func isAllowedExtension(extension string) bool {
	for _, ext := range allowExtensions {
		if ext == extension {
			return true
		}
	}
	return false
}

func isIgnoredFile(file string) bool {
	flags := fnmatch.FNM_PERIOD | fnmatch.FNM_NOESCAPE
	for _, pattern := range ignoreList {
		if fnmatch.Match(pattern, file, flags) {
			return true
		}
	}
	return false
}

func getPromptContentForFile(codePath, relFile string) string {
	content, err := readFileContent(path.Join(codePath, relFile))
	if err != nil {
		logme.DebugFln("Error reading file %s: %v", relFile, err)
		// we are ignoring this error because this might be a non-text file
		return ""
	}

	if !utf8.ValidString(content) {
		return ""
	}

	if isMinifiedJsFile(content) {
		return ""
	}

	if len(content) == 0 {
		return ""
	}

	promptContent := fmt.Sprintf(`
----##----
Source filename: %s
Source Content:
%s
----##----
`, relFile, content)

	return promptContent
}

func isMinifiedJsFile(jsCode string) bool {
	lines := strings.Split(jsCode, "\n")
	totalLength := 0
	for _, line := range lines {
		totalLength += len(line)
	}
	averageLineLength := float64(totalLength) / float64(len(lines))
	return averageLineLength > 100
}

func readFileContent(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var content strings.Builder

	for scanner.Scan() {
		if !utf8.ValidString(string(scanner.Text())) {
			return "", fmt.Errorf("invalid UTF-8 in file %s", filePath)
		}
		content.WriteString(scanner.Text() + "\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return content.String(), nil
}
