//go:build !darwin

package launcher

import "errors"

// copySecretToClipboard is only used by the macOS RDP launcher.
func copySecretToClipboard(secret string) (clear func(), err error) {
	return nil, errors.New("clipboard hand-off is only implemented on macOS")
}
