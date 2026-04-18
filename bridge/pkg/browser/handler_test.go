package browser

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlerName(t *testing.T) {
	assert.Equal(t, "browser_execute", HandlerName)
}

func newTestServer(handler http.HandlerFunc) *Client {
	srv := httptest.NewServer(handler)
	return NewClient(srv.URL)
}

func TestHandler_NavigateAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ServiceResponse{
			Success: true,
			Data:    map[string]interface{}{"title": "Test Page"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{
		"url":    "https://example.com",
		"action": "navigate",
	})

	out, err := h(context.Background(), input)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, "https://example.com", result["url"])
	assert.Equal(t, "Test Page", result["title"])
}

func TestHandler_FillAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ServiceResponse{Success: true})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]any{
		"url":         "https://example.com/form",
		"action":      "fill",
		"form_fields": []string{"#name", "#email"},
	})

	out, err := h(context.Background(), input)
	require.NoError(t, err)

	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &result))
	assert.Equal(t, "https://example.com/form", result["url"])
}

func TestHandler_ExtractAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ServiceResponse{
			Success: true,
			Data:    map[string]interface{}{"price": "$99.99"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]any{
		"url":         "https://shop.example.com",
		"action":      "extract",
		"form_fields": []string{".price"},
	})

	out, err := h(context.Background(), input)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestHandler_ScreenshotAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ServiceResponse{
			Success:    true,
			Screenshot: "base64imagedata",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{
		"url":    "https://example.com",
		"action": "screenshot",
	})

	out, err := h(context.Background(), input)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestHandler_WorkflowAction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ServiceWorkflowResponse{Success: true})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]any{
		"url":         "https://example.com",
		"action":      "workflow",
		"form_fields": []string{"fill", "submit"},
	})

	out, err := h(context.Background(), input)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestHandler_InvalidJSON(t *testing.T) {
	client := NewClient("http://unused")
	h := Handler(client)

	_, err := h(context.Background(), []byte("{invalid"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "browser_execute: parse config")
}

func TestHandler_MissingAction(t *testing.T) {
	client := NewClient("http://unused")
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{"url": "https://example.com"})
	_, err := h(context.Background(), input)
	require.Error(t, err)
}

func TestHandler_UnsupportedAction(t *testing.T) {
	client := NewClient("http://unused")
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{
		"url":    "https://example.com",
		"action": "fly_to_moon",
	})
	_, err := h(context.Background(), input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported browser action")
}

func TestHandler_NestedIntent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ServiceResponse{Success: true})
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]any{
		"intent": map[string]any{
			"url":    "https://example.com",
			"action": "navigate",
		},
	})

	out, err := h(context.Background(), input)
	require.NoError(t, err)
	assert.NotNil(t, out)
}

func TestHandler_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	client := NewClient(srv.URL)
	h := Handler(client)

	input, _ := json.Marshal(map[string]string{
		"url":    "https://example.com",
		"action": "navigate",
	})

	_, err := h(context.Background(), input)
	require.Error(t, err)
}

func TestServiceResponseToBrowserResult_WithTitle(t *testing.T) {
	resp := &ServiceResponse{
		Data: map[string]interface{}{"title": "My Page"},
	}
	result := serviceResponseToBrowserResult("https://example.com", resp)
	assert.Equal(t, "https://example.com", result.URL)
	assert.Equal(t, "My Page", result.Title)
}

func TestServiceResponseToBrowserResult_NilResponse(t *testing.T) {
	result := serviceResponseToBrowserResult("https://example.com", nil)
	assert.Equal(t, "https://example.com", result.URL)
	assert.Empty(t, result.Title)
}

func TestServiceResponseToBrowserResult_NoTitle(t *testing.T) {
	resp := &ServiceResponse{Data: map[string]interface{}{"other": "value"}}
	result := serviceResponseToBrowserResult("https://example.com", resp)
	assert.Empty(t, result.Title)
}
