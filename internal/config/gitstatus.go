package config

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// GitStatus describes whether the local repo is in sync with its remote.
type GitStatus int

const (
	GitStatusUpToDate  GitStatus = iota // local HEAD matches remote
	GitStatusBehind                     // remote has commits not present locally
	GitStatusNotGitRepo                 // config file is not inside a git repository
	GitStatusUnknown                    // could not determine (no remote, network error, detached HEAD, etc.)
)

// CheckConfigGitStatus compares the local HEAD hash of the repository containing
// configPath against the remote using git ls-remote (no fetch, no local state change).
// Returns GitStatusUnknown on any error so callers can silently skip the notification.
func CheckConfigGitStatus(configPath string) GitStatus {
	dir := filepath.Dir(configPath)
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	// Confirm we are inside a git repository.
	if err := gitCmd(dir, env, "rev-parse", "--is-inside-work-tree"); err != nil {
		return GitStatusNotGitRepo
	}

	// Determine the current branch name.
	branch, err := gitOutput(dir, env, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "HEAD" {
		return GitStatusUnknown // detached HEAD or error
	}

	// Get local HEAD commit hash.
	localHash, err := gitOutput(dir, env, "rev-parse", "HEAD")
	if err != nil {
		return GitStatusUnknown
	}

	// Query the remote for the branch tip — no local state is modified.
	remoteHash, err := lsRemote(dir, env, branch)
	if err != nil {
		return GitStatusUnknown
	}

	if remoteHash == localHash {
		return GitStatusUpToDate
	}
	return GitStatusBehind
}

// PullConfig runs git pull in the directory containing configPath.
// It uses a 30-second timeout and disables credential prompts so it never hangs.
func PullConfig(configPath string) error {
	dir := filepath.Dir(configPath)
	env := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", dir, "pull")
	cmd.Env = env
	hideConsoleWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// lsRemote runs git ls-remote with a 5-second timeout and returns the commit hash
// for the given branch, or an error if the remote cannot be reached or the branch
// is not found.
func lsRemote(dir string, env []string, branch string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "-C", dir, "ls-remote", "origin", branch)
	cmd.Env = env
	hideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil || ctx.Err() != nil {
		return "", err
	}

	// Output format: "<hash>\t<ref>\n"
	fields := strings.Fields(string(out))
	if len(fields) == 0 {
		return "", nil
	}
	return fields[0], nil
}

func gitCmd(dir string, env []string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = env
	hideConsoleWindow(cmd)
	return cmd.Run()
}

func gitOutput(dir string, env []string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = env
	hideConsoleWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
