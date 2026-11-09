# Password Management

MremoteGO provides flexible and secure password management with multiple options.

## Overview

| Method | Security | Team Sharing | Auto-Login | Best For |
|--------|----------|--------------|------------|----------|
| 1Password | ✅ High | ✅ Yes | ✅ Yes | **Recommended** - Teams |
| Plain Text | ⚠️ Low | ❌ No | ✅ Yes | Personal/testing only |
| No Password | ✅ Manual | N/A | ❌ No | SSH keys, certificates |

## 1Password Integration (Recommended)

Store passwords securely in 1Password and reference them in config files.

### Setup

See [1PASSWORD-SETUP.md](1PASSWORD-SETUP.md) for complete setup instructions.

### Usage

```yaml
connections:
  - name: "My Server"
    password: op://Private/My Server/password  # Secure reference
```

### Benefits
- ✅ Passwords never stored in config files
- ✅ Safe to commit configs to git
- ✅ Team password sharing
- ✅ Biometric unlock
- ✅ Automatic password rotation support
- ✅ Audit logs

## Plain Text Passwords

For personal use or testing environments:

```yaml
connections:
  - name: "Dev Server"
    password: mypassword123  # Plain text (not recommended for production)
```

### Security Considerations
- ⚠️ Readable in config file
- ⚠️ Should not be committed to git
- ⚠️ No audit trail
- ✅ Still works with auto-login features

### Best Practices
- Use `.gitignore` to exclude config files
- Set file permissions: `icacls %APPDATA%\mremotego\config.yaml /inheritance:r /grant:r "%USERNAME%:F"`
- Consider encryption at rest (BitLocker, FileVault)

## RDP Auto-Login

On Windows, MremoteGO uses **Windows Credential Manager** for seamless RDP connections. For macOS, see [macOS](#macos) below.

### How It Works

1. **First Connection**: Password (from 1Password or plain text) is stored in Windows Credential Manager
   ```
   cmdkey /generic:TERMSRV/hostname /user:username /pass:password
   ```

2. **Subsequent Connections**: Windows automatically retrieves credentials
   ```
   mstsc /v:hostname
   ```

3. **Auto-Login**: No password prompt needed

### Benefits
- ✅ Native Windows integration
- ✅ Persistent across sessions
- ✅ User-specific security
- ✅ Works with 1Password references
- ✅ No passwords in temporary files

### Managing Stored Credentials

**View all RDP credentials:**
```powershell
cmdkey /list | Select-String "TERMSRV"
```

**Delete specific credential:**
```powershell
cmdkey /delete:TERMSRV/hostname
```

**Delete all RDP credentials:**
```powershell
cmdkey /list | Select-String "TERMSRV" | ForEach-Object { 
  $target = ($_ -split " ")[1]
  cmdkey /delete:$target
}
```

### macOS

- **sdl-freerdp** (preferred, used automatically when installed with `brew install freerdp`): username, domain and password, gateway credentials included, are passed with `/args-from:stdin`. They never appear on the command line or in `ps`, and nothing is written to disk.
- **Windows App / Microsoft Remote Desktop** (fallback when FreeRDP is not installed): these clients ignore passwords in `.rdp` files. MremoteGO opens a temporary `.rdp` file and puts the password on the clipboard, marked so clipboard managers don't record it. It is removed after 45 seconds, unless something else was copied in the meantime. A separate RD Gateway password is not handed off.
- Force a client with `MREMOTEGO_MAC_RDP_CLIENT=freerdp` or `MREMOTEGO_MAC_RDP_CLIENT=windowsapp`.

## SSH Password Handling

### Windows (PuTTY)
MremoteGO uses PuTTY with password auto-fill:
```
putty.exe -ssh -P 22 -l username -pw password hostname
```

### Linux/Mac (Native SSH)
Uses native ssh client:
```
ssh username@hostname -p 22
```
Note: Password is passed via environment or expected to use SSH keys.

### SSH Key Authentication (Recommended)
For better security, use SSH keys instead of passwords:

1. Generate key pair: `ssh-keygen -t ed25519`
2. Copy to server: `ssh-copy-id user@host`
3. Leave password field empty in MremoteGO
4. SSH will use key authentication automatically

## VNC Connections

VNC passwords are passed to vncviewer:
```
vncviewer hostname:port -password password
```

## Security Comparison

### 1Password References
```yaml
password: op://Private/Server/password
```
- ✅ Config file safe to commit to git
- ✅ Passwords centrally managed in 1Password
- ✅ Team sharing with access control
- ✅ Biometric unlock
- ✅ Audit logs
- ✅ Automatic rotation support

### Plain Text Passwords
```yaml
password: mypassword123
```
- ⚠️ Visible in config file
- ⚠️ Should not be in version control
- ⚠️ No audit trail
- ⚠️ Manual password rotation
- ✅ Simple for personal use
- ✅ Works offline

### Windows Credential Manager (RDP)
- ✅ OS-level secure storage
- ✅ User and machine specific
- ✅ Protected by Windows login
- ✅ Integrated with DPAPI
- ⚠️ Local to machine (not synced)

## Best Practices

### For Teams
1. ✅ Use 1Password for all passwords
2. ✅ Store configs in git with `op://` references
3. ✅ Use shared vaults for team credentials
4. ✅ Enable biometric unlock
5. ✅ Regular access audits

### For Personal Use
1. ✅ Use 1Password if you have it
2. ⚠️ Plain text is acceptable for local dev
3. ✅ Use SSH keys where possible
4. ✅ Keep config file permissions restricted
5. ✅ Don't commit passwords to public repos

### For Production
1. ✅ Use 1Password or enterprise password manager
2. ✅ Certificate-based authentication where possible
3. ✅ SSH keys instead of passwords
4. ✅ Regular credential rotation
5. ✅ Audit all password access
6. ✅ Multi-factor authentication

## Troubleshooting

### RDP asks for password despite stored credentials
- Check Windows Credential Manager: `cmdkey /list`
- Delete and reconnect to refresh: `cmdkey /delete:TERMSRV/hostname`
- Verify username format (use `DOMAIN\username` if needed)

### 1Password reference not working
- Verify 1Password desktop app is running and unlocked
- Check CLI integration is enabled: Settings → Developer
- Test reference: `op read "op://Private/Server/password"`
- Verify vault name and item name (case-sensitive)

### SSH password not auto-filling
- Windows: Verify PuTTY is installed and in PATH
- Check password field is not empty
- For 1Password: Verify reference is correct
- Consider using SSH keys instead

### Password visible in process list
- This is normal for command-line tools
- Use 1Password to minimize exposure
- Passwords are only visible briefly during connection
- Windows Credential Manager used for RDP (not in process list)
- macOS with sdl-freerdp: credentials go through stdin (not in process list)
