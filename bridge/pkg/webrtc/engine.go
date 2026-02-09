package webrtc

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/turn"
	"github.com/pion/webrtc/v3"
)

// Engine manages WebRTC peer connections and media handling
// All WebRTC operations happen in the Bridge, not in containers
type Engine struct {
	config       EngineConfig
	mediaAPI     *webrtc.API
	mu           sync.RWMutex
	connections  map[string]*PeerConnectionWrapper
	stopChan     chan struct{}
	wg           sync.WaitGroup
	turnManager  *turn.Manager // TURN credential manager
}

// EngineConfig holds configuration for the WebRTC engine
type EngineConfig struct {
	// ICE servers configuration
	ICEServers []webrtc.ICEServer

	// Codec preferences
	AudioCodecs []webrtc.RTPCodecParameters

	// Configuration for peer connections
	Configuration webrtc.Configuration

	// Media configuration
	MediaConfig MediaConfig
}

// MediaConfig holds media-related configuration
type MediaConfig struct {
	SampleRate     uint32 // Audio sample rate (e.g., 48000)
	Channels       uint8  // Number of audio channels (1 = mono, 2 = stereo)
	Bitrate        uint32 // Target bitrate in bps
	PayloadType    uint8  // RTP payload type for audio
}

// DefaultEngineConfig returns the default engine configuration
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
		AudioCodecs: []webrtc.RTPCodecParameters{},
		Configuration: webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{
				{
					URLs: []string{"stun:stun.l.google.com:19302"},
				},
			},
		},
		MediaConfig: MediaConfig{
			SampleRate:  48000,
			Channels:    1,
			Bitrate:     64000,
			PayloadType: 111,
		},
	}
}

// PeerConnectionWrapper wraps a WebRTC PeerConnection with session metadata
type PeerConnectionWrapper struct {
	pc            *webrtc.PeerConnection
	sessionID     string
	audioTrack    *webrtc.TrackLocalStaticSample
	dataChannel   *webrtc.DataChannel
	onICECandidate func(candidate *webrtc.ICECandidate)
	onTrack       func(track *webrtc.TrackRemote)
	onDataChannel func(dc *webrtc.DataChannel)
	closeOnce     sync.Once
	closed        bool
}

// NewEngine creates a new WebRTC engine with the given configuration
func NewEngine(config EngineConfig) (*Engine, error) {
	// Create WebRTC API with default configuration
	mediaEngine := &webrtc.MediaEngine{}

	// Setup audio codecs
	// Use Opus as the preferred codec
	if err := mediaEngine.RegisterCodec(webrtc.RTPCodecParameters{
		RTPCodecCapability: webrtc.RTPCodecCapability{
			MimeType:     webrtc.MimeTypeOpus,
			ClockRate:    config.MediaConfig.SampleRate,
			Channels:     config.MediaConfig.Channels,
			SDPFmtpLine:  "minptime=10;useinbandfec=1",
		},
		PayloadType: config.MediaConfig.PayloadType,
	}, webrtc.RTPCodecTypeAudio); err != nil {
		return nil, fmt.Errorf("failed to register codec: %w", err)
	}

	// Create API
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithSettingEngine(webrtc.SettingEngine{}),
	)

	return &Engine{
		config:      config,
		mediaAPI:    api,
		connections: make(map[string]*PeerConnectionWrapper),
		stopChan:    make(chan struct{}),
	}, nil
}

