// Package anthropicprovider implements the llmprovider.Provider interface
// using the official Anthropic Go SDK (github.com/anthropics/anthropic-sdk-go).
package anthropicprovider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/grafana/plugin-validator/pkg/llmprovider"
	"github.com/grafana/plugin-validator/pkg/logme"
)

// Client implements llmprovider.Provider for Anthropic.
type Client struct {
	client    *anthropic.Client
	modelName string
}

// New creates an Anthropic provider client.
func New(apiKey, modelName string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic: API key is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("anthropic: model name is required")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Client{client: &client, modelName: modelName}, nil
}

// GenerateContent sends messages to Anthropic and returns the response.
func (c *Client) GenerateContent(
	ctx context.Context,
	messages []llmprovider.Message,
	options ...llmprovider.CallOption,
) (*llmprovider.Response, error) {
	opts := &llmprovider.CallOptions{}
	for _, o := range options {
		o(opts)
	}

	system, msgs := extractSystemAndMessages(messages)

	maxTokens := int64(4096)
	if opts.MaxTokens > 0 {
		maxTokens = int64(opts.MaxTokens)
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(c.modelName),
		MaxTokens: maxTokens,
		Messages:  msgs,
	}

	if len(system) > 0 {
		params.System = system
	}

	if opts.Temperature > 0 {
		params.Temperature = anthropic.Float(opts.Temperature)
	}
	if opts.TopP > 0 {
		params.TopP = anthropic.Float(opts.TopP)
	}
	if len(opts.StopWords) > 0 {
		params.StopSequences = opts.StopWords
	}

	if len(opts.Tools) > 0 {
		params.Tools = toAnthropicTools(opts.Tools)
	}

	resp, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic: API error: %w", err)
	}

	return fromAnthropicResponse(resp), nil
}

// --- Message conversion: llmprovider → anthropic ---

// extractSystemAndMessages separates system messages (which go in a top-level
// param) from conversation messages.
func extractSystemAndMessages(messages []llmprovider.Message) ([]anthropic.TextBlockParam, []anthropic.MessageParam) {
	var system []anthropic.TextBlockParam
	var result []anthropic.MessageParam

	for _, msg := range messages {
		switch msg.Role {
		case llmprovider.RoleSystem:
			text := extractText(msg.Parts)
			if text != "" {
				system = append(system, anthropic.TextBlockParam{Text: text})
			}

		case llmprovider.RoleHuman:
			blocks := toUserBlocks(msg.Parts)
			if len(blocks) > 0 {
				result = append(result, anthropic.NewUserMessage(blocks...))
			}

		case llmprovider.RoleAI:
			blocks := toAssistantBlocks(msg.Parts)
			if len(blocks) > 0 {
				result = append(result, anthropic.NewAssistantMessage(blocks...))
			}

		case llmprovider.RoleTool:
			// Anthropic sends tool results as user messages
			blocks := toToolResultBlocks(msg.Parts)
			if len(blocks) > 0 {
				result = append(result, anthropic.NewUserMessage(blocks...))
			}
		}
	}

	return system, result
}

func toUserBlocks(parts []llmprovider.Part) []anthropic.ContentBlockParamUnion {
	var blocks []anthropic.ContentBlockParamUnion
	for _, p := range parts {
		switch v := p.(type) {
		case llmprovider.TextPart:
			blocks = append(blocks, anthropic.NewTextBlock(v.Text))
		case llmprovider.ToolResultPart:
			blocks = append(blocks, anthropic.NewToolResultBlock(v.ToolCallID, v.Content, false))
		}
	}
	return blocks
}

