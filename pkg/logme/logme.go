package logme

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var isDebugMode = os.Getenv("DEBUG") == "1"

var (
	llmLogger *log.Logger
	llmOnce   sync.Once
	llmPath   string
)

func initLLMLogger() {
	llmOnce.Do(func() {
		if !isDebugMode {
			llmLogger = log.New(io.Discard, "", 0)
			return
		}

		llmPath = filepath.Join(os.TempDir(), "validator-llm.log")
		f, err := os.OpenFile(llmPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logme: failed to open LLM log file: %v\n", err)
			llmLogger = log.New(io.Discard, "", 0)
			return
		}

		llmLogger = log.New(f, "", log.Ltime|log.Lmicroseconds)
	})
}

// LLMLog writes a formatted message to the LLM debug log file in /tmp.
// Only active when DEBUG=1.
func LLMLog(format string, args ...interface{}) {
	initLLMLogger()
	llmLogger.Printf(format, args...)
}

// LLMLogPath returns the path to the LLM log file, or "" if not active.
func LLMLogPath() string {
	initLLMLogger()
	return llmPath
}

func DebugFln(msg string, args ...interface{}) {
	// check if ENV DEBUG is 1
	if isDebugMode {
		fmt.Print("[DEBUG] ")
		fmt.Fprintf(os.Stdout, msg, args...)
		fmt.Println()
	}
}

func Debugln(args ...interface{}) {
	// check if ENV DEBUG is 1
	if isDebugMode {
		fmt.Print("[DEBUG] ")
		fmt.Fprintln(os.Stdout, args...)
	}
}

func InfoF(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, msg, args...)
}

func Infoln(arg ...interface{}) {
	fmt.Fprintln(os.Stdout, arg...)
}

func ErrorF(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
}

func Errorln(arg ...interface{}) {
	fmt.Fprintln(os.Stderr, arg...)
}
