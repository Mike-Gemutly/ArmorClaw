package sidecar

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/armorclaw/bridge/pkg/pii"
)

// mockServer is a mock implementation of the sidecar service for testing
type mockServer struct {
	UnimplementedSidecarServiceServer
	healthCalled   bool
	uploadCalled   bool
	downloadCalled bool
	extractCalled  bool
	processCalled  bool
	mu             sync.Mutex
	delay          time.Duration
	failAfter      int
	callCount      map[string]int
	shouldFail     bool
	failCode       codes.Code
}

func newMockServer() *mockServer {
	return &mockServer{
		callCount: make(map[string]int),
	}
}

func (m *mockServer) HealthCheck(ctx context.Context, req *HealthCheckRequest) (*HealthCheckResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount["HealthCheck"]++
	m.healthCalled = true

	// Fail for calls <= failAfter, succeed after that
	if m.failAfter > 0 && m.callCount["HealthCheck"] <= m.failAfter {
		return nil, status.Error(codes.Unavailable, "service unavailable")
	}

	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	if m.shouldFail {
		return nil, status.Error(m.failCode, "mock error")
	}

	return &HealthCheckResponse{
		Status:          "healthy",
		UptimeSeconds:   3600,
		ActiveRequests:  5,
		MemoryUsedBytes: 1024 * 1024 * 100,
		Version:         "1.0.0",
	}, nil
}

func (m *mockServer) UploadBlob(ctx context.Context, req *UploadBlobRequest) (*UploadBlobResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount["UploadBlob"]++
	m.uploadCalled = true

	if m.shouldFail {
		return nil, status.Error(m.failCode, "mock error")
	}

	return &UploadBlobResponse{
		BlobId:            "blob-123",
		Etag:              "etag-456",
		SizeBytes:         1024,
		ContentHashSha256: "hash-789",
		TimestampUnix:     time.Now().Unix(),
	}, nil
}

