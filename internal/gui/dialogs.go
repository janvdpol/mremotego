package gui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/jaydenthorup/mremotego/pkg/models"
)

// collectAllFolders recursively collects all folders with their full paths
func (w *MainWindow) collectAllFolders(connections []*models.Connection, prefix string, folderMap map[string]*models.Connection, folderNames *[]string) {
	for _, conn := range connections {
		if conn.IsFolder() {
			var fullPath string
			if prefix == "" {
				fullPath = conn.Name
			} else {
				fullPath = prefix + " / " + conn.Name
			}
			*folderNames = append(*folderNames, fullPath)
			folderMap[fullPath] = conn

			// Recursively process children
			if len(conn.Children) > 0 {
				w.collectAllFolders(conn.Children, fullPath, folderMap, folderNames)
			}
		}
	}
}

// findConnectionParent recursively finds the parent folder and path of a connection
func (w *MainWindow) findConnectionParent(conn *models.Connection, connections []*models.Connection, prefix string) (string, *models.Connection) {
	for _, c := range connections {
		if c.IsFolder() {
			// Check direct children
			for _, child := range c.Children {
				if child == conn {
					var fullPath string
					if prefix == "" {
						fullPath = c.Name
					} else {
						fullPath = prefix + " / " + c.Name
					}
					return fullPath, c
				}
			}

			// Recursively check nested folders
			if len(c.Children) > 0 {
				var fullPath string
				if prefix == "" {
					fullPath = c.Name
				} else {
					fullPath = prefix + " / " + c.Name
				}
				if path, parent := w.findConnectionParent(conn, c.Children, fullPath); parent != nil {
					return path, parent
				}
			}
		}
	}
	return "", nil
}

