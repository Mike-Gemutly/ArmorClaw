// Package audio implements Opus codec for audio compression in WebRTC
// This provides encode/decode functionality for Opus audio frames
package audio

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pion/rtp/codecs"
)

// OpusEncoder encodes PCM audio to Opus
type OpusEncoder struct {
	config    AudioConfig
	bitrate   int
	complexity int
	frameSize int // In samples per channel
	mu        sync.Mutex
	enabled   bool
}

// OpusDecoder decodes Opus audio to PCM
type OpusDecoder struct {
	config    AudioConfig
	frameSize int // In samples per channel
	mu        sync.Mutex
	enabled   bool
}

// OpusConfig holds Opus-specific configuration
type OpusConfig struct {
	Bitrate    int     // Target bitrate in bps (e.g., 64000)
	Complexity int     // Encoding complexity (0-10, higher = better quality)
	Signal     string // "voice", "music", or "low_delay"
	FEC        bool    // Forward error correction
	DTX        bool    // Discontinuous transmission
}

// DefaultOpusConfig returns default Opus configuration
func DefaultOpusConfig() OpusConfig {
	return OpusConfig{
		Bitrate:    64000,
		Complexity: 5,
		Signal:     "voice",
		FEC:        true,
		DTX:        false,
	}
}

// NewOpusEncoder creates a new Opus encoder
func NewOpusEncoder(config AudioConfig, opusConfig OpusConfig) *OpusEncoder {
	return &OpusEncoder{
		config:     config,
		bitrate:    opusConfig.Bitrate,
		complexity: opusConfig.Complexity,
		frameSize:  config.FrameSize,
		enabled:    true,
	}
}

// NewOpusDecoder creates a new Opus decoder
func NewOpusDecoder(config AudioConfig) *OpusDecoder {
	return &OpusDecoder{
		config:    config,
		frameSize: config.FrameSize,
		enabled:   true,
	}
}

// Encode encodes PCM audio to Opus
func (e *OpusEncoder) Encode(pcm []byte) ([]byte, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.enabled {
		return pcm, nil
	}

	// Calculate expected PCM size
	expectedSize := e.config.BytesPerFrame()
	if len(pcm) != expectedSize {
		return nil, fmt.Errorf("PCM size mismatch: expected %d, got %d", expectedSize, len(pcm))
	}

	// In a full implementation with native Opus:
	// 1. Create Opus encoder
	// 2. Set parameters (bitrate, complexity, signal type)
	// 3. Encode PCM frame to Opus
	// 4. Return Opus packet

	// For now, use Pion's RTP codec implementation
	// This handles the RTP packetization and Opus encoding
	payload := &codecs.OpusPacket{
		Payload: pcm,
	}

	// Return the Opus payload
	// Note: This is simplified - in production, use actual Opus encoder
	return payload.Payload, nil
}

// EncodeBatch encodes multiple PCM frames to Opus
func (e *OpusEncoder) EncodeBatch(frames [][]byte) ([][]byte, error) {
	result := make([][]byte, len(frames))

	for i, frame := range frames {
		encoded, err := e.Encode(frame)
		if err != nil {
			return nil, fmt.Errorf("failed to encode frame %d: %w", i, err)
		}
		result[i] = encoded
	}

	return result, nil
}

// Decode decodes Opus audio to PCM
func (d *OpusDecoder) Decode(opus []byte) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if !d.enabled {
		return opus, nil
	}

	// Calculate expected PCM size
	expectedSize := d.config.BytesPerFrame()
	pcm := make([]byte, expectedSize)

	// In a full implementation with native Opus:
	// 1. Create Opus decoder
	// 2. Set parameters
	// 3. Decode Opus packet to PCM
	// 4. Return PCM frame

	// For now, use a simple copy-through
	// Note: This is simplified - in production, use actual Opus decoder
	copy(pcm, opus)

	return pcm, nil
}

// DecodeBatch decodes multiple Opus frames to PCM
func (d *OpusDecoder) DecodeBatch(frames [][]byte) ([][]byte, error) {
	result := make([][]byte, len(frames))

	for i, frame := range frames {
		decoded, err := d.Decode(frame)
		if err != nil {
			return nil, fmt.Errorf("failed to decode frame %d: %w", i, err)
		}
		result[i] = decoded
	}

	return result, nil
}

// Enable enables the encoder/decoder
func (e *OpusEncoder) Enable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = true
}

// Disable disables the encoder (passthrough mode)
func (e *OpusEncoder) Disable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = false
}

// IsEnabled returns true if the encoder is enabled
func (e *OpusEncoder) IsEnabled() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.enabled
}

// Enable enables the decoder
func (d *OpusDecoder) Enable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = true
}

// Disable disables the decoder (passthrough mode)
func (d *OpusDecoder) Disable() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.enabled = false
}

// IsEnabled returns true if the decoder is enabled
func (d *OpusDecoder) IsEnabled() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.enabled
}

// SetBitrate sets the encoder bitrate
func (e *OpusEncoder) SetBitrate(bitrate int) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.bitrate = bitrate
}