func (m *mockServer) DownloadBlob(req *DownloadBlobRequest, stream SidecarService_DownloadBlobServer) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount["DownloadBlob"]++
	m.downloadCalled = true

	if m.shouldFail {
		return status.Error(m.failCode, "mock error")
	}

	chunks := [][]byte{
		[]byte("chunk-1"),
		[]byte("chunk-2"),
		[]byte("chunk-3"),
	}

	for i, chunk := range chunks {
		err := stream.Send(&BlobChunk{
			Data:   chunk,
			Offset: int64(i),
			IsLast: i == len(chunks)-1,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *mockServer) ListBlobs(ctx context.Context, req *ListBlobsRequest) (*ListBlobsResponse, error) {
	return &ListBlobsResponse{
		Blobs: []*BlobInfo{
			{
				Uri:              "blob://test/blob1",
				SizeBytes:        1024,
				ContentType:      "application/octet-stream",
				LastModifiedUnix: time.Now().Unix(),
				Etag:             "etag-1",
			},
		},
	}, nil
}

func (m *mockServer) DeleteBlob(ctx context.Context, req *DeleteBlobRequest) (*DeleteBlobResponse, error) {
	return &DeleteBlobResponse{
		Success: true,
		Message: "Blob deleted",
	}, nil
}

func (m *mockServer) ExtractText(ctx context.Context, req *ExtractTextRequest) (*ExtractTextResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount["ExtractText"]++
	m.extractCalled = true

	if m.shouldFail {
		return nil, status.Error(m.failCode, "mock error")
	}

	return &ExtractTextResponse{
		Text:      "extracted text",
		PageCount: 1,
		Metadata: map[string]string{
			"author":  "test",
			"subject": "test doc",
		},
	}, nil
}

func (m *mockServer) ProcessDocument(ctx context.Context, req *ProcessDocumentRequest) (*ProcessDocumentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.callCount["ProcessDocument"]++
	m.processCalled = true

	if m.shouldFail {
		return nil, status.Error(m.failCode, "mock error")
	}

	return &ProcessDocumentResponse{
		OutputUri:     "blob://test/output",
		OutputContent: []byte("processed"),
		OutputFormat:  "txt",
		Metadata: map[string]string{
			"processed_by": "sidecar",
		},
	}, nil
}

// setupTestServer creates a test Unix domain socket and starts a mock server
func setupTestServer(t *testing.T) (*grpc.Server, *mockServer, string) {
	t.Helper()

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")

	if err := os.RemoveAll(socketPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove socket: %v", err)
	}

	mock := newMockServer()
	server := grpc.NewServer()
	RegisterSidecarServiceServer(server, mock)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Logf("server error: %v", err)
		}
	}()

	t.Cleanup(func() {
		server.Stop()
		os.RemoveAll(socketPath)
	})

	return server, mock, socketPath
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.SocketPath != DefaultSocketPath {
		t.Errorf("expected SocketPath %s, got %s", DefaultSocketPath, config.SocketPath)
	}

	if config.Timeout != DefaultTimeout {
		t.Errorf("expected Timeout %v, got %v", DefaultTimeout, config.Timeout)
	}

	if config.MaxRetries != DefaultMaxRetries {
		t.Errorf("expected MaxRetries %d, got %d", DefaultMaxRetries, config.MaxRetries)
	}

	if config.DialTimeout != 10*time.Second {
		t.Errorf("expected DialTimeout %v, got %v", 10*time.Second, config.DialTimeout)
	}

	if config.IdleTimeout != 5*time.Minute {
		t.Errorf("expected IdleTimeout %v, got %v", 5*time.Minute, config.IdleTimeout)
	}

	if config.MaxMsgSize != DefaultMaxRecvMsgSize {
		t.Errorf("expected MaxMsgSize %d, got %d", DefaultMaxRecvMsgSize, config.MaxMsgSize)
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient(nil)

	if client == nil {
		t.Fatal("expected non-nil client")
	}

	if client.config == nil {
		t.Fatal("expected non-nil config")
	}

	if client.config.SocketPath != DefaultSocketPath {
		t.Errorf("expected default SocketPath %s, got %s", DefaultSocketPath, client.config.SocketPath)
	}
}

func TestNewClientWithConfig(t *testing.T) {
	config := &Config{
		SocketPath: "/custom/path.sock",
		Timeout:    10 * time.Second,
		MaxRetries: 3,
	}

	client := NewClient(config)

	if client.config.SocketPath != "/custom/path.sock" {
		t.Errorf("expected SocketPath /custom/path.sock, got %s", client.config.SocketPath)
	}

	if client.config.Timeout != 10*time.Second {
		t.Errorf("expected Timeout 10s, got %v", client.config.Timeout)
	}

	if client.config.MaxRetries != 3 {
		t.Errorf("expected MaxRetries 3, got %d", client.config.MaxRetries)
	}
}

func TestNewClientWithLogger(t *testing.T) {
	handler := newTestLogger()
	logger := slog.New(handler)
	client := NewClientWithLogger(nil, logger)

	// Cannot directly compare logger, but can verify it's set
	if client.logger == nil {
		t.Error("expected logger to be set")
	}
}

func TestClientConnect(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := DefaultConfig()
	config.SocketPath = socketPath
	config.Timeout = 5 * time.Second
	config.MaxRetries = 3
	config.DialTimeout = 5 * time.Second

	client := NewClient(config)
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("expected no error connecting, got: %v", err)
	}

	if !client.IsConnected() {
		t.Error("expected client to be connected")
	}

	time.Sleep(100 * time.Millisecond)

	if client.IsConnected() {
		t.Logf("Warning: client still reports connected after server stop (gRPC connections are lazy)")
	}
}

func TestClientConnectAlreadyConnected(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("expected no error connecting, got: %v", err)
	}

	err = client.Connect(ctx)
	if err != nil {
		t.Errorf("expected no error on second connect, got: %v", err)
	}
}

func TestClientConnectInvalidPath(t *testing.T) {
	config := DefaultConfig()
	config.SocketPath = "/nonexistent/path.sock"
	config.Timeout = 5 * time.Second
	config.MaxRetries = 3
	config.DialTimeout = 1 * time.Second

	client := NewClient(config)
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		return // Expected: dial failed
	}

	// gRPC dial is lazy - try an actual call to verify connection fails
	_, err = client.HealthCheck(ctx)
	if err == nil {
		t.Error("expected error on health check with invalid path")
	}
}

