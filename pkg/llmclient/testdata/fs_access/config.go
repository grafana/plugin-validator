package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	DatabaseURL string `json:"database_url"`
	Port        int    `json:"port"`
	LogLevel    string `json:"log_level"`
	PluginDir   string `json:"plugin_dir"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) PluginPath(name string) string {
	return filepath.Join(c.PluginDir, name)
}

func DefaultConfig() *Config {
	return &Config{
		DatabaseURL: "postgres://localhost:5432/plugins",
		Port:        8080,
		LogLevel:    "info",
		PluginDir:   "/var/lib/plugins",
	}
}
