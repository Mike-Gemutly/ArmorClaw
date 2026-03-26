package voice

import (
	"context"
	"errors"
	"testing"
	"time"
)

 
func TestTTSService_Synthesize_Success(t *testing.T) {
	mockClient := &MockTTSClient{
		synthesizeFunc: func(ctx context.Context, text string) (*SynthesisResult, error) {
		return &SynthesisResult{
			AudioData:  []byte("fake audio data"),
		 TextLength: len(text),
            Duration:   2 * time.Second,
            Timestamp:  time.Now(),
            Latency:    150 * time.Millisecond,
        }, nil
    },
    }

 
    service := NewTTSService(Synthesizer(mockClient))
 
    ctx := context.Background()
    text := "hello world"
 
    result, err := service.Synthesize(ctx, text)
 
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
 
    if len(result.AudioData) == 0 {
        t.Error("expected audio data, got empty")
    }
 
    if result.TextLength != len(text) {
        t.Errorf("expected text length %d, got %d", len(text), result.TextLength)
    }
 
    if result.Duration != 2*time.Second {
        t.Errorf("expected duration 2s, got %v", result.Duration)
    }
}
 
func TestTTSService_Synthesize_ContextCancellation(t *testing.T)
 {
    mockClient := &MockTTSClient{
        synthesizeFunc: func(ctx context.Context, text string) (*SynthesisResult, error)
 {
            select {
            case <-ctx.Done():
                return nil, ctx.Err()
            case <-time.After(10 * time.Second):
                return &SynthesisResult{}, nil
            }
        },
    }
 
    service := NewTTSService(Synthesizer(mockClient))
 
    ctx, cancel := context.WithCancel(context.Background())
    cancel()
 
    _, err := service.Synthesize(ctx, "test")
 
    if err == nil {
        t.Fatal("expected context cancellation error, got nil")
    }
 
    if err != context.Canceled {
        t.Errorf("expected context.Canceled, got %v", err)
    }
}
 
func TestTTSService_Synthesize_EmptyText(t *testing.T)
 {
    mockClient := &MockTTSClient{
        synthesizeFunc: func(ctx context.Context, text string) (*SynthesisResult, error) {
        if text == "" {
            return nil, ErrEmptyText
        }
        return &SynthesisResult{}, nil
    },
    }
 
    service := NewTTSService(Synthesizer(mockClient))
 
    ctx := context.Background()
 
    _, err := service.Synthesize(ctx, "")
 
    if err == nil {
        t.Fatal("expected error for empty text, got nil")
    }
 
    if err != ErrEmptyText {
        t.Errorf("expected ErrEmptyText, got %v", err)
    }
}
 
func TestTTSService_Synthesize_ServiceUnavailable(t *testing.T)
 {
    mockClient := &MockTTSClient{
        synthesizeFunc: func(ctx context.Context, text string) (*SynthesisResult, error) {
            return nil, errors.New("service unavailable")
        },
    }
 
    service := NewTTSService(Synthesizer(mockClient))
 
    ctx := context.Background()
 
    _, err := service.Synthesize(ctx, "test")
 
    if err == nil {
        t.Fatal("expected error for unavailable service, got nil")
    }
 
    if err.Error() != "service unavailable" {
        t.Errorf("expected 'service unavailable', got %v", err)
    }
}
 
func TestTTSService_Synthesize_TextTooLong(t *testing.T)
 {
    mockClient := &MockTTSClient{
        synthesizeFunc: func(ctx context.Context, text string) (*SynthesisResult, error) {
            return nil, errors.New("text too long: 10000 characters (max: 5000)")
        },
    }
 
    service := NewTTSService(Synthesizer(mockClient))
 
    ctx := context.Background()
    longText := string(make([]byte, 10000))
 
    _, err := service.Synthesize(ctx, longText)
 
    if err == nil {
        t.Fatal("expected error for text too long, got nil")
    }
}
 
// MockTTSClient implements Synthesizer for testing
type MockTTSClient struct {
    synthesizeFunc func(ctx context.Context, text string) (*SynthesisResult, error)
}
 
func (m *MockTTSClient) Synthesize(ctx context.Context, text string) (*SynthesisResult, error) {
    return m.synthesizeFunc(ctx, text)
}
