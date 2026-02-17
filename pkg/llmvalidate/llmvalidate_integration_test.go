package llmvalidate

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

// TestLLMIntegration_PositiveCase tests that the LLM correctly identifies
// when the marker exists in the code
func TestLLMIntegration_PositiveCase(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping LLM integration test")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := New(context.Background(), "google", "gemini-3-flash-preview", apiKey)
	require.NoError(t, err, "Failed to create LLM client")

	testDataPath := filepath.Join("testdata", "src")

	questions := []LLMQuestion{
		{
			Question:       "Does the file sample1.go contain a comment with the exact text 'MAGIC_MARKER_BANANA_12345'?",
			ExpectedAnswer: true,
		},
	}

	answers, err := client.AskLLMAboutCode(testDataPath, questions, []string{"."})
	logme.DebugFln("LLM answers :")
	prettyprint.Print(answers)

	require.NoError(t, err, "AskLLMAboutCode should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]

	// Verify structured output fields are populated
	require.NotEmpty(t, answer.Question, "Question field should be populated")
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(t, true, answer.ShortAnswer, "ShortAnswer should be true when marker exists")
	require.Equal(
		t,
		questions[0].ExpectedAnswer,
		answer.ExpectedShortAnswer,
		"ExpectedShortAnswer should match",
	)

	// The question should be preserved
	require.Contains(
		t,
		answer.Question,
		"MAGIC_MARKER_BANANA_12345",
		"Question should contain the marker text",
	)

	t.Logf("LLM Answer: %s", answer.Answer)
}

// TestLLMIntegration_PositiveCase_Anthropic tests that the LLM correctly identifies
// when the marker exists in the code using Anthropic Claude
func TestLLMIntegration_PositiveCase_Anthropic(t *testing.T) {
	if !hasAnthropicAPIKey() {
		t.Skip("ANTHROPIC_API_KEY not set, skipping Anthropic LLM integration test")
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	client, err := New(context.Background(), "anthropic", "claude-sonnet-4-5", apiKey)
	require.NoError(t, err, "Failed to create LLM client")

	testDataPath := filepath.Join("testdata", "src")

	questions := []LLMQuestion{
		{
			Question:       "Does the file sample1.go contain a comment with the exact text 'MAGIC_MARKER_BANANA_12345'?",
			ExpectedAnswer: true,
		},
	}

	answers, err := client.AskLLMAboutCode(testDataPath, questions, []string{"."})
	logme.DebugFln("LLM answers :")
	prettyprint.Print(answers)

	require.NoError(t, err, "AskLLMAboutCode should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]

	// Verify structured output fields are populated
	require.NotEmpty(t, answer.Question, "Question field should be populated")
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(t, true, answer.ShortAnswer, "ShortAnswer should be true when marker exists")
	require.Equal(
		t,
		questions[0].ExpectedAnswer,
		answer.ExpectedShortAnswer,
		"ExpectedShortAnswer should match",
	)

	// The question should be preserved
	require.Contains(
		t,
		answer.Question,
		"MAGIC_MARKER_BANANA_12345",
		"Question should contain the marker text",
	)

	t.Logf("LLM Answer: %s", answer.Answer)
}

// TestLLMIntegration_NegativeCase tests that the LLM correctly identifies
// when the marker does NOT exist in a different file
func TestLLMIntegration_NegativeCase(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping LLM integration test")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := New(context.Background(), "google", "gemini-3-flash-preview", apiKey)

	require.NoError(t, err, "Failed to create LLM client")

	testDataPath := filepath.Join("testdata", "src")

	questions := []LLMQuestion{
		{
			Question:       "Does the file sample2.go contain a comment with the exact text 'MAGIC_MARKER_BANANA_12345'?",
			ExpectedAnswer: false,
		},
	}

	answers, err := client.AskLLMAboutCode(testDataPath, questions, []string{"."})

	require.NoError(t, err, "AskLLMAboutCode should not return error")
	require.Len(t, answers, 1, "Should return exactly one answer")

	answer := answers[0]

	// Verify structured output fields are populated
	require.NotEmpty(t, answer.Question, "Question field should be populated")
	require.NotEmpty(t, answer.Answer, "Answer field should be populated")
	require.Equal(
		t,
		false,
		answer.ShortAnswer,
		"ShortAnswer should be false when marker doesn't exist",
	)
	require.Equal(
		t,
		questions[0].ExpectedAnswer,
		answer.ExpectedShortAnswer,
		"ExpectedShortAnswer should match",
	)

	t.Logf("LLM Answer: %s", answer.Answer)
}

// TestLLMIntegration_MultipleQuestions tests that structured output works with multiple questions
func TestLLMIntegration_MultipleQuestions(t *testing.T) {
	if !hasGeminiAPIKey() {
		t.Skip("GEMINI_API_KEY not set, skipping LLM integration test")
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := New(context.Background(), "google", "gemini-3-flash-preview", apiKey)
	require.NoError(t, err, "Failed to create LLM client")

	testDataPath := filepath.Join("testdata", "src")

	questions := []LLMQuestion{
		{
			Question:       "Does the file sample1.go contain the text 'MAGIC_MARKER_BANANA_12345'?",
			ExpectedAnswer: true,
		},
		{
			Question:       "Does the file sample1.go contain the function name 'HelloWorld'?",
			ExpectedAnswer: true,
		},
		{
			Question:       "Does the file sample1.go contain the text 'NONEXISTENT_MARKER'?",
			ExpectedAnswer: false,
		},
	}

	answers, err := client.AskLLMAboutCode(testDataPath, questions, []string{"."})

	require.NoError(t, err, "AskLLMAboutCode should not return error")
	require.Len(t, answers, 3, "Should return three answers")

	// Verify all answers have structured output
	for i, answer := range answers {
		require.NotEmpty(t, answer.Question, "Question %d should be populated", i)
		require.NotEmpty(t, answer.Answer, "Answer %d should be populated", i)
		require.Equal(
			t,
			questions[i].ExpectedAnswer,
			answer.ExpectedShortAnswer,
			"ExpectedShortAnswer %d should match",
			i,
		)

		t.Logf("Question %d: %s", i+1, answer.Question)
		t.Logf("Answer %d: %s", i+1, answer.Answer)
		t.Logf("ShortAnswer %d: %v", i+1, answer.ShortAnswer)
		t.Logf("---")
	}

	// Verify the specific expected answers
	require.Equal(t, true, answers[0].ShortAnswer, "First question: marker should exist")
	require.Equal(t, true, answers[1].ShortAnswer, "Second question: HelloWorld should exist")
	require.Equal(
		t,
		false,
		answers[2].ShortAnswer,
		"Third question: NONEXISTENT_MARKER should not exist",
	)
}
