package llmclient

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	piAgentTimeout = 5 * time.Minute
)

//go:embed pi-extension/extension.ts
var piExtensionTS []byte

// AnswerSchema represents the structured response from the agentic client
type AnswerSchema struct {
	Answer      string   `json:"answer"`
	ShortAnswer bool     `json:"short_answer"`
	Files       []string `json:"files,omitempty"`
	CodeSnippet string   `json:"code_snippet,omitempty"`
}

// AgenticCallOptions contains configuration for the agentic LLM call
type AgenticCallOptions struct {
	Model    string // e.g. "gemini-2.5-flash"
	Provider string // "google", "anthropic", "openai"
	APIKey   string
}

// AgenticClient is an interface for agentic LLM interactions
type AgenticClient interface {
	CallLLM(ctx context.Context, prompt, repositoryPath string) ([]AnswerSchema, error)
}

// agenticClientImpl implements AgenticClient using pi in RPC mode
type agenticClientImpl struct {
	apiKey   string
	model    string
	provider string
}

// NewAgenticClient creates a new AgenticClient with the given options
func NewAgenticClient(opts *AgenticCallOptions) (AgenticClient, error) {
	if opts == nil {
		return nil, fmt.Errorf("options are required")
	}
	if opts.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if opts.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if opts.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	return &agenticClientImpl{
		apiKey:   opts.APIKey,
		model:    opts.Model,
		provider: opts.Provider,
	}, nil
}

// piModelString converts provider/model to pi's --model format.
// Pi uses "provider/model" format where provider names match pi's model registry
// (e.g. "google/gemini-2.5-flash", "anthropic/claude-sonnet-4-5").
func piModelString(provider, model string) string {
	return provider + "/" + model
}

// apiKeyEnvVar returns the environment variable name for the given provider
func apiKeyEnvVar(provider string) string {
	switch provider {
	case "google":
		return "GEMINI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "openai":
		return "OPENAI_API_KEY"
	default:
		return strings.ToUpper(provider) + "_API_KEY"
	}
}

// writeExtensionFile writes the embedded extension to a temp file and returns its path.
// The caller is responsible for cleaning up the returned directory.
func writeExtensionFile() (extensionPath string, cleanupDir string, err error) {
	dir, err := os.MkdirTemp("", "pi-extension-*")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	p := filepath.Join(dir, "extension.ts")
	if err := os.WriteFile(p, piExtensionTS, 0644); err != nil {
		os.RemoveAll(dir)
		return "", "", fmt.Errorf("failed to write extension file: %w", err)
	}

	return p, dir, nil
}

// rpcEvent represents a generic JSON event from pi's RPC stdout.
// We only parse the fields we care about.
type rpcEvent struct {
	Type     string `json:"type"`
	ToolName string `json:"toolName,omitempty"`
	Args     json.RawMessage `json:"args,omitempty"`
	Result   *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content,omitempty"`
		Details json.RawMessage `json:"details,omitempty"`
	} `json:"result,omitempty"`
	IsError bool `json:"isError,omitempty"`

	// For response events
	Command string `json:"command,omitempty"`
	Success *bool  `json:"success,omitempty"`
	Error   string `json:"error,omitempty"`

	// For message events (message_start, message_end, turn_end)
	Message *rpcMessage `json:"message,omitempty"`
	
	// For text content in various events
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content,omitempty"`
}

