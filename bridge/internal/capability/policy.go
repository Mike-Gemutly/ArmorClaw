package capability

import (
	"strings"
	"sync"
)

type Level string

const (
	LevelNone    Level = "none"
	LevelLow     Level = "low"
	LevelMedium  Level = "medium"
	LevelHigh    Level = "high"
	LevelFull    Level = "full"
)

type Policy struct {
	mu            sync.RWMutex
	defaultLevel  Level
	roomLevels    map[string]Level
	blockedTools  map[string]bool
	requiredTools map[string]bool
}

type PolicyConfig struct {
	DefaultLevel  Level
	RoomLevels    map[string]Level
	BlockedTools  []string
	RequiredTools []string
}

func NewPolicy(cfg PolicyConfig) *Policy {
	if cfg.DefaultLevel == "" {
		cfg.DefaultLevel = LevelMedium
	}

	p := &Policy{
		defaultLevel:  cfg.DefaultLevel,
		roomLevels:    make(map[string]Level),
		blockedTools:  make(map[string]bool),
		requiredTools: make(map[string]bool),
	}

	for room, level := range cfg.RoomLevels {
		p.roomLevels[room] = level
	}

	for _, tool := range cfg.BlockedTools {
		p.blockedTools[strings.ToLower(tool)] = true
	}

	for _, tool := range cfg.RequiredTools {
		p.requiredTools[strings.ToLower(tool)] = true
	}

	return p
}

func (p *Policy) GetLevel(roomID string) Level {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if level, ok := p.roomLevels[roomID]; ok {
		return level
	}
	return p.defaultLevel
}

func (p *Policy) SetLevel(roomID string, level Level) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.roomLevels[roomID] = level
}

func (p *Policy) IsToolAllowed(roomID, toolName string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.blockedTools[strings.ToLower(toolName)] {
		return false
	}

	level := p.defaultLevel
	if l, ok := p.roomLevels[roomID]; ok {
		level = l
	}

	toolLevel := getToolLevel(toolName)
	return canUseTool(level, toolLevel)
}

func (p *Policy) FilterTools(roomID string, tools []string) []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	level := p.defaultLevel
	if l, ok := p.roomLevels[roomID]; ok {
		level = l
	}

	var allowed []string
	for _, tool := range tools {
		if p.blockedTools[strings.ToLower(tool)] {
			continue
		}
		toolLevel := getToolLevel(tool)
		if canUseTool(level, toolLevel) {
			allowed = append(allowed, tool)
		}
	}
	return allowed
}

func (p *Policy) BlockTool(toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blockedTools[strings.ToLower(toolName)] = true
}

func (p *Policy) UnblockTool(toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.blockedTools, strings.ToLower(toolName))
}

func (p *Policy) RequireTool(toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.requiredTools[strings.ToLower(toolName)] = true
}

func (p *Policy) UnrequireTool(toolName string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.requiredTools, strings.ToLower(toolName))
}

func getToolLevel(toolName string) Level {
	lower := strings.ToLower(toolName)

	if strings.Contains(lower, "read") || strings.Contains(lower, "get") || strings.Contains(lower, "list") {
		if strings.Contains(lower, "file") || strings.Contains(lower, "system") {
			return LevelMedium
		}
		return LevelLow
	}

	if strings.Contains(lower, "write") || strings.Contains(lower, "create") || strings.Contains(lower, "update") {
		return LevelHigh
	}

	if strings.Contains(lower, "delete") || strings.Contains(lower, "remove") || strings.Contains(lower, "execute") {
		return LevelFull
	}

	if strings.Contains(lower, "shell") || strings.Contains(lower, "exec") || strings.Contains(lower, "cmd") {
		return LevelFull
	}

	return LevelMedium
}

func canUseTool(userLevel, toolLevel Level) bool {
	levels := map[Level]int{
		LevelNone:    0,
		LevelLow:     1,
		LevelMedium:  2,
		LevelHigh:    3,
		LevelFull:    4,
	}

	userRank := levels[userLevel]
	toolRank := levels[toolLevel]

	return userRank >= toolRank
}
