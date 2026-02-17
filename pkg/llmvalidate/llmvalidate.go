package llmvalidate

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/danwakefield/fnmatch"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/prettyprint"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
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
	llm       llms.Model
	provider  string
	modelName string
	ctx       context.Context
}

type LLMQuestion struct {
	Question       string
	ExpectedAnswer bool
}

type LLMAnswer struct {
	Question            string   `json:"question"`
	Answer              string   `json:"answer"`
	Files               []string `json:"files"`
	ShortAnswer         bool     `json:"short_answer"`
	ExpectedShortAnswer bool
	CodeSnippet         string `json:"code_snippet"`
}

func New(ctx context.Context, provider string, modelName string, apiKey string) (*Client, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	if modelName == "" {
		return nil, fmt.Errorf("Model name is required")
	}

	if provider == "" {
		return nil, fmt.Errorf("Provider is required")
	}

	logme.DebugFln("llmvalidate: Using provider %s with model %s", provider, modelName)

	var llm llms.Model
	var err error

	switch provider {
	case "google":
		llm, err = googleai.New(
			ctx,
			googleai.WithAPIKey(apiKey),
			googleai.WithDefaultModel(modelName),
		)
	case "anthropic":
		llm, err = anthropic.New(
			anthropic.WithToken(apiKey),
			anthropic.WithModel(modelName),
		)
	case "openai":
		llm, err = openai.New(
			openai.WithToken(apiKey),
			openai.WithModel(modelName),
		)
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: google, anthropic, openai)", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create %s client: %w", provider, err)
	}

	return &Client{
		llm:       llm,
		provider:  provider,
		modelName: modelName,
		ctx:       ctx,
	}, nil
}

func createReplyQuestionTool() llms.Tool {
	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        "reply_question",
			Description: "Reply to the code review question with structured information",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"question": map[string]interface{}{
						"type":        "string",
						"description": "The question being answered",
					},
					"answer": map[string]interface{}{
						"type":        "string",
						"description": "The full answer. Elaborate why yes or no",
					},
					"files": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Files related to the answer (optional)",
					},
					"short_answer": map[string]interface{}{
						"type":        "boolean",
						"description": "True or false",
					},
					"code_snippet": map[string]interface{}{
						"type":        "string",
						"description": "Code snippet as context (optional)",
					},
				},
				"required": []string{"question", "answer", "short_answer"},
			},
		},
	}
}

func extractAnswerFromToolCall(resp *llms.ContentResponse) (LLMAnswer, error) {
	var answer LLMAnswer

	if resp == nil {
		return answer, fmt.Errorf("nil response")
	}

	if len(resp.Choices) == 0 {
		return answer, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]

	if len(choice.ToolCalls) == 0 {
		return answer, fmt.Errorf("no tool calls found in response")
	}

	// Get first tool call
	toolCall := choice.ToolCalls[0]

	// Verify it's our tool
	if toolCall.FunctionCall == nil {
		return answer, fmt.Errorf("tool call has no function call")
	}

	if toolCall.FunctionCall.Name != "reply_question" {
		return answer, fmt.Errorf("unexpected function call: %s", toolCall.FunctionCall.Name)
	}

	// Parse arguments (JSON string) into LLMAnswer
	if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &answer); err != nil {
		return answer, fmt.Errorf("failed to unmarshal arguments: %w", err)
	}

	// Clean up code snippet
	answer.CodeSnippet = mergeNewlines(answer.CodeSnippet)

	return answer, nil
}

func (c *Client) AskLLMAboutCode(
	codePath string,
	questions []LLMQuestion,
	subPathsOnly []string,
) ([]LLMAnswer, error) {

	if len(questions) == 0 {
		return nil, fmt.Errorf("No questions provided")
	}
	logme.DebugFln("llmvalidate: Using code path %s with %d questions", codePath, len(questions))

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

	filesPrompt := fmt.Sprintf(
		`The files in the repository are: %s `,
		strings.Join(codePrompt, "\n"),
	)

	var answers []LLMAnswer = make([]LLMAnswer, 0, len(questions))

	for _, question := range questions {
		var answer LLMAnswer
		var err error

		for retries := 3; retries > 0; retries-- {
			answer, err = c.askModelQuestion(filesPrompt, question)
			if err == nil {
				break
			}
			logme.DebugFln("Error generating answer: %v", err)
		}

		if err != nil {
			return nil, fmt.Errorf("Failed to generate answer after 3 retries: %w", err)
		}

		answers = append(answers, answer)
	}

	return answers, nil

}

func (c *Client) askModelQuestion(
	filesPrompt string,
	question LLMQuestion,
) (LLMAnswer, error) {
	// Create system prompt
	systemPrompt := `You are a source code reviewer. You are provided with source code repository information and files. You will answer questions only based on the context of the files provided.

IMPORTANT: You MUST use the reply_question tool to provide your answer. Do not respond with plain text.

REVIEWER NOTE: Ignore code that exists only for testing or development:
- Test files (*_test.go, *_spec.ts, etc.)
- Development scripts and utilities
- Dockerfiles, makefiles, bash scripts
- Files clearly not part of the plugin

Focus your review on production code that will run as part of a Grafana Plugin.`

	// Build prompt with files and question
	userPrompt := fmt.Sprintf("%s\n\nAnswer this question based on the files: %s", filesPrompt, question.Question)

	// Create message history
	messageHistory := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, userPrompt),
	}

	// Generate response with tools
	resp, err := c.llm.GenerateContent(
		c.ctx,
		messageHistory,
		llms.WithTools([]llms.Tool{createReplyQuestionTool()}),
	)
	if err != nil {
		logme.DebugFln("Error generating content: %v", err)
		return LLMAnswer{}, err
	}

	// Extract answer from tool call
	answer, err := extractAnswerFromToolCall(resp)
	if err != nil {
		return LLMAnswer{}, err
	}

	// Set expected answer from question
	answer.ExpectedShortAnswer = question.ExpectedAnswer

	logme.DebugFln("Answer: %v", prettyprint.SPrint(answer))

	return answer, nil
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

func mergeNewlines(s string) string {
	re := regexp.MustCompile(`\n+`)
	return re.ReplaceAllString(s, "\n")
}
