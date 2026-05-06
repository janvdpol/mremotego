package secrets

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

type opField struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// commandRunner executes a named command and returns its combined output.
// Extracted so ResolveMultiple can be tested without a real op binary.
type commandRunner func(name string, args ...string) ([]byte, error)

func realCommandRunner(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	hideConsoleWindow(cmd)
	return cmd.CombinedOutput()
}

// OnePasswordProvider handles retrieving secrets from 1Password CLI
type OnePasswordProvider struct {
	enabled bool
	run     commandRunner
}

// NewOnePasswordProvider creates a new 1Password provider
func NewOnePasswordProvider() *OnePasswordProvider {
	return &OnePasswordProvider{
		enabled: isOnePasswordCLIAvailable(),
		run:     realCommandRunner,
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

// IsReference checks if a string is a 1Password reference (starts with op://)
func (p *OnePasswordProvider) IsReference(value string) bool {
	return strings.HasPrefix(value, "op://")
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

// ResolveMultiple resolves a slice of 1Password references. References from the
// same vault+item are fetched in a single op call. Non-references are returned unchanged.
// Returns a map of reference → value.
func (p *OnePasswordProvider) ResolveMultiple(refs []string) (map[string]string, error) {
	if !p.enabled {
		return nil, fmt.Errorf("1Password CLI is not available")
	}

	result := make(map[string]string, len(refs))

	type itemKey struct{ vault, item string }
	type fieldReq struct{ ref, field string }

	grouped := make(map[itemKey][]fieldReq)
	seen := make(map[string]bool)

	for _, ref := range refs {
		if !p.IsReference(ref) {
			result[ref] = ref
			continue
		}
		if seen[ref] {
			continue
		}
		seen[ref] = true

		vault, item, field, err := p.parseReference(ref)
		if err != nil {
			return nil, fmt.Errorf("invalid 1Password reference %q: %w", ref, err)
		}
		key := itemKey{vault, item}
		grouped[key] = append(grouped[key], fieldReq{ref: ref, field: field})
	}

	for key, fields := range grouped {
		labels := make([]string, len(fields))
		for i, f := range fields {
			labels[i] = "label=" + f.field
		}

		output, err := p.run("op", "item", "get", key.item,
			"--vault="+key.vault,
			"--fields", strings.Join(labels, ","),
			"--format", "json",
			"--reveal",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve secrets for item %q in vault %q: %s", key.item, key.vault, strings.TrimSpace(string(output)))
		}

		// op returns an array when multiple fields are requested, or a single object for one field.
		// Normalise to a slice in both cases.
		var parsed []opField
		if err := json.Unmarshal(output, &parsed); err != nil {
			var single opField
			if err2 := json.Unmarshal(output, &single); err2 != nil {
				return nil, fmt.Errorf("failed to parse op output for item %q: %w", key.item, err)
			}
			parsed = []opField{single}
		}

		byLabel := make(map[string]string, len(parsed))
		for _, f := range parsed {
			byLabel[strings.ToLower(f.Label)] = f.Value
		}

		for _, f := range fields {
			val, ok := byLabel[strings.ToLower(f.field)]
			if !ok {
				return nil, fmt.Errorf("field %q not found in 1Password item %q (vault %q)", f.field, key.item, key.vault)
			}
			result[f.ref] = val
		}
	}

	return result, nil
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

