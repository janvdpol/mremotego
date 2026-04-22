package gui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/jaydenthorup/mremotego/internal/config"
	"github.com/jaydenthorup/mremotego/internal/launcher"
	"github.com/jaydenthorup/mremotego/pkg/models"
)

// MainWindow represents the main application window
type MainWindow struct {
	app            fyne.App
	window         fyne.Window
	manager        *config.Manager
	launcher       *launcher.Launcher
	tree           *widget.Tree
	detailsCard    *widget.Card
	selectedConn   *models.Connection
	connectionData map[string]*models.Connection
	searchEntry    *widget.Entry
	allConnections []*models.Connection
	filteredIDs    []string
	statusLabel    *widget.Label
	activeSessions *widget.List
	sessionList    []*models.Connection
}

// NewMainWindow creates a new main window
func NewMainWindow(app fyne.App, manager *config.Manager) *MainWindow {
	w := &MainWindow{
		app:            app,
		window:         app.NewWindow("MremoteGO - Remote Connection Manager"),
		manager:        manager,
		launcher:       launcher.NewLauncher(),
		connectionData: make(map[string]*models.Connection),
	}

	w.setupUI()
	w.setupKeyboardShortcuts()
	return w
}

// setupKeyboardShortcuts sets up keyboard shortcuts
func (w *MainWindow) setupKeyboardShortcuts() {
	// Ctrl+F for search focus
	w.window.Canvas().AddShortcut(&fyne.ShortcutCopy{}, func(shortcut fyne.Shortcut) {})

	// Enter to connect when tree is focused
	// Delete to delete selected item
}

// setupUI initializes the user interface
func (w *MainWindow) setupUI() {
	// Create menu bar
	w.setupMenuBar()

	// Create toolbar
	toolbar := w.createToolbar()

	// Create search bar
	searchBar := w.createSearchBar()

	// Create connection tree
	w.tree = w.createConnectionTree()

	// Create details panel
	detailsContainer := w.createDetailsPanel()

	// Left panel with search and tree
	leftPanel := container.NewBorder(
		searchBar, // top
		nil,       // bottom
		nil,       // left
		nil,       // right
		w.tree,    // center
	)

	// Create split container
	split := container.NewHSplit(
		leftPanel,
		detailsContainer,
	)
	split.Offset = 0.3 // 30% for tree, 70% for details

	// Create status bar
	w.statusLabel = widget.NewLabel("Ready")
	statusBar := container.NewHBox(
		w.statusLabel,
	)

	// Main content with toolbar and status bar
	content := container.NewBorder(
		toolbar,   // top
		statusBar, // bottom
		nil,       // left
		nil,       // right
		split,     // center
	)

	w.window.SetContent(content)
	w.window.Resize(fyne.NewSize(1200, 800))
	w.window.CenterOnScreen()

	// Update status with connection count
	w.updateStatus()
}

// setupMenuBar creates the menu bar
func (w *MainWindow) setupMenuBar() {
	fileMenu := fyne.NewMenu("File",
		fyne.NewMenuItem("Open Config...", func() { w.openConfig() }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("New Connection", func() { w.showAddConnectionDialog() }),
		fyne.NewMenuItem("New Folder", func() { w.showAddFolderDialog() }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Export Config", func() { w.exportConfig() }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Quit", func() { w.app.Quit() }),
	)

	connectMenu := fyne.NewMenu("Connection",
		fyne.NewMenuItem("Connect", func() { w.connectToSelected() }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Edit", func() { w.editSelected() }),
		fyne.NewMenuItem("Delete", func() { w.deleteSelected() }),
	)

	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() { w.showAbout() }),
	)

	mainMenu := fyne.NewMainMenu(fileMenu, connectMenu, helpMenu)
	w.window.SetMainMenu(mainMenu)
}

// createSearchBar creates the search bar
func (w *MainWindow) createSearchBar() *fyne.Container {
	w.searchEntry = widget.NewEntry()
	w.searchEntry.SetPlaceHolder("Search connections...")

	w.searchEntry.OnChanged = func(query string) {
		w.filterConnections(query)
	}

	clearBtn := widget.NewButtonWithIcon("", theme.CancelIcon(), func() {
		w.searchEntry.SetText("")
	})

	return container.NewBorder(nil, nil, nil, clearBtn, w.searchEntry)
}