// findFolderByPath finds a folder by its full path (e.g., "Dev-Ops / Infrastructure / Builders")
func (w *MainWindow) findFolderByPath(path string, connections []*models.Connection) *models.Connection {
	parts := strings.Split(path, " / ")
	current := connections

	for _, part := range parts {
		found := false
		for _, conn := range current {
			if conn.IsFolder() && conn.Name == part {
				if len(parts) == 1 {
					return conn
				}
				current = conn.Children
				parts = parts[1:]
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}
	return nil
}

// showAddConnectionDialog shows the dialog to add a new connection
func (w *MainWindow) showAddConnectionDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Connection Name")

	protocolSelect := widget.NewSelect([]string{"ssh", "rdp", "vnc", "http", "https", "telnet"}, nil)
	protocolSelect.SetSelected("ssh")

	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("hostname or IP")

	portEntry := widget.NewEntry()
	portEntry.SetPlaceHolder("port (leave empty for default)")

	usernameEntry := widget.NewEntry()
	usernameEntry.SetPlaceHolder("username")

	passwordEntry := widget.NewEntry()
	passwordEntry.SetPlaceHolder("password or op://vault/item/field")

	domainEntry := widget.NewEntry()
	domainEntry.SetPlaceHolder("domain (for RDP)")

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.SetPlaceHolder("Description")

	// Folder selection - recursively collect all folders
	folderNames := []string{"(Root)"}
	folderMap := make(map[string]*models.Connection)
	folderMap["(Root)"] = nil
	w.collectAllFolders(w.manager.GetConfig().Connections, "", folderMap, &folderNames)

	folderSelect := widget.NewSelect(folderNames, nil)
	folderSelect.SetSelected("(Root)")

	// 1Password integration - get available vaults and create display name -> ID mapping
	vaultDisplayNames := []string{}
	vaultNameToID := make(map[string]string)

	if w.launcher.GetOnePasswordProvider().IsEnabled() {
		vaults := w.launcher.GetOnePasswordProvider().GetVaults()
		for _, v := range vaults {
			// Use title if available and not encrypted, otherwise show "Vault (ID: ...)"
			displayName := v.Title
			if v.Title == "[Encrypted]" || v.Title == "" {
				displayName = fmt.Sprintf("Vault (ID: %s...)", v.ID[:8])
			}
			vaultDisplayNames = append(vaultDisplayNames, displayName)
			vaultNameToID[displayName] = v.ID
		}
	}

	if len(vaultDisplayNames) == 0 {
		vaultDisplayNames = []string{"No vaults available"}
	}

	storeTo1PasswordCheck := widget.NewCheck("Store password in 1Password", nil)
	vaultSelect := widget.NewSelect(vaultDisplayNames, nil)
	if len(vaultDisplayNames) > 0 && vaultDisplayNames[0] != "No vaults available" {
		vaultSelect.SetSelected(vaultDisplayNames[0])
	}
	vaultSelect.Hide()

	storeTo1PasswordCheck.OnChanged = func(checked bool) {
		if checked {
			// Try to refresh vault names when checkbox is enabled
			if w.launcher.GetOnePasswordProvider().IsEnabled() {
				if w.launcher.GetOnePasswordProvider().RefreshVaultNames() {
					// Vault names were decrypted, rebuild the dropdown
					vaultDisplayNames = []string{}
					vaultNameToID = make(map[string]string)
					vaults := w.launcher.GetOnePasswordProvider().GetVaults()
					for _, v := range vaults {
						displayName := v.Title
						if v.Title == "[Encrypted]" || v.Title == "" {
							displayName = fmt.Sprintf("Vault (ID: %s...)", v.ID[:8])
						}
						vaultDisplayNames = append(vaultDisplayNames, displayName)
						vaultNameToID[displayName] = v.ID
					}
					vaultSelect.Options = vaultDisplayNames
					if len(vaultDisplayNames) > 0 {
						vaultSelect.SetSelected(vaultDisplayNames[0])
					}
					vaultSelect.Refresh()
				}
			}
			vaultSelect.Show()
		} else {
			vaultSelect.Hide()
		}
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: nameEntry},
			{Text: "Protocol", Widget: protocolSelect},
			{Text: "Host", Widget: hostEntry},
			{Text: "Port", Widget: portEntry},
			{Text: "Username", Widget: usernameEntry},
			{Text: "Password", Widget: passwordEntry},
			{Text: "Domain", Widget: domainEntry},
			{Text: "Description", Widget: descriptionEntry},
			{Text: "Folder", Widget: folderSelect},
			{Text: "", Widget: storeTo1PasswordCheck},
			{Text: "Vault", Widget: vaultSelect},
		},
		OnSubmit: func() {
			conn := models.NewConnection(nameEntry.Text, models.Protocol(protocolSelect.Selected))
			conn.Host = hostEntry.Text
			conn.Username = usernameEntry.Text
			conn.Password = passwordEntry.Text
			conn.Domain = domainEntry.Text
			conn.Description = descriptionEntry.Text
			conn.Created = time.Now().Format(time.RFC3339)
			conn.Modified = conn.Created

			if portEntry.Text != "" {
				if port, err := strconv.Atoi(portEntry.Text); err == nil {
					conn.Port = port
				}
			} else {
				conn.Port = conn.Protocol.GetDefaultPort()
			}

			// If user wants to store in 1Password, create the item
			if storeTo1PasswordCheck.Checked && conn.Password != "" && !w.manager.IsOnePasswordReference(conn.Password) {
				displayName := vaultSelect.Selected
				vaultID := vaultNameToID[displayName]
				if vaultID == "" {
					vaultID = displayName // Fallback to using the display name as ID
				}
				reference, err := w.manager.CreateOnePasswordItem(vaultID, conn.Name, conn.Username, conn.Password)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to create 1Password item: %w", err), w.window)
					return
				}
				// Replace password with 1Password reference
				conn.Password = reference
				dialog.ShowInformation("Success", "Password stored in 1Password vault", w.window)
			}

			// Add to selected folder or root
			selectedFolder := folderSelect.Selected
			if selectedFolder == "(Root)" {
				w.manager.GetConfig().Connections = append(w.manager.GetConfig().Connections, conn)
			} else {
				// Find the folder and add to its children
				folder := folderMap[selectedFolder]
				if folder != nil {
					folder.Children = append(folder.Children, conn)
				}
			}

			if err := w.manager.Save(); err != nil {
				dialog.ShowError(err, w.window)
				return
			}

			w.refreshTree()
			dialog.ShowInformation("Success", "Connection added successfully", w.window)
		},
	}

	d := dialog.NewCustom("Add Connection", "Close", form, w.window)
	d.Resize(fyne.NewSize(500, 700))
	d.Show()
}

