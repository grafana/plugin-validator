package llmclient

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/logme"
)

type LLMClient interface {
	CallLLM(prompt, repositoryPath string) error
}

type GeminiClient struct{}

func NewGeminiClient() *GeminiClient {
	return &GeminiClient{}
}

func (g *GeminiClient) CallLLM(prompt, repositoryPath string) error {
	_, err := exec.LookPath("npx")
	if err != nil {
		return errors.New("npx is not available in PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"npx",
		"-y",
		"https://github.com/google-gemini/gemini-cli",
		"-y",
	)
	cmd.Dir = repositoryPath
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	logme.Debugln("Running gemini CLI analysis in directory:", repositoryPath)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			logme.Debugln("Gemini CLI timed out after 5 minutes")
		} else {
			logme.Debugln("Gemini CLI failed:", err)
		}
	}

	return nil
}

