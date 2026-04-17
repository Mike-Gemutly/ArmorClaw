package secretary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestContainerStepResultHappyPath(t *testing.T) {
	dir := t.TempDir()

	payload := ContainerStepResult{
		Status:     "success",
		Output:     "Booked flight BA123",
		Data:       map[string]any{"booking_ref": "ABC123", "price": 299.99},
		DurationMS: 5432,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "result.json"), raw, 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	got, err := ParseContainerStepResult(dir)
	if err != nil {
		t.Fatalf("ParseContainerStepResult returned error: %v", err)
	}
	if got == nil {
		t.Fatal("ParseContainerStepResult returned nil result")
	}

	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	if got.Output != "Booked flight BA123" {
		t.Errorf("Output = %q, want %q", got.Output, "Booked flight BA123")
	}
	if got.DurationMS != 5432 {
		t.Errorf("DurationMS = %d, want %d", got.DurationMS, 5432)
	}
	if ref, ok := got.Data["booking_ref"].(string); !ok || ref != "ABC123" {
		t.Errorf("Data[booking_ref] = %v, want %q", got.Data["booking_ref"], "ABC123")
	}
	if price, ok := got.Data["price"].(float64); !ok || price != 299.99 {
		t.Errorf("Data[price] = %v, want %v", got.Data["price"], 299.99)
	}
}

func TestContainerStepResultFileMissing(t *testing.T) {
	dir := t.TempDir()

	got, err := ParseContainerStepResult(dir)
	if err != nil {
		t.Fatalf("ParseContainerStepResult returned error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil result for missing file, got %+v", got)
	}
}

func TestContainerStepResultMalformedJSON(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte("{bad json"), 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	got, err := ParseContainerStepResult(dir)
	if err == nil {
		t.Fatalf("expected error for malformed JSON, got nil (result=%+v)", got)
	}
	if got != nil {
		t.Errorf("expected nil result for malformed JSON, got %+v", got)
	}
}

func TestContainerStepResultExtraFields(t *testing.T) {
	dir := t.TempDir()

	extra := `{
		"status": "success",
		"output": "done",
		"duration_ms": 100,
		"future_field": "some value",
		"nested": {"a": 1}
	}`
	if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte(extra), 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	got, err := ParseContainerStepResult(dir)
	if err != nil {
		t.Fatalf("ParseContainerStepResult returned error: %v", err)
	}
	if got == nil {
		t.Fatal("ParseContainerStepResult returned nil result")
	}

	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	if got.Output != "done" {
		t.Errorf("Output = %q, want %q", got.Output, "done")
	}
	if got.DurationMS != 100 {
		t.Errorf("DurationMS = %d, want %d", got.DurationMS, 100)
	}
}

func TestParseExtendedWithAllFields(t *testing.T) {
	dir := t.TempDir()

	payload := `{
		"status": "partial",
		"output": "completed 2 of 3 steps",
		"duration_ms": 12000,
		"data": {"items_done": 2, "items_total": 3},
		"_comments": ["Step 3 skipped due to rate limit", "Retry recommended"],
		"_blockers": [
			{"blocker_type": "auth", "message": "Captcha detected", "suggestion": "Switch to API mode", "field": "login_form"}
		],
		"_skill_candidates": [
			{"name": "hotel_search", "description": "Repeat hotel search pattern", "pattern_type": "dom_sequence", "pattern_data": "form.hotel-search", "confidence": 0.89}
		],
		"_events_summary": {"total": 42, "types": {"click": 20, "type": 15, "nav": 7}}
	}`
	if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	eventsJSONL := `{"seq":1,"type":"nav","name":"open_url","ts_ms":1000}
{"seq":2,"type":"click","name":"submit_btn","ts_ms":2500,"detail":{"selector":"#submit"},"duration_ms":300}
{"seq":3,"type":"type","name":"fill_email","ts_ms":3000,"detail":{"field":"email","length":24}}
`
	if err := os.WriteFile(filepath.Join(dir, "_events.jsonl"), []byte(eventsJSONL), 0o644); err != nil {
		t.Fatalf("write _events.jsonl: %v", err)
	}

	got, err := ParseExtendedStepResult(dir)
	if err != nil {
		t.Fatalf("ParseExtendedStepResult error: %v", err)
	}
	if got == nil {
		t.Fatal("ParseExtendedStepResult returned nil")
	}

	if got.Status != "partial" {
		t.Errorf("Status = %q, want %q", got.Status, "partial")
	}
	if got.Output != "completed 2 of 3 steps" {
		t.Errorf("Output = %q, want %q", got.Output, "completed 2 of 3 steps")
	}
	if got.DurationMS != 12000 {
		t.Errorf("DurationMS = %d, want %d", got.DurationMS, 12000)
	}

	if len(got.Comments) != 2 {
		t.Fatalf("Comments len = %d, want 2", len(got.Comments))
	}
	if got.Comments[0] != "Step 3 skipped due to rate limit" {
		t.Errorf("Comments[0] = %q", got.Comments[0])
	}

	if len(got.Blockers) != 1 {
		t.Fatalf("Blockers len = %d, want 1", len(got.Blockers))
	}
	if got.Blockers[0].BlockerType != "auth" {
		t.Errorf("Blocker.BlockerType = %q, want %q", got.Blockers[0].BlockerType, "auth")
	}
	if got.Blockers[0].Field != "login_form" {
		t.Errorf("Blocker.Field = %q, want %q", got.Blockers[0].Field, "login_form")
	}

	if len(got.SkillCandidates) != 1 {
		t.Fatalf("SkillCandidates len = %d, want 1", len(got.SkillCandidates))
	}
	if got.SkillCandidates[0].Confidence != 0.89 {
		t.Errorf("SkillCandidate.Confidence = %f, want 0.89", got.SkillCandidates[0].Confidence)
	}

	if got.EventsSummary == nil {
		t.Fatal("EventsSummary is nil")
	}
	if got.EventsSummary.Total != 42 {
		t.Errorf("EventsSummary.Total = %d, want 42", got.EventsSummary.Total)
	}
	if got.EventsSummary.Types["click"] != 20 {
		t.Errorf("EventsSummary.Types[click] = %d, want 20", got.EventsSummary.Types["click"])
	}

	if len(got.Events) != 3 {
		t.Fatalf("Events len = %d, want 3", len(got.Events))
	}
	if got.Events[0].Seq != 1 || got.Events[0].Type != "nav" {
		t.Errorf("Events[0] = %+v", got.Events[0])
	}
	if got.Events[1].DurationMs == nil || *got.Events[1].DurationMs != 300 {
		t.Errorf("Events[1].DurationMs = %v, want 300", got.Events[1].DurationMs)
	}
}

