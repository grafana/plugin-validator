package llmclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grafana/plugin-validator/pkg/logme"
	"github.com/grafana/plugin-validator/pkg/prettyprint"
	"github.com/stretchr/testify/require"
)

func hasGeminiAPIKey() bool {
	return os.Getenv("GEMINI_API_KEY") != ""
}

func hasAnthropicAPIKey() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != ""
}

// TestAgenticClient_NoFilesystemAccess tests that the agent correctly identifies
// when an application does NOT access the filesystem
func TestAgenticClient_NoFilesystemAccess(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping agentic client integration test")
	}

	client := NewAgenticClient()

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "no_fs_access"))
	require.NoError(t, err)

	opts := &AgenticCallOptions{
		Provider: "google",
		Model:    "gemini-2.0-flash",
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), prompt, testDataPath, opts)
	logme.DebugFln("Agent answers:")
	prettyprint.Print(answers)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(t, false, answer.ShortAnswer, "ShortAnswer should be false - this app does not access the filesystem")

	t.Logf("Agent Answer: %s", answer.Answer)
	t.Logf("Short Answer: %v", answer.ShortAnswer)
	if len(answer.Files) > 0 {
		t.Logf("Files: %v", answer.Files)
	}
}

// TestAgenticClient_FilesystemAccess tests that the agent correctly identifies
// when an application DOES access the filesystem
func TestAgenticClient_FilesystemAccess(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping agentic client integration test")
	}

	client := NewAgenticClient()

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
	require.NoError(t, err)

	opts := &AgenticCallOptions{
		Provider: "google",
		Model:    "gemini-2.0-flash",
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), prompt, testDataPath, opts)
	logme.DebugFln("Agent answers:")
	prettyprint.Print(answers)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(t, true, answer.ShortAnswer, "ShortAnswer should be true - this app accesses the filesystem via os.ReadFile")

	t.Logf("Agent Answer: %s", answer.Answer)
	t.Logf("Short Answer: %v", answer.ShortAnswer)
	if len(answer.Files) > 0 {
		t.Logf("Files: %v", answer.Files)
	}
}

// TestAgenticClient_NoFilesystemAccess_Anthropic tests the same scenario using Anthropic Claude
func TestAgenticClient_NoFilesystemAccess_Anthropic(t *testing.T) {
	if !hasAnthropicAPIKey() {
		t.Skip("ANTHROPIC_API_KEY not set, skipping Anthropic agentic client integration test")
	}

	client := NewAgenticClient()

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "no_fs_access"))
	require.NoError(t, err)

	opts := &AgenticCallOptions{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	}

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), prompt, testDataPath, opts)
	logme.DebugFln("Agent answers:")
	prettyprint.Print(answers)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(t, false, answer.ShortAnswer, "ShortAnswer should be false - this app does not access the filesystem")

	t.Logf("Agent Answer: %s", answer.Answer)
	t.Logf("Short Answer: %v", answer.ShortAnswer)
	if len(answer.Files) > 0 {
		t.Logf("Files: %v", answer.Files)
	}
}

// TestAgenticClient_FilesystemAccess_Anthropic tests the same scenario using Anthropic Claude
func TestAgenticClient_FilesystemAccess_Anthropic(t *testing.T) {
	if !hasAnthropicAPIKey() {
		t.Skip("ANTHROPIC_API_KEY not set, skipping Anthropic agentic client integration test")
	}

	client := NewAgenticClient()

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
	require.NoError(t, err)

	opts := &AgenticCallOptions{
		Provider: "anthropic",
		Model:    "claude-sonnet-4-5",
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	}

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), prompt, testDataPath, opts)
	logme.DebugFln("Agent answers:")
	prettyprint.Print(answers)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(t, true, answer.ShortAnswer, "ShortAnswer should be true - this app accesses the filesystem via os.ReadFile")

	t.Logf("Agent Answer: %s", answer.Answer)
	t.Logf("Short Answer: %v", answer.ShortAnswer)
	if len(answer.Files) > 0 {
		t.Logf("Files: %v", answer.Files)
	}
}
