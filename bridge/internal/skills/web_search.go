package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSearchResult represents a web search result
type WebSearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	DisplayURL  string `json:"display_url"`
	Source      string `json:"source"`
	Position    int    `json:"position"`
}

// WebSearchResponse represents a complete web search response
type WebSearchResponse struct {
	Query        string           `json:"query"`
	Engine       string           `json:"engine"`
	Results      []WebSearchResult `json:"results"`
	TotalResults int             `json:"total_results,omitempty"`
	SearchTime   float64         `json:"search_time,omitempty"`
	Page         int             `json:"page"`
	PerPage      int             `json:"per_page"`
}

// WebSearchParams represents parameters for web search
type WebSearchParams struct {
	Query      string `json:"query"`
	Engine     string `json:"engine,omitempty"`
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"per_page,omitempty"`
	SafeSearch bool   `json:"safe_search,omitempty"`
	Language   string `json:"language,omitempty"`
	Region     string `json:"region,omitempty"`
}

// ExecuteWebSearch performs a web search using the specified engine
func ExecuteWebSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	searchParams, err := parseWebSearchParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid search parameters: %w", err)
	}

	// Validate query
	if strings.TrimSpace(searchParams.Query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	// Set defaults
	if searchParams.Engine == "" {
		searchParams.Engine = "duckduckgo" // Default to DuckDuckGo
	}
	if searchParams.Page <= 0 {
		searchParams.Page = 1
	}
	if searchParams.PerPage <= 0 || searchParams.PerPage > 20 {
		searchParams.PerPage = 10
	}

	// Execute search based on engine
	switch searchParams.Engine {
	case "google":
		return executeGoogleSearch(ctx, searchParams)
	case "bing":
		return executeBingSearch(ctx, searchParams)
	case "duckduckgo":
		return executeDuckDuckGoSearch(ctx, searchParams)
	default:
		return nil, fmt.Errorf("unsupported search engine: %s", searchParams.Engine)
	}
}

// parseWebSearchParams parses web search parameters from input
func parseWebSearchParams(params map[string]interface{}) (*WebSearchParams, error) {
	searchParams := &WebSearchParams{}

	// Extract required parameters
	if query, ok := params["query"].(string); ok {
		searchParams.Query = query
	} else {
		return nil, fmt.Errorf("query parameter is required and must be a string")
	}

	// Extract optional parameters
	if engine, ok := params["engine"].(string); ok {
		searchParams.Engine = strings.ToLower(engine)
	}
	
	if page, ok := params["page"].(float64); ok {
		searchParams.Page = int(page)
	}
	
	if perPage, ok := params["per_page"].(float64); ok {
		searchParams.PerPage = int(perPage)
	}
	
	if safeSearch, ok := params["safe_search"].(bool); ok {
		searchParams.SafeSearch = safeSearch
	}
	
	if language, ok := params["language"].(string); ok {
		searchParams.Language = language
	}
	
	if region, ok := params["region"].(string); ok {
		searchParams.Region = region
	}

	return searchParams, nil
}

// executeGoogleSearch performs a Google search
func executeGoogleSearch(ctx context.Context, params *WebSearchParams) (*WebSearchResponse, error) {
	// Note: This is a mock implementation for Phase 2
	// In production, you would use Google's Custom Search JSON API
	// This requires an API key and search engine ID
	
	// For now, return mock data to demonstrate the structure
	results := []WebSearchResult{
		{
			Title:      fmt.Sprintf("Google search results for: %s", params.Query),
			URL:        "https://www.google.com/search?q=" + url.QueryEscape(params.Query),
			Snippet:    fmt.Sprintf("Mock Google search results for query: %s", params.Query),
			DisplayURL: "www.google.com",
			Source:     "Google",
			Position:   1,
		},
	}

	return &WebSearchResponse{
		Query:        params.Query,
		Engine:       "google",
		Results:      results,
		TotalResults: 1,
		SearchTime:   0.45,
		Page:         params.Page,
		PerPage:      params.PerPage,
	}, nil
}

// executeBingSearch performs a Bing search
func executeBingSearch(ctx context.Context, params *WebSearchParams) (*WebSearchResponse, error) {
	// Note: This is a mock implementation for Phase 2
	// In production, you would use Bing Web Search API
	// This requires an API key from Azure Cognitive Services
	
	results := []WebSearchResult{
		{
			Title:      fmt.Sprintf("Bing search results for: %s", params.Query),
			URL:        "https://www.bing.com/search?q=" + url.QueryEscape(params.Query),
			Snippet:    fmt.Sprintf("Mock Bing search results for query: %s", params.Query),
			DisplayURL: "www.bing.com",
			Source:     "Bing",
			Position:   1,
		},
	}

	return &WebSearchResponse{
		Query:        params.Query,
		Engine:       "bing",
		Results:      results,
		TotalResults: 1,
		SearchTime:   0.32,
		Page:         params.Page,
		PerPage:      params.PerPage,
	}, nil
}

// executeDuckDuckGoSearch performs a DuckDuckGo search
func executeDuckDuckGoSearch(ctx context.Context, params *WebSearchParams) (*WebSearchResponse, error) {
	// DuckDuckGo provides a free, no-API-key-required instant answer API
	// We'll use this for Phase 2 implementation
	
	// Build the DuckDuckGo API URL
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_redirect=1", url.QueryEscape(params.Query))
	
	// Create HTTP request with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ArmorClaw/1.0)")
	
	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute DuckDuckGo search: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("DuckDuckGo API returned status: %d", resp.StatusCode)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	
	// Parse DuckDuckGo response
	var ddgResponse struct {
		AbstractText     string        `json:"AbstractText"`
		AbstractSource   string        `json:"AbstractSource"`
		AbstractURL      string        `json:"AbstractURL"`
		RelatedTopics    []interface{} `json:"RelatedTopics"`
		Results          []interface{} `json:"Results"`
		Heading          string        `json:"Heading"`
	}
	
	if err := json.Unmarshal(body, &ddgResponse); err != nil {
		return nil, fmt.Errorf("failed to parse DuckDuckGo response: %w", err)
	}
	
	// Convert DuckDuckGo response to our format
	results := make([]WebSearchResult, 0)
	
	// Add abstract result if available
	if ddgResponse.AbstractText != "" {
		results = append(results, WebSearchResult{
			Title:      ddgResponse.Heading,
			URL:        ddgResponse.AbstractURL,
			Snippet:    ddgResponse.AbstractText,
			DisplayURL: ddgResponse.AbstractSource,
			Source:     "DuckDuckGo",
			Position:   1,
		})
	}
	
	// Add related topics
	for i, topic := range ddgResponse.RelatedTopics {
		if topicMap, ok := topic.(map[string]interface{}); ok {
			if text, ok := topicMap["Text"].(string); ok {
				if firstURL, ok := topicMap["FirstURL"].(string); ok {
					results = append(results, WebSearchResult{
						Title:      fmt.Sprintf("Related Topic %d", i+1),
						URL:        firstURL,
						Snippet:    text,
						DisplayURL: extractDomain(firstURL),
						Source:     "DuckDuckGo",
						Position:   i + 2,
					})
				}
			}
		}
	}
	
	// Add results
	for i, result := range ddgResponse.Results {
		if resultMap, ok := result.(map[string]interface{}); ok {
			if text, ok := resultMap["Text"].(string); ok {
				if firstURL, ok := resultMap["FirstURL"].(string); ok {
					results = append(results, WebSearchResult{
						Title:      fmt.Sprintf("Result %d", i+1),
						URL:        firstURL,
						Snippet:    text,
						DisplayURL: extractDomain(firstURL),
						Source:     "DuckDuckGo",
						Position:   len(results) + 1,
					})
				}
			}
		}
	}
	
	return &WebSearchResponse{
		Query:        params.Query,
		Engine:       "duckduckgo",
		Results:      results,
		TotalResults: len(results),
		SearchTime:   0.25,
		Page:         params.Page,
		PerPage:      params.PerPage,
	}, nil
}

// extractDomain extracts domain from URL
func extractDomain(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return urlStr
	}
	return u.Hostname()
}

// ValidateWebSearchParams validates web search parameters
func ValidateWebSearchParams(params map[string]interface{}) error {
	// Check required query parameter
	if query, ok := params["query"].(string); !ok || strings.TrimSpace(query) == "" {
		return fmt.Errorf("query parameter is required and must be a non-empty string")
	}
	
	// Validate engine if provided
	if engine, ok := params["engine"].(string); ok {
		lowerEngine := strings.ToLower(engine)
		if lowerEngine != "google" && lowerEngine != "bing" && lowerEngine != "duckduckgo" {
			return fmt.Errorf("unsupported search engine: %s (supported: google, bing, duckduckgo)", engine)
		}
	}
	
	// Validate page number
	if page, ok := params["page"].(float64); ok {
		if page < 1 || page > 100 {
			return fmt.Errorf("page number must be between 1 and 100")
		}
	}
	
	// Validate per page
	if perPage, ok := params["per_page"].(float64); ok {
		if perPage < 1 || perPage > 20 {
			return fmt.Errorf("per_page must be between 1 and 20")
		}
	}
	
	return nil
}