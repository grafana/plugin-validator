package llmclient

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	debugLogger *log.Logger
	debugOnce   sync.Once
	debugPath   string
)

func initDebugLogger() {
	debugOnce.Do(func() {
		debugVal := os.Getenv("DEBUG")
		if debugVal != "1" && debugVal != "true" {
			debugLogger = log.New(io.Discard, "", 0)
			return
		}

		timestamp := time.Now().Format("20060102-150405")
		debugPath = filepath.Join(os.TempDir(), fmt.Sprintf("validator-agentic-%s.log", timestamp))

		f, err := os.OpenFile(debugPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "AgenticClient: failed to create debug log file: %v\n", err)
			debugLogger = log.New(io.Discard, "", 0)
			return
		}

		debugLogger = log.New(f, "", log.Ltime|log.Lmicroseconds)
	})
}

// debugLog writes a formatted message to the debug log file if DEBUG=1 or DEBUG=true
func debugLog(format string, args ...interface{}) {
	initDebugLogger()
	debugLogger.Printf(format, args...)
}

// printDebugLogPath prints the debug log file path to stderr if debug is enabled
func printDebugLogPath() {
	initDebugLogger()
	if debugPath != "" {
		fmt.Fprintf(os.Stderr, "AgenticClient: debug log: %s\n", debugPath)
	}
}
