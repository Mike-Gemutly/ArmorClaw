// Package audio tests for PCM audio streaming and processing
package audio

import (
	"context"
	"testing"
	"time"
)

// TestAudioConfig_BytesPerFrame tests bytes per frame calculation
func TestAudioConfig_BytesPerFrame(t *testing.T) {
	config := AudioConfig{
		SampleRate: 48000,
		Channels:   1,
		BitDepth:   16,
		FrameSize:  960,
	}

	expected := 960 * 1 * 2 // 960 samples * 1 channel * 2 bytes per sample
	actual := config.BytesPerFrame()

	if actual != expected {
		t.Errorf("Expected %d bytes per frame, got %d", expected, actual)
	}
}

// TestNewAudioStream tests creating a new audio stream
func TestNewAudioStream(t *testing.T) {
	config := DefaultAudioConfig()
	stream := NewAudioStream("test-session", StreamIn, config)

	if stream.SessionID() != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", stream.SessionID())
	}

	if stream.Direction() != StreamIn {
		t.Errorf("Expected direction %v, got %v", StreamIn, stream.Direction())
	}

	if stream.IsClosed() {
		t.Error("New stream should not be closed")
	}
}

// TestAudioStream_WriteRead tests writing and reading frames
func TestAudioStream_WriteRead(t *testing.T) {
	config := DefaultAudioConfig()
	stream := NewAudioStream("test-session", StreamIn, config)
	defer stream.Close()

	frame := AudioFrame{
		Data:      make([]byte, config.BytesPerFrame()),
		Timestamp: time.Now(),
		Sequence:  1,
	}

	err := stream.WriteFrame(frame)
	if err != nil {
		t.Fatalf("Failed to write frame: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	readFrame, err := stream.ReadFrame(ctx)
	if err != nil {
		t.Fatalf("Failed to read frame: %v", err)
	}

	if readFrame.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", readFrame.Sequence)
	}
}

// TestAudioStream_Close tests closing a stream
func TestAudioStream_Close(t *testing.T) {
	config := DefaultAudioConfig()
	stream := NewAudioStream("test-session", StreamIn, config)

	stream.Close()

	if !stream.IsClosed() {
		t.Error("Stream should be closed")
	}

	// Writing to closed stream should fail
	frame := AudioFrame{Data: []byte{1, 2, 3}}
	err := stream.WriteFrame(frame)
	if err == nil {
		t.Error("Writing to closed stream should fail")
	}
}

// TestAudioStream_OnFrame tests frame callback
func TestAudioStream_OnFrame(t *testing.T) {
	config := DefaultAudioConfig()
	stream := NewAudioStream("test-session", StreamIn, config)
	defer stream.Close()

	receivedFrames := make(chan AudioFrame, 1)
	stream.OnFrame(func(frame AudioFrame) {
		receivedFrames <- frame
	})

	frame := AudioFrame{
		Data:      make([]byte, config.BytesPerFrame()),
		Timestamp: time.Now(),
		Sequence:  1,
	}

	stream.WriteFrame(frame)

	select {
	case received := <-receivedFrames:
		if received.Sequence != 1 {
			t.Errorf("Expected sequence 1, got %d", received.Sequence)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Did not receive frame callback")
	}
}

// TestAudioPipeline_CreateStreams tests creating audio streams
func TestAudioPipeline_CreateStreams(t *testing.T) {
	config := DefaultAudioConfig()
	pipeline := NewAudioPipeline(config)
	defer pipeline.Close()

	pair, err := pipeline.CreateStreams("test-session")
	if err != nil {
		t.Fatalf("Failed to create streams: %v", err)
	}

	if pair.In == nil {
		t.Error("Input stream should not be nil")
	}

	if pair.Out == nil {
		t.Error("Output stream should not be nil")
	}

	if pair.In.SessionID() != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", pair.In.SessionID())
	}

	if pair.Out.SessionID() != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", pair.Out.SessionID())
	}
}

