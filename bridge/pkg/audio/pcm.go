// Package audio handles PCM audio streaming between WebRTC and agent containers
// All audio processing happens in the Bridge, not in containers
package audio

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
)

// AudioConfig holds audio configuration
type AudioConfig struct {
	SampleRate   int           // Audio sample rate in Hz (e.g., 48000)
	Channels     int           // Number of channels (1 = mono, 2 = stereo)
	BitDepth     int           // Bits per sample (16 = PCM16)
	FrameSize    int           // Number of samples per frame
	FrameDuration time.Duration // Duration of each frame
	BufferSize   int           // Size of audio buffer in frames
}

// DefaultAudioConfig returns default audio configuration
func DefaultAudioConfig() AudioConfig {
	return AudioConfig{
		SampleRate:     48000,
		Channels:       1,
		BitDepth:       16,
		FrameSize:      960, // 20ms at 48kHz
		FrameDuration:  20 * time.Millisecond,
		BufferSize:     10, // 10 frames = 200ms buffer
	}
}

// BytesPerFrame returns the number of bytes per audio frame
func (c AudioConfig) BytesPerFrame() int {
	return c.FrameSize * c.Channels * (c.BitDepth / 8)
}

// AudioFrame represents a single frame of PCM audio data
type AudioFrame struct {
	Data      []byte    // PCM audio data
	Timestamp time.Time // When the frame was captured/received
	Sequence  uint64    // Sequence number
}

// StreamDirection indicates the direction of audio flow
type StreamDirection int

const (
	// StreamIn is audio from client to agent (microphone)
	StreamIn StreamDirection = iota
	// StreamOut is audio from agent to client (speaker)
	StreamOut
)

// AudioStream manages a bidirectional audio stream
type AudioStream struct {
	config      AudioConfig
	sessionID   string
	direction   StreamDirection
	frames      chan AudioFrame
	closeOnce   sync.Once
	closeChan   chan struct{}
	mu          sync.RWMutex
	closed      bool
	onFrame     func(AudioFrame)
}

// NewAudioStream creates a new audio stream
func NewAudioStream(sessionID string, direction StreamDirection, config AudioConfig) *AudioStream {
	return &AudioStream{
		config:    config,
		sessionID: sessionID,
		direction: direction,
		frames:    make(chan AudioFrame, config.BufferSize),
		closeChan: make(chan struct{}),
	}
}

// SessionID returns the session ID for this stream
func (s *AudioStream) SessionID() string {
	return s.sessionID
}

// Direction returns the stream direction
func (s *AudioStream) Direction() StreamDirection {
	return s.direction
}

// WriteFrame writes an audio frame to the stream
func (s *AudioStream) WriteFrame(frame AudioFrame) error {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return fmt.Errorf("stream is closed")
	}
	s.mu.RUnlock()

	select {
	case s.frames <- frame:
		if s.onFrame != nil {
			s.onFrame(frame)
		}
		return nil
	case <-s.closeChan:
		return fmt.Errorf("stream is closed")
	case <-time.After(1 * time.Second):
		return fmt.Errorf("stream write timeout")
	}
}

// ReadFrame reads an audio frame from the stream
func (s *AudioStream) ReadFrame(ctx context.Context) (AudioFrame, error) {
	select {
	case frame := <-s.frames:
		return frame, nil
	case <-s.closeChan:
		return AudioFrame{}, fmt.Errorf("stream is closed")
	case <-ctx.Done():
		return AudioFrame{}, ctx.Err()
	}
}

// Close closes the audio stream
func (s *AudioStream) Close() {
	s.closeOnce.Do(func() {
		s.mu.Lock()
		s.closed = true
		s.mu.Unlock()
		close(s.closeChan)
		close(s.frames)
	})
}

// IsClosed returns true if the stream is closed
func (s *AudioStream) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// OnFrame sets a callback for incoming frames
func (s *AudioStream) OnFrame(handler func(AudioFrame)) {
	s.mu.Lock()
	s.onFrame = handler
	s.mu.Unlock()
}

// AudioPipeline manages audio streams between WebRTC and agent containers
type AudioPipeline struct {
	config    AudioConfig
	streams   sync.Map // map[sessionID]*AudioStreamPair
	closeOnce sync.Once
	closeChan chan struct{}
	mu        sync.RWMutex
}

// AudioStreamPair holds the bidirectional streams for a session
type AudioStreamPair struct {
	In  *AudioStream // Client → Agent (microphone)
	Out *AudioStream // Agent → Client (speaker)
}

