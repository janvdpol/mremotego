package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/jaydenthorup/mremotego/internal/secrets"
	"github.com/jaydenthorup/mremotego/pkg/models"
)

// Launcher handles launching connections
type Launcher struct {
	onePasswordProvider *secrets.OnePasswordProvider
}

// NewLauncher creates a new launcher
func NewLauncher() *Launcher {
	return &Launcher{
		onePasswordProvider: secrets.NewOnePasswordProvider(),
	}
}

// GetOnePasswordProvider returns the 1Password provider for checking authentication status
func (l *Launcher) GetOnePasswordProvider() *secrets.OnePasswordProvider {
	return l.onePasswordProvider
}

// Launch launches a connection based on its protocol
func (l *Launcher) Launch(conn *models.Connection) error {
	if conn.IsFolder() {
		return fmt.Errorf("cannot launch a folder")
	}

	// Resolve 1Password references if needed (make a copy to avoid modifying the original)
	resolvedConn := *conn
	if l.onePasswordProvider.IsReference(conn.Username) {
		resolved, err := l.onePasswordProvider.ResolveSecret(conn.Username)
		if err != nil {
			return fmt.Errorf("failed to resolve username from 1Password: %w", err)
		}
		resolvedConn.Username = resolved
	}
	if l.onePasswordProvider.IsReference(conn.Password) {
		resolved, err := l.onePasswordProvider.ResolveSecret(conn.Password)
		if err != nil {
			// For RDP, we can continue without a password (will prompt)
			// For other protocols that require a password, return the error
			if conn.Protocol != models.ProtocolRDP {
				return fmt.Errorf("failed to resolve password from 1Password: %w", err)
			}
			// RDP: Clear the password so it doesn't try to use the op:// reference
			fmt.Printf("Warning: Failed to resolve password from 1Password: %v (RDP will prompt for credentials)\n", err)
			resolvedConn.Password = ""
		} else {
			resolvedConn.Password = resolved
		}
	}
	if l.onePasswordProvider.IsReference(conn.GatewayUsername) {
		resolved, err := l.onePasswordProvider.ResolveSecret(conn.GatewayUsername)
		if err != nil {
			return fmt.Errorf("failed to resolve gateway username from 1Password: %w", err)
		}
		resolvedConn.GatewayUsername = resolved
	}
	if l.onePasswordProvider.IsReference(conn.GatewayPassword) {
		resolved, err := l.onePasswordProvider.ResolveSecret(conn.GatewayPassword)
		if err != nil {
			fmt.Printf("Warning: Failed to resolve gateway password from 1Password: %v (gateway will prompt for credentials)\n", err)
			resolvedConn.GatewayPassword = ""
		} else {
			resolvedConn.GatewayPassword = resolved
		}
	}

	switch resolvedConn.Protocol {
	case models.ProtocolSSH:
		return l.launchSSH(&resolvedConn)
	case models.ProtocolRDP:
		return l.launchRDP(&resolvedConn)
	case models.ProtocolVNC:
		return l.launchVNC(&resolvedConn)
	case models.ProtocolHTTP, models.ProtocolHTTPS:
		return l.launchHTTP(&resolvedConn)
	case models.ProtocolTelnet:
		return l.launchTelnet(&resolvedConn)
	default:
		return fmt.Errorf("unsupported protocol: %s", resolvedConn.Protocol)
	}
}

// launchSSH launches an SSH connection using PuTTY on Windows, ssh otherwise
func (l *Launcher) launchSSH(conn *models.Connection) error {
	port := conn.Port
	if port == 0 {
		port = models.ProtocolSSH.GetDefaultPort()
	}

	if runtime.GOOS == "windows" {
		// Use PuTTY on Windows
		args := []string{}

		// Add SSH protocol
		args = append(args, "-ssh")

		// Add port
		args = append(args, "-P", strconv.Itoa(port))

		// Add username if provided
		if conn.Username != "" {
			args = append(args, "-l", conn.Username)
		}

		// Add password if provided (for auto-login)
		if conn.Password != "" {
			args = append(args, "-pw", conn.Password)
		}

		// Add extra args if provided
		if conn.ExtraArgs != "" {
			args = append(args, conn.ExtraArgs)
		}

		// Add hostname last
		args = append(args, conn.Host)

		// Try putty.exe first, fall back to ssh if not found
		cmd := exec.Command("putty.exe", args...)
		// Don't hide putty - it's a GUI application we want to see
		if err := cmd.Start(); err != nil {
			// Fall back to ssh command
			return l.launchSSHFallback(conn)
		}
		return nil
	}

	// Use ssh on Linux/Mac
	return l.launchSSHFallback(conn)
}