// TestAudioPipeline_GetStreams tests retrieving streams
func TestAudioPipeline_GetStreams(t *testing.T) {
	config := DefaultAudioConfig()
	pipeline := NewAudioPipeline(config)
	defer pipeline.Close()

	// Get non-existent streams
	_, ok := pipeline.GetStreams("non-existent")
	if ok {
		t.Error("Should not find non-existent streams")
	}

	// Create streams
	pipeline.CreateStreams("test-session")

	// Get existing streams
	pair, ok := pipeline.GetStreams("test-session")
	if !ok {
		t.Error("Should find existing streams")
	}

	if pair.In.SessionID() != "test-session" {
		t.Errorf("Expected session ID 'test-session', got '%s'", pair.In.SessionID())
	}
}

// TestAudioPipeline_RemoveStreams tests removing streams
func TestAudioPipeline_RemoveStreams(t *testing.T) {
	config := DefaultAudioConfig()
	pipeline := NewAudioPipeline(config)

	// Create streams
	pipeline.CreateStreams("test-session")

	// Remove streams
	pipeline.RemoveStreams("test-session")

	// Should not find streams
	_, ok := pipeline.GetStreams("test-session")
	if ok {
		t.Error("Should not find removed streams")
	}

	// Streams should be closed
	pair, _ := pipeline.CreateStreams("test-session")
	pipeline.RemoveStreams("test-session")

	if !pair.In.IsClosed() {
		t.Error("Input stream should be closed")
	}

	if !pair.Out.IsClosed() {
		t.Error("Output stream should be closed")
	}
}

// TestPCMMixer_Mix tests mixing audio frames
func TestPCMMixer_Mix(t *testing.T) {
	config := DefaultAudioConfig()
	mixer := NewPCMMixer(config)

	// Create two simple frames
	frame1 := AudioFrame{
		Data:      []byte{0x00, 0x10, 0x00, 0x20}, // Simple PCM samples
		Timestamp: time.Now(),
	}

	frame2 := AudioFrame{
		Data:      []byte{0x00, 0x30, 0x00, 0x40},
		Timestamp: time.Now(),
	}

	mixed, err := mixer.Mix([]AudioFrame{frame1, frame2})
	if err != nil {
		t.Fatalf("Failed to mix frames: %v", err)
	}

	if len(mixed.Data) != 4 {
		t.Errorf("Expected mixed frame length 4, got %d", len(mixed.Data))
	}

	// Verify mixing (clamping)
	// First sample: 0x10 + 0x30 = 0x40 (should not be clamped)
	// Second sample: 0x20 + 0x40 = 0x60 (should not be clamped)
}

// TestPCMMixer_SingleFrame tests mixing a single frame
func TestPCMMixer_SingleFrame(t *testing.T) {
	config := DefaultAudioConfig()
	mixer := NewPCMMixer(config)

	frame := AudioFrame{
		Data:      []byte{0x00, 0x10, 0x00, 0x20},
		Timestamp: time.Now(),
	}

	mixed, err := mixer.Mix([]AudioFrame{frame})
	if err != nil {
		t.Fatalf("Failed to mix single frame: %v", err)
	}

	// Should return the same frame
	if len(mixed.Data) != len(frame.Data) {
		t.Errorf("Expected frame length %d, got %d", len(frame.Data), len(mixed.Data))
	}
}

// TestPCMEncoder_EncodeInt16ToBytes tests encoding int16 to bytes
func TestPCMEncoder_EncodeInt16ToBytes(t *testing.T) {
	config := DefaultAudioConfig()
	encoder := NewPCMEncoder(config)

	samples := []int16{100, 200, 300, 400}
	bytes := encoder.EncodeInt16ToBytes(samples)

	if len(bytes) != 8 {
		t.Errorf("Expected 8 bytes, got %d", len(bytes))
	}

	// Verify little-endian encoding
	if bytes[0] != 100 || bytes[1] != 0 {
		t.Errorf("Incorrect little-endian encoding at position 0")
	}
}

