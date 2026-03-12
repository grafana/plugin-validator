package llmconfig

import "os"

// ProviderConfig holds the resolved LLM provider, model, and API key.
type ProviderConfig struct {
	Provider string
	Model    string
	APIKey   string
}

// providerDefaults maps each provider to its default model and API key env var.
var providerDefaults = []struct {
	EnvVar       string
	Provider     string
	DefaultModel string
}{
	{"ANTHROPIC_API_KEY", "anthropic", "claude-opus-4-6"},
	{"OPENAI_API_KEY", "openai", "gpt-5.4"},
	{"GEMINI_API_KEY", "google", "gemini-3.1-flash-lite-preview"},
}

// Resolve returns the LLM provider configuration by checking environment
// variables in order: ANTHROPIC_API_KEY, OPENAI_API_KEY, GEMINI_API_KEY.
// The first key found wins. Returns nil if no API key is set.
func Resolve() *ProviderConfig {
	for _, p := range providerDefaults {
		if key := os.Getenv(p.EnvVar); key != "" {
			return &ProviderConfig{
				Provider: p.Provider,
				Model:    p.DefaultModel,
				APIKey:   key,
			}
		}
	}
	return nil
}
