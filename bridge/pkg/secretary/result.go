// Package secretary provides core types for Secretary features including templates, workflows, and approvals.
package secretary

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//=============================================================================
// Container Step Result (result.json convention)
//=============================================================================

// ContainerStepResult represents the structured output a container writes to
// result.json inside its bind-mounted state directory.
//
// Convention:
//
//	The container runs with NetworkMode "none" — its only output channels are
//	the process exit code and the bind-mounted state directory. A container
//	may (optionally) write a result.json file to /home/claw/.openclaw/result.json
//	inside the container, which maps to
//	/var/lib/armorclaw/agent-state/{definitionID}/result.json on the host.
//
//	The Bridge reads this file via ParseContainerStepResult, passing the
//	host-side state directory path. If the file does not exist the container
//	simply chose not to produce results — this is not an error.
//
//	Forward compatibility: unknown JSON fields are silently ignored so that
//	new container versions can add fields without breaking older Bridge code.
//
// Permission model:
//
//	The state directory is chown'd to UID 10001 (the container user) by
//	factory.go's Spawn() method. The container writes as 10001, Bridge reads
//	as root. The bind mount is synchronous (local filesystem), so there is no
//	race between the container writing the file and Bridge reading it — Docker
//	reports container exit only after all process file descriptors are closed.
type ContainerStepResult struct {
	// Status is the step outcome: "success", "failed", "partial", etc.
	Status string `json:"status"`

	// Output is human-readable output or log summary from the container step.
	Output string `json:"output"`

	// Data holds arbitrary structured data returned by the container step.
	Data map[string]any `json:"data,omitempty"`

	// Error describes the error if the step failed.
	Error string `json:"error,omitempty"`

	// DurationMS is the step execution duration in milliseconds.
	DurationMS int64 `json:"duration_ms"`
}

// ParseContainerStepResult reads and parses result.json from the given
// host-side state directory. The stateDir path is the bind-mount source
// (e.g., /var/lib/armorclaw/agent-state/{definitionID}).
//
// Returns (nil, nil) if result.json does not exist — the container chose
// not to produce results. Returns (nil, error) for I/O or JSON errors.
func ParseContainerStepResult(stateDir string) (*ContainerStepResult, error) {
	path := stateDir
	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}
	path += "result.json"

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read result.json: %w", err)
	}

	var result ContainerStepResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse result.json: %w", err)
	}

	return &result, nil
}

// StepEvent is a structured event recorded during container step execution.
type StepEvent struct {
	Seq        int                    `json:"seq"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	TsMs       int64                  `json:"ts_ms"`
	Detail     map[string]interface{} `json:"detail,omitempty"`
	DurationMs *int                   `json:"duration_ms,omitempty"`
}

// Blocker describes an obstacle that prevented step completion.
type Blocker struct {
	BlockerType string `json:"blocker_type"`
	Message     string `json:"message"`
	Suggestion  string `json:"suggestion,omitempty"`
	Field       string `json:"field,omitempty"`
}

// SkillCandidate is a detected automation opportunity for reuse.
type SkillCandidate struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PatternType string  `json:"pattern_type"`
	PatternData string  `json:"pattern_data,omitempty"`
	Confidence  float64 `json:"confidence"`
}

// EventsSummary holds aggregated event counts by type.
type EventsSummary struct {
	Total int            `json:"total"`
	Types map[string]int `json:"types"`
}

// ExtendedStepResult extends ContainerStepResult with underscore-prefixed metadata
// from newer container versions. Embeds *ContainerStepResult for field access.
type ExtendedStepResult struct {
	*ContainerStepResult

	Comments        []string         `json:"_comments,omitempty"`
	Blockers        []Blocker        `json:"_blockers,omitempty"`
	SkillCandidates []SkillCandidate `json:"_skill_candidates,omitempty"`
	EventsSummary   *EventsSummary   `json:"_events_summary,omitempty"`
	Events          []StepEvent      `json:"-"`
}

// rawExtended unmarshals underscore-prefixed fields from result.json.
type rawExtended struct {
	Comments        []string         `json:"_comments"`
	Blockers        []Blocker        `json:"_blockers"`
	SkillCandidates []SkillCandidate `json:"_skill_candidates"`
	EventsSummary   *EventsSummary   `json:"_events_summary"`
}

// ParseExtendedStepResult parses result.json (base + underscore-prefixed fields)
// and _events.jsonl. Returns (nil, nil) if result.json does not exist.
func ParseExtendedStepResult(stateDir string) (*ExtendedStepResult, error) {
	base, err := ParseContainerStepResult(stateDir)
	if err != nil {
		return nil, err
	}
	if base == nil {
		return nil, nil
	}

	resultPath := filepath.Join(stateDir, "result.json")
	data, err := os.ReadFile(resultPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &ExtendedStepResult{ContainerStepResult: base}, nil
		}
		return nil, fmt.Errorf("read result.json for extended fields: %w", err)
	}

	var raw rawExtended
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse extended fields: %w", err)
	}

	ext := &ExtendedStepResult{
		ContainerStepResult: base,
		Comments:            raw.Comments,
		Blockers:            raw.Blockers,
		SkillCandidates:     raw.SkillCandidates,
		EventsSummary:       raw.EventsSummary,
	}

	events, err := ReadEventsFile(stateDir)
	if err != nil {
		return nil, fmt.Errorf("read events: %w", err)
	}
	ext.Events = events

	return ext, nil
}

// ReadEventsFile parses _events.jsonl line by line. Returns nil if file missing.
func ReadEventsFile(stateDir string) ([]StepEvent, error) {
	eventsPath := filepath.Join(stateDir, "_events.jsonl")

	f, err := os.Open(eventsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open _events.jsonl: %w", err)
	}
	defer f.Close()

	var events []StepEvent
	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var evt StepEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			return nil, fmt.Errorf("parse _events.jsonl line %d: %w", lineNum, err)
		}
		events = append(events, evt)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read _events.jsonl: %w", err)
	}

	return events, nil
}
