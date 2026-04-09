package network

import (
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	if cb == nil {
		t.Fatal("NewCircuitBreaker returned nil")
	}

	if cb.GetState() != StateClosed {
		t.Errorf("Expected state %v, got %v", StateClosed, cb.GetState())
	}
}

func TestCircuitBreaker_Allow_ClosedState(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	if !cb.Allow() {
		t.Error("Expected Allow() to return true in CLOSED state")
	}
}

func TestCircuitBreaker_Allow_AfterThreshold(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if !cb.IsOpen() {
		t.Error("Expected circuit to be OPEN after threshold")
	}

	if cb.Allow() {
		t.Error("Expected Allow() to return false in OPEN state before timeout")
	}
}

func TestCircuitBreaker_Allow_HalfOpenTransition(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      100 * time.Millisecond,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if !cb.IsOpen() {
		t.Error("Expected circuit to be OPEN")
	}

	time.Sleep(150 * time.Millisecond)

	if !cb.Allow() {
		t.Error("Expected Allow() to return true in HALF_OPEN state after timeout")
	}

	if !cb.IsHalfOpen() {
		t.Error("Expected state to be HALF_OPEN")
	}
}

func TestCircuitBreaker_RecordSuccess_ClosedState(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	cb.RecordFailure()
	cb.RecordSuccess()

	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count to be 0, got %d", cb.GetFailureCount())
	}
}

func TestCircuitBreaker_RecordSuccess_HalfOpenToClosed(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      100 * time.Millisecond,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	time.Sleep(150 * time.Millisecond)
	cb.Allow()
	cb.RecordSuccess()

	if !cb.IsClosed() {
		t.Error("Expected state to be CLOSED after success in HALF_OPEN")
	}
}

func TestCircuitBreaker_RecordFailure_OpensCircuit(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  2,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	err := cb.RecordFailure()
	if err != nil {
		t.Errorf("First failure should not open circuit: %v", err)
	}

	err = cb.RecordFailure()
	if err == nil {
		t.Error("Second failure should open circuit and return error")
	}

	if !cb.IsOpen() {
		t.Error("Expected circuit to be OPEN")
	}
}

func TestCircuitBreaker_GetState(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	state := cb.GetState()
	if state != StateClosed {
		t.Errorf("Expected state %v, got %v", StateClosed, state)
	}

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	state = cb.GetState()
	if state != StateOpen {
		t.Errorf("Expected state %v, got %v", StateOpen, state)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if !cb.IsOpen() {
		t.Error("Expected circuit to be OPEN")
	}

	cb.Reset()

	if !cb.IsClosed() {
		t.Error("Expected circuit to be CLOSED after reset")
	}

	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count to be 0 after reset, got %d", cb.GetFailureCount())
	}
}

func TestCircuitBreaker_IsOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	if cb.IsOpen() {
		t.Error("Expected IsOpen() to return false in CLOSED state")
	}

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if !cb.IsOpen() {
		t.Error("Expected IsOpen() to return true in OPEN state")
	}
}

func TestCircuitBreaker_IsClosed(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      30 * time.Second,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	if !cb.IsClosed() {
		t.Error("Expected IsClosed() to return true in CLOSED state")
	}

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	if cb.IsClosed() {
		t.Error("Expected IsClosed() to return false in OPEN state")
	}
}

func TestCircuitBreaker_IsHalfOpen(t *testing.T) {
	config := CircuitBreakerConfig{
		FailureThreshold:  3,
		ResetTimeout:      100 * time.Millisecond,
		HalfOpenThreshold: 1,
	}

	cb := NewCircuitBreaker(config)

	if cb.IsHalfOpen() {
		t.Error("Expected IsHalfOpen() to return false in CLOSED state")
	}

	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	time.Sleep(150 * time.Millisecond)
	cb.Allow()

	if !cb.IsHalfOpen() {
		t.Error("Expected IsHalfOpen() to return true in HALF_OPEN state")
	}
}

func TestCircuitBreaker_StateString(t *testing.T) {
	tests := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF_OPEN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.state.String() != tt.expected {
				t.Errorf("State.String() = %s, want %s", tt.state.String(), tt.expected)
			}
		})
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{})

	if cb.failureThreshold != 3 {
		t.Errorf("Expected default failure threshold 3, got %d", cb.failureThreshold)
	}

	if cb.resetTimeout != 30*time.Second {
		t.Errorf("Expected default reset timeout 30s, got %v", cb.resetTimeout)
	}

	if cb.halfOpenThreshold != 1 {
		t.Errorf("Expected default half-open threshold 1, got %d", cb.halfOpenThreshold)
	}
}
