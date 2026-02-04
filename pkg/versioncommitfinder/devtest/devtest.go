// Package main provides a manual test harness for versioncommitfinder.
// Run from repo root: go run ./pkg/versioncommitfinder/devtest -url <github-url>
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/grafana/plugin-validator/pkg/versioncommitfinder"
)

func main() {
	githubURL := flag.String("url", "", "GitHub URL to analyze (e.g., https://github.com/OpenNMS/grafana-plugin/tree/v9.0.16)")
	repoPath := flag.String("repo-path", "", "Optional: local path to already cloned repo (speeds up testing)")
	flag.Parse()

	if *githubURL == "" {
		fmt.Fprintln(os.Stderr, "Error: -url flag is required")
		fmt.Fprintln(os.Stderr, "Usage: go run ./pkg/versioncommitfinder/devtest -url <github-url>")
		fmt.Fprintln(os.Stderr, "Example: go run ./pkg/versioncommitfinder/devtest -url https://github.com/OpenNMS/grafana-plugin/tree/v9.0.16")
		os.Exit(1)
	}

	fmt.Printf("Analyzing: %s\n", *githubURL)
	if *repoPath != "" {
		fmt.Printf("Using local repo: %s\n", *repoPath)
	}
	fmt.Println("---")

	result, cleanup, err := versioncommitfinder.FindPluginVersionsRefs(*githubURL, *repoPath)
	if cleanup != nil {
		defer cleanup()
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	fmt.Println("---")
	fmt.Println("Result:")
	output, marshalErr := json.MarshalIndent(result, "", "  ")
	if marshalErr != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling result: %v\n", marshalErr)
		os.Exit(1)
	}
	fmt.Println(string(output))

	if err != nil {
		os.Exit(1)
	}
}