// showAddFolderDialog shows the dialog to add a new folder
func (w *MainWindow) showAddFolderDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Folder Name")

	// Folder selection - allow creating folders within folders
	folderNames := []string{"(Root)"}
	folderMap := make(map[string]*models.Connection)
	folderMap["(Root)"] = nil
	w.collectAllFolders(w.manager.GetConfig().Connections, "", folderMap, &folderNames)

	folderSelect := widget.NewSelect(folderNames, nil)
	folderSelect.SetSelected("(Root)")

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: nameEntry},
			{Text: "Parent Folder", Widget: folderSelect},
		},
		OnSubmit: func() {
			folder := models.NewFolder(nameEntry.Text)

			// Add to selected folder or root
			selectedFolder := folderSelect.Selected
			if selectedFolder == "(Root)" {
				w.manager.GetConfig().Connections = append(w.manager.GetConfig().Connections, folder)
			} else {
				// Find the folder and add to its children
				parentFolder := folderMap[selectedFolder]
				if parentFolder != nil {
					parentFolder.Children = append(parentFolder.Children, folder)
				}
			}

			if err := w.manager.Save(); err != nil {
				dialog.ShowError(err, w.window)
				return
			}

			w.refreshTree()
			dialog.ShowInformation("Success", "Folder added successfully", w.window)
		},
	}

	dialog.NewCustom("Add Folder", "Close", form, w.window).Show()
}

