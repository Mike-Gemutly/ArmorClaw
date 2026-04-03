// Package sidecar provides gRPC client for Rust sidecar communication
package sidecar

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// DefaultSocketPath is the default Unix domain socket path for the sidecar
	DefaultSocketPath = "/run/armorclaw/sidecar.sock"

	// DefaultTimeout is the default timeout for operations
	DefaultTimeout = 30 * time.Second

	// DefaultMaxRetries is the maximum number of retry attempts
	DefaultMaxRetries = 5

	// DefaultMaxRecvMsgSize is the maximum receive message size (256MB)
	DefaultMaxRecvMsgSize = 256 * 1024 * 1024

	// DefaultMaxSendMsgSize is the maximum send message size (256MB)
	DefaultMaxSendMsgSize = 256 * 1024 * 1024
)

// Config holds configuration for the sidecar client
type Config struct {
	SocketPath  string        // Unix domain socket path
	Timeout     time.Duration // Default operation timeout
	MaxRetries  int           // Maximum retry attempts
	DialTimeout time.Duration // Connection dial timeout
	IdleTimeout time.Duration // Connection idle timeout
	MaxMsgSize  int           // Maximum message size
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		SocketPath:  DefaultSocketPath,
		Timeout:     DefaultTimeout,
		MaxRetries:  DefaultMaxRetries,
		DialTimeout: 10 * time.Second,
		IdleTimeout: 5 * time.Minute,
		MaxMsgSize:  DefaultMaxRecvMsgSize,
	}
}

// Client is the gRPC client for communicating with the Rust sidecar
type Client struct {
	config     *Config
	conn       *grpc.ClientConn
	client     SidecarServiceClient
	mu         sync.RWMutex
	connClosed bool
	logger     *slog.Logger
}

// NewClient creates a new sidecar client
func NewClient(config *Config) *Client {
	if config == nil {
		config = DefaultConfig()
	}

	return &Client{
		config: config,
		logger: slog.Default(),
	}
}

// NewClientWithLogger creates a new sidecar client with a custom logger
func NewClientWithLogger(config *Config, logger *slog.Logger) *Client {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = slog.Default()
	}

	return &Client{
		config: config,
		logger: logger,
	}
}

// Connect establishes a connection to the sidecar via Unix Domain Socket
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil && !c.connClosed {
		return nil // Already connected
	}

	dialCtx, cancel := context.WithTimeout(ctx, c.config.DialTimeout)
	defer cancel()

	// Create a custom dialer for Unix Domain Socket
	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		d := net.Dialer{}
		return d.DialContext(ctx, "unix", c.config.SocketPath)
	}

	// Configure backoff strategy for retries
	backoffConfig := backoff.Config{
		MaxDelay: 5 * time.Second,
	}

	callOptions := []grpc.CallOption{}
	if c.config.MaxMsgSize > 0 {
		callOptions = append(callOptions,
			grpc.MaxCallRecvMsgSize(c.config.MaxMsgSize),
			grpc.MaxCallSendMsgSize(c.config.MaxMsgSize),
		)
	}

	conn, err := grpc.DialContext(dialCtx,
		"unix://"+c.config.SocketPath,
		grpc.WithContextDialer(dialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(callOptions...),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff:           backoffConfig,
			MinConnectTimeout: 5 * time.Second,
		}),
	)

	if err != nil {
		c.logger.Error("failed to connect to sidecar",
			"socket", c.config.SocketPath,
			"error", err,
		)
		return fmt.Errorf("connect to sidecar: %w", err)
	}

	c.conn = conn
	c.client = NewSidecarServiceClient(conn)
	c.connClosed = false

	c.logger.Info("connected to sidecar", "socket", c.config.SocketPath)

	return nil
}

// Close closes the connection to the sidecar
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil || c.connClosed {
		return nil // Already closed
	}

	err := c.conn.Close()
	c.connClosed = true

	if err != nil {
		c.logger.Error("error closing sidecar connection", "error", err)
		return fmt.Errorf("close connection: %w", err)
	}

	c.logger.Info("closed sidecar connection")
	return nil
}

// IsConnected returns true if the client is connected to the sidecar
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn != nil && !c.connClosed
}

// ensureConnection ensures the client is connected, establishing a connection if necessary
func (c *Client) ensureConnection(ctx context.Context) error {
	if c.IsConnected() {
		return nil
	}

	return c.Connect(ctx)
}

// withRetry executes a function with retry logic using exponential backoff
func (c *Client) withRetry(ctx context.Context, operation string, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt < c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2^attempt * 100ms
			backoffTime := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
			if backoffTime > 5*time.Second {
				backoffTime = 5 * time.Second
			}

			c.logger.Debug("retrying operation",
				"operation", operation,
				"attempt", attempt+1,
				"max_retries", c.config.MaxRetries,
				"backoff", backoffTime,
			)

			select {
			case <-time.After(backoffTime):
				// Continue with retry
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Ensure connection before each attempt
		if err := c.ensureConnection(ctx); err != nil {
			lastErr = err
			c.logger.Debug("connection failed before operation",
				"operation", operation,
				"attempt", attempt+1,
				"error", err,
			)
			continue
		}

		// Execute the operation
		err := fn(ctx)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if error is context cancellation
		if ctx.Err() != nil {
			return ctx.Err()
		}

		c.logger.Debug("operation failed",
			"operation", operation,
			"attempt", attempt+1,
			"error", err,
		)

		// For non-context errors, continue retrying
	}

	return fmt.Errorf("operation '%s' failed after %d attempts: %w", operation, c.config.MaxRetries, lastErr)
}

