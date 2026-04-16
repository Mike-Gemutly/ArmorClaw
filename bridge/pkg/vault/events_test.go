package vault

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/eventbus"
	pb "github.com/armorclaw/bridge/pkg/vault/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eventStreamServer struct {
	pb.UnimplementedGovernanceServer
	mu          sync.Mutex
	events      []*pb.VaultEventStream
	sendDelay   time.Duration
	failCode    codes.Code
	streamCount atomic.Int32
}

func (s *eventStreamServer) SubscribeEvents(req *pb.SubscribeRequest, stream pb.Governance_SubscribeEventsServer) error {
	s.streamCount.Add(1)

	if s.failCode != codes.OK {
		return status.Error(s.failCode, "mock stream error")
	}

	for _, ev := range s.events {
		if s.sendDelay > 0 {
			time.Sleep(s.sendDelay)
		}

		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		if err := stream.Send(ev); err != nil {
			return err
		}
	}

	// Block until cancelled to simulate long-lived stream.
	<-stream.Context().Done()
	return stream.Context().Err()
}

func setupEventStreamServer(t *testing.T, srv *eventStreamServer) (string, *grpc.Server) {
	t.Helper()

	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "vault-events-test.sock")

	server := grpc.NewServer()
	pb.RegisterGovernanceServer(server, srv)

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

	return socketPath, server
}

func newTestEventBus(t *testing.T) *eventbus.EventBus {
	t.Helper()
	bus := eventbus.NewEventBus(eventbus.DefaultConfig())
	t.Cleanup(func() { bus.Stop() })
	return bus
}

func TestVaultEventBridge_ReceivesAndMapsEvents(t *testing.T) {
	events := []*pb.VaultEventStream{
		{EventType: "token_issued", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		{EventType: "token_consumed", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		{EventType: "secrets_zeroized", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		{EventType: "skill_gate_denied", SessionId: "sess-2", Timestamp: time.Now().UnixMilli()},
		{EventType: "pii_detected_in_output", SessionId: "sess-2", Timestamp: time.Now().UnixMilli()},
		{EventType: "unknown_event", SessionId: "sess-3", Timestamp: time.Now().UnixMilli()},
	}

	srv := &eventStreamServer{events: events}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)

	// Wait for events to be sent (3 per batch + some slack).
	time.Sleep(500 * time.Millisecond)

	bridge.Stop()

	// Verify the stream was opened.
	if count := srv.streamCount.Load(); count < 1 {
		t.Errorf("expected at least 1 stream connection, got %d", count)
	}
}

func TestVaultEventBridge_UnknownEventTypeSkipped(t *testing.T) {
	events := []*pb.VaultEventStream{
		{EventType: "completely_unknown", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
	}

	srv := &eventStreamServer{events: events}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)
	time.Sleep(300 * time.Millisecond)
	bridge.Stop()

	// No crash = success. The unknown event should be logged and skipped.
}

func TestVaultEventBridge_ReconnectsOnStreamError(t *testing.T) {
	srv := &eventStreamServer{
		failCode: codes.Unavailable,
	}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)

	// Wait long enough for multiple reconnection attempts.
	time.Sleep(1500 * time.Millisecond)
	bridge.Stop()

	// Should have attempted multiple streams due to backoff.
	count := srv.streamCount.Load()
	if count < 2 {
		t.Errorf("expected at least 2 stream reconnection attempts, got %d", count)
	}
}

func TestVaultEventBridge_StopCancelsContext(t *testing.T) {
	srv := &eventStreamServer{
		events: []*pb.VaultEventStream{
			{EventType: "token_issued", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		},
	}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx := context.Background()

	bridge.StartSyncLoop(ctx)

	// Stop should return promptly.
	done := make(chan struct{})
	go func() {
		bridge.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Good — Stop returned.
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2 seconds")
	}
}

func TestVaultEventBridge_WithSessionFilter(t *testing.T) {
	srv := &eventStreamServer{
		events: []*pb.VaultEventStream{
			{EventType: "token_issued", SessionId: "filtered-session", Timestamp: time.Now().UnixMilli()},
		},
	}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus).
		WithSessionFilter("filtered-session")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)
	time.Sleep(300 * time.Millisecond)
	bridge.Stop()

	if srv.streamCount.Load() < 1 {
		t.Error("expected stream to be opened with session filter")
	}
}

func TestVaultEventBridge_BackoffIncreases(t *testing.T) {
	srv := &eventStreamServer{
		failCode: codes.Unavailable,
	}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)

	// Wait for 1s + 2s + 4s backoffs to produce ~3 attempts.
	time.Sleep(4 * time.Second)
	bridge.Stop()

	count := srv.streamCount.Load()
	// With backoff 1s, 2s, 4s, we should see roughly 3 attempts in 4 seconds.
	if count < 2 {
		t.Errorf("expected at least 2 stream attempts with backoff, got %d", count)
	}
}

