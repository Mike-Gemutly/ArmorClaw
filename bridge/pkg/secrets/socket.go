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

	"github.com/armorclaw/bridge/pkg/audit"
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
	socketDir   string                    // Directory for secret sockets
	sockets     map[string]*secretSession // Active socket sessions
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	securityLog *logger.SecurityLogger
	log         *logger.Logger // Component-scoped operational logger
	auditLogger *audit.CriticalOperationLogger
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
		ctx:         ctx,
		cancel:      cancel,
		securityLog: secLog,
		log:         logger.Global().WithComponent("secrets"),
	}, nil
}

// SetAuditLogger sets the audit logger for critical operation logging
func (si *SecretInjector) SetAuditLogger(logger *audit.CriticalOperationLogger) {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.auditLogger = logger
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

	// Audit logging for secret injection
	if si.auditLogger != nil {
		_ = si.auditLogger.LogSecretInjection(si.ctx, containerName, cred.ID, true)
	}

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
			si.log.Error("secret_socket_accept_error",
				"container", filepath.Base(session.socketPath),
				"error", err.Error(),
			)
		}
		return

	case <-time.After(SecretSocketTimeout):
		si.log.Error("secret_socket_timeout",
			"container", filepath.Base(session.socketPath),
		)
		return

	case <-si.ctx.Done():
		return
	}
}

// deliverSecrets sends credential data over the socket connection
func (si *SecretInjector) deliverSecrets(session *secretSession, conn net.Conn) {
	containerName := filepath.Base(session.socketPath)
	success := false
	defer func() {
		// Audit logging for secret delivery result
		if si.auditLogger != nil {
			_ = si.auditLogger.LogSecretInjection(si.ctx, containerName, session.credential.ID, success)
		}
	}()

	// Prepare secrets JSON (same format as file-based for compatibility)
	secretsJSON := map[string]interface{}{
		"provider":     session.credential.Provider,
		"token":        session.credential.Token,
		"display_name": session.credential.DisplayName,
	}

	secretsData, err := json.Marshal(secretsJSON)
	if err != nil {
		si.log.Error("secret_marshal_failed",
			"container", containerName,
			"error", err.Error(),
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
		si.log.Error("secret_write_length_failed",
			"container", filepath.Base(session.socketPath),
			"error", err.Error(),
		)
		return
	}

	// Write secrets data
	totalWritten := 0
	for totalWritten < len(secretsData) {
		written, err := conn.Write(secretsData[totalWritten:])
		if err != nil {
			si.log.Error("secret_write_data_failed",
				"container", filepath.Base(session.socketPath),
				"written_bytes", totalWritten,
				"error", err.Error(),
			)
			return
		}
		totalWritten += written
	}

	// Mark as successful for audit logging
	success = true

	// Log successful delivery
	si.log.Info("secret_delivered",
		"container", filepath.Base(session.socketPath),
		"bytes_sent", totalWritten+4,
		"provider", string(session.credential.Provider),
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

	// Audit logging for secret cleanup
	if si.auditLogger != nil {
		_ = si.auditLogger.LogSecretCleanup(si.ctx, containerName, true)
	}

	return nil
}

// UpdateSecrets sends updated secrets to a running container (P0-CRIT-3)
// This is used by send_secret RPC method to deliver new credentials to running containers.
func (si *SecretInjector) UpdateSecrets(containerName string, cred keystore.Credential) error {
	si.mu.Lock()
	defer si.mu.Unlock()

	// Check if container exists
	_, exists := si.sockets[containerName]
	if !exists {
		return fmt.Errorf("container not found or not running: %s", containerName)
	}

	// Create a new socket for update delivery
	socketPath := filepath.Join(si.socketDir, containerName+".update."+fmt.Sprint(time.Now().Unix())+".sock")

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to create update socket: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(socketPath, 0660); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Create update session
	updateSession := &secretSession{
		socketPath:  socketPath,
		credential:  cred,
		ready:      make(chan struct{}),
		expiresAt:  time.Now().Add(SecretSocketTimeout),
		server:     listener,
		closed:     false,
	}

	// Add to tracking (use special key to avoid conflict)
	si.sockets[containerName+".update"] = updateSession

	// Start handler in background
	si.wg.Add(1)
	go si.handleSecretUpdate(updateSession)

	// Wait for connection with timeout
	select {
	case <-updateSession.ready:
		// Socket ready for connection
	case <-time.After(SecretSocketTimeout):
		si.cleanupSession(updateSession)
		return fmt.Errorf("container did not connect to update socket: %s", containerName)
	}

	// Get connection and deliver secrets
	conn, err := updateSession.server.Accept()
	if err != nil {
		si.cleanupSession(updateSession)
		return err
	}
	defer conn.Close()

	si.deliverSecrets(updateSession, conn)

	// Log the update
	si.securityLog.LogSecretInject(si.ctx, containerName, cred.ID,
		slog.String("method", "send_secret"),
		slog.String("reason", "credential_update"),
	)

	// Audit logging for secret update
	if si.auditLogger != nil {
		_ = si.auditLogger.LogSecretInjection(si.ctx, containerName, cred.ID, true)
	}

	// Cleanup update session immediately after delivery
	si.cleanupSession(updateSession)

	return nil
}

// handleSecretUpdate handles update socket connection and delivery
func (si *SecretInjector) handleSecretUpdate(session *secretSession) {
	defer si.cleanupSession(session)

	// Accept connection or timeout
	connChan := make(chan net.Conn, 1)
	errChan := make(chan error, 1)

	go func() {
		conn, err := session.server.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				errChan <- err
			}
			return
		}
		connChan <- conn
	}()

	select {
	case conn := <-connChan:
		// Connection established
		si.deliverSecrets(session, conn)
	case err := <-errChan:
		if !errors.Is(err, net.ErrClosed) {
			si.log.Error("update_socket_error",
				"error", err.Error(),
			)
		}
	case <-time.After(SecretSocketTimeout):
		// Timeout
		si.log.Error("update_timeout",
			"container", filepath.Base(session.socketPath),
		)
	}
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
