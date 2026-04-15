package secretary

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBackwardChannelResultRead(t *testing.T) {
	stateDir := t.TempDir()

	containerPayload := ContainerStepResult{
		Status:     "success",
		Output:     "Researched 3 restaurants in Manhattan",
		Data:       map[string]any{"venues": 3, "top_pick": "Le Bernardin"},
		DurationMS: 12450,
	}
	raw, err := json.Marshal(containerPayload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "result.json"), raw, 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	got, err := ParseContainerStepResult(stateDir)
	if err != nil {
		t.Fatalf("ParseContainerStepResult error: %v", err)
	}
	if got == nil {
		t.Fatal("got nil result")
	}

	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	if got.Output != containerPayload.Output {
		t.Errorf("Output = %q, want %q", got.Output, containerPayload.Output)
	}
	if got.DurationMS != containerPayload.DurationMS {
		t.Errorf("DurationMS = %d, want %d", got.DurationMS, containerPayload.DurationMS)
	}
	if venues, ok := got.Data["venues"].(float64); !ok || int(venues) != 3 {
		t.Errorf("Data[venues] = %v, want 3", got.Data["venues"])
	}
	if pick, ok := got.Data["top_pick"].(string); !ok || pick != "Le Bernardin" {
		t.Errorf("Data[top_pick] = %v, want %q", got.Data["top_pick"], "Le Bernardin")
	}
}

func TestBackwardChannelNoResult(t *testing.T) {
	stateDir := t.TempDir()

	got, err := ParseContainerStepResult(stateDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result for empty state dir, got %+v", got)
	}
}

func TestBackwardChannelMalformedResult(t *testing.T) {
	stateDir := t.TempDir()

	badJSON := []byte(`{"status":"success","output": BREAK`)
	if err := os.WriteFile(filepath.Join(stateDir, "result.json"), badJSON, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ParseContainerStepResult(stateDir)
	if err == nil {
		t.Fatalf("expected error for malformed JSON, got result=%+v", got)
	}
	if got != nil {
		t.Errorf("expected nil result alongside error, got %+v", got)
	}
}

func TestBackwardChannelContainerWritesResult(t *testing.T) {
	stateDir := t.TempDir()

	realisticPayload := map[string]any{
		"status":      "success",
		"output":      "Filled checkout form at shop.example.com. Order placed for 2 items totaling $47.98.",
		"error":       "",
		"duration_ms": json.Number("8712"),
		"data": map[string]any{
			"url":         "https://shop.example.com/checkout",
			"items":       2,
			"total_cents": 4798,
			"order_id":    "ORD-2026-0414-7721",
		},
	}
	raw, err := json.MarshalIndent(realisticPayload, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "result.json"), raw, 0o644); err != nil {
		t.Fatalf("write result.json: %v", err)
	}

	got, err := ParseContainerStepResult(stateDir)
	if err != nil {
		t.Fatalf("ParseContainerStepResult error: %v", err)
	}
	if got == nil {
		t.Fatal("got nil result")
	}

	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	if got.Output == "" {
		t.Error("Output is empty, expected realistic output text")
	}
	if got.DurationMS != 8712 {
		t.Errorf("DurationMS = %d, want 8712", got.DurationMS)
	}
	if got.Data == nil {
		t.Fatal("Data is nil")
	}
	if orderID, ok := got.Data["order_id"].(string); !ok || orderID != "ORD-2026-0414-7721" {
		t.Errorf("Data[order_id] = %v, want %q", got.Data["order_id"], "ORD-2026-0414-7721")
	}
	if items, ok := got.Data["items"].(float64); !ok || int(items) != 2 {
		t.Errorf("Data[items] = %v, want 2", got.Data["items"])
	}
	if totalCents, ok := got.Data["total_cents"].(float64); !ok || int(totalCents) != 4798 {
		t.Errorf("Data[total_cents] = %v, want 4798", got.Data["total_cents"])
	}
}

