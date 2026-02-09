package webrtc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// MessageType represents the type of signaling message
type MessageType string

const (
	// MessageTypeOffer is an SDP offer from the client
	MessageTypeOffer MessageType = "offer"
	// MessageTypeAnswer is an SDP answer from the bridge
	MessageTypeAnswer MessageType = "answer"
	// MessageTypeICE is an ICE candidate
	MessageTypeICE MessageType = "ice"
	// MessageTypeBye signals the end of the session
	MessageTypeBye MessageType = "bye"
	// MessageTypeError is an error message
	MessageTypeError MessageType = "error"
)

// SignalingMessage represents a message sent over the signaling channel
type SignalingMessage struct {
	Type      MessageType         `json:"type"`
	SessionID string              `json:"session_id"`
	Payload   json.RawMessage     `json:"payload"`
	Token     string              `json:"token,omitempty"` // Call session token for auth
	Timestamp time.Time           `json:"timestamp"`
}

// SignalingServer handles WebSocket connections for WebRTC signaling
// It provides secure, authenticated signaling for SDP and ICE exchange
type SignalingServer struct {
	addr      string           // WebSocket listen address (e.g., "0.0.0.0:8443")
	path      string           // WebSocket path (e.g., "/webrtc")
	tlsCert   string           // Path to TLS certificate file
	tlsKey    string           // Path to TLS key file
	upgrader  websocket.Upgrader
	hub       *Hub             // Connection hub
	sessions  *SessionManager  // Session manager for auth
	tokens    *TokenManager    // Token manager for validation
	mu        sync.RWMutex
	running   bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// Hub maintains active client connections and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// Client represents a WebSocket client connection
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte
	sessionID string // Associated session ID
}

// NewSignalingServer creates a new signaling server
func NewSignalingServer(addr, path string, sessions *SessionManager, tokens *TokenManager) *SignalingServer {
	return &SignalingServer{
		addr:     addr,
		path:     path,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
		},
		hub:      NewHub(),
		sessions: sessions,
		tokens:   tokens,
		stopChan: make(chan struct{}),
	}
}

// SetTLS sets the TLS certificate and key for secure WebSocket (wss://)
func (ss *SignalingServer) SetTLS(certFile, keyFile string) {
	ss.tlsCert = certFile
	ss.tlsKey = keyFile
}

// Start starts the signaling server
func (ss *SignalingServer) Start() error {
	ss.mu.Lock()
	if ss.running {
		ss.mu.Unlock()
		return fmt.Errorf("signaling server already running")
	}
	ss.running = true
	ss.mu.Unlock()

	// Start hub
	ss.hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc(ss.path, ss.handleWebSocket)

	server := &http.Server{
		Addr:    ss.addr,
		Handler: mux,
	}

	var err error
	if ss.tlsCert != "" && ss.tlsKey != "" {
		// Start HTTPS server (wss://)
		ss.wg.Add(1)
		go func() {
			defer ss.wg.Done()
			err = server.ListenAndServeTLS(ss.tlsCert, ss.tlsKey)
		}()
	} else {
		// Start HTTP server (ws://) - not recommended for production
		ss.wg.Add(1)
		go func() {
			defer ss.wg.Done()
			err = server.ListenAndServe()
		}()
	}

	return err
}

// Stop stops the signaling server
func (ss *SignalingServer) Stop() {
	ss.mu.Lock()
	if !ss.running {
		ss.mu.Unlock()
		return
	}
	ss.running = false
	ss.mu.Unlock()

	close(ss.stopChan)
	ss.wg.Wait()
}