// launchSSHFallback uses the standard ssh command as fallback
func (l *Launcher) launchSSHFallback(conn *models.Connection) error {
	port := conn.Port
	if port == 0 {
		port = models.ProtocolSSH.GetDefaultPort()
	}

	args := []string{}

	// Add port
	args = append(args, "-p", strconv.Itoa(port))

	// Automatically accept new host keys (security note: this trusts on first use)
	// This prevents the "authenticity of host" prompt from blocking connections
	args = append(args, "-o", "StrictHostKeyChecking=accept-new")

	// Add username if provided
	target := conn.Host
	if conn.Username != "" {
		target = conn.Username + "@" + conn.Host
	}

	args = append(args, target)

	// Add extra args if provided
	if conn.ExtraArgs != "" {
		args = append(args, conn.ExtraArgs)
	}

	var cmd *exec.Cmd

	// If password is provided, try to use sshpass (if available)
	if conn.Password != "" {
		// Check if sshpass is available
		if _, err := exec.LookPath("sshpass"); err == nil {
			// Use sshpass to provide password
			sshpassArgs := []string{"-p", conn.Password, "ssh"}
			sshpassArgs = append(sshpassArgs, args...)

			// Launch in a terminal emulator
			cmd = l.launchInTerminal("sshpass", sshpassArgs...)
		} else {
			// sshpass not available, just use ssh (will prompt for password)
			cmd = l.launchInTerminal("ssh", args...)
		}
	} else {
		cmd = l.launchInTerminal("ssh", args...)
	}

	if cmd == nil {
		return fmt.Errorf("failed to create terminal command")
	}

	return cmd.Start()
}

