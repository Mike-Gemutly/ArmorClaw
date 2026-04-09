package cdp

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

type FallbackHandler struct {
	mu               sync.Mutex
	dummySuccess     bool
	lastLoggedMethod string
}

func NewFallbackHandler() *FallbackHandler {
	return &FallbackHandler{
		dummySuccess: false,
	}
}

func (fh *FallbackHandler) EnableDummySuccess() {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.dummySuccess = true
}

func (fh *FallbackHandler) DisableDummySuccess() {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fh.dummySuccess = false
}

func (fh *FallbackHandler) HandleUnsupported(msg *CDPMessage) (*CDPMessage, error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()

	fh.lastLoggedMethod = msg.Method

	if fh.dummySuccess {
		result := map[string]any{
			"success": true,
		}
		resultJSON, _ := json.Marshal(result)
		return &CDPMessage{
			ID:     msg.ID,
			Result: resultJSON,
		}, nil
	}

	log.Printf("[JETSKI FALLBACK]: Unsupported method %s (ID: %d) - log and drop", msg.Method, msg.ID)

	return &CDPMessage{
		ID: msg.ID,
		Error: &CDPError{
			Code:    -32601,
			Message: fmt.Sprintf("Method not supported: %s", msg.Method),
		},
	}, nil
}

func (fh *FallbackHandler) GetLastLoggedMethod() string {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.lastLoggedMethod
}