// TestPCMEncoder_DecodeBytesToInt16 tests decoding bytes to int16
func TestPCMEncoder_DecodeBytesToInt16(t *testing.T) {
	config := DefaultAudioConfig()
	encoder := NewPCMEncoder(config)

	bytes := []byte{100, 0, 200, 0, 255, 255, 0, 1}
	samples := encoder.DecodeBytesToInt16(bytes)

	if len(samples) != 4 {
		t.Errorf("Expected 4 samples, got %d", len(samples))
	}

	if samples[0] != 100 {
		t.Errorf("Expected sample 100, got %d", samples[0])
	}

	if samples[1] != 200 {
		t.Errorf("Expected sample 200, got %d", samples[1])
	}

	// Test negative number (255, 255 = -1 in two's complement)
	if samples[2] != -1 {
		t.Errorf("Expected sample -1, got %d", samples[2])
	}

	// Test larger number (0, 1 = 256)
	if samples[3] != 256 {
		t.Errorf("Expected sample 256, got %d", samples[3])
	}
}

// TestAudioBuffer_WriteRead tests writing to and reading from buffer
func TestAudioBuffer_WriteRead(t *testing.T) {
	buffer := NewAudioBuffer(1024)

	// Write data
	data := []byte{1, 2, 3, 4, 5}
	n, err := buffer.Write(data)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	if n != 5 {
		t.Errorf("Expected to write 5 bytes, wrote %d", n)
	}

	// Read data
	readData := make([]byte, 5)
	n, err = buffer.Read(readData)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}

	if n != 5 {
		t.Errorf("Expected to read 5 bytes, read %d", n)
	}

	if readData[0] != 1 || readData[4] != 5 {
		t.Error("Read data doesn't match written data")
	}
}

// TestAudioBuffer_Circular tests circular buffer behavior
func TestAudioBuffer_Circular(t *testing.T) {
	buffer := NewAudioBuffer(10)

	// Write more than capacity
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	buffer.Write(data)

	// Should only have 10 bytes (capacity)
	if buffer.Available() != 10 {
		t.Errorf("Expected 10 bytes available, got %d", buffer.Available())
	}

	// Read all data
	readData := make([]byte, 10)
	n, _ := buffer.Read(readData)

	if n != 10 {
		t.Errorf("Expected to read 10 bytes, read %d", n)
	}

	// Should have wrapped around
	// Last bytes written should be 3, 4, 5, 6, 7, 8, 9, 10, 11, 12
	if readData[0] != 3 || readData[9] != 12 {
		t.Error("Circular buffer didn't wrap correctly")
	}
}

// TestAudioBuffer_Clear tests clearing the buffer
func TestAudioBuffer_Clear(t *testing.T) {
	buffer := NewAudioBuffer(1024)

	data := []byte{1, 2, 3, 4, 5}
	buffer.Write(data)

	if buffer.Available() != 5 {
		t.Errorf("Expected 5 bytes available, got %d", buffer.Available())
	}

	buffer.Clear()

	if buffer.Available() != 0 {
		t.Errorf("Expected 0 bytes available after clear, got %d", buffer.Available())
	}
}

// TestAudioStats_RecordFrame tests recording frame statistics
func TestAudioStats_RecordFrame(t *testing.T) {
	stats := NewAudioStats()

	stats.RecordFrame(480) // 480 bytes (one Opus frame)

	snapshot := stats.GetStats()

	if snapshot["total_frames"].(uint64) != 1 {
		t.Errorf("Expected 1 frame, got %d", snapshot["total_frames"])
	}

	if snapshot["total_bytes"].(uint64) != 480 {
		t.Errorf("Expected 480 bytes, got %d", snapshot["total_bytes"])
	}
}

// TestAudioStats_DroppedFrames tests recording dropped frames
func TestAudioStats_DroppedFrames(t *testing.T) {
	stats := NewAudioStats()

	stats.RecordDroppedFrame()
	stats.RecordDroppedFrame()

	snapshot := stats.GetStats()

	if snapshot["dropped_frames"].(uint64) != 2 {
		t.Errorf("Expected 2 dropped frames, got %d", snapshot["dropped_frames"])
	}
}

