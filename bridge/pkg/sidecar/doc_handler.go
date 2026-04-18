// Package sidecar provides the doc_query bridge-local handler that queries
// document collections through the sidecar gRPC service.
package sidecar

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/armorclaw/bridge/pkg/capability"
)

// DocQueryInput wraps a DocumentRef with a query string for the doc_query handler.
// Validation delegates to capability.DocumentRef.Validate() plus query check.
type DocQueryInput struct {
	CollectionID string   `json:"collection_id"`
	ChunkIDs     []string `json:"chunk_ids,omitempty"`
	Query        string   `json:"query"`
}

// NewDocQueryHandler returns a bridge-local handler function matching
// secretary.BridgeLocalHandler. It routes through ProcessDocument with
// operation="query_documents" since no dedicated QueryDocuments RPC exists yet.
// The caller registers the returned function as "doc_query" with BridgeLocalRegistry.
func NewDocQueryHandler(client *Client, logger *slog.Logger) func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
	if logger == nil {
		logger = slog.Default()
	}

	return func(ctx context.Context, config json.RawMessage) (json.RawMessage, error) {
		var input DocQueryInput
		if err := json.Unmarshal(config, &input); err != nil {
			return nil, fmt.Errorf("doc_query: parse input: %w", err)
		}

		if err := ValidateDocQueryInput(&input); err != nil {
			return nil, err
		}

		opParams := map[string]string{
			"collection_id": input.CollectionID,
			"query":         input.Query,
		}
		if len(input.ChunkIDs) > 0 {
			chunkJSON, err := json.Marshal(input.ChunkIDs)
			if err != nil {
				return nil, fmt.Errorf("doc_query: marshal chunk_ids: %w", err)
			}
			opParams["chunk_ids"] = string(chunkJSON)
		}

		req := &ProcessDocumentRequest{
			Operation:       "query_documents",
			InputUri:        input.CollectionID,
			OperationParams: opParams,
		}

		resp, err := client.ProcessDocument(ctx, req)
		if err != nil {
			logger.Warn("doc_query: sidecar call failed",
				"collection_id", input.CollectionID,
				"error", err,
			)
			return nil, fmt.Errorf("doc_query: sidecar query failed: %w", err)
		}

		result := &capability.ExtractedChunkSet{
			Chunks: []string{},
		}

		if len(resp.GetOutputContent()) > 0 {
			if unmarshalErr := json.Unmarshal(resp.GetOutputContent(), result); unmarshalErr != nil {
				result.Chunks = []string{string(resp.GetOutputContent())}
			}
		}

		if result.Summary == "" {
			if s, ok := resp.GetMetadata()["summary"]; ok {
				result.Summary = s
			}
		}

		if err := result.Validate(); err != nil {
			return nil, fmt.Errorf("doc_query: invalid result: %w", err)
		}

		raw, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("doc_query: marshal result: %w", err)
		}

		logger.Info("doc_query: completed",
			"collection_id", input.CollectionID,
			"chunks_returned", len(result.Chunks),
		)

		return raw, nil
	}
}

// ValidateDocQueryInput validates a DocQueryInput using capability.DocumentRef.
func ValidateDocQueryInput(input *DocQueryInput) error {
	if input == nil {
		return fmt.Errorf("doc_query: input is nil")
	}

	docRef := &capability.DocumentRef{
		CollectionID: input.CollectionID,
		ChunkIDs:     input.ChunkIDs,
	}
	if err := docRef.Validate(); err != nil {
		return fmt.Errorf("doc_query: %w", err)
	}

	if input.Query == "" {
		return fmt.Errorf("doc_query: query text is required")
	}

	return nil
}
