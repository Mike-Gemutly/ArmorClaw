package skills

import (
	"context"
	"testing"
	"time"

	"github.com/armorclaw/bridge/pkg/governor"
)

// TestDefaultTimeout_S12 verifies that a skill with Timeout=0 gets the 30s
// default instead of an immediate (zero-duration) timeout. The skill should
// execute successfully, not time out.
func TestDefaultTimeout_S12(t *testing.T) {
	se := NewSkillExecutorWithConfig(SkillExecutorConfig{
		SkillGate: governor.NewGovernor(nil, nil),
	})
	se.registry.skills["zerotimeout"] = &Skill{
		Name:    "zerotimeout",
		Domain:  "weather",
		Enabled: true,
		Timeout: 0,
	}
	se.AllowSkill("zerotimeout")

	result, err := se.ExecuteSkill(context.Background(), "zerotimeout", map[string]interface{}{"city": "Berlin"})
	if err != nil {
		t.Fatalf("expected no error with zero timeout (should default to 30s), got %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Success {
		t.Errorf("expected Success=true, got Success=false; Type=%s, Error=%s", result.Type, result.Error)
	}
	if result.Type == "timeout" {
		t.Error("skill should not have timed out with the 30s default")
	}
}

// TestTimeoutDetection_S13 verifies that when a skill's execution context
// deadline is exceeded, the error Type is reported as "timeout" (not "error").
// Uses a custom handler that blocks until context cancellation.
func TestTimeoutDetection_S13(t *testing.T) {
	se := NewSkillExecutorWithConfig(SkillExecutorConfig{
		SkillGate: governor.NewGovernor(nil, nil),
	})
	se.customHandlers["blocking"] = func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	se.registry.skills["quick-expire"] = &Skill{
		Name:    "quick-expire",
		Domain:  "blocking",
		Enabled: true,
		Timeout: 1 * time.Nanosecond,
	}
	se.AllowSkill("quick-expire")

	result, err := se.ExecuteSkill(context.Background(), "quick-expire", nil)
	if err == nil {
		t.Fatal("expected error due to timeout")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Success {
		t.Error("expected Success=false")
	}
	if result.Type != "timeout" {
		t.Errorf("expected Type=timeout, got Type=%s", result.Type)
	}
}

// TestParentCancellationNotMisreported_S13 verifies that when the parent
// context is cancelled (not a skill timeout), the error type is NOT
// misreported as "timeout". It should be "error" (context.Canceled).
func TestParentCancellationNotMisreported_S13(t *testing.T) {
	se := NewSkillExecutorWithConfig(SkillExecutorConfig{
		SkillGate: governor.NewGovernor(nil, nil),
	})
	se.customHandlers["blocking"] = func(ctx context.Context, params map[string]interface{}) (interface{}, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	se.registry.skills["parent-cancel"] = &Skill{
		Name:    "parent-cancel",
		Domain:  "blocking",
		Enabled: true,
		Timeout: 30 * time.Second,
	}
	se.AllowSkill("parent-cancel")

	parentCtx, parentCancel := context.WithCancel(context.Background())
	parentCancel()

	result, err := se.ExecuteSkill(parentCtx, "parent-cancel", nil)
	if err == nil {
		t.Fatal("expected error due to parent cancellation")
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Success {
		t.Error("expected Success=false")
	}
	if result.Type == "timeout" {
		t.Errorf("error should NOT be reported as timeout (parent was cancelled, not deadline exceeded); got Type=%s", result.Type)
	}
}
