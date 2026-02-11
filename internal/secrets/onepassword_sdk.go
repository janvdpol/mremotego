package secrets

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/1password/onepassword-sdk-go"
)

// OnePasswordSDKProvider handles retrieving secrets using the 1Password SDK
// This provides native desktop app integration with biometric unlock
// Falls back to CLI provider if SDK is not available
type OnePasswordSDKProvider struct {
	client       *onepassword.Client
	accountName  string
	enabled      bool
	vaults       []VaultInfo          // Stores vault ID and title (title may be [Encrypted] until auth)
	vaultNameMap map[string]string    // Fallback mapping for encrypted vault names (SDK v0.4.1+ decrypts automatically)
	cliProvider  *OnePasswordProvider // Fallback to CLI if SDK not available
}

// VaultInfo holds vault information
type VaultInfo struct {
	ID    string
	Title string
}

// NewOnePasswordSDKProvider creates a new 1Password SDK provider with desktop app integration
// accountName should be the account name shown at the top of the 1Password desktop app sidebar
// If SDK initialization fails, it automatically falls back to the CLI provider
func NewOnePasswordSDKProvider(accountName string) *OnePasswordSDKProvider {
	provider := &OnePasswordSDKProvider{
		accountName: accountName,
		enabled:     false,
		cliProvider: NewOnePasswordProvider(), // Initialize CLI fallback
	}

	// Validate input
	if accountName == "" {
		fmt.Println("[1Password SDK] ⚠️  Account name required. Checking CLI fallback...")
		if provider.cliProvider.IsEnabled() {
			fmt.Println("[1Password] ✅ Falling back to CLI provider (op)")
			return provider // Will use CLI provider
		}
		fmt.Println("[1Password] ❌ Neither SDK nor CLI available. 1Password integration disabled.")
		return provider
	}

	// Try to initialize the SDK client with desktop app integration
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fmt.Printf("[1Password SDK] Connecting to account: %s\n", accountName)
	client, err := onepassword.NewClient(
		ctx,
		onepassword.WithDesktopAppIntegration(accountName),
		onepassword.WithIntegrationInfo("MremoteGO", "v2.0.0"),
	)

	if err != nil {
		// SDK initialization failed - desktop app integration not available
		// Try to fall back to CLI provider
		fmt.Printf("[1Password SDK] ❌ Failed to initialize: %v\n", err)

		if provider.cliProvider.IsEnabled() {
			fmt.Println("[1Password] ✅ Falling back to CLI provider (op)")
			fmt.Println("")
			return provider // Will use CLI provider for operations
		}

		// Neither SDK nor CLI available
		fmt.Println("")
		fmt.Println("To enable 1Password integration, choose one:")
		fmt.Println("")
		fmt.Println("OPTION 1: Use 1Password SDK (Desktop App)")
		fmt.Println("  1. Install 1Password desktop app (BETA) and ensure it's running")
		fmt.Println("  2. Settings → Developer → 'Integrate with the 1Password SDKs' enabled")
		fmt.Println("  3. Settings → Developer → 'Integrate with other apps' enabled")
		fmt.Println("  4. Verify the correct account name (check top of 1Password sidebar)")
		fmt.Println("")
		fmt.Println("OPTION 2: Use 1Password CLI")
		fmt.Println("  1. Install 1Password CLI (op): https://developer.1password.com/docs/cli/get-started/")
		fmt.Println("  2. Sign in: op signin")
		fmt.Println("")
		return provider
	}

	provider.client = client
	provider.enabled = true
	provider.vaults = make([]VaultInfo, 0)
	fmt.Println("[1Password SDK] ✅ Successfully initialized with desktop app integration!")

	// List available vaults for debugging and store vault info
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel2()
	vaults, err := client.Vaults().List(ctx2)
	if err == nil && len(vaults) > 0 {
		fmt.Printf("[1Password SDK] Found %d vault(s):\n", len(vaults))
		for _, v := range vaults {
			fmt.Printf("  • %s (ID: %s)\n", v.Title, v.ID)
			provider.vaults = append(provider.vaults, VaultInfo{
				ID:    v.ID,
				Title: v.Title,
			})
		}

		// Force authentication by attempting to read actual secret data
		// This triggers biometric auth which should decrypt vault names
		fmt.Println("[1Password SDK] Triggering authentication to decrypt vault names...")
		ctxAuth, cancelAuth := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelAuth()

		// Try to list items and read the first item's password field to force full auth
		items, itemsErr := client.Items().List(ctxAuth, vaults[0].ID)
		fmt.Printf("[DEBUG] Items list error: %v, items count: %d\n", itemsErr, len(items))
		if itemsErr == nil && len(items) > 0 {
			fmt.Printf("[DEBUG] First item ID: %s\n", items[0].ID)
			// Try to get the first item with its fields to trigger biometric unlock
			ctxItem, cancelItem := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancelItem()
			item, itemErr := client.Items().Get(ctxItem, vaults[0].ID, items[0].ID)
			fmt.Printf("[DEBUG] Item get error: %v, fields count: %d\n", itemErr, len(item.Fields))
			if itemErr == nil && len(item.Fields) > 0 {
				// Access a field value to force decryption
				fmt.Printf("[DEBUG] First field: %+v\n", item.Fields[0])
				_ = item.Fields[0].Value
				fmt.Println("[1Password SDK] Biometric authentication triggered")
			}
		} else if itemsErr != nil {
			fmt.Printf("[DEBUG] Failed to list items: %v\n", itemsErr)
		} else {
			fmt.Println("[DEBUG] No items found in first vault")
		}

		// Re-list vaults to get decrypted names
		ctx3, cancel3 := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel3()
		vaultsDecrypted, err := client.Vaults().List(ctx3)
		if err == nil && len(vaultsDecrypted) > 0 {
			// Update our stored vault info with decrypted names
			provider.vaults = make([]VaultInfo, 0)
			fmt.Println("[1Password SDK] Vault names after authentication:")
			for i, v := range vaultsDecrypted {
				fmt.Printf("[DEBUG] Vault %d: %+v\n", i, v)
				fmt.Printf("  • %s (ID: %s)\n", v.Title, v.ID)
				provider.vaults = append(provider.vaults, VaultInfo{
					ID:    v.ID,
					Title: v.Title,
				})
			}
		}
	}

	return provider
}

