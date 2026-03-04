// Package gemini implements the llmprovider.Provider interface using the
// Google GenAI SDK (google.golang.org/genai).  It properly preserves
// thought_signatures for Gemini 3.x models.
package gemini

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/grafana/plugin-validator/pkg/llmprovider"
	"github.com/grafana/plugin-validator/pkg/logme"
	"google.golang.org/genai"
)

// Client implements llmprovider.Provider for Gemini via AI Studio.
type Client struct {
	client    *genai.Client
	modelName string
}

// New creates a Gemini provider client using an AI Studio API key.
func New(ctx context.Context, apiKey, modelName string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("gemini: API key is required")
	}
	if modelName == "" {
		return nil, fmt.Errorf("gemini: model name is required")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to create client: %w", err)
	}

	return &Client{client: client, modelName: modelName}, nil
}

// GenerateContent sends messages to Gemini and returns the response.
// It preserves thought_signatures for Gemini 3.x compatibility.
func (c *Client) GenerateContent(
	ctx context.Context,
	messages []llmprovider.Message,
	options ...llmprovider.CallOption,
) (*llmprovider.Response, error) {
	opts := &llmprovider.CallOptions{}
	for _, o := range options {
		o(opts)
	}

	// Extract system instruction from messages (Gemini handles it separately)
	systemInstruction, conversationMessages := extractSystemMessage(messages)

	// Convert our messages to genai.Content
	contents, err := toGenAIContents(conversationMessages)
	if err != nil {
		return nil, fmt.Errorf("gemini: failed to convert messages: %w", err)
	}

	// Build config
	config := buildConfig(opts, systemInstruction)

	// Call Gemini API
	resp, err := c.client.Models.GenerateContent(ctx, c.modelName, contents, config)
	if err != nil {
		return nil, fmt.Errorf("gemini: API error: %w", err)
	}

	// Convert response, preserving thought_signatures
	return fromGenAIResponse(resp)
}

// extractSystemMessage pulls the system instruction out of the message list.
// Gemini takes system instructions via config, not as a message role.
func extractSystemMessage(messages []llmprovider.Message) (string, []llmprovider.Message) {
	var system string
	var rest []llmprovider.Message

	for _, m := range messages {
		if m.Role == llmprovider.RoleSystem {
			// Concatenate all text parts from system messages
			for _, p := range m.Parts {
				if tp, ok := p.(llmprovider.TextPart); ok {
					if system != "" {
						system += "\n"
					}
					system += tp.Text
				}
			}
		} else {
			rest = append(rest, m)
		}
	}

	return system, rest
}

// buildConfig creates the GenAI generation config from our options.
func buildConfig(opts *llmprovider.CallOptions, systemInstruction string) *genai.GenerateContentConfig {
	config := &genai.GenerateContentConfig{}

	if systemInstruction != "" {
		config.SystemInstruction = &genai.Content{
			Parts: []*genai.Part{genai.NewPartFromText(systemInstruction)},
		}
	}

	if opts.Temperature > 0 {
		t := float32(opts.Temperature)
		config.Temperature = &t
	}
	if opts.MaxTokens > 0 {
		config.MaxOutputTokens = int32(opts.MaxTokens)
	}
	if opts.TopP > 0 {
		p := float32(opts.TopP)
		config.TopP = &p
	}
	if opts.TopK > 0 {
		k := float32(opts.TopK)
		config.TopK = &k
	}
	if len(opts.StopWords) > 0 {
		config.StopSequences = opts.StopWords
	}

	// Convert tools
	if len(opts.Tools) > 0 {
		var declarations []*genai.FunctionDeclaration
		for _, tool := range opts.Tools {
			if tool.Function != nil {
				decl := &genai.FunctionDeclaration{
					Name:        tool.Function.Name,
					Description: tool.Function.Description,
				}
				// Use ParametersJsonSchema for raw JSON schema passthrough
				if tool.Function.Parameters != nil {
					decl.ParametersJsonSchema = tool.Function.Parameters
				}
				declarations = append(declarations, decl)
			}
		}
		if len(declarations) > 0 {
			config.Tools = []*genai.Tool{{
				FunctionDeclarations: declarations,
			}}
			config.ToolConfig = &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{
					Mode: genai.FunctionCallingConfigModeAuto,
				},
			}
		}
	}

	return config
}

// --- Message conversion: llmprovider → genai ---

func toGenAIContents(messages []llmprovider.Message) ([]*genai.Content, error) {
	var contents []*genai.Content

	for _, msg := range messages {
		content := &genai.Content{
			Role: toGenAIRole(msg.Role),
		}

		for _, part := range msg.Parts {
			genaiPart, err := toGenAIPart(part)
			if err != nil {
				return nil, err
			}
			if genaiPart != nil {
				content.Parts = append(content.Parts, genaiPart)
			}
		}

		if len(content.Parts) > 0 {
			contents = append(contents, content)
		}
	}

	return contents, nil
}

func toGenAIRole(role llmprovider.Role) string {
	switch role {
	case llmprovider.RoleHuman:
		return "user"
	case llmprovider.RoleAI:
		return "model"
	case llmprovider.RoleTool:
		return "user"
	default:
		return "user"
	}
}

