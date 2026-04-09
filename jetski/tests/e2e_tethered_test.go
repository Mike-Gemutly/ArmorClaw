package jetski_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/armorclaw/jetski/internal/approval"
	"github.com/armorclaw/jetski/internal/cdp"
	"github.com/armorclaw/jetski/internal/security"
	"github.com/armorclaw/jetski/internal/sonar"
	"github.com/gorilla/websocket"
)

func skipNoCGO(t *testing.T) {
	t.Helper()
	if runtime.Compiler != "gc" {
		t.Skip("CGO not available: compiler is not gc")
	}
	if os.Getenv("CGO_ENABLED") == "0" {
		t.Skip("CGO explicitly disabled")
	}
}

type captureEcho struct {
	mu       sync.Mutex
	messages [][]byte
}

func (c *captureEcho) last() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.messages) == 0 {
		return ""
	}
	return string(c.messages[len(c.messages)-1])
}

func (c *captureEcho) waitUntil(n int, timeout time.Duration) error {
	deadline := time.After(timeout)
	for {
		c.mu.Lock()
		count := len(c.messages)
		c.mu.Unlock()
		if count >= n {
			return nil
		}
		select {
		case <-deadline:
			return fmt.Errorf("timed out waiting for %d messages, got %d", n, count)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func newCaptureEchoServer(c *captureEcho) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			mt, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			c.mu.Lock()
			c.messages = append(c.messages, data)
			c.mu.Unlock()
			_ = conn.WriteMessage(mt, data)
		}
	}))
}

func wsURL(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

func sendCDPMessage(t *testing.T, conn *websocket.Conn, id int, method string, params any) {
	t.Helper()
	msg := map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal CDP message: %v", err)
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("write CDP message: %v", err)
	}
}

func startProxy(t *testing.T, engineURL string, tethered bool, buf *sonar.CircularBuffer) (*cdp.Proxy, *websocket.Conn) {
	t.Helper()
	router := cdp.NewMethodRouter(cdp.NewTranslator())
	scanner := security.NewPIIScanner()
	proxy := cdp.NewProxy(engineURL, router, scanner, tethered)
	if buf != nil {
		proxy.SetRecorder(func(method string, params json.RawMessage) {
			sonar.RecordFrame(buf, method, params, "e2e-session")
		})
	}
	clientConn, _, err := websocket.DefaultDialer.Dial(engineURL, nil)
	if err != nil {
		t.Fatalf("dial mock engine: %v", err)
	}
	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("proxy.Start: %v", err)
	}
	return proxy, clientConn
}

func TestE2ETethered_FullPipeline(t *testing.T) {
	var cap captureEcho
	server := newCaptureEchoServer(&cap)
	defer server.Close()

	buf := sonar.NewCircularBuffer(100)
	proxy, clientConn := startProxy(t, wsURL(server), true, buf)
	defer proxy.Stop()
	defer clientConn.Close()

	sendCDPMessage(t, clientConn, 1, "Input.insertText", map[string]string{"text": "123-45-6789"})

	if err := cap.waitUntil(1, 3*time.Second); err != nil {
		t.Fatal(err)
	}

	received := cap.last()
	if strings.Contains(received, "123-45-6789") {
		t.Errorf("raw SSN reached engine (tethered should scrub): %s", received)
	}
	if !strings.Contains(received, "[REDACTED_SSN]") {
		t.Errorf("expected [REDACTED_SSN] in engine-bound message: %s", received)
	}

	if buf.Count() == 0 {
		t.Fatal("Sonar buffer should record frames in tethered mode")
	}
	all := buf.GetAll()
	found := false
	for _, f := range all {
		if f.Method == "Input.insertText" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Sonar did not record Input.insertText; methods: %v", frameMethods(all))
	}

	t.Run("SQLCipherSession", func(t *testing.T) {
		skipNoCGO(t)
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "e2e.db")

		store, err := security.NewSQLCipherSessionStore(dbPath, "e2e-passphrase")
		if err != nil {
			t.Fatalf("NewSQLCipherSessionStore: %v", err)
		}

		session := security.Session{
			ID:        "e2e-sess-1",
			UserAgent: "E2E-Test-Agent",
			Cookies:   []byte("session-cookie-data"),
			ExpiresAt: 9999999999,
		}
		if err := store.CreateSession(session); err != nil {
			t.Fatalf("CreateSession: %v", err)
		}

		got, err := store.GetSession("e2e-sess-1")
		if err != nil {
			t.Fatalf("GetSession: %v", err)
		}
		if got.ID != session.ID || string(got.Cookies) != string(session.Cookies) {
			t.Errorf("round-trip mismatch: got %+v, want %+v", got, session)
		}

		if err := store.CloseSession("e2e-sess-1"); err != nil {
			t.Fatalf("CloseSession: %v", err)
		}

		_, err = store.GetSession("e2e-sess-1")
		if err == nil {
			t.Fatal("expected error after closing session")
		}
		store.Close()
	})
}

