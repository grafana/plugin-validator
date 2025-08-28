package versioncommitfinder

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestVersionComparisonIntegration(t *testing.T) {
	// Hardcoded test values
	githubURL := "https://github.com/taosdata/grafanaplugin/tree/v3.7.3"
	archivePath := "/home/academo/repos/random-plugins/tdengine-datasource/"

	fmt.Printf("Testing version comparison with:\n")
	fmt.Printf("GitHub URL: %s\n", githubURL)
	fmt.Printf("Archive Path: %s\n", archivePath)
	fmt.Printf("\n")

	fmt.Println("=== Full version comparison ===")
	result, err := FindPluginVersionsRefs(githubURL, archivePath)
	if err != nil {
		t.Fatalf("Version comparison failed: %v", err)
	}

	resultJSON, _ := json.MarshalIndent(result, "", "  ")
	fmt.Printf("Comparison result:\n%s\n", string(resultJSON))
}
