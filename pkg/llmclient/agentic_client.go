package llmclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/anthropic"
	"github.com/tmc/langchaingo/llms/googleai"
	"github.com/tmc/langchaingo/llms/openai"
)

const (
	maxToolCallsFirstQuestion = 60
	maxToolCallsFollowUp      = 20
	maxLLMRetries             = 3
	maxConsecutiveNoTools     = 5
	retryDelay                = 2 * time.Second

	systemPrompt = `You are a code analysis assistant. You have tools to explore code in a repository.

AVAILABLE TOOLS:
- list_directory: List files at a path. Use "." for root.
- read_file: Read a file's contents. This is your primary tool for understanding code.
- grep: Search for a pattern across files.
- git: Run read-only git commands (log, show, diff, status, etc.)
- submit_answer: Submit your answers.

STRATEGY:
1. Use list_directory to see what files exist
2. Use read_file to read the source code files
3. Analyze the code to answer the question

You can only use one tool at a time.
IMPORTANT: You are in non-interactive mode. No one will read your text answers, only tools.
When you have gathered enough information, use submit_answer to provide your answer.`

	budgetNudgePrompt = `You have only %d tool calls remaining. Wrap up your investigation and call submit_answer now with whatever information you have gathered so far.`

	useToolsReminderPrompt = `You are in non-interactive mode. You must start using your tools now to explore the repository. When you have enough information, use submit_answer to provide your answer.`

	submitAnswerAloneError = `Error: submit_answer must be called alone. When you have an answer, call submit_answer as a single tool call without any other tools in the same response.`
)

// AnswerSchema represents the structured response from the agentic client
type AnswerSchema struct {
	Question    string   `json:"question"`
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
	CallLLM(ctx context.Context, questions []string, repositoryPath string) ([]AnswerSchema, error)
}

// agenticClientImpl implements AgenticClient
type agenticClientImpl struct {
	apiKey   string
	model    string
	provider string
	tools    []llms.Tool
	executor *toolExecutor
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
// Each question is processed sequentially, with follow-up questions benefiting
// from the context accumulated by earlier questions.
func (c *agenticClientImpl) CallLLM(
	ctx context.Context,
	questions []string,
	repositoryPath string,
) ([]AnswerSchema, error) {
	if len(questions) == 0 {
		return nil, fmt.Errorf("at least one question is required")
	}

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

	// Initialize tools and executor for this repository
	c.tools = buildAgenticTools()
	c.executor = newToolExecutor(repositoryPath)

	// Build initial messages with system prompt only (no user message yet)
	messages := []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeSystem, systemPrompt),
	}

	// Print debug log file path before starting the loop
	printDebugLogPath()
	debugLog("\n\n\n")
	debugLog("################################################################")
	debugLog("# NEW CallLLM - provider=%s model=%s", c.provider, c.model)
	debugLog("# repo=%s", repositoryPath)
	debugLog("# questions=%d", len(questions))
	debugLog("################################################################")

	// Collect answers
	var answers []AnswerSchema

	// Process each question sequentially
	for questionIndex, question := range questions {
		debugLog(
			"\n========== Processing question %d/%d ==========",
			questionIndex+1,
			len(questions),
		)
		debugLog("Question: %s", truncateString(question, 200))

		originalQuestion := question

		// Determine budget for this question
		toolsBudget := maxToolCallsFirstQuestion
		if questionIndex > 0 {
			toolsBudget = maxToolCallsFollowUp
		}
		debugLog("Budget: %d tool calls", toolsBudget)

		// Add the question as a human message
		messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, question))

		// Run the question loop
		updatedMessages, answer, err := c.runQuestionLoop(
			ctx,
			llm,
			messages,
			toolsBudget,
			questionIndex,
		)
		messages = updatedMessages

		if err != nil {
			// Return partial results on error
			debugLog("AgenticClient: question %d failed: %v", questionIndex+1, err)
			if len(answers) > 0 {
				debugLog("AgenticClient: returning %d partial answers", len(answers))
				return answers, nil
			}
			return nil, err
		}

		if answer != nil {
			// Set the question field
			answer.Question = originalQuestion
			answers = append(answers, *answer)
			debugLog("AgenticClient: collected answer %d/%d", len(answers), len(questions))
		} else {
			// Budget exhausted without answer - stop processing further questions
			debugLog("AgenticClient: question %d exhausted budget without answer, stopping", questionIndex+1)
			if len(answers) > 0 {
				debugLog("AgenticClient: returning %d partial answers", len(answers))
				return answers, nil
			}
			return nil, fmt.Errorf("question %d exhausted budget without providing answer", questionIndex+1)
		}
	}

	debugLog("AgenticClient: successfully answered all %d questions", len(questions))
	return answers, nil
}

