// Package budget provides token budget tracking and enforcement for ArmorClaw
package budget

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/armorclaw/bridge/pkg/notification"
)

// TokenCosts defines approximate costs per 1M tokens (input + output average)
// Prices are in USD as of 2024
var TokenCosts = map[string]float64{
	"gpt-4":          30.00,
	"gpt-4-turbo":    10.00,
	"gpt-3.5-turbo":  0.50,
	"claude-3-opus":  15.00,
	"claude-3-sonnet": 3.00,
	"claude-3-haiku":  0.25,
}

// UsageRecord tracks token usage for a session
type UsageRecord struct {
	SessionID    string    `json:"session_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Timestamp    time.Time `json:"timestamp"`
	CostUSD      float64   `json:"cost_usd"`
}

// BudgetConfig defines budget limits
type BudgetConfig struct {
	DailyLimitUSD   float64 `toml:"daily_limit_usd" env:"ARMORCLAW_DAILY_LIMIT"`
	MonthlyLimitUSD float64 `toml:"monthly_limit_usd" env:"ARMORCLAW_MONTHLY_LIMIT"`
	AlertThreshold  float64 `toml:"alert_threshold" env:"ARMORCLAW_ALERT_THRESHOLD"` // % of limit
	HardStop        bool    `toml:"hard_stop" env:"ARMORCLAW_HARD_STOP"`
	ProviderCosts   map[string]float64 `toml:"provider_costs"`
}

// BudgetTracker monitors and enforces token budgets
type BudgetTracker struct {
	config         BudgetConfig         `json:"config"`
	usageHistory   []UsageRecord        `json:"usage_history"`
	dailyUsage     map[string]float64   `json:"daily_usage"`
	monthlyUsage   map[string]float64   `json:"monthly_usage"`
	sessionUsage   map[string]float64   `json:"session_usage"`
	mutex          sync.RWMutex         `json:"-"`
	stateFilePath  string               `json:"-"`
	costs          map[string]float64   `json:"costs"`
	notifier       *notification.Notifier `json:"-"`
}

// NewBudgetTracker creates a new budget tracker
func NewBudgetTracker(config BudgetConfig, stateDir string) *BudgetTracker {
	// Initialize state directory
	if stateDir == "" {
		stateDir = "/var/lib/armorclaw"
	}

	// Merge custom provider costs with defaults
	costs := make(map[string]float64)
	for k, v := range TokenCosts {
		costs[k] = v
	}
	if config.ProviderCosts != nil {
		for k, v := range config.ProviderCosts {
			costs[k] = v
		}
	}

	return &BudgetTracker{
		config:        config,
		usageHistory:  make([]UsageRecord, 0),
		dailyUsage:    make(map[string]float64),
		monthlyUsage:  make(map[string]float64),
		sessionUsage:  make(map[string]float64),
		stateFilePath: filepath.Join(stateDir, "budget_state.json"),
		costs:         costs,
	}
}

// RecordUsage records token usage for a session
func (b *BudgetTracker) RecordUsage(record UsageRecord) error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	// Calculate cost
	costPer1M := b.costs[record.Model]
	if costPer1M == 0 {
		costPer1M = 0.001 // Default fallback
	}

	totalTokens := record.InputTokens + record.OutputTokens
	record.CostUSD = float64(totalTokens) / 1000000.0 * costPer1M
	record.Timestamp = time.Now()

	// Add to history
	b.usageHistory = append(b.usageHistory, record)

	// Update tracking maps
	date := record.Timestamp.Format("2006-01-02")
	month := record.Timestamp.Format("2006-01")
	b.dailyUsage[date] += record.CostUSD
	b.monthlyUsage[month] += record.CostUSD
	b.sessionUsage[record.SessionID] += record.CostUSD

	// Check limits
	return b.checkLimits(&record)
}

// checkLimits verifies if usage is within budget limits
func (b *BudgetTracker) checkLimits(record *UsageRecord) error {
	date := record.Timestamp.Format("2006-01-02")
	month := record.Timestamp.Format("2006-01")

	dailyCost := b.dailyUsage[date]
	monthlyCost := b.monthlyUsage[month]

	// Check daily limit
	if b.config.DailyLimitUSD > 0 && dailyCost >= b.config.DailyLimitUSD {
		if b.config.HardStop {
			return fmt.Errorf("daily budget limit exceeded: $%.2f / $%.2f",
				dailyCost, b.config.DailyLimitUSD)
		}
		// Send alert
		b.sendAlert("daily_limit_exceeded", dailyCost, b.config.DailyLimitUSD)
	}

	// Check monthly limit
	if b.config.MonthlyLimitUSD > 0 && monthlyCost >= b.config.MonthlyLimitUSD {
		if b.config.HardStop {
			return fmt.Errorf("monthly budget limit exceeded: $%.2f / $%.2f",
				monthlyCost, b.config.MonthlyLimitUSD)
		}
		b.sendAlert("monthly_limit_exceeded", monthlyCost, b.config.MonthlyLimitUSD)
	}

	// Check alert threshold
	if b.config.AlertThreshold > 0 {
		if b.config.DailyLimitUSD > 0 {
			dailyPercent := (dailyCost / b.config.DailyLimitUSD) * 100
			if dailyPercent >= b.config.AlertThreshold {
				b.sendAlert("daily_budget_warning", dailyCost, b.config.DailyLimitUSD)
			}
		}

		if b.config.MonthlyLimitUSD > 0 {
			monthlyPercent := (monthlyCost / b.config.MonthlyLimitUSD) * 100
			if monthlyPercent >= b.config.AlertThreshold {
				b.sendAlert("monthly_budget_warning", monthlyCost, b.config.MonthlyLimitUSD)
			}
		}
	}

	return nil
}

// CanStartSession checks if a new session can be started based on budget
func (b *BudgetTracker) CanStartSession() error {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	now := time.Now()
	date := now.Format("2006-01-02")
	month := now.Format("2006-01")

	dailyCost := b.dailyUsage[date]
	monthlyCost := b.monthlyUsage[month]

	// Check if we've already hit the limits
	if b.config.HardStop {
		if b.config.DailyLimitUSD > 0 && dailyCost >= b.config.DailyLimitUSD {
			return fmt.Errorf("daily budget limit reached: $%.2f / $%.2f",
				dailyCost, b.config.DailyLimitUSD)
		}

		if b.config.MonthlyLimitUSD > 0 && monthlyCost >= b.config.MonthlyLimitUSD {
			return fmt.Errorf("monthly budget limit reached: $%.2f / $%.2f",
				monthlyCost, b.config.MonthlyLimitUSD)
		}
	}

	return nil
}

// GetDailyUsage returns current daily usage in USD
func (b *BudgetTracker) GetDailyUsage() float64 {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	date := time.Now().Format("2006-01-02")
	return b.dailyUsage[date]
}

// GetMonthlyUsage returns current monthly usage in USD
func (b *BudgetTracker) GetMonthlyUsage() float64 {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	month := time.Now().Format("2006-01")
	return b.monthlyUsage[month]
}

// GetSessionUsage returns usage for a specific session
func (b *BudgetTracker) GetSessionUsage(sessionID string) float64 {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.sessionUsage[sessionID]
}

// GetDailyLimit returns the configured daily limit
func (b *BudgetTracker) GetDailyLimit() float64 {
	return b.config.DailyLimitUSD
}

// GetMonthlyLimit returns the configured monthly limit
func (b *BudgetTracker) GetMonthlyLimit() float64 {
	return b.config.MonthlyLimitUSD
}

// sendAlert sends a budget alert via the notification system
func (b *BudgetTracker) sendAlert(alertType string, current, limit float64) {
	// Try to send via notification system first
	if b.notifier != nil {
		// Determine session ID (use "global" for budget-wide alerts)
		sessionID := "global"
		if alertType == "session_limit" || alertType == "session_warning" {
			// For session-specific alerts, we'd need to track the current session
			// For now, use "session" as a placeholder
			sessionID = "session"
		}

		err := b.notifier.SendBudgetAlert(alertType, sessionID, current, limit)
		if err == nil {
			return // Successfully sent via Matrix
		}
		// Fall through to logging if Matrix send fails
	}

	// Fallback to console logging
	fmt.Printf("[BUDGET ALERT] %s - Current: $%.2f, Limit: $%.2f\n",
		alertType, current, limit)
}

// SetNotifier sets the notification system for budget alerts
func (b *BudgetTracker) SetNotifier(notifier *notification.Notifier) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.notifier = notifier
}

// jsonState is used for JSON serialization
type jsonState struct {
	Config       BudgetConfig           `json:"config"`
	UsageHistory []UsageRecord          `json:"usage_history"`
	DailyUsage   map[string]float64     `json:"daily_usage"`
	MonthlyUsage map[string]float64     `json:"monthly_usage"`
	SessionUsage map[string]float64     `json:"session_usage"`
	Costs        map[string]float64     `json:"costs"`
}

// SaveState persists budget state to disk
func (b *BudgetTracker) SaveState() error {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(b.stateFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	// Create JSON-serializable state
	state := jsonState{
		Config:       b.config,
		UsageHistory: b.usageHistory,
		DailyUsage:   b.dailyUsage,
		MonthlyUsage: b.monthlyUsage,
		SessionUsage: b.sessionUsage,
		Costs:        b.costs,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(b.stateFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadState restores budget state from disk
func (b *BudgetTracker) LoadState() error {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	data, err := os.ReadFile(b.stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// First run, no state yet
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal into jsonState struct
	var state jsonState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	// Copy data back to the tracker
	b.config = state.Config
	b.usageHistory = state.UsageHistory
	b.dailyUsage = state.DailyUsage
	b.monthlyUsage = state.MonthlyUsage
	b.sessionUsage = state.SessionUsage
	b.costs = state.Costs

	return nil
}

// Reset clears all usage data
func (b *BudgetTracker) Reset() {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.usageHistory = make([]UsageRecord, 0)
	b.dailyUsage = make(map[string]float64)
	b.monthlyUsage = make(map[string]float64)
	b.sessionUsage = make(map[string]float64)
}

// GetUsageHistory returns the usage history
func (b *BudgetTracker) GetUsageHistory() []UsageRecord {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Return a copy to prevent external modification
	history := make([]UsageRecord, len(b.usageHistory))
	copy(history, b.usageHistory)
	return history
}

// SetCost sets a custom cost for a model
func (b *BudgetTracker) SetCost(model string, costPer1M float64) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.costs[model] = costPer1M
}

// GetCost returns the cost per 1M tokens for a model
func (b *BudgetTracker) GetCost(model string) float64 {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if cost, ok := b.costs[model]; ok {
		return cost
	}
	return 0.001 // Default fallback
}
