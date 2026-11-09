//go:build darwin

package launcher

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// jxaCopyConcealed reads a secret from stdin and puts it on the general
// pasteboard, marked with the nspasteboard.org Concealed and Transient types so
// clipboard managers (Maccy, Alfred, Raycast, 1Password, ...) don't record it.
// Prints the pasteboard's changeCount so the clear step can tell whether the
// clipboard was overwritten since.
const jxaCopyConcealed = `ObjC.import('AppKit');` +
	`var data = $.NSFileHandle.fileHandleWithStandardInput.readDataToEndOfFile;` +
	`var str = $.NSString.alloc.initWithDataEncoding(data, $.NSUTF8StringEncoding);` +
	`var pb = $.NSPasteboard.generalPasteboard;` +
	`pb.clearContents;` +
	`pb.setStringForType(str, $.NSPasteboardTypeString);` +
	`pb.setStringForType($(''), $('org.nspasteboard.ConcealedType'));` +
	`pb.setStringForType($(''), $('org.nspasteboard.TransientType'));` +
	`String(pb.changeCount);`

// jxaClearIfUnchanged clears the pasteboard only if its changeCount still
// matches (%d), i.e. the user hasn't copied anything else in the meantime.
const jxaClearIfUnchanged = `ObjC.import('AppKit');` +
	`var pb = $.NSPasteboard.generalPasteboard;` +
	`if (pb.changeCount == %d) { pb.clearContents; }` +
	`'';`

// copySecretToClipboard puts secret on the macOS clipboard without exposing it
// on a command line, and returns a function that removes it again, provided the
// clipboard still holds it. The returned function is safe to call more than once.
func copySecretToClipboard(secret string) (clear func(), err error) {
	if out, err := runWithStdin(secret, "osascript", "-l", "JavaScript", "-e", jxaCopyConcealed); err == nil {
		if n, perr := strconv.Atoi(strings.TrimSpace(out)); perr == nil {
			return once(func() {
				_, _ = runWithStdin("", "osascript", "-l", "JavaScript", "-e", fmt.Sprintf(jxaClearIfUnchanged, n))
			}), nil
		}
	}

	// Fallback: plain pbcopy (no concealed marker), cleared by comparing contents.
	if _, err := runWithStdin(secret, "pbcopy"); err != nil {
		return nil, fmt.Errorf("pbcopy: %w", err)
	}
	return once(func() {
		if cur, err := exec.Command("pbpaste").Output(); err == nil && string(cur) == secret {
			_, _ = runWithStdin("", "pbcopy")
		}
	}), nil
}

func runWithStdin(stdin string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	return string(out), err
}

func once(fn func()) func() {
	var o sync.Once
	return func() { o.Do(fn) }
}
