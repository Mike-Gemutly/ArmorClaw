package browser

import (
	"sync"
	"testing"
)

func TestAllocateNew(t *testing.T) {
	m := NewBrowserContextManager()
	cid := m.AllocateContext("agent-1")
	if cid == "" {
		t.Fatal("expected non-empty context ID")
	}
}

func TestAllocateExisting(t *testing.T) {
	m := NewBrowserContextManager()
	first := m.AllocateContext("agent-1")
	second := m.AllocateContext("agent-1")
	if first != second {
		t.Fatalf("same agent should get same context: got %q then %q", first, second)
	}
}

func TestAllocateDifferent(t *testing.T) {
	m := NewBrowserContextManager()
	a := m.AllocateContext("agent-1")
	b := m.AllocateContext("agent-2")
	if a == b {
		t.Fatalf("different agents should get different contexts: both got %q", a)
	}
}

func TestRelease(t *testing.T) {
	m := NewBrowserContextManager()
	m.AllocateContext("agent-1")

	if !m.ReleaseContext("agent-1") {
		t.Fatal("expected ReleaseContext to return true for existing agent")
	}
	if m.ReleaseContext("agent-1") {
		t.Fatal("expected ReleaseContext to return false for already-released agent")
	}
	if m.ActiveCount() != 0 {
		t.Fatalf("expected 0 active contexts, got %d", m.ActiveCount())
	}
}

func TestGetContext_Exists(t *testing.T) {
	m := NewBrowserContextManager()
	cid := m.AllocateContext("agent-1")

	got, ok := m.GetContext("agent-1")
	if !ok {
		t.Fatal("expected to find context")
	}
	if got != cid {
		t.Fatalf("expected %q, got %q", cid, got)
	}
}

func TestGetContext_NotFound(t *testing.T) {
	m := NewBrowserContextManager()
	_, ok := m.GetContext("no-such-agent")
	if ok {
		t.Fatal("expected not to find context")
	}
}

func TestConcurrentAllocation(t *testing.T) {
	m := NewBrowserContextManager()
	const goroutines = 100

	var wg sync.WaitGroup
	results := make(chan struct {
		agentID string
		cid     string
	}, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			agentID := "agent-concurrent"
			cid := m.AllocateContext(agentID)
			results <- struct {
				agentID string
				cid     string
			}{agentID: agentID, cid: cid}
		}(i)
	}
	wg.Wait()
	close(results)

	var first string
	count := 0
	for r := range results {
		count++
		if first == "" {
			first = r.cid
		}
		if r.cid != first {
			t.Fatalf("concurrent allocations for same agent returned different contexts: %q vs %q", first, r.cid)
		}
	}
	if count != goroutines {
		t.Fatalf("expected %d results, got %d", goroutines, count)
	}
	if m.ActiveCount() != 1 {
		t.Fatalf("expected 1 active context, got %d", m.ActiveCount())
	}
}

func TestConcurrentDifferentAgents(t *testing.T) {
	m := NewBrowserContextManager()
	const goroutines = 100

	var wg sync.WaitGroup
	results := make(chan string, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cid := m.AllocateContext("agent-diff")
			results <- cid
		}(i)
	}
	wg.Wait()
	close(results)

	seen := map[string]bool{}
	for cid := range results {
		seen[cid] = true
	}
	if len(seen) != 1 {
		t.Fatalf("expected all goroutines to get the same context for one agent, got %d unique", len(seen))
	}
}
