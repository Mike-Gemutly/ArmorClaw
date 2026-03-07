package wizard

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CatwalkProvider struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	BaseURL       string         `json:"base_url"`
	APIType       string         `json:"api_type"`
	AuthType      string         `json:"auth_type"`
	Models        []CatwalkModel `json:"models,omitempty"`
	Requires      []string       `json:"requires,omitempty"`
	Configuration map[string]any `json:"configuration,omitempty"`
}

type CatwalkModel struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	ContextSize int      `json:"context_size,omitempty"`
	Supports    []string `json:"supports,omitempty"`
}

type CatwalkResponse struct {
	Providers []CatwalkProvider `json:"providers"`
}

type CatwalkClient struct {
	baseURL string
	client  *http.Client
}

func NewCatwalkClient(baseURL string) *CatwalkClient {
	return &CatwalkClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:          10,
				IdleConnTimeout:       30 * time.Second,
				TLSHandshakeTimeout:   10 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
	}
}

func (c *CatwalkClient) FetchProviders() ([]CatwalkProvider, error) {
	url := fmt.Sprintf("%s/v2/providers", c.baseURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch providers from Catwalk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Catwalk returned status %d", resp.StatusCode)
	}

	var result CatwalkResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse Catwalk response: %w", err)
	}

	return result.Providers, nil
}

func (c *CatwalkClient) IsAvailable() bool {
	url := fmt.Sprintf("%s/healthz", c.baseURL)

	resp, err := c.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

func ProvidersFromCatwalk(providers []CatwalkProvider) []apiProviderOption {
	result := make([]apiProviderOption, 0, len(providers))

	for _, p := range providers {
		result = append(result, apiProviderOption{
			Name:    p.Name,
			Key:     p.ID,
			BaseURL: p.BaseURL,
		})
	}

	return result
}
