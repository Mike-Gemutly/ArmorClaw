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

type DocQueryInput struct {
	CollectionID string   `json:"collection_id"`
	ChunkIDs     []string `json:"chunk_ids,omitempty"`
	Query        string   `json:"query"`
}

type DocQueryOutput struct {
	Chunks  []string `json:"chunks"`
	Summary string   `json:"summary,omitempty"`
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

		if input.CollectionID == "" {
			return nil, fmt.Errorf("doc_query: collection_id is required")
		}
		if input.Query == "" {
			return nil, fmt.Errorf("doc_query: query text is required")
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

		output := DocQueryOutput{
			Chunks: []string{},
		}

		if len(resp.GetOutputContent()) > 0 {
			if unmarshalErr := json.Unmarshal(resp.GetOutputContent(), &output); unmarshalErr != nil {
				output.Chunks = []string{string(resp.GetOutputContent())}
			}
		}

		if output.Summary == "" {
			if s, ok := resp.GetMetadata()["summary"]; ok {
				output.Summary = s
			}
		}

		result, err := json.Marshal(output)
		if err != nil {
			return nil, fmt.Errorf("doc_query: marshal result: %w", err)
		}

		logger.Info("doc_query: completed",
			"collection_id", input.CollectionID,
			"chunks_returned", len(output.Chunks),
		)

		return result, nil
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