func toAssistantBlocks(parts []llmprovider.Part) []anthropic.ContentBlockParamUnion {
	var blocks []anthropic.ContentBlockParamUnion
	for _, p := range parts {
		switch v := p.(type) {
		case llmprovider.TextPart:
			blocks = append(blocks, anthropic.NewTextBlock(v.Text))
		case llmprovider.ToolCallPart:
			// Parse the arguments string back to any for the input field
			var input any
			if err := json.Unmarshal([]byte(v.Arguments), &input); err != nil {
				input = map[string]any{}
			}
			blocks = append(blocks, anthropic.NewToolUseBlock(v.ID, input, v.Name))
		case llmprovider.ThinkingPart:
			if v.Encrypted != "" {
				blocks = append(blocks, anthropic.NewRedactedThinkingBlock(v.Encrypted))
			} else if v.Text != "" {
				blocks = append(blocks, anthropic.NewThinkingBlock(v.Signature, v.Text))
			}
		}
	}
	return blocks
}

func toToolResultBlocks(parts []llmprovider.Part) []anthropic.ContentBlockParamUnion {
	var blocks []anthropic.ContentBlockParamUnion
	for _, p := range parts {
		if tr, ok := p.(llmprovider.ToolResultPart); ok {
			blocks = append(blocks, anthropic.NewToolResultBlock(tr.ToolCallID, tr.Content, false))
		}
	}
	return blocks
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

func toAnthropicTools(tools []llmprovider.Tool) []anthropic.ToolUnionParam {
	var result []anthropic.ToolUnionParam
	for _, tool := range tools {
		if tool.Function == nil {
			continue
		}

		param := anthropic.ToolParam{
			Name:        tool.Function.Name,
			Description: anthropic.String(tool.Function.Description),
		}

		// Convert parameters to ToolInputSchemaParam
		if tool.Function.Parameters != nil {
			schema := toInputSchema(tool.Function.Parameters)
			param.InputSchema = schema
		}

		result = append(result, anthropic.ToolUnionParam{OfTool: &param})
	}
	return result
}

func toInputSchema(params any) anthropic.ToolInputSchemaParam {
	schema := anthropic.ToolInputSchemaParam{}

	var m map[string]any
	switch p := params.(type) {
	case map[string]any:
		m = p
	default:
		data, err := json.Marshal(p)
		if err != nil {
			return schema
		}
		if err := json.Unmarshal(data, &m); err != nil {
			return schema
		}
	}

	if props, ok := m["properties"]; ok {
		schema.Properties = props
	}
	if req, ok := m["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				schema.Required = append(schema.Required, s)
			}
		}
	}

	return schema
}

// --- Response conversion: anthropic → llmprovider ---

func fromAnthropicResponse(resp *anthropic.Message) *llmprovider.Response {
	choice := &llmprovider.Choice{
		StopReason:     string(resp.StopReason),
		GenerationInfo: make(map[string]any),
	}

	for _, block := range resp.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			if choice.Content != "" {
				choice.Content += "\n"
			}
			choice.Content += v.Text

		case anthropic.ThinkingBlock:
			logme.LLMLog("anthropic: thinking block (signature=%s, len=%d)", v.Signature[:min(20, len(v.Signature))], len(v.Thinking))
			choice.Thinking = append(choice.Thinking, llmprovider.ThinkingPart{
				Text:      v.Thinking,
				Signature: v.Signature,
			})

		case anthropic.RedactedThinkingBlock:
			logme.LLMLog("anthropic: redacted thinking block (data_len=%d)", len(v.Data))
			choice.Thinking = append(choice.Thinking, llmprovider.ThinkingPart{
				Encrypted: v.Data,
			})

		case anthropic.ToolUseBlock:
			args := string(v.Input)
			logme.LLMLog("anthropic: tool call: name=%s id=%s", v.Name, v.ID)
			choice.ToolCalls = append(choice.ToolCalls, llmprovider.ToolCallPart{
				ID:        v.ID,
				Name:      v.Name,
				Arguments: args,
			})
		}
	}

	choice.GenerationInfo["usage"] = map[string]any{
		"input_tokens":  resp.Usage.InputTokens,
		"output_tokens": resp.Usage.OutputTokens,
	}

	return &llmprovider.Response{
		Choices: []*llmprovider.Choice{choice},
		Usage: llmprovider.Usage{
			InputTokens:  int(resp.Usage.InputTokens),
			OutputTokens: int(resp.Usage.OutputTokens),
			TotalTokens:  int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
