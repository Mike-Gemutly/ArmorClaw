package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/armorclaw/bridge/pkg/keystore"
)

type KeyLister interface {
	List(provider keystore.Provider) ([]keystore.KeyInfo, error)
}

var (
	runtime     *AIRuntime
	runtimeOnce sync.Once
)

type AIRuntime struct {
	mu       sync.RWMutex
	provider string
	model    string
	keystore KeyLister
}

type AIConfig struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

func GetRuntime(ks KeyLister) *AIRuntime {
	runtimeOnce.Do(func() {
		loaded := loadAIConfig()
		runtime = &AIRuntime{
			keystore: ks,
			provider: loaded.Provider,
			model:    loaded.Model,
		}
	})
	return runtime
}

func loadAIConfig() AIConfig {
	data, err := os.ReadFile("/run/armorclaw/ai-config.json")
	if err != nil {
		return AIConfig{Provider: "openai", Model: "gpt-4o"}
	}
	var cfg AIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AIConfig{Provider: "openai", Model: "gpt-4o"}
	}
	return cfg
}

func (r *AIRuntime) GetCurrent() (provider, model string) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.provider, r.model
}

func (r *AIRuntime) ListProviders() ([]string, error) {
	keys, err := r.keystore.List(keystore.Provider(""))
	if err != nil {
		return nil, fmt.Errorf("failed to list keys: %w", err)
	}

	providerSet := make(map[string]bool)
	for _, k := range keys {
		providerSet[string(k.Provider)] = true
	}

	providers := make([]string, 0, len(providerSet))
	for p := range providerSet {
		providers = append(providers, p)
	}

	if len(providers) == 0 {
		providers = []string{"openai", "anthropic", "openrouter", "google", "xai"}
	}

	return providers, nil
}

func (r *AIRuntime) ListModels(provider string) ([]string, error) {
	keys, err := r.keystore.List(keystore.Provider(provider))
	if err != nil {
		return nil, fmt.Errorf("failed to list keys for provider %s: %w", provider, err)
	}

	models := make([]string, 0, len(keys))
	for _, k := range keys {
		models = append(models, k.ID)
	}

	if len(models) == 0 {
		models = r.getDefaultModels(provider)
	}

	return models, nil
}

func (r *AIRuntime) getDefaultModels(provider string) []string {
	defaults := map[string][]string{
		"openai":     {"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-3.5-turbo"},
		"anthropic":  {"claude-3-5-sonnet", "claude-3-opus", "claude-3-haiku"},
		"openrouter": {"anthropic/claude-3.5-sonnet", "openai/gpt-4o", "google/gemini-pro"},
		"google":     {"gemini-1.5-pro", "gemini-1.5-flash"},
		"xai":        {"grok-2", "grok-2-vision"},
	}
	if models, ok := defaults[provider]; ok {
		return models
	}
	return []string{"default"}
}

func (r *AIRuntime) SwitchProvider(provider, model string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	keys, err := r.keystore.List(keystore.Provider(provider))
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}

	if len(keys) == 0 {
		return fmt.Errorf("no API keys found for provider: %s", provider)
	}

	validModel := false
	for _, k := range keys {
		if model == "default" || model == k.ID {
			validModel = true
			break
		}
	}

	if !validModel && model != "default" {
		return fmt.Errorf("invalid model %s for provider %s", model, provider)
	}

	r.provider = provider
	if model != "default" {
		r.model = model
	}

	cfg := AIConfig{Provider: r.provider, Model: r.model}
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile("/run/armorclaw/ai-config.json", data, 0644); err != nil {
		return fmt.Errorf("failed to write ai config: %w", err)
	}

	return nil
}

func (r *AIRuntime) GetStatus() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return fmt.Sprintf("Provider: %s\nModel: %s", r.provider, r.model)
}