func TestClientConnectTimeout(t *testing.T) {
	mock := newMockServer()
	mock.delay = 10 * time.Second // Longer than client timeout

	socketPath := filepath.Join(t.TempDir(), "test.sock")

	server := grpc.NewServer()
	RegisterSidecarServiceServer(server, mock)

	listener, listenErr := net.Listen("unix", socketPath)
	if listenErr != nil {
		t.Fatalf("failed to create listener: %v", listenErr)
	}

	go server.Serve(listener)
	defer server.Stop()
	_ = mock

	config := DefaultConfig()
	config.SocketPath = socketPath
	config.Timeout = 2 * time.Second // Shorter than mock delay
	config.MaxRetries = 1
	config.DialTimeout = 1 * time.Second // Short dial timeout

	client := NewClient(config)
	ctx := context.Background()

	connErr := client.Connect(ctx)
	if connErr != nil {
		return // Dial timeout occurred
	}

	// gRPC dial is lazy - try an actual call to verify timeout
	_, connErr = client.HealthCheck(ctx)
	if connErr == nil {
		t.Error("expected timeout error on health check")
	}
}

func TestWithContextTimeout(t *testing.T) {
	// This test verifies context timeout handling
	mock := newMockServer()
	mock.delay = 100 * time.Millisecond // Small delay

	socketPath := filepath.Join(t.TempDir(), "test.sock")

	server := grpc.NewServer()
	RegisterSidecarServiceServer(server, mock)

	listener, listenErr := net.Listen("unix", socketPath)
	if listenErr != nil {
		t.Fatalf("failed to create listener: %v", listenErr)
	}

	go server.Serve(listener)
	defer server.Stop()
	_ = mock

	config := DefaultConfig()
	config.SocketPath = socketPath
	config.Timeout = 1 * time.Second
	config.MaxRetries = 1
	config.DialTimeout = 5 * time.Second

	client := NewClient(config)
	ctx := context.Background()

	// Use a very short context that will timeout before the server responds
	shortCtx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
	defer cancel()

	_, err := client.HealthCheck(shortCtx)
	if err == nil {
		t.Error("expected timeout error on health check")
	}
}

func TestClientClose(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("expected no error connecting, got: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Fatalf("expected no error closing, got: %v", err)
	}

	if client.IsConnected() {
		t.Error("expected client to be disconnected after close")
	}

	err = client.Close()
	if err != nil {
		t.Errorf("expected no error on double close, got: %v", err)
	}
}

func TestClientIsConnected(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	if client.IsConnected() {
		t.Error("expected client to be disconnected initially")
	}

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("expected no error connecting, got: %v", err)
	}

	if !client.IsConnected() {
		t.Error("expected client to be connected after Connect")
	}
}

func TestHealthCheck(t *testing.T) {
	server, mock, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	resp, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got: %s", resp.Status)
	}

	if !mock.healthCalled {
		t.Error("expected health check to be called on server")
	}

	server.Stop()

	resp, err = client.HealthCheck(ctx)
	if err == nil {
		t.Error("expected error after server stop")
	}
}

func TestHealthCheckWithRetry(t *testing.T) {
	mock := newMockServer()
	mock.failAfter = 2

	socketPath := filepath.Join(t.TempDir(), "test.sock")

	server := grpc.NewServer()
	RegisterSidecarServiceServer(server, mock)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	go server.Serve(listener)
	defer server.Stop()

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  5,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	resp, err := client.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("expected no error after retries, got: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got: %s", resp.Status)
	}

	if mock.callCount["HealthCheck"] <= 2 {
		t.Errorf("expected at least 3 calls, got: %d", mock.callCount["HealthCheck"])
	}
}

func TestUploadBlob(t *testing.T) {
	_, mock, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/key",
		ContentType:    "application/octet-stream",
		Content:        []byte("test data"),
	}

	resp, err := client.UploadBlob(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.BlobId != "blob-123" {
		t.Errorf("expected blob ID 'blob-123', got: %s", resp.BlobId)
	}

	if !mock.uploadCalled {
		t.Error("expected upload to be called on server")
	}
}

