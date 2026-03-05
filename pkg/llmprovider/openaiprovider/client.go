// Package openai implements the llmprovider.Provider interface using the
// official OpenAI Go SDK (github.com/openai/openai-go).
package openaiprovider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grafana/plugin-validator/pkg/llmprovider"
	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

// Client implements llmprovider.Provider for OpenAI.
type Client struct {
	client    *openai.Client
	modelName string
}

// New creates an OpenAI provider client.
func New(apiKey, modelName string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("openai: API key is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("openai: model name is required")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	return &Client{client: &client, modelName: modelName}, nil
}

// GenerateContent sends messages to OpenAI and returns the response.
func (c *Client) GenerateContent(
	ctx context.Context,
	messages []llmprovider.Message,
	options ...llmprovider.CallOption,
) (*llmprovider.Response, error) {
	opts := &llmprovider.CallOptions{}
	for _, o := range options {
		o(opts)
	}

	params := openai.ChatCompletionNewParams{
		Model:    shared.ChatModel(c.modelName),
		Messages: toOpenAIMessages(messages),
	}

	if opts.Temperature > 0 {
		params.Temperature = openai.Float(opts.Temperature)
	}
	if opts.MaxTokens > 0 {
		params.MaxCompletionTokens = openai.Int(int64(opts.MaxTokens))
	}
	if opts.TopP > 0 {
		params.TopP = openai.Float(opts.TopP)
	}
	if len(opts.StopWords) > 0 {
		params.Stop = openai.ChatCompletionNewParamsStopUnion{
			OfStringArray: opts.StopWords,
		}
	}

	if len(opts.Tools) > 0 {
		params.Tools = toOpenAITools(opts.Tools)
	}

	resp, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai: API error: %w", err)
	}

	return fromOpenAIResponse(resp), nil
}

// --- Message conversion: llmprovider → openai ---

func toOpenAIMessages(messages []llmprovider.Message) []openai.ChatCompletionMessageParamUnion {
	var result []openai.ChatCompletionMessageParamUnion

	for _, msg := range messages {
		switch msg.Role {
		case llmprovider.RoleSystem:
			text := extractText(msg.Parts)
			result = append(result, openai.SystemMessage(text))

		case llmprovider.RoleHuman:
			text := extractText(msg.Parts)
			result = append(result, openai.UserMessage(text))

		case llmprovider.RoleAI:
			result = append(result, toAssistantMessage(msg))

		case llmprovider.RoleTool:
			for _, part := range msg.Parts {
				if tr, ok := part.(llmprovider.ToolResultPart); ok {
					result = append(result, openai.ToolMessage(tr.Content, tr.ToolCallID))
				}
			}
		}
	}

	return result
}

func toAssistantMessage(msg llmprovider.Message) openai.ChatCompletionMessageParamUnion {
	text := extractText(msg.Parts)

	var toolCalls []openai.ChatCompletionMessageToolCallParam
	for _, part := range msg.Parts {
		if tc, ok := part.(llmprovider.ToolCallPart); ok {
			toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
				ID: tc.ID,
				Function: openai.ChatCompletionMessageToolCallFunctionParam{
					Name:      tc.Name,
					Arguments: tc.Arguments,
				},
			})
		}
	}

	asst := openai.ChatCompletionAssistantMessageParam{}
	if text != "" {
		asst.Content.OfString = openai.String(text)
	}
	if len(toolCalls) > 0 {
		asst.ToolCalls = toolCalls
	}

	return openai.ChatCompletionMessageParamUnion{OfAssistant: &asst}
}

func extractText(parts []llmprovider.Part) string {
	var text string
	for _, p := range parts {
		if tp, ok := p.(llmprovider.TextPart); ok {
			if text != "" {
				text += "\n"
			}
			text += tp.Text
		}
	}
	return text
}

// --- Tool conversion ---

func toOpenAITools(tools []llmprovider.Tool) []openai.ChatCompletionToolParam {
	var result []openai.ChatCompletionToolParam
	for _, tool := range tools {
		if tool.Function == nil {
			continue
		}

		param := openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        tool.Function.Name,
				Description: openai.String(tool.Function.Description),
			},
		}

		// Convert parameters to FunctionParameters (map[string]any)
		if tool.Function.Parameters != nil {
			switch p := tool.Function.Parameters.(type) {
			case map[string]any:
				param.Function.Parameters = shared.FunctionParameters(p)
			default:
				// Marshal and unmarshal to get map[string]any
				data, err := json.Marshal(p)
				if err == nil {
					var m map[string]any
					if json.Unmarshal(data, &m) == nil {
						param.Function.Parameters = shared.FunctionParameters(m)
					}
				}
			}
		}

		result = append(result, param)
	}
	return result
}

// --- Response conversion: openai → llmprovider ---

func fromOpenAIResponse(resp *openai.ChatCompletion) *llmprovider.Response {
	result := &llmprovider.Response{
		Choices: make([]*llmprovider.Choice, 0, len(resp.Choices)),
	}

	for _, c := range resp.Choices {
		choice := &llmprovider.Choice{
			Content:        c.Message.Content,
			StopReason:     c.FinishReason,
			GenerationInfo: make(map[string]any),
		}

		for _, tc := range c.Message.ToolCalls {
			logme.LLMLog("openai: tool call: name=%s id=%s", tc.Function.Name, tc.ID)
			choice.ToolCalls = append(choice.ToolCalls, llmprovider.ToolCallPart{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		choice.GenerationInfo["usage"] = map[string]any{
			"prompt_tokens":     resp.Usage.PromptTokens,
			"completion_tokens": resp.Usage.CompletionTokens,
			"total_tokens":      resp.Usage.TotalTokens,
		}

		result.Choices = append(result.Choices, choice)
	}

	return result
}