// launchInTerminal launches a command in a terminal emulator
func (l *Launcher) launchInTerminal(command string, args ...string) *exec.Cmd {
	// Build the command with proper shell quoting
	cmdParts := []string{command}
	cmdParts = append(cmdParts, args...)

	// Join command parts, using shell quoting for safety
	var quotedParts []string
	for _, part := range cmdParts {
		// Use single quotes and escape any single quotes in the string
		quotedParts = append(quotedParts, fmt.Sprintf("'%s'", strings.ReplaceAll(part, "'", "'\\''")))
	}
	fullCmd := strings.Join(quotedParts, " ")

	// For SSH commands on Linux, wrap with error detection for host key changes
	var wrappedCmd string
	if runtime.GOOS == "linux" && (command == "ssh" || command == "sshpass") {
		// Extract hostname from args for ssh-keygen -R
		hostname := ""
		for i, arg := range args {
			if !strings.HasPrefix(arg, "-") && i > 0 {
				// This is likely the hostname or user@hostname
				parts := strings.Split(arg, "@")
				if len(parts) > 1 {
					hostname = parts[1]
				} else {
					hostname = arg
				}
				break
			}
		}

		// Create wrapper that detects host key mismatch and shows helpful message
		// Use a temp file to capture stderr for error detection
		wrappedCmd = fmt.Sprintf(`
			tmpfile=$(mktemp)
			%s 2>&1 | tee "$tmpfile"
			exit_code=${PIPESTATUS[0]}
			
			if [ $exit_code -ne 0 ]; then
				if grep -q "REMOTE HOST IDENTIFICATION HAS CHANGED" "$tmpfile"; then
					echo ""
					echo -e "\033[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"
					echo -e "\033[1;31m  WARNING: HOST KEY HAS CHANGED FOR %s\033[0m"
					echo -e "\033[1;31m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m"
					echo ""
					echo "This could mean:"
					echo "  1. The server was reinstalled (normal)"
					echo "  2. Someone is intercepting your connection (security risk)"
					echo ""
					echo -e "\033[1;33mTo fix this, run:\033[0m"
					echo -e "\033[1;36m  ssh-keygen -R %s\033[0m"
					echo ""
					echo "Then try connecting again."
				fi
				echo ""
				read -p "Press Enter to close..."
			fi
			rm -f "$tmpfile"
		`, fullCmd, hostname, hostname)
	} else {
		wrappedCmd = fullCmd
	}

	if runtime.GOOS == "darwin" {
		// macOS - use Terminal.app with osascript
		// AppleScript to open Terminal and run command
		script := fmt.Sprintf(`tell application "Terminal"
			do script "%s"
			activate
		end tell`, strings.ReplaceAll(wrappedCmd, `"`, `\"`))

		return exec.Command("osascript", "-e", script)
	} else if runtime.GOOS == "linux" {
		// Try common terminal emulators on Linux
		terminals := []struct {
			name     string
			argStyle string // "dash-e", "dash-dash", "direct"
		}{
			{"gnome-terminal", "dash-dash"},
			{"zutty", "dash-e"}, // WSL common terminal
			{"x-terminal-emulator", "dash-e"},
			{"konsole", "dash-e"},
			{"xfce4-terminal", "dash-e"},
			{"xterm", "dash-e"},
		}

		for _, term := range terminals {
			if termPath, err := exec.LookPath(term.name); err == nil {
				switch term.argStyle {
				case "dash-dash":
					// gnome-terminal uses -- to separate
					return exec.Command(termPath, "--", "bash", "-c", wrappedCmd)
				case "dash-e":
					// Most terminals use -e
					return exec.Command(termPath, "-e", "bash", "-c", wrappedCmd)
				}
			}
		}

		fmt.Println("Warning: No terminal emulator found")
	}

	// Fallback: try to run without terminal (will need stdin/stdout)
	fmt.Println("Fallback: Running without terminal")
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// launchRDP launches an RDP connection
func (l *Launcher) launchRDP(conn *models.Connection) error {
	port := conn.Port
	if port == 0 {
		port = models.ProtocolRDP.GetDefaultPort()
	}

	target := conn.Host
	if port != 3389 {
		target = fmt.Sprintf("%s:%d", conn.Host, port)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Store credentials in Windows Credential Manager if password is provided
		if conn.Username != "" && conn.Password != "" {
			if err := l.storeWindowsCredential(conn); err != nil {
				// Continue anyway - mstsc will prompt if credentials aren't stored
				fmt.Printf("Warning: Failed to store credentials: %v\n", err)
			}
		}
		if conn.UseGateway && conn.GatewayHostname != "" {
			if conn.GatewayCredentials == "different" && conn.GatewayPassword != "" {
				if err := l.storeGatewayCredential(conn); err != nil {
					fmt.Printf("Warning: Failed to store gateway credentials: %v\n", err)
				}
			} else if (conn.GatewayCredentials == "same" || conn.GatewayCredentials == "") && conn.Username != "" && conn.Password != "" {
				gwUsername := conn.Username
				if conn.Domain != "" {
					gwUsername = fmt.Sprintf("%s\\%s", conn.Domain, conn.Username)
				}
				// mstsc looks up RD Gateway credentials by bare hostname (no TERMSRV/ prefix),
				// unlike RDP host credentials which use TERMSRV/<host>.
				if err := writeWindowsCredential(conn.GatewayHostname, gwUsername, conn.Password); err != nil {
					fmt.Printf("Warning: Failed to store gateway credentials: %v\n", err)
				}
			}
		}

		// Create a temporary .rdp file with connection settings
		rdpFile, err := l.createRDPFile(conn, target)
		if err != nil {
			return fmt.Errorf("failed to create RDP file: %w", err)
		}

		// Launch mstsc with the RDP file
		cmd = exec.Command("mstsc", rdpFile)
		// Don't hide mstsc - it's a GUI application we want to see

	case "darwin":
		// Use Microsoft Remote Desktop on macOS
		// Format: rdp://[username@]hostname[:port]
		rdpURL := "rdp://"
		if conn.Username != "" {
			rdpURL += conn.Username
			if conn.Domain != "" {
				rdpURL += "@" + conn.Domain
			}
			rdpURL += "@"
		}
		rdpURL += conn.Host
		if port != 3389 {
			rdpURL += fmt.Sprintf(":%d", port)
		}

		// Use 'open' to launch Microsoft Remote Desktop with rdp:// URL
		cmd = exec.Command("open", rdpURL)
		// Note: Password cannot be passed via URL for security reasons
		// Microsoft Remote Desktop will prompt or use saved credentials

	case "linux":
		// Use xfreerdp on Linux
		args := []string{
			"/v:" + target,
			"/cert:ignore",
		}

		if conn.Username != "" {
			args = append(args, "/u:"+conn.Username)
		}

		if conn.Domain != "" {
			args = append(args, "/d:"+conn.Domain)
		}

		if conn.Password != "" {
			args = append(args, "/p:"+conn.Password)
		}

		if conn.Resolution != "" {
			args = append(args, "/size:"+conn.Resolution)
		} else {
			args = append(args, "/f") // Fullscreen by default
		}

		if conn.ColorDepth > 0 {
			args = append(args, fmt.Sprintf("/bpp:%d", conn.ColorDepth))
		}

		if conn.ExtraArgs != "" {
			args = append(args, conn.ExtraArgs)
		}

		cmd = exec.Command("xfreerdp", args...)
		// Don't hide xfreerdp - it's a GUI application we want to see

	default:
		return fmt.Errorf("RDP not supported on this platform")
	}

	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Start()
}

// launchVNC launches a VNC connection
func (l *Launcher) launchVNC(conn *models.Connection) error {
	port := conn.Port
	if port == 0 {
		port = models.ProtocolVNC.GetDefaultPort()
	}

	target := fmt.Sprintf("%s:%d", conn.Host, port)

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Try common VNC clients on Windows
		cmd = exec.Command("vncviewer", target)
	case "linux":
		// Try vncviewer on Linux
		cmd = exec.Command("vncviewer", target)
	case "darwin":
		// Use open with vnc:// protocol on Mac
		target = fmt.Sprintf("vnc://%s:%d", conn.Host, port)
		cmd = exec.Command("open", target)
	default:
		return fmt.Errorf("VNC not supported on this platform")
	}

	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Don't hide VNC - it's a GUI application we want to see

	return cmd.Start()
}

