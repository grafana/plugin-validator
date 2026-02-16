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
	"github.com/joakimcarlsson/ai/message"
	"github.com/joakimcarlsson/ai/model"
	llm "github.com/joakimcarlsson/ai/providers"
	"github.com/joakimcarlsson/ai/schema"
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

// Config defines the LLM provider configuration for code validation
type Config struct {
	Model  model.Model // The LLM model to use (e.g., model.GeminiModels[model.Gemini3Flash])
	APIKey string      // Provider API key
}

type Client struct {
	llmClient llm.LLM
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

func New(ctx context.Context, config Config) (*Client, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	// Model validation - check if it's a zero value
	if config.Model.ID == "" {
		return nil, fmt.Errorf("Model is required")
	}

	logme.DebugFln("llmvalidate: Using provider %s with model %s", config.Model.Provider, config.Model.Name)

	// Create LLM client with the model object
	llmClient, err := llm.NewLLM(
		config.Model.Provider,
		llm.WithAPIKey(config.APIKey),
		llm.WithModel(config.Model),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Verify structured output support
	if !llmClient.SupportsStructuredOutput() {
		return nil, fmt.Errorf("model %s does not support structured output", config.Model.Name)
	}

	return &Client{
		llmClient: llmClient,
		ctx:       ctx,
	}, nil
}

func (c *Client) AskLLMAboutCode(
	codePath string,
	questions []LLMQuestion,
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

	filesPrompt := fmt.Sprintf(
		`The files in the repository are: %s `,
		strings.Join(codePrompt, "\n"),
	)

	// Define structured output schema for code question answers
	outputSchema := schema.NewStructuredOutputInfo(
		"code_question_answer",
		"Answer a question about code with structured output",
		map[string]any{
			"question": map[string]any{
				"type":        "string",
				"description": "The question to answer",
			},
			"answer": map[string]any{
				"type":        "string",
				"description": "The full answer to the question. Elaborate why yes or no",
			},
			"files": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "string",
				},
				"description": "An array of files related to the answer. Only if applicable",
			},
			"short_answer": map[string]any{
				"type":        "boolean",
				"description": "True or false",
			},
			"code_snippet": map[string]any{
				"type":        "string",
				"description": "Code snippet as context for the answer. Only if applicable",
			},
		},
		[]string{"question", "answer", "short_answer"},
	)

	// Create system message with instructions
	systemMsg := message.NewSystemMessage(`You are source code reviewer. You are provided with a source code repository information and files. You will answer questions only based on the context of the files provided. The output must be a valid JSON object

REVIEWER NOTE: Ignore code that exists only for testing or to setup development:
	- Test files (*_test.go, *_spec.ts, etc.)
	- Scripts dedicated  for development (e.g. test servers, seeding)
	- Code that is clearly marked as development-only with comments
	- Local development servers and database setup utilities
	- dockerfiles
	- make files, bash scripts, etc.
	- files clearly not part of the plugin and intended for development

Focus your review on code that will run in production environments as part of a Grafana Plugin`)

	// Log files prompt length for debugging (token counting not available for all providers)
	logme.DebugFln("llmvalidate: Files prompt length: %d characters", len(filesPrompt))

	var answers []LLMAnswer = make([]LLMAnswer, 0, len(questions))

	for _, question := range questions {
		var answer LLMAnswer
		var err error

		for retries := 3; retries > 0; retries-- {
			answer, err = c.askModelQuestion(systemMsg, filesPrompt, question, outputSchema)
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
	systemMsg message.Message,
	filesPrompt string,
	question LLMQuestion,
	outputSchema *schema.StructuredOutputInfo,
) (LLMAnswer, error) {
	questionPrompt := fmt.Sprintf(
		"%s\n\n Answer this question based on the previous files: %s",
		filesPrompt,
		question.Question,
	)

	messages := []message.Message{
		systemMsg,
		message.NewUserMessage(questionPrompt),
	}

	var answer LLMAnswer
	modelResponse, err := c.llmClient.SendMessagesWithStructuredOutput(
		c.ctx,
		messages,
		nil, // no tools
		outputSchema,
	)
	if err != nil {
		logme.DebugFln("Error generating content: %v", err)
		return answer, err
	}

	// Parse structured output from response
	if modelResponse.StructuredOutput == nil {
		return answer, fmt.Errorf("no structured output in response")
	}

	err = json.Unmarshal([]byte(*modelResponse.StructuredOutput), &answer)
	if err != nil {
		logme.DebugFln("Failed to unmarshal structured output: %v", *modelResponse.StructuredOutput)
		return answer, err
	}

	// Clean up code snippet formatting
	answer.CodeSnippet = mergeNewlines(answer.CodeSnippet)
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