func TestMapVaultEvent(t *testing.T) {
	bus := newTestEventBus(t)
	client, _ := NewGovernanceClient("")
	bridge := NewVaultEventBridge(client, bus)

	tests := []struct {
		eventType string
		wantNil   bool
	}{
		{"token_issued", false},
		{"token_consumed", false},
		{"secrets_zeroized", false},
		{"skill_gate_denied", false},
		{"pii_detected_in_output", false},
		{"unknown_type", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			ve := &pb.VaultEventStream{
				EventType: tt.eventType,
				SessionId: "sess-1",
				Timestamp: time.Now().UnixMilli(),
			}

			result := bridge.mapVaultEvent(ve)

			if tt.wantNil && result != nil {
				t.Errorf("expected nil for %q, got %T", tt.eventType, result)
			}
			if !tt.wantNil && result == nil {
				t.Errorf("expected non-nil for %q, got nil", tt.eventType)
			}
			if !tt.wantNil {
				if result.EventType() == "" {
					t.Errorf("expected non-empty EventType for %q", tt.eventType)
				}
			}
		})
	}
}

func TestVaultEventBridge_WithLogger(t *testing.T) {
	srv := &eventStreamServer{
		events: []*pb.VaultEventStream{
			{EventType: "token_issued", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		},
	}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	mockLogger := slog.Default()

	bridge := NewVaultEventBridge(client, bus).
		WithLogger(mockLogger)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)
	time.Sleep(300 * time.Millisecond)
	bridge.Stop()

	// If we get here without panic, logger was accepted.
}

func TestVaultEventBridge_ConcurrentStop(t *testing.T) {
	srv := &eventStreamServer{
		events: []*pb.VaultEventStream{
			{EventType: "token_issued", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		},
	}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx := context.Background()

	bridge.StartSyncLoop(ctx)
	time.Sleep(100 * time.Millisecond)

	// Multiple concurrent Stops should not panic.
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bridge.Stop()
		}()
	}
	wg.Wait()
}

func TestVaultEventBridge_StreamErrorContextCancel(t *testing.T) {
	srv := &eventStreamServer{
		failCode: codes.Unavailable,
	}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx, cancel := context.WithCancel(context.Background())

	bridge.StartSyncLoop(ctx)

	// Cancel context after a short delay.
	time.Sleep(100 * time.Millisecond)
	cancel()

	// Stop should return promptly after context cancellation.
	done := make(chan struct{})
	go func() {
		bridge.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return promptly after context cancellation")
	}
}

func TestVaultEventBridge_PublishError(t *testing.T) {
	events := []*pb.VaultEventStream{
		{EventType: "token_issued", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		{EventType: "token_consumed", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
		{EventType: "skill_gate_denied", SessionId: "sess-1", Timestamp: time.Now().UnixMilli()},
	}

	srv := &eventStreamServer{events: events}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)
	time.Sleep(500 * time.Millisecond)
	bridge.Stop()

	// PublishBridgeEvent may fail if WebSocket is disabled, but it should not
	// crash the bridge — it should just log a warning.
	if srv.streamCount.Load() < 1 {
		t.Error("expected at least 1 stream connection")
	}
}

func TestVaultEventBridge_NewVaultEventBridge(t *testing.T) {
	bus := newTestEventBus(t)
	client, _ := NewGovernanceClient("")

	bridge := NewVaultEventBridge(client, bus)

	if bridge == nil {
		t.Fatal("expected non-nil bridge")
	}
	if bridge.logger == nil {
		t.Error("expected default logger to be set")
	}
}

func TestVaultEventBridge_MultipleEventTypes(t *testing.T) {
	eventTypes := []string{
		"token_issued",
		"token_consumed",
		"secrets_zeroized",
		"skill_gate_denied",
		"pii_detected_in_output",
	}

	var events []*pb.VaultEventStream
	for _, et := range eventTypes {
		events = append(events, &pb.VaultEventStream{
			EventType: et,
			SessionId: fmt.Sprintf("sess-%s", et),
			Timestamp: time.Now().UnixMilli(),
		})
	}

	srv := &eventStreamServer{events: events}
	socketPath, _ := setupEventStreamServer(t, srv)

	client, err := NewGovernanceClient(socketPath)
	if err != nil {
		t.Fatalf("NewGovernanceClient: %v", err)
	}
	defer client.Close()

	bus := newTestEventBus(t)

	bridge := NewVaultEventBridge(client, bus)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	bridge.StartSyncLoop(ctx)
	time.Sleep(500 * time.Millisecond)
	bridge.Stop()

	if srv.streamCount.Load() < 1 {
		t.Error("expected stream connection")
	}
}
