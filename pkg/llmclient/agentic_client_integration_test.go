package llmclient

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func hasGeminiAPIKey() bool {
	return os.Getenv("GEMINI_API_KEY") != "" && os.Getenv("DEBUG") == "1"
}

// only run with debug=1
func hasAnthropicAPIKey() bool {
	return os.Getenv("ANTHROPIC_API_KEY") != "" && os.Getenv("DEBUG") == "1"
}

const (
	googleAgenticTestModel    = defaultGoogleAgenticModel
	anthropicAgenticTestModel = defaultAnthropicModel
)

// TestAgenticClient_NoFilesystemAccess tests that the agent correctly identifies
// when an application does NOT access the filesystem
func TestAgenticClient_NoFilesystemAccess(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping agentic client integration test")
	}

	opts := &AgenticCallOptions{
		Provider: "google",
		Model:    googleAgenticTestModel,
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "no_fs_access"))
	require.NoError(t, err)

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), []string{prompt}, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(
		t,
		false,
		answer.ShortAnswer,
		"ShortAnswer should be false - this app does not access the filesystem",
	)

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

	opts := &AgenticCallOptions{
		Provider: "google",
		Model:    googleAgenticTestModel,
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
	require.NoError(t, err)

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), []string{prompt}, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(
		t,
		true,
		answer.ShortAnswer,
		"ShortAnswer should be true - this app accesses the filesystem via os.ReadFile",
	)

	if len(answer.Files) > 0 {
		t.Logf("Files: %v", answer.Files)
	}
}

// TestAgenticClient_NoFilesystemAccess_Anthropic tests the same scenario using Anthropic Claude
func TestAgenticClient_NoFilesystemAccess_Anthropic(t *testing.T) {
	if !hasAnthropicAPIKey() {
		t.Skip("ANTHROPIC_API_KEY not set, skipping Anthropic agentic client integration test")
	}

	opts := &AgenticCallOptions{
		Provider: "anthropic",
		Model:    anthropicAgenticTestModel,
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "no_fs_access"))
	require.NoError(t, err)

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), []string{prompt}, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(
		t,
		false,
		answer.ShortAnswer,
		"ShortAnswer should be false - this app does not access the filesystem",
	)

	if len(answer.Files) > 0 {
		t.Logf("Files: %v", answer.Files)
	}
}