// createToolbar creates the toolbar with action buttons
func (w *MainWindow) createToolbar() *widget.Toolbar {
	return widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			w.showAddConnectionDialog()
		}),
		widget.NewToolbarAction(theme.FolderNewIcon(), func() {
			w.showAddFolderDialog()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.MediaPlayIcon(), func() {
			w.connectToSelected()
		}),
		widget.NewToolbarAction(theme.DocumentCreateIcon(), func() {
			w.editSelected()
		}),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			w.deleteSelected()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.ViewRefreshIcon(), func() {
			w.refreshTree()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.SearchIcon(), func() {
			w.window.Canvas().Focus(w.searchEntry)
		}),
	)
}

// createConnectionTree creates the tree widget for connections
func (w *MainWindow) createConnectionTree() *widget.Tree {
	w.buildConnectionMap()

	tree := widget.NewTree(
		// ChildUIDs
		func(uid string) []string {
			if uid == "" {
				// Root level
				// If filtering, show only filtered connections
				if len(w.filteredIDs) > 0 {
					return w.filteredIDs
				}

				var folders []string
				var connections []string
				for _, conn := range w.manager.GetConfig().Connections {
					id := w.getConnectionID(conn)
					if conn.IsFolder() {
						folders = append(folders, id)
					} else {
						connections = append(connections, id)
					}
				}
				// Return folders first, then connections
				return append(folders, connections...)
			}

			conn, exists := w.connectionData[uid]
			if !exists || !conn.IsFolder() {
				return []string{}
			}

			var ids []string
			for _, child := range conn.Children {
				ids = append(ids, w.getConnectionID(child))
			}
			return ids
		},
		// IsBranch
		func(uid string) bool {
			if uid == "" {
				return true
			}
			conn, exists := w.connectionData[uid]
			return exists && conn.IsFolder()
		},
		// Create
		func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		// Update
		func(uid string, branch bool, obj fyne.CanvasObject) {
			conn, exists := w.connectionData[uid]
			if !exists {
				return
			}

			label := obj.(*widget.Label)
			icon := w.getConnectionIcon(conn)
			label.SetText(icon + " " + conn.Name)
		},
	)

	tree.OnSelected = func(uid string) {
		if conn, exists := w.connectionData[uid]; exists {
			w.selectedConn = conn
			w.updateDetailsPanel(conn)
		}
	}

	return tree
}

// filterConnections filters the connection list based on search query
func (w *MainWindow) filterConnections(query string) {
	if query == "" {
		w.filteredIDs = nil
		w.tree.Refresh()
		w.updateStatus()
		return
	}

	query = strings.ToLower(query)
	w.filteredIDs = []string{}

	for id, conn := range w.connectionData {
		if !conn.IsFolder() {
			if strings.Contains(strings.ToLower(conn.Name), query) ||
				strings.Contains(strings.ToLower(conn.Host), query) ||
				strings.Contains(strings.ToLower(conn.Username), query) ||
				strings.Contains(strings.ToLower(string(conn.Protocol)), query) ||
				strings.Contains(strings.ToLower(conn.Description), query) {
				w.filteredIDs = append(w.filteredIDs, id)
			}
		}
	}

	w.tree.Refresh()
	w.updateStatus()
}

// updateStatus updates the status bar with current information
func (w *MainWindow) updateStatus() {
	totalConns := len(w.manager.ListConnections())
	configPath := w.manager.GetConfigPath()

	if len(w.filteredIDs) > 0 {
		w.statusLabel.SetText(fmt.Sprintf("Showing %d of %d connections | %s", len(w.filteredIDs), totalConns, configPath))
	} else {
		w.statusLabel.SetText(fmt.Sprintf("%d connections | %s", totalConns, configPath))
	}
}

// createDetailsPanel creates the details panel on the right
func (w *MainWindow) createDetailsPanel() *fyne.Container {
	w.detailsCard = widget.NewCard("Connection Details", "", widget.NewLabel("Select a connection to view details"))

	return container.NewBorder(
		nil, nil, nil, nil,
		container.NewVScroll(w.detailsCard),
	)
}

