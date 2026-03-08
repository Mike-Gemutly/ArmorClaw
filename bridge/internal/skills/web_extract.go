package skills

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// WebExtractParams represents parameters for web content extraction
type WebExtractParams struct {
	URL        string `json:"url"`
	ContentType string `json:"content_type,omitempty"`
	MaxLength  int    `json:"max_length,omitempty"`
	IncludeLinks bool `json:"include_links,omitempty"`
	IncludeImages bool `json:"include_images,omitempty"`
	IncludeTables bool `json:"include_tables,omitempty"`
}

// WebExtractResult represents the result of web content extraction
type WebExtractResult struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	ContentType string            `json:"content_type"`
	Content     string            `json:"content"`
	Links       []WebLink         `json:"links,omitempty"`
	Images      []WebImage        `json:"images,omitempty"`
	Tables      []WebTable        `json:"tables,omitempty"`
	Metadata    map[string]string `json:"metadata"`
	WordCount   int               `json:"word_count"`
	ExtractedAt time.Time         `json:"extracted_at"`
}

// WebLink represents a extracted link
type WebLink struct {
	Text string `json:"text"`
	URL  string `json:"url"`
	Type string `json:"type,omitempty"` // internal, external, etc.
}

// WebImage represents a extracted image
type WebImage struct {
	Src   string `json:"src"`
	Alt   string `json:"alt"`
	Title string `json:"title"`
	Width int    `json:"width,omitempty"`
	Height int   `json:"height,omitempty"`
}

// WebTable represents a extracted table
type WebTable struct {
	Headers []string        `json:"headers"`
	Rows    [][]string      `json:"rows"`
	Caption string          `json:"caption,omitempty"`
}

// ExecuteWebExtract extracts content from a web URL
func ExecuteWebExtract(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	extractParams, err := parseWebExtractParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid extraction parameters: %w", err)
	}

	// Validate URL
	if err := validateExtractURL(extractParams.URL); err != nil {
		return nil, fmt.Errorf("URL validation failed: %w", err)
	}

	// Set defaults
	if extractParams.MaxLength <= 0 || extractParams.MaxLength > 1000000 {
		extractParams.MaxLength = 100000 // 1MB default limit
	}

	// Fetch the web content
	content, contentType, err := fetchWebContent(ctx, extractParams.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch web content: %w", err)
	}

	// Extract content based on type
	switch {
	case strings.Contains(contentType, "html"):
		return extractHTMLContent(content, extractParams)
	case strings.Contains(contentType, "text/plain"):
		return extractTextContent(content, extractParams)
	default:
		// Default to text extraction
		return extractTextContent(content, extractParams)
	}
}

// parseWebExtractParams parses web extraction parameters
func parseWebExtractParams(params map[string]interface{}) (*WebExtractParams, error) {
	extractParams := &WebExtractParams{}

	// Extract required URL parameter
	if urlStr, ok := params["url"].(string); ok {
		extractParams.URL = strings.TrimSpace(urlStr)
	} else {
		return nil, fmt.Errorf("url parameter is required and must be a string")
	}

	// Extract optional parameters
	if contentType, ok := params["content_type"].(string); ok {
		extractParams.ContentType = strings.ToLower(contentType)
	}
	
	if maxLength, ok := params["max_length"].(float64); ok {
		extractParams.MaxLength = int(maxLength)
	}
	
	if includeLinks, ok := params["include_links"].(bool); ok {
		extractParams.IncludeLinks = includeLinks
	}
	
	if includeImages, ok := params["include_images"].(bool); ok {
		extractParams.IncludeImages = includeImages
	}
	
	if includeTables, ok := params["include_tables"].(bool); ok {
		extractParams.IncludeTables = includeTables
	}

	return extractParams, nil
}

// validateExtractURL validates a URL for extraction
func validateExtractURL(urlStr string) error {
	if strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse URL to validate format
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// Check scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTP and HTTPS URLs are supported")
	}

	// Check for localhost/private networks
	if isPrivateNetwork(parsedURL.Hostname()) {
		return fmt.Errorf("private network URLs are not allowed")
	}

	return nil
}