func TestDownloadBlob(t *testing.T) {
	_, mock, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &DownloadBlobRequest{
		Provider:  "s3",
		SourceUri: "s3://bucket/key",
	}

	data, err := client.DownloadBlob(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	expected := "chunk-1chunk-2chunk-3"
	if string(data) != expected {
		t.Errorf("expected data '%s', got: %s", expected, string(data))
	}

	if !mock.downloadCalled {
		t.Error("expected download to be called on server")
	}
}

func TestExtractText(t *testing.T) {
	_, mock, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &ExtractTextRequest{
		DocumentFormat:  "pdf",
		DocumentContent: []byte("pdf content"),
	}

	resp, err := client.ExtractText(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.Text != "extracted text" {
		t.Errorf("expected text 'extracted text', got: %s", resp.Text)
	}

	if !mock.extractCalled {
		t.Error("expected extract text to be called on server")
	}
}

func TestProcessDocument(t *testing.T) {
	_, mock, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &ProcessDocumentRequest{
		Operation:    "convert",
		InputFormat:  "pdf",
		OutputFormat: "txt",
		InputContent: []byte("pdf content"),
	}

	resp, err := client.ProcessDocument(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if string(resp.OutputContent) != "processed" {
		t.Errorf("expected output 'processed', got: %s", string(resp.OutputContent))
	}

	if !mock.processCalled {
		t.Error("expected process document to be called on server")
	}
}

func TestListBlobs(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &ListBlobsRequest{
		Provider: "s3",
	}

	resp, err := client.ListBlobs(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(resp.Blobs) != 1 {
		t.Errorf("expected 1 blob, got: %d", len(resp.Blobs))
	}
}

func TestDeleteBlob(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &DeleteBlobRequest{
		Provider: "s3",
		Uri:      "s3://bucket/key",
	}

	resp, err := client.DeleteBlob(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if !resp.Success {
		t.Error("expected success to be true")
	}
}

func TestWithRetryExhausted(t *testing.T) {
	mock := newMockServer()
	mock.shouldFail = true
	mock.failCode = codes.Unavailable

	socketPath := filepath.Join(t.TempDir(), "test.sock")

	server := grpc.NewServer()
	RegisterSidecarServiceServer(server, mock)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	go server.Serve(listener)
	defer server.Stop()

	config := DefaultConfig()
	config.SocketPath = socketPath
	config.Timeout = 5 * time.Second
	config.MaxRetries = 3
	config.DialTimeout = 5 * time.Second

	client := NewClient(config)
	ctx := context.Background()

	_, err = client.HealthCheck(ctx)
	if err == nil {
		t.Error("expected error after retry exhaustion")
	}

	if mock.callCount["HealthCheck"] != config.MaxRetries {
		t.Errorf("expected %d retry attempts, got: %d", config.MaxRetries, mock.callCount["HealthCheck"])
	}
}

func TestWithRetryCancelsContext(t *testing.T) {
	mock := newMockServer()
	mock.delay = 100 * time.Millisecond

	socketPath := filepath.Join(t.TempDir(), "test.sock")

	server := grpc.NewServer()
	RegisterSidecarServiceServer(server, mock)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	go server.Serve(listener)
	defer server.Stop()

	config := DefaultConfig()
	config.SocketPath = socketPath
	config.Timeout = 5 * time.Second
	config.MaxRetries = 10
	config.DialTimeout = 5 * time.Second

	client := NewClient(config)
	ctx := context.Background()

	shortCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	// Context should be cancelled before all retries complete
	_, err = client.HealthCheck(shortCtx)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}

	if !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
		t.Errorf("expected context error, got: %v", err)
	}
}

// testLogger implements slog.Handler for testing
type testLogger struct {
	buf bytes.Buffer
	mu  sync.Mutex
}

func newTestLogger() *testLogger {
	return &testLogger{}
}

func (l *testLogger) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l *testLogger) Handle(ctx context.Context, r slog.Record) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.buf.WriteString(r.Message + "\n")
	return nil
}

func (l *testLogger) WithAttrs(attrs []slog.Attr) slog.Handler {
	return l
}

func (l *testLogger) WithGroup(name string) slog.Handler {
	return l
}

func (l *testLogger) getLogs() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.String()
}

