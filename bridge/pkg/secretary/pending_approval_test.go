package secretary

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/armorclaw/bridge/internal/events"
)

func TestPendingApprovalApproved(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	stepID := "step-approved-001"

	fieldsCh := make(chan []string, 1)
	errCh := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		f, err := PendingApproval(ctx, bus, "!room:test", stepID, []string{"payment.card_number"})
		if err != nil {
			errCh <- err
			return
		}
		fieldsCh <- f
	}()

	time.Sleep(100 * time.Millisecond)

	HandlePIIResponse(stepID, true, []string{"payment.card_number"})

	select {
	case fields := <-fieldsCh:
		if len(fields) != 1 || fields[0] != "payment.card_number" {
			t.Errorf("expected [payment.card_number], got %v", fields)
		}
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for approval")
	}
}

func TestPendingApprovalDenied(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	stepID := "step-denied-001"

	errCh := make(chan error, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		_, err := PendingApproval(ctx, bus, "!room:test", stepID, []string{"payment.card_number"})
		errCh <- err
	}()

	time.Sleep(100 * time.Millisecond)

	HandlePIIResponse(stepID, false, nil)

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected error for denied approval")
		}
		if !strings.Contains(err.Error(), "denied") {
			t.Errorf("expected error to contain 'denied', got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for denial")
	}
}

func TestPendingApprovalTimeout(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	stepID := "step-timeout-001"

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	start := time.Now()
	_, err := PendingApproval(ctx, bus, "!room:test", stepID, []string{"payment.card_number"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error on timeout")
	}
	if !strings.Contains(err.Error(), "cancelled") && !strings.Contains(err.Error(), "timed out") {
		t.Errorf("expected timeout/cancel error, got: %v", err)
	}
	if elapsed > 5*time.Second {
		t.Errorf("took too long: %v", elapsed)
	}
}

func TestPendingApprovalEmitsEvent(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	stepID := "step-emit-001"

	sub := bus.Subscribe()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		PendingApproval(ctx, bus, "!room:test", stepID, []string{"payment.card_number"})
	}()

	select {
	case evt := <-sub:
		if evt.Type != PIIRequestEventType {
			t.Errorf("expected event type %q, got %q", PIIRequestEventType, evt.Type)
		}
		if evt.RoomID != "!room:test" {
			t.Errorf("expected room ID !room:test, got %q", evt.RoomID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	HandlePIIResponse(stepID, false, nil)
}

func TestPendingApprovalCtxCancelled(t *testing.T) {
	bus := events.NewMatrixEventBus(64)
	stepID := "step-ctx-cancel-001"

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := PendingApproval(ctx, bus, "!room:test", stepID, []string{"payment.card_number"})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Errorf("expected 'cancelled' in error, got: %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("should return immediately on cancelled ctx, took %v", elapsed)
	}
}
