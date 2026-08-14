package gui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/jaydenthorup/mremotego/internal/update"
)

// CheckForUpdates checks GitHub for a newer release than the running build.
// It performs network I/O, so callers should invoke it via `go w.CheckForUpdates(...)`
// rather than calling it directly on the UI goroutine.
//
// notifyIfUpToDate controls whether "you're up to date" / "check failed" are
// surfaced: true for the manual "Check for Updates..." menu action, false for
// the silent background check on startup (which should only interrupt the
// user when there's actually something to do).
func (w *MainWindow) CheckForUpdates(notifyIfUpToDate bool) {
	rel, ok, err := update.CheckLatest(w.version)

	fyne.Do(func() {
		if err != nil {
			if notifyIfUpToDate {
				dialog.ShowError(fmt.Errorf("Failed to check for updates: %w", err), w.window)
			}
			return
		}
		if !ok {
			if notifyIfUpToDate {
				dialog.ShowInformation("Up to Date", fmt.Sprintf("You're running the latest version (%s).", w.displayVersion()), w.window)
			}
			return
		}
		w.promptToInstallUpdate(rel)
	})
}

// displayVersion returns w.version, falling back to "dev" for local builds
// that weren't built with the version ldflag set.
func (w *MainWindow) displayVersion() string {
	if w.version == "" {
		return "dev"
	}
	return w.version
}

// promptToInstallUpdate shows the available release's notes and asks the user
// to confirm before downloading and installing it. Must run on the UI thread.
func (w *MainWindow) promptToInstallUpdate(rel *update.Release) {
	notes := rel.Notes
	if notes == "" {
		notes = "(no release notes provided)"
	}
	notesLabel := widget.NewLabel(notes)
	notesLabel.Wrapping = fyne.TextWrapWord

	notesScroll := container.NewVScroll(notesLabel)
	notesScroll.SetMinSize(fyne.NewSize(420, 220))

	content := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("Version %s is available. You have %s.", rel.Version, w.displayVersion())),
		widget.NewLabel("Release notes:"),
		notesScroll,
	)

	dialog.ShowCustomConfirm("Update Available", "Update Now", "Later", content, func(confirmed bool) {
		if !confirmed {
			return
		}
		w.installUpdate(rel)
	}, w.window)
}

// installUpdate downloads rel, replaces the running executable, and
// relaunches into it. Must run on the UI thread (it shows a progress dialog
// synchronously before backgrounding the actual work).
func (w *MainWindow) installUpdate(rel *update.Release) {
	progress := dialog.NewCustomWithoutButtons("Updating MremoteGO",
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf("Downloading and installing version %s...", rel.Version)),
			widget.NewProgressBarInfinite(),
		),
		w.window)
	progress.Show()

	go func() {
		err := update.Apply(rel)
		fyne.Do(func() {
			progress.Hide()
			if err != nil {
				dialog.ShowError(fmt.Errorf("Update failed: %w", err), w.window)
				return
			}
			// update.Apply already spawned the new version; hand off to it.
			w.Quit()
		})
	}()
}
