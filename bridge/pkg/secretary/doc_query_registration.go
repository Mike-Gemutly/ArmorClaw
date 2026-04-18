package secretary

import (
	"log/slog"

	"github.com/armorclaw/bridge/pkg/sidecar"
)

// RegisterDocQueryHandler creates a doc_query handler using the sidecar client
// and registers it with the given BridgeLocalRegistry.
func RegisterDocQueryHandler(registry *BridgeLocalRegistry, client *sidecar.Client, logger *slog.Logger) {
	handler := sidecar.NewDocQueryHandler(client, logger)
	registry.Register("doc_query", handler)
}