func TestBackwardChannelMinimalPayload(t *testing.T) {
	stateDir := t.TempDir()

	minimal := `{"status":"success","output":"done"}`
	if err := os.WriteFile(filepath.Join(stateDir, "result.json"), []byte(minimal), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ParseContainerStepResult(stateDir)
	if err != nil {
		t.Fatalf("ParseContainerStepResult error: %v", err)
	}
	if got == nil {
		t.Fatal("got nil result")
	}
	if got.Status != "success" {
		t.Errorf("Status = %q, want %q", got.Status, "success")
	}
	if got.Output != "done" {
		t.Errorf("Output = %q, want %q", got.Output, "done")
	}
	if got.Data != nil {
		t.Errorf("Data = %v, want nil for omitted data field", got.Data)
	}
	if got.DurationMS != 0 {
		t.Errorf("DurationMS = %d, want 0 for omitted field", got.DurationMS)
	}
	if got.Error != "" {
		t.Errorf("Error = %q, want empty for omitted field", got.Error)
	}
}

func TestBackwardChannelUIDPermission(t *testing.T) {
	stateDir := t.TempDir()

	payload := ContainerStepResult{
		Status:     "success",
		Output:     "Task completed",
		DurationMS: 1500,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	resultPath := filepath.Join(stateDir, "result.json")

	for _, perm := range []os.FileMode{0o644, 0o640, 0o600} {
		if err := os.WriteFile(resultPath, raw, perm); err != nil {
			t.Fatalf("write with perm %04o: %v", perm, err)
		}

		info, err := os.Stat(resultPath)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}

		got, err := ParseContainerStepResult(stateDir)
		if err != nil {
			t.Fatalf("ParseContainerStepResult with perm %04o: %v", info.Mode().Perm(), err)
		}
		if got == nil {
			t.Fatalf("nil result with perm %04o", info.Mode().Perm())
		}
		if got.Status != "success" {
			t.Errorf("Status with perm %04o = %q, want %q", info.Mode().Perm(), got.Status, "success")
		}

		os.Remove(resultPath)
	}
}

func TestBackwardChannelFailedStep(t *testing.T) {
	stateDir := t.TempDir()

	failedPayload := ContainerStepResult{
		Status:     "failed",
		Output:     "Partial progress: filled 2 of 3 fields",
		Error:      "timeout waiting for page load after 30s",
		DurationMS: 30123,
		Data: map[string]any{
			"fields_filled": 2,
			"fields_total":  3,
			"last_url":      "https://shop.example.com/payment",
		},
	}
	raw, err := json.Marshal(failedPayload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "result.json"), raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ParseContainerStepResult(stateDir)
	if err != nil {
		t.Fatalf("ParseContainerStepResult error: %v", err)
	}
	if got == nil {
		t.Fatal("got nil result")
	}
	if got.Status != "failed" {
		t.Errorf("Status = %q, want %q", got.Status, "failed")
	}
	if got.Error == "" {
		t.Error("Error is empty, expected failure description")
	}
	if filled, ok := got.Data["fields_filled"].(float64); !ok || int(filled) != 2 {
		t.Errorf("Data[fields_filled] = %v, want 2", got.Data["fields_filled"])
	}
}

func TestBackwardChannelTrailingSlashStateDir(t *testing.T) {
	stateDir := t.TempDir()

	payload := ContainerStepResult{Status: "success", Output: "ok", DurationMS: 100}
	raw, _ := json.Marshal(payload)
	if err := os.WriteFile(filepath.Join(stateDir, "result.json"), raw, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ParseContainerStepResult(stateDir + "/")
	if err != nil {
		t.Fatalf("ParseContainerStepResult with trailing slash: %v", err)
	}
	if got == nil || got.Status != "success" {
		t.Errorf("got nil or wrong status with trailing slash: %+v", got)
	}
}
