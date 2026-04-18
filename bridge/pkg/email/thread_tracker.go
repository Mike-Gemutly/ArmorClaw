package email

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// ThreadMessage represents a single email message with thread-linking headers.
type ThreadMessage struct {
	MessageID   string    `json:"message_id"`
	InReplyTo   string    `json:"in_reply_to,omitempty"`
	References  []string  `json:"references,omitempty"`
	Subject     string    `json:"subject"`
	From        string    `json:"from"`
	To          []string  `json:"to"`
	Date        time.Time `json:"date"`
	BodySnippet string    `json:"body_snippet"`
	EmailID     string    `json:"email_id"`
}

// ThreadTracker groups emails by thread ID derived from Message-ID,
// In-Reply-To, and References headers (RFC 2822).
type ThreadTracker struct {
	mu       sync.RWMutex
	messages map[string]*ThreadMessage
	threads  map[string][]string
}

// NewThreadTracker creates a ready-to-use ThreadTracker.
func NewThreadTracker() *ThreadTracker {
	return &ThreadTracker{
		messages: make(map[string]*ThreadMessage),
		threads:  make(map[string][]string),
	}
}

// AddMessage records a message and links it into the correct thread.
// If the Message-ID is empty, the call is a no-op.
func (t *ThreadTracker) AddMessage(msg ThreadMessage) {
	if msg.MessageID == "" {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if _, exists := t.messages[msg.MessageID]; exists {
		return
	}

	clone := msg
	t.messages[msg.MessageID] = &clone

	rootID := t.resolveThreadIDLocked(&clone)

	t.threads[rootID] = append(t.threads[rootID], clone.MessageID)
}

// GetThreadContext returns all messages in the thread identified by threadID
// in chronological order. Returns an error if the thread is unknown.
func (t *ThreadTracker) GetThreadContext(threadID string) ([]ThreadMessage, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ids, ok := t.threads[threadID]
	if !ok {
		return nil, fmt.Errorf("thread %q not found", threadID)
	}

	msgs := make([]ThreadMessage, 0, len(ids))
	for _, id := range ids {
		if m, exists := t.messages[id]; exists {
			msgs = append(msgs, *m)
		}
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].Date.Before(msgs[j].Date)
	})

	return msgs, nil
}

// ResolveThreadID resolves the root Message-ID for a message by walking the
// References and In-Reply-To chain. The public version acquires the read lock.
func (t *ThreadTracker) ResolveThreadID(msg ThreadMessage) string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.resolveThreadIDLocked(&msg)
}

// resolveThreadIDLocked does the actual thread resolution without acquiring
// the lock (caller must hold at least a read lock).
//
// Algorithm (RFC 2822):
//  1. If References is non-empty, the first entry is the oldest ancestor —
//     use it as the root. Walk backwards through References to see if we
//     know a deeper root.
//  2. If References is empty but In-ReplyTo is set, look up that message
//     and recursively resolve its thread root.
//  3. Otherwise the message starts its own thread (its own Message-ID).
func (t *ThreadTracker) resolveThreadIDLocked(msg *ThreadMessage) string {
	// Walk References backwards to find the oldest known ancestor.
	for i := len(msg.References) - 1; i >= 0; i-- {
		refID := msg.References[i]
		if parent, ok := t.messages[refID]; ok {
			return t.resolveThreadIDLocked(parent)
		}
	}

	// If References is empty but we have In-Reply-To, follow it.
	if msg.InReplyTo != "" {
		if parent, ok := t.messages[msg.InReplyTo]; ok {
			return t.resolveThreadIDLocked(parent)
		}
		// Parent not yet seen — use In-ReplyTo as provisional root.
		return msg.InReplyTo
	}

	// First element of References is the root per RFC 2822.
	if len(msg.References) > 0 {
		return msg.References[0]
	}

	// Orphan message — it is its own thread root.
	return msg.MessageID
}
