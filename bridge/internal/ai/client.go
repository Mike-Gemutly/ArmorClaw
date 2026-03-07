package ai

import (
	"context"
	"net/http"
	"time"
)

type AIClient interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error)
}

type ClientOptions struct {
	Provider   ProviderType
	APIKey     string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

func newHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:          10,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
}

func DefaultOptions(provider ProviderType, apiKey string) ClientOptions {
	opts := ClientOptions{
		Provider:   provider,
		APIKey:     apiKey,
		MaxRetries: 3,
	}
	
	if timeout, ok := ProviderTimeouts[provider]; ok {
		opts.Timeout = timeout
	} else {
		opts.Timeout = 60 * time.Second
	}
	
	return opts
}