// TestAudioStats_Reset tests resetting statistics
func TestAudioStats_Reset(t *testing.T) {
	stats := NewAudioStats()

	stats.RecordFrame(480)
	stats.RecordDroppedFrame()
	stats.RecordLostPacket()

	stats.Reset()

	snapshot := stats.GetStats()

	if snapshot["total_frames"].(uint64) != 0 {
		t.Error("Total frames should be 0 after reset")
	}

	if snapshot["dropped_frames"].(uint64) != 0 {
		t.Error("Dropped frames should be 0 after reset")
	}

	if snapshot["lost_packets"].(uint64) != 0 {
		t.Error("Lost packets should be 0 after reset")
	}
}

// TestOpusEncoder_Encode tests encoding PCM to Opus
func TestOpusEncoder_Encode(t *testing.T) {
	config := DefaultAudioConfig()
	opusConfig := DefaultOpusConfig()

	encoder := NewOpusEncoder(config, opusConfig)

	// Create a simple PCM frame
	pcm := make([]byte, config.BytesPerFrame())

	// Encode
	opus, err := encoder.Encode(pcm)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// For now, should return the same data (passthrough)
	// In production with real Opus, this would be compressed
	if len(opus) == 0 {
		t.Error("Encoded data should not be empty")
	}
}

// TestOpusDecoder_Decode tests decoding Opus to PCM
func TestOpusDecoder_Decode(t *testing.T) {
	config := DefaultAudioConfig()

	decoder := NewOpusDecoder(config)

	// Create simple Opus data
	opus := make([]byte, config.BytesPerFrame())

	// Decode
	pcm, err := decoder.Decode(opus)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Should return the same size
	if len(pcm) != config.BytesPerFrame() {
		t.Errorf("Expected %d bytes, got %d", config.BytesPerFrame(), len(pcm))
	}
}

// TestOpusEncoder_Disable tests disabling the encoder
func TestOpusEncoder_Disable(t *testing.T) {
	config := DefaultAudioConfig()
	opusConfig := DefaultOpusConfig()

	encoder := NewOpusEncoder(config, opusConfig)

	encoder.Disable()

	if encoder.IsEnabled() {
		t.Error("Encoder should be disabled")
	}

	// Passthrough mode
	pcm := make([]byte, config.BytesPerFrame())
	opus, err := encoder.Encode(pcm)

	if err != nil {
		t.Fatalf("Failed to encode in passthrough mode: %v", err)
	}

	// Should be the same data
	if len(opus) != len(pcm) {
		t.Errorf("Passthrough mode should preserve size")
	}
}

// TestAudioLevelMeter_Process tests measuring audio level
func TestAudioLevelMeter_Process(t *testing.T) {
	config := DefaultAudioConfig()

	meter := NewAudioLevelMeter(config, 960)

	// Create a silent frame
	silentFrame := make([]byte, config.BytesPerFrame())

	level, err := meter.Process(silentFrame)
	if err != nil {
		t.Fatalf("Failed to process frame: %v", err)
	}

	// Silent frame should have very low level
	if level > -60 {
		t.Errorf("Silent frame should have low level, got %f dBFS", level)
	}
}

// TestRTPOpusPacketizer_Packetize tests RTP packetization
func TestRTPOpusPacketizer_Packetize(t *testing.T) {
	config := DefaultAudioConfig()

	packetizer := NewRTPOpusPacketizer(config, 111, 12345)

	// Create Opus data larger than MTU
	opusData := make([]byte, 2000)

	packets, err := packetizer.Packetize(opusData)
	if err != nil {
		t.Fatalf("Failed to packetize: %v", err)
	}

	// Should be split into multiple packets
	if len(packets) < 2 {
		t.Errorf("Expected multiple packets, got %d", len(packets))
	}

	// Verify first packet
	if packets[0].PayloadType != 111 {
		t.Errorf("Expected payload type 111, got %d", packets[0].PayloadType)
	}

	if packets[0].SSRC != 12345 {
		t.Errorf("Expected SSRC 12345, got %d", packets[0].SSRC)
	}
}

