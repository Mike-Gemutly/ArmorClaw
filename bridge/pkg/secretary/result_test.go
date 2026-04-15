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
