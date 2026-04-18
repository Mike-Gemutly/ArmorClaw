package email

import (
	"sync"
	"testing"
	"time"
)

func TestAddMessage_InReplyToChain(t *testing.T) {
	tracker := NewThreadTracker()

	baseTime := time.Now().Add(-2 * time.Hour)

	parent := ThreadMessage{
		MessageID:   "<parent@example.com>",
		Subject:     "Original topic",
		From:        "alice@example.com",
		To:          []string{"bob@example.com"},
		Date:        baseTime,
		BodySnippet: "Starting a discussion",
		EmailID:     "email-001",
	}

	reply := ThreadMessage{
		MessageID:   "<reply@example.com>",
		InReplyTo:   "<parent@example.com>",
		Subject:     "Re: Original topic",
		From:        "bob@example.com",
		To:          []string{"alice@example.com"},
		Date:        baseTime.Add(time.Hour),
		BodySnippet: "My response",
		EmailID:     "email-002",
	}

	tracker.AddMessage(parent)
	tracker.AddMessage(reply)

	threadID := tracker.ResolveThreadID(reply)
	if threadID != "<parent@example.com>" {
		t.Fatalf("expected thread root %q, got %q", "<parent@example.com>", threadID)
	}

	ctx, err := tracker.GetThreadContext(threadID)
	if err != nil {
		t.Fatalf("GetThreadContext returned error: %v", err)
	}

	if len(ctx) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(ctx))
	}

	if ctx[0].MessageID != "<parent@example.com>" {
		t.Errorf("first message should be parent, got %q", ctx[0].MessageID)
	}
	if ctx[1].MessageID != "<reply@example.com>" {
		t.Errorf("second message should be reply, got %q", ctx[1].MessageID)
	}
}

func TestAddMessage_ReferencesWithAncestors(t *testing.T) {
	tracker := NewThreadTracker()

	baseTime := time.Now().Add(-3 * time.Hour)

	root := ThreadMessage{
		MessageID:   "<root@example.com>",
		Subject:     "Deep thread",
		From:        "alice@example.com",
		To:          []string{"bob@example.com"},
		Date:        baseTime,
		BodySnippet: "Root message",
		EmailID:     "email-010",
	}

	mid := ThreadMessage{
		MessageID:   "<mid@example.com>",
		InReplyTo:   "<root@example.com>",
		References:  []string{"<root@example.com>"},
		Subject:     "Re: Deep thread",
		From:        "bob@example.com",
		To:          []string{"alice@example.com"},
		Date:        baseTime.Add(time.Hour),
		BodySnippet: "Middle message",
		EmailID:     "email-011",
	}

	leaf := ThreadMessage{
		MessageID:   "<leaf@example.com>",
		InReplyTo:   "<mid@example.com>",
		References:  []string{"<root@example.com>", "<mid@example.com>"},
		Subject:     "Re: Deep thread",
		From:        "carol@example.com",
		To:          []string{"bob@example.com"},
		Date:        baseTime.Add(2 * time.Hour),
		BodySnippet: "Leaf message",
		EmailID:     "email-012",
	}

	tracker.AddMessage(root)
	tracker.AddMessage(mid)
	tracker.AddMessage(leaf)

	threadID := tracker.ResolveThreadID(leaf)
	if threadID != "<root@example.com>" {
		t.Fatalf("expected root %q, got %q", "<root@example.com>", threadID)
	}

	ctx, err := tracker.GetThreadContext(threadID)
	if err != nil {
		t.Fatalf("GetThreadContext returned error: %v", err)
	}

	if len(ctx) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(ctx))
	}

	expected := []string{"<root@example.com>", "<mid@example.com>", "<leaf@example.com>"}
	for i, msg := range ctx {
		if msg.MessageID != expected[i] {
			t.Errorf("position %d: expected %q, got %q", i, expected[i], msg.MessageID)
		}
	}
}

