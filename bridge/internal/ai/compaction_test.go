package ai

import (
	"context"
	"strings"
	"testing"
)

// mockAIClient is a test double for AIClient that returns canned responses.
type mockAIClient struct {
	response *ChatResponse
	err      error
	called   bool
}

func (m *mockAIClient) Chat(_ context.Context, _ ChatRequest) (*ChatResponse, error) {
	m.called = true
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func (m *mockAIClient) ChatStream(_ context.Context, _ ChatRequest) (<-chan ChatChunk, error) {
	return nil, nil
}

// makeMessages creates n messages each with contentLen characters.
func makeMessages(n, contentLen int) []Message {
	content := strings.Repeat("x", contentLen)
	msgs := make([]Message, n)
	for i := range msgs {
		msgs[i] = Message{Role: "user", Content: content}
	}
	return msgs
}

func TestEstimateMessageTokens(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		wantMin  int // lower bound (inclusive)
		wantMax  int // upper bound (inclusive)
	}{
		{
			name:     "empty slice",
			messages: nil,
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "single short message",
			messages: []Message{{Role: "user", Content: "hello"}},
			wantMin:  1,
			wantMax:  10,
		},
		{
			name:     "message scales with content",
			messages: makeMessages(1, 4000), // 4000 chars / 4 = ~1000 tokens
			wantMin:  900,
			wantMax:  1100,
		},
		{
			name:     "multiple messages are summed",
			messages: makeMessages(10, 400), // 10 * ~100 tokens = ~1000 tokens
			wantMin:  900,
			wantMax:  1100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateMessageTokens(tt.messages)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("EstimateMessageTokens() = %d, want between %d and %d", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestShouldCompact(t *testing.T) {
	t.Run("below threshold returns false", func(t *testing.T) {
		msgs := makeMessages(1, 400) // ~100 tokens
		if ShouldCompact(msgs, 50000) {
			t.Error("ShouldCompact() = true, want false (below threshold)")
		}
	})

	t.Run("above threshold returns true", func(t *testing.T) {
		// 100 messages * 4000 chars each = ~100000 tokens, well over 50000
		msgs := makeMessages(100, 4000)
		if !ShouldCompact(msgs, 50000) {
			t.Error("ShouldCompact() = false, want true (above threshold)")
		}
	})

	t.Run("zero threshold uses default", func(t *testing.T) {
		msgs := makeMessages(1, 100)
		// Should not trigger compaction with default 100k threshold
		if ShouldCompact(msgs, 0) {
			t.Error("ShouldCompact() with zero threshold should use default and return false for small messages")
		}
	})
}

func TestCompactHistory_NoCompactionBelowThreshold(t *testing.T) {
	msgs := makeMessages(5, 100) // well below any reasonable threshold
	threshold := 50000

	result, err := CompactHistory(context.Background(), nil, msgs, threshold)
	if err != nil {
		t.Fatalf("CompactHistory() error: %v", err)
	}

	if len(result) != len(msgs) {
		t.Errorf("CompactHistory() returned %d messages, want %d (no compaction expected)", len(result), len(msgs))
	}
}

func TestCompactHistory_TriggersAboveThreshold(t *testing.T) {
	// Build messages that exceed 50000 tokens.
	// 100 msgs * 4000 chars = ~100k tokens > 50k threshold.
	msgs := makeMessages(100, 4000)
	threshold := 50000

	mock := &mockAIClient{
		response: &ChatResponse{
			Content:      "This is a compacted summary of the session.",
			FinishReason: "stop",
		},
	}

	result, err := CompactHistory(context.Background(), mock, msgs, threshold)
	if err != nil {
		t.Fatalf("CompactHistory() error: %v", err)
	}

	if len(result) >= len(msgs) {
		t.Errorf("CompactHistory() returned %d messages, expected fewer than %d", len(result), len(msgs))
	}

	if len(result) == 0 {
		t.Fatal("CompactHistory() returned empty messages")
	}

	// The compacted result should be a single system message with the summary.
	if result[0].Role != "system" {
		t.Errorf("compacted message role = %q, want %q", result[0].Role, "system")
	}

	if !strings.Contains(result[0].Content, "compacted summary") {
		t.Errorf("compacted message should contain summary text, got: %s", result[0].Content)
	}

	if !mock.called {
		t.Error("expected LLM Chat() to be called for summarization")
	}
}

func TestCompactHistory_CompactedOutputIsShorter(t *testing.T) {
	// 200 messages * 4000 chars = ~200k tokens > 50k threshold.
	msgs := makeMessages(200, 4000)
	threshold := 50000

	mock := &mockAIClient{
		response: &ChatResponse{
			Content:      "Short summary.",
			FinishReason: "stop",
		},
	}

	result, err := CompactHistory(context.Background(), mock, msgs, threshold)
	if err != nil {
		t.Fatalf("CompactHistory() error: %v", err)
	}

	originalTokens := EstimateMessageTokens(msgs)
	compactedTokens := EstimateMessageTokens(result)

	if compactedTokens >= originalTokens {
		t.Errorf("compacted tokens (%d) should be less than original (%d)", compactedTokens, originalTokens)
	}
}

func TestCompactHistory_FallbackOnNilClient(t *testing.T) {
	// Large messages with no client → should use truncation fallback.
	msgs := makeMessages(100, 4000)
	threshold := 50000

	result, err := CompactHistory(context.Background(), nil, msgs, threshold)
	if err != nil {
		t.Fatalf("CompactHistory() error: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("CompactHistory() with nil client should still produce output via fallback")
	}

	if result[0].Role != "system" {
		t.Errorf("fallback role = %q, want %q", result[0].Role, "system")
	}

	// Fallback should mention "session had" or "[First]"
	if !strings.Contains(result[0].Content, "Session had") {
		t.Errorf("fallback should contain session metadata, got: %s", result[0].Content)
	}
}

func TestCompactHistory_FallbackOnClientError(t *testing.T) {
	msgs := makeMessages(100, 4000)
	threshold := 50000

	mock := &mockAIClient{
		err:    context.DeadlineExceeded,
		called: false,
	}

	result, err := CompactHistory(context.Background(), mock, msgs, threshold)
	if err != nil {
		t.Fatalf("CompactHistory() error: %v", err)
	}

	if len(result) == 0 {
		t.Fatal("CompactHistory() should produce fallback output even on client error")
	}

	if !mock.called {
		t.Error("expected mock client to be attempted")
	}

	// Verify it used truncation fallback, not LLM response.
	if !strings.Contains(result[0].Content, "Session had") {
		t.Errorf("should use truncation fallback on error, got: %s", result[0].Content)
	}
}
