package versioncompare

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestVersionComparisonIntegration(t *testing.T) {
	// Hardcoded test values
	githubURL := "https://github.com/taosdata/grafanaplugin/"
	archivePath := "/home/academo/repos/random-plugins/tdengine-datasource/"

	fmt.Printf("Testing version comparison with:\n")
	fmt.Printf("GitHub URL: %s\n", githubURL)
	fmt.Printf("Archive Path: %s\n", archivePath)
	fmt.Printf("\n")

	// Create version comparer
	comparer := New()

	// Test full comparison
	fmt.Println("=== Full version comparison ===")
	result, err := comparer.CompareVersions(githubURL, archivePath)
	if err != nil {
		t.Fatalf("Version comparison failed: %v", err)
	}

	// Pretty print the result
	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Comparison result:\n%s\n", string(resultJSON))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
