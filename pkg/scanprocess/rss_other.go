//go:build !linux && !darwin

package scanprocess

import "os"

func peakRSS(*os.ProcessState) *int64 { return nil }
