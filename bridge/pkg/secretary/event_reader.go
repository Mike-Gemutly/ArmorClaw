package secretary

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const maxEventLogSize = 10 * 1024 * 1024 // 10 MB

var ErrEventLogExceeded = errors.New("event log exceeded 10MB cap")

// EventReader incrementally reads new events from a state directory's
// _events.jsonl file.  Each call to ReadNew returns only lines that have
// been appended since the previous call, tracked via byte offset and
// sequence number.
type EventReader struct {
	// stateDir is the directory that contains the _events.jsonl file.
	stateDir string

	// byteOffset is the file offset to seek to on the next ReadNew call.
	byteOffset int64

	// lastSeq is the highest sequence number already returned to the caller.
	// Lines with seq <= lastSeq are silently skipped (deduplication).
	lastSeq int
}

// NewEventReader creates an EventReader that tails <stateDir>/_events.jsonl.
func NewEventReader(stateDir string) *EventReader {
	return &EventReader{
		stateDir:   stateDir,
		byteOffset: 0,
		lastSeq:    0,
	}
}

// ReadNew reads all events appended to _events.jsonl since the last call.
//
// Returns:
//   - events: slice of newly-parsed StepEvent values (may be empty/nil)
//   - fileSize: the current size of _events.jsonl in bytes
//   - error: non-nil on I/O error or if the file exceeds 10 MB
//
// If the file does not exist, returns (nil, 0, nil) so callers can poll
// without special-casing the "not started yet" state.
func (r *EventReader) ReadNew() ([]StepEvent, int64, error) {
	path := filepath.Join(r.stateDir, "_events.jsonl")

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("stat event log: %w", err)
	}

	fileSize := info.Size()

	if fileSize > maxEventLogSize {
		return nil, fileSize, ErrEventLogExceeded
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, fileSize, fmt.Errorf("open event log: %w", err)
	}
	defer f.Close()

	if _, err := f.Seek(r.byteOffset, io.SeekStart); err != nil {
		return nil, fileSize, fmt.Errorf("seek event log: %w", err)
	}

	var events []StepEvent
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxEventLogSize+1)

	for scanner.Scan() {
		line := scanner.Text()

		r.byteOffset += int64(len(line)) + 1

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		var evt StepEvent
		if err := json.Unmarshal([]byte(trimmed), &evt); err != nil {
			log.Printf("event_reader: skipping malformed JSON line: %q: %v", trimmed, err)
			continue
		}

		if evt.Seq <= r.lastSeq {
			continue
		}

		events = append(events, evt)

		if evt.Seq > r.lastSeq {
			r.lastSeq = evt.Seq
		}
	}

	if err := scanner.Err(); err != nil {
		return events, fileSize, fmt.Errorf("scan event log: %w", err)
	}

	return events, fileSize, nil
}