func TestParseExtendedBackwardCompatible(t *testing.T) {
	dir := t.TempDir()

	oldFormat := `{"status":"success","output":"done","duration_ms":500}`
	if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte(oldFormat), 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	got, err := ParseExtendedStepResult(dir)
	if err != nil {
		t.Fatalf("ParseExtendedStepResult error: %v", err)
	}
	if got == nil {
		t.Fatal("ParseExtendedStepResult returned nil")
	}

	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	if got.Output != "done" {
		t.Errorf("Output = %q, want %q", got.Output, "done")
	}
	if got.DurationMS != 500 {
		t.Errorf("DurationMS = %d, want %d", got.DurationMS, 500)
	}
	if len(got.Comments) != 0 {
		t.Errorf("Comments len = %d, want 0", len(got.Comments))
	}
	if len(got.Blockers) != 0 {
		t.Errorf("Blockers len = %d, want 0", len(got.Blockers))
	}
	if got.EventsSummary != nil {
		t.Errorf("EventsSummary = %+v, want nil", got.EventsSummary)
	}
	if got.Events != nil {
		t.Errorf("Events = %+v, want nil", got.Events)
	}
}

func TestParseExtendedMissingEventsFile(t *testing.T) {
	dir := t.TempDir()

	payload := `{"status":"success","output":"ok","duration_ms":100,"_comments":["note"]}`
	if err := os.WriteFile(filepath.Join(dir, "result.json"), []byte(payload), 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	got, err := ParseExtendedStepResult(dir)
	if err != nil {
		t.Fatalf("ParseExtendedStepResult error: %v", err)
	}
	if got == nil {
		t.Fatal("ParseExtendedStepResult returned nil")
	}

	if got.Events != nil {
		t.Errorf("Events = %+v, want nil (no _events.jsonl)", got.Events)
	}
	if len(got.Comments) != 1 || got.Comments[0] != "note" {
		t.Errorf("Comments = %v, want [note]", got.Comments)
	}
}

func TestReadEventsFileWithValidEvents(t *testing.T) {
	dir := t.TempDir()

	eventsJSONL := `{"seq":1,"type":"click","name":"btn","ts_ms":1000}
{"seq":2,"type":"type","name":"input","ts_ms":2000,"detail":{"key":"Enter"}}
{"seq":3,"type":"nav","name":"page_load","ts_ms":3000,"duration_ms":150}
`
	if err := os.WriteFile(filepath.Join(dir, "_events.jsonl"), []byte(eventsJSONL), 0o644); err != nil {
		t.Fatalf("write _events.jsonl: %v", err)
	}

	events, err := ReadEventsFile(dir)
	if err != nil {
		t.Fatalf("ReadEventsFile error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("events len = %d, want 3", len(events))
	}

	if events[0].Seq != 1 || events[0].Type != "click" || events[0].Name != "btn" {
		t.Errorf("events[0] = %+v", events[0])
	}
	if events[1].Type != "type" {
		t.Errorf("events[1].Type = %q, want %q", events[1].Type, "type")
	}
	if events[1].Detail["key"] != "Enter" {
		t.Errorf("events[1].Detail[key] = %v, want Enter", events[1].Detail["key"])
	}
	if events[2].DurationMs == nil || *events[2].DurationMs != 150 {
		t.Errorf("events[2].DurationMs = %v, want 150", events[2].DurationMs)
	}
}