// runQuestionLoop runs the tool-calling loop for a single question.
// Returns updated messages, the answer (or nil if budget exhausted), and error.
func (c *agenticClientImpl) runQuestionLoop(
	ctx context.Context,
	llm llms.Model,
	messages []llms.MessageContent,
	toolsBudget int,
	questionIndex int,
) ([]llms.MessageContent, *AnswerSchema, error) {
	toolCallsRemaining := toolsBudget
	consecutiveNoTools := 0
	iteration := 0

	budgetNudged := false

	for toolCallsRemaining > 0 {
		iteration++
		debugLog("========== Question %d iteration %d ==========", questionIndex+1, iteration)
		debugLog("AgenticClient: %d tool calls remaining", toolCallsRemaining)

		if !budgetNudged && toolCallsRemaining <= 5 {
			budgetNudged = true
			debugLog("AgenticClient: nudging model about low budget")
			messages = append(messages, llms.TextParts(
				llms.ChatMessageTypeHuman,
				fmt.Sprintf(budgetNudgePrompt, toolCallsRemaining),
			))
		}

		// Call LLM with retry logic
		debugLog("AgenticClient: calling LLM...")
		resp, err := c.callLLMWithRetry(ctx, llm, messages)
		if err != nil {
			debugLog("AgenticClient: LLM call failed: %v", err)
			return messages, nil, fmt.Errorf(
				"LLM call failed after %d retries: %w",
				maxLLMRetries,
				err,
			)
		}

		if len(resp.Choices) == 0 {
			debugLog("AgenticClient: no choices in response")
			return messages, nil, fmt.Errorf("no response from LLM")
		}

		// Merge all choices into one unified view for processing.
		// 
		// Background: Anthropic's API returns separate content blocks (text, tool_use, thinking)
		// which go-langchain converts into separate ContentChoice objects. For example, a response
		// with text + 2 tool calls becomes 3 separate Choices.
		//
		// We merge them here to process the complete response, but later (around line 360) we must
		// split them back into separate AI messages because go-langchain's handleAIMessage() only
		// serializes Parts[0] when sending back to Anthropic. Putting multiple tool calls in one
		// message would lose all but the first.
		//
		// See docs/anthropic-choices-behavior.md for detailed explanation of this pattern.
		mergedChoice := llms.ContentChoice{}
		var allToolCalls []llms.ToolCall
		var contentParts []string

		for i, ch := range resp.Choices {
			debugLog("AgenticClient: processing choice %d: Content=%q, ToolCalls=%d",
				i, truncateString(ch.Content, 100), len(ch.ToolCalls))

			if ch.Content != "" {
				contentParts = append(contentParts, ch.Content)
			}
			if len(ch.ToolCalls) > 0 {
				allToolCalls = append(allToolCalls, ch.ToolCalls...)
			}
			// Use StopReason from first non-empty one
			if mergedChoice.StopReason == "" && ch.StopReason != "" {
				mergedChoice.StopReason = ch.StopReason
			}
			// Use GenerationInfo from first choice
			if i == 0 {
				mergedChoice.GenerationInfo = ch.GenerationInfo
			}
		}

		// Build merged choice — join all content parts so nothing is lost
		// when Anthropic returns multiple text blocks (e.g. thinking + response).
		if len(contentParts) > 0 {
			mergedChoice.Content = strings.Join(contentParts, "\n")
		}
		mergedChoice.ToolCalls = allToolCalls

		choice := mergedChoice
		debugLog("AgenticClient: merged choice - Content=%q, ToolCalls=%d",
			truncateString(choice.Content, 200), len(choice.ToolCalls))
		if choice.Content != "" {
			debugLog("AgenticClient: AI message: %s", truncateString(choice.Content, 200))
		}

		// If no tool calls, check if we should nudge the agent
		if len(choice.ToolCalls) == 0 {
			debugLog("AgenticClient: no tool calls in response")

			consecutiveNoTools++
			debugLog(
				"AgenticClient: consecutive no-tool responses: %d/%d",
				consecutiveNoTools,
				maxConsecutiveNoTools,
			)
			if consecutiveNoTools >= maxConsecutiveNoTools {
				return messages, nil, fmt.Errorf(
					"agent failed to use tools after %d consecutive attempts",
					maxConsecutiveNoTools,
				)
			}

			// Add the AI response and remind to use tools
			if choice.Content != "" {
				messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, choice.Content))
			}
			debugLog("AgenticClient: reminding agent to use tools")
			messages = append(messages, llms.TextParts(
				llms.ChatMessageTypeHuman,
				useToolsReminderPrompt,
			))
			toolCallsRemaining--
			continue
		}

		// Reset consecutive no-tool counter when tools are used
		consecutiveNoTools = 0

		// Validate submit_answer is called alone
		hasSubmitAnswer := false
		for _, toolCall := range choice.ToolCalls {
			if toolCall.FunctionCall.Name == "submit_answer" {
				hasSubmitAnswer = true
				break
			}
		}
		if hasSubmitAnswer && len(choice.ToolCalls) > 1 {
			debugLog("AgenticClient: submit_answer called with other tools - rejecting all")
			// Add a single AI message with ALL tool calls so every
			// tool_result below has a matching tool_use in the preceding
			// assistant message.
			aiParts := make([]llms.ContentPart, len(choice.ToolCalls))
			for i, tc := range choice.ToolCalls {
				aiParts[i] = tc
			}
			aiMessage := llms.MessageContent{
				Role:  llms.ChatMessageTypeAI,
				Parts: aiParts,
			}
			messages = append(messages, aiMessage)
			for _, toolCall := range choice.ToolCalls {
				toolCallsRemaining--
				errorResponse := llms.MessageContent{
					Role: llms.ChatMessageTypeTool,
					Parts: []llms.ContentPart{
						llms.ToolCallResponse{
							ToolCallID: toolCall.ID,
							Name:       toolCall.FunctionCall.Name,
							Content:    submitAnswerAloneError,
						},
					},
				}
				messages = append(messages, errorResponse)
			}
			continue
		}

		// Process each tool call as a separate AI message + tool result pair.
		// This is the "split" part of the merge-then-split pattern.
		//
		// Why: go-langchain's Anthropic handleAIMessage() only serializes Parts[0], so
		// MessageContent{Parts: [toolCall1, toolCall2]} would lose toolCall2 when sent back.
		// By creating one AI message per tool call, we ensure all tool calls are properly
		// serialized. Each tool_use then has its matching tool_result in the following message.
		//
		// See docs/anthropic-choices-behavior.md for details on this serialization constraint.
		for i, toolCall := range choice.ToolCalls {
			toolCallsRemaining--

			aiMessage := llms.MessageContent{
				Role:  llms.ChatMessageTypeAI,
				Parts: []llms.ContentPart{toolCall},
			}
			messages = append(messages, aiMessage)

			response, answer := c.processToolCall(toolCall, i, len(choice.ToolCalls))
			messages = append(messages, response)
			if answer != nil {
				debugLog("AgenticClient: received answer for question %d", questionIndex+1)
				return messages, answer, nil
			}
		}
	}

	// Budget exhausted without answer
	debugLog("AgenticClient: question %d exhausted budget", questionIndex+1)
	return messages, nil, nil
}

