package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LineageEntry struct {
	ID             string    `json:"id"`
	SourceType     string    `json:"source_type"`
	SourceID       string    `json:"source_id"`
	Transformation string    `json:"transformation"`
	OutputType     string    `json:"output_type"`
	OutputID       string    `json:"output_id"`
	AgentID        string    `json:"agent_id"`
	TeamID         string    `json:"team_id,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

type LineageTracker struct {
	mu   sync.RWMutex
	path string
}

func NewLineageTracker(path string) *LineageTracker {
	return &LineageTracker{path: path}
}

func (lt *LineageTracker) LogArtifactLineage(
	ctx context.Context,
	sourceType, sourceID, transformation, outputType, outputID, agentID string,
) error {
	entry := LineageEntry{
		ID:             generateLineageID(sourceID, outputID, time.Now()),
		SourceType:     sourceType,
		SourceID:       sourceID,
		Transformation: transformation,
		OutputType:     outputType,
		OutputID:       outputID,
		AgentID:        agentID,
		Timestamp:      time.Now(),
	}
	return lt.appendEntry(entry)
}

func (lt *LineageTracker) GetArtifactLineage(ctx context.Context, artifactID string) ([]LineageEntry, error) {
	entries, err := lt.loadAll()
	if err != nil {
		return nil, err
	}

	var chain []LineageEntry
	current := artifactID
	visited := map[string]bool{}

	for {
		if visited[current] {
			break
		}
		visited[current] = true

		found := false
		for _, e := range entries {
			if e.OutputID == current {
				chain = append([]LineageEntry{e}, chain...)
				current = e.SourceID
				found = true
				break
			}
		}
		if !found {
			break
		}
	}

	return chain, nil
}

func (lt *LineageTracker) appendEntry(entry LineageEntry) error {
	lt.mu.Lock()
	defer lt.mu.Unlock()

	entries, _ := lt.loadAll()
	entries = append(entries, entry)

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lineage entries: %w", err)
	}

	dir := filepath.Dir(lt.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create lineage dir: %w", err)
	}

	return os.WriteFile(lt.path, data, 0600)
}

func (lt *LineageTracker) loadAll() ([]LineageEntry, error) {
	data, err := os.ReadFile(lt.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read lineage file: %w", err)
	}
	var entries []LineageEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("unmarshal lineage entries: %w", err)
	}
	return entries, nil
}

func generateLineageID(sourceID, outputID string, t time.Time) string {
	return fmt.Sprintf("lin-%d-%s-%s", t.UnixNano(), min8(sourceID), min8(outputID))
}

func min8(s string) string {
	if len(s) > 8 {
		return s[:8]
	}
	return s
}

// ComplianceEntryV2 extends the audit schema with team-aware fields.
type ComplianceEntryV2 struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	EventType       string    `json:"event_type"`
	Action          string    `json:"action"`
	Status          string    `json:"status"`
	TeamID          string    `json:"team_id,omitempty"`
	MemberRole      string    `json:"member_role,omitempty"`
	DelegationFrom  string    `json:"delegation_from,omitempty"`
	DelegationTo    string    `json:"delegation_to,omitempty"`
}
