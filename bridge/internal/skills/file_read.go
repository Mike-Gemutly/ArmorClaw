package skills

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileReadResult represents the result of reading a file
type FileReadResult struct {
	Path        string                 `json:"path"`
	Type        string                 `json:"type"`
	Size        int64                  `json:"size"`
	Content     interface{}            `json:"content"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Encoding    string                 `json:"encoding,omitempty"`
	Lines       int                    `json:"lines,omitempty"`
	Columns     []string               `json:"columns,omitempty"`
}

// FileReadParams represents parameters for file reading
type FileReadParams struct {
	Path     string `json:"path"`
	Type     string `json:"type,omitempty"`     // "auto", "text", "json", "csv", "pdf"
	Encoding string `json:"encoding,omitempty"` // "utf-8", "ascii", "latin1"
	MaxSize  int64  `json:"max_size,omitempty"`  // Maximum file size in bytes
	Limit    int    `json:"limit,omitempty"`     // Maximum lines/records to read
	Offset   int    `json:"offset,omitempty"`    // Starting line/record
}

// ExecuteFileRead reads and parses a file
func ExecuteFileRead(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse parameters
	fileParams, err := parseFileReadParams(params)
	if err != nil {
		return nil, fmt.Errorf("invalid file read parameters: %w", err)
	}

	// Security validation
	if err := ValidateFileReadParams(fileParams); err != nil {
		return nil, fmt.Errorf("security validation failed: %w", err)
	}

	// Read file based on type
	switch fileParams.Type {
	case "text":
		return readTextFile(ctx, fileParams)
	case "json":
		return readJSONFile(ctx, fileParams)
	case "csv":
		return readCSVFile(ctx, fileParams)
	case "pdf":
		return readPDFFile(ctx, fileParams)
	default:
		// Auto-detect type based on extension
		return autoDetectAndRead(ctx, fileParams)
	}
}

// parseFileReadParams parses file read parameters from input
func parseFileReadParams(params map[string]interface{}) (*FileReadParams, error) {
	fileParams := &FileReadParams{}

	// Extract required parameters
	if path, ok := params["path"].(string); ok {
		fileParams.Path = path
	} else {
		return nil, fmt.Errorf("path parameter is required and must be a string")
	}

	// Extract optional parameters
	if file_type, ok := params["type"].(string); ok {
		fileParams.Type = strings.ToLower(file_type)
	}

	if encoding, ok := params["encoding"].(string); ok {
		fileParams.Encoding = strings.ToLower(encoding)
	} else {
		fileParams.Encoding = "utf-8"
	}

	if maxSize, ok := params["max_size"].(float64); ok {
		fileParams.MaxSize = int64(maxSize)
	} else {
		fileParams.MaxSize = 10 * 1024 * 1024 // 10MB default
	}

	if limit, ok := params["limit"].(float64); ok {
		fileParams.Limit = int(limit)
	}

	if offset, ok := params["offset"].(float64); ok {
		fileParams.Offset = int(offset)
	}

	return fileParams, nil
}

// readTextFile reads a text file
func readTextFile(ctx context.Context, params *FileReadParams) (*FileReadResult, error) {
	// Security check file path
	if err := validateFilePath(params.Path); err != nil {
		return nil, err
	}

	// Check file size
	fileInfo, err := os.Stat(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() > params.MaxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum size %d", fileInfo.Size(), params.MaxSize)
	}

	// Read file content
	data, err := os.ReadFile(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	content := string(data)
	lines := strings.Split(content, "\n")

	// Apply limit and offset
	if params.Offset > 0 {
		if params.Offset >= len(lines) {
			lines = []string{}
		} else {
			lines = lines[params.Offset:]
		}
	}

	if params.Limit > 0 && params.Limit < len(lines) {
		lines = lines[:params.Limit]
	}

	result := &FileReadResult{
		Path:     params.Path,
		Type:     "text",
		Size:     fileInfo.Size(),
		Content:  strings.Join(lines, "\n"),
		Encoding: params.Encoding,
		Lines:    len(lines),
		Metadata: map[string]interface{}{
			"word_count":   len(strings.Fields(content)),
			"char_count":   len(content),
			"line_count":   len(strings.Split(content, "\n")),
			"created_at":   fileInfo.ModTime(),
			"mode":         fileInfo.Mode(),
		},
	}

	return result, nil
}

// readJSONFile reads and parses a JSON file
func readJSONFile(ctx context.Context, params *FileReadParams) (*FileReadResult, error) {
	if err := validateFilePath(params.Path); err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() > params.MaxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum size %d", fileInfo.Size(), params.MaxSize)
	}

	data, err := os.ReadFile(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to read JSON file: %w", err)
	}

	var content interface{}
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	result := &FileReadResult{
		Path:    params.Path,
		Type:    "json",
		Size:    fileInfo.Size(),
		Content: content,
		Encoding: params.Encoding,
		Metadata: map[string]interface{}{
			"created_at": fileInfo.ModTime(),
			"mode":       fileInfo.Mode(),
			"valid_json": true,
		},
	}

	return result, nil
}

// readCSVFile reads and parses a CSV file
func readCSVFile(ctx context.Context, params *FileReadParams) (*FileReadResult, error) {
	if err := validateFilePath(params.Path); err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() > params.MaxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum size %d", fileInfo.Size(), params.MaxSize)
	}

	file, err := os.Open(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}

	// Apply limit and offset
	if params.Offset > 0 {
		if params.Offset >= len(records) {
			records = [][]string{}
		} else {
			records = records[params.Offset:]
		}
	}

	if params.Limit > 0 && params.Limit < len(records) {
		records = records[:params.Limit]
	}

	var columns []string
	if len(records) > 0 {
		columns = records[0]
	}

	result := &FileReadResult{
		Path:    params.Path,
		Type:    "csv",
		Size:    fileInfo.Size(),
		Content: records,
		Columns: columns,
		Encoding: params.Encoding,
		Lines:   len(records),
		Metadata: map[string]interface{}{
			"created_at": fileInfo.ModTime(),
			"mode":       fileInfo.Mode(),
			"row_count":  len(records),
			"col_count":  len(columns),
		},
	}

	return result, nil
}

// readPDFFile reads a PDF file (basic implementation)
func readPDFFile(ctx context.Context, params *FileReadParams) (*FileReadResult, error) {
	if err := validateFilePath(params.Path); err != nil {
		return nil, err
	}

	fileInfo, err := os.Stat(params.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if fileInfo.Size() > params.MaxSize {
		return nil, fmt.Errorf("file size %d exceeds maximum size %d", fileInfo.Size(), params.MaxSize)
	}

	// Note: This is a basic implementation for Phase 2
	// In production, you would use a PDF parsing library like github.com/ledongthuc/pdf
	// This is simplified for demonstration

	result := &FileReadResult{
		Path:    params.Path,
		Type:    "pdf",
		Size:    fileInfo.Size(),
		Content: "PDF parsing requires additional library integration in production",
		Encoding: params.Encoding,
		Metadata: map[string]interface{}{
			"created_at": fileInfo.ModTime(),
			"mode":       fileInfo.Mode(),
			"note":       "Basic PDF support - add library for full text extraction",
		},
	}

	return result, nil
}

// autoDetectAndRead detects file type and reads accordingly
func autoDetectAndRead(ctx context.Context, params *FileReadParams) (*FileReadResult, error) {
	ext := strings.ToLower(filepath.Ext(params.Path))
	
	switch ext {
	case ".json":
		params.Type = "json"
		return readJSONFile(ctx, params)
	case ".csv":
		params.Type = "csv"
		return readCSVFile(ctx, params)
	case ".txt", ".md", ".log", ".conf", ".yaml", ".yml":
		params.Type = "text"
		return readTextFile(ctx, params)
	case ".pdf":
		params.Type = "pdf"
		return readPDFFile(ctx, params)
	default:
		// Default to text file
		params.Type = "text"
		return readTextFile(ctx, params)
	}
}

// validateFilePath validates the file path for security
func validateFilePath(path string) error {
	// Check for empty path
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Check for directory traversal attempts
	if strings.Contains(path, "..") || strings.Contains(path, "~/") {
		return fmt.Errorf("directory traversal not allowed: %s", path)
	}

	// Check for absolute paths (only relative paths allowed for security)
	if filepath.IsAbs(path) {
		return fmt.Errorf("absolute paths not allowed: %s", path)
	}

	// Clean the path
	cleanPath := filepath.Clean(path)
	if cleanPath != path {
		return fmt.Errorf("path contains suspicious elements: %s", path)
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(path))
	allowedExtensions := []string{".txt", ".json", ".csv", ".pdf", ".md", ".log", ".conf", ".yaml", ".yml"}
	
	allowed := false
	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			allowed = true
			break
		}
	}
	
	if !allowed {
		return fmt.Errorf("file type not allowed: %s", ext)
	}

	return nil
}

// ValidateFileReadParams validates file read parameters
func ValidateFileReadParams(params *FileReadParams) error {
	// Validate file path
	if err := validateFilePath(params.Path); err != nil {
		return err
	}

	// Validate file type
	if params.Type != "" {
		allowedTypes := []string{"text", "json", "csv", "pdf"}
		allowed := false
		for _, allowedType := range allowedTypes {
			if params.Type == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("unsupported file type: %s (supported: text, json, csv, pdf)", params.Type)
		}
	}

	// Validate encoding
	if params.Encoding != "" {
		allowedEncodings := []string{"utf-8", "ascii", "latin1"}
		allowed := false
		for _, allowedEncoding := range allowedEncodings {
			if params.Encoding == allowedEncoding {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("unsupported encoding: %s (supported: utf-8, ascii, latin1)", params.Encoding)
		}
	}

	// Validate max size
	if params.MaxSize <= 0 {
		return fmt.Errorf("max_size must be positive")
	}
	if params.MaxSize > 100*1024*1024 { // 100MB absolute maximum
		return fmt.Errorf("max_size cannot exceed 100MB")
	}

	// Validate limit and offset
	if params.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}
	if params.Offset < 0 {
		return fmt.Errorf("offset cannot be negative")
	}

	return nil
}