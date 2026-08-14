//go:build !windows
// +build !windows

package update

// replaceExecutable swaps newExePath into currentExe's place. Unix allows a
// running executable's directory entry to be replaced directly: the
// currently-executing process keeps running off its already-open handle to
// the old (now unlinked) inode, and currentExe's path immediately points at
// the new file for any future invocation.
func replaceExecutable(currentExe, newExePath string) error {
	return renameOrCopy(newExePath, currentExe)
}
