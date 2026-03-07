package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
)

type AnthropicClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewAnthropicClient(apiKey, baseURL string) *AnthropicClient {
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	
	return &AnthropicClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: newHTTPClient(ProviderTimeouts[ProviderAnthropic]),
		logger:     slog.Default().With("provider", "anthropic"),
	}
}

type anthropicRequest struct {
	Model       string            `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string            `json:"system,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float32           `json:"temperature,omitempty"`
	TopP        float32           `json:"top_p,omitempty"`
	Stop        []string          `json:"stop_sequences,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	ID      string           `json:"id"`
	Type    string           `json:"type"`
	Model   string           `json:"model"`
	Content []contentBlock   `json:"content"`
	Usage   anthropicUsage   `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (c *AnthropicClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
  if req.RequestID == "" {
    req.RequestID = uuid.New().String()[:8]
  }
  
  anthropicReq, err := c.convertRequest(req)
  if err != nil {
    return nil, err
  }
  
  body, err := json.Marshal(anthropicReq)
  if err != nil {
    return nil, fmt.Errorf("failed to marshal request: %w", err)
  }
  
  httpResp, err := executeWithRetry(ctx, DefaultRetryConfig, func() (*http.Response, []byte, error) {
    httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/messages", bytes.NewReader(body))
    if err != nil {
      return nil, nil, err
    }
    
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("x-api-key", c.apiKey)
    httpReq.Header.Set("anthropic-version", "2023-06-01")
    
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
      return nil, nil, err
    }
    defer resp.Body.Close()
    
    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
      return nil, nil, err
    }
    
    return resp, bodyBytes, nil
  })
  
  if err != nil {
    c.logger.Error("AI request failed", "request_id", req.RequestID, "model", req.Model, "error", err)
    return nil, err
  }
  
  // Parse HTTP response into ChatResponse
  var anthropicResp anthropicResponse
  if err := json.Unmarshal(httpResp.Body, &anthropicResp); err != nil {
    c.logger.Error("Failed to parse Anthropic response", "error", err)
    return nil, fmt.Errorf("failed to parse response: %w", err)
  }
  
  response, err := c.parseResponse(anthropicResp)
  if err != nil {
    c.logger.Error("Failed to convert Anthropic response", "error", err)
    return nil, fmt.Errorf("failed to convert response: %w", err)
  }
  
  return response, nil
}

func (c *AnthropicClient) convertRequest(req ChatRequest) (*anthropicRequest, error) {
	anthropicReq := &anthropicRequest{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
	}
	
	var messages []anthropicMessage
	var systemPrompt string
	
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt = msg.Content
		} else {
			role := msg.Role
			if role == "assistant" {
				role = "assistant"
			}
			messages = append(messages, anthropicMessage{
				Role:    role,
				Content: msg.Content,
			})
		}
	}
	
	anthropicReq.Messages = messages
	anthropicReq.System = systemPrompt
	
	return anthropicReq, nil
}

func (c *AnthropicClient) parseResponse(resp anthropicResponse) (*ChatResponse, error) {
	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}
	
	return &ChatResponse{
		Model:        resp.Model,
		Content:      content,
		FinishReason: "end_turn",
		Usage: Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}, nil
}

func (c *AnthropicClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	return nil, &AIError{
		Code:    "not_implemented",
		Message: "Streaming not yet implemented for Anthropic",
	}
}
