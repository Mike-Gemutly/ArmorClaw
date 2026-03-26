package voice

import (
	"context"

	"log/slog"
)

// SpeechDetector is the interface for voice activity detection
type SpeechDetector interface {
	DetectSpeech(ctx context.Context, audioData []byte) (*VADResult, error)
}

// VADService wraps a VAD client with logging and provides the SpeechDetector interface
type VADService struct {
	client SpeechDetector
	logger *slog.Logger
}

// NewVADService creates a new VAD service wrapper
func NewVADService(client SpeechDetector) *VADService {
	return &VADService{
		client: client,
		logger: slog.Default(),
	}
}

// DetectSpeech detects speech activity using the underlying VAD client
func (s *VADService) DetectSpeech(ctx context.Context, audioData []byte) (*VADResult, error) {
	result, err := s.client.DetectSpeech(ctx, audioData)
	if err != nil {
		return nil, err
	}

	return result, nil
}
