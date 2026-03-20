// Package providers defines the embedded provider registry and fallback model lists.
package providers

// FallbackModels provides static model lists when Catwalk is unavailable.
// Each key is a provider ID, value is list of model IDs.
var FallbackModels = map[string][]ModelInfo{
	"openai": {
		{ID: "gpt-4.1", Name: "GPT-4.1", ContextSize: 0},
		{ID: "gpt-4o", Name: "GPT-4o", ContextSize: 128000},
		{ID: "gpt-4o-mini", Name: "GPT-4o Mini", ContextSize: 128000},
		{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", ContextSize: 128000},
		{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", ContextSize: 16385},
	},
	"anthropic": {
		{ID: "claude-3-opus", Name: "Claude 3 Opus", ContextSize: 200000},
		{ID: "claude-3-sonnet", Name: "Claude 3 Sonnet", ContextSize: 200000},
		{ID: "claude-3-haiku", Name: "Claude 3 Haiku", ContextSize: 200000},
	},
	"google": {
		{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", ContextSize: 1000000},
		{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", ContextSize: 1000000},
		{ID: "gemini-pro", Name: "Gemini Pro", ContextSize: 32760},
	},
	"xai": {
		{ID: "grok-2", Name: "Grok 2", ContextSize: 131072},
		{ID: "grok-2-vision", Name: "Grok 2 Vision", ContextSize: 32768},
		{ID: "grok-beta", Name: "Grok Beta", ContextSize: 131072},
	},
	"zhipu": {
		{ID: "glm-4", Name: "GLM-4", ContextSize: 128000},
		{ID: "glm-4v", Name: "GLM-4V", ContextSize: 128000},
		{ID: "glm-4-flash", Name: "GLM-4 Flash", ContextSize: 128000},
		{ID: "glm-3-turbo", Name: "GLM-3 Turbo", ContextSize: 128000},
	},
	"deepseek": {
		{ID: "deepseek-chat", Name: "DeepSeek Chat", ContextSize: 64000},
		{ID: "deepseek-coder", Name: "DeepSeek Coder", ContextSize: 64000},
	},
	"moonshot": {
		{ID: "moonshot-v1-8k", Name: "Moonshot V1 8K", ContextSize: 8192},
		{ID: "moonshot-v1-32k", Name: "Moonshot V1 32K", ContextSize: 32768},
		{ID: "moonshot-v1-128k", Name: "Moonshot V1 128K", ContextSize: 131072},
	},
	"groq": {
		{ID: "llama-3.1-70b-versatile", Name: "Llama 3.1 70B Versatile", ContextSize: 131072},
		{ID: "llama-3.1-8b-instant", Name: "Llama 3.1 8B Instant", ContextSize: 131072},
		{ID: "mixtral-8x7b-32768", Name: "Mixtral 8x7B", ContextSize: 32768},
	},
	"openrouter": {
		{ID: "openai/gpt-4o", Name: "OpenAI GPT-4o", ContextSize: 128000},
		{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", ContextSize: 200000},
		{ID: "google/gemini-pro-1.5", Name: "Gemini Pro 1.5", ContextSize: 1000000},
	},
	"nvidia": {
		{ID: "meta/llama-3.1-405b-instruct", Name: "Llama 3.1 405B", ContextSize: 128000},
		{ID: "meta/llama-3.1-70b-instruct", Name: "Llama 3.1 70B", ContextSize: 128000},
	},
	"cloudflare": {
		{ID: "@cf/meta/llama-3.1-8b-instruct", Name: "Llama 3.1 8B", ContextSize: 8192},
	},
	"ollama": {
		{ID: "llama3.2", Name: "Llama 3.2", ContextSize: 128000},
		{ID: "llama3.1", Name: "Llama 3.1", ContextSize: 128000},
		{ID: "mistral", Name: "Mistral", ContextSize: 32768},
	},
}

// ModelInfo represents a model's metadata.
type ModelInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ContextSize int    `json:"context_size,omitempty"`
}

// GetModels returns models for a provider, or empty slice if not found.
func GetModels(providerID string) []ModelInfo {
	return FallbackModels[providerID]
}
