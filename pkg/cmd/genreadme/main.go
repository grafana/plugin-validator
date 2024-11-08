package main

import (
	"fmt"
	"os"

	"github.com/grafana/plugin-validator/pkg/genreadme"
)

const readmeFileName = "README.md"

func _main() error {
	// Read existing README
	f, err := os.Open(readmeFileName)
	if err != nil {
		return fmt.Errorf("open readme: %w", err)
	}
	var closed bool
	defer func() {
		if closed {
			return
		}
		_ = f.Close()
	}()
	generatedReadme, err := genreadme.Generate(f)
	if err != nil {
		return fmt.Errorf("generate new readme: %w", err)
	}
	closed = true
	if err = f.Close(); err != nil {
		return fmt.Errorf("close readme: %w", err)
	}

	// Overwrite the README
	outF, err := os.Create(readmeFileName)
	if err != nil {
		return fmt.Errorf("create new readme: %w", err)
	}
	if _, err := outF.WriteString(generatedReadme); err != nil {
		return fmt.Errorf("write new readme: %w", err)
	}
	if err := outF.Close(); err != nil {
		return fmt.Errorf("close new readme: %w", err)
	}
	return nil
}

func main() {
	if err := _main(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
