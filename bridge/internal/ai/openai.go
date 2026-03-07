package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type OpenAIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewOpenAIClient(apiKey, baseURL string) *OpenAIClient {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	
	return &OpenAIClient{
		apiKey:     apiKey,
		baseURL:    baseURL,
		httpClient: newHTTPClient(ProviderTimeouts[ProviderOpenAI]),
		logger:     slog.Default().With("provider", "openai"),
	}
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float32         `json:"temperature,omitempty"`
	TopP        float32         `json:"top_p,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string           `json:"id"`
	Object  string           `json:"object"`
	Created int64            `json:"created"`
	Model   string           `json:"model"`
	Choices []openAIChoice   `json:"choices"`
	Usage   openAIUsage      `json:"usage"`
	Error   *openAIErrorBody `json:"error,omitempty"`
}

type openAIChoice struct {
	Index        int           `json:"index"`
	Message      openAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type openAIErrorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	start := time.Now()
	
	if req.RequestID == "" {
		req.RequestID = uuid.New().String()[:8]
	}
	
	openAIReq := c.convertRequest(req)
	
	body, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	result, err := executeWithRetry(ctx, DefaultRetryConfig, func() (*http.Response, []byte, error) {
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			return nil, nil, err
		}
		
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
		
		resp, err := c.httpClient.Do(httpReq)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, nil, err
		}
		
		return resp, respBody, nil
	})
	
	if err != nil {
		c.logger.Error("AI request failed", "request_id", req.RequestID, "model", req.Model, "error", err)
		return nil, err
	}
	
	response, err := c.parseResponse(result.Body)
	if err != nil {
		return nil, err
	}
	
	response.RequestID = req.RequestID
	
	latency := time.Since(start)
	c.logger.Info("AI request completed",
		"provider", "openai",
		"model", req.Model,
		"request_id", req.RequestID,
		"latency_ms", latency.Milliseconds(),
		"tokens_used", response.Usage.TotalTokens,
	)
	
	return response, nil
}

func (c *OpenAIClient) convertRequest(req ChatRequest) *openAIRequest {
	oaiReq := &openAIRequest{
		Model:       req.Model,
		Messages:    make([]openAIMessage, len(req.Messages)),
		Temperature: req.Temperature,
		TopP:        req.TopP,
		MaxTokens:   req.MaxTokens,
		Stop:        req.Stop,
	}
	
	for i, msg := range req.Messages {
		oaiReq.Messages[i] = openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	
	return oaiReq
}

func (c *OpenAIClient) parseResponse(body []byte) (*ChatResponse, error) {
	var resp openAIResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	if resp.Error != nil {
		return nil, &AIError{
			Code:      resp.Error.Code,
			Message:   resp.Error.Message,
			Retryable: false,
		}
	}
	
	if len(resp.Choices) == 0 {
		return nil, &AIError{
			Code:    "no_response",
			Message: "No response choices returned",
		}
	}
	
	choice := resp.Choices[0]
	
	return &ChatResponse{
		Model:        resp.Model,
		Content:      choice.Message.Content,
		FinishReason: choice.FinishReason,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

func (c *OpenAIClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
	return nil, &AIError{
		Code:    "not_implemented",
		Message: "Streaming not yet implemented for OpenAI",
	}
}
