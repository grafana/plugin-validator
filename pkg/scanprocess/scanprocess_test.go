package scanprocess

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestScannerEvents(t *testing.T) {
	for _, tc := range []struct {
		name   string
		script string
		code   int
	}{
		{"success", "printf result", 0},
		{"failure", "printf result; exit 2", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "events")
			require.NoError(t, err)
			defer file.Close()
			previous := os.Stderr
			os.Stderr = file
			defer func() { os.Stderr = previous }()
			cmd := exec.Command("sh", "-c", tc.script)
			out, runErr := Output(cmd)
			require.Equal(t, "result", string(out))
			require.Equal(t, tc.code != 0, runErr != nil)
			data, err := os.ReadFile(file.Name())
			require.NoError(t, err)
			lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
			require.Len(t, lines, 2)
			var start, finish Event
			require.NoError(t, json.Unmarshal(bytes.TrimPrefix(lines[0], []byte(Prefix)), &start))
			require.NoError(t, json.Unmarshal(bytes.TrimPrefix(lines[1], []byte(Prefix)), &finish))
			require.Equal(t, "scanner_started", start.Event)
			require.Equal(t, "scanner_finished", finish.Event)
			require.Equal(t, "sh", finish.Scanner)
			require.NotNil(t, finish.ExitCode)
			require.Equal(t, tc.code, *finish.ExitCode)
			require.False(t, strings.Contains(string(data), tc.script))
			if rss := peakRSS(cmd.ProcessState); rss != nil {
				require.NotNil(t, finish.ProcessPeakRSSBytes)
				require.Greater(t, *finish.ProcessPeakRSSBytes, int64(0))
			}
		})
	}
}
