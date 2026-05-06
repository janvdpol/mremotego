package main

import (
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/jaydenthorup/mremotego/cmd/mremotego/cmd"
	"github.com/jaydenthorup/mremotego/internal/config"
	"github.com/jaydenthorup/mremotego/internal/gui"
	"github.com/jaydenthorup/mremotego/internal/launcher"
)

func main() {
	// If command-line arguments are provided (other than just the program name),
	// run in CLI mode
	if len(os.Args) > 1 {
		cmd.Execute()
		return
	}

	// Otherwise, launch the GUI
	runGUI()
}

func runGUI() {
	launcher.CleanupOnStartup()

	// Set locale to avoid warnings on non-standard systems
	if os.Getenv("LANG") == "" || os.Getenv("LANG") == "C" {
		os.Setenv("LANG", "en_US.UTF-8")
	}

	// Create Fyne application
	myApp := app.NewWithID("com.mremotego.app")
	myApp.Settings().SetTheme(&customTheme{})

	// Set application icon (ignore errors - icon is optional)
	if icon := gui.GetAppIcon(); icon != nil {
		myApp.SetIcon(icon)
	}

	// Get config path
	cfgPath, err := config.GetDefaultConfigPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting config path: %v\n", err)
		os.Exit(1)
	}

	// Create config manager
	manager := config.NewManager(cfgPath)

	// Create the main window first but don't show it yet
	mainWindow := gui.NewMainWindow(myApp, manager)

	// Check if config exists and if it has encrypted passwords
	_, statErr := os.Stat(cfgPath)
	configExists := statErr == nil

	skipPasswordDialog := false
	if configExists {
		// Try to read the config file and check for encrypted passwords
		data, err := os.ReadFile(cfgPath)
		if err == nil {
			// Check if file contains "enc:" prefix (encrypted passwords)
			skipPasswordDialog = !strings.Contains(string(data), "enc:")
		}
	}

	if skipPasswordDialog {
		// No encrypted passwords, load without password
		if err := manager.Load(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		}
		mainWindow.Reload()
		mainWindow.Show()
		myApp.Run()
	} else {
		// Show password dialog
		showPasswordDialog(myApp, manager, mainWindow)
		myApp.Run()
	}
}

func showPasswordDialog(myApp fyne.App, manager *config.Manager, mainWindow *gui.MainWindow) {
	w := myApp.NewWindow("MremoteGO - Master Password")
	w.Resize(fyne.NewSize(400, 200))

	// Check if config file exists
	_, err := os.Stat(manager.GetConfigPath())
	configExists := err == nil

	var message string
	if configExists {
		message = "Enter your master password to unlock the configuration.\n(Leave blank if no encryption is used)"
	} else {
		message = "Set a master password to encrypt passwords in the configuration.\n(Leave blank to store passwords unencrypted)"
	}

	// Create password entry
	passwordEntry := widget.NewPasswordEntry()
	passwordEntry.SetPlaceHolder("Master password (optional)")

	// Create label
	label := widget.NewLabel(message)
	label.Wrapping = fyne.TextWrapWord

	// Create buttons
	okButton := widget.NewButton("OK", func() {
		password := passwordEntry.Text

		// Set the master password (even if empty)
		manager.SetMasterPassword(password)

		// Try to load config
		if err := manager.Load(); err != nil {
			if configExists {
				// If config exists but fails to load, might be wrong password
				dialog.ShowError(fmt.Errorf("Failed to load config (wrong password?): %w", err), w)
				return
			}
			// If config doesn't exist, that's okay
		}

		// Reload the main window with the loaded config
		mainWindow.Reload()

		// Close the password window and show main window
		w.Close()
		mainWindow.Show()
	})
	okButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Cancel", func() {
		myApp.Quit()
	})

	// Handle Enter key in password field
	passwordEntry.OnSubmitted = func(text string) {
		okButton.OnTapped()
	}

	// Layout
	content := container.NewVBox(
		label,
		passwordEntry,
		container.NewHBox(okButton, cancelButton),
	)

	w.SetContent(content)
	w.CenterOnScreen()
	w.Show()
}
