package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	maxToolCalls            = 100
	maxLLMRetries           = 3
	maxConsecutiveNoTools   = 5
	retryDelay              = 2 * time.Second
)

// AnswerSchema represents the structured response from the agentic client
type AnswerSchema struct {
	Answer      string   `json:"answer"`
	ShortAnswer bool     `json:"short_answer"`
	Files       []string `json:"files,omitempty"`
	CodeSnippet string   `json:"code_snippet,omitempty"`
}

// AgenticCallOptions contains configuration for the agentic LLM call
type AgenticCallOptions struct {
	Model    string // e.g. "gemini-2.0-flash"
	Provider string // "google", "anthropic", "openai"
	APIKey   string
}

// AgenticClient is an interface for agentic LLM interactions
type AgenticClient interface {
	CallLLM(ctx context.Context, prompt, repositoryPath string) ([]AnswerSchema, error)
}

// agenticClientImpl implements AgenticClient
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

// CallLLM executes an agentic loop with tools to answer questions about code.
// The prompt may contain multiple questions, in which case the agent will call
// submit_answer multiple times. All answers are collected and returned.
func (c *agenticClientImpl) CallLLM(ctx context.Context, prompt, repositoryPath string) ([]AnswerSchema, error) {
	// Initialize LLM based on provider using the client's configured settings
	opts := &AgenticCallOptions{
		APIKey:   c.apiKey,
		Model:    c.model,
		Provider: c.provider,
	}
	llm, err := initLLM(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	// Build tools
	tools := buildAgenticTools()

	// Create tool executor
	executor := newToolExecutor(repositoryPath)

	// System prompt
	systemPrompt := `You are a code analysis assistant. You have tools to explore code in a repository.

AVAILABLE TOOLS:
- list_directory: List files at a path. Use "." for root.
- read_file: Read a file's contents. This is your primary tool for understanding code.
- grep: Search for a pattern across files.
- git: Run read-only git commands (log, show, diff, status, etc.)
- submit_answer: Submit your final answer.

STRATEGY:
1. Use list_directory to see what files exist
2. Use read_file to read the source code files
3. Analyze the code to answer the question

You can only use one tool at a time.
IMPORTANT: You are in non-interactive mode. Start working and using your tools immediately.
When ready, use submit_answer. For multiple questions, call submit_answer once per question.`

	// Build initial messages
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}

	// Collect answers
	var answers []AnswerSchema

	// Agentic loop
	toolCallsRemaining := maxToolCalls

	// Print debug log file path before starting the loop
	printDebugLogPath()
	debugLog("\n\n\n")
	debugLog("################################################################")
	debugLog("# NEW CallLLM - provider=%s model=%s", c.provider, c.model)
	debugLog("# repo=%s", repositoryPath)
	debugLog("# prompt=%s", truncateString(prompt, 200))
	debugLog("################################################################")

	iteration := 0
	consecutiveNoTools := 0
	for toolCallsRemaining > 0 {
		iteration++
		debugLog("========== AgenticClient: iteration %d ==========", iteration)
		debugLog("AgenticClient: %d tool calls remaining, %d answers collected", toolCallsRemaining, len(answers))

		// Call LLM with retry logic
		debugLog("AgenticClient: calling LLM...")
		resp, err := callLLMWithRetry(ctx, llm, messages, tools)
		if err != nil {
			debugLog("AgenticClient: LLM call failed: %v", err)
			return nil, fmt.Errorf("LLM call failed after %d retries: %w", maxLLMRetries, err)
		}

		// resp.Choices contains the LLM's response options. Each choice has Content (text)
		// and/or ToolCalls (function calls the model wants to make). Typically there's
		// only one choice unless you request multiple completions.
		if len(resp.Choices) == 0 {
			debugLog("AgenticClient: no choices in response")
			return nil, fmt.Errorf("no response from LLM")
		}

		// Use first choice. Google puts all tool calls in choices[0].ToolCalls.
		// Anthropic creates a separate choice per content block (text or tool_use),
		// but langchaingo's handleAIMessage only supports Parts[0] as either
		// TextContent or ToolCall, so we process one choice at a time.
		choice := resp.Choices[0]
		debugLog("AgenticClient: received response with %d tool calls", len(choice.ToolCalls))
		if choice.Content != "" {
			debugLog("AgenticClient: AI message: %s", truncateString(choice.Content, 200))
		}

		// If no tool calls, check if we have answers
		if len(choice.ToolCalls) == 0 {
			debugLog("AgenticClient: no tool calls in response")

			// If we have collected answers, the agent is done
			if len(answers) > 0 {
				debugLog("AgenticClient: agent finished with %d answers", len(answers))
				return answers, nil
			}

			consecutiveNoTools++
			debugLog("AgenticClient: consecutive no-tool responses: %d/%d", consecutiveNoTools, maxConsecutiveNoTools)
			if consecutiveNoTools >= maxConsecutiveNoTools {
				return nil, fmt.Errorf("agent failed to use tools after %d consecutive attempts", maxConsecutiveNoTools)
			}

			// No answers yet - add the AI response and remind to use tools
			if choice.Content != "" {
				messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, choice.Content))
			}
			debugLog("AgenticClient: no answers yet, reminding agent to use tools")
			messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman,
				"You are in non-interactive mode. You must start using your tools now to explore the repository. When you have enough information, use submit_answer to provide your answer."))
			toolCallsRemaining--
			continue
		}

		// Reset consecutive no-tool counter when tools are used
		consecutiveNoTools = 0

		// Build AI message with tool calls
		aiMessage := llms.MessageContent{
			Role: llms.ChatMessageTypeAI,
		}
		if choice.Content != "" {
			aiMessage.Parts = append(aiMessage.Parts, llms.TextContent{Text: choice.Content})
		}
		for _, toolCall := range choice.ToolCalls {
			aiMessage.Parts = append(aiMessage.Parts, toolCall)
		}
		messages = append(messages, aiMessage)

		// Process tool calls
		for i, toolCall := range choice.ToolCalls {
			toolCallsRemaining--
			response, answer := processToolCall(toolCall, i, len(choice.ToolCalls), len(answers), executor)
			messages = append(messages, response)
			if answer != nil {
				answers = append(answers, *answer)
			}
		}
	}

	// If we collected some answers but ran out of tool calls, return what we have
	if len(answers) > 0 {
		debugLog("AgenticClient: ran out of tool calls, returning %d answers", len(answers))
		return answers, nil
	}

	return nil, fmt.Errorf("exceeded maximum tool calls (%d), agent did not complete", maxToolCalls)
}

