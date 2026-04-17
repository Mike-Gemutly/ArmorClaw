package skills

import (
	"testing"

	"github.com/armorclaw/bridge/pkg/secretary"
)

func TestExtractCommandSequence(t *testing.T) {
	result := &secretary.ExtendedStepResult{
		Events: []secretary.StepEvent{
			{Type: "command_run", Name: "apt-get update", Detail: map[string]interface{}{"exit_code": 0}},
			{Type: "command_run", Name: "apt-get install -y nginx", Detail: map[string]interface{}{"exit_code": 0}},
			{Type: "command_run", Name: "systemctl start nginx", Detail: map[string]interface{}{"exit_code": 0}},
		},
	}

	skills := ExtractFromResult(result, "install nginx", "task-1", "tpl-1")

	var found bool
	for _, sk := range skills {
		if sk.PatternType == PatternCommandSequence {
			found = true
			if sk.Confidence != 0.6 {
				t.Errorf("expected confidence 0.6, got %f", sk.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected a command_sequence skill from 3 command_run events")
	}
}

func TestExtractCommandSequenceSingle(t *testing.T) {
	result := &secretary.ExtendedStepResult{
		Events: []secretary.StepEvent{
			{Type: "command_run", Name: "echo hello", Detail: map[string]interface{}{"exit_code": 0}},
		},
	}

	skills := ExtractFromResult(result, "say hello", "task-2", "tpl-2")

	for _, sk := range skills {
		if sk.PatternType == PatternCommandSequence {
			t.Error("single command should not produce command_sequence skill")
		}
	}
}

func TestExtractFileOperations(t *testing.T) {
	result := &secretary.ExtendedStepResult{
		Events: []secretary.StepEvent{
			{Type: "file_read", Name: "/etc/config.yaml"},
			{Type: "file_write", Name: "/etc/config.yaml", Detail: map[string]interface{}{"path": "/etc/config.yaml"}},
		},
	}

	skills := ExtractFromResult(result, "update config", "task-3", "tpl-3")

	var found bool
	for _, sk := range skills {
		if sk.PatternType == PatternFileTransform {
			found = true
			if sk.Confidence != 0.5 {
				t.Errorf("expected confidence 0.5, got %f", sk.Confidence)
			}
		}
	}
	if !found {
		t.Error("expected a file_transform skill from file_read + file_write events")
	}
}

func TestExtractFileOperationsThreshold(t *testing.T) {
	result := &secretary.ExtendedStepResult{
		Events: []secretary.StepEvent{
			{Type: "file_read", Name: "/etc/hosts"},
		},
	}

	skills := ExtractFromResult(result, "read hosts", "task-4", "tpl-4")

	for _, sk := range skills {
		if sk.PatternType == PatternFileTransform {
			t.Error("1 read + 0 writes should not produce file_transform skill")
		}
	}
}

func TestExtractSelfReported(t *testing.T) {
	result := &secretary.ExtendedStepResult{
		SkillCandidates: []secretary.SkillCandidate{
			{
				Name:        "deploy-nginx",
				Description: "Deploy nginx with standard config",
				PatternType: PatternConfigTemplate,
				PatternData: `{"template":"nginx.conf"}`,
				Confidence:  0.85,
			},
		},
	}

	skills := ExtractFromResult(result, "deploy nginx", "task-5", "tpl-5")

	if len(skills) == 0 {
		t.Fatal("expected at least 1 skill from self-reported candidates")
	}

	sk := skills[0]
	if sk.Name != "deploy-nginx" {
		t.Errorf("expected name 'deploy-nginx', got %q", sk.Name)
	}
	if sk.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", sk.Confidence)
	}
	if sk.SourceTaskID != "task-5" {
		t.Errorf("expected source_task_id 'task-5', got %q", sk.SourceTaskID)
	}
}

func TestExtractNoEvents(t *testing.T) {
	result := &secretary.ExtendedStepResult{}

	skills := ExtractFromResult(result, "do nothing", "task-6", "tpl-6")

	if len(skills) != 0 {
		t.Errorf("expected 0 skills from empty result, got %d", len(skills))
	}
}

func TestDeduplicateSkills(t *testing.T) {
	input := []LearnedSkill{
		{Name: "dup", PatternType: PatternCommandSequence, Confidence: 0.6},
		{Name: "dup", PatternType: PatternFileTransform, Confidence: 0.5},
		{Name: "unique", PatternType: PatternConfigTemplate, Confidence: 0.7},
	}

	deduped := deduplicateSkills(input)
	if len(deduped) != 2 {
		t.Fatalf("expected 2 deduplicated skills, got %d", len(deduped))
	}
	if deduped[0].Name != "dup" {
		t.Errorf("expected first skill name 'dup', got %q", deduped[0].Name)
	}
	if deduped[0].Confidence != 0.6 {
		t.Errorf("expected first skill confidence 0.6 (first occurrence), got %f", deduped[0].Confidence)
	}
	if deduped[1].Name != "unique" {
		t.Errorf("expected second skill name 'unique', got %q", deduped[1].Name)
	}
}
