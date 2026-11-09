# macOS: sdl-freerdp support — test checklist

This branch adds sdl-freerdp as the preferred RDP client on macOS, so that
credentials from 1Password are passed in automatically. Windows App needs the
password pasted from the clipboard. The code was written and unit-tested on
Windows only. **Nobody has run it on a Mac yet.** This document lists what still
needs to be checked, and where to change things if something doesn't work.

Code: `internal/launcher/freerdp.go` (detection, arguments, launch) and the
`case "darwin":` branch of `launchRDP` in `internal/launcher/launcher.go`.

## How it is supposed to work

1. On an RDP connect, MremoteGO looks for `sdl-freerdp3` or `sdl-freerdp`: first in
   `PATH`, then in `/opt/homebrew/bin`, `/usr/local/bin` and `/opt/local/bin`.
2. If found, it starts `sdl-freerdp /args-from:stdin` and writes all arguments,
   one per line, to stdin, passwords included. They never appear in `ps`.
3. If not found, it opens a temporary `.rdp` file in Windows App and puts the
   password on the clipboard for 45 seconds. This is the existing behaviour.
4. Connections whose RD Gateway uses smartcard authentication always go to
   Windows App.
5. `MREMOTEGO_MAC_RDP_CLIENT=freerdp` or `=windowsapp` forces one client.

## Getting a build

Pick one of these:

- **GitHub Actions:** Actions → *Build Cross-Platform* → *Run workflow*, select
  this branch, then download the `darwin-arm64` artifact.
- **Local build:** needs Go 1.24+ and the Xcode command line tools
  (`xcode-select --install`):

  ```sh
  git checkout feature/macos-sdl-freerdp
  go build -o mremotego ./cmd/mremotego-gui
  ./mremotego
  ```

Install FreeRDP: `brew install freerdp`.

## Checklist

Fill in the result column and push the changes, or send them back.

| # | Check | How | Result |
|---|---|---|---|
| 1 | Does Homebrew's `freerdp` install the SDL client? | `ls $(brew --prefix)/bin/*freerdp*`. If only `xfreerdp` is there, detection finds nothing and Windows App stays in use. See "If something fails" below. | |
| 2 | Does `/args-from:stdin` work with one argument per line? **Most critical.** | Manual test below. If this fails, FreeRDP connections fail completely. | |
| 3 | Plain connection with 1Password credentials | Connect from MremoteGO to a server with an `op://` username/password. You should be logged in without typing anything. | |
| 4 | Password not visible in the process list | While connected: `ps -ef \| grep freerdp`. Only `/args-from:stdin` should be shown, no `/p:`. | |
| 5 | Started from Finder/Dock (minimal PATH) | Start MremoteGO not from a terminal. Check that sdl-freerdp is still found and `op` still works. | |
| 6 | RD Gateway, *same* credentials | Connection with a gateway and gateway credentials "same". FreeRDP should reuse the session login for the gateway without asking. | |
| 7 | RD Gateway, *different* credentials | Gateway with its own username/password (from 1Password). | |
| 8 | Certificate setting *fail* | Set "authentication level" to fail and connect to a server with a self-signed certificate. `/cert:deny` should abort the connection. | |
| 9 | Certificate setting *warn* | FreeRDP should ask whether to trust the certificate. | |
| 10 | Drive redirection | Try `*` (`/drives` + `/drive:hotplug,*`), `dynamic` (`/drive:hotplug,*`), `local` (`/drives`) and a letter list such as `C,D` (`/home-drive`). Do the drives show up under `\\tsclient` in the session? | |
| 11 | Resolution / full screen | Without a resolution: full screen (`/f`). With e.g. `1280x800`: windowed at that size. | |
| 12 | Error reporting | Connect with a wrong password. The status bar of MremoteGO should show "FreeRDP session for … ended with an error" plus FreeRDP's last output line. | |
| 13 | Fallback to Windows App | `MREMOTEGO_MAC_RDP_CLIENT=windowsapp ./mremotego`. Windows App opens, the password is on the clipboard (Cmd+V), and it is gone after 45 seconds. | |
| 14 | Clipboard managers | With the Windows App fallback, the password should **not** appear in Maccy/Raycast/Alfred/1Password clipboard history. | |
| 15 | Keyboard, Cmd key, Retina scaling | Is working in an sdl-freerdp session comfortable enough compared with Windows App? | |

### Manual test for check 2

This runs without MremoteGO. Replace the placeholders. Typing the password
at the `read` prompt keeps it out of your shell history.

```sh
read -s -p "Password: " PW; echo
printf '%s\n' "/v:<host>" "/u:<user>" "/d:<domain>" "/p:$PW" "/cert:ignore" \
  | sdl-freerdp /args-from:stdin
unset PW
```

If this doesn't connect, find out which input FreeRDP expects:

```sh
sdl-freerdp /help | grep -A3 args-from
```

## If something fails

| Problem | Where to change it |
|---|---|
| Homebrew installs only `xfreerdp` (needs XQuartz) | `sdlFreeRDPNames` in `freerdp.go`. `xfreerdp` accepts the same arguments, so adding it to the list may be enough. Decide whether XQuartz is acceptable. |
| `/args-from:stdin` uses a different format, or is missing | `launchSDLFreeRDP` in `freerdp.go`. The fallback is `/from-stdin` (FreeRDP prompts for the password on stdin) for the session password. A separate gateway password would then have to go on the command line. |
| An option name is wrong or unknown (`/cert:deny`, `/drives`, `/drive:hotplug,*`, `/home-drive`, `+auto-reconnect`, `usage-method:direct`) | `freeRDPArgs` in `freerdp.go`, and update the expectations in `freerdp_test.go`. |
| Gateway with same credentials asks for a password | Add `u:`/`d:`/`p:` from the session credentials to the `/gateway:` option in `freeRDPArgs` for the "same" case. |
| sdl-freerdp is not found when started from Finder | `extraBinDirs` in `freerdp.go`. Note the actual install path from check 1. |

Run `go test ./internal/launcher/` after changes.