// showEditConnectionDialog shows the dialog to edit a connection
func (w *MainWindow) showEditConnectionDialog(conn *models.Connection) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(conn.Name)

	protocolSelect := widget.NewSelect([]string{"ssh", "rdp", "vnc", "http", "https", "telnet"}, nil)
	protocolSelect.SetSelected(string(conn.Protocol))

	hostEntry := widget.NewEntry()
	hostEntry.SetText(conn.Host)

	portEntry := widget.NewEntry()
	portEntry.SetText(strconv.Itoa(conn.Port))

	usernameEntry := widget.NewEntry()
	usernameEntry.SetText(conn.Username)

	passwordEntry := widget.NewEntry()
	passwordEntry.SetText(conn.Password)

	domainEntry := widget.NewEntry()
	domainEntry.SetText(conn.Domain)

	descriptionEntry := widget.NewMultiLineEntry()
	descriptionEntry.SetText(conn.Description)

	// Folder selection - find current parent folder using recursive search
	currentFolder := "(Root)"
	var parentFolder *models.Connection
	currentFolder, parentFolder = w.findConnectionParent(conn, w.manager.GetConfig().Connections, "")
	if currentFolder == "" {
		currentFolder = "(Root)"
		parentFolder = nil
	}

	folderNames := []string{"(Root)"}
	folderMap := make(map[string]*models.Connection)
	folderMap["(Root)"] = nil
	w.collectAllFolders(w.manager.GetConfig().Connections, "", folderMap, &folderNames)

	folderSelect := widget.NewSelect(folderNames, nil)
	folderSelect.SetSelected(currentFolder)

	// 1Password integration for edit - get available vaults and create display name -> ID mapping
	vaultDisplayNames2 := []string{}
	vaultNameToID2 := make(map[string]string)

	if w.launcher.GetOnePasswordProvider().IsEnabled() {
		vaults := w.launcher.GetOnePasswordProvider().GetVaults()
		for _, v := range vaults {
			// Use title if available and not encrypted, otherwise show "Vault (ID: ...)"
			displayName := v.Title
			if v.Title == "[Encrypted]" || v.Title == "" {
				displayName = fmt.Sprintf("Vault (ID: %s...)", v.ID[:8])
			}
			vaultDisplayNames2 = append(vaultDisplayNames2, displayName)
			vaultNameToID2[displayName] = v.ID
		}
	}

	if len(vaultDisplayNames2) == 0 {
		vaultDisplayNames2 = []string{"No vaults available"}
	}

	storeTo1PasswordCheck := widget.NewCheck("Push password to 1Password", nil)
	vaultSelect := widget.NewSelect(vaultDisplayNames2, nil)
	if len(vaultDisplayNames2) > 0 && vaultDisplayNames2[0] != "No vaults available" {
		vaultSelect.SetSelected(vaultDisplayNames2[0])
	}
	vaultSelect.Hide()

	storeTo1PasswordCheck.OnChanged = func(checked bool) {
		if checked {
			// Try to refresh vault names when checkbox is enabled
			if w.launcher.GetOnePasswordProvider().IsEnabled() {
				if w.launcher.GetOnePasswordProvider().RefreshVaultNames() {
					// Vault names were decrypted, rebuild the dropdown
					vaultDisplayNames2 = []string{}
					vaultNameToID2 = make(map[string]string)
					vaults := w.launcher.GetOnePasswordProvider().GetVaults()
					for _, v := range vaults {
						displayName := v.Title
						if v.Title == "[Encrypted]" || v.Title == "" {
							displayName = fmt.Sprintf("Vault (ID: %s...)", v.ID[:8])
						}
						vaultDisplayNames2 = append(vaultDisplayNames2, displayName)
						vaultNameToID2[displayName] = v.ID
					}
					vaultSelect.Options = vaultDisplayNames2
					if len(vaultDisplayNames2) > 0 {
						vaultSelect.SetSelected(vaultDisplayNames2[0])
					}
					vaultSelect.Refresh()
				}
			}
			vaultSelect.Show()
		} else {
			vaultSelect.Hide()
		}
	}

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: nameEntry},
			{Text: "Protocol", Widget: protocolSelect},
			{Text: "Host", Widget: hostEntry},
			{Text: "Port", Widget: portEntry},
			{Text: "Username", Widget: usernameEntry},
			{Text: "Password", Widget: passwordEntry},
			{Text: "Domain", Widget: domainEntry},
			{Text: "Description", Widget: descriptionEntry},
			{Text: "Folder", Widget: folderSelect},
			{Text: "", Widget: storeTo1PasswordCheck},
			{Text: "Vault", Widget: vaultSelect},
		},
		OnSubmit: func() {
			// If user wants to push password to 1Password
			if storeTo1PasswordCheck.Checked && passwordEntry.Text != "" && !w.manager.IsOnePasswordReference(passwordEntry.Text) {
				displayName := vaultSelect.Selected
				vaultID := vaultNameToID2[displayName]
				if vaultID == "" {
					vaultID = displayName // Fallback to using the display name as ID
				}
				reference, err := w.manager.CreateOnePasswordItem(vaultID, nameEntry.Text, usernameEntry.Text, passwordEntry.Text)
				if err != nil {
					dialog.ShowError(fmt.Errorf("Failed to create 1Password item: %w", err), w.window)
					return
				}
				// Replace password with 1Password reference
				conn.Password = reference
				dialog.ShowInformation("Success", "Password stored in 1Password vault", w.window)
			} else {
				conn.Password = passwordEntry.Text
			}

			conn.Name = nameEntry.Text
			conn.Protocol = models.Protocol(protocolSelect.Selected)
			conn.Host = hostEntry.Text
			conn.Username = usernameEntry.Text
			conn.Domain = domainEntry.Text
			conn.Description = descriptionEntry.Text
			conn.Modified = time.Now().Format(time.RFC3339)

			if port, err := strconv.Atoi(portEntry.Text); err == nil {
				conn.Port = port
			}

			// Handle folder change
			selectedFolder := folderSelect.Selected
			if selectedFolder != currentFolder {
				// Remove from old parent
				if parentFolder != nil {
					// Remove from parent's children
					for i, child := range parentFolder.Children {
						if child == conn {
							parentFolder.Children = append(parentFolder.Children[:i], parentFolder.Children[i+1:]...)
							break
						}
					}
				} else {
					// Remove from root
					for i, c := range w.manager.GetConfig().Connections {
						if c == conn {
							w.manager.GetConfig().Connections = append(w.manager.GetConfig().Connections[:i], w.manager.GetConfig().Connections[i+1:]...)
							break
						}
					}
				}

				// Add to new parent
				if selectedFolder == "(Root)" {
					w.manager.GetConfig().Connections = append(w.manager.GetConfig().Connections, conn)
				} else {
					newParent := folderMap[selectedFolder]
					if newParent != nil {
						newParent.Children = append(newParent.Children, conn)
					}
				}
			}

			if err := w.manager.Save(); err != nil {
				dialog.ShowError(err, w.window)
				return
			}

			w.refreshTree()
			w.updateDetailsPanel(conn)
			dialog.ShowInformation("Success", "Connection updated successfully", w.window)
		},
	}

	d := dialog.NewCustom("Edit Connection", "Close", form, w.window)
	d.Resize(fyne.NewSize(500, 700))
	d.Show()
}

