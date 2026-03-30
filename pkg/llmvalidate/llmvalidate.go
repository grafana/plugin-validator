package llmvalidate

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/danwakefield/fnmatch"
	"github.com/grafana/plugin-validator/pkg/llmclient"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/prettyprint"
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

const reviewerSystemPrompt = `You are a source code reviewer. You are provided with source code repository information and files. You will answer questions only based on the context of the files provided.

REVIEWER NOTE: Ignore code that exists only for testing or development:
- Test files (*_test.go, *_spec.ts, etc.)
- Development scripts and utilities
- Dockerfiles, makefiles, bash scripts
- Files clearly not part of the plugin

Focus your review on production code that will run as part of a Grafana Plugin.

RESPONSE FORMAT: Be extremely concise. This is a purely investigative task — report findings only.
1. Start with "Yes" or "No".
2. If Yes, add ONE sentence explaining the issue.
3. For code_snippet, include ONLY the single most relevant snippet (max 5 lines). Do NOT repeat similar patterns — if the same issue appears in multiple places, just list the files.
4. Never include full function bodies. Show only the specific problematic line(s).

Do NOT:
- Suggest fixes or improvements
- Explain how to resolve the issue
- Discuss mitigations, workarounds, or risk levels
- Offer recommendations or next steps
- Provide context about why the pattern is problematic
- Qualify findings with "however" or "the risk is mitigated by"

Your sole job is to answer Yes/No and point to the evidence. Nothing more.

Example good answer: "Yes, user input flows into template.Execute via BuildUserPrompt."
Example bad answer: A multi-paragraph explanation with full function bodies, mitigations, suggestions, and repeated code blocks.`

type Client struct {
	provider  string
	modelName string
	apiKey    string
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

	return &Client{
		provider:  provider,
		modelName: modelName,
		apiKey:    apiKey,
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
		"The files in the repository are:\n%s",
		strings.Join(codePrompt, "\n"),
	)

	// Combine reviewer instructions and file contents into the system prompt.
	// This allows providers like Anthropic to cache the system prompt across
	// multiple question calls, avoiding re-processing the file contents each time.
	combinedSystemPrompt := fmt.Sprintf("%s\n\n%s", reviewerSystemPrompt, filesPrompt)

	agenticClient, err := llmclient.NewAgenticClient(&llmclient.AgenticCallOptions{
		Model:        c.modelName,
		Provider:     c.provider,
		APIKey:       c.apiKey,
		ToolSet:      llmclient.NoTools,
		SystemPrompt: combinedSystemPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agentic client: %w", err)
	}

	var answers []LLMAnswer = make([]LLMAnswer, 0, len(questions))

	for _, question := range questions {
		userPrompt := fmt.Sprintf("Answer this question based on the source code files provided in the system prompt: %s", question.Question)

		agenticAnswers, err := agenticClient.CallLLM(c.ctx, []string{userPrompt}, absCodePath)
		if err != nil {
			logme.DebugFln("Error calling LLM for question %q: %v. Skipping.", question.Question, err)
			continue
		}

		if len(agenticAnswers) == 0 {
			logme.DebugFln("No answer returned from LLM for question: %s. Skipping.", question.Question)
			continue
		}

		agenticAnswer := agenticAnswers[0]

		// Check if the agentic client reported an error for this question
		if agenticAnswer.Error != "" {
			logme.DebugFln("LLM error for question %q: %s. Skipping.", question.Question, agenticAnswer.Error)
			continue
		}

		answer := LLMAnswer{
			Question:            agenticAnswer.Question,
			Answer:              agenticAnswer.Answer,
			ShortAnswer:         agenticAnswer.ShortAnswer,
			Files:               agenticAnswer.Files,
			CodeSnippet:         mergeNewlines(agenticAnswer.CodeSnippet),
			ExpectedShortAnswer: question.ExpectedAnswer,
		}

		logme.DebugFln("Answer: %v", prettyprint.SPrint(answer))

		answers = append(answers, answer)
	}

	return answers, nil

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
