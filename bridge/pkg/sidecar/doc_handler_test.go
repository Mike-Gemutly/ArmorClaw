package sidecar

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateDocQueryInput_Valid(t *testing.T) {
	input := &DocQueryInput{
		CollectionID: "team-abc-docs",
		Query:        "find invoice amounts",
	}
	err := ValidateDocQueryInput(input)
	assert.NoError(t, err)
}

func TestValidateDocQueryInput_WithChunkIDs(t *testing.T) {
	input := &DocQueryInput{
		CollectionID: "team-abc-docs",
		ChunkIDs:     []string{"chunk-1", "chunk-2"},
		Query:        "find invoice amounts",
	}
	err := ValidateDocQueryInput(input)
	assert.NoError(t, err)
}

func TestValidateDocQueryInput_EmptyCollectionID(t *testing.T) {
	input := &DocQueryInput{
		CollectionID: "",
		Query:        "find something",
	}
	err := ValidateDocQueryInput(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collection_id")
}

func TestValidateDocQueryInput_EmptyQuery(t *testing.T) {
	input := &DocQueryInput{
		CollectionID: "team-abc-docs",
		Query:        "",
	}
	err := ValidateDocQueryInput(input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query text is required")
}

func TestValidateDocQueryInput_NilInput(t *testing.T) {
	err := ValidateDocQueryInput(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "input is nil")
}

func TestValidateDocQueryInput_AllEmpty(t *testing.T) {
	input := &DocQueryInput{}
	err := ValidateDocQueryInput(input)
	require.Error(t, err)
}

func TestNewDocQueryHandler_InvalidJSON(t *testing.T) {
	handler := NewDocQueryHandler(nil, nil)
	_, err := handler(context.Background(), []byte("{bad json"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "doc_query: parse input")
}

func TestNewDocQueryHandler_MissingCollectionID(t *testing.T) {
	handler := NewDocQueryHandler(nil, nil)
	input, _ := json.Marshal(DocQueryInput{Query: "test"})
	_, err := handler(context.Background(), input)
	require.Error(t, err)
}

func TestNewDocQueryHandler_MissingQuery(t *testing.T) {
	handler := NewDocQueryHandler(nil, nil)
	input, _ := json.Marshal(DocQueryInput{CollectionID: "col-1"})
	_, err := handler(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query text is required")
}

func TestNewDocQueryHandler_NilLogger(t *testing.T) {
	handler := NewDocQueryHandler(nil, nil)
	assert.NotNil(t, handler, "should not panic with nil logger")
}

func TestDocQueryInput_JSONRoundTrip(t *testing.T) {
	original := DocQueryInput{
		CollectionID: "team-docs",
		ChunkIDs:     []string{"c1", "c2"},
		Query:        "search terms",
	}
	raw, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded DocQueryInput
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, original.CollectionID, decoded.CollectionID)
	assert.Equal(t, original.Query, decoded.Query)
	assert.Equal(t, original.ChunkIDs, decoded.ChunkIDs)
}
