// Package scanprocess records scanner lifecycle events without logging arguments or source data.
package scanprocess

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Prefix distinguishes structured events from arbitrary scanner stderr.
const Prefix = "PLUGIN_VALIDATOR_EVENT "

type Event struct {
	Event               string `json:"event"`
	Analyzer            string `json:"analyzer,omitempty"`
	Scanner             string `json:"scanner,omitempty"`
	DurationMS          int64  `json:"duration_ms,omitempty"`
	ExitCode            *int   `json:"exit_code,omitempty"`
	ProcessPeakRSSBytes *int64 `json:"process_peak_rss_bytes,omitempty"`
	Failed              bool   `json:"failed"`
}

func Emit(event Event) {
	data, err := json.Marshal(event)
	if err == nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s%s\n", Prefix, data)
	}
}

func observe(cmd *exec.Cmd) func(error) {
	name := filepath.Base(cmd.Path)
	Emit(Event{Event: "scanner_started", Scanner: name})
	started := time.Now()
	return func(err error) {
		event := Event{Event: "scanner_finished", Scanner: name, DurationMS: time.Since(started).Milliseconds(), Failed: err != nil}
		if cmd.ProcessState != nil {
			code := cmd.ProcessState.ExitCode()
			event.ExitCode = &code
			event.ProcessPeakRSSBytes = peakRSS(cmd.ProcessState)
		}
		Emit(event)
	}
}

func Run(cmd *exec.Cmd) error {
	finish := observe(cmd)
	err := cmd.Run()
	finish(err)
	return err
}

func Output(cmd *exec.Cmd) ([]byte, error) {
	finish := observe(cmd)
	out, err := cmd.Output()
	finish(err)
	return out, err
}

func CombinedOutput(cmd *exec.Cmd) ([]byte, error) {
	finish := observe(cmd)
	out, err := cmd.CombinedOutput()
	finish(err)
	return out, err
}
