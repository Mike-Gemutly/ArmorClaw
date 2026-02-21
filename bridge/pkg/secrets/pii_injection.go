// Package secrets provides PII injection for blind fill capability.
// PII is injected into containers via memory-only methods (environment variables
// or Unix sockets), never written to disk.
package secrets

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/audit"
	"github.com/armorclaw/bridge/pkg/logger"
	"github.com/armorclaw/bridge/pkg/pii"
)

const (
	// PIISocketTimeout is how long to wait for container to connect
	PIISocketTimeout = 5 * time.Second

	// PIIEnvPrefix is the prefix for PII environment variables
	PIIEnvPrefix = "PII_"
)

var (
	// ErrPIIInjectionFailed is returned when PII injection fails
	ErrPIIInjectionFailed = errors.New("PII injection failed")

	// ErrPIITimeout is returned when container doesn't connect in time
	ErrPIITimeout = errors.New("container did not connect to PII socket")
)

// PIIInjectionConfig configures how PII is injected into containers
type PIIInjectionConfig struct {
	// Method is the injection method: "socket" or "env"
	Method string `json:"method"`

	// SocketDir is the directory for PII sockets
	SocketDir string `json:"socket_dir"`

	// EnvPrefix is the prefix for environment variable names
	EnvPrefix string `json:"env_prefix"`

	// TTL is how long the injection is valid
	TTL time.Duration `json:"ttl"`
}

// DefaultPIIInjectionConfig returns the default configuration
func DefaultPIIInjectionConfig() *PIIInjectionConfig {
	return &PIIInjectionConfig{
		Method:    "socket",
		SocketDir: "/run/armorclaw/pii",
		EnvPrefix: PIIEnvPrefix,
		TTL:       10 * time.Second,
	}
}

// PIIInjectionResult represents the result of a PII injection
type PIIInjectionResult struct {
	Success       bool              `json:"success"`
	Method        string            `json:"method"`
	ContainerID   string            `json:"container_id"`
	FieldsInjected []string         `json:"fields_injected"`
	SocketPath    string            `json:"socket_path,omitempty"`
	EnvVars       map[string]string `json:"env_vars,omitempty"`
	ExpiresAt     time.Time         `json:"expires_at"`
}

// PIIInjector handles injecting PII into containers
type PIIInjector struct {
	config       *PIIInjectionConfig
	sessions     map[string]*piiSession
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	securityLog  *logger.SecurityLogger
	auditLogger  *audit.CriticalOperationLogger
	log          *logger.Logger
}

// piiSession represents an active PII delivery session
type piiSession struct {
	socketPath string
	resolved   *pii.ResolvedVariables
	ready      chan struct{}
	expiresAt  time.Time
	server     net.Listener
	mu         sync.Mutex
	closed     bool
}

// NewPIIInjector creates a new PII injector
func NewPIIInjector(config *PIIInjectionConfig, securityLog *logger.SecurityLogger) (*PIIInjector, error) {
	if config == nil {
		config = DefaultPIIInjectionConfig()
	}

	// Ensure socket directory exists
	if err := os.MkdirAll(config.SocketDir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create PII socket directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &PIIInjector{
		config:      config,
		sessions:    make(map[string]*piiSession),
		ctx:         ctx,
		cancel:      cancel,
		securityLog: securityLog,
		log:         logger.Global().WithComponent("pii_injector"),
	}, nil
}

// SetAuditLogger sets the audit logger
func (i *PIIInjector) SetAuditLogger(auditLogger *audit.CriticalOperationLogger) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.auditLogger = auditLogger
}

// InjectPII injects resolved PII variables into a container
func (i *PIIInjector) InjectPII(
	ctx context.Context,
	containerName string,
	resolved *pii.ResolvedVariables,
	config *PIIInjectionConfig,
) (*PIIInjectionResult, error) {
	if config == nil {
		config = i.config
	}

	// Validate resolution
	if resolved == nil {
		return nil, errors.New("resolved variables is nil")
	}

	if resolved.IsExpired() {
		return nil, errors.New("resolved variables have expired")
	}

	// Choose injection method
	switch config.Method {
	case "socket":
		return i.injectViaSocket(ctx, containerName, resolved, config)
	case "env":
		return i.injectViaEnv(ctx, containerName, resolved, config)
	default:
		return nil, fmt.Errorf("unknown injection method: %s", config.Method)
	}
}