// IsEnabled returns whether the 1Password SDK or CLI is available
func (p *OnePasswordSDKProvider) IsEnabled() bool {
	// Check if SDK is enabled first
	if p.enabled {
		return true
	}
	// Fall back to CLI provider
	if p.cliProvider != nil && p.cliProvider.IsEnabled() {
		return true
	}
	return false
}

// IsAuthenticated checks if the user is currently authenticated with 1Password
// For SDK-based integration, we check if we can list vaults
// For CLI fallback, we check CLI authentication
func (p *OnePasswordSDKProvider) IsAuthenticated() bool {
	if p.enabled && p.client != nil {
		// Try to list vaults as an auth check
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := p.client.Vaults().List(ctx)
		return err == nil
	}

	// Fall back to CLI provider
	if p.cliProvider != nil && p.cliProvider.IsEnabled() {
		return p.cliProvider.IsAuthenticated()
	}
	return false
}

// GetAuthenticationInstructions returns instructions for authenticating with 1Password SDK
func (p *OnePasswordSDKProvider) GetAuthenticationInstructions() string {
	// Check if CLI is available as fallback
	cliAvailable := p.cliProvider != nil && p.cliProvider.IsEnabled()
	cliAuthenticated := cliAvailable && p.cliProvider.IsAuthenticated()

	baseInstructions := ""
	if p.accountName == "" {
		baseInstructions = `MremoteGO supports two ways to connect to 1Password:

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔷 OPTION 1: Desktop App (Recommended - Biometric Auth)

1. Install the 1Password desktop app (BETA version required)
   Download from: https://releases.1password.com/

2. Open 1Password and sign in to your account

3. Enable SDK Integration:
   • Go to Settings → Developer
   • Turn on "Integrate with the 1Password SDKs"
   • Turn on "Integrate with other apps"

4. In your config.yaml, set:
   settings:
     onePasswordAccount: "Your Account Name"
   (Find this at the top of the 1Password sidebar)

5. Restart MremoteGO and unlock with biometrics!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

� OPTION 2: CLI (Fallback - Session Token)

1. Install 1Password CLI:
   https://developer.1password.com/docs/cli/get-started/

2. Sign in:
   op signin

3. Launch MremoteGO from the same terminal
   (Inherits your session token automatically)`
	} else {
		baseInstructions = fmt.Sprintf(`MremoteGO supports two ways to connect to 1Password:

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

🔷 OPTION 1: Desktop App (Configured: %s)

1. Install the 1Password desktop app (BETA version required)
   Download from: https://releases.1password.com/

2. Open 1Password and sign in to: %s

3. Enable SDK Integration:
   • Go to Settings → Developer
   • Turn on "Integrate with the 1Password SDKs"
   • Turn on "Integrate with other apps"

4. Restart MremoteGO and unlock with biometrics!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

� OPTION 2: CLI (Fallback - Session Token)

1. Install 1Password CLI:
   https://developer.1password.com/docs/cli/get-started/

2. Sign in:
   op signin

3. Launch MremoteGO from the same terminal
   (Inherits your session token automatically)`, p.accountName, p.accountName)
	}

	// Add status information
	statusInfo := "\n\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n📊 Current Status:\n"

	if p.enabled {
		statusInfo += "   ✅ SDK: Connected (using desktop app)\n"
	} else {
		statusInfo += "   ❌ SDK: Not available\n"
	}

	if cliAvailable {
		if cliAuthenticated {
			statusInfo += "   ✅ CLI: Available and authenticated\n"
		} else {
			statusInfo += "   ⚠️  CLI: Available but not authenticated (run 'op signin')\n"
		}
	} else {
		statusInfo += "   ❌ CLI: Not installed\n"
	}

	return baseInstructions + statusInfo
}

