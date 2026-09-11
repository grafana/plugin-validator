//go:build linux || darwin

package scanprocess

import (
	"os"
	"runtime"
	"syscall"
)

// This is the OS-reported process high-water mark, not aggregate process-tree memory.
func peakRSS(state *os.ProcessState) *int64 {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok {
		return nil
	}
	bytes := int64(usage.Maxrss)
	if runtime.GOOS == "linux" {
		bytes *= 1024
	}
	return &bytes
}