// launchHTTP launches an HTTP/HTTPS connection in default browser
func (l *Launcher) launchHTTP(conn *models.Connection) error {
	url := fmt.Sprintf("%s://%s", conn.Protocol, conn.Host)

	if conn.Port != 0 {
		url = fmt.Sprintf("%s:%d", url, conn.Port)
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
		hideConsoleWindow(cmd)
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("cannot open browser on this platform")
	}

	return cmd.Start()
}

// launchTelnet launches a telnet connection
func (l *Launcher) launchTelnet(conn *models.Connection) error {
	port := conn.Port
	if port == 0 {
		port = models.ProtocolTelnet.GetDefaultPort()
	}

	args := []string{conn.Host, strconv.Itoa(port)}

	cmd := exec.Command("telnet", args...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	// Don't hide telnet - it needs a terminal window

	return cmd.Start()
}

// createRDPFile creates a temporary .rdp file with connection settings
func (l *Launcher) createRDPFile(conn *models.Connection, target string) (string, error) {
	// Create temp directory for RDP files
	tempDir := filepath.Join(os.TempDir(), "mremotego")
	if err := os.MkdirAll(tempDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Sanitize connection name for filename (remove invalid characters)
	safeName := sanitizeFilename(conn.Name)

	// Create RDP file with .rdp extension
	rdpPath := filepath.Join(tempDir, safeName+".rdp")

	// Build RDP file content (UTF-8 encoding)
	rdpContent := fmt.Sprintf("full address:s:%s\r\n", target)

	if conn.Username != "" {
		// Include domain if specified
		if conn.Domain != "" {
			rdpContent += fmt.Sprintf("domain:s:%s\r\n", conn.Domain)
			rdpContent += fmt.Sprintf("username:s:%s\\%s\r\n", conn.Domain, conn.Username)
		} else {
			rdpContent += fmt.Sprintf("username:s:%s\r\n", conn.Username)
		}
	}

	// Note: Password is stored in Windows Credential Manager, not in RDP file
	// This is the same approach that mstsc uses for saved credentials

	// Common RDP settings (must use \r\n line endings for Windows)
	rdpContent += "screen mode id:i:2\r\n" // Fullscreen
	rdpContent += "use multimon:i:0\r\n"
	rdpContent += "session bpp:i:32\r\n"
	rdpContent += "compression:i:1\r\n"
	rdpContent += "keyboardhook:i:2\r\n"
	rdpContent += "audiocapturemode:i:0\r\n"
	rdpContent += "videoplaybackmode:i:1\r\n"
	rdpContent += "connection type:i:7\r\n"
	rdpContent += "networkautodetect:i:1\r\n"
	rdpContent += "bandwidthautodetect:i:1\r\n"
	rdpContent += "displayconnectionbar:i:1\r\n"
	rdpContent += "enableworkspacereconnect:i:0\r\n"
	rdpContent += "disable wallpaper:i:0\r\n"
	rdpContent += "allow font smoothing:i:1\r\n"
	rdpContent += "allow desktop composition:i:1\r\n"
	rdpContent += "disable full window drag:i:0\r\n"
	rdpContent += "disable menu anims:i:0\r\n"
	rdpContent += "disable themes:i:0\r\n"
	rdpContent += "disable cursor setting:i:0\r\n"
	rdpContent += "bitmapcachepersistenable:i:1\r\n"
	rdpContent += "audiomode:i:0\r\n"
	rdpContent += "redirectprinters:i:0\r\n"
	rdpContent += "redirectcomports:i:0\r\n"
	rdpContent += "redirectsmartcards:i:0\r\n"
	rdpContent += "redirectclipboard:i:1\r\n"
	rdpContent += "redirectposdevices:i:0\r\n"
	rdpContent += "autoreconnection enabled:i:1\r\n"

	// Authentication level: 0 = always connect, 1 = don't connect on failure, 2 = warn
	authLevel := 0
	switch conn.RDPAuthenticationLevel {
	case "fail":
		authLevel = 1
	case "warn":
		authLevel = 2
	}
	rdpContent += fmt.Sprintf("authentication level:i:%d\r\n", authLevel)

	rdpContent += "prompt for credentials:i:0\r\n"
	rdpContent += "negotiate security layer:i:1\r\n"

	// Use CredSSP
	credSSP := 1
	if !conn.UseCredSSP {
		credSSP = 0
	}
	rdpContent += fmt.Sprintf("use cred ssp:i:%d\r\n", credSSP)

	rdpContent += "remoteapplicationmode:i:0\r\n"
	rdpContent += "alternate shell:s:\r\n"
	rdpContent += "shell working directory:s:\r\n"
	if conn.UseGateway && conn.GatewayHostname != "" {
		rdpContent += fmt.Sprintf("gatewayhostname:s:%s\r\n", conn.GatewayHostname)

		usageMethod := 1 // default: always
		switch conn.GatewayUsageMethod {
		case "detect":
			usageMethod = 2
		case "never":
			usageMethod = 4
		}
		rdpContent += fmt.Sprintf("gatewayusagemethod:i:%d\r\n", usageMethod)

		credSource := 0 // default: NTLM password, uses Credential Manager silently
		switch conn.GatewayCredentials {
		case "smartcard":
			credSource = 1
		}
		rdpContent += fmt.Sprintf("gatewaycredentialssource:i:%d\r\n", credSource)

		if conn.GatewayCredentials == "different" && conn.GatewayUsername != "" {
			rdpContent += fmt.Sprintf("gatewayusername:s:%s\r\n", conn.GatewayUsername)
			if conn.GatewayDomain != "" {
				rdpContent += fmt.Sprintf("gatewaydomain:s:%s\r\n", conn.GatewayDomain)
			}
		}
	} else {
		rdpContent += "gatewayhostname:s:\r\n"
		rdpContent += "gatewayusagemethod:i:4\r\n"
		rdpContent += "gatewaycredentialssource:i:4\r\n"
	}

	// Gateway profile usage method: 0 = use default profile, 1 = use explicit settings (default)
	profileMethod := 1
	if conn.GatewayProfileUsageMethod == "default" {
		profileMethod = 0
	}
	rdpContent += fmt.Sprintf("gatewayprofileusagemethod:i:%d\r\n", profileMethod)

	rdpContent += "promptcredentialonce:i:0\r\n"
	rdpContent += "gatewaybrokeringtype:i:0\r\n"
	rdpContent += "use redirection server name:i:1\r\n"
	rdpContent += "rdgiskdcproxy:i:0\r\n"
	rdpContent += "kdcproxyname:s:\r\n"

	// Bypass gateway for local addresses
	bypassLocal := 0
	if conn.GatewayBypassIfLocal {
		bypassLocal = 1
	}
	rdpContent += fmt.Sprintf("gatewayisbypassiflocal:i:%d\r\n", bypassLocal)

	// Resolution settings
	if conn.Resolution != "" {
		// Parse resolution like "1920x1080"
		rdpContent += "desktopwidth:i:1920\r\n"
		rdpContent += "desktopheight:i:1080\r\n"
	} else {
		rdpContent += "smart sizing:i:1\r\n"
	}

	// Color depth
	if conn.ColorDepth > 0 {
		rdpContent += fmt.Sprintf("session bpp:i:%d\r\n", conn.ColorDepth)
	}

	// Write RDP file
	if err := os.WriteFile(rdpPath, []byte(rdpContent), 0600); err != nil {
		return "", fmt.Errorf("failed to write RDP file: %w", err)
	}

	return rdpPath, nil
}

// sanitizeFilename removes or replaces invalid characters from a filename
func sanitizeFilename(name string) string {
	// Replace invalid Windows filename characters with underscore
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	// Trim spaces and dots from start/end
	result = strings.Trim(result, " .")
	// If empty after sanitization, use a default name
	if result == "" {
		result = "connection"
	}
	return result
}

// storeWindowsCredential stores RDP credentials in Windows Credential Manager
func (l *Launcher) storeWindowsCredential(conn *models.Connection) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// Build the target name (hostname or hostname:port)
	target := conn.Host
	port := conn.Port
	if port == 0 {
		port = models.ProtocolRDP.GetDefaultPort()
	}
	if port != 3389 {
		target = fmt.Sprintf("%s:%d", conn.Host, port)
	}

	// Build the username (with domain if specified)
	username := conn.Username
	if conn.Domain != "" {
		username = fmt.Sprintf("%s\\%s", conn.Domain, conn.Username)
	}

	return writeWindowsCredential("TERMSRV/"+target, username, conn.Password)
}

// storeGatewayCredential stores RD Gateway credentials in Windows Credential Manager
func (l *Launcher) storeGatewayCredential(conn *models.Connection) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	username := conn.GatewayUsername
	if conn.GatewayDomain != "" {
		username = fmt.Sprintf("%s\\%s", conn.GatewayDomain, conn.GatewayUsername)
	}

	return writeWindowsCredential("TERMSRV/"+conn.GatewayHostname, username, conn.GatewayPassword)
}

// RemoveWindowsCredential removes RDP credentials from Windows Credential Manager
func (l *Launcher) RemoveWindowsCredential(conn *models.Connection) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	// Build the target name (hostname or hostname:port)
	target := conn.Host
	port := conn.Port
	if port == 0 {
		port = models.ProtocolRDP.GetDefaultPort()
	}
	if port != 3389 {
		target = fmt.Sprintf("%s:%d", conn.Host, port)
	}

	return deleteWindowsCredential("TERMSRV/" + target)
}

// CleanupAllCredentials removes all MremoteGO-stored credentials from Windows Credential Manager
func (l *Launcher) CleanupAllCredentials(connections []*models.Connection) error {
	if runtime.GOOS != "windows" {
		return nil
	}

	for _, conn := range connections {
		if conn.Protocol == models.ProtocolRDP {
			_ = l.RemoveWindowsCredential(conn) // Ignore errors, continue with others
		}
	}

	return nil
}
