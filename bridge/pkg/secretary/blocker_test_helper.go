package secretary

import "sync"

// PendingBlockersMap returns the package-level pendingBlockers sync.Map
// for external test packages that need to register blocker channels.
func PendingBlockersMap() *sync.Map {
	return &pendingBlockers
}