// IsReference checks if a string is a 1Password reference (starts with op://)
func (p *OnePasswordSDKProvider) IsReference(value string) bool {
	return strings.HasPrefix(value, "op://")
}

// ResolveSecret retrieves a secret from 1Password using the SDK or CLI fallback
// Reference format: op://vault/item/field
// Example: op://Private/MyServer/password
func (p *OnePasswordSDKProvider) ResolveSecret(reference string) (string, error) {
	if !p.IsReference(reference) {
		return "", fmt.Errorf("not a 1Password reference: %s", reference)
	}

	// Try SDK first
	if p.enabled && p.client != nil {
		// Use SDK's Resolve method which handles the full reference format
		// This includes special characters, URL encoding, etc.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		secret, err := p.client.Secrets().Resolve(ctx, reference)
		if err != nil {
			// Check for authentication errors
			if strings.Contains(err.Error(), "not authenticated") || strings.Contains(err.Error(), "authorization") {
				return "", fmt.Errorf("not authenticated with 1Password - please unlock the desktop app: %w", err)
			}
			return "", fmt.Errorf("failed to retrieve secret from 1Password SDK: %w", err)
		}

		return secret, nil
	}

	// Fall back to CLI provider
	if p.cliProvider != nil && p.cliProvider.IsEnabled() {
		return p.cliProvider.ResolveSecret(reference)
	}

	return "", fmt.Errorf("1Password is not available - neither SDK nor CLI is configured")
}

