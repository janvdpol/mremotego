# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

MremoteGO is a cross-platform remote connection manager written in Go. It replaces mRemoteNG with git-friendly YAML configs and 1Password integration. Supports SSH, RDP, VNC, HTTP/HTTPS, and Telnet. Has both a Fyne-based GUI and a Cobra-based CLI in a single binary.

## Build Commands

```bash
# Build all (CLI + GUI)
make build-all

# Build GUI only
make build-gui

# Build CLI only
make build-cli

# Run tests (no tests currently exist)
make test

# Clean build artifacts
make clean
```

Windows-specific GUI build (handles syso resource embedding):
```powershell
.\build-gui.ps1
```

The GUI binary hides the console window on Windows via `-ldflags "-H windowsgui"`.

## Architecture

### Dual-mode entry point

`cmd/mremotego-gui/main.go` launches GUI if called with no arguments, otherwise delegates to CLI (Cobra). `cmd/mremotego/` is the standalone CLI-only binary.

### Key packages

- **`pkg/models/`** — Core types: `Connection`, `Config`, `Protocol`, `NodeType`. All other packages depend on this; change here with care.
- **`internal/config/`** — Loads/saves YAML config, handles encryption detection, manages recent-file tracking.
- **`internal/crypto/`** — AES-256-GCM encryption with PBKDF2 (100k iterations). Encrypted passwords are stored as `enc:<base64(salt+nonce+ciphertext)>`.
- **`internal/secrets/`** — Wraps the `op` CLI (1Password). Passwords prefixed `op://` are resolved at launch time via subprocess. Platform split: `_windows.go` / `_unix.go` for console-hiding.
- **`internal/launcher/`** — Protocol launchers. Platform split: `launcher_windows.go` handles RDP (via `mstsc`) and WinRM; `launcher_unix.go` handles SSH/VNC/browser via `xdg-open` and `xterm`. Add new protocol support here.
- **`internal/gui/`** — Fyne UI: `mainwindow.go` (layout, toolbar), `treewidget.go` (connection tree), `dialogs.go` (add/edit/delete/gateway dialogs).

### Config file location

- Linux/Mac: `~/.config/mremotego/config.yaml`
- Windows: `%APPDATA%/mremotego/config.yaml`

Last-used path is persisted in `recent.txt` alongside the binary.

### Connection model

`Connection` supports both leaf nodes (actual connections) and folder nodes (`NodeType`: `connection` vs `folder`). Folders have a `Children []Connection` slice enabling unlimited nesting. The entire tree is a single `Config.Connections []Connection`.

### RDP Gateway support

Windows launcher passes gateway credentials to `mstsc` via a temporary `.rdp` file written to `%TEMP%`, then deleted after launch. Gateway config lives on the `Connection` struct fields (`GatewayHostname`, `GatewayUsername`, `GatewayPassword`).

### 1Password integration

The `secrets` package calls `op read <reference>` as a subprocess and returns stdout. On Windows it hides the console window by setting `SysProcAttr.HideWindow`. Special characters in passwords are handled by writing to a temp file rather than passing via command line.

## CI/CD

GitHub Actions (`.github/workflows/`) builds on version tag pushes (`v*`). Produces binaries for Windows AMD64, Linux AMD64/ARM64, macOS ARM64, and creates a GitHub release automatically.
