package voice

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestVADService_DetectSpeech_Success(t *testing.T) {
	mockClient := &MockVADClient{
		detectSpeechFunc: func(ctx context.Context, audioData []byte) (*VADResult, error) {
			return &VADResult{
				SpeechDetected: true,
				Confidence:     0.92,
				Timestamp:      time.Now(),
				Latency:        50 * time.Millisecond,
			}, nil
		},
	}

	service := NewVADService(SpeechDetector(mockClient))

	ctx := context.Background()
	audioData := []byte("fake audio data")

	result, err := service.DetectSpeech(ctx, audioData)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !result.SpeechDetected {
		t.Error("expected speech detected, got false")
	}

	if result.Confidence != 0.92 {
		t.Errorf("expected confidence 0.92, got %f", result.Confidence)
	}
}

func TestVADService_DetectSpeech_NoSpeech(t *testing.T) {
	mockClient := &MockVADClient{
		detectSpeechFunc: func(ctx context.Context, audioData []byte) (*VADResult, error) {
			return &VADResult{
				SpeechDetected: false,
				Confidence:     0.15,
				Timestamp:      time.Now(),
				Latency:        45 * time.Millisecond,
			}, nil
		},
	}

	service := NewVADService(SpeechDetector(mockClient))

	ctx := context.Background()
	audioData := []byte("silence")

	result, err := service.DetectSpeech(ctx, audioData)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if result.SpeechDetected {
		t.Error("expected no speech detected, got true")
	}

	if result.Confidence != 0.15 {
		t.Errorf("expected confidence 0.15, got %f", result.Confidence)
	}
}

func TestVADService_DetectSpeech_ContextCancellation(t *testing.T) {
	mockClient := &MockVADClient{
		detectSpeechFunc: func(ctx context.Context, audioData []byte) (*VADResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(10 * time.Second):
				return &VADResult{}, nil
			}
		},
	}

	service := NewVADService(SpeechDetector(mockClient))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	audioData := []byte("fake audio data")

	_, err := service.DetectSpeech(ctx, audioData)

	if err == nil {
		t.Fatal("expected context cancellation error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestVADService_DetectSpeech_EmptyAudio(t *testing.T) {
	mockClient := &MockVADClient{
		detectSpeechFunc: func(ctx context.Context, audioData []byte) (*VADResult, error) {
			if len(audioData) == 0 {
				return nil, ErrEmptyAudioData
			}
			return &VADResult{}, nil
		},
	}

	service := NewVADService(SpeechDetector(mockClient))

	ctx := context.Background()
	audioData := []byte{}

	_, err := service.DetectSpeech(ctx, audioData)

	if err == nil {
		t.Fatal("expected error for empty audio, got nil")
	}

	if err != ErrEmptyAudioData {
		t.Errorf("expected ErrEmptyAudioData, got %v", err)
	}
}

func TestVADService_DetectSpeech_ServiceUnavailable(t *testing.T) {
	mockClient := &MockVADClient{
		detectSpeechFunc: func(ctx context.Context, audioData []byte) (*VADResult, error) {
			return nil, errors.New("service unavailable")
		},
	}

	service := NewVADService(SpeechDetector(mockClient))

	ctx := context.Background()
	audioData := []byte("fake audio data")

	_, err := service.DetectSpeech(ctx, audioData)

	if err == nil {
		t.Fatal("expected error for unavailable service, got nil")
	}

	if err.Error() != "service unavailable" {
		t.Errorf("expected 'service unavailable', got %v", err)
	}
}

// MockVADClient implements SpeechDetector for testing
type MockVADClient struct {
	detectSpeechFunc func(ctx context.Context, audioData []byte) (*VADResult, error)
}

func (m *MockVADClient) DetectSpeech(ctx context.Context, audioData []byte) (*VADResult, error) {
	return m.detectSpeechFunc(ctx, audioData)
}
