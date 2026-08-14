# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.0] - 2026-08-14

### Added
- **RDP local drive redirection**: RDP connections can now map local drives into the remote session. Configurable via the new "Redirect local drives" option in the Add/Edit Connection dialogs, or the `rdp_redirect_drives` YAML field. Four modes are supported:
  - **All drives** (`"*"`) — every local drive present now, plus any connected later
  - **All local disks only (no network drives)** (`"local"`) — local fixed/removable drives present at launch time, resolved via `GetDriveTypeW` to exclude network-mapped drive letters
  - **Dynamic drives** (`"dynamic"`) — only drives connected after the session starts (e.g. USB sticks), maps to the `.rdp` file's `DynamicDrives` token
  - **Specific drives** (e.g. `"C,D"`) — an explicit comma-separated list of drive letters

  Windows exposes redirected drives both as new volumes under "This PC" and as subdirectories under `\\tsclient`. Implemented via the `.rdp` file's `drivestoredirect` setting.

### Fixed
- **Startup crash from git status checking**: the background config-git-status check (added in 1.3.0) called Fyne widget and dialog methods directly from a goroutine instead of marshalling them onto the main UI thread via `fyne.Do`, which newer Fyne versions treat as a fatal thread-safety violation. Affected `notifyIfConfigOutdated`, `startPeriodicGitCheck`, and the "pull latest config" confirmation flow in `cmd/mremotego-gui/main.go`.

## [1.3.0] - 2026-05-27

### Added
- **Git status checking for config updates**: on startup (and every 15 minutes thereafter), MremoteGo checks whether the config repository's local `HEAD` is behind its remote via `git ls-remote` (read-only, no fetch). A status indicator appears in the status bar, and if the config is behind, a dialog offers to `git pull` the latest version and reload it automatically.

## [1.2.0] - 2026-05-06

### Added
- **Automatic RDP session cleanup**: credentials and temporary `.rdp` files created by MRemoteGo are now fully cleaned up at the right moment — whichever comes first: immediately after authentication completes (detected by mstsc's window title containing the hostname), when the session closes, when the app is closed, or on next startup after a crash. Every credential is tagged with `Comment = "Managed by MRemoteGo"` so orphaned entries can be identified and removed. Multiple concurrent RDP sessions are tracked and cleaned up independently.

## [1.1.1] - 2026-05-06

### Added
- **Windows taskbar icon**: application icon is now embedded in the `.exe` PE resource section via `go-winres`, so it persists when the app is pinned to the Windows taskbar
- **1Password batched resolution**: credentials from the same vault+item are now fetched in a single `op item get --format json` call instead of one call per field, reducing biometric prompts
- **Unit tests for 1Password**: `ResolveMultiple` is covered by four tests (same-item batching, different items, non-reference passthrough, duplicate deduplication)

### Changed
- **Double-click to connect**: now uses Fyne's `DoubleTappable` interface for reliable detection; single-click still selects and shows connection details
- Double-clicking a connection selects the item, populates the details panel, and shows the knight-rider progress bar — matching the behaviour of the Connect button
- `build-gui.ps1` and GitHub Actions Windows build now run `go-winres` before compilation to regenerate icon resources automatically
- Windows build now compiles from the package path (`./cmd/mremotego-gui/`) so `.syso` resource files are automatically included

### Fixed
- Double-click no longer silently fails when clicking an already-selected tree item (root cause: Fyne fires `Tapped` *or* `DoubleTapped`, never both)
- Single-click selection broke when `DoubleTappable` was added; fixed by implementing both `Tappable` and `DoubleTappable` on the tree node widget and driving `tree.Select` explicitly

## [1.0.4] - 2026-01-28

### Fixed
- **Critical Fix**: 1Password references with special characters now work correctly
- Changed from `op read` to `op item get` for better special character handling
- Item names with parentheses, brackets, and other special chars now resolve properly
- Automatically decodes URL-encoded item names in references

### Technical Details
- `op://vault/(item)/field` references now work without encoding issues
- More robust error messages when 1Password retrieval fails

## [1.0.3] - 2026-01-28

### Fixed
- 1Password item names with special characters (parentheses, brackets, etc.) now work correctly
- Item names are now URL-encoded in references (e.g., `Server (Production)` → `Server%20%28Production%29`)
- GitHub Actions workflow now creates releases with proper binary assets when version tags are pushed

## [1.0.2] - 2026-01-28

### Added
- Recursive folder support in GUI dialogs - can now select nested folders at any depth
- Folder paths displayed with " / " separator (e.g., "Dev-Ops / Infrastructure / Builders")
- Add Folder dialog now allows creating subfolders within existing folders
- Helper functions: `collectAllFolders()`, `findConnectionParent()`, `findFolderByPath()`

### Changed
- Add Connection dialog now shows all nested folders in dropdown
- Edit Connection dialog correctly finds and displays deeply nested folder locations
- Can now move connections between folders at any nesting level

### Fixed
- GUI dialogs previously only supported root-level folders, now supports unlimited nesting depth

## [1.0.1] - 2026-01-28

### Changed
- Updated GUI with nested folder organization support
- Improved folder structure management

## [1.0.0] - 2026-01-27

### Added
- Initial release of MremoteGO
- Cross-platform connection manager (Windows, Linux, macOS)
- Support for SSH, RDP, VNC, HTTP, HTTPS, Telnet protocols
- 1Password integration for secure password management
- YAML-based connection storage
- GUI application using Fyne framework
- CLI commands: init, list, connect, add, edit, delete, export
- Folder organization for connections
- Connection search and filtering
- Password encryption support
- mRemoteNG XML import capability
