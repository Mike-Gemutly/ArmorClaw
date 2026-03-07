package wizard

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/armorclaw/bridge/pkg/setup"
	"github.com/charmbracelet/huh"
)

// apiProviderOption maps user-facing names to base URLs.
type apiProviderOption struct {
	Name    string
	Key     string
	BaseURL string
}

// fallbackModels provides static model lists when Catwalk is unavailable.
var fallbackModels = map[string][]string{
	"openai":     {"gpt-4.1", "gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"},
	"anthropic":  {"claude-3-opus", "claude-3-sonnet", "claude-3-haiku"},
	"google":     {"gemini-1.5-pro", "gemini-1.5-flash", "gemini-pro"},
	"xai":        {"grok-2", "grok-2-vision", "grok-beta"},
	"zhipu":      {"glm-4", "glm-4v", "glm-4-flash", "glm-3-turbo"},
	"deepseek":   {"deepseek-chat", "deepseek-coder"},
	"moonshot":   {"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"},
	"groq":       {"llama-3.1-70b-versatile", "llama-3.1-8b-instant", "mixtral-8x7b-32768"},
	"openrouter": {"openai/gpt-4o", "anthropic/claude-3.5-sonnet", "google/gemini-pro-1.5"},
}

// getProviderOptions returns provider options, first trying Catwalk then falling back to hardcoded list.
func getProviderOptions() ([]apiProviderOption, bool) {
	catwalk := NewCatwalkClient("http://localhost:8080")

	if catwalk.IsAvailable() {
		providers, err := catwalk.FetchProviders()
		if err == nil && len(providers) > 0 {
			fmt.Println("  [Catwalk] Discovered providers from local registry")
			return ProvidersFromCatwalk(providers), true
		}
	}

	fmt.Println("  [Wizard] Using default provider list")
	return apiProviders, false
}

// getModelOptions returns available models for a provider.
// First tries Catwalk, then falls back to static list.
func getModelOptions(providerKey string) []string {
	catwalk := NewCatwalkClient("http://localhost:8080")

	// Try to fetch from Catwalk
	if catwalk.IsAvailable() {
		providers, err := catwalk.FetchProviders()
		if err == nil {
			for _, p := range providers {
				if p.ID == providerKey && len(p.Models) > 0 {
					models := make([]string, len(p.Models))
					for i, m := range p.Models {
						models[i] = m.ID
					}
					fmt.Println("  [Catwalk] Discovered models for provider:", providerKey)
					return models
				}
			}
		}
	}

	// Fallback to static list
	if models, ok := fallbackModels[providerKey]; ok {
		fmt.Println("  [Wizard] Using default model list for provider:", providerKey)
		return models
	}

	// No models known - return generic default
	return []string{"default"}
}

var apiProviders = []apiProviderOption{
	{Name: "OpenAI", Key: "openai", BaseURL: "https://api.openai.com/v1"},
	{Name: "Anthropic (Claude)", Key: "anthropic", BaseURL: "https://api.anthropic.com/v1"},
	{Name: "Google (Gemini)", Key: "google", BaseURL: "https://generativelanguage.googleapis.com/v1"},
	{Name: "xAI (Grok)", Key: "xai", BaseURL: "https://api.x.ai/v1"},
	{Name: "Zhipu AI (GLM)", Key: "openai", BaseURL: "https://open.bigmodel.cn/api/paas/v4"},
	{Name: "DeepSeek", Key: "openai", BaseURL: "https://api.deepseek.com/v1"},
	{Name: "Moonshot AI", Key: "openai", BaseURL: "https://api.moonshot.cn/v1"},
	{Name: "NVIDIA NIM", Key: "openai", BaseURL: "https://integrate.api.nvidia.com/v1"},
	{Name: "OpenRouter", Key: "openai", BaseURL: "https://openrouter.ai/api/v1"},
	{Name: "Groq", Key: "openai", BaseURL: "https://api.groq.com/openai/v1"},
	{Name: "Cloudflare AI Gateway", Key: "openai", BaseURL: "https://gateway.ai.cloudflare.com/v1"},
	{Name: "Custom (OpenAI-compatible)", Key: "custom", BaseURL: ""},
}

