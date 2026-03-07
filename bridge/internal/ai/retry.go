package ai

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

var DefaultRetryConfig = RetryConfig{
	MaxRetries:   3,
	InitialDelay: 1 * time.Second,
	MaxDelay:     4 * time.Second,
	Multiplier:   2.0,
}

func isRetryable(statusCode int) bool {
	switch statusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable:
		return true
	default:
		return false
	}
}

func normalizeError(statusCode int, body []byte) *AIError {
	err := &AIError{
		Retryable: isRetryable(statusCode),
	}
	
	switch statusCode {
	case http.StatusUnauthorized:
		err.Code = "invalid_api_key"
		err.Message = "Authentication failed"
		err.Retryable = false
	case http.StatusTooManyRequests:
		err.Code = "rate_limited"
		err.Message = "Rate limit exceeded"
		err.Retryable = true
	case http.StatusBadRequest:
		err.Code = "bad_request"
		err.Message = extractErrorMessage(body)
		err.Retryable = false
	case http.StatusNotFound:
		err.Code = "not_found"
		err.Message = "Model or resource not found"
		err.Retryable = false
	case http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable:
		err.Code = "server_error"
		err.Message = "Provider server error"
		err.Retryable = true
	default:
		err.Code = "unknown"
		err.Message = extractErrorMessage(body)
	}
	
	return err
}

func extractErrorMessage(body []byte) string {
	bodyStr := string(body)
	if strings.Contains(bodyStr, `"message"`) {
		start := strings.Index(bodyStr, `"message"`)
		if start != -1 {
			rest := bodyStr[start+11:]
			quoteStart := strings.Index(rest, `"`)
			if quoteStart != -1 {
				rest = rest[quoteStart+1:]
				quoteEnd := strings.Index(rest, `"`)
				if quoteEnd != -1 {
					return rest[:quoteEnd]
				}
			}
		}
	}
	if len(bodyStr) > 200 {
		return bodyStr[:200]
	}
	return bodyStr
}

func getRetryAfter(resp *http.Response) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}
	
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}
	
	return 0
}

func calculateBackoff(attempt int, cfg RetryConfig) time.Duration {
	delay := float64(cfg.InitialDelay)
	for i := 0; i < attempt; i++ {
		delay *= cfg.Multiplier
	}
	
	if delay > float64(cfg.MaxDelay) {
		delay = float64(cfg.MaxDelay)
	}
	
	jitter := rand.Float64() * 0.1 * delay
	return time.Duration(delay + jitter)
}

type httpResponse struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func executeWithRetry(ctx context.Context, cfg RetryConfig, fn func() (*http.Response, []byte, error)) (*httpResponse, error) {
	var lastErr error
	
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		resp, body, err := fn()
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil, err
			}
			lastErr = err
			if attempt < cfg.MaxRetries {
				delay := calculateBackoff(attempt, cfg)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}
			}
			continue
		}
		
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return &httpResponse{
				StatusCode: resp.StatusCode,
				Header:     resp.Header,
				Body:       body,
			}, nil
		}
		
		aiErr := normalizeError(resp.StatusCode, body)
		if !aiErr.Retryable {
			return nil, aiErr
		}
		
		lastErr = aiErr
		
		if attempt < cfg.MaxRetries {
			var delay time.Duration
			if resp.StatusCode == http.StatusTooManyRequests {
				delay = getRetryAfter(resp)
			}
			if delay == 0 {
				delay = calculateBackoff(attempt, cfg)
			}
			
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
	}
	
	if lastErr == nil {
		lastErr = errors.New("max retries exceeded")
	}
	return nil, lastErr
}