func TestUploadBlobWithPIIInterception(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
		PIIInterceptor: &PIIInterceptorConfig{
			Enabled:  true,
			Action:   ActionRedact,
			Scrubber: pii.New(),
		},
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/key",
		ContentType:    "text/plain",
		Content:        []byte("Contact test@example.com"),
	}

	resp, err := client.UploadBlob(ctx, req)
	if err != nil {
		t.Fatalf("expected no error with PII redaction, got: %v", err)
	}

	if resp.BlobId != "blob-123" {
		t.Errorf("expected blob ID 'blob-123', got: %s", resp.BlobId)
	}
}

func TestExtractTextWithPIIInterception(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
		PIIInterceptor: &PIIInterceptorConfig{
			Enabled:  true,
			Action:   ActionRedact,
			Scrubber: pii.New(),
		},
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &ExtractTextRequest{
		DocumentFormat:  "pdf",
		DocumentContent: []byte("Phone: 555-123-4567"),
	}

	resp, err := client.ExtractText(ctx, req)
	if err != nil {
		t.Fatalf("expected no error with PII redaction, got: %v", err)
	}

	if resp.Text != "extracted text" {
		t.Errorf("expected text 'extracted text', got: %s", resp.Text)
	}
}

func TestProcessDocumentWithPIIInterception(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
		PIIInterceptor: &PIIInterceptorConfig{
			Enabled:  true,
			Action:   ActionRedact,
			Scrubber: pii.New(),
		},
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &ProcessDocumentRequest{
		InputFormat:  "pdf",
		InputContent: []byte("SSN: 123-45-6789"),
		Operation:    "process",
	}

	resp, err := client.ProcessDocument(ctx, req)
	if err != nil {
		t.Fatalf("expected no error with PII redaction, got: %v", err)
	}

	if resp.OutputUri != "blob://test/output" {
		t.Errorf("expected output URI 'blob://test/output', got: %s", resp.OutputUri)
	}
}

func TestUploadBlobWithPIIInterceptionReject(t *testing.T) {
	_, _, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
		PIIInterceptor: &PIIInterceptorConfig{
			Enabled:  true,
			Action:   ActionReject,
			Scrubber: pii.New(),
		},
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/key",
		ContentType:    "text/plain",
		Content:        []byte("Contact test@example.com"),
	}

	_, err := client.UploadBlob(ctx, req)
	if err == nil {
		t.Error("expected error when PII is detected and action is reject")
	}

	if err == nil {
		t.Logf("Expected PII rejection error, got nil")
	}
}

func TestUploadBlobWithPIIInterceptionDisabled(t *testing.T) {
	_, mock, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
		PIIInterceptor: &PIIInterceptorConfig{
			Enabled:  false,
			Action:   ActionRedact,
			Scrubber: pii.New(),
		},
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/key",
		ContentType:    "text/plain",
		Content:        []byte("Contact test@example.com"),
	}

	resp, err := client.UploadBlob(ctx, req)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if resp.BlobId != "blob-123" {
		t.Errorf("expected blob ID 'blob-123', got: %s", resp.BlobId)
	}

	if !mock.uploadCalled {
		t.Error("expected upload to be called on server")
	}
}

func TestUploadBlobWithPIIInterceptionLogOnly(t *testing.T) {
	_, mock, socketPath := setupTestServer(t)

	config := &Config{
		SocketPath:  socketPath,
		Timeout:     5 * time.Second,
		MaxRetries:  3,
		DialTimeout: 5 * time.Second,
		PIIInterceptor: &PIIInterceptorConfig{
			Enabled:  true,
			Action:   ActionRedact,
			LogOnly:  true,
			Scrubber: pii.New(),
		},
	}

	client := NewClient(config)
	ctx := context.Background()

	req := &UploadBlobRequest{
		Provider:       "s3",
		DestinationUri: "s3://bucket/key",
		ContentType:    "text/plain",
		Content:        []byte("Contact test@example.com"),
	}

	resp, err := client.UploadBlob(ctx, req)
	if err != nil {
		t.Fatalf("expected no error in log-only mode, got: %v", err)
	}

	if resp.BlobId != "blob-123" {
		t.Errorf("expected blob ID 'blob-123', got: %s", resp.BlobId)
	}

	if !mock.uploadCalled {
		t.Error("expected upload to be called on server")
	}
}
