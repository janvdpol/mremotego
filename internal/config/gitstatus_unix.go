//go:build !windows
// +build !windows

package config

import "os/exec"

func hideConsoleWindow(cmd *exec.Cmd) {}
