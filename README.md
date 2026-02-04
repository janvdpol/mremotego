# MremoteGO

> A modern, cross-platform alternative to mRemoteNG with git-friendly YAML configs, 1Password integration, and biometric authentication support.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](https://github.com/jaydenthorup/mremotego)

## Why MremoteGO?

**The Problem with mRemoteNG**: XML configuration files are difficult to diff, merge, and manage in version control. Passwords require per-machine encryption, making team collaboration painful.

**MremoteGO Solution**: Clean YAML configuration files that work seamlessly with git, plus native 1Password integration with biometric unlock for secure, hassle-free password management across your team.

## ✨ Key Features

- 🎨 **Modern GUI** - Clean interface with connection tree, search, and quick actions
- 🔐 **Secure Password Encryption** - AES-256-GCM encryption at rest with master password
- 🔑 **1Password Integration** - Native SDK with biometric authentication OR CLI fallback for teams
- 🗂️ **1Password Vault Mapping** - Use friendly vault names instead of cryptic UUIDs
- 📝 **Git-Friendly YAML** - Diffs, merges, and pull requests that actually make sense
- 🖥️ **Cross-Platform** - Windows, Linux (AMD64/ARM64), macOS (Intel/Apple Silicon)
- ⚡ **Fast Native Performance** - Built with Go and Fyne for instant connections
- 🚀 **Multi-Protocol Support** - SSH, RDP, VNC, HTTP/HTTPS, Telnet
- 📁 **Folder Organization** - Tree structure with search filtering
- 🔒 **SSH Auto-Login** - Automatic password injection for SSH connections
- 💻 **Dual Mode** - GUI (no arguments) or CLI commands for automation
- 🧪 **Well-Tested** - 35+ unit tests covering CLI commands, models, and config management

## 🔑 1Password + mRemoteNG Migration

**Looking to migrate from mRemoteNG to 1Password?** MremoteGO makes it easy:

1. **Import your mRemoteNG connections** using the YAML config format
2. **Replace passwords with 1Password references** like `op://Private/Server/password`
3. **Enable 1Password SDK or CLI** in your settings
4. **Unlock with biometrics** - Touch ID, Windows Hello, or your 1Password master password
5. **Share with your team** - Git-friendly YAML configs + 1Password shared vaults

No more per-machine password encryption headaches. Your team gets secure, centralized password management with mRemoteNG-style organization.

See [docs/1PASSWORD-SETUP.md](docs/1PASSWORD-SETUP.md) for complete integration guide.

## 🚀 Quick Start

### Download

Download the latest release for your platform from the [Releases](https://github.com/jaydenthorup/mremotego/releases) page.

**Windows, Linux, and macOS builds available** - including ARM64 support for Apple Silicon and Linux ARM.

### Build from Source

**Prerequisites**: Go 1.23+, CGO-enabled compiler (GCC on Linux, Xcode Command Line Tools on macOS, MinGW-w64 on Windows)

```bash
# Clone the repository
git clone https://github.com/jaydenthorup/mremotego.git
cd mremotego

# Build (all platforms)
go build -o mremotego ./cmd/mremotego-gui

# Or use platform-specific build scripts
# Windows: .\build-gui.ps1
# Linux/Mac: ./build-gui.sh
```

### First Run

1. Launch `mremotego` (or `mremotego.exe` on Windows)
2. Optionally set a master password for encryption
3. Create your first connection or import from mRemoteNG

That's it! 🎉

## 📖 Usage

### GUI Mode

Simply run the executable without arguments:

```bash
./mremotego        # Linux/Mac
.\mremotego.exe    # Windows
```

**Creating Connections:**

1. Click **[+]** or press `Ctrl+N`
2. Fill in connection details (name, protocol, host, credentials)
3. Optionally push password to 1Password
4. Click **Save**

**Connecting:**

- **Double-click** a connection in the tree
- **Right-click** → **Connect**
- Select and press **Enter**

**Searching:**

- Use the search box at the top
- Filter by connection name, host, or protocol
- Results update in real-time

### CLI Mode

Run with arguments for command-line operations:

```bash
# List all connections
mremotego list

# Connect to a specific host
mremotego connect "Production Server"

# Add a new connection
mremotego add --name "New Server" --protocol ssh --host 192.168.1.100

# Export connections
mremotego export --output connections-backup.yaml

# Edit a connection
mremotego edit "Production Server" --host new.example.com

# Delete a connection
mremotego delete "Old Server"
```

### Example YAML Configuration

```yaml
version: "1.0"
connections:
  - name: Production
    type: folder
    children:
      - name: Web Server
        type: connection
        protocol: ssh
        host: web.prod.example.com
        port: 22
        username: admin
        password: op://DevOps/web-server/password  # 1Password reference
        description: "Primary web server"
        tags:
          - production
          - web
      
      - name: Database Server
        type: connection
        protocol: ssh
        host: db.prod.example.com
        port: 22
        username: dbadmin
        password: "enc:base64..."  # AES-256-GCM encrypted
        
  - name: Development
    type: folder
    children:
      - name: Dev Desktop
        type: connection
        protocol: rdp
        host: dev.example.com
        port: 3389
        username: developer
```

## 🔐 Security

### Password Storage Options

MremoteGO supports three password storage methods:

1. **1Password Integration** (Recommended for teams):
   - Store passwords securely in 1Password vaults
   - Use `op://Vault/Item/field` references in your config
   - Safe to commit configs to git
   - **Two connection methods:**
     - **Desktop App (Recommended)**: Native SDK with biometric unlock - requires 1Password BETA app with SDK enabled
     - **CLI Fallback**: Automatic fallback to `op` CLI if desktop app unavailable
   - Vault name mapping: Use friendly names instead of UUIDs
   - See [1Password Setup Guide](docs/1PASSWORD-SETUP.md)

2. **Encrypted** (Recommended for local use):
   - AES-256-GCM encryption with PBKDF2 key derivation (100,000 iterations)
   - Master password required on startup
   - Passwords stored as `enc:base64(salt+nonce+ciphertext)`
   - See [Encryption Guide](docs/ENCRYPTION.md)

3. **Plain Text** (Not recommended):
   - For testing or when other methods aren't suitable
   - Should not be committed to git
   - Use `.gitignore` to exclude `connections.yaml` and `config.yaml`

### Best Practices

- ✅ Use 1Password for team environments (SDK with biometric OR CLI fallback)
- ✅ Use encryption for personal configs
- ✅ Add `config.yaml` and `connections.yaml` to `.gitignore`
- ✅ Use separate configs for different environments
- ✅ Configure vault name mappings for easier reference management
- ✅ Regularly rotate credentials
- ⚠️ Never commit plain-text passwords to git

## 📚 Documentation

- **[Quick Start Guide](docs/QUICKSTART.md)** - Get started in 5 minutes
- **[GUI Guide](docs/GUI-GUIDE.md)** - Complete GUI reference
- **[Encryption Guide](docs/ENCRYPTION.md)** - Password encryption details
- **[1Password Setup](docs/1PASSWORD-SETUP.md)** - Secure password management
- **[Password Management](docs/PASSWORD-MANAGEMENT.md)** - Security best practices

## 🛠️ Development

### Prerequisites

- Go 1.23 or later
- CGO enabled (for GUI and 1Password SDK)
- For Windows: GCC/MinGW (TDM-GCC or MSYS2)
- For Linux: `gcc`, `libgl1-mesa-dev`, `xorg-dev`
- For GUI builds: Fyne dependencies

### Building

```bash
# Build GUI + CLI (single executable)
# CGO is required for both Fyne GUI and 1Password SDK
export CGO_ENABLED=1  # Linux/Mac
# or
$env:CGO_ENABLED="1"  # Windows PowerShell

go build -o mremotego ./cmd/mremotego-gui

# Build without console window (Windows only)
go build -ldflags "-H windowsgui" -o mremotego.exe ./cmd/mremotego-gui

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Build for all platforms (using GitHub Actions)
# See .github/workflows/build.yml
```

### VS Code Development

The repository includes VS Code configuration for easy development:

- **Build Tasks**: Press `Ctrl+Shift+B` to build
- **Debug/Run**: Press `F5` to run the GUI
- **Recommended Extensions**: Go extension will be suggested
- **CGO Configuration**: Automatically configured in workspace settings

### Project Structure

```
mremotego/
├── cmd/
│   ├── mremotego-gui/     # Main application (GUI + CLI)
│   └── encrypt-passwords/  # Password encryption tool
├── internal/
│   ├── config/            # Configuration management
│   ├── crypto/            # Encryption/decryption
│   ├── gui/               # Fyne GUI components
│   ├── launcher/          # Protocol launchers (SSH, RDP, etc.)
│   └── secrets/           # 1Password integration
├── pkg/
│   └── models/            # Data models
└── docs/                  # Documentation
```

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. **Report Bugs**: Open an issue with detailed reproduction steps
2. **Suggest Features**: Describe your use case and proposed solution
3. **Submit PRs**: Fork, create a feature branch, and submit a pull request
4. **Improve Docs**: Help make documentation clearer and more comprehensive

### Development Workflow

```bash
# Fork and clone
git clone https://github.com/yourusername/mremotego.git
cd mremotego

# Create a feature branch
git checkout -b feature/amazing-feature

# Make your changes
# ... code code code ...

# Test your changes
go test ./...
go build -o mremotego ./cmd/mremotego-gui

# Commit and push
git commit -m "Add amazing feature"
git push origin feature/amazing-feature

# Open a Pull Request on GitHub
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by [mRemoteNG](https://mremoteng.org/)
- Built with [Fyne](https://fyne.io/) GUI toolkit
- Uses [Cobra](https://github.com/spf13/cobra) for CLI
- 1Password integration via [1Password CLI](https://developer.1password.com/docs/cli)

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/jaydenthorup/mremotego/issues)
- **Discussions**: [GitHub Discussions](https://github.com/jaydenthorup/mremotego/discussions)

## 🗺️ Roadmap

### ✅ Completed
- [x] Core connection management (SSH, RDP, VNC, HTTP, Telnet)
- [x] GUI with tree view and search
- [x] 1Password integration with special character support
- [x] AES-256-GCM password encryption
- [x] Cross-platform builds (Windows, Linux, macOS ARM64)
- [x] CLI mode for automation
- [x] Nested folder support with unlimited depth
- [x] Import from mRemoteNG XML
- [x] GitHub Actions CI/CD with automated releases

### 🚧 In Progress
- [ ] Improved settings panel with more options


### 📋 Planned Features

#### Password Managers
- [ ] Bitwarden CLI integration (`bw://` references)
- [ ] LastPass CLI integration (`lpass://` references)
- [ ] HashiCorp Vault integration
- [ ] Pass (password-store) integration for Linux

#### Connection Management
- [ ] Connection groups with credential inheritance
- [ ] SSH key management and agent forwarding
- [ ] Bulk connection operations (edit multiple, duplicate, move)
- [ ] Connection history and favorites
- [ ] Quick connect with recent connections
- [ ] Connection testing (ping, port check)
- [ ] Connection templates for quick setup

#### UI/UX Improvements
- [ ] Multi-tab connections within GUI
- [ ] Dark/light theme toggle
- [ ] Drag-and-drop folder/connection reorganization
- [ ] Customizable keyboard shortcuts
- [ ] Connection icons and colors
- [ ] Grid/list view toggle
- [ ] Advanced search with filters (protocol, tags, etc.)

#### Security & Logging
- [ ] Session recording/logging for audit trails
- [ ] Connection activity timestamps
- [ ] Failed login attempt tracking
- [ ] Security audit reports
- [ ] Two-factor authentication for master password

#### Advanced Features
- [ ] Plugin system for custom protocols
- [ ] Scripting support (pre/post connection commands)
- [ ] Port forwarding configuration
- [ ] Proxy/jump host support
- [ ] VPN integration
- [ ] Connection macros/automation

#### Platform-Specific
- [ ] Windows: Hide console window on launch
- [ ] Linux: System tray integration
- [ ] macOS: Menu bar app mode

### 💡 Ideas (Vote on GitHub Issues!)
- [ ] Cloud sync option (encrypted)


**Want to contribute?** Pick an item from the roadmap and open an issue or PR!

---

## 🔍 Frequently Asked Questions

### How is MremoteGO different from mRemoteNG?

MremoteGO is a modern alternative to mRemoteNG built from the ground up in Go. Key differences:

- **Configuration Format**: YAML instead of XML - easier to read, diff, and merge in git
- **1Password Integration**: Native support for 1Password SDK with biometric authentication
- **Cross-Platform**: Native builds for Windows, Linux (AMD64/ARM64), and macOS (Intel/Apple Silicon)
- **Modern Stack**: Built with Go and Fyne for better performance and maintainability
- **Git-Friendly**: YAML configs work seamlessly with version control and team collaboration
- **Dual Mode**: Single executable works as both GUI and CLI

### Can I migrate from mRemoteNG to MremoteGO?

Yes! MremoteGO uses a YAML format that's easy to populate from mRemoteNG XML exports. You can:

1. Export your mRemoteNG connections to XML
2. Manually convert to YAML format (see example configs in [docs/](docs/))
3. Use 1Password references to replace embedded passwords
4. Import into MremoteGO and start using immediately

A migration tool is on the roadmap to automate this process.

### Does MremoteGO work with 1Password teams/business accounts?

Yes! MremoteGO supports both individual and team 1Password accounts. Use shared vaults for team password management with the vault name mapping feature to reference vaults by friendly names instead of UUIDs.

### What protocols does MremoteGO support?

Currently supported:
- SSH (with password auto-injection)
- RDP (Remote Desktop Protocol)
- VNC (Virtual Network Computing)
- HTTP/HTTPS (opens in browser)
- Telnet

More protocols are planned for future releases.

### Is MremoteGO secure for storing passwords?

Yes! MremoteGO offers multiple secure storage options:

- **1Password integration** (most secure for teams): Passwords stored in 1Password vaults with biometric unlock
- **AES-256-GCM encryption** (secure for local use): Master password-protected with PBKDF2 key derivation (100,000 iterations)
- Plain text passwords should only be used for testing and never committed to git

See the [Security](#-security) section for best practices.

### Can I use MremoteGO in automation/CI pipelines?

Absolutely! MremoteGO includes a full CLI mode. Run with command-line arguments to list, connect, add, edit, delete, or export connections. Perfect for automation scripts, CI/CD pipelines, and infrastructure-as-code workflows.

### What's required to build from source?

- Go 1.23 or later
- CGO-enabled compiler (GCC on Linux, Xcode on macOS, MinGW-w64 on Windows)
- Platform-specific GUI dependencies (see [Development](#-development) section)

Pre-built binaries are available on the [Releases](https://github.com/jaydenthorup/mremotego/releases) page if you don't want to build from source.

---

## 📝 Keywords for Search

**mRemoteNG alternative**, **remote connection manager**, **1Password integration**, **SSH connection manager**, **RDP manager**, **git-friendly remote connections**, **YAML connection config**, **biometric authentication remote desktop**, **cross-platform connection manager**, **mRemoteNG YAML**, **mRemoteNG 1Password**, **secure password manager for SSH/RDP**, **team password sharing**, **remote desktop manager Linux**, **remote desktop manager macOS**, **connection manager with encryption**

---

**Made with ❤️ by [Jayden Thorup](https://github.com/jaydenthorup)**