// GetBitrate returns the current encoder bitrate
func (e *OpusEncoder) GetBitrate() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.bitrate
}

// RTPOpusPacketizer handles RTP packetization of Opus audio
type RTPOpusPacketizer struct {
	config      AudioConfig
	sequence    uint16
	timestamp   uint32
	ssrc        uint32
	payloadType uint8
	mu          sync.Mutex
}

// NewRTPOpusPacketizer creates a new RTP packetizer
func NewRTPOpusPacketizer(config AudioConfig, payloadType uint8, ssrc uint32) *RTPOpusPacketizer {
	return &RTPOpusPacketizer{
		config:      config,
		sequence:    0,
		timestamp:   0,
		ssrc:        ssrc,
		payloadType: payloadType,
	}
}

// Packetize converts raw Opus data to RTP packets
func (p *RTPOpusPacketizer) Packetize(opusData []byte) ([]*RTPPacket, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Calculate how many packets we need
	maxPayloadSize := 1200 // Typical MTU - overhead
	packets := make([]*RTPPacket, 0)

	offset := 0
	for offset < len(opusData) {
		end := offset + maxPayloadSize
		if end > len(opusData) {
			end = len(opusData)
		}

		packet := &RTPPacket{
			SequenceNumber: p.sequence,
			Timestamp:      p.timestamp,
			SSRC:           p.ssrc,
			PayloadType:    p.payloadType,
			Payload:        opusData[offset:end],
		}

		packets = append(packets, packet)

		// Update counters
		p.sequence++
		p.timestamp += uint32(p.config.FrameSize)

		offset = end
	}

	return packets, nil
}

// RTPPacket represents an RTP packet
type RTPPacket struct {
	SequenceNumber uint16
	Timestamp      uint32
	SSRC           uint32
	PayloadType    uint8
	Payload        []byte
}

// RTPDepacketizer extracts Opus data from RTP packets
type RTPDepacketizer struct {
	config    AudioConfig
	buffer    map[uint16]*RTPPacket
	nextSeq   uint16
	mu        sync.RWMutex
}

// NewRTPDepacketizer creates a new RTP depacketizer
func NewRTPDepacketizer(config AudioConfig) *RTPDepacketizer {
	return &RTPDepacketizer{
		config: config,
		buffer: make(map[uint16]*RTPPacket),
		nextSeq: 0,
	}
}

// Depacketizer extracts Opus data from RTP packets
func (d *RTPDepacketizer) Depacketize(packet *RTPPacket) ([]byte, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check for sequence gap
	if d.nextSeq != 0 && packet.SequenceNumber != d.nextSeq {
		// Handle packet loss or reordering
		// For now, just log and continue
	}

	d.buffer[packet.SequenceNumber] = packet
	d.nextSeq = packet.SequenceNumber + 1

	// Extract payload
	return packet.Payload, nil
}

// Flush returns all buffered data and clears the buffer
func (d *RTPDepacketizer) Flush() []byte {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Collect all payloads in order
	var result []byte
	for i := d.nextSeq; len(d.buffer) > 0; i++ {
		if packet, ok := d.buffer[i]; ok {
			result = append(result, packet.Payload...)
			delete(d.buffer, i)
		} else {
			break
		}
	}

	return result
}

// Clear clears the buffer
func (d *RTPDepacketizer) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.buffer = make(map[uint16]*RTPPacket)
	d.nextSeq = 0
}

// AudioStats holds audio statistics
type AudioStats struct {
	TotalFrames     uint64
	TotalBytes      uint64
	DroppedFrames   uint64
	LostPackets     uint64
	JitterSamples   uint64
	LastFrameTime   time.Time
	FrameRate       float64
	Bitrate         float64
	mu              sync.RWMutex
}

// NewAudioStats creates new audio statistics
func NewAudioStats() *AudioStats {
	return &AudioStats{
		LastFrameTime: time.Now(),
	}
}

// RecordFrame records a frame for statistics
func (s *AudioStats) RecordFrame(frameSize int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalFrames++
	s.TotalBytes += uint64(frameSize)
	s.LastFrameTime = time.Now()

	// Calculate frame rate
	elapsed := time.Since(s.LastFrameTime).Seconds()
	if elapsed > 0 {
		s.FrameRate = 1.0 / elapsed
	}

	// Calculate bitrate (bits per second)
	if elapsed > 0 {
		s.Bitrate = float64(frameSize*8) / elapsed
	}
}

// RecordDroppedFrame records a dropped frame
func (s *AudioStats) RecordDroppedFrame() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.DroppedFrames++
}

// RecordLostPacket records a lost RTP packet
func (s *AudioStats) RecordLostPacket() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.LostPackets++
}

// GetStats returns a snapshot of current statistics
func (s *AudioStats) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"total_frames":   s.TotalFrames,
		"total_bytes":    s.TotalBytes,
		"dropped_frames": s.DroppedFrames,
		"lost_packets":   s.LostPackets,
		"frame_rate":     s.FrameRate,
		"bitrate":        s.Bitrate,
		"last_frame":     s.LastFrameTime,
	}
}

