// Package providers defines the embedded provider registry and fallback model lists.
package providers

// FallbackModels provides static model lists when Catwalk is unavailable.
// Each key is a provider ID, value is list of model IDs.
var FallbackModels = map[string][]ModelInfo{
	"openai": {
		{ID: "gpt-5.4-pro", Name: "GPT-5.4 Pro", ContextSize: 400000},
		{ID: "gpt-5.4-thinking", Name: "GPT-5.4 Thinking", ContextSize: 400000},
		{ID: "gpt-5.4-mini", Name: "GPT-5.4 Mini", ContextSize: 400000},
		{ID: "gpt-5.4-nano", Name: "GPT-5.4 Nano", ContextSize: 128000},
	},
	"anthropic": {
		{ID: "claude-4.6-opus", Name: "Claude 4.6 Opus", ContextSize: 1000000},
		{ID: "claude-4.6-sonnet", Name: "Claude 4.6 Sonnet", ContextSize: 1000000},
		{ID: "claude-4.5-haiku", Name: "Claude 4.5 Haiku", ContextSize: 200000},
	},
	"google": {
		{ID: "gemini-3.1-pro", Name: "Gemini 3.1 Pro", ContextSize: 2000000},
		{ID: "gemini-3-flash", Name: "Gemini 3 Flash", ContextSize: 1000000},
		{ID: "gemini-3.1-flash-lite", Name: "Gemini 3.1 Flash-Lite", ContextSize: 1000000},
	},
	"xai": {
		{ID: "grok-4.20", Name: "Grok 4.20", ContextSize: 2000000},
		{ID: "grok-4-fast", Name: "Grok 4 Fast", ContextSize: 131072},
		{ID: "grok-code-fast-1", Name: "Grok Code Fast 1", ContextSize: 131072},
	},
	"zhipu": {
		{ID: "glm-5", Name: "GLM-5", ContextSize: 128000},
		{ID: "glm-4.7-flash", Name: "GLM-4.7 Flash", ContextSize: 128000},
		{ID: "glm-4", Name: "GLM-4", ContextSize: 128000},
		{ID: "glm-4v", Name: "GLM-4V", ContextSize: 128000},
		{ID: "glm-4-flash", Name: "GLM-4 Flash", ContextSize: 128000},
		{ID: "glm-3-turbo", Name: "GLM-3 Turbo", ContextSize: 128000},
	},
	"deepseek": {
		{ID: "deepseek-v4", Name: "DeepSeek V4", ContextSize: 1000000},
		{ID: "deepseek-r1-0528", Name: "DeepSeek R1", ContextSize: 164000},
		{ID: "deepseek-v3-0324", Name: "DeepSeek V3", ContextSize: 131000},
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
