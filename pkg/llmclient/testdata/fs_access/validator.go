package main

import (
	"fmt"
	"strings"
)

type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors  []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

type Validator struct {
	rules []ValidationRule
}

type ValidationRule struct {
	Name    string
	Check   func(string) error
}

func NewValidator() *Validator {
	return &Validator{
		rules: []ValidationRule{
			{Name: "not_empty", Check: checkNotEmpty},
			{Name: "valid_version", Check: checkVersion},
			{Name: "no_spaces", Check: checkNoSpaces},
		},
	}
}

func (v *Validator) Validate(pluginName string) ValidationResult {
	result := ValidationResult{Valid: true}

	for _, rule := range v.rules {
		if err := rule.Check(pluginName); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", rule.Name, err))
		}
	}

	return result
}

func checkNotEmpty(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}
	return nil
}

func checkVersion(name string) error {
	parts := strings.Split(name, "-")
	if len(parts) < 2 {
		return fmt.Errorf("plugin name must include version suffix (e.g., plugin-1.0.0)")
	}
	return nil
}

func checkNoSpaces(name string) error {
	if strings.Contains(name, " ") {
		return fmt.Errorf("plugin name cannot contain spaces")
	}
	return nil
}