// updateDetailsPanel updates the details panel with connection info
func (w *MainWindow) updateDetailsPanel(conn *models.Connection) {
	if conn.IsFolder() {
		childCount := len(conn.Children)
		w.detailsCard.SetTitle("📁 " + conn.Name)
		w.detailsCard.SetSubTitle("Folder")
		w.detailsCard.SetContent(widget.NewLabel(fmt.Sprintf("Contains %d item(s)", childCount)))
		return
	}

	icon := w.getConnectionIcon(conn)
	w.detailsCard.SetTitle(icon + " " + conn.Name)
	w.detailsCard.SetSubTitle(string(conn.Protocol))

	details := container.NewVBox(
		widget.NewLabel("Host: "+conn.Host),
		widget.NewLabel(fmt.Sprintf("Port: %d", conn.Port)),
	)

	if conn.Username != "" {
		details.Add(widget.NewLabel("Username: " + conn.Username))
	}

	if conn.Domain != "" {
		details.Add(widget.NewLabel("Domain: " + conn.Domain))
	}

	if conn.Description != "" {
		details.Add(widget.NewLabel(""))
		details.Add(widget.NewLabel("Description:"))
		details.Add(widget.NewLabel(conn.Description))
	}

	if len(conn.Tags) > 0 {
		details.Add(widget.NewLabel(""))
		tags := "Tags: "
		for i, tag := range conn.Tags {
			if i > 0 {
				tags += ", "
			}
			tags += tag
		}
		details.Add(widget.NewLabel(tags))
	}

	// Add action buttons
	details.Add(widget.NewLabel(""))

	statusContainer := container.NewVBox()
	var connectBtn *widget.Button

	connectBtn = widget.NewButton("🚀 Connect", func() {
		connectBtn.Disable()
		statusContainer.RemoveAll()
		statusContainer.Add(widget.NewProgressBarInfinite())
		statusContainer.Refresh()

		go func() {
			err := w.launcher.Launch(conn)
			w.window.Canvas().Content().Refresh()

			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to launch connection: %w", err), w.window)
			}

			connectBtn.Enable()
			statusContainer.RemoveAll()
			statusContainer.Refresh()
		}()
	})
	connectBtn.Importance = widget.HighImportance

	editBtn := widget.NewButton("✏️ Edit", func() {
		w.showEditConnectionDialog(conn)
	})

	deleteBtn := widget.NewButton("🗑️ Delete", func() {
		w.deleteSelected()
	})
	deleteBtn.Importance = widget.DangerImportance

	actionButtons := container.NewGridWithColumns(3, connectBtn, editBtn, deleteBtn)
	details.Add(actionButtons)
	details.Add(statusContainer)

	w.detailsCard.SetContent(details)
}

// Helper functions
func (w *MainWindow) buildConnectionMap() {
	w.connectionData = make(map[string]*models.Connection)
	w.buildConnectionMapRecursive(w.manager.GetConfig().Connections, "")
}

func (w *MainWindow) buildConnectionMapRecursive(connections []*models.Connection, prefix string) {
	for i, conn := range connections {
		id := fmt.Sprintf("%s%d", prefix, i)
		w.connectionData[id] = conn

		if conn.IsFolder() {
			w.buildConnectionMapRecursive(conn.Children, id+"_")
		}
	}
}

func (w *MainWindow) getConnectionID(conn *models.Connection) string {
	for id, c := range w.connectionData {
		if c == conn {
			return id
		}
	}
	return ""
}

func (w *MainWindow) getConnectionIcon(conn *models.Connection) string {
	if conn.IsFolder() {
		return "📁"
	}

	switch conn.Protocol {
	case models.ProtocolSSH:
		return "🔐"
	case models.ProtocolRDP:
		return "🖥️"
	case models.ProtocolVNC:
		return "📺"
	case models.ProtocolHTTP, models.ProtocolHTTPS:
		return "🌐"
	case models.ProtocolTelnet:
		return "📟"
	default:
		return "🔌"
	}
}

// Action handlers
func (w *MainWindow) connectToSelected() {
	if w.selectedConn == nil {
		dialog.ShowInformation("No Selection", "Please select a connection first", w.window)
		return
	}

	if w.selectedConn.IsFolder() {
		dialog.ShowInformation("Cannot Connect", "Cannot connect to a folder", w.window)
		return
	}

	w.connectToConnection(w.selectedConn)
}

func (w *MainWindow) connectToConnection(conn *models.Connection) {
	if err := w.launcher.Launch(conn); err != nil {
		dialog.ShowError(fmt.Errorf("Failed to launch connection: %w", err), w.window)
	}
}

func (w *MainWindow) editSelected() {
	if w.selectedConn == nil {
		dialog.ShowInformation("No Selection", "Please select a connection first", w.window)
		return
	}

	if w.selectedConn.IsFolder() {
		w.showEditFolderDialog(w.selectedConn)
	} else {
		w.showEditConnectionDialog(w.selectedConn)
	}
}

