package voice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/interfaces"
)

func TestSTTService_Transcribe_Success(t *testing.T) {
	mockClient := &MockSTTClient{
		transcribeFunc: func(ctx context.Context, audioData []byte) (*interfaces.TranscriptionResult, error) {
			return &interfaces.TranscriptionResult{
				Text:       "hello world",
				Confidence: 0.95,
				Duration:   1 * time.Second,
				WordCount:  2,
				Timestamp:  time.Now(),
				Latency:    100 * time.Millisecond,
			}, nil
		},
	}

	service := NewSTTService(Transcriber(mockClient))

	ctx := context.Background()
	audioData := []byte("fake audio data")

	result, err := service.Transcribe(ctx, audioData)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.Text != "hello world" {
		t.Errorf("expected text 'hello world', got '%s'", result.Text)
	}

	if result.Confidence != 0.95 {
		t.Errorf("expected confidence 0.95, got %f", result.Confidence)
	}

	if result.WordCount != 2 {
		t.Errorf("expected word count 2, got %d", result.WordCount)
	}

	zeroed := true
	for _, b := range audioData {
		if b != 0 {
			zeroed = false
			break
		}
	}

	if !zeroed {
		t.Error("expected audio buffer to be zeroed after transcription")
	}
}

func TestSTTService_Transcribe_ContextCancellation(t *testing.T) {
	mockClient := &MockSTTClient{
		transcribeFunc: func(ctx context.Context, audioData []byte) (*interfaces.TranscriptionResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Second):
				return &interfaces.TranscriptionResult{}, nil
			}
		},
	}

	service := NewSTTService(Transcriber(mockClient))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	audioData := []byte("fake audio data")

	_, err := service.Transcribe(ctx, audioData)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestSTTService_Transcribe_EmptyAudio(t *testing.T) {
	mockClient := &MockSTTClient{
		transcribeFunc: func(ctx context.Context, audioData []byte) (*interfaces.TranscriptionResult, error) {
			if len(audioData) == 0 {
				return nil, interfaces.ErrEmptyAudioData
			}
			return &interfaces.TranscriptionResult{}, nil
		},
	}

	service := NewSTTService(Transcriber(mockClient))

	ctx := context.Background()
	audioData := []byte{}

	_, err := service.Transcribe(ctx, audioData)

	if err == nil {
		t.Fatal("expected error for empty audio, got nil")
	}

	if err != interfaces.ErrEmptyAudioData {
		t.Errorf("expected interfaces.ErrEmptyAudioData, got %v", err)
	}
}

func TestSTTService_Transcribe_ServiceUnavailable(t *testing.T) {
	mockClient := &MockSTTClient{
		transcribeFunc: func(ctx context.Context, audioData []byte) (*interfaces.TranscriptionResult, error) {
			return nil, errors.New("service unavailable")
		},
	}

	service := NewSTTService(Transcriber(mockClient))

	ctx := context.Background()
	audioData := []byte("fake audio data")

	_, err := service.Transcribe(ctx, audioData)

	if err == nil {
		t.Fatal("expected error for unavailable service, got nil")
	}

	if err.Error() != "service unavailable" {
		t.Errorf("expected 'service unavailable', got %v", err)
	}
}

func TestSTTService_Transcribe_RetryExhaustion(t *testing.T) {
	mockClient := &MockSTTClient{
		transcribeFunc: func(ctx context.Context, audioData []byte) (*interfaces.TranscriptionResult, error) {
			return nil, errors.New("temporary error")
		},
	}

	service := NewSTTService(Transcriber(mockClient))

	ctx := context.Background()
	audioData := []byte("fake audio data")

	_, err := service.Transcribe(ctx, audioData)

	if err == nil {
		t.Fatal("expected error after retry exhaustion, got nil")
	}

	if err.Error() != "temporary error" {
		t.Errorf("expected 'temporary error', got %v", err)
	}
}

type MockSTTClient struct {
	transcribeFunc func(ctx context.Context, audioData []byte) (*interfaces.TranscriptionResult, error)
}

func (m *MockSTTClient) Transcribe(ctx context.Context, audioData []byte) (*interfaces.TranscriptionResult, error) {
	result, err := m.transcribeFunc(ctx, audioData)
	if err == nil {
		for i := range audioData {
			audioData[i] = 0
		}
	}
	return result, err
}