// runQuickStartForms collects Quick Start configuration through two form pages:
//
//	Page 1: AI provider selection + API key
//	Page 2: Admin password + deployment confirmation
//
// Secrets (API key, passwords) are stored in result.Secrets (memory only).
func runQuickStartForms(result *WizardResult, accessible bool) error {
	// --- Page 1: API Provider + Key ---
	var providerChoice string
	var apiKey string
	var customURL string

	providers, fromCatwalk := getProviderOptions()
	_ = fromCatwalk // Could be used for logging/debugging

	providerOptions := make([]huh.Option[string], 0, len(providers))
	for _, p := range providers {
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
		return wrapFormError(err, "Step 1: AI Provider Configuration", "page1.Run")
	}

	// Parse provider choice
	parts := strings.SplitN(providerChoice, "|", 2)
	if len(parts) < 2 {
		return &setup.SetupError{
			Code:     "INS-005",
			Title:    "Invalid provider selection",
			Cause:    fmt.Sprintf("Provider choice did not contain expected format 'key|url', got: %q", providerChoice),
			Fix:      []string{"Try running setup again", "If the issue persists, report this as a bug"},
			ExitCode: 1,
		}
	}
	providerKey := parts[0]
	baseURL := parts[1]

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
			return wrapFormError(err, "Custom Provider URL", "customForm.Run")
		}
		baseURL = customURL
		providerKey = "openai" // custom providers use the OpenAI-compatible interface
	}

	// --- Model Selection ---
	var modelChoice string
	models := getModelOptions(providerKey)

	if len(models) > 0 {
		modelOptions := make([]huh.Option[string], 0, len(models))
		for _, m := range models {
			modelOptions = append(modelOptions, huh.NewOption(m, m))
		}

		modelForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("AI Model").
					Description("Select the default model to use with this provider.").
					Options(modelOptions...).
					Value(&modelChoice),
			).Title("Model Selection"),
		)

		modelForm = formOpts(modelForm, accessible)
		if err := modelForm.Run(); err != nil {
			// Soft-fail: use first model as default
			modelChoice = models[0]
			fmt.Printf("  Using default model: %s\n", modelChoice)
		}
	} else {
		modelChoice = "default"
	}

	result.Config.APIProvider = providerKey
	result.Config.APIBaseURL = baseURL
	result.Config.DefaultModel = modelChoice
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
		return wrapFormError(err, "Step 2: Admin & Deployment", "page2.Run")
	}

	if !confirmDeploy {
		return huh.ErrUserAborted
	}

	// Auto-generate password if empty
	if adminPassword == "" {
		generated, err := generatePassword(16)
		if err != nil {
			return &setup.SetupError{
				Code:     "INS-003",
				Title:    "Failed to generate admin password",
				Cause:    fmt.Sprintf("Crypto/rand error: %v (function: generatePassword)", err),
				Fix:      []string{"This is likely a system issue - try restarting the container", "If the issue persists, set ARMORCLAW_ADMIN_PASSWORD env var manually"},
				ExitCode: 1,
			}
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
		return &setup.SetupError{
			Code:     "INS-003",
			Title:    "Failed to generate bridge password",
			Cause:    fmt.Sprintf("Crypto/rand error: %v (function: generatePassword)", err),
			Fix:      []string{"This is likely a system issue - try restarting the container"},
			ExitCode: 1,
		}
	}
	result.Secrets.BridgePassword = bridgePass

	return nil
}

// wrapFormError wraps a Huh? form error with context about which step failed.
func wrapFormError(err error, stepName, functionName string) error {
	if err == huh.ErrUserAborted {
		return err // Don't wrap user abort
	}

	return &setup.SetupError{
		Code:  "INS-002",
		Title: fmt.Sprintf("Form error in %s", stepName),
		Cause: fmt.Sprintf("Function %s failed: %v", functionName, err),
		Fix: []string{
			"Try running setup again",
			"If using a limited terminal, try: -e ARMORCLAW_ACCESSIBLE=true",
			"Or use environment variables for non-interactive setup",
		},
		ExitCode: 1,
	}
}

// generatePassword creates a random URL-safe password of the given length.
func generatePassword(length int) (string, error) {
	// Generate enough random bytes, then base64-encode and trim
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("crypto/rand.Read failed: %w", err)
	}
	encoded := base64.URLEncoding.EncodeToString(buf)
	// Remove padding and trim to desired length
	encoded = strings.TrimRight(encoded, "=")
	if len(encoded) > length {
		encoded = encoded[:length]
	}
	return encoded, nil
}
