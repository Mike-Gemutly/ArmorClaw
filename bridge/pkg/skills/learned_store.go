// Package skills provides learned skill storage for the ArmorClaw agent system.
// Learned skills are execution patterns extracted from successful task completions,
// stored as suggestions for future similar tasks.
package skills

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LearnedSkill represents a reusable execution pattern learned from task history.
// These are suggestions only — they are never auto-executed.
type LearnedSkill struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Description      string  `json:"description,omitempty"`
	SourceTaskID     string  `json:"source_task_id,omitempty"`
	SourceTemplateID string  `json:"source_template_id,omitempty"`
	PatternType      string  `json:"pattern_type"`
	PatternData      string  `json:"pattern_data"`
	TriggerKeywords  string  `json:"trigger_keywords"`
	SuccessCount     int     `json:"success_count"`
	FailureCount     int     `json:"failure_count"`
	LastUsedAt       *int64  `json:"last_used_at,omitempty"`
	CreatedAt        int64   `json:"created_at"`
	Confidence       float64 `json:"confidence"`
}

// LearnedStore provides persistence for learned skills using plain SQLite
// (NOT SQLCipher — learned skills contain no secrets).
type LearnedStore struct {
	db *sql.DB
}

// NewLearnedStore creates a new LearnedStore backed by the given database connection.
// The caller is responsible for ensuring the learned_skills table exists.
func NewLearnedStore(db *sql.DB) *LearnedStore {
	return &LearnedStore{db: db}
}