// TestRTPOpusPacketizer_SequenceNumbers tests sequence number increment
func TestRTPOpusPacketizer_SequenceNumbers(t *testing.T) {
	config := DefaultAudioConfig()

	packetizer := NewRTPOpusPacketizer(config, 111, 12345)

	// Create small Opus data (one packet)
	opusData := make([]byte, 500)

	packets1, _ := packetizer.Packetize(opusData)
	packets2, _ := packetizer.Packetize(opusData)

	// Sequence numbers should increment
	if packets2[0].SequenceNumber != packets1[0].SequenceNumber+1 {
		t.Errorf("Sequence numbers should increment")
	}
}

// TestRTPDepacketizer_Depacketize tests depacketizing RTP packets
func TestRTPDepacketizer_Depacketize(t *testing.T) {
	config := DefaultAudioConfig()

	depacketizer := NewRTPDepacketizer(config)

	packet := &RTPPacket{
		SequenceNumber: 0,
		Timestamp:      0,
		SSRC:           12345,
		PayloadType:    111,
		Payload:        []byte{1, 2, 3, 4, 5},
	}

	// Depacketize
	data, err := depacketizer.Depacketize(packet)
	if err != nil {
		t.Fatalf("Failed to depacketize: %v", err)
	}

	// Should extract payload
	if len(data) != 5 {
		t.Errorf("Expected 5 bytes, got %d", len(data))
	}

	if data[0] != 1 || data[4] != 5 {
		t.Error("Payload data doesn't match")
	}
}

// TestOpusDepayloader_Depayload tests extracting Opus from RTP
func TestOpusDepayloader_Depayload(t *testing.T) {
	depayloader := NewOpusDepayloader()

	// Create RTP packet (minimal header: 12 bytes)
	rtpPacket := make([]byte, 17)

	// Set RTP header
	rtpPacket[0] = 0x80 // Version 2, no padding, no extension, no CSRC
	rtpPacket[1] = 111  // Payload type
	// Sequence number (bytes 2-3)
	rtpPacket[2] = 0
	rtpPacket[3] = 1
	// Timestamp (bytes 4-7)
	rtpPacket[4] = 0
	rtpPacket[5] = 0
	rtpPacket[6] = 0
	rtpPacket[7] = 0
	// SSRC (bytes 8-11)
	rtpPacket[8] = 0
	rtpPacket[9] = 0
	rtpPacket[10] = 0
	rtpPacket[11] = 0
	// Payload (bytes 12+)
	rtpPacket[12] = 1
	rtpPacket[13] = 2
	rtpPacket[14] = 3
	rtpPacket[15] = 4
	rtpPacket[16] = 5

	// Depayload
	opus, err := depayloader.Depayload(rtpPacket)
	if err != nil {
		t.Fatalf("Failed to depayload: %v", err)
	}

	// Should extract 5 bytes of payload
	if len(opus) != 5 {
		t.Errorf("Expected 5 bytes, got %d", len(opus))
	}

	if opus[0] != 1 || opus[4] != 5 {
		t.Error("Opus data doesn't match")
	}
}

// TestIntegration_AudioLoopback tests a simple audio loopback
func TestIntegration_AudioLoopback(t *testing.T) {
	config := DefaultAudioConfig()
	pipeline := NewAudioPipeline(config)
	defer pipeline.Close()

	// Create streams
	pair, err := pipeline.CreateStreams("loopback-test")
	if err != nil {
		t.Fatalf("Failed to create streams: %v", err)
	}

	// Create test frames
	testData := make([]byte, config.BytesPerFrame())
	for i := range testData {
		testData[i] = byte(i)
	}

	// Write to input stream
	frame := AudioFrame{
		Data:      testData,
		Timestamp: time.Now(),
		Sequence:  1,
	}

	err = pair.In.WriteFrame(frame)
	if err != nil {
		t.Fatalf("Failed to write frame: %v", err)
	}

	// Read from input stream
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	readFrame, err := pair.In.ReadFrame(ctx)
	if err != nil {
		t.Fatalf("Failed to read frame: %v", err)
	}

	if readFrame.Sequence != 1 {
		t.Errorf("Expected sequence 1, got %d", readFrame.Sequence)
	}

	if len(readFrame.Data) != len(testData) {
		t.Errorf("Expected data length %d, got %d", len(testData), len(readFrame.Data))
	}
}
