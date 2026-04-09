package cdp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMethodRouter_Route(t *testing.T) {
	router := NewMethodRouter()

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
	router := NewMethodRouter()

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
	router := NewMethodRouter()

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
	router := NewMethodRouter()
	proxy := NewProxy("ws://localhost:9222", router)

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

	router := NewMethodRouter()
	proxy := NewProxy(wsURL, router)

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

	router := NewMethodRouter()
	proxy := NewProxy(wsURL, router)

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
	router := NewMethodRouter()
	proxy := NewProxy("ws://localhost:9222", router)

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
	router := NewMethodRouter()
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
