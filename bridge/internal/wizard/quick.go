package wizard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
)

// apiProviderOption maps user-facing names to base URLs.
type apiProviderOption struct {
	Name    string
	Key     string
	BaseURL string
}

var apiProviders = []apiProviderOption{
	{Name: "OpenAI", Key: "openai", BaseURL: "https://api.openai.com/v1"},
	{Name: "Anthropic (Claude)", Key: "anthropic", BaseURL: "https://api.anthropic.com/v1"},
	{Name: "GLM-5 (Zhipu AI)", Key: "openai", BaseURL: "https://api.z.ai/api/coding/paas/v4"},
	{Name: "Custom (OpenAI-compatible)", Key: "custom", BaseURL: ""},
}

// runQuickStartForms collects Quick Start configuration through two form pages:
//
//   Page 1: AI provider selection + API key
//   Page 2: Admin password + deployment confirmation
//
// Secrets (API key, passwords) are stored in result.Secrets (memory only).
func runQuickStartForms(result *WizardResult, accessible bool) error {
	// --- Page 1: API Provider + Key ---
	var providerChoice string
	var apiKey string
	var customURL string

	providerOptions := make([]huh.Option[string], 0, len(apiProviders))
	for _, p := range apiProviders {
		providerOptions = append(providerOptions, huh.NewOption(p.Name, p.Key+"|"+p.BaseURL))
	}

	page1 := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("AI Provider").
				Description("Select your AI provider. The API key will be stored in the encrypted keystore.").
				Options(providerOptions...).
				Value(&providerChoice),

			huh.NewInput().
				Title("API Key").
				Description("Your provider's API key (e.g. sk-... or sk-ant-...).").
				EchoMode(huh.EchoModePassword).
				Validate(ValidateAPIKey).
				Value(&apiKey),
		).Title("Step 1 of 2: AI Provider Configuration"),
	)

	page1 = formOpts(page1, accessible)
	if err := page1.Run(); err != nil {
		return err
	}

	// Parse provider choice
	parts := strings.SplitN(providerChoice, "|", 2)
	providerKey := parts[0]
	baseURL := ""
	if len(parts) > 1 {
		baseURL = parts[1]
	}

	// If custom provider, ask for URL
	if providerKey == "custom" {
		customForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Custom API Base URL").
					Description("Enter the OpenAI-compatible API endpoint (e.g. https://your-provider.com/v1).").
					Placeholder("https://your-provider.com/v1").
					Validate(ValidateURL).
					Value(&customURL),
			),
		)
		customForm = formOpts(customForm, accessible)
		if err := customForm.Run(); err != nil {
			return err
		}
		baseURL = customURL
		providerKey = "openai" // custom providers use the OpenAI-compatible interface
	}

	result.Config.APIProvider = providerKey
	result.Config.APIBaseURL = baseURL
	result.Secrets.APIKey = strings.TrimSpace(apiKey)

	// --- Page 2: Admin Password + Confirm ---
	var adminPassword string
	var confirmDeploy bool

	page2 := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Admin Password").
				Description("Password for Element X / ArmorChat login. Leave empty to auto-generate.").
				EchoMode(huh.EchoModePassword).
				Placeholder("(press Enter to auto-generate)").
				Validate(ValidatePassword).
				Value(&adminPassword),

			huh.NewConfirm().
				Title("Ready to deploy?").
				Description("ArmorClaw will configure the Matrix homeserver, generate SSL certificates, and start the bridge.").
				Affirmative("Deploy").
				Negative("Cancel").
				Value(&confirmDeploy),
		).Title("Step 2 of 2: Admin & Deployment"),
	)

	page2 = formOpts(page2, accessible)
	if err := page2.Run(); err != nil {
		return err
	}

	if !confirmDeploy {
		return huh.ErrUserAborted
	}

	// Auto-generate password if empty
	if adminPassword == "" {
		generated, err := generatePassword(16)
		if err != nil {
			return fmt.Errorf("generate admin password: %w", err)
		}
		adminPassword = generated
		fmt.Printf("  Generated admin password: %s\n", adminPassword)
		fmt.Println("  Save this now — it will not be shown again.")
		fmt.Println()
	}

	result.Secrets.AdminPassword = adminPassword

	// Generate bridge password (never user-facing in quick mode)
	bridgePass, err := generatePassword(16)
	if err != nil {
		return fmt.Errorf("generate bridge password: %w", err)
	}
	result.Secrets.BridgePassword = bridgePass

	return nil
}

// generatePassword creates a random URL-safe password of the given length.
func generatePassword(length int) (string, error) {
	// Generate enough random bytes, then base64-encode and trim
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	encoded := base64.URLEncoding.EncodeToString(buf)
	// Remove padding and trim to desired length
	encoded = strings.TrimRight(encoded, "=")
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded, nil
}