// rpcMessage captures relevant fields from assistant messages in events.
type rpcMessage struct {
	Role         string `json:"role,omitempty"`
	StopReason   string `json:"stopReason,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

const systemPrompt = `You are a code analysis assistant. You have tools to explore code in a repository.

STRATEGY:
1. Use bash to list files (ls) and explore the repository structure
2. Use the read tool to read source code files
3. Use bash to run git commands (git diff, git log, etc.) and grep/rg for searching
4. Analyze the code to answer the question

IMPORTANT: You are in non-interactive mode. Start working and using your tools immediately.
When ready, use submit_answer to provide your structured answer.
For multiple questions, call submit_answer once per question.
You MUST call submit_answer - do NOT just respond with text.`

// piProcess holds the state of a running pi RPC subprocess.
type piProcess struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	scanner *bufio.Scanner
}

// startPiProcess spawns pi in RPC mode and returns a handle to interact with it.
func (c *agenticClientImpl) startPiProcess(
	ctx context.Context,
	repositoryPath, extensionPath string,
) (*piProcess, error) {
	piModel := piModelString(c.provider, c.model)
	args := []string{
		"-y", "@mariozechner/pi-coding-agent",
		"--mode", "rpc",
		"--no-session",
		"--no-extensions",
		"--no-skills",
		"--no-prompt-templates",
		"-e", extensionPath,
		"--model", piModel,
		"--system-prompt", systemPrompt,
	}

	debugLog("AgenticClient: spawning pi with args: npx %s", strings.Join(args, " "))

	cmd := exec.CommandContext(ctx, "npx", args...)
	cmd.Dir = repositoryPath
	// Use minimal environment to avoid leaking sensitive parent env vars
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"TMPDIR=" + os.Getenv("TMPDIR"),
		apiKeyEnvVar(c.provider) + "=" + c.apiKey,
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start pi: %w", err)
	}

	// Drain stderr in background for debug logging
	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			debugLog("pi stderr: %s", s.Text())
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	return &piProcess{cmd: cmd, stdin: stdin, scanner: scanner}, nil
}

// sendPrompt writes a prompt command to pi's stdin.
func (p *piProcess) sendPrompt(prompt string) error {
	promptCmd := map[string]string{
		"type":    "prompt",
		"message": prompt,
	}
	promptJSON, err := json.Marshal(promptCmd)
	if err != nil {
		return fmt.Errorf("failed to marshal prompt command: %w", err)
	}

	debugLog("AgenticClient: RPC << %s", truncateString(string(promptJSON), 500))
	if _, err := fmt.Fprintf(p.stdin, "%s\n", promptJSON); err != nil {
		return fmt.Errorf("failed to write prompt to pi: %w", err)
	}
	return nil
}

// eventLoopResult holds the outcome of one event-reading pass.
type eventLoopResult struct {
	answers  []AnswerSchema
	gotError bool
	lastErr  string
	fatalErr error
}

// printEventSummary outputs a human-readable summary of the event.
func printEventSummary(event rpcEvent) {
	switch event.Type {
	case "tool_execution_start":
		argsPreview := ""
		if len(event.Args) > 0 && len(event.Args) < 100 {
			argsPreview = fmt.Sprintf(" %s", string(event.Args))
		} else if len(event.Args) >= 100 {
			argsPreview = fmt.Sprintf(" %s...", truncateString(string(event.Args), 80))
		}
		debugLog("🔧 Agent calling tool: %s%s", event.ToolName, argsPreview)
	case "tool_execution_end":
		if event.IsError {
			debugLog("❌ Tool %s failed", event.ToolName)
		} else if event.ToolName == "submit_answer" {
			debugLog("✅ Agent submitted answer")
		} else {
			resultPreview := ""
			if event.Result != nil && len(event.Result.Content) > 0 {
				for _, c := range event.Result.Content {
					if c.Type == "text" && c.Text != "" {
						resultPreview = truncateString(c.Text, 60)
						break
					}
				}
			}
			if resultPreview != "" {
				debugLog("   → %s", resultPreview)
			}
		}
	case "text":
		for _, c := range event.Content {
			if c.Type == "text" && c.Text != "" {
				debugLog("💭 Agent: %s", truncateString(c.Text, 100))
			}
		}
	case "agent_end":
		debugLog("🏁 Agent finished")
	}
}

// readEvents reads RPC events from pi until agent_end, collecting answers
// and tracking errors.
func (p *piProcess) readEvents() eventLoopResult {
	var result eventLoopResult

	for p.scanner.Scan() {
		line := p.scanner.Text()
		if line == "" {
			continue
		}

		var event rpcEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			debugLog(
				"AgenticClient: failed to parse event: %v (line: %s)",
				err,
				truncateString(line, 200),
			)
			continue
		}

		printEventSummary(event)

		if event.Type == "response" && event.Success != nil && !*event.Success {
			debugLog("AgenticClient: pi error response: %s", event.Error)
			result.fatalErr = fmt.Errorf("pi error: %s", event.Error)
			return result
		}

		// Track errors from message events (e.g. 429 rate limits)
		if event.Message != nil && event.Message.StopReason == "error" &&
			event.Message.ErrorMessage != "" {
			result.lastErr = event.Message.ErrorMessage
			result.gotError = true
		}

		// Collect answers from submit_answer tool execution
		if event.Type == "tool_execution_end" && event.ToolName == "submit_answer" &&
			!event.IsError {
			if event.Result != nil && event.Result.Details != nil {
				var answer AnswerSchema
				if err := json.Unmarshal(event.Result.Details, &answer); err != nil {
					debugLog("AgenticClient: failed to parse submit_answer details: %v", err)
					continue
				}
				debugLog("AgenticClient: received answer #%d: short_answer=%v, answer=%s",
					len(result.answers)+1, answer.ShortAnswer, truncateString(answer.Answer, 100))
				result.answers = append(result.answers, answer)
			}
		}

		if event.Type == "agent_end" {
			debugLog("AgenticClient: agent_end received, %d answers collected", len(result.answers))
			return result
		}
	}

	// Check if scanner exited due to an error (e.g., line too long)
	if err := p.scanner.Err(); err != nil {
		result.fatalErr = fmt.Errorf("scanner error reading pi output: %w", err)
	}

	return result
}

// close shuts down the pi process.
func (p *piProcess) close() error {
	p.stdin.Close()
	return p.cmd.Wait()
}

const maxRetries = 3

// CallLLM spawns pi in RPC mode, sends the prompt, and collects structured answers.
// Retries up to maxRetries times on transient errors (e.g. rate limits), reusing the
// same pi session.
func (c *agenticClientImpl) CallLLM(
	ctx context.Context,
	prompt, repositoryPath string,
) ([]AnswerSchema, error) {
	printDebugLogPath()
	debugLog("\n\n\n")
	debugLog("################################################################")
	debugLog("# NEW CallLLM (pi RPC) - provider=%s model=%s", c.provider, c.model)
	debugLog("# repo=%s", repositoryPath)
	debugLog("# prompt=%s", truncateString(prompt, 200))
	debugLog("################################################################")

	extensionPath, cleanupDir, err := writeExtensionFile()
	if err != nil {
		return nil, fmt.Errorf("failed to write pi extension: %w", err)
	}
	defer os.RemoveAll(cleanupDir)

	ctx, cancel := context.WithTimeout(ctx, piAgentTimeout)
	defer cancel()

	proc, err := c.startPiProcess(ctx, repositoryPath, extensionPath)
	if err != nil {
		return nil, err
	}
	defer proc.close()

	var lastError string
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var promptToSend string
		if attempt == 0 {
			promptToSend = prompt
		} else {
			// On retry, send a nudge instead of repeating the same prompt
			debugLog("AgenticClient: retry %d/%d after error: %s", attempt, maxRetries, lastError)
			time.Sleep(time.Duration(attempt) * 2 * time.Second)
			promptToSend = "The previous attempt encountered an error. Please try again and complete the task by calling submit_answer with your analysis."
		}

		if err := proc.sendPrompt(promptToSend); err != nil {
			return nil, err
		}

		result := proc.readEvents()
		if result.fatalErr != nil {
			return nil, result.fatalErr
		}
		if len(result.answers) > 0 {
			debugLog("AgenticClient: returning %d answers", len(result.answers))
			return result.answers, nil
		}

		lastError = result.lastErr
		if !result.gotError || attempt >= maxRetries {
			break
		}
	}

	if err := proc.scanner.Err(); err != nil {
		debugLog("AgenticClient: scanner error: %v", err)
	}

	if ctx.Err() == context.DeadlineExceeded {
		return nil, fmt.Errorf("pi timed out after %v", piAgentTimeout)
	}

	if lastError != "" {
		return nil, fmt.Errorf(
			"pi agent did not produce any answers (last error: %s)",
			truncateString(lastError, 200),
		)
	}
	return nil, fmt.Errorf("pi agent did not produce any answers")
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
