package secretary

import (
	"strings"
	"testing"
	"time"
)

func TestTimelineFromParsedEvents(t *testing.T) {
	d150 := 150
	result := &ExtendedStepResult{
		ContainerStepResult: &ContainerStepResult{
			Output:     "Deploy to staging",
			DurationMS: 2500,
		},
		Events: []StepEvent{
			{Seq: 1, Type: "step", Name: "Initialize", DurationMs: &d150},
			{Seq: 2, Type: "file_read", Name: "config.yaml", Detail: map[string]interface{}{"lines": 100}, DurationMs: &d150},
			{Seq: 3, Type: "command_run", Name: "go test ./...", Detail: map[string]interface{}{"exit_code": 0}},
			{Seq: 4, Type: "artifact", Name: "binary.zip", Detail: map[string]interface{}{"size_bytes": 2048}},
		},
		EventsSummary: &EventsSummary{
			Total: 4,
			Types: map[string]int{"step": 1, "file_read": 1, "command_run": 1, "artifact": 1},
		},
	}

	msg := FormatTimelineMessage(result)

	if !strings.Contains(msg, "📋 Deploy to staging") {
		t.Errorf("expected title line, got: %s", msg)
	}
	if !strings.Contains(msg, "🔹 Initialize") {
		t.Errorf("expected step icon and name, got: %s", msg)
	}
	if !strings.Contains(msg, "📄 config.yaml (100 lines)") {
		t.Errorf("expected file_read with lines, got: %s", msg)
	}
	if !strings.Contains(msg, "⌨️ go test ./... ✓") {
		t.Errorf("expected command_run with success, got: %s", msg)
	}
	if !strings.Contains(msg, "📦 binary.zip (2048 bytes)") {
		t.Errorf("expected artifact with size, got: %s", msg)
	}
	if !strings.Contains(msg, "⏱ 2.5s · 4 steps") {
		t.Errorf("expected footer, got: %s", msg)
	}
}

func TestTimelineEmptyFallback(t *testing.T) {
	result := &ExtendedStepResult{
		ContainerStepResult: &ContainerStepResult{
			Output: "Task complete",
		},
		Events: []StepEvent{},
	}

	msg := FormatTimelineMessage(result)
	if msg != "Task complete" {
		t.Errorf("expected plain text fallback, got: %q", msg)
	}
}

func TestTimelineTruncatedDetail(t *testing.T) {
	result := &ExtendedStepResult{
		ContainerStepResult: &ContainerStepResult{
			Output:     "Build project",
			DurationMS: 1000,
		},
		Events: []StepEvent{
			{
				Seq:    1,
				Type:   "file_read",
				Name:   "huge.log",
				Detail: map[string]interface{}{"lines": 50000, "_truncated": true},
			},
		},
		EventsSummary: &EventsSummary{Total: 1},
	}

	msg := FormatTimelineMessage(result)

	if !strings.Contains(msg, "[truncated]") {
		t.Errorf("expected [truncated] marker, got: %s", msg)
	}
	if !strings.Contains(msg, "📄 huge.log (50000 lines)") {
		t.Errorf("expected file_read with lines and truncated, got: %s", msg)
	}
}

func TestBlockerNotif_FormatSingle(t *testing.T) {
	blockers := []Blocker{
		{
			BlockerType: "missing_config",
			Message:     "API key not set",
			Suggestion:  "Set OPENROUTER_API_KEY in environment",
			Field:       "api_key",
		},
	}
	msg := FormatBlockerMessage(blockers, 5*time.Minute+30*time.Second+500*time.Millisecond)

	if !strings.Contains(msg, "🚧") {
		t.Error("expected 🚧 header")
	}
	if !strings.Contains(msg, "API key not set") {
		t.Error("expected blocker message")
	}
	if !strings.Contains(msg, "Set OPENROUTER_API_KEY in environment") {
		t.Error("expected suggestion")
	}
	if !strings.Contains(msg, "Expires in") {
		t.Error("expected timeout footer")
	}
	if !strings.Contains(msg, "5m31s") {
		t.Errorf("expected rounded timeout 5m31s, got: %s", msg)
	}
}

func TestBlockerNotif_FormatMultiple(t *testing.T) {
	blockers := []Blocker{
		{Message: "First issue"},
		{Message: "Second issue"},
	}
	msg := FormatBlockerMessage(blockers, 10*time.Minute)

	if !strings.Contains(msg, "Blocker 1:") {
		t.Error("expected 'Blocker 1'")
	}
	if !strings.Contains(msg, "Blocker 2:") {
		t.Error("expected 'Blocker 2'")
	}
	if !strings.Contains(msg, "───────────") {
		t.Error("expected separator between blockers")
	}
}

func TestBlockerNotif_FormatEmpty(t *testing.T) {
	msg := FormatBlockerMessage(nil, 5*time.Minute)
	if msg != "" {
		t.Errorf("expected empty string for nil blockers, got: %q", msg)
	}

	msg = FormatBlockerMessage([]Blocker{}, 5*time.Minute)
	if msg != "" {
		t.Errorf("expected empty string for empty slice, got: %q", msg)
	}
}

func TestBlockerNotif_FormatMinimal(t *testing.T) {
	blockers := []Blocker{
		{Message: "Something went wrong"},
	}
	msg := FormatBlockerMessage(blockers, 1*time.Minute)

	if !strings.Contains(msg, "Something went wrong") {
		t.Error("expected message")
	}
	if strings.Contains(msg, "💡") {
		t.Error("unexpected suggestion for minimal blocker")
	}
	if strings.Contains(msg, "🔑") {
		t.Error("unexpected field for minimal blocker")
	}
	if strings.Contains(msg, "📋") {
		t.Error("unexpected type for minimal blocker")
	}
}
