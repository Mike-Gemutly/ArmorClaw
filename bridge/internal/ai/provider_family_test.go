package ai

import "testing"

func TestProviderFamily(t *testing.T) {
	tests := []struct {
		name     string
		provider ProviderType
		baseURL  string
		want     ProviderFamily
		wantErr  bool
	}{
		// Supported: OpenAI-compatible
		{"openai", ProviderOpenAI, "", ProviderFamilyOpenAICompatible, false},
		{"xai", ProviderXAI, "", ProviderFamilyOpenAICompatible, false},
		{"openrouter", ProviderOpenRouter, "", ProviderFamilyOpenAICompatible, false},
		{"deepseek", ProviderDeepSeek, "", ProviderFamilyOpenAICompatible, false},
		{"groq", ProviderGroq, "", ProviderFamilyOpenAICompatible, false},
		{"moonshot", ProviderMoonshot, "", ProviderFamilyOpenAICompatible, false},
		{"nvidia", ProviderNVIDIA, "", ProviderFamilyOpenAICompatible, false},
		{"zhipu", ProviderZhipu, "", ProviderFamilyOpenAICompatible, false},

		// Supported: Anthropic
		{"anthropic", ProviderAnthropic, "", ProviderFamilyAnthropic, false},

		// Unsupported: explicit errors
		{"google unsupported", ProviderGoogle, "", ProviderFamilyUnsupported, true},
		{"cloudflare unsupported", ProviderCloudflare, "", ProviderFamilyUnsupported, true},

		// Ollama: requires baseURL
		{"ollama without base url", ProviderOllama, "", ProviderFamilyUnsupported, true},
		{"ollama with base url", ProviderOllama, "http://127.0.0.1:11434/v1", ProviderFamilyOpenAICompatible, false},
		{"ollama with whitespace only", ProviderOllama, "   ", ProviderFamilyUnsupported, true},

		// Unknown provider
		{"unknown provider", ProviderType("unknown"), "", ProviderFamilyUnsupported, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := providerFamily(tt.provider, tt.baseURL)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