// CreatePeerConnection creates a new WebRTC peer connection for a session
func (e *Engine) CreatePeerConnection(sessionID string) (*PeerConnectionWrapper, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if connection already exists
	if _, exists := e.connections[sessionID]; exists {
		return nil, fmt.Errorf("peer connection already exists for session: %s", sessionID)
	}

	// Create peer connection
	peerConnection, err := e.mediaAPI.NewPeerConnection(e.config.Configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create peer connection: %w", err)
	}

	// Create local audio track for sending audio to the client
	audioTrack, err := webrtc.NewTrackLocalStaticSample(
		webrtc.RTPCodecCapability{
			MimeType:    webrtc.MimeTypeOpus,
			ClockRate:   e.config.MediaConfig.SampleRate,
			Channels:    e.config.MediaConfig.Channels,
			SDPFmtpLine: "minptime=10;useinbandfec=1",
		},
		"audio",
		"armorclaw-audio",
	)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to create audio track: %w", err)
	}

	// Add track to peer connection
	rtpSender, err := peerConnection.AddTrack(audioTrack)
	if err != nil {
		peerConnection.Close()
		return nil, fmt.Errorf("failed to add track: %w", err)
	}

	// Create wrapper
	wrapper := &PeerConnectionWrapper{
		pc:         peerConnection,
		sessionID:  sessionID,
		audioTrack: audioTrack,
	}

	// Set up ICE candidate handler
	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if wrapper.onICECandidate != nil {
			wrapper.onICECandidate(candidate)
		}
	})

	// Set up track handler (for receiving audio from client)
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if wrapper.onTrack != nil {
			wrapper.onTrack(track)
		}
	})

	// Set up data channel handler
	peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
		if wrapper.onDataChannel != nil {
			wrapper.onDataChannel(dc)
		}
	})

	// Read RTP from the track to avoid blocking
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.readRTP(rtpSender, track)
	}()

	// Store connection
	e.connections[sessionID] = wrapper

	return wrapper, nil
}

// readRTP reads RTP packets from the track
func (e *Engine) readRTP(rtpSender *webrtc.RTPSender, track *webrtc.TrackRemote) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-e.stopChan:
			return
		default:
			_, _, err := rtpSender.Read(buf, nil)
			if err != nil {
				if err == io.EOF {
					return
				}
				continue
			}
			// Process incoming RTP packets (audio from client)
			// In a full implementation, this would decode and forward to the agent
		}
	}
}

// GetPeerConnection retrieves an existing peer connection
func (e *Engine) GetPeerConnection(sessionID string) (*PeerConnectionWrapper, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	wrapper, exists := e.connections[sessionID]
	return wrapper, exists
}

// ClosePeerConnection closes a peer connection
func (e *Engine) ClosePeerConnection(sessionID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	wrapper, exists := e.connections[sessionID]
	if !exists {
		return fmt.Errorf("peer connection not found for session: %s", sessionID)
	}

	wrapper.closeOnce.Do(func() {
		wrapper.closed = true
		if wrapper.pc != nil {
			wrapper.pc.Close()
		}
	})

	delete(e.connections, sessionID)
	return nil
}

// CreateOffer creates an SDP offer
func (pcw *PeerConnectionWrapper) CreateOffer() (string, error) {
	offer, err := pcw.pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create offer: %w", err)
	}

	// Set local description
	if err := pcw.pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("failed to set local description: %w", err)
	}

	return offer.SDP, nil
}

// CreateAnswer creates an SDP answer
func (pcw *PeerConnectionWrapper) CreateAnswer(offerSDP string) (string, error) {
	// Set remote description
	offer := webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offerSDP,
	}

	if err := pcw.pc.SetRemoteDescription(offer); err != nil {
		return "", fmt.Errorf("failed to set remote description: %w", err)
	}

	// Create answer
	answer, err := pcw.pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("failed to create answer: %w", err)
	}

	// Set local description
	if err := pcw.pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("failed to set local description: %w", err)
	}

	return answer.SDP, nil
}

// SetRemoteDescription sets the remote SDP description
func (pcw *PeerConnectionWrapper) SetRemoteDescription(sdpType, sdp string) error {
	var sd webrtc.SessionDescription

	switch sdpType {
	case "offer":
		sd = webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  sdp,
		}
	case "answer":
		sd = webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  sdp,
		}
	default:
		return fmt.Errorf("unknown SDP type: %s", sdpType)
	}

	return pcw.pc.SetRemoteDescription(sd)
}

// AddICECandidate adds an ICE candidate to the peer connection
func (pcw *PeerConnectionWrapper) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	return pcw.pc.AddICECandidate(candidate)
}

