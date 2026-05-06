# MremoteGO

> A modern, cross-platform remote connection manager with git-friendly YAML configs and 1Password integration.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20Linux%20%7C%20macOS-lightgrey)](https://github.com/jaydenthorup/mremotego)

## Why MremoteGO?

**The Problem**: mRemoteNG uses XML configs that are painful to diff, merge, and share with teams. Passwords are awkwardly encrypted per-machine.

**The Solution**: MremoteGO uses clean YAML configs that work beautifully with git, plus optional 1Password integration for secure team password sharing.

## ✨ Features

- 🎨 **Modern GUI** - Clean interface with connection tree, search, and quick actions
- 🔐 **Password Encryption** - AES-256-GCM encryption at rest with master password
- 🔑 **1Password Integration** - Store passwords securely using `op://` references
- 📝 **Git-Friendly** - YAML configs are easy to diff, merge, and review
- 🖥️ **Cross-Platform** - Windows, Linux, macOS (AMD64 & ARM64)
- ⚡ **Fast** - Native GUI with instant connections
- 🚀 **Multiple Protocols** - SSH, RDP, VNC, HTTP/HTTPS, Telnet
- 📁 **Organized** - Folders and search filtering
- 🔒 **Auto-Login** - Password injection for SSH connections
- 💻 **CLI & GUI** - Run without arguments for GUI, with arguments for CLI mode

## 🚀 Quick Start

### Download

Download the latest release for your platform from the [Releases](https://github.com/jaydenthorup/mremotego/releases) page.

### Build from Source

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

- **Double-click** a connection in the tree (shows details and starts connecting)
- Select a connection, then click the **🚀 Connect** button in the details panel
- Select a connection, then click the **▶** toolbar button

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
   - Supports biometric unlock
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

- ✅ Use 1Password for team environments
- ✅ Use encryption for personal configs
- ✅ Add `config.yaml` and `connections.yaml` to `.gitignore`
- ✅ Use separate configs for different environments
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
- For Linux: `gcc`, `libgl1-mesa-dev`, `xorg-dev`
- For GUI builds: Fyne dependencies

### Building

```bash
# Build GUI + CLI (single executable)
go build -o mremotego ./cmd/mremotego-gui

# Build without console window (Windows only)
go build -ldflags "-H windowsgui" -o mremotego.exe ./cmd/mremotego-gui

# Run tests
go test ./...

# Build for all platforms
./build-all.sh
```

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
- [x] 1Password batched credential resolution (single prompt per vault+item)
- [x] AES-256-GCM password encryption
- [x] Cross-platform builds (Windows, Linux, macOS ARM64)
- [x] CLI mode for automation
- [x] Nested folder support with unlimited depth
- [x] Import from mRemoteNG XML
- [x] GitHub Actions CI/CD with automated releases
- [x] Double-click to connect with progress indicator
- [x] Windows taskbar icon embedded in executable

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
- [ ] Linux: System tray integration
- [ ] macOS: Menu bar app mode

### 💡 Ideas (Vote on GitHub Issues!)
- [ ] Cloud sync option (encrypted)


**Want to contribute?** Pick an item from the roadmap and open an issue or PR!

---

**Made with ❤️ by [Jayden Thorup](https://github.com/jaydenthorup)**