func (w *MainWindow) deleteSelected() {
	if w.selectedConn == nil {
		dialog.ShowInformation("No Selection", "Please select a connection first", w.window)
		return
	}

	dialog.ShowConfirm("Delete Confirmation",
		fmt.Sprintf("Are you sure you want to delete '%s'?", w.selectedConn.Name),
		func(confirmed bool) {
			if confirmed {
				// Clean up Windows credentials for RDP connections
				if w.selectedConn.Protocol == models.ProtocolRDP {
					if err := w.launcher.RemoveWindowsCredential(w.selectedConn); err != nil {
						// Just log the error, don't block deletion
						fmt.Printf("Warning: Failed to remove credentials: %v\n", err)
					}
				}

				if err := w.manager.DeleteConnection(w.selectedConn.Name); err != nil {
					dialog.ShowError(err, w.window)
					return
				}

				if err := w.manager.Save(); err != nil {
					dialog.ShowError(err, w.window)
					return
				}

				w.refreshTree()
				dialog.ShowInformation("Success", "Connection deleted", w.window)
			}
		}, w.window)
}

func (w *MainWindow) refreshTree() {
	if err := w.manager.Load(); err != nil {
		dialog.ShowError(err, w.window)
		return
	}

	w.buildConnectionMap()
	w.tree.Refresh()
	w.selectedConn = nil
	w.detailsCard.SetContent(widget.NewLabel("Select a connection to view details"))
	w.updateStatus()
}

func (w *MainWindow) openConfig() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, w.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		// Get the file path
		filePath := reader.URI().Path()

		// Create a new manager with the selected file
		newManager := config.NewManager(filePath)
		if err := newManager.Load(); err != nil {
			dialog.ShowError(fmt.Errorf("failed to load config: %w", err), w.window)
			return
		}

		// Replace the current manager and reload the tree
		w.manager = newManager
		w.refreshTree()

		dialog.ShowInformation("Config Loaded", fmt.Sprintf("Loaded config from:\n%s", filePath), w.window)
	}, w.window)

	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
	fileDialog.Show()
}

func (w *MainWindow) exportConfig() {
	dialog.ShowInformation("Export", "Export functionality - save config to file", w.window)
}

func (w *MainWindow) showAbout() {
	dialog.ShowInformation("About MremoteGO",
		"MremoteGO v1.0\n\nA Git-compatible remote connection manager\n\n"+
			"Built with Go and Fyne\n\n"+
			"© 2026 Your Name",
		w.window)
}

// Show displays the main window
func (w *MainWindow) Show() {
	w.window.Show()

	// Check 1Password CLI authentication status after window is shown
	w.check1PasswordAuth()
}

// check1PasswordAuth checks if 1Password CLI needs authentication
func (w *MainWindow) check1PasswordAuth() {
	// Check if there are any 1Password references in use
	hasOpReferences := false
	var checkConnections func([]*models.Connection)
	checkConnections = func(conns []*models.Connection) {
		for _, conn := range conns {
			if conn.IsFolder() {
				checkConnections(conn.Children)
			} else if strings.HasPrefix(conn.Password, "op://") {
				hasOpReferences = true
				return
			}
		}
	}
	checkConnections(w.manager.GetConfig().Connections)

	// Only check if there are actually 1Password references in use
	if !hasOpReferences {
		return
	}

	// Only verify that the CLI is present in PATH. Running `op whoami` to
	// check the live authentication status spawns a visible console window
	// on Windows for ~2s at startup. Authentication errors surface when a
	// reference is actually resolved in the launcher.
	opProvider := w.launcher.GetOnePasswordProvider()
	if !opProvider.IsEnabled() {
		content := widget.NewLabel("Your connections use 1Password references (op://...) but the 1Password CLI (op) was not found in PATH. Install it and ensure it is on your PATH.")
		content.Wrapping = fyne.TextWrapWord

		scrollContainer := container.NewVScroll(content)
		scrollContainer.SetMinSize(fyne.NewSize(600, 200))

		dialog.ShowCustom(
			"1Password CLI Not Installed",
			"OK",
			scrollContainer,
			w.window,
		)
	}
} // Reload refreshes the window with the loaded config
func (w *MainWindow) Reload() {
	w.buildConnectionMap()
	w.tree.Refresh()
	w.updateStatus()
}
