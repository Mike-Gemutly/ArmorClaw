package interfaces

import "context"

// Message represents a single chat message with role and content.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the input for an AI chat completion request.
type ChatRequest struct {
	RequestID   string    `json:"request_id,omitempty"`
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float32   `json:"temperature,omitempty"`
	TopP        float32   `json:"top_p,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// Usage tracks token consumption for a chat completion.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatResponse is the result of a non-streaming chat completion.
type ChatResponse struct {
	RequestID    string `json:"request_id,omitempty"`
	Model        string `json:"model"`
	Content      string `json:"content"`
	Usage        Usage  `json:"usage"`
	FinishReason string `json:"finish_reason"`
}

// ChatChunk represents a single delta in a streamed chat completion.
type ChatChunk struct {
	RequestID string `json:"request_id,omitempty"`
	Delta     string `json:"delta"`
	Content   string `json:"content"`
}

// AIClient defines the interface for AI provider interactions.
// Extracted from internal/ai to allow pkg/ packages to reference
// AI capabilities without importing internal/.
type AIClient interface {
	// Chat sends a non-streaming chat completion request.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream sends a streaming chat completion request and returns
	// a channel of chunks. The channel is closed when the stream ends.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}
