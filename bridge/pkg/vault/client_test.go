package vault

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/armorclaw/bridge/pkg/vault/proto"
)

type mockGovernanceServer struct {
	pb.UnimplementedGovernanceServer
	mu           sync.Mutex
	lastIssueReq *pb.IssueTokenRequest
	lastConsume  *pb.ConsumeTokenRequest
	lastZeroize  *pb.ZeroizeRequest
	failCode     codes.Code
}

func (m *mockGovernanceServer) IssueEphemeralToken(ctx context.Context, req *pb.IssueTokenRequest) (*pb.IssueTokenResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastIssueReq = req
	return &pb.IssueTokenResponse{Success: true}, nil
}

func (m *mockGovernanceServer) ConsumeEphemeralToken(ctx context.Context, req *pb.ConsumeTokenRequest) (*pb.ConsumeTokenResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastConsume = req

	if m.failCode != codes.OK {
		return nil, status.Error(m.failCode, "mock error")
	}

	return &pb.ConsumeTokenResponse{Plaintext: "secret-value-123"}, nil
}

func (m *mockGovernanceServer) ZeroizeToolSecrets(ctx context.Context, req *pb.ZeroizeRequest) (*pb.ZeroizeResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastZeroize = req
	return &pb.ZeroizeResponse{SecretsDestroyed: 3}, nil
}

func setupMockServer(t *testing.T) (*grpc.Server, *mockGovernanceServer, string) {
	t.Helper()

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "vault-test.sock")

	if err := os.RemoveAll(socketPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove socket: %v", err)
	}

	mock := &mockGovernanceServer{}
	server := grpc.NewServer()
	pb.RegisterGovernanceServer(server, mock)

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
	})

	return server, mock, socketPath
}

func TestNewGovernanceClient(t *testing.T) {
	_, _, socketPath := setupMockServer(t)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient failed: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestIssueBlindFillToken(t *testing.T) {
	_, mock, socketPath := setupMockServer(t)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tokenID, err := client.IssueBlindFillToken(ctx, "sess-1", "browser", "api-key-secret", 30*time.Minute)
	if err != nil {
		t.Fatalf("IssueBlindFillToken failed: %v", err)
	}

	if tokenID == "" {
		t.Fatal("expected non-empty token_id")
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if mock.lastIssueReq == nil {
		t.Fatal("expected IssueEphemeralToken to be called on server")
	}
	if mock.lastIssueReq.SessionId != "sess-1" {
		t.Errorf("expected session_id 'sess-1', got %q", mock.lastIssueReq.SessionId)
	}
	if mock.lastIssueReq.ToolName != "browser" {
		t.Errorf("expected tool_name 'browser', got %q", mock.lastIssueReq.ToolName)
	}
	if mock.lastIssueReq.Plaintext != "api-key-secret" {
		t.Errorf("expected plaintext 'api-key-secret', got %q", mock.lastIssueReq.Plaintext)
	}
	expectedTTL := (30 * time.Minute).Milliseconds()
	if mock.lastIssueReq.TtlMs != expectedTTL {
		t.Errorf("expected ttl_ms %d, got %d", expectedTTL, mock.lastIssueReq.TtlMs)
	}
	if mock.lastIssueReq.TokenId != tokenID {
		t.Errorf("expected token_id to match, server got %q, client returned %q", mock.lastIssueReq.TokenId, tokenID)
	}
}

func TestConsumeTokenForSidecar(t *testing.T) {
	_, _, socketPath := setupMockServer(t)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	plaintext, err := client.ConsumeTokenForSidecar(ctx, "tok-1", "sess-1", "browser")
	if err != nil {
		t.Fatalf("ConsumeTokenForSidecar failed: %v", err)
	}

	if plaintext != "secret-value-123" {
		t.Errorf("expected plaintext 'secret-value-123', got %q", plaintext)
	}
}

func TestConsumeTokenForSidecar_NotFound(t *testing.T) {
	mock := &mockGovernanceServer{failCode: codes.NotFound}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "vault-test.sock")

	server := grpc.NewServer()
	pb.RegisterGovernanceServer(server, mock)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	go server.Serve(listener)
	defer server.Stop()

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.ConsumeTokenForSidecar(ctx, "tok-1", "sess-1", "browser")
	if err == nil {
		t.Fatal("expected error for NOT_FOUND, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Errorf("expected NOT_FOUND, got %v", st.Code())
	}
}

func TestConsumeTokenForSidecar_PermissionDenied(t *testing.T) {
	mock := &mockGovernanceServer{failCode: codes.PermissionDenied}

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "vault-test.sock")

	server := grpc.NewServer()
	pb.RegisterGovernanceServer(server, mock)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to create listener: %v", err)
	}

	go server.Serve(listener)
	defer server.Stop()

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = client.ConsumeTokenForSidecar(ctx, "tok-1", "sess-1", "browser")
	if err == nil {
		t.Fatal("expected error for PERMISSION_DENIED, got nil")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got: %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Errorf("expected PERMISSION_DENIED, got %v", st.Code())
	}
}

func TestZeroizeToolSecrets(t *testing.T) {
	_, mock, socketPath := setupMockServer(t)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient failed: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	destroyed, err := client.ZeroizeToolSecrets(ctx, "browser", "sess-1")
	if err != nil {
		t.Fatalf("ZeroizeToolSecrets failed: %v", err)
	}

	if destroyed != 3 {
		t.Errorf("expected 3 secrets destroyed, got %d", destroyed)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if mock.lastZeroize == nil {
		t.Fatal("expected ZeroizeToolSecrets to be called on server")
	}
	if mock.lastZeroize.ToolName != "browser" {
		t.Errorf("expected tool_name 'browser', got %q", mock.lastZeroize.ToolName)
	}
	if mock.lastZeroize.SessionId != "sess-1" {
		t.Errorf("expected session_id 'sess-1', got %q", mock.lastZeroize.SessionId)
	}
}