// WriteAudio writes audio samples to the audio track (sends to client)
func (pcw *PeerConnectionWrapper) WriteAudio(sample []byte, sampleRate uint32) error {
	if pcw.closed {
		return fmt.Errorf("peer connection is closed")
	}

	// Create audio sample from raw PCM data
	// In a full implementation, this would encode to Opus first
	sample := webrtc.Sample{
		Data:    sample,
		Duration: time.Duration(len(sample)) * time.Second / time.Duration(sampleRate),
	}

	return pcw.audioTrack.WriteSample(sample)
}

// OnICECandidate sets the handler for ICE candidate events
func (pcw *PeerConnectionWrapper) OnICECandidate(handler func(*webrtc.ICECandidate)) {
	pcw.onICECandidate = handler
}

// OnTrack sets the handler for remote track events (incoming audio)
func (pcw *PeerConnectionWrapper) OnTrack(handler func(*webrtc.TrackRemote)) {
	pcw.onTrack = handler
}

// OnDataChannel sets the handler for data channel events
func (pcw *PeerConnectionWrapper) OnDataChannel(handler func(*webrtc.DataChannel)) {
	pcw.onDataChannel = handler
}

// ConnectionState returns the current connection state
func (pcw *PeerConnectionWrapper) ConnectionState() webrtc.PeerConnectionState {
	return pcw.pc.ConnectionState()
}

// ICEConnectionState returns the current ICE connection state
func (pcw *PeerConnectionWrapper) ICEConnectionState() webrtc.ICEConnectionState {
	return pcw.pc.ICEConnectionState()
}

// Stop stops the WebRTC engine and closes all connections
func (e *Engine) Stop() {
	close(e.stopChan)

	e.mu.Lock()
	defer e.mu.Unlock()

	// Close all peer connections
	for sessionID, wrapper := range e.connections {
		e.ClosePeerConnection(sessionID)
	}

	e.wg.Wait()
}

// SetTURNServers configures TURN servers for the engine with ephemeral credentials
func (e *Engine) SetTURNServers(turnURL, turnUsername, turnPassword string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Add TURN server to ICE servers
	e.config.Configuration.ICEServers = append(e.config.Configuration.ICEServers,
		webrtc.ICEServer{
			URLs:       []string{turnURL},
			Username:   turnUsername,
			Credential: turnPassword,
		},
	)
}

// SetTURNServersWithManager configures TURN servers using the TURN manager
// This generates ephemeral credentials per session
func (e *Engine) SetTURNServersWithManager(sessionID string, ttl time.Duration) ([]webrtc.ICEServer, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.turnManager == nil {
		return nil, fmt.Errorf("TURN manager not initialized")
	}

	// Generate ephemeral TURN credentials
	turnCreds, err := e.turnManager.GenerateTURNCredentials(sessionID, ttl)
	if err != nil {
		return nil, fmt.Errorf("failed to generate TURN credentials: %w", err)
	}

	// Convert to Pion ICE servers
	iceServers := make([]webrtc.ICEServer, 0, len(turnCreds))
	for _, cred := range turnCreds {
		iceServers = append(iceServers, webrtc.ICEServer{
			URLs:       []string{cred.TURNServer},
			Username:   cred.Username,
			Credential: cred.Password,
		})
	}

	// Add to configuration
	e.config.Configuration.ICEServers = append(e.config.Configuration.ICEServers, iceServers...)

	return iceServers, nil
}

// SetTURNManager sets the TURN credential manager
func (e *Engine) SetTURNManager(manager *turn.Manager) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.turnManager = manager
}

// CreateDataChannel creates a new data channel on the peer connection
func (pcw *PeerConnectionWrapper) CreateDataChannel(label string, options *webrtc.DataChannelInit) (*webrtc.DataChannel, error) {
	return pcw.pc.CreateDataChannel(label, options)
}

// Errors
var (
	// ErrPeerConnectionNotFound is returned when a peer connection doesn't exist
	ErrPeerConnectionNotFound = fmt.Errorf("peer connection not found")

	// ErrPeerConnectionClosed is returned when operating on a closed connection
	ErrPeerConnectionClosed = fmt.Errorf("peer connection is closed")
)