// showEditFolderDialog shows the dialog to edit a folder
func (w *MainWindow) showEditFolderDialog(folder *models.Connection) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(folder.Name)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Name", Widget: nameEntry},
		},
		OnSubmit: func() {
			folder.Name = nameEntry.Text

			if err := w.manager.Save(); err != nil {
				dialog.ShowError(err, w.window)
				return
			}

			w.refreshTree()
			dialog.ShowInformation("Success", "Folder updated successfully", w.window)
		},
	}

	dialog.NewCustom("Edit Folder", "Close", form, w.window).Show()
}

// showSettingsDialog shows the application settings dialog
func (w *MainWindow) showSettingsDialog() {
	cfg := w.manager.GetConfig()
	if cfg == nil {
		dialog.ShowError(fmt.Errorf("no configuration loaded"), w.window)
		return
	}

	// Ensure settings struct exists - this is important for configs that don't have settings yet
	if cfg.Settings == nil {
		cfg.Settings = &models.Settings{}
	}

	// Create form fields
	accountEntry := widget.NewEntry()
	accountEntry.SetPlaceHolder("e.g., My 1Password, Work Account, company.1password.com")
	if cfg.Settings.OnePasswordAccount != "" {
		accountEntry.SetText(cfg.Settings.OnePasswordAccount)
	}

	// Instructions label
	instructions := widget.NewLabel(
		"1Password Account Name:\n" +
			"• Open 1Password desktop app\n" +
			"• Look at the top of the sidebar\n" +
			"• Enter the exact account name shown\n" +
			"• Leave empty to disable 1Password integration",
	)
	instructions.Wrapping = fyne.TextWrapWord

	// Vault mappings button
	vaultMappingsBtn := widget.NewButton("Configure Vault Names...", func() {
		w.showVaultMappingsDialog()
	})

	vaultInfo := widget.NewLabel(
		"Vault Names:\n" +
			"• Map vault IDs to friendly names\n" +
			"• Required because SDK can't decrypt vault names\n" +
			"• Click button above to configure",
	)
	vaultInfo.Wrapping = fyne.TextWrapWord

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "1Password Account", Widget: accountEntry},
			{Text: "", Widget: instructions},
			{Text: "Vault Mappings", Widget: vaultMappingsBtn},
			{Text: "", Widget: vaultInfo},
		},
		OnSubmit: func() {
			// Ensure settings struct exists before saving
			if cfg.Settings == nil {
				cfg.Settings = &models.Settings{}
			}

			// Update settings
			newAccountName := strings.TrimSpace(accountEntry.Text)
			cfg.Settings.OnePasswordAccount = newAccountName

			// Save configuration
			if err := w.manager.Save(); err != nil {
				dialog.ShowError(err, w.window)
				return
			}

			// Show restart message if account changed
			if newAccountName != "" {
				dialog.ShowInformation(
					"Settings Saved",
					"Settings saved successfully.\n\nPlease restart the application for 1Password changes to take effect.",
					w.window,
				)
			} else {
				dialog.ShowInformation("Settings Saved", "Settings saved successfully.", w.window)
			}
		},
		OnCancel: func() {
			// Dialog will be closed automatically
		},
	}

	d := dialog.NewCustom("Settings", "Close", form, w.window)
	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

