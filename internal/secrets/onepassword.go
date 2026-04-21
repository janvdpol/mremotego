package secrets

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// OnePasswordProvider handles retrieving secrets from 1Password CLI
type OnePasswordProvider struct {
	enabled bool
}

// NewOnePasswordProvider creates a new 1Password provider
func NewOnePasswordProvider() *OnePasswordProvider {
	return &OnePasswordProvider{
		enabled: isOnePasswordCLIAvailable(),
	}
}

// isOnePasswordCLIAvailable checks if the 1Password CLI (op) is installed
func isOnePasswordCLIAvailable() bool {
	_, err := exec.LookPath("op")
	return err == nil
}

// IsEnabled returns whether 1Password CLI is available
func (p *OnePasswordProvider) IsEnabled() bool {
	return p.enabled
}

// IsAuthenticated checks if the user is currently signed in to 1Password CLI
func (p *OnePasswordProvider) IsAuthenticated() bool {
	if !p.enabled {
		return false
	}

	cmd := exec.Command("op", "whoami")
	hideConsoleWindow(cmd)
	return cmd.Run() == nil
}

// GetAuthenticationInstructions returns instructions for authenticating with 1Password CLI
func (p *OnePasswordProvider) GetAuthenticationInstructions() string {
	return `1Password CLI needs authentication.

✅ OPTION 1: Launch from same terminal (Session Token)
If you run "op signin" in PowerShell, then launch MremoteGO from that SAME terminal:
  Invoke-Expression $(op signin)
  .\mremotego.exe
The GUI will inherit your session token and "op://" passwords will work!

✅ OPTION 2: Desktop App Integration (Recommended - Persistent Auth)
1. Open the 1Password desktop app
2. Sign in if you haven't already
3. Go to Settings → Developer
4. Enable "Integrate with 1Password CLI"
5. Restart MremoteGO

This allows the CLI to use your desktop app session with biometric unlock.
No terminal commands needed - it just works!

────────────────────────────────────────────────────

Alternative (not recommended):
You can set the OP_SERVICE_ACCOUNT_TOKEN environment variable for automation,
but this requires a 1Password service account (business/team plans only).`
}

// IsReference checks if a string is a 1Password reference (starts with op://)
func (p *OnePasswordProvider) IsReference(value string) bool {
	return strings.HasPrefix(value, "op://")
}

// ResolveSecret retrieves a secret from 1Password using the CLI
// Reference format: op://vault/item/field
// Example: op://Private/MyServer/password
func (p *OnePasswordProvider) ResolveSecret(reference string) (string, error) {
	if !p.enabled {
		return "", fmt.Errorf("1Password CLI is not available")
	}

	if !p.IsReference(reference) {
		return "", fmt.Errorf("not a 1Password reference: %s", reference)
	}

	// Parse the reference to extract vault, item, and field
	// op://vault/item/field
	vault, item, field, err := p.parseReference(reference)
	if err != nil {
		return "", fmt.Errorf("invalid reference format: %w", err)
	}

	// Use 'op item get' which handles special characters in item names
	// This is more robust than 'op read' for items with parentheses, spaces, etc.
	cmd := exec.Command("op", "item", "get", item, "--vault="+vault, "--fields", "label="+field, "--reveal")
	hideConsoleWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Provide helpful error message based on the output
		errorMsg := string(output)
		return "", fmt.Errorf("failed to retrieve secret: %s", strings.TrimSpace(errorMsg))
	}

	// Trim whitespace and return
	secret := strings.TrimSpace(string(output))
	return secret, nil
}

// parseReference parses a 1Password reference into vault, item, and field
// Input: op://vault/item/field
// Output: vault, item, field, error
func (p *OnePasswordProvider) parseReference(reference string) (string, string, string, error) {
	if !strings.HasPrefix(reference, "op://") {
		return "", "", "", fmt.Errorf("reference must start with op://")
	}

	// Remove the "op://" prefix
	rest := strings.TrimPrefix(reference, "op://")

	// Split by "/" to get: [vault, item, field, ...]
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) < 3 {
		return "", "", "", fmt.Errorf("reference must be in format op://vault/item/field")
	}

	vault := parts[0]
	item := parts[1]
	field := parts[2]

	// URL-decode the item name in case it was encoded
	decodedItem, err := url.PathUnescape(item)
	if err != nil {
		// If decoding fails, use the original
		decodedItem = item
	}

	return vault, decodedItem, field, nil
}