// processToolCall processes a single tool call and returns the response message and optional answer
func processToolCall(toolCall llms.ToolCall, index, total, currentAnswerCount int, executor *toolExecutor) (llms.MessageContent, *AnswerSchema) {
	debugLog("AgenticClient: [%d/%d] executing tool: %s", index+1, total, toolCall.FunctionCall.Name)
	debugLog("AgenticClient: tool args: %s", truncateString(toolCall.FunctionCall.Arguments, 500))

	// Check for submit_answer
	if toolCall.FunctionCall.Name == "submit_answer" {
		var answer AnswerSchema
		if err := json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &answer); err != nil {
			debugLog("AgenticClient: failed to parse submit_answer: %v", err)
			// Report parse error back to agent so it can retry
			return llms.MessageContent{
				Role: llms.ChatMessageTypeTool,
				Parts: []llms.ContentPart{
					llms.ToolCallResponse{
						ToolCallID: toolCall.ID,
						Name:       toolCall.FunctionCall.Name,
						Content:    fmt.Sprintf("Error parsing answer: %v. Please try again with valid JSON.", err),
					},
				},
			}, nil
		}
		debugLog("AgenticClient: received answer #%d: short_answer=%v, answer=%s",
			currentAnswerCount+1, answer.ShortAnswer, truncateString(answer.Answer, 100))

		// Return success response and the answer
		return llms.MessageContent{
			Role: llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{
				llms.ToolCallResponse{
					ToolCallID: toolCall.ID,
					Name:       toolCall.FunctionCall.Name,
					Content:    "Answer recorded successfully. If you have answered all questions, respond with a plain text message saying 'I am finished'. Otherwise, continue with the next question.",
				},
			},
		}, &answer
	}

	// Execute other tools
	result := executor.execute(toolCall.FunctionCall.Name, toolCall.FunctionCall.Arguments)
	debugLog("AgenticClient: tool result: %s", truncateString(result, 300))

	return llms.MessageContent{
		Role: llms.ChatMessageTypeTool,
		Parts: []llms.ContentPart{
			llms.ToolCallResponse{
				ToolCallID: toolCall.ID,
				Name:       toolCall.FunctionCall.Name,
				Content:    result,
			},
		},
	}, nil
}

// callLLMWithRetry calls the LLM with retry logic for transient errors
func callLLMWithRetry(ctx context.Context, llm llms.Model, messages []llms.MessageContent, tools []llms.Tool) (*llms.ContentResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= maxLLMRetries; attempt++ {
		resp, err := llm.GenerateContent(ctx, messages, llms.WithTools(tools))
		if err == nil {
			return resp, nil
		}
		lastErr = err
		debugLog("AgenticClient: LLM call failed (attempt %d/%d): %v", attempt, maxLLMRetries, err)

		if attempt < maxLLMRetries {
			debugLog("AgenticClient: retrying in %v...", retryDelay)
			time.Sleep(retryDelay)
		}
	}
	return nil, lastErr
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// initLLM initializes the appropriate LLM based on provider
func initLLM(ctx context.Context, opts *AgenticCallOptions) (llms.Model, error) {
	switch opts.Provider {
	case "google":
		return googleai.New(
			ctx,
			googleai.WithAPIKey(opts.APIKey),
			googleai.WithDefaultModel(opts.Model),
		)
	case "anthropic":
		return anthropic.New(
			anthropic.WithToken(opts.APIKey),
			anthropic.WithModel(opts.Model),
		)
	case "openai":
		return openai.New(
			openai.WithToken(opts.APIKey),
			openai.WithModel(opts.Model),
		)
	default:
		return nil, fmt.Errorf("unsupported provider: %s (supported: google, anthropic, openai)", opts.Provider)
	}
}