func TestE2EFreeRide_PIIWarningOnly(t *testing.T) {
	var cap captureEcho
	server := newCaptureEchoServer(&cap)
	defer server.Close()

	buf := sonar.NewCircularBuffer(100)
	proxy, clientConn := startProxy(t, wsURL(server), false, buf)
	defer proxy.Stop()
	defer clientConn.Close()

	sendCDPMessage(t, clientConn, 1, "Input.insertText", map[string]string{"text": "123-45-6789"})

	if err := cap.waitUntil(1, 3*time.Second); err != nil {
		t.Fatal(err)
	}

	received := cap.last()
	if !strings.Contains(received, "123-45-6789") {
		t.Errorf("free-ride mode should NOT scrub PII, but SSN missing: %s", received)
	}
	if strings.Contains(received, "[REDACTED_SSN]") {
		t.Errorf("free-ride mode should NOT contain [REDACTED_SSN]: %s", received)
	}

	if buf.Count() == 0 {
		t.Fatal("Sonar should record events in free-ride mode")
	}
}

func TestE2E_SQLCipherSessionPersistence(t *testing.T) {
	skipNoCGO(t)

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "persist.db")
	passphrase := "strong-passphrase-123"

	store1, err := security.NewSQLCipherSessionStore(dbPath, passphrase)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}

	original := security.Session{
		ID:        "persist-sess",
		UserAgent: "E2E-Persistence-Test",
		Cookies:   []byte("cookie-payload-for-persistence"),
		ExpiresAt: 1735689600,
	}
	if err := store1.CreateSession(original); err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	store1.Close()

	store2, err := security.NewSQLCipherSessionStore(dbPath, passphrase)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer store2.Close()

	got, err := store2.GetSession("persist-sess")
	if err != nil {
		t.Fatalf("GetSession after reopen: %v", err)
	}

	if got.ID != original.ID {
		t.Errorf("ID: got %q, want %q", got.ID, original.ID)
	}
	if got.UserAgent != original.UserAgent {
		t.Errorf("UserAgent: got %q, want %q", got.UserAgent, original.UserAgent)
	}
	if string(got.Cookies) != string(original.Cookies) {
		t.Errorf("Cookies: got %q, want %q", got.Cookies, original.Cookies)
	}
	if got.ExpiresAt != original.ExpiresAt {
		t.Errorf("ExpiresAt: got %d, want %d", got.ExpiresAt, original.ExpiresAt)
	}

	sessions, err := store2.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
}

func TestE2E_ApprovalTimeout(t *testing.T) {
	client := approval.NewApprovalClient("http://127.0.0.1:1", "!e2e:room", 100*time.Millisecond)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	approved, err := client.RequestApproval(ctx, approval.OpSessionCreate, "test operation")
	if err != nil {
		t.Fatalf("RequestApproval returned error (expected nil with auto-deny): %v", err)
	}
	if approved {
		t.Error("expected auto-deny (false) on timeout, got approved")
	}
}

func TestE2E_SonarTelemetry(t *testing.T) {
	buf := sonar.NewCircularBuffer(5)

	for i := range 7 {
		params := json.RawMessage(`{"seq":` + strconv.Itoa(i) + `}`)
		sonar.RecordFrame(buf, "Page.navigate", params, "sonar-session")
	}

	if buf.Count() != 5 {
		t.Errorf("expected buffer count 5, got %d", buf.Count())
	}

	all := buf.GetAll()
	if len(all) != 5 {
		t.Fatalf("expected 5 frames, got %d", len(all))
	}

	firstParams := string(all[0].Params)
	if !strings.Contains(firstParams, `"seq":2`) {
		t.Errorf("expected oldest surviving frame to be seq 2, got params: %s", firstParams)
	}

	lastParams := string(all[4].Params)
	if !strings.Contains(lastParams, `"seq":6`) {
		t.Errorf("expected newest frame to be seq 6, got params: %s", lastParams)
	}

	for _, f := range all {
		if f.Method != "Page.navigate" {
			t.Errorf("expected Page.navigate, got %s", f.Method)
		}
		if f.SessionID != "sonar-session" {
			t.Errorf("expected sonar-session, got %s", f.SessionID)
		}
	}

	for i := 1; i < len(all); i++ {
		prevSeq := extractSeq(all[i-1].Params)
		currSeq := extractSeq(all[i].Params)
		if currSeq <= prevSeq {
			t.Errorf("frames not in chronological order: seq %d before %d", currSeq, prevSeq)
		}
	}
}

func extractSeq(params json.RawMessage) int {
	var m map[string]int
	if json.Unmarshal(params, &m) != nil {
		return -1
	}
	return m["seq"]
}

func frameMethods(frames []sonar.CDPFrame) []string {
	names := make([]string, len(frames))
	for i, f := range frames {
		names[i] = f.Method
	}
	return names
}
