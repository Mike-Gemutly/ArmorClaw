package wizard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCatwalkClient_FetchProviders_Success(t *testing.T) {
	expectedProviders := []CatwalkProvider{
		{
			ID:      "openai",
			Name:    "OpenAI",
			BaseURL: "https://api.openai.com/v1",
			APIType: "openai",
			Models: []CatwalkModel{
				{ID: "gpt-4o", Name: "GPT-4o", ContextSize: 128000},
				{ID: "gpt-4o-mini", Name: "GPT-4o Mini", ContextSize: 128000},
			},
		},
		{
			ID:      "anthropic",
			Name:    "Anthropic",
			BaseURL: "https://api.anthropic.com/v1",
			APIType: "anthropic",
			Models: []CatwalkModel{
				{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", ContextSize: 200000},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/providers" {
			t.Errorf("expected path /v2/providers, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(CatwalkResponse{Providers: expectedProviders})
	}))
	defer server.Close()

	client := NewCatwalkClient(server.URL)
	providers, err := client.FetchProviders()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}

	if providers[0].ID != "openai" {
		t.Errorf("expected provider ID 'openai', got %s", providers[0].ID)
	}

	if len(providers[0].Models) != 2 {
		t.Errorf("expected 2 models for OpenAI, got %d", len(providers[0].Models))
	}

	if providers[0].Models[0].ID != "gpt-4o" {
		t.Errorf("expected model ID 'gpt-4o', got %s", providers[0].Models[0].ID)
	}
}

func TestCatwalkClient_FetchProviders_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewCatwalkClient(server.URL)
	_, err := client.FetchProviders()

	if err == nil {
		t.Fatal("expected error for non-200 status")
	}
}

func TestCatwalkClient_FetchProviders_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewCatwalkClient(server.URL)
	_, err := client.FetchProviders()

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestCatwalkClient_IsAvailable_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("expected path /healthz, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewCatwalkClient(server.URL)
	available := client.IsAvailable()

	if !available {
		t.Error("expected IsAvailable to return true")
	}
}

func TestCatwalkClient_IsAvailable_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewCatwalkClient(server.URL)
	available := client.IsAvailable()

	if available {
		t.Error("expected IsAvailable to return false")
	}
}

func TestCatwalkClient_IsAvailable_ConnectionError(t *testing.T) {
	client := NewCatwalkClient("http://localhost:9999")
	available := client.IsAvailable()

	if available {
		t.Error("expected IsAvailable to return false for unreachable server")
	}
}

func TestProvidersFromCatwalk(t *testing.T) {
	catwalkProviders := []CatwalkProvider{
		{
			ID:      "openai",
			Name:    "OpenAI",
			BaseURL: "https://api.openai.com/v1",
		},
		{
			ID:      "anthropic",
			Name:    "Anthropic",
			BaseURL: "https://api.anthropic.com/v1",
		},
	}

	result := ProvidersFromCatwalk(catwalkProviders)

	if len(result) != 2 {
		t.Errorf("expected 2 options, got %d", len(result))
	}

	if result[0].Key != "openai" {
		t.Errorf("expected key 'openai', got %s", result[0].Key)
	}

	if result[0].Name != "OpenAI" {
		t.Errorf("expected name 'OpenAI', got %s", result[0].Name)
	}

	if result[0].BaseURL != "https://api.openai.com/v1" {
		t.Errorf("expected base URL, got %s", result[0].BaseURL)
	}
}

func TestProvidersFromCatwalk_Empty(t *testing.T) {
	result := ProvidersFromCatwalk([]CatwalkProvider{})

	if len(result) != 0 {
		t.Errorf("expected 0 options, got %d", len(result))
	}
}

func TestGetProviderOptions_WithCatwalk(t *testing.T) {
	expectedProviders := []CatwalkProvider{
		{
			ID:      "deepseek",
			Name:    "DeepSeek",
			BaseURL: "https://api.deepseek.com/v1",
			Models: []CatwalkModel{
				{ID: "deepseek-chat", Name: "DeepSeek Chat"},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/healthz":
			w.WriteHeader(http.StatusOK)
		case "/v2/providers":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(CatwalkResponse{Providers: expectedProviders})
		}
	}))
	defer server.Close()

	originalCatwalkURL := "http://localhost:8080"

	oldFunc := func(baseURL string) *CatwalkClient {
		return NewCatwalkClient(server.URL)
	}

	_ = oldFunc
	_ = originalCatwalkURL

	client := NewCatwalkClient(server.URL)

	if !client.IsAvailable() {
		t.Fatal("expected Catwalk to be available")
	}

	providers, err := client.FetchProviders()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(providers))
	}

	options := ProvidersFromCatwalk(providers)
	if len(options) != 1 {
		t.Errorf("expected 1 option, got %d", len(options))
	}

	if options[0].Key != "deepseek" {
		t.Errorf("expected key 'deepseek', got %s", options[0].Key)
	}
}