// injectViaSocket injects PII via Unix domain socket (memory-only)
func (i *PIIInjector) injectViaSocket(
	ctx context.Context,
	containerName string,
	resolved *pii.ResolvedVariables,
	config *PIIInjectionConfig,
) (*PIIInjectionResult, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	// Check for existing session
	if _, exists := i.sessions[containerName]; exists {
		return nil, fmt.Errorf("PII session already exists for container: %s", containerName)
	}

	// Generate socket path
	socketPath := filepath.Join(config.SocketDir, containerName+".pii.sock")

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create PII socket: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(socketPath, 0660); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

	// Create session
	session := &piiSession{
		socketPath: socketPath,
		resolved:   resolved,
		ready:      make(chan struct{}),
		expiresAt:  time.Now().Add(PIISocketTimeout),
		server:     listener,
	}

	i.sessions[containerName] = session

	// Start socket handler
	i.wg.Add(1)
	go i.handlePIIConnection(session, containerName)

	// Wait for socket to be ready
	select {
	case <-session.ready:
		// Socket ready
	case <-time.After(1 * time.Second):
		i.cleanupSession(session, containerName)
		return nil, ErrPIIInjectionFailed
	}

	// Log injection
	i.log.Info("pii_socket_created",
		"container", containerName,
		"socket_path", socketPath,
		"fields", resolved.GrantedFields,
	)

	// Security logging
	if i.securityLog != nil {
		i.securityLog.LogPIIInjected(ctx, containerName, resolved.SkillID, resolved.GrantedFields, "socket")
	}

	// Audit logging
	if i.auditLogger != nil {
		_ = i.auditLogger.LogPIIInjected(ctx, containerName, resolved.SkillID, resolved.GrantedFields, "socket")
	}

	return &PIIInjectionResult{
		Success:        true,
		Method:         "socket",
		ContainerID:    containerName,
		FieldsInjected: resolved.GrantedFields,
		SocketPath:     socketPath,
		ExpiresAt:      session.expiresAt,
	}, nil
}

// injectViaEnv generates environment variables for PII injection
// Note: This should be used with caution as env vars may be visible in process listings
func (i *PIIInjector) injectViaEnv(
	ctx context.Context,
	containerName string,
	resolved *pii.ResolvedVariables,
	config *PIIInjectionConfig,
) (*PIIInjectionResult, error) {
	envVars := make(map[string]string)
	prefix := config.EnvPrefix
	if prefix == "" {
		prefix = PIIEnvPrefix
	}

	// Generate environment variables for each field
	for key, value := range resolved.Variables {
		envKey := prefix + key
		envVars[envKey] = value
	}

	// Add metadata
	envVars[prefix+"_REQUEST_ID"] = resolved.RequestID
	envVars[prefix+"_SKILL_ID"] = resolved.SkillID
	envVars[prefix+"_EXPIRES_AT"] = fmt.Sprintf("%d", resolved.ExpiresAt)

	// Log injection
	i.log.Info("pii_env_prepared",
		"container", containerName,
		"fields", resolved.GrantedFields,
	)

	// Security logging
	if i.securityLog != nil {
		i.securityLog.LogPIIInjected(ctx, containerName, resolved.SkillID, resolved.GrantedFields, "env")
	}

	// Audit logging
	if i.auditLogger != nil {
		_ = i.auditLogger.LogPIIInjected(ctx, containerName, resolved.SkillID, resolved.GrantedFields, "env")
	}

	return &PIIInjectionResult{
		Success:        true,
		Method:         "env",
		ContainerID:    containerName,
		FieldsInjected: resolved.GrantedFields,
		EnvVars:        envVars,
		ExpiresAt:      time.Now().Add(config.TTL),
	}, nil
}