// showVaultMappingsDialog shows a dialog to configure vault ID to name mappings
func (w *MainWindow) showVaultMappingsDialog() {
	if w.launcher.GetOnePasswordProvider() == nil || !w.launcher.GetOnePasswordProvider().IsEnabled() {
		dialog.ShowInformation(
			"1Password Not Available",
			"1Password SDK is not initialized. Please configure your 1Password account first.",
			w.window,
		)
		return
	}

	// Get current vaults
	vaults := w.launcher.GetOnePasswordProvider().GetVaults()
	if len(vaults) == 0 {
		dialog.ShowInformation(
			"No Vaults Found",
			"No vaults found. Make sure 1Password is unlocked.",
			w.window,
		)
		return
	}

	// Get existing mappings
	cfg := w.manager.GetConfig()
	existingMappings := make(map[string]string)
	if cfg.Settings != nil && cfg.Settings.VaultNames != nil {
		existingMappings = cfg.Settings.VaultNames
	}

	// Create entry fields for each vault
	vaultEntries := make(map[string]*widget.Entry)
	formItems := []*widget.FormItem{}

	for _, v := range vaults {
		entry := widget.NewEntry()
		entry.SetPlaceHolder("Enter friendly name")

		// Pre-fill with existing mapping or current title
		if friendlyName, ok := existingMappings[v.ID]; ok {
			entry.SetText(friendlyName)
		} else if v.Title != "" && v.Title != "[Encrypted]" {
			entry.SetText(v.Title)
		}

		vaultEntries[v.ID] = entry

		// Show vault ID in the label
		label := fmt.Sprintf("Vault ID: %s...", v.ID[:12])
		formItems = append(formItems, &widget.FormItem{
			Text:   label,
			Widget: entry,
		})
	}

	// Instructions
	instructions := widget.NewLabel(
		"Map each vault ID to a friendly name (e.g., DevOps, Employee, etc.).\n" +
			"These names will appear in the vault dropdown when creating connections.",
	)
	instructions.Wrapping = fyne.TextWrapWord

	formItems = append([]*widget.FormItem{{Text: "", Widget: instructions}}, formItems...)

	form := &widget.Form{
		Items: formItems,
		OnSubmit: func() {
			// Build new mappings
			newMappings := make(map[string]string)
			for vaultID, entry := range vaultEntries {
				name := strings.TrimSpace(entry.Text)
				if name != "" && name != "[Encrypted]" {
					newMappings[vaultID] = name
				}
			}

			// Save mappings
			if err := w.manager.SaveVaultNameMappings(newMappings); err != nil {
				dialog.ShowError(fmt.Errorf("failed to save vault mappings: %w", err), w.window)
				return
			}

			dialog.ShowInformation(
				"Vault Mappings Saved",
				fmt.Sprintf("Successfully saved mappings for %d vault(s).", len(newMappings)),
				w.window,
			)
		},
		OnCancel: func() {
			// Dialog will be closed automatically
		},
	}

	d := dialog.NewCustom("Configure Vault Names", "Close", form, w.window)
	d.Resize(fyne.NewSize(600, 400))
	d.Show()
}