// Save persists a LearnedSkill to the database. If the skill has no ID, one is generated.
// Returns an error on duplicate name or database failure.
func (s *LearnedStore) Save(skill LearnedSkill) (*LearnedSkill, error) {
	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}
	if skill.CreatedAt == 0 {
		skill.CreatedAt = time.Now().UnixMilli()
	}
	if skill.Confidence == 0 {
		skill.Confidence = 0.5
	}

	_, err := s.db.Exec(`
		INSERT INTO learned_skills (id, name, description, source_task_id, source_template_id,
			pattern_type, pattern_data, trigger_keywords, success_count, failure_count,
			last_used_at, created_at, confidence)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		skill.ID, skill.Name, skill.Description, skill.SourceTaskID, skill.SourceTemplateID,
		skill.PatternType, skill.PatternData, skill.TriggerKeywords,
		skill.SuccessCount, skill.FailureCount, skill.LastUsedAt, skill.CreatedAt, skill.Confidence,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, fmt.Errorf("skill with name %q already exists", skill.Name)
		}
		return nil, fmt.Errorf("failed to save learned skill: %w", err)
	}

	return &skill, nil
}

// FindForTask searches for learned skills matching a task description.
// It filters by confidence >= 0.4 and ranks results by keyword overlap with the description.
// The limit parameter controls the maximum number of results returned.
func (s *LearnedStore) FindForTask(taskDesc string, limit int) ([]*LearnedSkill, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, source_task_id, source_template_id,
			pattern_type, pattern_data, trigger_keywords,
			success_count, failure_count, last_used_at, created_at, confidence
		FROM learned_skills
		WHERE confidence >= 0.4
		ORDER BY confidence DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query learned skills: %w", err)
	}
	defer rows.Close()

	var skills []*LearnedSkill
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}

	taskLower := strings.ToLower(taskDesc)
	taskWords := splitWords(taskLower)
	ranked := rankByOverlap(skills, taskWords)

	if len(ranked) > limit {
		ranked = ranked[:limit]
	}

	return ranked, nil
}

// RecordOutcome updates a skill's confidence based on a success/failure outcome.
// Success increases confidence (capped at 1.0), failure decreases it (floored at 0.0).
func (s *LearnedStore) RecordOutcome(skillID string, success bool) error {
	var current float64
	var successes, failures int

	err := s.db.QueryRow(`
		SELECT confidence, success_count, failure_count
		FROM learned_skills WHERE id = ?`, skillID,
	).Scan(&current, &successes, &failures)
	if err != nil {
		return fmt.Errorf("failed to find skill %s: %w", skillID, err)
	}

	if success {
		successes++
		current = current + 0.1
		if current > 1.0 {
			current = 1.0
		}
	} else {
		failures++
		current = current - 0.2
		if current < 0.0 {
			current = 0.0
		}
	}

	now := time.Now().UnixMilli()
	_, err = s.db.Exec(`
		UPDATE learned_skills
		SET confidence = ?, success_count = ?, failure_count = ?, last_used_at = ?
		WHERE id = ?`, current, successes, failures, now, skillID,
	)
	if err != nil {
		return fmt.Errorf("failed to update skill outcome: %w", err)
	}

	return nil
}

// Delete removes a learned skill by ID.
func (s *LearnedStore) Delete(skillID string) error {
	_, err := s.db.Exec(`DELETE FROM learned_skills WHERE id = ?`, skillID)
	if err != nil {
		return fmt.Errorf("failed to delete skill %s: %w", skillID, err)
	}
	return nil
}

// ListForAgent returns learned skills ordered by confidence, suitable for
// an agent to browse. The limit parameter controls the maximum results.
func (s *LearnedStore) ListForAgent(limit int) ([]*LearnedSkill, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, source_task_id, source_template_id,
			pattern_type, pattern_data, trigger_keywords,
			success_count, failure_count, last_used_at, created_at, confidence
		FROM learned_skills
		ORDER BY confidence DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list skills: %w", err)
	}
	defer rows.Close()

	var skills []*LearnedSkill
	for rows.Next() {
		sk, err := scanSkill(rows)
		if err != nil {
			return nil, err
		}
		skills = append(skills, sk)
	}

	return skills, nil
}

// scanSkill scans a single row into a LearnedSkill.
func scanSkill(rows *sql.Rows) (*LearnedSkill, error) {
	sk := &LearnedSkill{}
	var lastUsedAt sql.NullInt64

	err := rows.Scan(
		&sk.ID, &sk.Name, &sk.Description, &sk.SourceTaskID, &sk.SourceTemplateID,
		&sk.PatternType, &sk.PatternData, &sk.TriggerKeywords,
		&sk.SuccessCount, &sk.FailureCount, &lastUsedAt, &sk.CreatedAt, &sk.Confidence,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan skill row: %w", err)
	}

	if lastUsedAt.Valid {
		sk.LastUsedAt = &lastUsedAt.Int64
	}

	return sk, nil
}

// splitWords splits a lowercase string into word tokens for matching.
func splitWords(s string) []string {
	return strings.Fields(s)
}

// rankByOverlap reorders skills by keyword overlap score with task words.
func rankByOverlap(skills []*LearnedSkill, taskWords []string) []*LearnedSkill {
	type scored struct {
		skill *LearnedSkill
		score int
	}

	taskSet := make(map[string]struct{}, len(taskWords))
	for _, w := range taskWords {
		taskSet[w] = struct{}{}
	}

	scoredSkills := make([]scored, 0, len(skills))
	for _, sk := range skills {
		kwLower := strings.ToLower(sk.TriggerKeywords)
		kwWords := splitWords(kwLower)
		overlap := 0
		for _, w := range kwWords {
			if _, ok := taskSet[w]; ok {
				overlap++
			}
		}
		scoredSkills = append(scoredSkills, scored{skill: sk, score: overlap})
	}

	for i := 0; i < len(scoredSkills)-1; i++ {
		for j := i + 1; j < len(scoredSkills); j++ {
			if scoredSkills[j].score > scoredSkills[i].score ||
				(scoredSkills[j].score == scoredSkills[i].score &&
					scoredSkills[j].skill.Confidence > scoredSkills[i].skill.Confidence) {
				scoredSkills[i], scoredSkills[j] = scoredSkills[j], scoredSkills[i]
			}
		}
	}

	result := make([]*LearnedSkill, len(scoredSkills))
	for i, s := range scoredSkills {
		result[i] = s.skill
	}
	return result
}
