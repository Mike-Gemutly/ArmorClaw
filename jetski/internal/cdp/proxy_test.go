package cdp

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/armorclaw/jetski/internal/approval"
	"github.com/armorclaw/jetski/internal/security"
	"github.com/gorilla/websocket"
)

func TestMethodRouter_Route(t *testing.T) {
	router := NewMethodRouter(NewTranslator())

	tests := []struct {
		name           string
		method         string
		expectedAction RouteAction
	}{
		{
			name:           "Mouse click should translate",
			method:         "Input.dispatchMouseEvent",
			expectedAction: ActionTranslate,
		},
		{
			name:           "Key input should translate",
			method:         "Input.dispatchKeyEvent",
			expectedAction: ActionTranslate,
		},
		{
			name:           "Text insert should translate",
			method:         "Input.insertText",
			expectedAction: ActionTranslate,
		},
		{
			name:           "Page method should passthrough",
			method:         "Page.navigate",
			expectedAction: ActionPassthrough,
		},
		{
			name:           "Runtime method should translate",
			method:         "Runtime.evaluate",
			expectedAction: ActionTranslate,
		},
		{
			name:           "Network method should passthrough",
			method:         "Network.enable",
			expectedAction: ActionPassthrough,
		},
		{
			name:           "Unknown method should be unsupported",
			method:         "UnknownDomain.unknownMethod",
			expectedAction: ActionUnsupported,
		},
		{
			name:           "Target wildcard should passthrough",
			method:         "Target.createTarget",
			expectedAction: ActionPassthrough,
		},
		{
			name:           "Another Target wildcard should passthrough",
			method:         "Target.closeTarget",
			expectedAction: ActionPassthrough,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			route := router.Route(tt.method)
			if route == nil {
				t.Error("Route should not be nil")
				return
			}
			if route.Action != tt.expectedAction {
				t.Errorf("Expected action %s, got %s", tt.expectedAction, route.Action)
			}
		})
	}
}

func TestMethodRouter_WildcardMatching(t *testing.T) {
	router := NewMethodRouter(NewTranslator())

	targetMethods := []string{
		"Target.createTarget",
		"Target.closeTarget",
		"Target.attachToTarget",
		"Target.detachFromTarget",
		"Target.sendMessageToTarget",
	}

	for _, method := range targetMethods {
		t.Run(method, func(t *testing.T) {
			route := router.Route(method)
			if route == nil {
				t.Error("Route should not be nil")
				return
			}
			if route.Action != ActionPassthrough {
				t.Errorf("Expected passthrough for %s, got %s", method, route.Action)
			}
		})
	}
}

func TestMethodRouter_HandlerInvocation(t *testing.T) {
	router := NewMethodRouter(NewTranslator())

	msg := &CDPMessage{
		ID:     1,
		Method: "Input.dispatchMouseEvent",
	}

	route := router.Route(msg.Method)
	if route == nil {
		t.Fatal("Route should not be nil")
	}

	if route.Handler == nil {
		t.Error("Handler should not be nil for mouse click")
	}

	result, err := route.Handler(msg)
	if err != nil {
		t.Errorf("Handler should not return error: %v", err)
	}
	if result == nil {
		t.Error("Handler should return a message")
	}
}

func TestProxy_NewProxy(t *testing.T) {
	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy("ws://localhost:9222", router, nil, false)

	if proxy == nil {
		t.Fatal("NewProxy should return a non-nil proxy")
	}
	if proxy.engineURL != "ws://localhost:9222" {
		t.Errorf("Expected engine URL ws://localhost:9222, got %s", proxy.engineURL)
	}
	if proxy.router == nil {
		t.Error("Router should not be nil")
	}
}

func TestProxy_BidirectionalMessageForwarding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()

	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy(wsURL, router, nil, false)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	testMessage := CDPMessage{
		ID:     1,
		Method: "Runtime.evaluate",
		Params: json.RawMessage(`{"expression":"1+1"}`),
	}

	data, err := json.Marshal(testMessage)
	if err != nil {
		t.Fatalf("Failed to marshal message: %v", err)
	}

	err = clientConn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	select {
	case err := <-proxy.Errors():
		if err != nil {
			t.Errorf("Proxy error: %v", err)
		}
	case <-time.After(1 * time.Second):
	}
}

func TestProxy_PingPong(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()

	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy(wsURL, router, nil, false)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	select {
	case err := <-proxy.Errors():
		if err != nil {
			t.Errorf("Proxy error: %v", err)
		}
	case <-time.After(100 * time.Millisecond):
	}
}

func TestProxy_Stop(t *testing.T) {
	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy("ws://localhost:9222", router, nil, false)

	proxy.Stop()

	if proxy.clientConn != nil {
		t.Error("clientConn should be nil after stop")
	}
	if proxy.engineConn != nil {
		t.Error("engineConn should be nil after stop")
	}
}

