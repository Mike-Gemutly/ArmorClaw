package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/armorclaw/bridge/pkg/trust"
)

// handleAccountDelete handles account.delete RPC method.
// Deactivates the Matrix account, revokes sessions, schedules cleanup.
// Requires hardening to be complete (DelegationReady).
func (s *Server) handleAccountDelete(ctx context.Context, req *Request) (interface{}, *ErrorObj) {
	var params struct {
		Password string `json:"password"`
		Erase    bool   `json:"erase"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "invalid parameters: " + err.Error(),
		}
	}

	if params.Password == "" {
		return nil, &ErrorObj{
			Code:    InvalidParams,
			Message: "password is required",
		}
	}

	matrixAdapter, ok := s.matrix.(interface {
		GetUserID() string
		IsLoggedIn() bool
		DeactivateAccount(ctx context.Context, password string, erase bool) error
	})
	if !ok || matrixAdapter == nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Matrix adapter not configured",
		}
	}

	if !matrixAdapter.IsLoggedIn() {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Not authenticated",
		}
	}

	userID := matrixAdapter.GetUserID()
	if userID == "" {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: "Not authenticated",
		}
	}

	if s.hardeningStore != nil {
		ready, err := s.hardeningStore.IsDelegationReady(userID)
		if err != nil {
			return nil, &ErrorObj{
				Code:    InternalError,
				Message: fmt.Sprintf("Failed to check hardening status: %s", err.Error()),
			}
		}
		if !ready {
			return nil, &ErrorObj{
				Code:    InternalError,
				Message: "Account hardening must be complete before deletion",
			}
		}
	}

	if err := matrixAdapter.DeactivateAccount(ctx, params.Password, params.Erase); err != nil {
		return nil, &ErrorObj{
			Code:    InternalError,
			Message: fmt.Sprintf("Account deactivation failed: %s", err.Error()),
		}
	}

	return map[string]interface{}{
		"status":       "deactivated",
		"user_id":      userID,
		"deactivated_at": time.Now().Format(time.RFC3339),
		"erase":        params.Erase,
	}, nil
}

// AccountDeleteChecker wraps the hardening store check for account.delete.
type AccountDeleteChecker struct {
	store trust.Store
}

func NewAccountDeleteChecker(store trust.Store) *AccountDeleteChecker {
	return &AccountDeleteChecker{store: store}
}

func (c *AccountDeleteChecker) IsHardeningComplete(userID string) (bool, error) {
	if c.store == nil {
		return false, fmt.Errorf("hardening store not configured")
	}
	return c.store.IsDelegationReady(userID)
}
