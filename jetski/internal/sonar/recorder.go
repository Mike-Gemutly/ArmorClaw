package sonar

import (
	"encoding/json"
	"time"
)

func RecordFrame(buf *CircularBuffer, method string, params json.RawMessage, sessionID string) {
	if buf == nil {
		return
	}

	buf.Push(CDPFrame{
		Timestamp: time.Now(),
		Method:    method,
		Params:    params,
		SessionID: sessionID,
	})
}
