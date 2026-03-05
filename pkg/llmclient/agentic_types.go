package llmclient

import (
	"fmt"
	"strings"

	"github.com/grafana/plugin-validator/pkg/llmprovider"
)

const defaultSystemPromptIntro = `You are a code analysis assistant. You have tools to explore code in a repository.

STRATEGY:
1. Use list_directory to see what files exist
2. Use read_file to read the source code files
3. Analyze the code to answer the question`

// AgenticTool identifies an exploration tool available to the agent.
type AgenticTool string

const (
	ToolReadFile      AgenticTool = "read_file"
	ToolListDirectory AgenticTool = "list_directory"
	ToolGrep          AgenticTool = "grep"
	ToolGit           AgenticTool = "git"
)

// ToolSet is a preset collection of tools.
type ToolSet int

const (
	// DefaultTooling includes all exploration tools (read_file, list_directory,
	// grep, git) plus submit_answer. This is the zero value.
	DefaultTooling ToolSet = iota
	// NoTools includes only submit_answer with no exploration tools.
	NoTools
)

// AgenticCallOptions contains configuration for the agentic LLM call.
type AgenticCallOptions struct {
	Model    string // e.g. "gemini-2.5-flash"
	Provider string // "google", "anthropic", "openai"
	APIKey   string

	// Tools selects specific exploration tools. When non-nil, takes precedence
	// over ToolSet. submit_answer is always included regardless.
	Tools []AgenticTool

	// ToolSet selects a preset collection of tools. Used when Tools is nil.
	// The zero value (DefaultTooling) includes all exploration tools.
	ToolSet ToolSet

	// SystemPrompt overrides the intro portion of the system prompt. The
	// AVAILABLE TOOLS section is always auto-appended. When empty, a default
	// intro is used.
	SystemPrompt string
}

// AnswerSchema represents the structured response from the agentic client.
type AnswerSchema struct {
	Question    string   `json:"question"`
	Answer      string   `json:"answer"`
	ShortAnswer bool     `json:"short_answer"`
	Files       []string `json:"files,omitempty"`
	CodeSnippet string   `json:"code_snippet,omitempty"`
}

// defaultTools returns the full set of exploration tools.
func defaultTools() []AgenticTool {
	return []AgenticTool{ToolReadFile, ToolListDirectory, ToolGrep, ToolGit}
}

// resolveTools builds the final []llmprovider.Tool list from the options.
// submit_answer is always appended.
func resolveTools(opts *AgenticCallOptions) ([]llmprovider.Tool, error) {
	var selected []AgenticTool
	if opts.Tools != nil {
		selected = opts.Tools
	} else {
		switch opts.ToolSet {
		case DefaultTooling:
			selected = defaultTools()
		case NoTools:
			// empty
		default:
			return nil, fmt.Errorf("unknown tool set: %d", opts.ToolSet)
		}
	}

	tools := make([]llmprovider.Tool, 0, len(selected)+1)
	for _, name := range selected {
		def, ok := toolRegistry[name]
		if !ok {
			return nil, fmt.Errorf("unknown tool: %q", name)
		}
		tools = append(tools, def)
	}
	tools = append(tools, submitAnswerTool())
	return tools, nil
}

// buildSystemPrompt composes the system prompt from an intro and the resolved tools.
func buildSystemPrompt(intro string, tools []llmprovider.Tool) string {
	if intro == "" {
		intro = defaultSystemPromptIntro
	}

	var b strings.Builder
	b.WriteString(intro)
	b.WriteString("\n\nAVAILABLE TOOLS:\n")
	for _, t := range tools {
		if t.Function != nil {
			fmt.Fprintf(&b, "- %s: %s\n", t.Function.Name, t.Function.Description)
		}
	}
	b.WriteString("\nIMPORTANT: You are in non-interactive mode. No one will read your text answers, only tools.\nWhen you have gathered enough information, use submit_answer to provide your answer.")

	return b.String()
}
