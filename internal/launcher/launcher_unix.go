//go:build !windows
// +build !windows

package launcher

import (
	"os/exec"
)

// hideConsoleWindow is a no-op on non-Windows platforms
func hideConsoleWindow(cmd *exec.Cmd) {
	// Nothing to do on Unix-like systems
}

func writeWindowsCredential(target, username, password string) error { return nil }
func deleteWindowsCredential(target string) error                    { return nil }