// handlePIIConnection waits for container to connect and delivers PII
func (i *PIIInjector) handlePIIConnection(session *piiSession, containerName string) {
	defer i.wg.Done()
	defer i.cleanupSession(session, containerName)

	// Signal socket is ready
	close(session.ready)

	// Wait for connection
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

	select {
	case conn := <-connChan:
		defer conn.Close()
		i.deliverPII(session, conn, containerName)

	case err := <-errChan:
		if !errors.Is(err, net.ErrClosed) {
			i.log.Error("pii_socket_error",
				"container", containerName,
				"error", err.Error(),
			)
		}

	case <-time.After(PIISocketTimeout):
		i.log.Error("pii_socket_timeout",
			"container", containerName,
		)

	case <-i.ctx.Done():
		return
	}
}

// deliverPII sends PII data over the socket connection
func (i *PIIInjector) deliverPII(session *piiSession, conn net.Conn, containerName string) {
	// Prepare PII JSON
	piiData := map[string]interface{}{
		"request_id": session.resolved.RequestID,
		"skill_id":   session.resolved.SkillID,
		"variables":  session.resolved.Variables,
		"expires_at": session.resolved.ExpiresAt,
	}

	data, err := json.Marshal(piiData)
	if err != nil {
		i.log.Error("pii_marshal_failed",
			"container", containerName,
			"error", err.Error(),
		)
		return
	}

	// Write length prefix
	length := len(data)
	lengthPrefix := []byte{
		byte(length >> 24),
		byte(length >> 16),
		byte(length >> 8),
		byte(length),
	}

	if _, err := conn.Write(lengthPrefix); err != nil {
		i.log.Error("pii_write_length_failed",
			"container", containerName,
			"error", err.Error(),
		)
		return
	}

	// Write data
	totalWritten := 0
	for totalWritten < len(data) {
		written, err := conn.Write(data[totalWritten:])
		if err != nil {
			i.log.Error("pii_write_data_failed",
				"container", containerName,
				"written", totalWritten,
				"error", err.Error(),
			)
			return
		}
		totalWritten += written
	}

	i.log.Info("pii_delivered",
		"container", containerName,
		"bytes_sent", totalWritten+4,
		"fields_count", len(session.resolved.GrantedFields),
	)
}

// cleanupSession removes a PII session
func (i *PIIInjector) cleanupSession(session *piiSession, containerName string) {
	session.mu.Lock()
	defer session.mu.Unlock()

	if session.closed {
		return
	}
	session.closed = true

	// Close listener
	if session.server != nil {
		session.server.Close()
	}

	// Remove socket file
	if session.socketPath != "" {
		os.Remove(session.socketPath)
	}

	// Remove from tracking
	i.mu.Lock()
	delete(i.sessions, containerName)
	i.mu.Unlock()
}

// Cleanup removes a PII injection session
func (i *PIIInjector) Cleanup(containerName string) error {
	i.mu.RLock()
	session, exists := i.sessions[containerName]
	i.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no PII session found for container: %s", containerName)
	}

	i.cleanupSession(session, containerName)
	return nil
}

// Stop stops the PII injector
func (i *PIIInjector) Stop() {
	i.cancel()

	i.mu.Lock()
	for name, session := range i.sessions {
		go i.cleanupSession(session, name)
	}
	i.mu.Unlock()

	i.wg.Wait()
}

// PreparePIIEnvironment creates environment variables from resolved PII
// This is a helper function for container creation
func PreparePIIEnvironment(resolved *pii.ResolvedVariables, prefix string) []string {
	if prefix == "" {
		prefix = PIIEnvPrefix
	}

	var envVars []string

	// Add PII variables
	for key, value := range resolved.Variables {
		envVars = append(envVars, fmt.Sprintf("%s%s=%s", prefix, key, value))
	}

	// Add metadata
	envVars = append(envVars, fmt.Sprintf("%s_REQUEST_ID=%s", prefix, resolved.RequestID))
	envVars = append(envVars, fmt.Sprintf("%s_SKILL_ID=%s", prefix, resolved.SkillID))
	envVars = append(envVars, fmt.Sprintf("%s_PROFILE_ID=%s", prefix, resolved.ProfileID))

	return envVars
}
