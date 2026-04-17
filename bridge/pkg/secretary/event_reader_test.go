package secretary

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeEvents(t *testing.T, dir string, events []StepEvent, append bool) {
	t.Helper()
	flag := os.O_CREATE | os.O_WRONLY
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	f, err := os.OpenFile(filepath.Join(dir, "_events.jsonl"), flag, 0644)
	if err != nil {
		t.Fatalf("open events file: %v", err)
	}
	defer f.Close()
	for _, e := range events {
		data, _ := json.Marshal(e)
		f.Write(data)
		f.Write([]byte("\n"))
	}
}

func writeRawLines(t *testing.T, dir string, lines []string, append bool) {
	t.Helper()
	flag := os.O_CREATE | os.O_WRONLY
	if append {
		flag |= os.O_APPEND
	} else {
		flag |= os.O_TRUNC
	}
	f, err := os.OpenFile(filepath.Join(dir, "_events.jsonl"), flag, 0644)
	if err != nil {
		t.Fatalf("open events file: %v", err)
	}
	defer f.Close()
	for _, l := range lines {
		f.Write([]byte(l))
		f.Write([]byte("\n"))
	}
}

func evt(seq int, typ string, name string) StepEvent {
	return StepEvent{Seq: seq, Type: typ, Name: name, TsMs: int64(seq) * 1000}
}

