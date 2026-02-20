package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PluginScanner struct {
	baseDir    string
	extensions []string
}

func NewPluginScanner(baseDir string) *PluginScanner {
	return &PluginScanner{
		baseDir:    baseDir,
		extensions: []string{".so", ".dll", ".dylib"},
	}
}

func (s *PluginScanner) Scan() ([]string, error) {
	var plugins []string

	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("scanning plugin directory %s: %w", s.baseDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		for _, ext := range s.extensions {
			if strings.HasSuffix(entry.Name(), ext) {
				plugins = append(plugins, filepath.Join(s.baseDir, entry.Name()))
			}
		}
	}

	return plugins, nil
}

func (s *PluginScanner) Exists(name string) bool {
	path := filepath.Join(s.baseDir, name)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