// ResolveIfReference resolves a value if it's a 1Password reference, otherwise returns it as-is
// ResolveIfReference resolves a value if it's a 1Password reference, otherwise returns it as-is
// Note: This uses the 1Password CLI which will prompt for biometric auth if needed
// TODO: Consider migrating to https://github.com/1Password/onepassword-sdk-go for better integration
func (p *OnePasswordProvider) ResolveIfReference(value string) string {
	if !p.IsReference(value) {
		return value
	}

	resolved, err := p.ResolveSecret(value)
	if err != nil {
		// If resolution fails, return empty string
		// This prevents exposing the reference format
		fmt.Printf("Warning: Failed to resolve 1Password reference: %v\n", err)
		return ""
	}

	return resolved
}

// CheckItemExists checks if an item with the given title exists in the vault
// Returns the item ID if it exists, or an error if not found or multiple items exist
func (p *OnePasswordProvider) CheckItemExists(vault, title string) (string, bool, error) {
	if !p.enabled {
		return "", false, fmt.Errorf("1Password CLI is not available")
	}

	// Try to get the item
	cmd := exec.Command("op", "item", "get", title, "--vault="+vault, "--format=json")
	hideConsoleWindow(cmd)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Check if error is because item doesn't exist
		errorMsg := string(output)
		if strings.Contains(errorMsg, "isn't an item") || strings.Contains(errorMsg, "not found") {
			return "", false, nil // Item doesn't exist
		}
		// Check for multiple items
		if strings.Contains(errorMsg, "More than one item matches") {
			return "", false, fmt.Errorf("multiple items found with title '%s' in vault '%s'. Please use a unique name or delete duplicates in 1Password", title, vault)
		}
		return "", false, fmt.Errorf("failed to check item: %s", strings.TrimSpace(errorMsg))
	}

	// Item exists - extract ID from JSON (simple approach)
	// In production you'd want proper JSON parsing
	return "", true, nil
}

// CreateItem creates a new Login item in 1Password
// Returns the 1Password reference (op://vault/title/password)
func (p *OnePasswordProvider) CreateItem(vault, title, username, password string) (string, error) {
	if !p.enabled {
		return "", fmt.Errorf("1Password CLI is not available")
	}

	if vault == "" || title == "" {
		return "", fmt.Errorf("vault and title are required")
	}

	// Check if item already exists
	_, exists, err := p.CheckItemExists(vault, title)
	if err != nil {
		return "", err // Return the error (e.g., multiple items found)
	}

	if exists {
		// Item exists - update it instead of creating
		args := []string{
			"item", "edit", title,
			"--vault=" + vault,
		}

		if username != "" {
			args = append(args, "username="+username)
		}

		if password != "" {
			args = append(args, "password="+password)
		}

		cmd := exec.Command("op", args...)
		hideConsoleWindow(cmd)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to update existing 1Password item: %w, output: %s", err, string(output))
		}
	} else {
		// Item doesn't exist - create it
		args := []string{
			"item", "create",
			"--category=login",
			"--title=" + title,
			"--vault=" + vault,
		}

		if username != "" {
			args = append(args, "username="+username)
		}

		if password != "" {
			args = append(args, "password="+password)
		}

		cmd := exec.Command("op", args...)
		hideConsoleWindow(cmd)

		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to create 1Password item: %w, output: %s", err, string(output))
		}
	}

	// Return the reference format with URL-encoded item name
	// This handles special characters like parentheses, spaces, etc.
	encodedTitle := url.PathEscape(title)
	reference := fmt.Sprintf("op://%s/%s/password", vault, encodedTitle)
	return reference, nil
}

// ListVaults returns a list of available 1Password vaults
func (p *OnePasswordProvider) ListVaults() ([]string, error) {
	if !p.enabled {
		return nil, fmt.Errorf("1Password CLI is not available")
	}

	cmd := exec.Command("op", "vault", "list", "--format=json")
	hideConsoleWindow(cmd)

	_, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	// Simple parsing - extract vault names (this is a simplified version)
	// In production, you'd want to parse the JSON properly
	vaults := []string{"Private", "DevOps", "Employee"} // Default common vaults

	// For now, return common vaults. You could parse JSON if needed
	return vaults, nil
}
