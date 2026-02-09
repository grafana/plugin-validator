package llmclient

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/logme"
)

var ErrAPIKeyNotSet = errors.New("GEMINI_API_KEY not set")

type CallLLMOptions struct {
	Model        string // e.g. "gemini-2.5-flash", empty = CLI default
	ApprovalMode string // "default", "yolo", etc. empty = default (stdin piping)
}

type LLMClient interface {
	CanUseLLM() error
	CallLLM(prompt, repositoryPath string, opts *CallLLMOptions) error
}

func (g *GeminiClient) CanUseLLM() error {
	if os.Getenv("GEMINI_API_KEY") == "" {
		return ErrAPIKeyNotSet
	}
	_, err := getGeminiBinaryPath()
	if err != nil {
		return err
	}
	return nil
}

type GeminiClient struct{}

func NewGeminiClient() *GeminiClient {
	return &GeminiClient{}
}

var cachedGeminiBinPath string

// getGeminiBinaryPath returns the path to the gemini binary.
// It first checks PATH, then falls back to a local npm install.
// The result is cached after the first successful resolution.
func getGeminiBinaryPath() (string, error) {
	if cachedGeminiBinPath != "" {
		return cachedGeminiBinPath, nil
	}

	if p, err := exec.LookPath("gemini"); err == nil {
		cachedGeminiBinPath = p
		return p, nil
	}

	if _, err := exec.LookPath("npm"); err != nil {
		return "", fmt.Errorf("neither gemini nor npm available in PATH")
	}

	dir, err := os.MkdirTemp("", "gemini-cli-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}

	logme.DebugFln("Installing gemini CLI locally to %s", dir)
	install := exec.Command("npm", "install", "@google/gemini-cli")
	install.Dir = dir
	if out, err := install.CombinedOutput(); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("npm install failed: %s: %w", string(out), err)
	}

	bin := filepath.Join(dir, "node_modules", ".bin", "gemini")
	if _, err := os.Stat(bin); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("gemini binary not found after install")
	}

	cachedGeminiBinPath = bin
	logme.DebugFln("Gemini CLI installed at %s", bin)
	return bin, nil
}

func (g *GeminiClient) CallLLM(prompt, repositoryPath string, opts *CallLLMOptions) error {
	if err := g.CanUseLLM(); err != nil {
		return err
	}

	geminiBin, err := getGeminiBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to get gemini CLI: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	args := []string{}

	if opts != nil && opts.Model != "" {
		args = append(args, "-m", opts.Model)
	}

	if opts != nil && opts.ApprovalMode != "" {
		args = append(args, "--approval-mode", opts.ApprovalMode)
	}

	cmd := exec.CommandContext(ctx, geminiBin, args...)
	cmd.Dir = repositoryPath
	cmd.Stdin = strings.NewReader(prompt)

	if os.Getenv("DEBUG") != "" {
		cmd.Stdout = os.Stdout
	}

	logme.Debugln("Running gemini CLI analysis in directory:", repositoryPath)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gemini CLI timed out after 5 minutes")
		}
		return fmt.Errorf("gemini CLI failed: %w", err)
	}

	return nil
}
