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

// these are not regular expressions
// these are unix filename patterns
var ignoreList = []string{
	// hidden files
	"**/.**",
	".**",

	//dependencies
	"node_modules/**",
	"*.lock",

	// dist files
	"dist/**",

	//external files
	"**/external/**",
	"**/*.min.js",

	// tests
	"**/tests/**",
	"**/test/**",
	"**/test-**",
	"**/__mocks__/**",
	"**/*.test.*",
	"**/*.spec.*",
	"**/*_test.go",
	"tests/**",
	"mocks/**",
	"server-*",
	"cypress/**",

	// config
	"jest.config.*",
	"babel.config.*",
	"jest-setup.*",
	"playwright.config.*",
	"vite.config.*",
	"**/tsconfig.*",
	"Gruntfile.*",
	"webpack.config.*",
	"rollup.config.*",
}

var allowExtensions = map[string]struct{}{
	".js":  {},
	".jsx": {},
	".ts":  {},
	".tsx": {},
	".cjs": {},
	".mjs": {},
	".go":  {},
}

var extensionToFileType = map[string]string{
	".js":  "javascript",
	".jsx": "javascript",
	".ts":  "typescript",
	".tsx": "typescript",
	".cjs": "javascript",
	".mjs": "javascript",
	".go":  "go",
}

type Client struct {
	genaiClient *genai.Client
	apiKey      string
	modelName   string
	ctx         context.Context
}

type LLMAnswer struct {
	Question    string   `json:"question"`
	Answer      string   `json:"answer"`
	Files       []string `json:"files"`
	ShortAnswer bool     `json:"short_answer"`
	CodeSnippet string   `json:"code_snippet"`
}

func New(ctx context.Context, apiKey string, modelName string) (*Client, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if modelName == "" {
		modelName = "gemini-2.0-flash-lite"
	}
	logme.DebugFln("llmvalidate: Using model %s", modelName)

	genaiClient, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	return &Client{
		genaiClient: genaiClient,
		modelName:   modelName,
		apiKey:      apiKey,
		ctx:         ctx,
	}, nil
}

func (c *Client) AskLLMAboutCode(
	codePath string,
	questions []string,
	subPathsOnly []string,
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

	codePrompt, err := getPromptContentForCode(absCodePath, subPathsOnly)
	if err != nil {
		return nil, fmt.Errorf("Error walking files inside %s: %v", codePath, err)
	}

	model := c.genaiClient.GenerativeModel(c.modelName)
	// ensure it outputs json
	model.GenerationConfig.ResponseMIMEType = "application/json"
	model.ResponseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"question": {
				Type:        genai.TypeString,
				Description: "The original question",
			},
			"answer": {
				Type:        genai.TypeString,
				Description: "The full answer to the question. Elaborate why yes or no",
			},
			"files": {
				Type: genai.TypeArray,
				Items: &genai.Schema{
					Type: genai.TypeString,
				},
				Description: "An array of files related to the answer",
			},
			"short_answer": {
				Type:        genai.TypeBoolean,
				Description: "True or false",
			},
			"code_snippet": {
				Type:        genai.TypeString,
				Description: "Code snippet as context for the answer if applicable",
			},
		},
	}

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(
				`You are source code reviewer. You are provided with a source code repository information and files. You will answer questions only based on the context of the files provided

The output should be a valid plain JSON array. Each element with an answer containing fields:

* question: The original question
* answer: The answer
* files: An array of related files if applicable.
* short_answer: True or false
* code_snippet: The code snippet relevant to the question. Empty if not applicable
        `,
			),
		},
	}

	filesPrompt := fmt.Sprintf(
		`The files in the repository are: %s `,
		strings.Join(codePrompt, "\n"),
	)
	var answers []LLMAnswer = make([]LLMAnswer, len(questions))

	for _, question := range questions {
		questionPrompt := fmt.Sprintf(
			"%s\n\n Answer the question based on the previous files: %s",
			filesPrompt,
			question,
		)
		var answer LLMAnswer
		modelResponse, err := model.GenerateContent(c.ctx, genai.Text(questionPrompt))
		if err != nil {
			return nil, err
		}

		content := getTextContentFromModelContentResponse(modelResponse)
		//unmarshall content into []LLMAnswer
		err = json.Unmarshal([]byte(content), &answer)
		if err != nil {
			logme.DebugFln("Failed to unmarshal content: %v", content)
			return nil, fmt.Errorf("failed to unmarshal content: %v", err)
		}
		logme.DebugFln("Got answer from LLM: %v", answer)
		logme.Debugln("Got response from LLM with char length", len(content))
		answers = append(answers, answer)
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
	// replace any duplicated new lines with a single new line
	finalContent = strings.ReplaceAll(finalContent, "\n\n", "\n")
	return finalContent
}

func getPromptContentForCode(codePath string, subPathsOnly []string) ([]string, error) {
	var prompts []string

	if len(subPathsOnly) == 0 {
		subPathsOnly = []string{"."}
	}

	for _, p := range subPathsOnly {
		subCodePath := filepath.Join(codePath, p)

		// skip if it doesn't exist
		_, err := os.Stat(subCodePath)
		if err != nil {
			continue
		}

		subPrompts, err := walkAndGetPrompts(subCodePath)
		if err != nil {
			return nil, err
		}
		prompts = append(prompts, subPrompts...)
	}

	return prompts, nil

}

func walkAndGetPrompts(codePath string) ([]string, error) {
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
	_, ok := allowExtensions[extension]
	return ok
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

	logme.DebugFln("llmvalidate: Including file %s", path.Join(codePath, relFile))

	if len(content) == 0 {
		return ""
	}

	fileExt := filepath.Ext(relFile)
	fileType, ok := extensionToFileType[fileExt]
	if !ok {
		fileType = fileExt
	}

	// this will format the content as:
	// path/to/filename:
	// ```filetype
	// content
	// ```
	promptContent := fmt.Sprintf("%s:\n```%s\n%s\n```\n", relFile, fileType, content)
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
		text := scanner.Text()
		if !utf8.ValidString(text) {
			return "", fmt.Errorf("invalid UTF-8 in file %s", filePath)
		}
		content.WriteString(text)
		content.WriteRune('\n')
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return content.String(), nil
}
