# 1Password Setup Guide

MremoteGO integrates with 1Password in two ways to securely store and retrieve passwords:

1. **Desktop App Integration (Recommended)** - Native SDK with biometric unlock
2. **CLI Fallback** - Automatic fallback to `op` command if desktop app unavailable

This allows you to:
- Keep passwords out of config files
- Share configs via git safely
- Use biometric unlock for password access
- Work in any environment (desktop or CLI-only)

## 🔷 Option 1: Desktop App Integration (Recommended)

### Prerequisites

- 1Password desktop app **NIGHTLY version** (required for SDK support)
- Download from: https://releases.1password.com/

### 1. Install 1Password Desktop App

Download and install the NIGHTLY version of the 1Password desktop app.

### 2. Enable SDK Integration

1. Open **1Password desktop app**
2. Go to **Settings → Developer**
3. Under  **"Integrate with the 1Password SDKs"**
4. Enable **"Integrate with other apps"**

### 3. Configure MremoteGO

In your `config.yaml`:

```yaml
settings:
  onePasswordAccount: "your-account-name"
  vaultNameMappings:
    Personal: "uuid-of-personal-vault"
    Work: "uuid-of-work-vault"
```

**Finding your account name:**
- Look at the top of the sidebar in the 1Password desktop app
- Example: "My Personal Account" or "work.1password.com"

**Vault name mappings (optional but recommended):**
- Maps friendly names to vault UUIDs
- Use `Personal` instead of long UUID in references
- Get vault IDs from the authentication instructions dialog

### 4. Verify Setup

Launch MremoteGO - it will show connection status:
- ✅ SDK: Connected (using desktop app)
- If not connected, check the instructions in the authentication dialog

### Benefits of Desktop App Integration

- 🔐 **Biometric authentication** - Touch ID, Face ID, Windows Hello
- ⚡ **Faster** - Direct integration, no CLI process spawning
- 🛡️ **More secure** - Session management handled by 1Password
- 💎 **Better UX** - Seamless authentication flow

## 🔷 Option 2: CLI Fallback

MremoteGO automatically falls back to CLI if desktop app integration isn't available.

### 1. Install 1Password CLI

```powershell
# Windows (Chocolatey)
choco install op

# macOS (Homebrew)
brew install 1password-cli

# Linux
# See: https://developer.1password.com/docs/cli/get-started/
```

### 2. Sign In

```powershell
# Sign in to your account
op signin

# Verify authentication
op whoami
```

### 3. Launch MremoteGO

Launch MremoteGO from the **same terminal** where you ran `op signin`:

```powershell
# The session token is inherited automatically
.\mremotego.exe
```

### Benefits of CLI Fallback

- 🖥️ **Works everywhere** - Headless servers, CI/CD, remote environments
- 🔄 **Automatic** - No configuration needed, just falls back
- 🛠️ **Scripting** - Great for automation and testing
- 📦 **Service accounts** - Supports OP_SERVICE_ACCOUNT_TOKEN

## Using 1Password References

### In Your Config

Instead of plain text passwords, use 1Password references:

```yaml
connections:
  - name: "Production Server"
    protocol: ssh
    host: prod.example.com
    username: admin
    password: op://Private/Production Server/password
```

Reference format: `op://vault-name/item-name/field-name`

### Creating Items in 1Password

**Option 1: Use the GUI (Recommended)**
1. Click **"Store in 1Password"** checkbox when adding/editing connections
2. Select the vault
3. MremoteGO creates the item automatically and handles special characters

**Option 2: Manual**
1. Open 1Password desktop app
2. Create a new **Login** item
3. Set the title to your server name
4. Add username and password fields
5. Use format: `op://vault-name/item-title/password` in MremoteGO

**Note:** If your item name contains special characters like `()`, `[]`, or spaces, MremoteGO will automatically URL-encode them in the reference. For example:
- Item: `Server (Production)` → Reference: `op://vault/Server%20%28Production%29/password`
- The GUI handles this automatically when you use "Store in 1Password"

### Using Vault Name Mappings

With vault name mappings, you can use friendly names instead of UUIDs:

**Without mappings:**
```yaml
password: op://abcd-1234-efgh-5678/Production Server/password
```

**With mappings:**
```yaml
# In config.yaml
settings:
  vaultNameMappings:
    Work: "abcd-1234-efgh-5678"

# In connections
password: op://Work/Production Server/password
```

### Common Vaults

- `Private` or `Personal` - Your personal vault
- `Shared` - Team shared vault  
- `Work` or `DevOps` - Custom team vault (configure in mappings)

## How It Works

### Desktop App Integration (SDK)

```
┌─────────────────────────────────────────────────┐
│ 1. User clicks "Connect" in MremoteGO          │
│    Connection has: op://Work/Server/password   │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 2. MremoteGO calls 1Password SDK               │
│    Maps "Work" → vault UUID using config       │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 3. 1Password desktop app prompts biometric     │
│    Touch ID / Face ID / Windows Hello          │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 4. Password returned to MremoteGO              │
│    - RDP: Stored in Windows Credential Manager │
│    - SSH: Passed to PuTTY with -pw flag        │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 5. Connection launches with auto-login         │
└─────────────────────────────────────────────────┘
```

### CLI Fallback

```
┌─────────────────────────────────────────────────┐
│ 1. User clicks "Connect" in MremoteGO          │
│    SDK not available, falls back to CLI        │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 2. MremoteGO calls: op read op://...           │
│    (Hidden console window - no popup)           │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 3. op CLI uses session token or service account│
│    No additional authentication needed         │
└─────────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────────┐
│ 4. Password returned and connection launches   │
└─────────────────────────────────────────────────┘
```

## Benefits

✅ **Secure** - Passwords never stored in config files  
✅ **Team Sharing** - Commit configs to git, passwords stay in 1Password  
✅ **Biometric Unlock** - Touch ID, Face ID, or Windows Hello  
✅ **No Console Windows** - All CLI calls are hidden  
✅ **Cross-Platform** - Works on Windows, Mac, Linux  
✅ **Persistent Sessions** - No repeated authentication while 1Password app is unlocked

## Troubleshooting

### "1Password CLI is not available"
- Run `op --version` to verify CLI is installed
- Make sure `op.exe` is in your PATH
- Restart terminal after installation

### "Failed to retrieve secret"
- Make sure 1Password desktop app is running and you're signed in
- Check "Integrate with 1Password CLI" is enabled (Settings → Developer)
- Verify the vault name and item name are correct
- Ensure you have access to the vault

### "Authentication required" every time
- Make sure biometric unlock is enabled in 1Password settings
- Keep 1Password desktop app running
- The app must be unlocked for CLI to work

### Wrong password retrieved
- Check the field name in your reference (usually "password")
- Verify the item title matches exactly (case-sensitive)
- Use 1Password app to view the item and confirm field names

## Advanced Usage

### Listing Vaults

```powershell
op vault list
```

### Viewing an Item

```powershell
op item get "Server Name" --vault Private
```

### Testing a Reference

```powershell
op read "op://Private/Production Server/password"
```

### Environment Variables

If using service account tokens (for automation):

```powershell
$env:OP_SERVICE_ACCOUNT_TOKEN = "ops_your_token_here"
```

See [1Password docs](https://developer.1password.com/docs/service-accounts/) for service account setup.

## Security Notes

- Config files with `op://` references are safe to commit to git
- Actual passwords never leave 1Password
- Each team member uses their own 1Password account
- Revoke vault access to remove password access
- Audit password usage through 1Password activity logs