// handleWebSocket handles incoming WebSocket connections
func (ss *SignalingServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket
	conn, err := ss.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Log error
		return
	}

	// Create client
	client := &Client{
		hub:  ss.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	// Register client
	ss.hub.register <- client

	// Start client pumps
	go client.writePump()
	go client.readPump(ss)
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump(server *SignalingServer) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log unexpected close
			}
			break
		}

		// Parse signaling message
		var msg SignalingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			// Send error response
			c.sendError("invalid_message_format")
			continue
		}

		// Set timestamp
		msg.Timestamp = time.Now()

		// Validate token if present
		if msg.Token != "" {
			token, err := TokenFromJSON(msg.Token)
			if err != nil {
				c.sendError("invalid_token")
				continue
			}

			// Validate token
			_, err = server.tokens.Validate(token)
			if err != nil {
				c.sendError("token_validation_failed")
				continue
			}

			// Store session ID
			c.sessionID = token.SessionID
		}

		// Handle message based on type
		switch msg.Type {
		case MessageTypeOffer, MessageTypeAnswer, MessageTypeICE:
			// Forward to session/peer
			// This will be handled by the WebRTC engine
			server.handleSignalingMessage(&msg, c)
		case MessageTypeBye:
			// End session
			if c.sessionID != "" {
				server.sessions.End(c.sessionID)
			}
		default:
			c.sendError("unknown_message_type")
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// sendError sends an error message to the client
func (c *Client) sendError(code string) {
	msg := SignalingMessage{
		Type:      MessageTypeError,
		Timestamp: time.Now(),
		Payload:   json.RawMessage(`{"code":"` + code + `"}`),
	}

	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
		close(c.send)
	}
}

// handleSignalingMessage processes a signaling message
func (ss *SignalingServer) handleSignalingMessage(msg *SignalingMessage, client *Client) {
	// Validate session exists
	if msg.SessionID == "" {
		msg.SessionID = client.sessionID
	}

	session, ok := ss.sessions.Get(msg.SessionID)
	if !ok {
		client.sendError("session_not_found")
		return
	}

	// Check session state
	if session.State == SessionEnded || session.State == SessionFailed || session.State == SessionExpired {
		client.sendError("session_ended")
		return
	}

	// Update session activity
	session.MarkActivity()

	// Store SDP
	if msg.Type == MessageTypeOffer {
		session.SDPOffer = string(msg.Payload)
	} else if msg.Type == MessageTypeAnswer {
		session.SDPAnswer = string(msg.Payload)
	}

	// Forward message to the appropriate peer
	// In a full implementation, this would route to the WebRTC engine
	// For now, we'll broadcast to all clients in the hub (for testing)
	ss.hub.broadcast <- msg.toJSON()
}

// toJSON converts the signaling message to JSON
func (sm *SignalingMessage) toJSON() []byte {
	data, _ := json.Marshal(sm)
	return data
}

// Send sends a message to a specific session
func (ss *SignalingServer) Send(sessionID string, msg *SignalingMessage) error {
	// Find client with this session ID
	ss.hub.mu.RLock()
	defer ss.hub.mu.RUnlock()

	for client := range ss.hub.clients {
		if client.sessionID == sessionID {
			select {
			case client.send <- msg.toJSON():
				return nil
			default:
				return fmt.Errorf("client send buffer full")
			}
		}
	}

	return fmt.Errorf("client not found for session: %s", sessionID)
}

// NewHub creates a new hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub's event loop
func (h *Hub) Run() {
	go func() {
		for {
			select {
			case client := <-h.register:
				h.mu.Lock()
				h.clients[client] = true
				h.mu.Unlock()
			case client := <-h.unregister:
				h.mu.Lock()
				if _, ok := h.clients[client]; ok {
					delete(h.clients, client)
					close(client.send)
				}
				h.mu.Unlock()
			case message := <-h.broadcast:
				h.mu.RLock()
				for client := range h.clients {
					select {
					case client.send <- message:
					default:
						close(client.send)
						delete(h.clients, client)
					}
				}
				h.mu.RUnlock()
			}
		}
	}()
}

// Errors
var (
	// ErrSignalingServerRunning is returned when trying to start an already running server
	ErrSignalingServerRunning = fmt.Errorf("signaling server already running")

	// ErrSignalingServerStopped is returned when trying to operate on a stopped server
	ErrSignalingServerStopped = fmt.Errorf("signaling server stopped")
)
