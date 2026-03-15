package secretary

import (
	"strings"
	"testing"
	"time"
)

func TestBlindFill_ValidateApprovalToken(t *testing.T) {
	executor := &BlindFillExecutor{}

	tests := []struct {
		name      string
		token     string
		execID    string
		wantError bool
	}{
		{
			name:      "valid token bound to execution",
			token:     "approval_exec_123_1700000000000",
			execID:    "exec_123",
			wantError: false,
		},
		{
			name:      "token bound to different execution",
			token:     "approval_exec_456_1700000000000",
			execID:    "exec_123",
			wantError: true,
		},
		{
			name:      "empty token",
			token:     "",
			execID:    "exec_123",
			wantError: true,
		},
		{
			name:      "malformed token",
			token:     "invalid_token_format",
			execID:    "exec_123",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &activeExecution{
				id:        tt.execID,
				template:  &TaskTemplate{ID: "tpl_1"},
				request:   &BlindFillRequest{},
				startedAt: time.Now(),
			}

			err := executor.validateApprovalToken(exec, tt.token)

			if tt.wantError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}

func TestBlindFill_ValidatePIIValues(t *testing.T) {
	executor := &BlindFillExecutor{}

	tests := []struct {
		name         string
		deniedFields []string
		piiValues    map[string]string
		wantError    bool
		errorField   string
	}{
		{
			name:         "allowed field passes",
			deniedFields: []string{"payment.cvv"},
			piiValues: map[string]string{
				"billing.address": "123 Main St",
			},
			wantError: false,
		},
		{
			name:         "denied field rejected",
			deniedFields: []string{"payment.card_number"},
			piiValues: map[string]string{
				"payment.card_number": "4242424242424242",
			},
			wantError:  true,
			errorField: "payment.card_number",
		},
		{
			name:         "denied nested field rejected",
			deniedFields: []string{"payment.cvv", "payment.card_number"},
			piiValues: map[string]string{
				"payment.cvv": "123",
			},
			wantError:  true,
			errorField: "payment.cvv",
		},
		{
			name:         "child of denied field rejected",
			deniedFields: []string{"payment.cvv"},
			piiValues: map[string]string{
				"payment.cvv.something": "value",
			},
			wantError:  true,
			errorField: "payment.cvv.something",
		},
		{
			name:         "empty pii values pass",
			deniedFields: []string{"payment.cvv"},
			piiValues:    nil,
			wantError:    false,
		},
		{
			name:         "no denied fields all pass",
			deniedFields: nil,
			piiValues: map[string]string{
				"any.field": "value",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exec := &activeExecution{
				id:       "exec_123",
				template: &TaskTemplate{ID: "tpl_1"},
				request: &BlindFillRequest{
					DeniedFields: tt.deniedFields,
				},
				startedAt: time.Now(),
			}

			err := executor.validatePIIValues(exec, tt.piiValues)

			if tt.wantError && err == nil {
				t.Error("expected error for denied field, got nil")
			}
			if !tt.wantError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if tt.wantError && err != nil && !strings.Contains(err.Error(), tt.errorField) {
				t.Errorf("expected error to mention field %s, got: %v", tt.errorField, err)
			}
		})
	}
}

func TestTrustedWorkflow_IsPolicyActive(t *testing.T) {
	engine := &TrustedWorkflowEngine{}

	now := time.Now()
	pastTime := now.Add(-1 * time.Hour)
	futureTime := now.Add(24 * time.Hour)

	tests := []struct {
		name     string
		policy   *TrustedWorkflowPolicy
		expected bool
	}{
		{
			name: "inactive policy",
			policy: &TrustedWorkflowPolicy{
				ID:       "policy_inactive",
				IsActive: false,
			},
			expected: false,
		},
		{
			name: "revoked policy",
			policy: &TrustedWorkflowPolicy{
				ID:        "policy_revoked",
				IsActive:  true,
				RevokedAt: &pastTime,
			},
			expected: false,
		},
		{
			name: "expired policy",
			policy: &TrustedWorkflowPolicy{
				ID:        "policy_expired",
				IsActive:  true,
				ExpiresAt: &pastTime,
			},
			expected: false,
		},
		{
			name: "max executions exceeded",
			policy: &TrustedWorkflowPolicy{
				ID:             "policy_max",
				IsActive:       true,
				MaxExecutions:  5,
				ExecutionCount: 5,
			},
			expected: false,
		},
		{
			name: "requires reconfirmation",
			policy: &TrustedWorkflowPolicy{
				ID:                         "policy_reconfirm",
				IsActive:                   true,
				RequireReconfirmationAfter: 3,
				ExecutionCount:             5,
				LastReconfirmedAt:          &pastTime,
			},
			expected: false,
		},
		{
			name: "valid active policy",
			policy: &TrustedWorkflowPolicy{
				ID:             "policy_valid",
				IsActive:       true,
				ExpiresAt:      &futureTime,
				MaxExecutions:  10,
				ExecutionCount: 2,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.isPolicyActive(tt.policy, now)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
