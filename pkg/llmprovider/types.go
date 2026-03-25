// Package llmprovider defines a unified interface for LLM providers.
//
// Each provider (Gemini, Anthropic, OpenAI) has a native SDK implementation
// that supports provider-specific features like Gemini thought_signatures,
// Anthropic thinking blocks, and OpenAI encrypted reasoning content.
package llmprovider

import "context"

// Provider is the core interface that all LLM provider clients implement.
type Provider interface {
	GenerateContent(ctx context.Context, messages []Message, options ...CallOption) (*Response, error)
}

// Role identifies the sender of a message.
type Role string

const (
	RoleSystem Role = "system"
	RoleHuman  Role = "human"
	RoleAI     Role = "ai"
	RoleTool   Role = "tool"
)

// Message is a single message in a conversation.
type Message struct {
	Role  Role
	Parts []Part
}

// Part is a piece of content within a message.
// Concrete types: TextPart, ToolCallPart, ToolResultPart, ThinkingPart.
type Part interface {
	partMarker()
}

// TextPart is plain text content.
type TextPart struct {
	Text string
}

func (TextPart) partMarker() {}

// ToolCallPart represents a model's request to call a tool.
type ToolCallPart struct {
	ID        string
	Name      string
	Arguments string // JSON string

	// Thought indicates whether this part was produced during model thinking.
	// Must be echoed back exactly as received from the API.
	Thought bool

	// ThoughtSignature is the opaque token Gemini 3.x attaches to function
	// call parts.  It must be echoed back in subsequent requests or the API
	// returns a 400.  Nil/empty means no signature was provided.
	ThoughtSignature string
}

func (ToolCallPart) partMarker() {}

// ToolResultPart is the response from executing a tool.
type ToolResultPart struct {
	ToolCallID string
	Name       string
	Content    string
}

func (ToolResultPart) partMarker() {}

// ThinkingPart holds reasoning/thinking content from the model.
// Different providers represent this differently:
//   - Gemini: thought text + thought_signature
//   - Anthropic: thinking block with signature, or redacted_thinking
//   - OpenAI: encrypted reasoning content
type ThinkingPart struct {
	Text      string
	Signature string // Gemini thought_signature or Anthropic thinking signature
	Encrypted string // OpenAI encrypted_content or Anthropic redacted_thinking data
}

func (ThinkingPart) partMarker() {}

// Usage tracks token usage metrics.
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int

	// CacheCreationInputTokens is the number of input tokens used to create
	// a cache entry. Only populated by providers that support prompt caching
	// (e.g. Anthropic). Zero for other providers.
	CacheCreationInputTokens int
	// CacheReadInputTokens is the number of input tokens read from cache.
	// Only populated by providers that support prompt caching. Zero for others.
	CacheReadInputTokens int
}

// Response is the result of a GenerateContent call.
type Response struct {
	Choices []*Choice
	Usage   Usage
}

// Choice is a single response candidate.
type Choice struct {
	// Content is the text content of the response.
	Content string

	// StopReason is why the model stopped generating.
	StopReason string

	// ToolCalls requested by the model. These preserve ThoughtSignature
	// so they can be echoed back in the next request.
	ToolCalls []ToolCallPart

	// Thinking contains reasoning/thinking content if the model produced any.
	Thinking []ThinkingPart

	// GenerationInfo holds arbitrary provider-specific metadata (token
	// counts, safety ratings, etc.).
	GenerationInfo map[string]any
}

// --- Call options ---

// CallOption configures a GenerateContent call.
type CallOption func(*CallOptions)

// CallOptions holds all configurable parameters for a GenerateContent call.
type CallOptions struct {
	MaxTokens   int
	Temperature float64
	TopP        float64
	TopK        int
	StopWords   []string
	Tools       []Tool
}

// Tool describes a tool the model can invoke.
type Tool struct {
	Type     string
	Function *FunctionDef
}

// FunctionDef describes a callable function.
type FunctionDef struct {
	Name        string
	Description string
	Parameters  any // JSON Schema
}

// --- Option helpers ---

func WithMaxTokens(n int) CallOption {
	return func(o *CallOptions) { o.MaxTokens = n }
}

func WithTemperature(t float64) CallOption {
	return func(o *CallOptions) { o.Temperature = t }
}

func WithTopP(p float64) CallOption {
	return func(o *CallOptions) { o.TopP = p }
}

func WithTopK(k int) CallOption {
	return func(o *CallOptions) { o.TopK = k }
}

func WithStopWords(words []string) CallOption {
	return func(o *CallOptions) { o.StopWords = words }
}

func WithTools(tools []Tool) CallOption {
	return func(o *CallOptions) { o.Tools = tools }
}

// --- Convenience constructors ---

// TextMessage creates a Message with a single text part.
func TextMessage(role Role, text string) Message {
	return Message{Role: role, Parts: []Part{TextPart{Text: text}}}
}
