// Package rpc provides delegation gate logic for first-login hardening.
package rpc

import (
	"fmt"

	"github.com/armorclaw/bridge/pkg/trust"
)

// ErrHardeningRequired is returned when hardening steps are not complete
var ErrHardeningRequired = fmt.Errorf("complete security hardening before performing this action")

// RequireDelegationReady checks if the user has completed all mandatory hardening steps
// Returns ErrHardeningRequired if delegation is not ready
// Returns nil if delegation is ready and the action may proceed
func RequireDelegationReady(store trust.Store, userID string) error {
	if store == nil {
		// If store is not configured, allow the action (graceful degradation)
		return nil
	}

	ready, err := store.IsDelegationReady(userID)
	if err != nil {
		// Wrap and return store errors
		return fmt.Errorf("failed to check delegation ready: %w", err)
	}

	if !ready {
		return ErrHardeningRequired
	}

	return nil
}