// isPrivateNetwork checks if a hostname is in a private network
func isPrivateNetwork(hostname string) bool {
	// Common private network patterns
	privatePatterns := []string{
		`^localhost$`,
		`^127\.`,
		`^10\.`,
		`^192\.168\.`,
		`^172\.(1[6-9]|2[0-9]|3[0-1])\.`,
		`^::1$`,
		`^fc00:`,
		`^fe80:`,
	}

	for _, pattern := range privatePatterns {
		if matched, _ := regexp.MatchString(pattern, hostname); matched {
			return true
		}
	}

	return false
}

// fetchWebContent fetches content from a URL
func fetchWebContent(ctx context.Context, urlStr string) ([]byte, string, error) {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers to appear like a browser
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; ArmorClaw/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Connection", "keep-alive")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/plain" // Default
	}

	return body, contentType, nil
}

// extractHTMLContent extracts content from HTML
func extractHTMLContent(htmlContent []byte, params *WebExtractParams) (*WebExtractResult, error) {
	// Parse HTML
	doc, err := html.Parse(strings.NewReader(string(htmlContent)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	result := &WebExtractResult{
		URL:         params.URL,
		ContentType: "text/html",
		Metadata:    make(map[string]string),
		ExtractedAt: time.Now(),
	}

	// Extract information from HTML
	var extractor htmlExtractor
	extractor.extractFromNode(doc, result, params)

	// Apply length limit
	if len(result.Content) > params.MaxLength {
		result.Content = result.Content[:params.MaxLength] + "...[truncated]"
	}

	// Count words
	result.WordCount = countWords(result.Content)

	return result, nil
}

// extractTextContent extracts plain text content
func extractTextContent(content []byte, params *WebExtractParams) (*WebExtractResult, error) {
	text := string(content)

	// Apply length limit
	if len(text) > params.MaxLength {
		text = text[:params.MaxLength] + "...[truncated]"
	}

	result := &WebExtractResult{
		URL:         params.URL,
		ContentType: "text/plain",
		Content:     text,
		Metadata:    make(map[string]string),
		WordCount:   countWords(text),
		ExtractedAt: time.Now(),
	}

	return result, nil
}

// htmlExtractor handles HTML content extraction
type htmlExtractor struct {
	currentLink *WebLink
	currentTable *WebTable
	inTable     bool
	inTableHead bool
	inTableRow  bool
	inTableCell bool
}

// extractFromNode recursively extracts content from HTML nodes
func (e *htmlExtractor) extractFromNode(n *html.Node, result *WebExtractResult, params *WebExtractParams) {
	switch n.Type {
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			e.extractFromNode(c, result, params)
		}

	case html.ElementNode:
		switch n.Data {
		case "title":
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				result.Title = strings.TrimSpace(n.FirstChild.Data)
			}

		case "a":
			if params.IncludeLinks {
				e.currentLink = &WebLink{}
				e.extractLinkAttributes(n, result, params)
			}

		case "img":
			if params.IncludeImages {
				e.extractImageAttributes(n, result, params)
			}

		case "table":
			if params.IncludeTables {
				e.inTable = true
				e.currentTable = &WebTable{}
				e.extractTableAttributes(n, result, params)
			}

		case "thead":
			if e.inTable {
				e.inTableHead = true
			}

		case "tr":
			if e.inTable {
				e.inTableRow = true
				if !e.inTableHead {
					e.currentTable.Rows = append(e.currentTable.Rows, []string{})
				}
			}

		case "th", "td":
			if e.inTable {
				e.inTableCell = true
			}

		case "meta":
			e.extractMetaAttributes(n, result, params)
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			e.extractFromNode(c, result, params)
		}

		// Reset state after processing children
		switch n.Data {
		case "a":
			if params.IncludeLinks && e.currentLink != nil {
				if e.currentLink.URL != "" {
					result.Links = append(result.Links, *e.currentLink)
				}
				e.currentLink = nil
			}

		case "table":
			if params.IncludeTables && e.currentTable != nil {
				result.Tables = append(result.Tables, *e.currentTable)
				e.currentTable = nil
			}
			e.inTable = false

		case "thead":
			e.inTableHead = false

		case "tr":
			e.inTableRow = false

		case "th", "td":
			e.inTableCell = false
		}

	case html.TextNode:
		if !e.inTable || !params.IncludeTables {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				result.Content += " " + text
			}
		} else if e.inTable && e.inTableCell {
			text := strings.TrimSpace(n.Data)
			if e.inTableHead {
				e.currentTable.Headers = append(e.currentTable.Headers, text)
			} else if len(e.currentTable.Rows) > 0 {
				lastRow := &e.currentTable.Rows[len(e.currentTable.Rows)-1]
				*lastRow = append(*lastRow, text)
			}
		}
	}
}

