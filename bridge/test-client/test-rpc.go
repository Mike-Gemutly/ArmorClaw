// Simple test client for ArmorClaw bridge RPC server
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
)

// JSONRPC 2.0 request/response structures
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *ErrorObj   `json:"error,omitempty"`
}

type ErrorObj struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func main() {
	socketPath := "/run/armorclaw/bridge.sock"

	// Check if socket exists
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		fmt.Printf("Socket not found at %s\n", socketPath)
		fmt.Println("Start the bridge server first:")
		fmt.Println("  sudo ./build/armorclaw-bridge")
		os.Exit(1)
	}

	// Connect to socket
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		fmt.Printf("Failed to connect to socket: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Test 1: Status check
	fmt.Println("=== Test 1: Status ===")
	if err := sendRequest(conn, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "status",
	}, nil); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Test 2: Health check
	fmt.Println("\n=== Test 2: Health ===")
	if err := sendRequest(conn, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "health",
	}, nil); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Test 3: Matrix status
	fmt.Println("\n=== Test 3: Matrix Status ===")
	if err := sendRequest(conn, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "matrix.status",
	}, nil); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Test 4: List keys (should be empty)
	fmt.Println("\n=== Test 4: List Keys ===")
	if err := sendRequest(conn, map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      4,
		"method":  "list_keys",
		"params": map[string]string{},
	}, nil); err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	fmt.Println("\n=== All tests completed ===")
}

func sendRequest(conn net.Conn, req map[string]interface{}, params json.RawMessage) error {
	// Marshal request
	var body []byte
	var err error

	if params != nil {
		req["params"] = params
	}

	body, err = json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Send request
	if _, err := conn.Write(body); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var resp Response
	decoder := json.NewDecoder(bytes.NewReader(buf[:n]))
	if err := decoder.Decode(&resp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// Print response
	prettyJSON, _ := json.MarshalIndent(resp.Result, "", "  ")
	if resp.Error != nil {
		fmt.Printf("Error: %s\n", resp.Error.Message)
	} else {
		fmt.Printf("Response: %s\n", string(prettyJSON))
	}

	return nil
}