func TestCDPMessage_MarshalUnmarshal(t *testing.T) {
	original := CDPMessage{
		ID:     42,
		Method: "Runtime.evaluate",
		Params: json.RawMessage(`{"expression":"test"}`),
		Error: &CDPError{
			Code:    -32601,
			Message: "Method not found",
			Data:    "Runtime.evaluate",
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded CDPMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID mismatch: got %d, want %d", decoded.ID, original.ID)
	}
	if decoded.Method != original.Method {
		t.Errorf("Method mismatch: got %s, want %s", decoded.Method, original.Method)
	}
	if decoded.Error.Code != original.Error.Code {
		t.Errorf("Error code mismatch: got %d, want %d", decoded.Error.Code, original.Error.Code)
	}
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if err := conn.WriteMessage(messageType, data); err != nil {
			break
		}
	}
}

func TestProxy_RecorderCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()

	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy(wsURL, router, nil, false)

	var recorded []string
	proxy.SetRecorder(func(method string, params json.RawMessage) {
		recorded = append(recorded, method)
	})

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	}
	data, _ := json.Marshal(msg)
	clientConn.WriteMessage(websocket.TextMessage, data)

	select {
	case err := <-proxy.Errors():
		if err != nil {
			t.Errorf("Proxy error: %v", err)
		}
	case <-time.After(2 * time.Second):
	}

	if len(recorded) == 0 {
		t.Fatal("expected recorder to be called, but it was not")
	}
	found := false
	for _, m := range recorded {
		if m == "Page.navigate" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected Page.navigate in recorded methods, got %v", recorded)
	}
}

func TestProxy_SetRecorder_Nil(t *testing.T) {
	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy("ws://localhost:9222", router, nil, false)

	proxy.SetRecorder(nil)

	if proxy.recorder != nil {
		t.Error("expected recorder to be nil after SetRecorder(nil)")
	}
}

// --- PII Scanner Integration Tests ---

type mockPIIScanner struct {
	called   chan struct{}
	findings []security.PIIFinding
}

func newMockPIIScanner() *mockPIIScanner {
	return &mockPIIScanner{
		called:   make(chan struct{}, 1),
		findings: nil,
	}
}

func (m *mockPIIScanner) ScanJSONMessage(jsonStr string) ([]security.PIIFinding, error) {
	m.called <- struct{}{}
	return m.findings, nil
}

func TestPII_NewProxyWithScanner(t *testing.T) {
	router := NewMethodRouter(NewTranslator())
	mock := newMockPIIScanner()
	proxy := NewProxy("ws://localhost:9222", router, mock, false)

	if proxy == nil {
		t.Fatal("NewProxy should return a non-nil proxy")
	}
	if proxy.piiScanner == nil {
		t.Error("PII scanner should not be nil when provided")
	}
}

func TestPII_NilScannerAllowed(t *testing.T) {
	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy("ws://localhost:9222", router, nil, false)

	if proxy == nil {
		t.Fatal("NewProxy should return a non-nil proxy")
	}
	if proxy.piiScanner != nil {
		t.Error("PII scanner should be nil when not provided")
	}
}

func TestPII_ScannerCalledOnForwardToEngine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	mock := newMockPIIScanner()
	proxy := NewProxy(wsURL, router, mock, false)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Input.insertText",
		Params: json.RawMessage(`{"text":"hello world"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	select {
	case <-mock.called:
	case <-time.After(2 * time.Second):
		t.Error("PII scanner should have been called when forwarding message")
	}
}

func TestPII_DetectionLogsWarning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	scanner := security.NewPIIScanner()
	proxy := NewProxy(wsURL, router, scanner, false)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Input.insertText",
		Params: json.RawMessage(`{"text":"123-45-6789"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "[JETSKI PII]") {
		t.Errorf("Expected [JETSKI PII] in log output, got: %s", output)
	}
	if !strings.Contains(output, "SSN") {
		t.Errorf("Expected SSN type in log output, got: %s", output)
	}
}

func TestPII_NoWarningForCleanMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	scanner := security.NewPIIScanner()
	proxy := NewProxy(wsURL, router, scanner, false)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	output := buf.String()
	if strings.Contains(output, "[JETSKI PII]") {
		t.Errorf("Expected no [JETSKI PII] for clean message, got: %s", output)
	}
}

