package llmclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

type providerConfig struct {
	name     string
	provider string
	model    string
	envKey   string
}

var providers = []providerConfig{
	{name: "Gemini", provider: "google", model: "gemini-2.5-flash", envKey: "GEMINI_API_KEY"},
	{
		name:     "Anthropic",
		provider: "anthropic",
		model:    "claude-haiku-4-5",
		envKey:   "ANTHROPIC_API_KEY",
	},
	{name: "OpenAI", provider: "openai", model: "gpt-5-mini", envKey: "OPENAI_API_KEY"},
}

func skipIfMissingKey(t *testing.T, p providerConfig) {
	t.Helper()
	if os.Getenv(p.envKey) == "" || os.Getenv("DEBUG") != "1" {
		t.Skipf("%s not set or DEBUG!=1, skipping %s integration test", p.envKey, p.name)
	}
}

func newClient(t *testing.T, p providerConfig) AgenticClient {
	t.Helper()
	client, err := NewAgenticClient(&AgenticCallOptions{
		Provider: p.provider,
		Model:    p.model,
		APIKey:   os.Getenv(p.envKey),
	})
	require.NoError(t, err)
	return client
}

func TestAgenticClient_EmptyQuestions(t *testing.T) {
	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			skipIfMissingKey(t, p)

			client := newClient(t, p)

			testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
			require.NoError(t, err)

			answers, err := client.CallLLM(context.Background(), []string{}, testDataPath)
			require.Error(t, err, "Empty questions should return error")
			require.Contains(t, err.Error(), "at least one question is required")
			require.Nil(t, answers)
		})
	}
}

func TestAgenticClient_NoFilesystemAccess(t *testing.T) {
	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			skipIfMissingKey(t, p)

			client := newClient(t, p)

			testDataPath, err := filepath.Abs(filepath.Join("testdata", "no_fs_access"))
			require.NoError(t, err)

			prompt := "Does this application access the filesystem (read or write files)?"

			answers, err := client.CallLLM(context.Background(), []string{prompt}, testDataPath)

			require.NoError(t, err, "CallLLM should not return error")
			require.Len(t, answers, 1, "Should return exactly one answer")

			answer := answers[0]
			require.Equal(t, prompt, answer.Question, "Question field should match input question")
			require.NotEmpty(t, answer.Answer, "Answer field should be populated")
			require.Equal(t, false, answer.ShortAnswer,
				"ShortAnswer should be false - this app does not access the filesystem")
		})
	}
}

func TestAgenticClient_FilesystemAccess(t *testing.T) {
	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			skipIfMissingKey(t, p)

			client := newClient(t, p)

			testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
			require.NoError(t, err)

			prompt := "Does this application access the filesystem (read or write files)?"

			answers, err := client.CallLLM(context.Background(), []string{prompt}, testDataPath)

			require.NoError(t, err, "CallLLM should not return error")
			require.Len(t, answers, 1, "Should return exactly one answer")

			answer := answers[0]
			require.Equal(t, prompt, answer.Question, "Question field should match input question")
			require.NotEmpty(t, answer.Answer, "Answer field should be populated")
			require.Equal(t, true, answer.ShortAnswer,
				"ShortAnswer should be true - this app accesses the filesystem via os.ReadFile")
		})
	}
}

func TestAgenticClient_TwoQuestions(t *testing.T) {
	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			skipIfMissingKey(t, p)

			client := newClient(t, p)

			testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
			require.NoError(t, err)

			questions := []string{
				"Does this application access the filesystem (read or write files)?",
				"Does this application make any external HTTP requests to a remote server?",
			}

			answers, err := client.CallLLM(context.Background(), questions, testDataPath)

			require.NoError(t, err, "CallLLM should not return error")
			require.Len(t, answers, 2, "Should return exactly two answers")

			require.Equal(
				t,
				questions[0],
				answers[0].Question,
				"First answer's question should match",
			)
			require.NotEmpty(t, answers[0].Answer, "First answer should be populated")
			require.Equal(t, true, answers[0].ShortAnswer,
				"First answer should be true - app accesses filesystem")

			require.Equal(
				t,
				questions[1],
				answers[1].Question,
				"Second answer's question should match",
			)
			require.NotEmpty(t, answers[1].Answer, "Second answer should be populated")
			require.Equal(t, false, answers[1].ShortAnswer,
				"Second answer should be false - app does not make HTTP requests")
		})
	}
}

func TestAgenticClient_ThreeQuestions(t *testing.T) {
	for _, p := range providers {
		t.Run(p.name, func(t *testing.T) {
			skipIfMissingKey(t, p)

			client := newClient(t, p)

			testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
			require.NoError(t, err)

			questions := []string{
				"Does this application access the filesystem (read or write files)?",
				"Does this application make any external HTTP requests to a remote server?",
				"Does this application use any caching mechanisms?",
			}

			answers, err := client.CallLLM(context.Background(), questions, testDataPath)

			require.NoError(t, err, "CallLLM should not return error")
			require.Len(t, answers, 3, "Should return exactly three answers")

			require.Equal(
				t,
				questions[0],
				answers[0].Question,
				"First answer's question should match",
			)
			require.NotEmpty(t, answers[0].Answer, "First answer should be populated")
			require.Equal(t, true, answers[0].ShortAnswer,
				"First answer should be true - app accesses filesystem")

			require.Equal(
				t,
				questions[1],
				answers[1].Question,
				"Second answer's question should match",
			)
			require.NotEmpty(t, answers[1].Answer, "Second answer should be populated")
			require.Equal(t, false, answers[1].ShortAnswer,
				"Second answer should be false - app does not make HTTP requests")

			require.Equal(
				t,
				questions[2],
				answers[2].Question,
				"Third answer's question should match",
			)
			require.NotEmpty(t, answers[2].Answer, "Third answer should be populated")
			require.Equal(t, true, answers[2].ShortAnswer,
				"Third answer should be true - app uses caching")
		})
	}
}
