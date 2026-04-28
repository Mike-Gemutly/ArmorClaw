package voice

import (
	"context"

	"github.com/armorclaw/bridge/pkg/interfaces"
	"log/slog"
)

// INTERFACE-ONLY: No concrete provider implementations exist. See doc/voice-stack.md.
// Synthesizer is the interface for text-to-speech synthesis
type Synthesizer interface {
	Synthesize(ctx context.Context, text string) (*interfaces.SynthesisResult, error)
}

// TTSService wraps a TTS client with logging
type TTSService struct {
	client Synthesizer
	logger *slog.Logger
}

// NewTTSService creates a new TTS service wrapper
func NewTTSService(client Synthesizer) *TTSService {
	return &TTSService{
		client: client,
		logger: slog.Default(),
	}
}

// Synthesize converts text to audio using the underlying TTS client
func (s *TTSService) Synthesize(ctx context.Context, text string) (*interfaces.SynthesisResult, error) {
	result, err := s.client.Synthesize(ctx, text)
	if err != nil {
		return nil, err
	}

	return result, nil
}
