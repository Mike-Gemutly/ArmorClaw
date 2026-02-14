// Package secrets provides memory-only secret injection via Unix domain sockets.
// This implements P0-CRIT-3 fix: eliminates TOCTTOU vulnerability from file-based passing.
package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/keystore"
	"github.com/armorclaw/bridge/pkg/logger"
)

const (
	// SecretSocketTimeout is how long to wait for container to connect
	SecretSocketTimeout = 5 * time.Second

	// SecretSocketBufferSize is buffer size for socket writes
	SecretSocketBufferSize = 4096
)

var (
	// ErrSecretTimeout is returned when container doesn't connect in time
	ErrSecretTimeout = errors.New("container did not connect to secret socket")

	// ErrSecretWriteFailed is returned when writing secrets to socket fails
	ErrSecretWriteFailed = errors.New("failed to write secrets to socket")
)

// SecretInjector manages in-memory secret injection via Unix sockets.
// Secrets are never written to disk - only transmitted through socket.
type SecretInjector struct {
	socketDir string           // Directory for secret sockets
	sockets   map[string]*secretSession // Active socket sessions
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	securityLog *logger.SecurityLogger
}

// secretSession represents an active secret delivery session
type secretSession struct {
	socketPath  string
	credential  keystore.Credential
	ready       chan struct{} // Closed when socket is ready
	expiresAt   time.Time
	server      net.Listener
	mu           sync.Mutex
	closed       bool
}

// NewSecretInjector creates a new secret injector
func NewSecretInjector(socketDir string, secLog *logger.SecurityLogger) (*SecretInjector, error) {
	if err := os.MkdirAll(socketDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create socket directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &SecretInjector{
		socketDir:   socketDir,
		sockets:     make(map[string]*secretSession),
		ctx:          ctx,
		cancel:       cancel,
		securityLog:  secLog,
	}, nil
}

// InjectSecrets prepares a Unix socket for secret delivery and waits for container to connect.
// Returns the socket path that should be mounted into the container.
// The caller must call Cleanup() after container is started to remove the socket.
func (si *SecretInjector) InjectSecrets(containerName string, cred keystore.Credential) (string, error) {
	si.mu.Lock()
	defer si.mu.Unlock()

	// Generate unique socket path for this container
	socketPath := filepath.Join(si.socketDir, containerName+".sock")

	// Check for existing session
	if _, exists := si.sockets[containerName]; exists {
		return "", fmt.Errorf("secret session already exists for container: %s", containerName)
	}

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return "", fmt.Errorf("failed to create secret socket: %w", err)
	}

	// Set socket permissions (owner + group read/write)
	if err := os.Chmod(socketPath, 0660); err != nil {
		listener.Close()
		return "", fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Create session
	session := &secretSession{
		socketPath: socketPath,
		credential:  cred,
		ready:      make(chan struct{}),
		expiresAt:  time.Now().Add(SecretSocketTimeout),
		server:     listener,
	}

	si.sockets[containerName] = session

	// Start socket handler in background
	si.wg.Add(1)
	go si.handleSecretConnection(session)

	// Log secret injection (socket-based, no file created)
	si.securityLog.LogSecretInject(si.ctx, containerName, cred.ID,
		slog.String("injection_method", "unix_socket"),
		slog.String("socket_path", socketPath),
	)

	return socketPath, nil
}

// handleSecretConnection waits for container to connect and delivers secrets
func (si *SecretInjector) handleSecretConnection(session *secretSession) {
	defer si.wg.Done()
	defer si.cleanupSession(session)

	// Signal that socket is ready
	close(session.ready)

	// Wait for connection with timeout
	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)

	go func() {
		conn, err := session.server.Accept()
		if err != nil {
			errChan <- err
			return
		}
		connChan <- conn
	}()

	// Wait for connection or timeout
	select {
	case conn := <-connChan:
		defer conn.Close()
		si.deliverSecrets(session, conn)

	case err := <-errChan:
		if !errors.Is(err, net.ErrClosed) {
			slog.LogAttrs(si.ctx, "secret_socket_accept_error",
				slog.String("container", filepath.Base(session.socketPath)),
				slog.String("error", err.Error()),
			)
		}
		return

	case <-time.After(SecretSocketTimeout):
		slog.LogAttrs(si.ctx, "secret_socket_timeout",
			slog.String("container", filepath.Base(session.socketPath)),
		)
		return

	case <-si.ctx.Done():
		return
	}
}

// deliverSecrets sends credential data over the socket connection
func (si *SecretInjector) deliverSecrets(session *secretSession, conn net.Conn) {
	// Prepare secrets JSON (same format as file-based for compatibility)
	secretsJSON := map[string]interface{}{
		"provider":     session.credential.Provider,
		"token":        session.credential.Token,
		"display_name": session.credential.DisplayName,
	}

	secretsData, err := json.Marshal(secretsJSON)
	if err != nil {
		slog.LogAttrs(si.ctx, "secret_marshal_failed",
			slog.String("container", filepath.Base(session.socketPath)),
			slog.String("error", err.Error()),
		)
		return
	}

	// Write length prefix (4 bytes) for message framing
	length := len(secretsData)
	lengthPrefix := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
	}

	// Write length prefix
	if _, err := conn.Write(lengthPrefix); err != nil {
		slog.LogAttrs(si.ctx, "secret_write_length_failed",
			slog.String("container", filepath.Base(session.socketPath)),
			slog.String("error", err.Error()),
		)
		return
	}

	// Write secrets data
	totalWritten := 0
	for totalWritten < len(secretsData) {
		written, err := conn.Write(secretsData[totalWritten:])
		if err != nil {
			slog.LogAttrs(si.ctx, "secret_write_data_failed",
				slog.String("container", filepath.Base(session.socketPath)),
				slog.Int("written_bytes", totalWritten),
				slog.String("error", err.Error()),
			)
			return
		}
		totalWritten += written
	}

	// Log successful delivery
	slog.LogAttrs(si.ctx, "secret_delivered",
		slog.String("container", filepath.Base(session.socketPath)),
		slog.Int("bytes_sent", totalWritten+4),
		slog.String("provider", string(session.credential.Provider)),
	)
}

// cleanupSession removes socket file and cleans up session
func (si *SecretInjector) cleanupSession(session *secretSession) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return
	}
	session.closed = true

	// Close listener
	session.server.Close()

	// Remove socket file
	os.Remove(session.socketPath)

	// Remove from tracking
	si.mu.Lock()
	delete(si.sockets, filepath.Base(session.socketPath))
	si.mu.Unlock()
}

// Cleanup removes a secret socket after container has started
// This should be called after container startup to remove the socket
func (si *SecretInjector) Cleanup(containerName string) error {
	si.mu.Lock()
	session, exists := si.sockets[containerName]
	si.mu.Unlock()

	if !exists {
		return fmt.Errorf("no secret session found for container: %s", containerName)
	}

	// Close and cleanup session
	si.cleanupSession(session)

	// Log cleanup
	si.securityLog.LogSecretCleanup(si.ctx, containerName, "socket_injection_complete")

	return nil
}

// Stop stops the secret injector and cleans up all sessions
func (si *SecretInjector) Stop() {
	si.cancel()

	// Close all active sessions
	si.mu.Lock()
	for _, session := range si.sockets {
		go si.cleanupSession(session)
	}
	si.mu.Unlock()

	si.wg.Wait()
}