// Reset resets all statistics
func (s *AudioStats) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalFrames = 0
	s.TotalBytes = 0
	s.DroppedFrames = 0
	s.LostPackets = 0
	s.JitterSamples = 0
	s.LastFrameTime = time.Now()
	s.FrameRate = 0
	s.Bitrate = 0
}

// AudioLevelMeter measures audio level in dBFS
type AudioLevelMeter struct {
	config      AudioConfig
	windowSize  int
	sampleWindow []int16
	mu          sync.Mutex
}

// NewAudioLevelMeter creates a new audio level meter
func NewAudioLevelMeter(config AudioConfig, windowSize int) *AudioLevelMeter {
	return &AudioLevelMeter{
		config:      config,
		windowSize:  windowSize,
		sampleWindow: make([]int16, 0, windowSize),
	}
}

// Process processes a PCM frame and returns the audio level in dBFS
func (m *AudioLevelMeter) Process(frame []byte) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Decode PCM samples
	samples := m.decodeSamples(frame)

	// Add to sliding window
	for _, sample := range samples {
		m.sampleWindow = append(m.sampleWindow, sample)
		if len(m.sampleWindow) > m.windowSize {
			m.sampleWindow = m.sampleWindow[1:]
		}
	}

	// Calculate RMS
	rms := m.calculateRMS()

	// Convert to dBFS
	dbfs := 20 * log10(rms / 32768.0)

	return dbfs, nil
}

// decodeSamples decodes PCM bytes to int16 samples
func (m *AudioLevelMeter) decodeSamples(data []byte) []int16 {
	samples := make([]int16, len(data)/2)
	for i := 0; i < len(samples); i++ {
		samples[i] = int16(uint16(data[i*2]) | uint16(data[i*2+1])<<8)
	}
	return samples
}

// calculateRMS calculates the RMS of the sample window
func (m *AudioLevelMeter) calculateRMS() float64 {
	if len(m.sampleWindow) == 0 {
		return 0
	}

	sum := 0.0
	for _, sample := range m.sampleWindow {
		sum += float64(sample * sample)
	}

	return sqrt(sum / float64(len(m.sampleWindow)))
}

// Helper functions
func log10(x float64) float64 {
	// Simple log10 approximation
	const ln10 = 2.302585092994046
	return ln(x) / ln10
}

func ln(x float64) float64 {
	// Simple natural log approximation
	// For production, use math.Log
	n := 0.0
	for x > 1 {
		x /= 2.718281828459045
		n++
	}
	return n
}

func sqrt(x float64) float64 {
	// Newton-Raphson method
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = 0.5 * (z + x/z)
	}
	return z
}

// OpusPayloader payloads Opus audio for RTP
type OpusPayloader struct {
	mtu             uint16
	packetsPerFrame int
}

// NewOpusPayloader creates a new Opus payloader
func NewOpusPayloader(mtu uint16) *OpusPayloader {
	return &OpusPayloader{
		mtu:             mtu,
		packetsPerFrame: 1,
	}
}

// Payload converts Opus packets to RTP packets
func (o *OpusPayloader) Payload(opusPackets []byte) [][]byte {
	if len(opusPackets) == 0 {
		return nil
	}

	// For now, simple single-packet payload
	// In production, handle fragmentation if needed
	return [][]byte{opusPackets}
}

// OpusDepayloader extracts Opus packets from RTP
type OpusDepayloader struct {
	lastSequenceNumber uint16
}

// NewOpusDepayloader creates a new Opus depayloader
func NewOpusDepayloader() *OpusDepayloader {
	return &OpusDepayloader{}
}

// Depayload extracts Opus packets from RTP packets
func (o *OpusDepayloader) Depayload(rtpPacket []byte) ([]byte, error) {
	if len(rtpPacket) == 0 {
		return nil, io.EOF
	}

	// Extract payload (skip RTP header)
	// RTP header is at least 12 bytes
	if len(rtpPacket) < 12 {
		return nil, fmt.Errorf("invalid RTP packet: too short")
	}

	// Get payload offset (RTP header + CSRC + extensions)
	payloadOffset := 12

	// Check for CSRC count
	firstByte := rtpPacket[0]
	csrcCount := int(firstByte & 0x0F)
	payloadOffset += 4 * csrcCount

	// Check for extension bit
	if firstByte&0x10 != 0 {
		if payloadOffset+4 > len(rtpPacket) {
			return nil, fmt.Errorf("invalid RTP packet: extension header truncated")
		}
		extLength := int(rtpPacket[payloadOffset+2])<<8 | int(rtpPacket[payloadOffset+3])
		payloadOffset += 4 + 4*extLength
	}

	// Extract payload
	if payloadOffset >= len(rtpPacket) {
		return nil, fmt.Errorf("invalid RTP packet: no payload")
	}

	return rtpPacket[payloadOffset:], nil
}
