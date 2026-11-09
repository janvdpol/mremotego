package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jaydenthorup/mremotego/pkg/models"
)

// macRDPClientEnv overrides the automatic RDP client choice on macOS:
//
//	"freerdp"    always use sdl-freerdp (error if it is not installed)
//	"windowsapp" always use Windows App / Microsoft Remote Desktop
//	unset/other  automatic: sdl-freerdp if installed, otherwise Windows App
const macRDPClientEnv = "MREMOTEGO_MAC_RDP_CLIENT"

// sdlFreeRDPNames are the binary names of FreeRDP 3's SDL client; distributions
// that version their binaries append a "3".
var sdlFreeRDPNames = []string{"sdl-freerdp3", "sdl-freerdp"}

// extraBinDirs are searched after PATH. A Mac app started from Finder or the
// Dock gets a minimal PATH without the Homebrew and MacPorts directories.
var extraBinDirs = []string{"/opt/homebrew/bin", "/usr/local/bin", "/opt/local/bin"}

// findSDLFreeRDP returns the path of an installed sdl-freerdp binary, or "".
func findSDLFreeRDP() string {
	for _, name := range sdlFreeRDPNames {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	for _, dir := range extraBinDirs {
		for _, name := range sdlFreeRDPNames {
			p := filepath.Join(dir, name)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
				return p
			}
		}
	}
	return ""
}

// chooseMacRDPClient returns the sdl-freerdp binary to launch conn with, or ""
// when Windows App should be used instead.
func chooseMacRDPClient(conn *models.Connection) (string, error) {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(macRDPClientEnv))) {
	case "windowsapp":
		return "", nil
	case "freerdp":
		bin := findSDLFreeRDP()
		if bin == "" {
			return "", fmt.Errorf("%s=freerdp, but sdl-freerdp was not found (install it with 'brew install freerdp')", macRDPClientEnv)
		}
		return bin, nil
	}

	// Automatic. FreeRDP's support for smartcard authentication at the RD
	// Gateway is limited, so leave those connections to Windows App.
	if conn.UseGateway && conn.GatewayCredentials == "smartcard" && conn.GatewayUsageMethod != "never" {
		return "", nil
	}
	return findSDLFreeRDP(), nil
}

// freeRDPArgs builds the FreeRDP 3 command line for conn, mirroring the
// settings that createRDPFile writes for mstsc. The result contains the
// password, so it must never be passed as process arguments; see
// launchSDLFreeRDP.
func freeRDPArgs(conn *models.Connection, target string) ([]string, error) {
	args := []string{"/v:" + target}
	if conn.Name != "" {
		args = append(args, "/t:"+conn.Name)
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
		args = append(args, "/f")
	}
	if conn.ColorDepth > 0 {
		args = append(args, fmt.Sprintf("/bpp:%d", conn.ColorDepth))
	}

	// Server certificate handling, matching mstsc's "authentication level".
	switch conn.RDPAuthenticationLevel {
	case "fail":
		args = append(args, "/cert:deny")
	case "warn":
		// FreeRDP's default: ask the user about an untrusted certificate.
	default:
		args = append(args, "/cert:ignore")
	}

	// mstsc's "use cred ssp:i:0" turns off NLA; do the same here.
	if !conn.UseCredSSP {
		args = append(args, "-sec-nla")
	}

	args = append(args, "+clipboard", "/sound", "+auto-reconnect")

	switch spec := strings.TrimSpace(conn.RDPRedirectDrives); spec {
	case "":
	case "*":
		args = append(args, "/drives", "/drive:hotplug,*")
	case "dynamic":
		args = append(args, "/drive:hotplug,*")
	case "local":
		args = append(args, "/drives")
	default:
		// A list of Windows drive letters means nothing on a Mac; share the
		// home directory as the closest equivalent of "my local files".
		args = append(args, "/home-drive")
	}

	if conn.UseGateway && conn.GatewayHostname != "" && conn.GatewayUsageMethod != "never" {
		gw := "/gateway:g:" + conn.GatewayHostname
		if conn.GatewayUsageMethod == "detect" {
			gw += ",usage-method:detect"
		} else {
			gw += ",usage-method:direct" // FreeRDP's name for "always use the gateway"
		}
		// Without u:/p: FreeRDP uses the session credentials for the gateway,
		// which is what "same" means.
		if conn.GatewayCredentials == "different" && conn.GatewayUsername != "" {
			// Gateway options are comma-separated, so a comma inside a value
			// would be read as the start of the next option.
			for _, v := range []string{conn.GatewayUsername, conn.GatewayDomain, conn.GatewayPassword} {
				if strings.Contains(v, ",") {
					return nil, fmt.Errorf("FreeRDP cannot take gateway credentials that contain a comma; set %s=windowsapp to use Windows App for this connection", macRDPClientEnv)
				}
			}
			gw += ",u:" + conn.GatewayUsername
			if conn.GatewayDomain != "" {
				gw += ",d:" + conn.GatewayDomain
			}
			if conn.GatewayPassword != "" {
				gw += ",p:" + conn.GatewayPassword
			}
		}
		args = append(args, gw)
	}

	args = append(args, strings.Fields(conn.ExtraArgs)...)

	for _, a := range args {
		if strings.ContainsAny(a, "\r\n") {
			return nil, fmt.Errorf("connection settings for FreeRDP may not contain line breaks")
		}
	}
	return args, nil
}

// launchSDLFreeRDP starts sdl-freerdp for conn. All arguments, including the
// passwords, are fed through stdin with /args-from:stdin (one argument per
// line) so they never appear in the process list ('ps').
func (l *Launcher) launchSDLFreeRDP(bin string, conn *models.Connection, target string) error {
	args, err := freeRDPArgs(conn, target)
	if err != nil {
		return err
	}

	out := &tailBuffer{max: 4096}
	cmd := exec.Command(bin, "/args-from:stdin")
	cmd.Stdin = strings.NewReader(strings.Join(args, "\n") + "\n")
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start %s: %w", filepath.Base(bin), err)
	}
	l.notify(fmt.Sprintf("Connecting to %s with FreeRDP", conn.Name))

	go func() {
		if err := cmd.Wait(); err != nil {
			msg := fmt.Sprintf("FreeRDP session for %s ended with an error (%v)", conn.Name, err)
			if last := out.lastLine(); last != "" {
				msg += ": " + last
			}
			l.notify(msg)
		}
	}()
	return nil
}

// tailBuffer keeps only the last max bytes written to it, so a long-running
// session's log output can't grow without bound.
type tailBuffer struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.buf = append(t.buf, p...)
	if over := len(t.buf) - t.max; over > 0 {
		t.buf = append(t.buf[:0], t.buf[over:]...)
	}
	return len(p), nil
}

func (t *tailBuffer) lastLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	lines := strings.Split(strings.TrimSpace(string(t.buf)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}
