package skills

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// WebDAVResponse represents WebDAV PROPFIND response
type WebDAVResponse struct {
	XMLName xml.Name `xml:"D:multistatus"`
	Status  string   `xml:"D:response"`
	Prop    struct {
		DisplayName   string `xml:"D:displayname"`
		ContentLength int64  `xml:"D:getcontentlength"`
		ContentType   string `xml:"D:getcontenttype"`
		IsCollection  bool   `xml:"D:collection"`
	} `xml:"D:prop"`
	Href string `xml:"D:href"`
}

// WebDAVListResult represents the result of listing WebDAV contents
type WebDAVListResult struct {
	URL     string                 `json:"url"`
	Entries []WebDAVListItem       `json:"entries"`
	Total   int                    `json:"total"`
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// WebDAVListItem represents a single item in WebDAV listing
type WebDAVListItem struct {
	Name          string `json:"name"`
	IsDirectory   bool   `json:"is_directory"`
	ContentLength int64  `json:"content_length,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
	Href          string `json:"href"`
}

// WebDAVDownloadResult represents the result of downloading from WebDAV
type WebDAVDownloadResult struct {
	URL         string                 `json:"url"`
	Content     []byte                 `json:"content"`
	Size        int64                  `json:"size"`
	Success     bool                   `json:"success"`
	Message     string                 `json:"message,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WebDAVUploadResult represents the result of uploading to WebDAV
type WebDAVUploadResult struct {
	URL      string                 `json:"url"`
	Success  bool                   `json:"success"`
	Message  string                 `json:"message,omitempty"`
	NewURL   string                 `json:"new_url,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WebDAVDeleteResult represents the result of deleting from WebDAV
type WebDAVDeleteResult struct {
	URL      string                 `json:"url"`
	Success  bool                   `json:"success"`
	Message  string                 `json:"message,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// WebDAVParams represents parameters for WebDAV operations
type WebDAVParams struct {
	URL           string `json:"url"`
	Operation     string `json:"operation"` // "list", "get", "put", "delete"
	Username      string `json:"username,omitempty"`
	Password      string `json:"password,omitempty"`
	Content       []byte `json:"content,omitempty"`
	ContentLength int64  `json:"content_length,omitempty"`
	ContentType   string `json:"content_type,omitempty"`
}

// ExecuteWebDAV executes WebDAV operations
func ExecuteWebDAV(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	webDAVParams, err := parseWebDAVParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid WebDAV parameters: %w", err)
	}

	// Security validation
	if err := ValidateWebDAVParams(webDAVParams); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Perform WebDAV operation
	switch webDAVParams.Operation {
	case "list":
		return executeWebDAVList(ctx, webDAVParams)
	case "get":
		return executeWebDAVGet(ctx, webDAVParams)
	case "put":
		return executeWebDAVPut(ctx, webDAVParams)
	case "delete":
		return executeWebDAVDelete(ctx, webDAVParams)
	default:
		return nil, fmt.Errorf("unsupported operation: %s (supported: list, get, put, delete)", webDAVParams.Operation)
	}
}

// parseWebDAVParams parses WebDAV parameters from input
func parseWebDAVParams(params map[string]interface{}) (*WebDAVParams, error) {
	webDAVParams := &WebDAVParams{}

	// Extract required parameters
	if urlStr, ok := params["url"].(string); ok {
		webDAVParams.URL = urlStr
	} else {
		return nil, fmt.Errorf("url parameter is required and must be a string")
	}

	// Extract operation
	if operation, ok := params["operation"].(string); ok {
		webDAVParams.Operation = strings.ToLower(operation)
	} else {
		return nil, fmt.Errorf("operation parameter is required and must be a string")
	}

	// Extract optional authentication
	if username, ok := params["username"].(string); ok {
		webDAVParams.Username = username
	}

	if password, ok := params["password"].(string); ok {
		webDAVParams.Password = password
	}

	// Extract content for PUT operation
	if content, ok := params["content"].([]byte); ok {
		webDAVParams.Content = content
	}

	if contentLength, ok := params["content_length"].(float64); ok {
		webDAVParams.ContentLength = int64(contentLength)
	}

	if contentType, ok := params["content_type"].(string); ok {
		webDAVParams.ContentType = contentType
	}

	return webDAVParams, nil
}

// executeWebDAVList lists contents of a WebDAV directory
func executeWebDAVList(ctx context.Context, params *WebDAVParams) (*WebDAVListResult, error) {
	// Validate URL using SSRF protection
	validator := NewSSRFValidator()
	if err := validator.ValidateURL(params.URL); err != nil {
		return nil, err
	}

	// Create HTTP client
	client := &http.Client{}

	// Build PROPFIND request
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", params.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create PROPFIND request: %w", err)
	}

	// Set headers
	req.Header.Set("Depth", "1")
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", "application/xml, */*")

	// Add authentication if provided
	if params.Username != "" && params.Password != "" {
		req.SetBasicAuth(params.Username, params.Password)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PROPFIND: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusMultiStatus {
		return nil, fmt.Errorf("PROPFIND failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse response
	var multistatus struct {
		XMLName   xml.Name `xml:"D:multistatus"`
		Responses []struct {
			XMLName xml.Name `xml:"D:response"`
			Href    string   `xml:"D:href"`
			Prop    struct {
				DisplayName   string `xml:"D:displayname"`
				ContentLength int64  `xml:"D:getcontentlength"`
				ContentType   string `xml:"D:getcontenttype"`
				IsCollection  bool   `xml:"D:collection"`
			} `xml:"D:prop"`
		} `xml:"D:response"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&multistatus); err != nil {
		return nil, fmt.Errorf("failed to parse WebDAV response: %w", err)
	}

	// Extract entries
	entries := make([]WebDAVListItem, 0, len(multistatus.Responses))
	for _, response := range multistatus.Responses {
		href := strings.TrimSuffix(response.Href, "/")
		name := filepath.Base(href)

		entries = append(entries, WebDAVListItem{
			Name:          name,
			IsDirectory:   response.Prop.IsCollection,
			ContentLength: response.Prop.ContentLength,
			ContentType:   response.Prop.ContentType,
			Href:          href,
		})
	}

	result := &WebDAVListResult{
		URL:     params.URL,
		Entries: entries,
		Total:   len(entries),
		Success: true,
		Message: fmt.Sprintf("Successfully listed %d items", len(entries)),
		Details: map[string]interface{}{
			"url":         params.URL,
			"total_items": len(entries),
			"operation":   "list",
		},
	}

	return result, nil
}

// executeWebDAVGet downloads content from WebDAV
func executeWebDAVGet(ctx context.Context, params *WebDAVParams) (*WebDAVDownloadResult, error) {
	// Validate URL using SSRF protection
	validator := NewSSRFValidator()
	if err := validator.ValidateURL(params.URL); err != nil {
		return nil, err
	}

	// Create HTTP client
	client := &http.Client{}

	// Build GET request
	req, err := http.NewRequestWithContext(ctx, "GET", params.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	// Add authentication if provided
	if params.Username != "" && params.Password != "" {
		req.SetBasicAuth(params.Username, params.Password)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute GET: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	result := &WebDAVDownloadResult{
		URL:         params.URL,
		Content:     content,
		Size:        int64(len(content)),
		Success:     true,
		ContentType: resp.Header.Get("Content-Type"),
		Metadata: map[string]interface{}{
			"url":          params.URL,
			"size_bytes":   len(content),
			"content_type": resp.Header.Get("Content-Type"),
			"operation":    "get",
		},
	}

	return result, nil
}

// executeWebDAVPut uploads content to WebDAV
func executeWebDAVPut(ctx context.Context, params *WebDAVParams) (*WebDAVUploadResult, error) {
	// Validate URL using SSRF protection
	validator := NewSSRFValidator()
	if err := validator.ValidateURL(params.URL); err != nil {
		return nil, err
	}

	// Create HTTP client
	client := &http.Client{}

	// Prepare content
	bodyReader := io.NopCloser(strings.NewReader(string(params.Content)))

	// Build PUT request
	req, err := http.NewRequestWithContext(ctx, "PUT", params.URL, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create PUT request: %w", err)
	}

	// Set content type if provided
	if params.ContentType != "" {
		req.Header.Set("Content-Type", params.ContentType)
	}

	// Set content length
	if params.ContentLength > 0 {
		req.ContentLength = params.ContentLength
	}

	// Add authentication if provided
	if params.Username != "" && params.Password != "" {
		req.SetBasicAuth(params.Username, params.Password)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute PUT: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("PUT failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Get new URL from Location header if available
	newURL := params.URL
	if location := resp.Header.Get("Location"); location != "" {
		// Resolve relative location
		if parsedURL, err := url.Parse(location); err == nil && parsedURL.IsAbs() {
			newURL = location
		} else {
			// Treat as relative to original URL
			if parsedURL, err := url.Parse(params.URL); err == nil {
				parsedURL.Path = filepath.Join(filepath.Dir(parsedURL.Path), location)
				newURL = parsedURL.String()
			}
		}
	}

	result := &WebDAVUploadResult{
		URL:     params.URL,
		Success: true,
		Message: "Successfully uploaded content",
		NewURL:  newURL,
		Metadata: map[string]interface{}{
			"original_url": params.URL,
			"new_url":      newURL,
			"content_size": len(params.Content),
			"operation":    "put",
		},
	}

	return result, nil
}

// executeWebDAVDelete deletes content from WebDAV
func executeWebDAVDelete(ctx context.Context, params *WebDAVParams) (*WebDAVDeleteResult, error) {
	// Validate URL using SSRF protection
	validator := NewSSRFValidator()
	if err := validator.ValidateURL(params.URL); err != nil {
		return nil, err
	}

	// Create HTTP client
	client := &http.Client{}

	// Build DELETE request
	req, err := http.NewRequestWithContext(ctx, "DELETE", params.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DELETE request: %w", err)
	}

	// Add authentication if provided
	if params.Username != "" && params.Password != "" {
		req.SetBasicAuth(params.Username, params.Password)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute DELETE: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return nil, fmt.Errorf("DELETE failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	result := &WebDAVDeleteResult{
		URL:     params.URL,
		Success: true,
		Message: "Successfully deleted item",
		Metadata: map[string]interface{}{
			"url":       params.URL,
			"operation": "delete",
		},
	}

	return result, nil
}

// ValidateWebDAVParams validates WebDAV parameters for security
func ValidateWebDAVParams(params *WebDAVParams) error {
	// Validate URL is not empty
	if params.URL == "" {
		return fmt.Errorf("url parameter is required")
	}

	// Validate operation is one of the supported operations
	supportedOperations := []string{"list", "get", "put", "delete"}
	for _, op := range supportedOperations {
		if params.Operation == op {
			break
		}
	}
	return fmt.Errorf("operation must be one of: %s", strings.Join(supportedOperations, ", "))

	// For PUT operation, validate content is provided
	if params.Operation == "put" {
		if len(params.Content) == 0 {
			return fmt.Errorf("content is required for PUT operation")
		}

		if params.ContentLength <= 0 {
			return fmt.Errorf("content_length is required for PUT operation")
		}

		if params.ContentLength != int64(len(params.Content)) {
			return fmt.Errorf("content_length (%d) does not match actual content length (%d)",
				params.ContentLength, len(params.Content))
		}
	}

	// Validate content length if provided
	if params.ContentLength > 0 && params.ContentLength > 100*1024*1024 { // 100MB absolute maximum
		return fmt.Errorf("content_length cannot exceed 100MB")
	}

	return nil
}