func TestAddMessage_OrphanEmail(t *testing.T) {
	tracker := NewThreadTracker()

	orphan := ThreadMessage{
		MessageID:   "<orphan@example.com>",
		Subject:     "Standalone message",
		From:        "alice@example.com",
		To:          []string{"bob@example.com"},
		Date:        time.Now(),
		BodySnippet: "No replies",
		EmailID:     "email-020",
	}

	tracker.AddMessage(orphan)

	threadID := tracker.ResolveThreadID(orphan)
	if threadID != "<orphan@example.com>" {
		t.Fatalf("orphan should be its own thread root, got %q", threadID)
	}

	ctx, err := tracker.GetThreadContext(threadID)
	if err != nil {
		t.Fatalf("GetThreadContext returned error: %v", err)
	}

	if len(ctx) != 1 {
		t.Fatalf("expected 1 message, got %d", len(ctx))
	}
	if ctx[0].EmailID != "email-020" {
		t.Errorf("expected email-020, got %q", ctx[0].EmailID)
	}
}

func TestAddMessage_ConcurrentSafety(t *testing.T) {
	tracker := NewThreadTracker()
	var wg sync.WaitGroup

	baseTime := time.Now()

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			msg := ThreadMessage{
				MessageID:   fmtMsgID(n),
				Subject:     "concurrent test",
				From:        "sender@example.com",
				To:          []string{"recv@example.com"},
				Date:        baseTime.Add(time.Duration(n) * time.Minute),
				BodySnippet: "body",
				EmailID:     fmtEmailID(n),
			}
			tracker.AddMessage(msg)
		}(i)
	}

	wg.Wait()

	count := 0
	for i := 0; i < 100; i++ {
		id := fmtMsgID(i)
		threadID := tracker.ResolveThreadID(ThreadMessage{MessageID: id})
		ctx, err := tracker.GetThreadContext(threadID)
		if err != nil {
			t.Errorf("thread for %q: %v", id, err)
			continue
		}
		count += len(ctx)
	}

	if count != 100 {
		t.Errorf("expected 100 total messages across threads, got %d", count)
	}
}

func TestAddMessage_DuplicateIgnored(t *testing.T) {
	tracker := NewThreadTracker()

	msg := ThreadMessage{
		MessageID:   "<dup@example.com>",
		Subject:     "test",
		From:        "a@b.com",
		Date:        time.Now(),
		BodySnippet: "first",
		EmailID:     "e1",
	}

	tracker.AddMessage(msg)
	msg.BodySnippet = "second"
	msg.EmailID = "e2"
	tracker.AddMessage(msg)

	threadID := tracker.ResolveThreadID(msg)
	ctx, err := tracker.GetThreadContext(threadID)
	if err != nil {
		t.Fatalf("GetThreadContext: %v", err)
	}

	if len(ctx) != 1 {
		t.Fatalf("duplicate should be ignored, got %d messages", len(ctx))
	}
	if ctx[0].BodySnippet != "first" {
		t.Errorf("should retain original, got %q", ctx[0].BodySnippet)
	}
}

func TestAddMessage_EmptyMessageID(t *testing.T) {
	tracker := NewThreadTracker()

	msg := ThreadMessage{
		Subject:     "no id",
		From:        "a@b.com",
		Date:        time.Now(),
		BodySnippet: "body",
		EmailID:     "e1",
	}

	tracker.AddMessage(msg)

	if len(tracker.threads) != 0 {
		t.Fatal("empty Message-ID should not create a thread entry")
	}
}

func TestGetThreadContext_NotFound(t *testing.T) {
	tracker := NewThreadTracker()

	_, err := tracker.GetThreadContext("<nonexistent@example.com>")
	if err == nil {
		t.Fatal("expected error for unknown thread ID")
	}
}

func fmtMsgID(n int) string {
	return "<msg-" + string(rune('A'+n%26)) + string(rune('0'+n%10)) + "@example.com>"
}

func fmtEmailID(n int) string {
	return "email-" + string(rune('0'+n%10))
}
