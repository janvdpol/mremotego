# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.5] - 2026-05-06

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
