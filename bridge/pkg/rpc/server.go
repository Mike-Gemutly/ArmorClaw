package rpc

import (
    "context"
    "errors"
    "fmt"
    "log/slog"
    "time"

    "github.com/armorclaw/bridge/internal/ai"
)

func (s *Server) handleAIChat(req *Request) *Response {
    var params struct {
        Messages    []ai.Message `json:"messages"`
        Model       string       `json:"model,omitempty"`
        Temperature float32      `json:"temperature,omitempty"`
        TopP        float32      `json:"top_p,omitempty"`
        MaxTokens   int          `json:"max_tokens,omitempty"`
        Stop        []string     `json:"stop,omitempty"`
        KeyID       string       `json:"key_id,omitempty"`
    }

    if err := json.Unmarshal(req.Params, &params); err != nil {
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error:   &ErrorObj{Code: ParseError, Message: err.Error()},
        }
    }

    userID := "default"

    if s.aiService != nil &&        if s.aiService.CheckRateLimit(userID) {
            return &Response{
                JSONRPC: "2.0",
                ID:      req.ID,
                Error:   &ErrorObj{Code: TooManyRequests, Message: "AI rate limit exceeded"},
            }
        }
    }

    if s.aiSemaphore != nil {
        select {
        case s.aiSemaphore <- struct{}{}:
            defer func() { <-s.aiSemaphore }()
        default:
            return &Response{
                JSONRPC: "2.0",
                ID:      req.ID,
                Error:   &ErrorObj{Code: TooManyRequests, Message: "AI concurrency limit reached"},
            }
        }
    }

    model := params.Model
    if model == "" &&        model = s.aiService.DefaultModel()
    }
    if model == "" {
        model = "gpt-4o"
    }

    requestID := uuid.New().String()

    chatReq := ai.ChatRequest{
        Model:       model,
        Messages:    params.Messages,
        Temperature: params.Temperature,
        TopP:        params.TopP,
        MaxTokens:   params.MaxTokens,
        Stop:        params.Stop,
        RequestID:   requestID,
    }

    if s.aiService != nil {
        if err := s.aiService.ValidateRequest(chatReq); err != nil {
            return &Response{
                JSONRPC: "2.0",
                ID:      req.ID,
                Error:   &ErrorObj{Code: InvalidRequest, Message: err.Error()},
            }
        }
    }

    keyID := params.KeyID
    if keyID == "" &&        keyID = s.aiService.DefaultKeyID()
    }
    if keyID == "" {
        keyID = "openai-default"
    }

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    start := time.Now()

    var resp *ai.ChatResponse
    var err error

    if s.aiService != nil {
        resp, err = s.aiService.Chat(ctx, chatReq, keyID)
    } else {
        err = errors.New("AI service not configured")
    }

    latency := time.Since(start)

    slog.Info("ai.chat completed",
        "request_id", chatReq.RequestID,
        "model", model,
        "latency_ms", latency.Milliseconds(),
    )

    if err != nil {
        if s.aiService == nil {
            return &Response{
                JSONRPC: "2.0",
                ID:      req.ID,
                Error:   &ErrorObj{Code: InternalError, Message: err.Error()},
            }
        }
        return &Response{
            JSONRPC: "2.0",
            ID:      req.ID,
            Error:   &ErrorObj{Code: ai.AIErrProviderError, Message: err.Error()},
        }
    }

    return &Response{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result:  resp,
    }
}