// extractLinkAttributes extracts attributes from <a> tags
func (e *htmlExtractor) extractLinkAttributes(n *html.Node, result *WebExtractResult, params *WebExtractParams) {
	for _, attr := range n.Attr {
		switch attr.Key {
		case "href":
			if strings.TrimSpace(attr.Val) != "" {
				e.currentLink.URL = attr.Val
				// Check if internal or external link
				if strings.HasPrefix(attr.Val, "/") {
					e.currentLink.Type = "internal"
				} else {
					e.currentLink.Type = "external"
				}
			}
		}
	}

	// Extract link text
	if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
		e.currentLink.Text = strings.TrimSpace(n.FirstChild.Data)
	}
}

// extractImageAttributes extracts attributes from <img> tags
func (e *htmlExtractor) extractImageAttributes(n *html.Node, result *WebExtractResult, params *WebExtractParams) {
	img := WebImage{}

	for _, attr := range n.Attr {
		switch attr.Key {
		case "src":
			img.Src = attr.Val
		case "alt":
			img.Alt = attr.Val
		case "title":
			img.Title = attr.Val
		case "width":
			fmt.Sscanf(attr.Val, "%d", &img.Width)
		case "height":
			fmt.Sscanf(attr.Val, "%d", &img.Height)
		}
	}

	if img.Src != "" {
		result.Images = append(result.Images, img)
	}
}

// extractTableAttributes extracts attributes from <table> tags
func (e *htmlExtractor) extractTableAttributes(n *html.Node, result *WebExtractResult, params *WebExtractParams) {
	for _, attr := range n.Attr {
		if attr.Key == "summary" || attr.Key == "title" {
			e.currentTable.Caption = attr.Val
		}
	}
}

// extractMetaAttributes extracts attributes from <meta> tags
func (e *htmlExtractor) extractMetaAttributes(n *html.Node, result *WebExtractResult, params *WebExtractParams) {
	var name, content string
	for _, attr := range n.Attr {
		switch attr.Key {
		case "name":
			name = attr.Val
		case "property":
			name = attr.Val
		case "content":
			content = attr.Val
		}
	}

	if name != "" && content != "" {
		result.Metadata[strings.ToLower(name)] = content
	}
}

// countWords counts words in text
func countWords(text string) int {
	// Remove extra whitespace and split
	words := regexp.MustCompile(`\s+`).Split(strings.TrimSpace(text), -1)
	
	// Count non-empty words
	count := 0
	for _, word := range words {
		if strings.TrimSpace(word) != "" {
			count++
		}
	}
	
	return count
}

// ValidateWebExtractParams validates web extraction parameters
func ValidateWebExtractParams(params map[string]interface{}) error {
	// Check required URL parameter
	if urlStr, ok := params["url"].(string); !ok || strings.TrimSpace(urlStr) == "" {
		return fmt.Errorf("url parameter is required and must be a non-empty string")
	}

	// Validate URL format
	if urlStr, ok := params["url"].(string); ok {
		if err := validateExtractURL(urlStr); err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}
	}

	// Validate max_length
	if maxLength, ok := params["max_length"].(float64); ok {
		if maxLength < 1 || maxLength > 10000000 {
			return fmt.Errorf("max_length must be between 1 and 10000000")
		}
	}

	return nil
}