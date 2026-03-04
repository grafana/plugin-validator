package llmprovider

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// LangchainAdapter wraps an llms.Model to implement the Provider interface.
// This allows existing langchain-based providers (Anthropic, OpenAI) to be
// used alongside native providers during the migration.
type LangchainAdapter struct {
	LLM llms.Model
}

func NewLangchainAdapter(llm llms.Model) *LangchainAdapter {
	return &LangchainAdapter{LLM: llm}
}

func (a *LangchainAdapter) GenerateContent(
	ctx context.Context,
	messages []Message,
	options ...CallOption,
) (*Response, error) {
	// Convert our messages to langchain messages
	lcMessages := MessagesToLangchain(messages)

	// Convert our options to langchain options
	opts := &CallOptions{}
	for _, o := range options {
		o(opts)
	}
	lcOpts := toLangchainCallOptions(opts)

	// Call the langchain model
	lcResp, err := a.LLM.GenerateContent(ctx, lcMessages, lcOpts...)
	if err != nil {
		return nil, err
	}

	// Convert response back to our types
	return ResponseFromLangchain(lcResp), nil
}

// toLangchainCallOptions converts our CallOptions to langchain CallOptions.
func toLangchainCallOptions(opts *CallOptions) []llms.CallOption {
	var lcOpts []llms.CallOption

	if opts.MaxTokens > 0 {
		lcOpts = append(lcOpts, llms.WithMaxTokens(opts.MaxTokens))
	}
	if opts.Temperature > 0 {
		lcOpts = append(lcOpts, llms.WithTemperature(opts.Temperature))
	}
	if opts.TopP > 0 {
		lcOpts = append(lcOpts, llms.WithTopP(opts.TopP))
	}
	if opts.TopK > 0 {
		lcOpts = append(lcOpts, llms.WithTopK(opts.TopK))
	}
	if len(opts.StopWords) > 0 {
		lcOpts = append(lcOpts, llms.WithStopWords(opts.StopWords))
	}
	if len(opts.Tools) > 0 {
		lcOpts = append(lcOpts, llms.WithTools(ToolsToLangchain(opts.Tools)))
	}

	return lcOpts
}
