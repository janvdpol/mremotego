package launcher

import (
	"slices"
	"strings"
	"testing"

	"github.com/jaydenthorup/mremotego/pkg/models"
)

func TestFreeRDPArgsBasic(t *testing.T) {
	conn := &models.Connection{
		Name:       "srv",
		Host:       "srv.example.com",
		Username:   "jan",
		Domain:     "CORP",
		Password:   "p@ss word",
		UseCredSSP: true,
	}
	args, err := freeRDPArgs(conn, "srv.example.com:3390")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/v:srv.example.com:3390", "/t:srv", "/u:jan", "/d:CORP", "/p:p@ss word", "/f", "/cert:ignore", "+clipboard"} {
		if !slices.Contains(args, want) {
			t.Errorf("missing %q in %q", want, args)
		}
	}
	if slices.Contains(args, "-sec-nla") {
		t.Errorf("NLA disabled although UseCredSSP is set: %q", args)
	}
}

func TestFreeRDPArgsOptions(t *testing.T) {
	conn := &models.Connection{
		Name:                   "srv",
		Resolution:             "1280x800",
		ColorDepth:             16,
		RDPAuthenticationLevel: "fail",
		RDPRedirectDrives:      "C,D",
		ExtraArgs:              "/scale:140  +fonts",
	}
	args, err := freeRDPArgs(conn, "srv")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/size:1280x800", "/bpp:16", "/cert:deny", "-sec-nla", "/home-drive", "/scale:140", "+fonts"} {
		if !slices.Contains(args, want) {
			t.Errorf("missing %q in %q", want, args)
		}
	}
	if slices.Contains(args, "/f") {
		t.Errorf("fullscreen requested although a resolution is set: %q", args)
	}
}

func TestFreeRDPArgsGateway(t *testing.T) {
	conn := &models.Connection{
		UseGateway:         true,
		GatewayHostname:    "gw.example.com",
		GatewayUsageMethod: "detect",
		GatewayCredentials: "different",
		GatewayUsername:    "gwuser",
		GatewayDomain:      "GW",
		GatewayPassword:    "secret",
	}
	args, err := freeRDPArgs(conn, "srv")
	if err != nil {
		t.Fatal(err)
	}
	want := "/gateway:g:gw.example.com,usage-method:detect,u:gwuser,d:GW,p:secret"
	if !slices.Contains(args, want) {
		t.Errorf("missing %q in %q", want, args)
	}

	conn.GatewayPassword = "a,b"
	if _, err := freeRDPArgs(conn, "srv"); err == nil {
		t.Error("expected an error for a gateway password containing a comma")
	}

	conn.GatewayUsageMethod = "never"
	conn.GatewayPassword = "secret"
	args, _ = freeRDPArgs(conn, "srv")
	for _, a := range args {
		if strings.HasPrefix(a, "/gateway") {
			t.Errorf("gateway used although usage method is never: %q", args)
		}
	}
}

func TestFreeRDPArgsRejectsLineBreaks(t *testing.T) {
	conn := &models.Connection{Password: "line1\nline2"}
	if _, err := freeRDPArgs(conn, "srv"); err == nil {
		t.Error("expected an error for a password containing a line break")
	}
}

func TestTailBuffer(t *testing.T) {
	b := &tailBuffer{max: 20} // keeps "st line\nerror: boom\n"
	_, _ = b.Write([]byte("first line\n"))
	_, _ = b.Write([]byte("error: boom\n"))
	if got := b.lastLine(); got != "error: boom" {
		t.Errorf("lastLine = %q", got)
	}
}
