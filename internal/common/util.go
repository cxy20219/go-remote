package common

import (
	"log/slog"
	"os"
	"runtime"
)

// GetHostname returns the hostname of the current machine
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		slog.Warn("failed to get hostname", "error", err)
		return "unknown"
	}
	return hostname
}

// GetOS returns the operating system name
func GetOS() string {
	return runtime.GOOS
}
