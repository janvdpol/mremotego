package gui

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/jaydenthorup/mremotego/pkg/models"
)

// stableFormLayout is a two-column form layout with a fixed label column width,
// so field widths stay constant regardless of which rows are shown or hidden.
type stableFormLayout struct{ labelWidth float32 }

func (l *stableFormLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	pad := theme.Padding()
	fieldX := l.labelWidth + pad
	fieldW := size.Width - fieldX
	y := float32(0)
	first := true
	for i := 0; i+1 < len(objects); i += 2 {
		label, field := objects[i], objects[i+1]
		if !label.Visible() && !field.Visible() {
			continue
		}
		if !first {
			y += pad
		}
		first = false
		rowH := fyne.Max(label.MinSize().Height, field.MinSize().Height)
		label.Move(fyne.NewPos(0, y))
		label.Resize(fyne.NewSize(l.labelWidth, rowH))
		field.Move(fyne.NewPos(fieldX, y))
		field.Resize(fyne.NewSize(fieldW, rowH))
		y += rowH
	}
}

func (l *stableFormLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	pad := theme.Padding()
	minH := float32(0)
	first := true
	for i := 0; i+1 < len(objects); i += 2 {
		if !objects[i].Visible() && !objects[i+1].Visible() {
			continue
		}
		if !first {
			minH += pad
		}
		first = false
		minH += fyne.Max(objects[i].MinSize().Height, objects[i+1].MinSize().Height)
	}
	return fyne.NewSize(l.labelWidth+pad, minH)
}

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

	// RDP Gateway widgets
	useGatewayCheck := widget.NewCheck("Use RDP Gateway", nil)

	gatewayUsageSelect := widget.NewSelect([]string{"Never", "Always", "Detect"}, nil)
	gatewayUsageSelect.SetSelected("Always")
	gatewayHostnameEntry := widget.NewEntry()
	gatewayHostnameEntry.SetPlaceHolder("gateway.example.com")
	gatewayCredentialsSelect := widget.NewSelect([]string{
		"Use the same username and password",
		"Use a different username and password",
		"Use a smart card",
	}, nil)
	gatewayCredentialsSelect.SetSelected("Use the same username and password")
	gatewayUsernameEntry := widget.NewEntry()
	gatewayUsernameEntry.SetPlaceHolder("username or op://vault/item/field")
	gatewayPasswordEntry := widget.NewEntry()
	gatewayPasswordEntry.SetPlaceHolder("password or op://vault/item/field")
	gatewayDomainEntry := widget.NewEntry()
	gatewayDomainEntry.SetPlaceHolder("gateway domain")

	gatewayUsageLabel := widget.NewLabelWithStyle("Gateway Usage", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayHostnameLabel := widget.NewLabelWithStyle("Gateway Hostname", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayCredentialsLabel := widget.NewLabelWithStyle("Gateway Credentials", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayUsernameLabel := widget.NewLabelWithStyle("Gateway Username", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayPasswordLabel := widget.NewLabelWithStyle("Gateway Password", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayDomainLabel := widget.NewLabelWithStyle("Gateway Domain", fyne.TextAlignTrailing, fyne.TextStyle{})

	// All gateway rows start hidden
	gatewayUsageLabel.Hide()
	gatewayUsageSelect.Hide()
	gatewayHostnameLabel.Hide()
	gatewayHostnameEntry.Hide()
	gatewayCredentialsLabel.Hide()
	gatewayCredentialsSelect.Hide()
	gatewayUsernameLabel.Hide()
	gatewayUsernameEntry.Hide()
	gatewayPasswordLabel.Hide()
	gatewayPasswordEntry.Hide()
	gatewayDomainLabel.Hide()
	gatewayDomainEntry.Hide()

	// 1Password integration
	storeTo1PasswordCheck := widget.NewCheck("Store password in 1Password", nil)
	vaultSelect := widget.NewSelect([]string{"DevOps", "Private", "Employee"}, nil)
	vaultSelect.SetSelected("DevOps")
	vaultLabel := widget.NewLabelWithStyle("Vault", fyne.TextAlignTrailing, fyne.TextStyle{})
	vaultLabel.Hide()
	vaultSelect.Hide()

	gatewayCheckLabel := widget.NewLabel("")
	gatewayCheckLabel.Hide()
	useGatewayCheck.Hide()

	var formContainer *fyne.Container

	hideAllGatewayRows := func() {
		gatewayCheckLabel.Hide(); useGatewayCheck.Hide()
		gatewayUsageLabel.Hide(); gatewayUsageSelect.Hide()
		gatewayHostnameLabel.Hide(); gatewayHostnameEntry.Hide()
		gatewayCredentialsLabel.Hide(); gatewayCredentialsSelect.Hide()
		gatewayUsernameLabel.Hide(); gatewayUsernameEntry.Hide()
		gatewayPasswordLabel.Hide(); gatewayPasswordEntry.Hide()
		gatewayDomainLabel.Hide(); gatewayDomainEntry.Hide()
	}

	storeTo1PasswordCheck.OnChanged = func(checked bool) {
		if checked {
			vaultLabel.Show()
			vaultSelect.Show()
		} else {
			vaultLabel.Hide()
			vaultSelect.Hide()
		}
		formContainer.Refresh()
	}

	showGatewayDifferentCreds := func(show bool) {
		if show {
			gatewayUsernameLabel.Show()
			gatewayUsernameEntry.Show()
			gatewayPasswordLabel.Show()
			gatewayPasswordEntry.Show()
			gatewayDomainLabel.Show()
			gatewayDomainEntry.Show()
		} else {
			gatewayUsernameLabel.Hide()
			gatewayUsernameEntry.Hide()
			gatewayPasswordLabel.Hide()
			gatewayPasswordEntry.Hide()
			gatewayDomainLabel.Hide()
			gatewayDomainEntry.Hide()
		}
		formContainer.Refresh()
	}

	gatewayCredentialsSelect.OnChanged = func(selected string) {
		showGatewayDifferentCreds(selected == "Use a different username and password")
	}

	useGatewayCheck.OnChanged = func(checked bool) {
		if checked {
			gatewayUsageLabel.Show(); gatewayUsageSelect.Show()
			gatewayHostnameLabel.Show(); gatewayHostnameEntry.Show()
			gatewayCredentialsLabel.Show(); gatewayCredentialsSelect.Show()
			showGatewayDifferentCreds(gatewayCredentialsSelect.Selected == "Use a different username and password")
		} else {
			gatewayUsageLabel.Hide(); gatewayUsageSelect.Hide()
			gatewayHostnameLabel.Hide(); gatewayHostnameEntry.Hide()
			gatewayCredentialsLabel.Hide(); gatewayCredentialsSelect.Hide()
			gatewayUsernameLabel.Hide(); gatewayUsernameEntry.Hide()
			gatewayPasswordLabel.Hide(); gatewayPasswordEntry.Hide()
			gatewayDomainLabel.Hide(); gatewayDomainEntry.Hide()
		}
		formContainer.Refresh()
	}

	protocolSelect.OnChanged = func(selected string) {
		if selected == "rdp" {
			gatewayCheckLabel.Show()
			useGatewayCheck.Show()
		} else {
			hideAllGatewayRows()
			useGatewayCheck.SetChecked(false)
		}
		formContainer.Refresh()
	}

	labelMinWidth := widget.NewLabelWithStyle("Gateway Credentials", fyne.TextAlignTrailing, fyne.TextStyle{}).MinSize().Width
	formContainer = container.New(&stableFormLayout{labelWidth: labelMinWidth},
		widget.NewLabelWithStyle("Name", fyne.TextAlignTrailing, fyne.TextStyle{}), nameEntry,
		widget.NewLabelWithStyle("Protocol", fyne.TextAlignTrailing, fyne.TextStyle{}), protocolSelect,
		widget.NewLabelWithStyle("Host", fyne.TextAlignTrailing, fyne.TextStyle{}), hostEntry,
		widget.NewLabelWithStyle("Port", fyne.TextAlignTrailing, fyne.TextStyle{}), portEntry,
		widget.NewLabelWithStyle("Username", fyne.TextAlignTrailing, fyne.TextStyle{}), usernameEntry,
		widget.NewLabelWithStyle("Password", fyne.TextAlignTrailing, fyne.TextStyle{}), passwordEntry,
		widget.NewLabelWithStyle("Domain", fyne.TextAlignTrailing, fyne.TextStyle{}), domainEntry,
		widget.NewLabelWithStyle("Description", fyne.TextAlignTrailing, fyne.TextStyle{}), descriptionEntry,
		widget.NewLabelWithStyle("Folder", fyne.TextAlignTrailing, fyne.TextStyle{}), folderSelect,
		gatewayCheckLabel, useGatewayCheck,
		gatewayUsageLabel, gatewayUsageSelect,
		gatewayHostnameLabel, gatewayHostnameEntry,
		gatewayCredentialsLabel, gatewayCredentialsSelect,
		gatewayUsernameLabel, gatewayUsernameEntry,
		gatewayPasswordLabel, gatewayPasswordEntry,
		gatewayDomainLabel, gatewayDomainEntry,
		widget.NewLabel(""), storeTo1PasswordCheck,
		vaultLabel, vaultSelect,
	)

	scroll := container.NewScroll(container.NewPadded(formContainer))
	scroll.SetMinSize(fyne.NewSize(500, 500))
	d := dialog.NewCustomConfirm("Add Connection", "Submit", "Close", scroll, func(confirmed bool) {
		if !confirmed {
			return
		}

		conn := models.NewConnection(nameEntry.Text, models.Protocol(protocolSelect.Selected))
		conn.Host = hostEntry.Text
		conn.Username = usernameEntry.Text
		conn.Password = passwordEntry.Text
		conn.Domain = domainEntry.Text
		conn.Description = descriptionEntry.Text
		conn.Created = time.Now().Format(time.RFC3339)
		conn.Modified = conn.Created

		conn.UseGateway = useGatewayCheck.Checked
		if conn.UseGateway {
			conn.GatewayHostname = gatewayHostnameEntry.Text
			switch gatewayUsageSelect.Selected {
			case "Never":
				conn.GatewayUsageMethod = "never"
			case "Detect":
				conn.GatewayUsageMethod = "detect"
			default:
				conn.GatewayUsageMethod = "always"
			}
			switch gatewayCredentialsSelect.Selected {
			case "Use a different username and password":
				conn.GatewayCredentials = "different"
				conn.GatewayDomain = gatewayDomainEntry.Text
				gwUser := gatewayUsernameEntry.Text
				gwPass := gatewayPasswordEntry.Text
				if storeTo1PasswordCheck.Checked && gwPass != "" && !w.manager.IsOnePasswordReference(gwPass) {
					vault := vaultSelect.Selected
					gwTitle := nameEntry.Text + " Gateway"
					ref, err := w.manager.CreateOnePasswordItem(vault, gwTitle, gwUser, gwPass)
					if err != nil {
						dialog.ShowError(fmt.Errorf("Failed to create 1Password item for gateway: %w", err), w.window)
						return
					}
					conn.GatewayPassword = ref
					conn.GatewayUsername = fmt.Sprintf("op://%s/%s/username", vault, url.PathEscape(gwTitle))
				} else {
					conn.GatewayUsername = gwUser
					conn.GatewayPassword = gwPass
				}
			case "Use a smart card":
				conn.GatewayCredentials = "smartcard"
			default:
				conn.GatewayCredentials = "same"
			}
		}

		if portEntry.Text != "" {
			if port, err := strconv.Atoi(portEntry.Text); err == nil {
				conn.Port = port
			}
		} else {
			conn.Port = conn.Protocol.GetDefaultPort()
		}

		// If user wants to store in 1Password, create the item
		if storeTo1PasswordCheck.Checked && conn.Password != "" && !w.manager.IsOnePasswordReference(conn.Password) {
			vault := vaultSelect.Selected
			reference, err := w.manager.CreateOnePasswordItem(vault, conn.Name, conn.Username, conn.Password)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to create 1Password item: %w", err), w.window)
				return
			}
			conn.Password = reference
		}

		// Add to selected folder or root
		selectedFolder := folderSelect.Selected
		if selectedFolder == "(Root)" {
			w.manager.GetConfig().Connections = append(w.manager.GetConfig().Connections, conn)
		} else {
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
	}, w.window)
	d.Resize(fyne.NewSize(620, 600))
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

	// RDP Gateway widgets
	useGatewayCheck := widget.NewCheck("Use RDP Gateway", nil)
	useGatewayCheck.SetChecked(conn.UseGateway)

	gatewayUsageSelect := widget.NewSelect([]string{"Never", "Always", "Detect"}, nil)
	switch conn.GatewayUsageMethod {
	case "never":
		gatewayUsageSelect.SetSelected("Never")
	case "detect":
		gatewayUsageSelect.SetSelected("Detect")
	default:
		gatewayUsageSelect.SetSelected("Always")
	}

	gatewayHostnameEntry := widget.NewEntry()
	gatewayHostnameEntry.SetPlaceHolder("gateway.example.com")
	gatewayHostnameEntry.SetText(conn.GatewayHostname)

	gatewayCredentialsSelect := widget.NewSelect([]string{
		"Use the same username and password",
		"Use a different username and password",
		"Use a smart card",
	}, nil)
	switch conn.GatewayCredentials {
	case "different":
		gatewayCredentialsSelect.SetSelected("Use a different username and password")
	case "smartcard":
		gatewayCredentialsSelect.SetSelected("Use a smart card")
	default:
		gatewayCredentialsSelect.SetSelected("Use the same username and password")
	}

	gatewayUsernameEntry := widget.NewEntry()
	gatewayUsernameEntry.SetPlaceHolder("username or op://vault/item/field")
	gatewayUsernameEntry.SetText(conn.GatewayUsername)

	gatewayPasswordEntry := widget.NewEntry()
	gatewayPasswordEntry.SetPlaceHolder("password or op://vault/item/field")
	gatewayPasswordEntry.SetText(conn.GatewayPassword)

	gatewayDomainEntry := widget.NewEntry()
	gatewayDomainEntry.SetPlaceHolder("gateway domain")
	gatewayDomainEntry.SetText(conn.GatewayDomain)

	gatewayUsageLabel := widget.NewLabelWithStyle("Gateway Usage", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayHostnameLabel := widget.NewLabelWithStyle("Gateway Hostname", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayCredentialsLabel := widget.NewLabelWithStyle("Gateway Credentials", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayUsernameLabel := widget.NewLabelWithStyle("Gateway Username", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayPasswordLabel := widget.NewLabelWithStyle("Gateway Password", fyne.TextAlignTrailing, fyne.TextStyle{})
	gatewayDomainLabel := widget.NewLabelWithStyle("Gateway Domain", fyne.TextAlignTrailing, fyne.TextStyle{})

	// Set initial gateway row visibility
	if !conn.UseGateway {
		gatewayUsageLabel.Hide()
		gatewayUsageSelect.Hide()
		gatewayHostnameLabel.Hide()
		gatewayHostnameEntry.Hide()
		gatewayCredentialsLabel.Hide()
		gatewayCredentialsSelect.Hide()
		gatewayUsernameLabel.Hide()
		gatewayUsernameEntry.Hide()
		gatewayPasswordLabel.Hide()
		gatewayPasswordEntry.Hide()
		gatewayDomainLabel.Hide()
		gatewayDomainEntry.Hide()
	} else if conn.GatewayCredentials != "different" {
		gatewayUsernameLabel.Hide()
		gatewayUsernameEntry.Hide()
		gatewayPasswordLabel.Hide()
		gatewayPasswordEntry.Hide()
		gatewayDomainLabel.Hide()
		gatewayDomainEntry.Hide()
	}

	// 1Password integration for edit
	storeTo1PasswordCheck := widget.NewCheck("Push password to 1Password", nil)
	vaultSelect := widget.NewSelect([]string{"DevOps", "Private", "Employee"}, nil)
	vaultSelect.SetSelected("DevOps")
	vaultLabel := widget.NewLabelWithStyle("Vault", fyne.TextAlignTrailing, fyne.TextStyle{})
	vaultLabel.Hide()
	vaultSelect.Hide()

	// Show gateway checkbox only when protocol is RDP
	gatewayCheckLabel := widget.NewLabel("")
	if conn.Protocol != "rdp" {
		gatewayCheckLabel.Hide()
		useGatewayCheck.Hide()
	}

	var formContainer *fyne.Container

	hideAllGatewayRows := func() {
		gatewayCheckLabel.Hide(); useGatewayCheck.Hide()
		gatewayUsageLabel.Hide(); gatewayUsageSelect.Hide()
		gatewayHostnameLabel.Hide(); gatewayHostnameEntry.Hide()
		gatewayCredentialsLabel.Hide(); gatewayCredentialsSelect.Hide()
		gatewayUsernameLabel.Hide(); gatewayUsernameEntry.Hide()
		gatewayPasswordLabel.Hide(); gatewayPasswordEntry.Hide()
		gatewayDomainLabel.Hide(); gatewayDomainEntry.Hide()
	}

	storeTo1PasswordCheck.OnChanged = func(checked bool) {
		if checked {
			vaultLabel.Show()
			vaultSelect.Show()
		} else {
			vaultLabel.Hide()
			vaultSelect.Hide()
		}
		formContainer.Refresh()
	}

	showGatewayDifferentCreds := func(show bool) {
		if show {
			gatewayUsernameLabel.Show()
			gatewayUsernameEntry.Show()
			gatewayPasswordLabel.Show()
			gatewayPasswordEntry.Show()
			gatewayDomainLabel.Show()
			gatewayDomainEntry.Show()
		} else {
			gatewayUsernameLabel.Hide()
			gatewayUsernameEntry.Hide()
			gatewayPasswordLabel.Hide()
			gatewayPasswordEntry.Hide()
			gatewayDomainLabel.Hide()
			gatewayDomainEntry.Hide()
		}
		formContainer.Refresh()
	}

	gatewayCredentialsSelect.OnChanged = func(selected string) {
		showGatewayDifferentCreds(selected == "Use a different username and password")
	}

	useGatewayCheck.OnChanged = func(checked bool) {
		if checked {
			gatewayUsageLabel.Show(); gatewayUsageSelect.Show()
			gatewayHostnameLabel.Show(); gatewayHostnameEntry.Show()
			gatewayCredentialsLabel.Show(); gatewayCredentialsSelect.Show()
			showGatewayDifferentCreds(gatewayCredentialsSelect.Selected == "Use a different username and password")
		} else {
			gatewayUsageLabel.Hide(); gatewayUsageSelect.Hide()
			gatewayHostnameLabel.Hide(); gatewayHostnameEntry.Hide()
			gatewayCredentialsLabel.Hide(); gatewayCredentialsSelect.Hide()
			gatewayUsernameLabel.Hide(); gatewayUsernameEntry.Hide()
			gatewayPasswordLabel.Hide(); gatewayPasswordEntry.Hide()
			gatewayDomainLabel.Hide(); gatewayDomainEntry.Hide()
		}
		formContainer.Refresh()
	}

	protocolSelect.OnChanged = func(selected string) {
		if selected == "rdp" {
			gatewayCheckLabel.Show()
			useGatewayCheck.Show()
		} else {
			hideAllGatewayRows()
			useGatewayCheck.SetChecked(false)
		}
		formContainer.Refresh()
	}

	labelMinWidth := widget.NewLabelWithStyle("Gateway Credentials", fyne.TextAlignTrailing, fyne.TextStyle{}).MinSize().Width
	formContainer = container.New(&stableFormLayout{labelWidth: labelMinWidth},
		widget.NewLabelWithStyle("Name", fyne.TextAlignTrailing, fyne.TextStyle{}), nameEntry,
		widget.NewLabelWithStyle("Protocol", fyne.TextAlignTrailing, fyne.TextStyle{}), protocolSelect,
		widget.NewLabelWithStyle("Host", fyne.TextAlignTrailing, fyne.TextStyle{}), hostEntry,
		widget.NewLabelWithStyle("Port", fyne.TextAlignTrailing, fyne.TextStyle{}), portEntry,
		widget.NewLabelWithStyle("Username", fyne.TextAlignTrailing, fyne.TextStyle{}), usernameEntry,
		widget.NewLabelWithStyle("Password", fyne.TextAlignTrailing, fyne.TextStyle{}), passwordEntry,
		widget.NewLabelWithStyle("Domain", fyne.TextAlignTrailing, fyne.TextStyle{}), domainEntry,
		widget.NewLabelWithStyle("Description", fyne.TextAlignTrailing, fyne.TextStyle{}), descriptionEntry,
		widget.NewLabelWithStyle("Folder", fyne.TextAlignTrailing, fyne.TextStyle{}), folderSelect,
		gatewayCheckLabel, useGatewayCheck,
		gatewayUsageLabel, gatewayUsageSelect,
		gatewayHostnameLabel, gatewayHostnameEntry,
		gatewayCredentialsLabel, gatewayCredentialsSelect,
		gatewayUsernameLabel, gatewayUsernameEntry,
		gatewayPasswordLabel, gatewayPasswordEntry,
		gatewayDomainLabel, gatewayDomainEntry,
		widget.NewLabel(""), storeTo1PasswordCheck,
		vaultLabel, vaultSelect,
	)

	scroll := container.NewScroll(container.NewPadded(formContainer))
	scroll.SetMinSize(fyne.NewSize(500, 500))
	d := dialog.NewCustomConfirm("Edit Connection", "Submit", "Close", scroll, func(confirmed bool) {
		if !confirmed {
			return
		}

		// If user wants to push password to 1Password
		if storeTo1PasswordCheck.Checked && passwordEntry.Text != "" && !w.manager.IsOnePasswordReference(passwordEntry.Text) {
			vault := vaultSelect.Selected
			reference, err := w.manager.CreateOnePasswordItem(vault, nameEntry.Text, usernameEntry.Text, passwordEntry.Text)
			if err != nil {
				dialog.ShowError(fmt.Errorf("Failed to create 1Password item: %w", err), w.window)
				return
			}
			conn.Password = reference
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

		conn.UseGateway = useGatewayCheck.Checked
		if conn.UseGateway {
			conn.GatewayHostname = gatewayHostnameEntry.Text
			switch gatewayUsageSelect.Selected {
			case "Never":
				conn.GatewayUsageMethod = "never"
			case "Detect":
				conn.GatewayUsageMethod = "detect"
			default:
				conn.GatewayUsageMethod = "always"
			}
			switch gatewayCredentialsSelect.Selected {
			case "Use a different username and password":
				conn.GatewayCredentials = "different"
				conn.GatewayDomain = gatewayDomainEntry.Text
				gwUser := gatewayUsernameEntry.Text
				gwPass := gatewayPasswordEntry.Text
				if storeTo1PasswordCheck.Checked && gwPass != "" && !w.manager.IsOnePasswordReference(gwPass) {
					vault := vaultSelect.Selected
					gwTitle := nameEntry.Text + " Gateway"
					ref, err := w.manager.CreateOnePasswordItem(vault, gwTitle, gwUser, gwPass)
					if err != nil {
						dialog.ShowError(fmt.Errorf("Failed to create 1Password item for gateway: %w", err), w.window)
						return
					}
					conn.GatewayPassword = ref
					conn.GatewayUsername = fmt.Sprintf("op://%s/%s/username", vault, url.PathEscape(gwTitle))
				} else {
					conn.GatewayUsername = gwUser
					conn.GatewayPassword = gwPass
				}
			case "Use a smart card":
				conn.GatewayCredentials = "smartcard"
			default:
				conn.GatewayCredentials = "same"
			}
		} else {
			conn.GatewayHostname = ""
			conn.GatewayUsageMethod = ""
			conn.GatewayCredentials = ""
			conn.GatewayUsername = ""
			conn.GatewayPassword = ""
			conn.GatewayDomain = ""
		}

		if port, err := strconv.Atoi(portEntry.Text); err == nil {
			conn.Port = port
		}

		// Handle folder change
		selectedFolder := folderSelect.Selected
		if selectedFolder != currentFolder {
			// Remove from old parent
			if parentFolder != nil {
				for i, child := range parentFolder.Children {
					if child == conn {
						parentFolder.Children = append(parentFolder.Children[:i], parentFolder.Children[i+1:]...)
						break
					}
				}
			} else {
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
	}, w.window)
	d.Resize(fyne.NewSize(620, 600))
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
