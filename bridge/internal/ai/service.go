package ai

import (
	 "context"
    "fmt"
    "log/slog"
    "sync"

    "github.com/armorclaw/bridge/pkg/keystore"
)

type AIService struct {
    keystore  KeyRetriever
    clients   map[ProviderType]AIClient
    rateLimit *RateLimiter
    concLimit *ConcurrentLimiter
    logger    *slog.Logger
    mu        sync.RWMutex
}

type KeyRetriever interface {
    Retrieve(id string) (*keystore.Credential, error)
}

func NewAIService(ks KeyRetriever) *AIService {
    return &AIService{
        keystore:  ks,
        clients:   make(map[ProviderType]AIClient),
        rateLimit: DefaultRateLimiter(),
        concLimit: NewConcurrentLimiter(MaxConcurrent),
        logger:    slog.Default().With("component", "ai_service"),
    }
}

func (s *AIService) Chat(ctx context.Context, req ChatRequest, keyID string) (*ChatResponse, error) {
    release, ok := s.concLimit.TryAcquire()
    if !ok {
        return nil, &AIError{
            Code:      "too_many_requests",
            Message:   "Too many concurrent requests",
            Retryable: true,
        }
    }
    defer release()
    
    cred, err := s.keystore.Retrieve(keyID)
    if err != nil {
        return nil, fmt.Errorf("failed to retrieve API key: %w", err)
    }
    
    provider := ProviderType(cred.Provider)
    client, err := s.getClient(provider, cred.Token, cred.BaseURL)
    if err != nil {
        return nil, err
    }
    
    return client.Chat(ctx, req)
}

func (s *AIService) getClient(provider ProviderType, apiKey, baseURL string) (AIClient, error) {
    s.mu.RLock()
    client, ok := s.clients[provider]
    s.mu.RUnlock()
    
    if ok {
        return client, nil
    }
    
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if client, ok := s.clients[provider]; ok {
        return client, nil
    }
    
    switch provider {
    case ProviderOpenAI:
        client = NewOpenAIClient(apiKey, baseURL)
    case ProviderAnthropic:
        client = NewAnthropicClient(apiKey, baseURL)
    default:
        return nil, &AIError{
            Code:      "unsupported_provider",
            Message:   fmt.Sprintf("Unsupported provider: %s", provider),
            Retryable: false,
        }
    }
    
    s.clients[provider] = client
    return client, nil
}
func (s *AIService) CheckRateLimit(userID string) bool {
    return s.rateLimit.Allow(userID)
}
func (s *AIService) ValidateRequest(req ChatRequest) error {
    if len(req.Messages) == 0 {
        return &AIError{Code: "invalid_request", Message: "Messages cannot be empty"}
    }
    
    if len(req.Messages) > MaxMessages {
        return &AIError{Code: "invalid_request", Message: fmt.Sprintf("Too many messages (max %d)", MaxMessages)}
    }
    
    totalSize := 0
    for _, msg := range req.Messages {
        totalSize += len(msg.Content)
    }
    
    if totalSize > MaxPromptSize {
        return &AIError{Code: "invalid_request", Message: fmt.Sprintf("Prompt too large (max %d bytes)", MaxPromptSize)}
    }
    
    maxTokens := req.MaxTokens
    if maxTokens == 0 {
        maxTokens = 1024
    }
    if maxTokens < 16 {
        maxTokens = 16
    }
    
    estimatedPromptTokens := totalSize / CharsPerToken
    if estimatedPromptTokens > MaxPromptTokens {
        return &AIError{
            Code:      "prompt_too_large",
            Message:   fmt.Sprintf("Prompt too large (estimated %d tokens, max %d)", estimatedPromptTokens, MaxPromptTokens),
        }
    }
    
    if maxTokens > MaxCompletionTokens {
        return &AIError{
            Code:      "max_tokens_too_large",
            Message:   fmt.Sprintf("Max tokens too large (requested %d, max %d)", maxTokens, MaxCompletionTokens),
        }
    }
    
    totalTokens := estimatedPromptTokens + maxTokens
    if totalTokens > MaxTotalTokens {
        return &AIError{
            Code:      "total_tokens_exceeded",
            Message:   fmt.Sprintf("Total tokens would exceed limit (%d, max %d)", totalTokens, MaxTotalTokens),
        }
    }
    
    return nil
}

func (s *AIService) DefaultKeyID() string {
    return "openai-default"
}
func (s *AIService) DefaultModel() string {
    return "gpt-4o"
}