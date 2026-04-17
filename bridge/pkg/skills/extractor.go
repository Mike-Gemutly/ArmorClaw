package skills

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/secretary"
)

const (
	PatternCommandSequence    = "command_sequence"
	PatternFileTransform      = "file_transform"
	PatternConfigTemplate     = "config_template"
	PatternStepSequence       = "step_sequence"
	PatternCheckpointSequence = "checkpoint_sequence"
)

// ExtractFromResult analyzes an ExtendedStepResult using multiple strategies
// to produce LearnedSkill suggestions. Skills are never auto-executed.
// Callers should check task success status before calling this function.
func ExtractFromResult(
	result *secretary.ExtendedStepResult,
	taskDesc, taskID, templateID string,
) []LearnedSkill {
	if result == nil {
		return nil
	}

	var skills []LearnedSkill

	// Strategy 1: Agent self-reported skill candidates.
	for _, cand := range result.SkillCandidates {
		conf := cand.Confidence
		if conf <= 0 {
			conf = 0.5
		}
		skills = append(skills, LearnedSkill{
			ID:               fmt.Sprintf("ls_%s_%d", taskID, time.Now().UnixMilli()),
			Name:             cand.Name,
			Description:      cand.Description,
			SourceTaskID:     taskID,
			SourceTemplateID: templateID,
			PatternType:      cand.PatternType,
			PatternData:      cand.PatternData,
			Confidence:       conf,
			TriggerKeywords:  taskDesc,
			CreatedAt:        time.Now().UnixMilli(),
		})
	}

	// Strategy 2: Command sequence extraction (2+ commands required).
	cmds := extractCommandSequence(result.Events)
	if len(cmds) >= 2 {
		pd, _ := json.Marshal(cmds)
		skills = append(skills, LearnedSkill{
			ID:               fmt.Sprintf("ls_%s_%d", taskID, time.Now().UnixMilli()),
			Name:             generateSkillName(taskDesc, "cmdseq"),
			SourceTaskID:     taskID,
			SourceTemplateID: templateID,
			PatternType:      PatternCommandSequence,
			PatternData:      string(pd),
			Confidence:       0.6,
			TriggerKeywords:  taskDesc,
			CreatedAt:        time.Now().UnixMilli(),
		})
	}

	// Strategy 3: File operations (1+ writes or 2+ reads).
	fileOps := extractFileOperations(result.Events)
	writeCount := len(fileOps["file_write"])
	readCount := len(fileOps["file_read"])
	if writeCount >= 1 || readCount >= 2 {
		pd, _ := json.Marshal(fileOps)
		skills = append(skills, LearnedSkill{
			ID:               fmt.Sprintf("ls_%s_%d", taskID, time.Now().UnixMilli()),
			Name:             generateSkillName(taskDesc, "fileops"),
			SourceTaskID:     taskID,
			SourceTemplateID: templateID,
			PatternType:      PatternFileTransform,
			PatternData:      string(pd),
			Confidence:       0.5,
			TriggerKeywords:  taskDesc,
			CreatedAt:        time.Now().UnixMilli(),
		})
	}

	// Strategy 2b: Step sequence extraction (3+ distinct step names required).
	stepSeq := extractStepSequence(result.Events)
	if len(stepSeq) >= 3 {
		pd, _ := json.Marshal(stepSeq)
		skills = append(skills, LearnedSkill{
			ID:               fmt.Sprintf("ls_%s_%d", taskID, time.Now().UnixMilli()),
			Name:             generateSkillName(taskDesc, "stepseq"),
			SourceTaskID:     taskID,
			SourceTemplateID: templateID,
			PatternType:      PatternStepSequence,
			PatternData:      string(pd),
			Confidence:       0.5,
			TriggerKeywords:  taskDesc,
			CreatedAt:        time.Now().UnixMilli(),
		})
	}

	// Strategy 3b: Checkpoint sequence extraction (any checkpoint events).
	checkpoints := extractCheckpoints(result.Events)
	if len(checkpoints) > 0 {
		pd, _ := json.Marshal(checkpoints)
		skills = append(skills, LearnedSkill{
			ID:               fmt.Sprintf("ls_%s_%d", taskID, time.Now().UnixMilli()),
			Name:             generateSkillName(taskDesc, "ckptseq"),
			SourceTaskID:     taskID,
			SourceTemplateID: templateID,
			PatternType:      PatternCheckpointSequence,
			PatternData:      string(pd),
			Confidence:       0.4,
			TriggerKeywords:  taskDesc,
			CreatedAt:        time.Now().UnixMilli(),
		})
	}

	return deduplicateSkills(skills)
}

func extractCommandSequence(events []secretary.StepEvent) []map[string]interface{} {
	var cmds []map[string]interface{}
	for _, evt := range events {
		if evt.Type == "command_run" {
			entry := map[string]interface{}{
				"command":   evt.Name,
				"exit_code": 0,
			}
			if evt.Detail != nil {
				if ec, ok := evt.Detail["exit_code"]; ok {
					entry["exit_code"] = ec
				}
				if cmd, ok := evt.Detail["command"]; ok {
					entry["command"] = cmd
				}
			}
			cmds = append(cmds, entry)
		}
	}
	return cmds
}

func extractFileOperations(events []secretary.StepEvent) map[string][]string {
	result := map[string][]string{
		"file_read":   {},
		"file_write":  {},
		"file_delete": {},
	}

	for _, evt := range events {
		switch evt.Type {
		case "file_read", "file_write", "file_delete":
			path := evt.Name
			if evt.Detail != nil {
				if p, ok := evt.Detail["path"]; ok {
					if ps, ok := p.(string); ok {
						path = ps
					}
				}
			}
			result[evt.Type] = append(result[evt.Type], path)
		}
	}

	return result
}

func extractStepSequence(events []secretary.StepEvent) []map[string]interface{} {
	seen := make(map[string]struct{})
	var steps []map[string]interface{}
	for _, evt := range events {
		if evt.Type == "step" {
			if _, ok := seen[evt.Name]; ok {
				continue
			}
			seen[evt.Name] = struct{}{}
			entry := map[string]interface{}{
				"name": evt.Name,
				"seq":  evt.Seq,
			}
			steps = append(steps, entry)
		}
	}
	return steps
}

func extractCheckpoints(events []secretary.StepEvent) []map[string]interface{} {
	var checkpoints []map[string]interface{}
	for _, evt := range events {
		if evt.Type == "checkpoint" {
			entry := map[string]interface{}{
				"name": evt.Name,
				"seq":  evt.Seq,
			}
			checkpoints = append(checkpoints, entry)
		}
	}
	return checkpoints
}

func generateSkillName(taskDesc, suffix string) string {
	h := sha256.Sum256([]byte(taskDesc + suffix))
	return fmt.Sprintf("skill_%x_%s", h[:8], suffix)
}

func deduplicateSkills(skills []LearnedSkill) []LearnedSkill {
	seen := make(map[string]struct{}, len(skills))
	var result []LearnedSkill
	for _, sk := range skills {
		if _, ok := seen[sk.Name]; ok {
			continue
		}
		seen[sk.Name] = struct{}{}
		result = append(result, sk)
	}
	return result
}