func TestPII_EmailDetectionLogsWarning(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	scanner := security.NewPIIScanner()
	proxy := NewProxy(wsURL, router, scanner, false)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Input.insertText",
		Params: json.RawMessage(`{"text":"user@example.com"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	output := buf.String()
	if !strings.Contains(output, "[JETSKI PII]") {
		t.Errorf("Expected [JETSKI PII] in log output, got: %s", output)
	}
	if !strings.Contains(output, "EMAIL") {
		t.Errorf("Expected EMAIL type in log output, got: %s", output)
	}
}

// --- Approval Gating Tests ---

type mockApprovalClient struct {
	called   chan approvalCall
	response bool   // approved or denied
	err      error  // error to return
	opType   string // captured operation type
	detail   string // captured detail
}

type approvalCall struct {
	opType approval.OperationType
	detail string
}

func newMockApprovalClient(response bool) *mockApprovalClient {
	return &mockApprovalClient{
		called:   make(chan approvalCall, 10),
		response: response,
	}
}

func (m *mockApprovalClient) RequestApproval(ctx context.Context, op approval.OperationType, detail string) (bool, error) {
	m.opType = string(op)
	m.detail = detail
	m.called <- approvalCall{opType: op, detail: detail}
	if m.err != nil {
		return false, m.err
	}
	return m.response, nil
}

func TestApproval_InputInsertTextWithSSN(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	scanner := security.NewPIIScanner()
	proxy := NewProxy(wsURL, router, scanner, false)

	mockApproval := newMockApprovalClient(true)
	proxy.SetApprovalClient(mockApproval)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Input.insertText",
		Params: json.RawMessage(`{"text":"SSN: 123-45-6789"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	select {
	case call := <-mockApproval.called:
		if call.opType != approval.OpPIIInput {
			t.Errorf("Expected opType %s, got %s", approval.OpPIIInput, call.opType)
		}
		if !strings.Contains(call.detail, "SSN") {
			t.Errorf("Expected detail to contain 'SSN', got %s", call.detail)
		}
	case <-time.After(2 * time.Second):
		t.Error("Approval should have been requested for SSN in Input.insertText")
	}
}

func TestApproval_InputInsertTextWithCreditCard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	scanner := security.NewPIIScanner()
	proxy := NewProxy(wsURL, router, scanner, false)

	mockApproval := newMockApprovalClient(true)
	proxy.SetApprovalClient(mockApproval)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Input.insertText",
		Params: json.RawMessage(`{"text":"Card: 4111-1111-1111-1111"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	select {
	case call := <-mockApproval.called:
		if call.opType != approval.OpPIIInput {
			t.Errorf("Expected opType %s, got %s", approval.OpPIIInput, call.opType)
		}
		if !strings.Contains(call.detail, "CREDIT_CARD") {
			t.Errorf("Expected detail to contain 'CREDIT_CARD', got %s", call.detail)
		}
	case <-time.After(2 * time.Second):
		t.Error("Approval should have been requested for credit card in Input.insertText")
	}
}

func TestApproval_PageNavigateNewDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy(wsURL, router, nil, false)

	mockApproval := newMockApprovalClient(true)
	proxy.SetApprovalClient(mockApproval)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://evil-phishing.com/login"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	select {
	case call := <-mockApproval.called:
		if call.opType != approval.OpNavigation {
			t.Errorf("Expected opType %s, got %s", approval.OpNavigation, call.opType)
		}
		if !strings.Contains(call.detail, "evil-phishing.com") {
			t.Errorf("Expected detail to contain URL, got %s", call.detail)
		}
	case <-time.After(2 * time.Second):
		t.Error("Approval should have been requested for Page.navigate")
	}
}

func TestApproval_NonSensitiveMethodsPassthrough(t *testing.T) {
	nonSensitive := []struct {
		method string
		params string
	}{
		{"Runtime.evaluate", `{"expression":"1+1"}`},
		{"DOM.querySelector", `{"selector":"div"}`},
		{"Network.enable", `{}`},
		{"Input.insertText", `{"text":"hello world"}`},
	}

	for _, tc := range nonSensitive {
		t.Run(tc.method, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(testHandler))
			defer server.Close()
			wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

			router := NewMethodRouter(NewTranslator())
			proxy := NewProxy(wsURL, router, nil, false)

			mockApproval := newMockApprovalClient(true)
			proxy.SetApprovalClient(mockApproval)

			clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err != nil {
				t.Fatalf("Failed to dial websocket: %v", err)
			}

			if err := proxy.Start(clientConn); err != nil {
				clientConn.Close()
				t.Fatalf("Failed to start proxy: %v", err)
			}

			msg := CDPMessage{
				ID:     1,
				Method: tc.method,
				Params: json.RawMessage(tc.params),
			}
			data, _ := json.Marshal(msg)
			if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
				proxy.Stop()
				t.Fatalf("Failed to write message for %s: %v", tc.method, err)
			}

			select {
			case <-mockApproval.called:
				proxy.Stop()
				t.Errorf("Approval should NOT be requested for non-sensitive method %s", tc.method)
			case <-time.After(100 * time.Millisecond):
			}

			proxy.Stop()
		})
	}
}

func TestApproval_DenialBlocksForwarding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy(wsURL, router, nil, false)

	// Approval denied
	mockApproval := newMockApprovalClient(false)
	proxy.SetApprovalClient(mockApproval)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     42,
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://evil.com"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Wait for approval to be called
	select {
	case <-mockApproval.called:
		// Good - approval was requested
	case <-time.After(2 * time.Second):
		t.Fatal("Approval should have been requested")
	}

	// Read response from client conn - should be error response
	clientConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, respData, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("Expected error response on client conn: %v", err)
	}

	var resp CDPMessage
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.ID != 42 {
		t.Errorf("Expected response ID 42, got %d", resp.ID)
	}
	if resp.Error == nil {
		t.Error("Expected error in response when approval denied")
	}
	if resp.Error.Code != -32000 {
		t.Errorf("Expected error code -32000, got %d", resp.Error.Code)
	}
}

func TestApproval_TimeoutReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy(wsURL, router, nil, false)

	// Mock that never responds (simulates timeout)
	timeoutMock := &timeoutApprovalMock{
		called: make(chan approvalCall, 10),
	}
	proxy.SetApprovalClient(timeoutMock)

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	// Set very short approval timeout for test
	proxy.SetApprovalTimeout(200 * time.Millisecond)

	msg := CDPMessage{
		ID:     99,
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://evil.com"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Wait for approval to be called
	select {
	case <-timeoutMock.called:
		// Good
	case <-time.After(2 * time.Second):
		t.Fatal("Approval should have been requested")
	}

	// Should get error response after timeout
	clientConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, respData, err := clientConn.ReadMessage()
	if err != nil {
		t.Fatalf("Expected error response after timeout: %v", err)
	}

	var resp CDPMessage
	if err := json.Unmarshal(respData, &resp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if resp.ID != 99 {
		t.Errorf("Expected response ID 99, got %d", resp.ID)
	}
	if resp.Error == nil {
		t.Error("Expected error in response when approval times out")
	}
	if !strings.Contains(resp.Error.Message, "timed out") && !strings.Contains(resp.Error.Message, "denied") {
		t.Errorf("Expected timeout/denied error message, got: %s", resp.Error.Message)
	}
}

// timeoutApprovalMock never responds - simulates timeout
type timeoutApprovalMock struct {
	called chan approvalCall
}

func (m *timeoutApprovalMock) RequestApproval(ctx context.Context, op approval.OperationType, detail string) (bool, error) {
	m.called <- approvalCall{opType: op, detail: detail}
	// Block until context cancelled (simulates timeout)
	<-ctx.Done()
	return false, ctx.Err()
}

func TestApproval_NilClientPassthrough(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(testHandler))
	defer server.Close()
	wsURL := strings.Replace(server.URL, "http://", "ws://", 1)

	router := NewMethodRouter(NewTranslator())
	proxy := NewProxy(wsURL, router, nil, false)
	// No approval client set - should passthrough everything

	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to dial websocket: %v", err)
	}
	defer clientConn.Close()

	if err := proxy.Start(clientConn); err != nil {
		t.Fatalf("Failed to start proxy: %v", err)
	}
	defer proxy.Stop()

	msg := CDPMessage{
		ID:     1,
		Method: "Page.navigate",
		Params: json.RawMessage(`{"url":"https://example.com"}`),
	}
	data, _ := json.Marshal(msg)
	if err := clientConn.WriteMessage(websocket.TextMessage, data); err != nil {
		t.Fatalf("Failed to write message: %v", err)
	}

	// Should NOT get an error - message should passthrough
	select {
	case err := <-proxy.Errors():
		if err != nil {
			t.Errorf("Expected no proxy errors with nil approval client: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		// Good - no errors, message forwarded
	}
}

func BenchmarkMatchWildcard(b *testing.B) {
	tests := []struct {
		pattern string
		method  string
	}{
		{"Target.*", "Target.createTarget"},
		{"Target.*", "Target.closeTarget"},
		{"Target.*", "Target.attachToTarget"},
		{"Target.*", "Input.dispatchMouseEvent"},
		{"Input.dispatchMouseEvent", "Input.dispatchMouseEvent"},
		{"Input.dispatchMouseEvent", "Input.dispatchKeyEvent"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tt := range tests {
			matchWildcard(tt.pattern, tt.method)
		}
	}
}

func BenchmarkMethodRouter_Route(b *testing.B) {
	router := NewMethodRouter(NewTranslator())
	methods := []string{
		"Target.createTarget",
		"Target.closeTarget",
		"Input.dispatchMouseEvent",
		"Runtime.evaluate",
		"Page.navigate",
		"Network.enable",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, method := range methods {
			router.Route(method)
		}
	}
}
