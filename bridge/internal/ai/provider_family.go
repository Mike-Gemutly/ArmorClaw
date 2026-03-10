package ai

import (
	"fmt"
	"strings"
)

type ProviderFamily string

const (
	ProviderFamilyOpenAICompatible ProviderFamily = "openai_compatible"
	ProviderFamilyAnthropic        ProviderFamily = "anthropic"
	ProviderFamilyUnsupported      ProviderFamily = "unsupported"
)

func providerFamily(provider ProviderType, baseURL string) (ProviderFamily, error) {
	switch provider {
	case ProviderOpenAI:
		return ProviderFamilyOpenAICompatible, nil

	case ProviderAnthropic:
		return ProviderFamilyAnthropic, nil

	case ProviderXAI,
		ProviderOpenRouter,
		ProviderDeepSeek,
		ProviderGroq,
		ProviderMoonshot,
		ProviderNVIDIA,
		ProviderZhipu:
		return ProviderFamilyOpenAICompatible, nil

	case ProviderOllama:
		if strings.TrimSpace(baseURL) == "" {
			return ProviderFamilyUnsupported, fmt.Errorf("ollama requires explicit base_url")
		}
		return ProviderFamilyOpenAICompatible, nil

	case ProviderGoogle:
		return ProviderFamilyUnsupported, fmt.Errorf("provider google requires a native adapter")

	case ProviderCloudflare:
		return ProviderFamilyUnsupported, fmt.Errorf("provider cloudflare requires a native adapter")

	default:
		return ProviderFamilyUnsupported, fmt.Errorf("provider %q is not supported", provider)
	}
}

// IsSupportedProvider checks if a provider is supported by the AI service.
func IsSupportedProvider(provider ProviderType) bool {
	// Ollama is special: it's supported if base URL is provided.
	// We assume it's supported here to allow it in selection,
	// but it will fail later if baseURL is missing.
	if provider == ProviderOllama {
		return true
	}

	family, err := providerFamily(provider, "dummy")
	if err != nil {
		return false
	}
	return family != ProviderFamilyUnsupported
}
