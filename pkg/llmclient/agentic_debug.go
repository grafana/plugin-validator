package llmclient

import (
	"fmt"
	"os"

	"github.com/grafana/plugin-validator/pkg/logme"
)

// debugLog writes a formatted message to the LLM debug log file
func debugLog(format string, args ...interface{}) {
	logme.LLMLog(format, args...)
}

// printDebugLogPath prints the debug log file path to stderr if debug is enabled
func printDebugLogPath() {
	if p := logme.LLMLogPath(); p != "" {
		fmt.Fprintf(os.Stderr, "AgenticClient: debug log: %s\n", p)
	}
}