func TestEventReader_IncrementalReadsAcrossPollCycles(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	// Poll 1: no file yet
	evts, _, err := r.ReadNew()
	if err != nil {
		t.Fatalf("poll 1: %v", err)
	}
	if len(evts) != 0 {
		t.Fatalf("poll 1: expected 0 events, got %d", len(evts))
	}

	// Poll 2: write 2 events
	writeEvents(t, dir, []StepEvent{evt(1, "start", "step1"), evt(2, "progress", "step1")}, false)
	evts, _, err = r.ReadNew()
	if err != nil {
		t.Fatalf("poll 2: %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("poll 2: expected 2 events, got %d", len(evts))
	}
	if evts[0].Seq != 1 || evts[1].Seq != 2 {
		t.Fatalf("poll 2: unexpected seq values: %d, %d", evts[0].Seq, evts[1].Seq)
	}

	// Poll 3: write 3 more events
	writeEvents(t, dir, []StepEvent{evt(3, "progress", "step1"), evt(4, "progress", "step1"), evt(5, "end", "step1")}, true)
	evts, _, err = r.ReadNew()
	if err != nil {
		t.Fatalf("poll 3: %v", err)
	}
	if len(evts) != 3 {
		t.Fatalf("poll 3: expected 3 events, got %d", len(evts))
	}
	if evts[0].Seq != 3 || evts[2].Seq != 5 {
		t.Fatalf("poll 3: unexpected seq values: %d, %d", evts[0].Seq, evts[2].Seq)
	}

	// Poll 4: nothing new
	evts, _, err = r.ReadNew()
	if err != nil {
		t.Fatalf("poll 4: %v", err)
	}
	if len(evts) != 0 {
		t.Fatalf("poll 4: expected 0 events, got %d", len(evts))
	}

	// Poll 5: one more event
	writeEvents(t, dir, []StepEvent{evt(6, "done", "step1")}, true)
	evts, _, err = r.ReadNew()
	if err != nil {
		t.Fatalf("poll 5: %v", err)
	}
	if len(evts) != 1 || evts[0].Seq != 6 {
		t.Fatalf("poll 5: expected 1 event with seq 6, got %d events", len(evts))
	}
}

func TestEventReader_Exactly10MB_Succeeds(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	// Build lines that total exactly 10 MB.
	targetSize := int64(10 * 1024 * 1024)
	rawPath := filepath.Join(dir, "_events.jsonl")
	f, err := os.Create(rawPath)
	if err != nil {
		t.Fatal(err)
	}

	written := int64(0)
	seq := 1
	for written < targetSize {
		evt := StepEvent{Seq: seq, Type: "filler", Name: "pad", TsMs: int64(seq) * 1000}
		data, _ := json.Marshal(evt)
		line := append(data, '\n')
		if written+int64(len(line)) > targetSize {
			// Pad the detail field to hit exactly 10 MB.
			remaining := targetSize - written - int64(len(line))
			padding := int(remaining)
			if padding > 0 {
				pad := make([]byte, padding)
				for i := range pad {
					pad[i] = 'x'
				}
				evt.Detail = map[string]interface{}{"p": string(pad)}
				data, _ = json.Marshal(evt)
				line = append(data, '\n')
			}
		}
		n, _ := f.Write(line)
		written += int64(n)
		seq++
	}
	f.Close()

	info, _ := os.Stat(rawPath)
	if info.Size() != targetSize {
		f2, _ := os.OpenFile(rawPath, os.O_WRONLY, 0644)
		f2.Truncate(targetSize)
		f2.Close()
	}

	evts, _, err := r.ReadNew()
	if err != nil {
		t.Fatalf("exactly 10MB should succeed, got: %v", err)
	}
	if len(evts) == 0 {
		t.Fatal("should parse at least one event")
	}
}

func TestEventReader_Over10MB_Fails(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	// Create a file that is 10 MB + 1 byte.
	f, err := os.Create(filepath.Join(dir, "_events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	f.Truncate(10*1024*1024 + 1)
	f.Close()

	evts, size, err := r.ReadNew()
	if !errors.Is(err, ErrEventLogExceeded) {
		t.Fatalf("expected ErrEventLogExceeded, got: %v", err)
	}
	if evts != nil {
		t.Fatal("expected nil events")
	}
	if size != 10*1024*1024+1 {
		t.Fatalf("expected size 10MB+1, got %d", size)
	}
}

func TestEventReader_MalformedJSONSkipping(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	lines := []string{
		`{"seq":1,"type":"start","name":"ok","ts_ms":1000}`,
		`{bad json here`,
		`also not json`,
		`{"seq":2,"type":"end","name":"ok","ts_ms":2000}`,
	}
	writeRawLines(t, dir, lines, false)

	evts, _, err := r.ReadNew()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 valid events, got %d", len(evts))
	}
	if evts[0].Seq != 1 || evts[1].Seq != 2 {
		t.Fatalf("expected seq 1 and 2, got %d and %d", evts[0].Seq, evts[1].Seq)
	}
}

func TestEventReader_DuplicateSeqFiltering(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	// First write: seq 1, 2
	writeEvents(t, dir, []StepEvent{evt(1, "a", "x"), evt(2, "b", "x")}, false)
	evts, _, _ := r.ReadNew()
	if len(evts) != 2 {
		t.Fatalf("first read: expected 2, got %d", len(evts))
	}

	// Second write: seq 2 again (duplicate) + seq 3 (new)
	writeEvents(t, dir, []StepEvent{evt(2, "b", "x"), evt(3, "c", "x")}, true)
	evts, _, _ = r.ReadNew()
	if len(evts) != 1 {
		t.Fatalf("second read: expected 1 (dup filtered), got %d", len(evts))
	}
	if evts[0].Seq != 3 {
		t.Fatalf("expected seq 3, got %d", evts[0].Seq)
	}
}

func TestEventReader_MissingFileReturnsNil(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	evts, size, err := r.ReadNew()
	if err != nil {
		t.Fatalf("expected no error for missing file, got: %v", err)
	}
	if evts != nil {
		t.Fatal("expected nil events")
	}
	if size != 0 {
		t.Fatalf("expected size 0, got %d", size)
	}
}

func TestEventReader_CommentAndBlankLineSkipping(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	lines := []string{
		`# this is a comment`,
		``,
		`   `,
		`{"seq":1,"type":"start","name":"ok","ts_ms":1000}`,
		`# another comment`,
		`{"seq":2,"type":"end","name":"ok","ts_ms":2000}`,
		``,
	}
	writeRawLines(t, dir, lines, false)

	evts, _, err := r.ReadNew()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("expected 2 events, got %d", len(evts))
	}
	if evts[0].Seq != 1 || evts[1].Seq != 2 {
		t.Fatalf("expected seq 1 and 2, got %d and %d", evts[0].Seq, evts[1].Seq)
	}
}

func TestEventReader_EmptyFileHandling(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	// Create empty file
	f, err := os.Create(filepath.Join(dir, "_events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	evts, size, err := r.ReadNew()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(evts) != 0 {
		t.Fatalf("expected 0 events from empty file, got %d", len(evts))
	}
	if size != 0 {
		t.Fatalf("expected size 0, got %d", size)
	}
}

func TestEventReader_OffsetTrackingAcrossCalls(t *testing.T) {
	dir := t.TempDir()
	r := NewEventReader(dir)

	// Write 3 events, read them
	writeEvents(t, dir, []StepEvent{evt(1, "a", "x"), evt(2, "b", "x"), evt(3, "c", "x")}, false)
	evts, size1, _ := r.ReadNew()
	if len(evts) != 3 {
		t.Fatalf("first read: expected 3, got %d", len(evts))
	}

	// Write 2 more, read only new ones
	writeEvents(t, dir, []StepEvent{evt(4, "d", "x"), evt(5, "e", "x")}, true)
	evts, size2, _ := r.ReadNew()
	if len(evts) != 2 {
		t.Fatalf("second read: expected 2, got %d", len(evts))
	}
	if evts[0].Seq != 4 || evts[1].Seq != 5 {
		t.Fatalf("expected seq 4,5 got %d,%d", evts[0].Seq, evts[1].Seq)
	}
	if size2 <= size1 {
		t.Fatalf("file should have grown: before=%d after=%d", size1, size2)
	}

	// Third read: nothing new
	evts, size3, _ := r.ReadNew()
	if len(evts) != 0 {
		t.Fatalf("third read: expected 0, got %d", len(evts))
	}
	if size3 != size2 {
		t.Fatalf("file size should be stable: %d vs %d", size2, size3)
	}
}
