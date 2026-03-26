// Package interfaces defines shared interfaces for voice processing
// This package breaks the import cycle between pkg/voice and internal/adapter
package interfaces

import (
	"context"
	"errors"
	"time"
)

// VoiceManager handles Matrix call events for voice processing
type VoiceManager interface {
	HandleMatrixCallEvent(roomID, eventID, senderID string, event interface{}) error
}

// Transcriber defines speech-to-text transcription
type Transcriber interface {
	Transcribe(ctx context.Context, audioData []byte) (*TranscriptionResult, error)
}

// Synthesizer defines text-to-speech synthesis
type Synthesizer interface {
	Synthesize(ctx context.Context, text string) (*SynthesisResult, error)
}

// SpeechDetector defines voice activity detection
type SpeechDetector interface {
	DetectSpeech(ctx context.Context, audioData []byte) (*VADResult, error)
}

// TranscriptionResult represents the result of speech-to-text transcription
type TranscriptionResult struct {
	Text       string
	Confidence float64
	Duration   time.Duration
	WordCount  int
	Timestamp  time.Time
	Latency    time.Duration
}

// SynthesisResult represents the result of text-to-speech synthesis
type SynthesisResult struct {
	AudioData  []byte
	TextLength int
	Duration   time.Duration
	Timestamp  time.Time
	Latency    time.Duration
}

// VADResult represents the result of voice activity detection
type VADResult struct {
	SpeechDetected bool
	Confidence     float64
	Timestamp      time.Time
	Latency        time.Duration
}

// Voice processing errors
var (
	// ErrEmptyAudioData is returned when audio data is empty
	ErrEmptyAudioData = errors.New("empty audio data")
)