// NewAudioPipeline creates a new audio pipeline
func NewAudioPipeline(config AudioConfig) *AudioPipeline {
	return &AudioPipeline{
		config:    config,
		streams:   sync.Map{},
		closeChan: make(chan struct{}),
	}
}

// CreateStreams creates bidirectional audio streams for a session
func (p *AudioPipeline) CreateStreams(sessionID string) (*AudioStreamPair, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check if streams already exist
	if _, exists := p.streams.Load(sessionID); exists {
		return nil, fmt.Errorf("streams already exist for session: %s", sessionID)
	}

	// Create stream pair
	pair := &AudioStreamPair{
		In:  NewAudioStream(sessionID, StreamIn, p.config),
		Out: NewAudioStream(sessionID, StreamOut, p.config),
	}

	p.streams.Store(sessionID, pair)

	return pair, nil
}

// GetStreams retrieves streams for a session
func (p *AudioPipeline) GetStreams(sessionID string) (*AudioStreamPair, bool) {
	if pair, ok := p.streams.Load(sessionID); ok {
		return pair.(*AudioStreamPair), true
	}
	return nil, false
}

// RemoveStreams removes streams for a session
func (p *AudioPipeline) RemoveStreams(sessionID string) {
	if pair, ok := p.streams.Load(sessionID); ok {
		streams := pair.(*AudioStreamPair)
		streams.In.Close()
		streams.Out.Close()
		p.streams.Delete(sessionID)
	}
}

// Close closes the audio pipeline and all streams
func (p *AudioPipeline) Close() {
	p.closeOnce.Do(func() {
		close(p.closeChan)

		p.streams.Range(func(key, value interface{}) bool {
			sessionID := key.(string)
			p.RemoveStreams(sessionID)
			return true
		})
	})
}

// PCMMixer mixes multiple PCM audio streams
type PCMMixer struct {
	config AudioConfig
}

// NewPCMMixer creates a new PCM mixer
func NewPCMMixer(config AudioConfig) *PCMMixer {
	return &PCMMixer{config: config}
}

// Mix mixes multiple audio frames into one
// Assumes all frames have the same length and configuration
func (m *PCMMixer) Mix(frames []AudioFrame) (AudioFrame, error) {
	if len(frames) == 0 {
		return AudioFrame{}, fmt.Errorf("no frames to mix")
	}

	if len(frames) == 1 {
		return frames[0], nil
	}

	// All frames should have the same length
	frameSize := len(frames[0].Data)
	output := make([]byte, frameSize)

	// Mix each sample
	for i := 0; i < frameSize; i += 2 {
		var sum int32
		for _, frame := range frames {
			if i+2 > len(frame.Data) {
				continue
			}
			// Read 16-bit sample
			sample := int16(binary.LittleEndian.Uint16(frame.Data[i : i+2]))
			sum += int32(sample)
		}

		// Clamp to 16-bit range
		if sum > 32767 {
			sum = 32767
		} else if sum < -32768 {
			sum = -32768
		}

		// Write mixed sample
		binary.LittleEndian.PutUint16(output[i:i+2], uint16(sum))
	}

	return AudioFrame{
		Data:      output,
		Timestamp: time.Now(),
	}, nil
}

// PCMEncoder handles PCM audio encoding/decoding
type PCMEncoder struct {
	config AudioConfig
}

// NewPCMEncoder creates a new PCM encoder
func NewPCMEncoder(config AudioConfig) *PCMEncoder {
	return &PCMEncoder{config: config}
}

// EncodeInt16ToBytes converts int16 samples to bytes
func (e *PCMEncoder) EncodeInt16ToBytes(samples []int16) []byte {
	bytes := make([]byte, len(samples)*2)
	for i, sample := range samples {
		binary.LittleEndian.PutUint16(bytes[i*2:i*2+2], uint16(sample))
	}
	return bytes
}

// DecodeBytesToInt16 converts bytes to int16 samples
func (e *PCMEncoder) DecodeBytesToInt16(data []byte) []int16 {
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	return samples
}

