package llmclient

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/plugin-validator/pkg/logme"
)

//go:embed settings.json
var embeddedSettings []byte

var ErrAPIKeyNotSet = errors.New("GEMINI_API_KEY not set")

// filesToClean are files that should be removed from the working directory
// before calling the LLM to avoid influencing its behavior.
var filesToClean = []string{
	"GEMINI.md", "gemini.md",
	"CLAUDE.md", "claude.md",
	"AGENTS.md", "agents.md",
	"COPILOT.md", "copilot.md",
	"replies.json",
	"output.json",
}

// CleanUpPromptFiles removes agent config files and known output files
// from the given directory to avoid influencing the LLM.
func CleanUpPromptFiles(dir string) {
	for _, file := range filesToClean {
		p := filepath.Join(dir, file)
		if _, err := os.Stat(p); err == nil {
			if err := os.Remove(p); err != nil {
				logme.DebugFln("Failed to remove %s: %v", p, err)
			}
		}
	}
}

type CallLLMOptions struct {
	Model string // e.g. "gemini-2.5-flash", empty = CLI default
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
var cachedSettingsPath string

// getSettingsPath writes the embedded settings.json to a temp file once and returns its path.
func getSettingsPath() (string, error) {
	if cachedSettingsPath != "" {
		return cachedSettingsPath, nil
	}

	dir, err := os.MkdirTemp("", "gemini-settings-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp settings dir: %w", err)
	}

	p := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(p, embeddedSettings, 0644); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("failed to write settings file: %w", err)
	}

	cachedSettingsPath = p
	logme.DebugFln("Gemini settings written to %s", cachedSettingsPath)
	return cachedSettingsPath, nil
}

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

	settingsPath, err := getSettingsPath()
	if err != nil {
		return fmt.Errorf("failed to prepare settings: %w", err)
	}

	args := []string{}

	if opts != nil && opts.Model != "" {
		args = append(args, "-m", opts.Model)
	}

	cmd := exec.CommandContext(ctx, geminiBin, args...)
	cmd.Dir = repositoryPath
	cmd.Stdin = strings.NewReader(prompt)
	cmd.Env = append(os.Environ(), "GEMINI_CLI_SYSTEM_SETTINGS_PATH="+settingsPath)

	if os.Getenv("DEBUG") != "" {
		cmd.Stdout = os.Stdout
	}

	logme.DebugFln("Running: GEMINI_CLI_SYSTEM_SETTINGS_PATH=%s %s %s", settingsPath, geminiBin, strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("gemini CLI timed out after 5 minutes")
		}
		return fmt.Errorf("gemini CLI failed: %w", err)
	}

	return nil
}
