//go:build windows
// +build windows

package update

import (
	"fmt"
	"os"
)

// replaceExecutable swaps newExePath into currentExe's place. Windows opens a
// running executable's image with FILE_SHARE_DELETE, which permits renaming
// (or deleting) it while it's still executing — but not overwriting its
// content in place. So the running exe is renamed aside to "<exe>.old" first
// (cleaned up on the next startup by CleanupOnStartup), freeing up currentExe's
// path for the new binary.
func replaceExecutable(currentExe, newExePath string) error {
	oldPath := currentExe + ".old"
	_ = os.Remove(oldPath) // best-effort: clear out any leftover from a previous update

	if err := os.Rename(currentExe, oldPath); err != nil {
		return fmt.Errorf("could not move aside the running executable: %w", err)
	}

	if err := renameOrCopy(newExePath, currentExe); err != nil {
		_ = os.Rename(oldPath, currentExe) // best-effort restore so the app isn't left missing its exe
		return err
	}

	return nil
}