func toGenAIPart(part llmprovider.Part) (*genai.Part, error) {
	switch p := part.(type) {
	case llmprovider.TextPart:
		return genai.NewPartFromText(p.Text), nil

	case llmprovider.ToolCallPart:
		// Parse arguments from JSON string to map
		var args map[string]any
		if p.Arguments != "" {
			if err := json.Unmarshal([]byte(p.Arguments), &args); err != nil {
				return nil, fmt.Errorf("gemini: failed to unmarshal tool arguments for %q: %w", p.Name, err)
			}
		}

		genaiPart := genai.NewPartFromFunctionCall(p.Name, args)
		if p.ID != "" {
			genaiPart.FunctionCall.ID = p.ID
		}

		// Echo back thought fields exactly as received from the API.
		genaiPart.Thought = p.Thought
		if p.ThoughtSignature != "" {
			genaiPart.ThoughtSignature = []byte(p.ThoughtSignature)
		}

		return genaiPart, nil

	case llmprovider.ToolResultPart:
		// Convert response content to map
		var responseMap map[string]any
		if err := json.Unmarshal([]byte(p.Content), &responseMap); err != nil {
			// If it's not JSON, wrap it
			responseMap = map[string]any{"result": p.Content}
		}

		genaiPart := genai.NewPartFromFunctionResponse(p.Name, responseMap)
		if p.ToolCallID != "" {
			genaiPart.FunctionResponse.ID = p.ToolCallID
		}

		return genaiPart, nil

	case llmprovider.ThinkingPart:
		// Thinking parts from previous responses need to be echoed back
		genaiPart := &genai.Part{
			Text:    p.Text,
			Thought: true,
		}
		if p.Signature != "" {
			genaiPart.ThoughtSignature = []byte(p.Signature)
		}
		return genaiPart, nil

	default:
		return nil, fmt.Errorf("gemini: unsupported part type %T", part)
	}
}

// --- Response conversion: genai → llmprovider ---

func fromGenAIResponse(resp *genai.GenerateContentResponse) (*llmprovider.Response, error) {
	if resp == nil {
		return nil, fmt.Errorf("gemini: nil response")
	}

	result := &llmprovider.Response{
		Choices: make([]*llmprovider.Choice, 0, len(resp.Candidates)),
	}

	for candidateIdx, candidate := range resp.Candidates {
		choice := &llmprovider.Choice{
			StopReason:     string(candidate.FinishReason),
			GenerationInfo: make(map[string]any),
		}

		if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
			raw, _ := json.MarshalIndent(candidate, "", "  ")
			logme.LLMLog("gemini: candidate[%d] empty/nil content, raw candidate:\n%s", candidateIdx, string(raw))
		}

		if candidate.Content != nil {
			for partIdx, part := range candidate.Content.Parts {
				if part == nil {
					continue
				}

				// Debug: log raw part fields so we can see exactly what the SDK returns
				debugLogPart(partIdx, part)

				// Thought/thinking parts
				if part.Thought && part.FunctionCall == nil {
					thinking := llmprovider.ThinkingPart{
						Text: part.Text,
					}
					if len(part.ThoughtSignature) > 0 {
						thinking.Signature = string(part.ThoughtSignature)
					}
					choice.Thinking = append(choice.Thinking, thinking)
					continue
				}

				// Text content (non-thought)
				if part.Text != "" && part.FunctionCall == nil && part.FunctionResponse == nil {
					choice.Content = part.Text
				}

				// Function calls
				if part.FunctionCall != nil {
					id := part.FunctionCall.ID
					if id == "" {
						id = generateCallID()
						logme.LLMLog("gemini: part[%d] FunctionCall has empty ID, generated: %s", partIdx, id)
					}
					tc := llmprovider.ToolCallPart{
						ID:      id,
						Name:    part.FunctionCall.Name,
						Thought: part.Thought,
					}

					if part.FunctionCall.Args != nil {
						argsJSON, err := json.Marshal(part.FunctionCall.Args)
						if err != nil {
							return nil, fmt.Errorf("gemini: failed to marshal function args: %w", err)
						}
						tc.Arguments = string(argsJSON)
					}

					// CRITICAL: Capture thought_signature from function call parts
					if len(part.ThoughtSignature) > 0 {
						tc.ThoughtSignature = string(part.ThoughtSignature)
					}

					choice.ToolCalls = append(choice.ToolCalls, tc)
				}
			}
		}

		// Token usage
		if resp.UsageMetadata != nil {
			choice.GenerationInfo["usage"] = map[string]any{
				"prompt_tokens":     resp.UsageMetadata.PromptTokenCount,
				"completion_tokens": resp.UsageMetadata.CandidatesTokenCount,
				"total_tokens":      resp.UsageMetadata.TotalTokenCount,
				"thoughts_tokens":   resp.UsageMetadata.ThoughtsTokenCount,
			}
		}

		result.Choices = append(result.Choices, choice)
	}

	return result, nil
}

func generateCallID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return "call_" + hex.EncodeToString(b)
}

func debugLogPart(idx int, p *genai.Part) {
	hasFuncCall := p.FunctionCall != nil
	hasFuncResp := p.FunctionResponse != nil
	textLen := len(p.Text)
	sigLen := len(p.ThoughtSignature)
	logme.LLMLog("gemini: part[%d] Thought=%v Text=%d bytes FuncCall=%v FuncResp=%v ThoughtSig=%d bytes",
		idx, p.Thought, textLen, hasFuncCall, hasFuncResp, sigLen)
	if p.Thought && textLen > 0 {
		preview := p.Text
		if len(preview) > 200 {
			preview = preview[:200] + "..."
		}
		logme.LLMLog("gemini: part[%d] thinking preview: %s", idx, preview)
	}
	if hasFuncCall {
		logme.LLMLog("gemini: part[%d] FunctionCall: name=%s id=%s", idx, p.FunctionCall.Name, p.FunctionCall.ID)
	}
}