// withTimeout applies a timeout to a context if not already set
func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		// Context already has a deadline, don't override
		return ctx, func() {}
	}

	return context.WithTimeout(ctx, c.config.Timeout)
}

// HealthCheck checks the health status of the sidecar
func (c *Client) HealthCheck(ctx context.Context) (*HealthCheckResponse, error) {
	var result *HealthCheckResponse

	err := c.withRetry(ctx, "HealthCheck", func(ctx context.Context) error {
		timeoutCtx, cancel := c.withTimeout(ctx)
		defer cancel()

		req := &HealthCheckRequest{}
		resp, err := c.client.HealthCheck(timeoutCtx, req)
		if err != nil {
			return err
		}

		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	return result, nil
}

// UploadBlob uploads a blob to cloud storage
func (c *Client) UploadBlob(ctx context.Context, req *UploadBlobRequest) (*UploadBlobResponse, error) {
	var result *UploadBlobResponse

	err := c.withRetry(ctx, "UploadBlob", func(ctx context.Context) error {
		timeoutCtx, cancel := c.withTimeout(ctx)
		defer cancel()

		resp, err := c.client.UploadBlob(timeoutCtx, req)
		if err != nil {
			return err
		}

		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("upload blob failed: %w", err)
	}

	return result, nil
}

// DownloadBlob downloads a blob from cloud storage (streaming)
func (c *Client) DownloadBlob(ctx context.Context, req *DownloadBlobRequest) ([]byte, error) {
	var chunks [][]byte

	err := c.withRetry(ctx, "DownloadBlob", func(ctx context.Context) error {
		timeoutCtx, cancel := c.withTimeout(ctx)
		defer cancel()

		stream, err := c.client.DownloadBlob(timeoutCtx, req)
		if err != nil {
			return err
		}

		for {
			chunk, err := stream.Recv()
			if err != nil {
				if err.Error() == "EOF" {
					return nil // End of stream
				}
				return err
			}

			chunks = append(chunks, chunk.Data)

			if chunk.IsLast {
				return nil
			}
		}
	})

	if err != nil {
		return nil, fmt.Errorf("download blob failed: %w", err)
	}

	// Combine chunks
	totalSize := 0
	for _, chunk := range chunks {
		totalSize += len(chunk)
	}

	result := make([]byte, 0, totalSize)
	for _, chunk := range chunks {
		result = append(result, chunk...)
	}

	return result, nil
}

// ListBlobs lists blobs in storage
func (c *Client) ListBlobs(ctx context.Context, req *ListBlobsRequest) (*ListBlobsResponse, error) {
	var result *ListBlobsResponse

	err := c.withRetry(ctx, "ListBlobs", func(ctx context.Context) error {
		timeoutCtx, cancel := c.withTimeout(ctx)
		defer cancel()

		resp, err := c.client.ListBlobs(timeoutCtx, req)
		if err != nil {
			return err
		}

		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("list blobs failed: %w", err)
	}

	return result, nil
}

// DeleteBlob deletes a blob from storage
func (c *Client) DeleteBlob(ctx context.Context, req *DeleteBlobRequest) (*DeleteBlobResponse, error) {
	var result *DeleteBlobResponse

	err := c.withRetry(ctx, "DeleteBlob", func(ctx context.Context) error {
		timeoutCtx, cancel := c.withTimeout(ctx)
		defer cancel()

		resp, err := c.client.DeleteBlob(timeoutCtx, req)
		if err != nil {
			return err
		}

		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("delete blob failed: %w", err)
	}

	return result, nil
}

// ExtractText extracts text from a document
func (c *Client) ExtractText(ctx context.Context, req *ExtractTextRequest) (*ExtractTextResponse, error) {
	var result *ExtractTextResponse

	err := c.withRetry(ctx, "ExtractText", func(ctx context.Context) error {
		timeoutCtx, cancel := c.withTimeout(ctx)
		defer cancel()

		resp, err := c.client.ExtractText(timeoutCtx, req)
		if err != nil {
			return err
		}

		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("extract text failed: %w", err)
	}

	return result, nil
}

// ProcessDocument processes a document with a specific operation
func (c *Client) ProcessDocument(ctx context.Context, req *ProcessDocumentRequest) (*ProcessDocumentResponse, error) {
	var result *ProcessDocumentResponse

	err := c.withRetry(ctx, "ProcessDocument", func(ctx context.Context) error {
		timeoutCtx, cancel := c.withTimeout(ctx)
		defer cancel()

		resp, err := c.client.ProcessDocument(timeoutCtx, req)
		if err != nil {
			return err
		}

		result = resp
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("process document failed: %w", err)
	}

	return result, nil
}