// processToolCall processes a single tool call and returns the response message and optional answer
func (c *agenticClientImpl) processToolCall(
	toolCall llms.ToolCall,
	index, total int,
) (llms.MessageContent, *AnswerSchema) {
	debugLog(
		"AgenticClient: [%d/%d] executing tool: %s",
		index+1,
		total,
		toolCall.FunctionCall.Name,
	)
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
						Content: fmt.Sprintf(
							"Error parsing answer: %v. Please try again with valid JSON.",
							err,
						),
					},
				},
			}, nil
		}
		debugLog("AgenticClient: received answer: short_answer=%v, answer=%s",
			answer.ShortAnswer, truncateString(answer.Answer, 100))

		// Return success response and the answer
		return llms.MessageContent{
			Role: llms.ChatMessageTypeTool,
			Parts: []llms.ContentPart{
				llms.ToolCallResponse{
					ToolCallID: toolCall.ID,
					Name:       toolCall.FunctionCall.Name,
					Content:    "Answer recorded successfully.",
				},
			},
		}, &answer
	}

	// Execute other tools
	result, err := c.executor.execute(toolCall.FunctionCall.Name, toolCall.FunctionCall.Arguments)
	if err != nil {
		result = fmt.Sprintf("Error: %v", err)
	}
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
func (c *agenticClientImpl) callLLMWithRetry(
	ctx context.Context,
	llm llms.Model,
	messages []llms.MessageContent,
) (*llms.ContentResponse, error) {
	var lastErr error
	for attempt := 1; attempt <= maxLLMRetries; attempt++ {
		resp, err := llm.GenerateContent(ctx, messages, llms.WithTools(c.tools))
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
		return nil, fmt.Errorf(
			"unsupported provider: %s (supported: google, anthropic, openai)",
			opts.Provider,
		)
	}
}