// Resample resamples audio from one sample rate to another
// This is a simple linear resampler - for production, use a proper resampling library
func (e *PCMEncoder) Resample(data []byte, fromRate, toRate int) ([]byte, error) {
	if fromRate == toRate {
		return data, nil
	}

	// Decode input samples
	samples := e.DecodeBytesToInt16(data)

	// Calculate output length
	ratio := float64(toRate) / float64(fromRate)
	outLength := int(float64(len(samples)) * ratio)
	output := make([]int16, outLength)

	// Linear interpolation
	for i := 0; i < outLength; i++ {
		pos := float64(i) / ratio
		index := int(pos)
		frac := pos - float64(index)

		if index >= len(samples)-1 {
			output[i] = samples[len(samples)-1]
		} else {
			sample1 := int(samples[index])
			sample2 := int(samples[index+1])
			output[i] = int16(float64(sample1) + frac*float64(sample2-sample1))
		}
	}

	return e.EncodeInt16ToBytes(output), nil
}

// AudioBuffer provides a circular buffer for audio data
type AudioBuffer struct {
	data     []byte
	readPos  int
	writePos int
	size     int
	capacity int
	mu       sync.Mutex
}

// NewAudioBuffer creates a new circular audio buffer
func NewAudioBuffer(capacity int) *AudioBuffer {
	return &AudioBuffer{
		data:     make([]byte, capacity),
		capacity: capacity,
	}
}

// Write writes data to the buffer
func (b *AudioBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, byte := range data {
		b.data[b.writePos] = byte
		b.writePos = (b.writePos + 1) % b.capacity
		if b.size < b.capacity {
			b.size++
		}
	}

	return len(data), nil
}

// Read reads data from the buffer
func (b *AudioBuffer) Read(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.size == 0 {
		return 0, io.EOF
	}

	n := 0
	for n < len(data) && b.size > 0 {
		data[n] = b.data[b.readPos]
		b.readPos = (b.readPos + 1) % b.capacity
		b.size--
		n++
	}

	return n, nil
}

// Available returns the number of bytes available to read
func (b *AudioBuffer) Available() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.size
}

// Capacity returns the buffer capacity
func (b *AudioBuffer) Capacity() int {
	return b.capacity
}

// Clear clears the buffer
func (b *AudioBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.readPos = 0
	b.writePos = 0
	b.size = 0
}

// WebRTCTrackReader reads audio from a WebRTC track
type WebRTCTrackReader struct {
	track    *webrtc.TrackRemote
	buffer   *AudioBuffer
	config   AudioConfig
	stopChan chan struct{}
	running  bool
	mu       sync.RWMutex
}

// NewWebRTCTrackReader creates a new track reader
func NewWebRTCTrackReader(track *webrtc.TrackRemote, config AudioConfig) *WebRTCTrackReader {
	return &WebRTCTrackReader{
		track:    track,
		buffer:   NewAudioBuffer(config.BytesPerFrame() * config.BufferSize),
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start starts reading from the track
func (r *WebRTCTrackReader) Start() error {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return fmt.Errorf("already running")
	}
	r.running = true
	r.mu.Unlock()

	go r.readLoop()
	return nil
}

// Stop stops reading from the track
func (r *WebRTCTrackReader) Stop() {
	r.mu.Lock()
	if !r.running {
		r.mu.Unlock()
		return
	}
	r.running = false
	r.mu.Unlock()

	close(r.stopChan)
}

// readLoop continuously reads from the track
func (r *WebRTCTrackReader) readLoop() {
	buf := make([]byte, 1500) // MTU size

	for {
		select {
		case <-r.stopChan:
			return
		default:
			n, _, err := r.track.Read(buf)
			if err != nil {
				if err == io.EOF {
					return
				}
				continue
			}

			// Write to buffer
			r.buffer.Write(buf[:n])
		}
	}
}

// Read reads audio data from the track
func (r *WebRTCTrackReader) Read(data []byte) (int, error) {
	return r.buffer.Read(data)
}

// WebRTCTrackWriter writes audio to a WebRTC track
type WebRTCTrackWriter struct {
	track    *webrtc.TrackLocalStaticSample
	config   AudioConfig
	stopChan chan struct{}
	running  bool
	mu       sync.RWMutex
}

// NewWebRTCTrackWriter creates a new track writer
func NewWebRTCTrackWriter(track *webrtc.TrackLocalStaticSample, config AudioConfig) *WebRTCTrackWriter {
	return &WebRTCTrackWriter{
		track:    track,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Write writes audio data to the track
func (w *WebRTCTrackWriter) Write(data []byte) (int, error) {
	sample := media.Sample{
		Data:     data,
		Duration: w.config.FrameDuration,
	}

	if err := w.track.WriteSample(sample); err != nil {
		return 0, err
	}

	return len(data), nil
}

// Close closes the track writer
func (w *WebRTCTrackWriter) Close() {
	w.mu.Lock()
	w.running = false
	w.mu.Unlock()
	close(w.stopChan)
}