// TestAgenticClient_FilesystemAccess_Anthropic tests the same scenario using Anthropic Claude
func TestAgenticClient_FilesystemAccess_Anthropic(t *testing.T) {
	if !hasAnthropicAPIKey() {
		t.Skip("ANTHROPIC_API_KEY not set, skipping Anthropic agentic client integration test")
	}

	opts := &AgenticCallOptions{
		Provider: "anthropic",
		Model:    anthropicAgenticTestModel,
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
	require.NoError(t, err)

	prompt := "Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations."

	answers, err := client.CallLLM(context.Background(), []string{prompt}, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(
		t,
		true,
		answer.ShortAnswer,
		"ShortAnswer should be true - this app accesses the filesystem via os.ReadFile",
	)

	if len(answer.Files) > 0 {
		t.Logf("Files: %v", answer.Files)
	}
}

// TestAgenticClient_MultiQuestion_2Questions tests sending 2 questions in a single
// pi session against the fs_access testdata.
func TestAgenticClient_MultiQuestion_2Questions(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping agentic client integration test")
	}

	opts := &AgenticCallOptions{
		Provider: "google",
		Model:    googleAgenticTestModel,
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
	require.NoError(t, err)

	prompts := []string{
		"Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations.",
		"Does this application start or use an HTTP server? Examine the code for any HTTP server setup, route handlers, or listener configuration.",
	}

	answers, err := client.CallLLM(context.Background(), prompts, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 2, "Should return exactly two answers")

	require.NotEmpty(t, answers[0].Answer, "Answer 1 should be populated")
	require.Equal(t, true, answers[0].ShortAnswer,
		"Answer 1: ShortAnswer should be true - this app accesses the filesystem")

	require.NotEmpty(t, answers[1].Answer, "Answer 2 should be populated")
	require.Equal(t, false, answers[1].ShortAnswer,
		"Answer 2: ShortAnswer should be false - this app does not use an HTTP server")

	for i, a := range answers {
		t.Logf("Answer %d: short_answer=%v, answer=%s", i+1, a.ShortAnswer, a.Answer)
	}
}

// TestAgenticClient_MultiQuestion_3Questions tests sending 3 questions in a single
// pi session against the no_fs_access testdata.
func TestAgenticClient_MultiQuestion_3Questions(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping agentic client integration test")
	}

	opts := &AgenticCallOptions{
		Provider: "google",
		Model:    googleAgenticTestModel,
		APIKey:   os.Getenv("GEMINI_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "no_fs_access"))
	require.NoError(t, err)

	prompts := []string{
		"Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations.",
		"Does this application use an HTTP server? Examine the code for any HTTP server setup, route handlers, or listener configuration.",
		"Does this application use cryptographic functions (hashing, encryption, digital signatures)? Examine the code for any use of crypto libraries or hash functions.",
	}

	answers, err := client.CallLLM(context.Background(), prompts, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 3, "Should return exactly three answers")

	require.NotEmpty(t, answers[0].Answer, "Answer 1 should be populated")
	require.Equal(t, false, answers[0].ShortAnswer,
		"Answer 1: ShortAnswer should be false - this app does not access the filesystem")

	require.NotEmpty(t, answers[1].Answer, "Answer 2 should be populated")
	require.Equal(t, true, answers[1].ShortAnswer,
		"Answer 2: ShortAnswer should be true - this app uses an HTTP server")

	require.NotEmpty(t, answers[2].Answer, "Answer 3 should be populated")
	require.Equal(t, true, answers[2].ShortAnswer,
		"Answer 3: ShortAnswer should be true - this app uses crypto/sha256")

	for i, a := range answers {
		t.Logf("Answer %d: short_answer=%v, answer=%s", i+1, a.ShortAnswer, a.Answer)
	}
}

// TestAgenticClient_MultiQuestion_2Questions_Anthropic tests sending 2 questions
// in a single pi session using Anthropic Claude.
func TestAgenticClient_MultiQuestion_2Questions_Anthropic(t *testing.T) {
	if !hasAnthropicAPIKey() {
		t.Skip("ANTHROPIC_API_KEY not set, skipping Anthropic agentic client integration test")
	}

	opts := &AgenticCallOptions{
		Provider: "anthropic",
		Model:    anthropicAgenticTestModel,
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "fs_access"))
	require.NoError(t, err)

	prompts := []string{
		"Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations.",
		"Does this application start or use an HTTP server? Examine the code for any HTTP server setup, route handlers, or listener configuration.",
	}

	answers, err := client.CallLLM(context.Background(), prompts, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 2, "Should return exactly two answers")

	require.NotEmpty(t, answers[0].Answer, "Answer 1 should be populated")
	require.Equal(t, true, answers[0].ShortAnswer,
		"Answer 1: ShortAnswer should be true - this app accesses the filesystem")

	require.NotEmpty(t, answers[1].Answer, "Answer 2 should be populated")
	require.Equal(t, false, answers[1].ShortAnswer,
		"Answer 2: ShortAnswer should be false - this app does not use an HTTP server")

	for i, a := range answers {
		t.Logf("Answer %d: short_answer=%v, answer=%s", i+1, a.ShortAnswer, a.Answer)
	}
}

// TestAgenticClient_MultiQuestion_3Questions_Anthropic tests sending 3 questions
// in a single pi session using Anthropic Claude.
func TestAgenticClient_MultiQuestion_3Questions_Anthropic(t *testing.T) {
	if !hasAnthropicAPIKey() {
		t.Skip("ANTHROPIC_API_KEY not set, skipping Anthropic agentic client integration test")
	}

	opts := &AgenticCallOptions{
		Provider: "anthropic",
		Model:    anthropicAgenticTestModel,
		APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
	}

	client, err := NewAgenticClient(opts)
	require.NoError(t, err)

	testDataPath, err := filepath.Abs(filepath.Join("testdata", "no_fs_access"))
	require.NoError(t, err)

	prompts := []string{
		"Does this application access the filesystem (read or write files)? Examine the code to determine if it performs any file I/O operations.",
		"Does this application use an HTTP server? Examine the code for any HTTP server setup, route handlers, or listener configuration.",
		"Does this application use cryptographic functions (hashing, encryption, digital signatures)? Examine the code for any use of crypto libraries or hash functions.",
	}

	answers, err := client.CallLLM(context.Background(), prompts, testDataPath)

	require.NoError(t, err, "CallLLM should not return error")
	require.Len(t, answers, 3, "Should return exactly three answers")

	require.NotEmpty(t, answers[0].Answer, "Answer 1 should be populated")
	require.Equal(t, false, answers[0].ShortAnswer,
		"Answer 1: ShortAnswer should be false - this app does not access the filesystem")

	require.NotEmpty(t, answers[1].Answer, "Answer 2 should be populated")
	require.Equal(t, true, answers[1].ShortAnswer,
		"Answer 2: ShortAnswer should be true - this app uses an HTTP server")

	require.NotEmpty(t, answers[2].Answer, "Answer 3 should be populated")
	require.Equal(t, true, answers[2].ShortAnswer,
		"Answer 3: ShortAnswer should be true - this app uses crypto/sha256")

	for i, a := range answers {
		t.Logf("Answer %d: short_answer=%v, answer=%s", i+1, a.ShortAnswer, a.Answer)
	}
}
