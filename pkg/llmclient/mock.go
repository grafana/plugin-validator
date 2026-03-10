package llmclient

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/logme"
)

type MockLLMClient struct {
	responses []MockResponse
}

type MockResponse struct {
	Question     string   `json:"question"`
	Answer       string   `json:"answer"`
	RelatedFiles []string `json:"related_files"`
	CodeSnippet  string   `json:"code_snippet"`
	ShortAnswer  string   `json:"short_answer"`
}

func NewMockLLMClient() *MockLLMClient {
	return &MockLLMClient{
		responses: []MockResponse{
			{
				Question:     "Are there any security vulnerabilities in the code changes?",
				Answer:       "No security vulnerabilities were found in the code changes.",
				RelatedFiles: []string{"src/ClockPanel.tsx"},
				CodeSnippet:  "// Mock code snippet",
				ShortAnswer:  "no",
			},
			{
				Question:     "Are there any performance issues in the code changes?",
				Answer:       "No performance issues were identified in the code changes.",
				RelatedFiles: []string{"src/migrations.ts"},
				CodeSnippet:  "// Mock migration code",
				ShortAnswer:  "no",
			},
		},
	}
}

func (m *MockLLMClient) WithResponses(responses []MockResponse) *MockLLMClient {
	m.responses = responses
	return m
}

func (m *MockLLMClient) CanUseLLM() error {
	return nil
}

func (m *MockLLMClient) CallLLM(prompt, repositoryPath string, opts *CallLLMOptions) error {
	logme.Debugln("Mock LLM client called with repository:", repositoryPath)

	repliesPath := filepath.Join(repositoryPath, "replies.json")

	responseData, err := json.MarshalIndent(m.responses, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(repliesPath, responseData, 0644)
}

// MockAgenticClient implements AgenticClient for testing.
type MockAgenticClient struct {
	answers []AnswerSchema
}

func NewMockAgenticClient(answers []AnswerSchema) *MockAgenticClient {
	return &MockAgenticClient{answers: answers}
}

func (m *MockAgenticClient) CallLLM(ctx context.Context, questions []string, repoPath string) ([]AnswerSchema, error) {
	logme.Debugln("Mock agentic client called with", len(questions), "questions")
	// Match real AgenticClient behavior: return one answer per question
	count := len(questions)
	if count > len(m.answers) {
		count = len(m.answers)
	}
	return m.answers[:count], nil
}

