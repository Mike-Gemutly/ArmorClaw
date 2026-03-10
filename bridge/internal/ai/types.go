package ai

import "time"

type ProviderType string

const (
	ProviderOpenAI     ProviderType = "openai"
	ProviderAnthropic  ProviderType = "anthropic"
	ProviderXAI        ProviderType = "xai"
	ProviderOpenRouter ProviderType = "openrouter"
	ProviderDeepSeek   ProviderType = "deepseek"
	ProviderGroq       ProviderType = "groq"
	ProviderMoonshot   ProviderType = "moonshot"
	ProviderNVIDIA     ProviderType = "nvidia"
	ProviderZhipu      ProviderType = "zhipu"
	ProviderOllama     ProviderType = "ollama"
	ProviderGoogle     ProviderType = "google"
	ProviderCloudflare ProviderType = "cloudflare"
)

var ProviderTimeouts = map[ProviderType]time.Duration{
	ProviderOpenAI:    60 * time.Second,
	ProviderAnthropic: 90 * time.Second,
}

var DefaultModels = map[ProviderType]string{
	ProviderOpenAI:    "gpt-4o",
	ProviderAnthropic: "claude-3-5-sonnet-20241022",
}

var FallbackModels = map[ProviderType]string{
	ProviderOpenAI:    "gpt-4o-mini",
	ProviderAnthropic: "claude-3-5-haiku-20241022",
}

type ChatRequest struct {
	RequestID   string    `json:"request_id,omitempty"`
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	TopP        float32   `json:"top_p,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	RequestID    string `json:"request_id,omitempty"`
	Model        string `json:"model"`
	Content      string `json:"content"`
	Usage        Usage  `json:"usage"`
	FinishReason string `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type AIError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func (e *AIError) Error() string {
	return e.Message
}

type ChatChunk struct {
	RequestID string `json:"request_id,omitempty"`
	Delta     string `json:"delta"`
	Content   string `json:"content"`
}

const (
	MaxMessages         = 100
	MaxPromptSize       = 32 * 1024 // ~8k tokens
	MaxTokens           = 4096
	MaxConcurrent       = 10
	RatePerMinute       = 30
	GlobalRateLimit     = 120
	MaxPromptTokens     = 8000
	MaxCompletionTokens = 4096
	MaxTotalTokens      = 12000
	UserBurstCapacity   = 60
	GlobalBurstCapacity = 200
	CharsPerToken       = 4 // Approximate token estimation (~4 characters per token)
)

const (
	AIErrPromptTooLarge    = -10001
	AIErrMaxTokensExceeded = -10002
	AIErrRateLimitExceeded = -10003
	AIErrProviderError     = -10004
)