// ResolveIfReference resolves a value if it's a 1Password reference, otherwise returns it as-is
// This uses the 1Password SDK with desktop app integration (biometric auth support)
func (p *OnePasswordSDKProvider) ResolveIfReference(value string) string {
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

// parseReference parses a 1Password reference into vault, item, and field
// Input: op://vault/item/field
// Output: vault, item, field, error
func (p *OnePasswordSDKProvider) parseReference(reference string) (string, string, string, error) {
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

// CreateItem creates a new Login item in 1Password using the SDK
// Returns the 1Password reference (op://vault/title/password)
func (p *OnePasswordSDKProvider) CreateItem(vault, title, username, password string) (string, error) {
	if !p.enabled || p.client == nil {
		return "", fmt.Errorf("1Password SDK is not available")
	}

	if vault == "" || title == "" {
		return "", fmt.Errorf("vault and title are required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// CRITICAL: Vault names are encrypted until biometric auth happens
	// We need to trigger auth by actually accessing secret data (not just listing vaults)
	// Try to resolve any secret reference to force biometric prompt
	fmt.Println("[1Password SDK] Triggering biometric authentication...")

	// First vault list will show [Encrypted] names
	vaultsBeforeAuth, _ := p.client.Vaults().List(ctx)
	if len(vaultsBeforeAuth) > 0 {
		// Try to list items in the first vault to force auth
		// This will trigger the biometric prompt
		_, _ = p.client.Items().List(ctx, vaultsBeforeAuth[0].ID)
	}

	// NOW list vaults again - names should be decrypted after auth
	vaults, err := p.client.Vaults().List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list vaults: %w", err)
	}

	// List vault titles for debugging
	fmt.Printf("[1Password SDK] Available vaults after auth:\n")
	for _, v := range vaults {
		fmt.Printf("  • '%s' (ID: %s)\n", v.Title, v.ID)
	}

	// Try to find vault by title first, then by ID as fallback
	var vaultID string
	for _, v := range vaults {
		if v.Title == vault || v.ID == vault {
			vaultID = v.ID
			break
		}
	}

	if vaultID == "" {
		// If we still can't find it, vault names might be encrypted
		// Provide helpful error with available info
		availableInfo := make([]string, len(vaults))
		for i, v := range vaults {
			if v.Title == "[Encrypted]" {
				availableInfo[i] = fmt.Sprintf("ID:%s", v.ID)
			} else {
				availableInfo[i] = fmt.Sprintf("'%s'", v.Title)
			}
		}
		return "", fmt.Errorf("vault '%s' not found. Available vaults: %v", vault, availableInfo)
	}

	// Create the login item with username and password fields
	itemParams := onepassword.ItemCreateParams{
		Title:    title,
		Category: onepassword.ItemCategoryLogin,
		VaultID:  vaultID,
		Fields:   []onepassword.ItemField{},
	}

	if username != "" {
		itemParams.Fields = append(itemParams.Fields, onepassword.ItemField{
			ID:        "username",
			Title:     "username",
			Value:     username,
			FieldType: onepassword.ItemFieldTypeText,
		})
	}

	if password != "" {
		itemParams.Fields = append(itemParams.Fields, onepassword.ItemField{
			ID:        "password",
			Title:     "password",
			Value:     password,
			FieldType: onepassword.ItemFieldTypeConcealed,
		})
	}

	_, err = p.client.Items().Create(ctx, itemParams)
	if err != nil {
		return "", fmt.Errorf("failed to create 1Password item: %w", err)
	}

	// Return the reference format with URL-encoded item name
	encodedTitle := url.PathEscape(title)
	reference := fmt.Sprintf("op://%s/%s/password", vault, encodedTitle)
	return reference, nil
}

// ListVaults returns a list of available 1Password vaults
func (p *OnePasswordSDKProvider) ListVaults() ([]string, error) {
	if !p.enabled || p.client == nil {
		return nil, fmt.Errorf("1Password SDK is not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vaults, err := p.client.Vaults().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list vaults: %w", err)
	}

	vaultNames := make([]string, 0, len(vaults))
	for _, vault := range vaults {
		vaultNames = append(vaultNames, vault.Title)
	}

	return vaultNames, nil
}

// GetVaults returns vault information (ID and title)
// With SDK v0.4.1-beta.1+: Vault names are decrypted after biometric authentication
// Manual vault name mappings are used only as fallback for encrypted names
func (p *OnePasswordSDKProvider) GetVaults() []VaultInfo {
	if !p.enabled {
		return []VaultInfo{}
	}

	// Prefer SDK-provided decrypted names, use manual mapping only for encrypted names
	vaults := make([]VaultInfo, 0, len(p.vaults))
	for _, v := range p.vaults {
		title := v.Title

		// If vault name is still encrypted, try to use manual mapping
		if title == "[Encrypted]" && len(p.vaultNameMap) > 0 {
			if friendlyName, ok := p.vaultNameMap[v.ID]; ok {
				title = friendlyName
			}
		}

		vaults = append(vaults, VaultInfo{
			ID:    v.ID,
			Title: title,
		})
	}
	return vaults
}

// SetVaultNameMap sets the vault ID to friendly name mapping from config
// Note: With SDK v0.4.1-beta.1+, manual mapping is only needed as fallback for encrypted names
// The SDK now provides decrypted vault names after biometric authentication
func (p *OnePasswordSDKProvider) SetVaultNameMap(nameMap map[string]string) {
	p.vaultNameMap = nameMap
}

// GetVaultNameMap returns the current vault name mapping
func (p *OnePasswordSDKProvider) GetVaultNameMap() map[string]string {
	return p.vaultNameMap
}

// RefreshVaultNames attempts to decrypt vault names by triggering authentication
// Returns true if vault names were successfully decrypted
func (p *OnePasswordSDKProvider) RefreshVaultNames() bool {
	if !p.enabled || p.client == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to list vaults - this may trigger biometric auth
	vaults, err := p.client.Vaults().List(ctx)
	if err != nil {
		return false
	}

	// Check if any vault names are still encrypted
	allDecrypted := true
	for i, v := range vaults {
		if i < len(p.vaults) {
			p.vaults[i].Title = v.Title
		}
		if v.Title == "[Encrypted]" || v.Title == "" {
			allDecrypted = false
		}
	}

	return allDecrypted
}

// GetVaultIDs returns just the vault IDs (for compatibility)
func (p *OnePasswordSDKProvider) GetVaultIDs() []string {
	if !p.enabled {
		return []string{}
	}

	ids := make([]string, 0, len(p.vaults))
	for _, v := range p.vaults {
		ids = append(ids, v.ID)
	}
	return ids
}

// CheckItemExists checks if an item with the given title exists in the vault
// Returns the item ID if it exists, or an error if not found
func (p *OnePasswordSDKProvider) CheckItemExists(vault, title string) (string, bool, error) {
	if !p.enabled || p.client == nil {
		return "", false, fmt.Errorf("1Password SDK is not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get the vault ID
	vaults, err := p.client.Vaults().List(ctx)
	if err != nil {
		return "", false, fmt.Errorf("failed to list vaults: %w", err)
	}

	var vaultID string
	for _, v := range vaults {
		if v.Title == vault {
			vaultID = v.ID
			break
		}
	}

	if vaultID == "" {
		return "", false, fmt.Errorf("vault '%s' not found", vault)
	}

	// List items in the vault and search for matching title
	items, err := p.client.Items().List(ctx, vaultID)
	if err != nil {
		return "", false, fmt.Errorf("failed to list items: %w", err)
	}

	for _, item := range items {
		if item.Title == title {
			return item.ID, true, nil
		}
	}

	return "", false, nil
}
