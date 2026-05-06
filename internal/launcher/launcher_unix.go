//go:build !windows
// +build !windows

package launcher

import (
	"os/exec"
	"time"
)

// hideConsoleWindow is a no-op on non-Windows platforms
func hideConsoleWindow(cmd *exec.Cmd) {}

func writeWindowsCredential(target, username, password string) error { return nil }
func deleteWindowsCredential(target string) error                    { return nil }
func waitForMstscConnected(pid int, timeout time.Duration, exited <-chan struct{}) {}
func cleanupOrphanedCredentials()                                    {}
