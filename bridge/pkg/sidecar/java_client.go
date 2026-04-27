package sidecar

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

const (
	// JavaSocketPath is the Unix domain socket path for the Java sidecar
	JavaSocketPath = "/run/armorclaw/sidecar-java/sidecar-java.sock"

	// JavaMaxMsgSize is the maximum message size for Java sidecar communication (100 MB)
	JavaMaxMsgSize = 100 * 1024 * 1024
)

// NewJavaClient creates a gRPC client pointing to the Java sidecar socket.
func NewJavaClient(piiConfig *PIIInterceptorConfig) *Client {
	config := &Config{
		SocketPath:     JavaSocketPath,
		Timeout:        DefaultTimeout,
		MaxRetries:     DefaultMaxRetries,
		DialTimeout:    10 * time.Second,
		IdleTimeout:    5 * time.Minute,
		MaxMsgSize:     JavaMaxMsgSize,
		PIIInterceptor: piiConfig,
	}
	return NewClient(config)
}

// ProvisionJavaSocketDir creates the socket directory for the Java sidecar
// with permissions allowing UID 10001 to write.
func ProvisionJavaSocketDir() error {
	dir := "/run/armorclaw/sidecar-java"
	if err := os.MkdirAll(dir, 0770); err != nil {
		return fmt.Errorf("failed to create java socket dir: %w", err)
	}
	if err := os.Chown(dir, 10001, 10001); err != nil {
		// Chown may fail in non-root environments (dev), log but don't fail
		slog.Warn("failed to chown java socket dir (non-root?)", "error", err)
	}
	return nil
}
