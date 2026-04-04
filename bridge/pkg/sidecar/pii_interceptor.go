// Package sidecar provides PII interception for sidecar client calls
package sidecar

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/armorclaw/bridge/pkg/pii"
)

// Action defines the action to take when PII is detected
type Action string

const (
	// ActionRedact redacts detected PII and continues with the request
	ActionRedact Action = "redact"
	// ActionReject rejects the request when PII is detected
	ActionReject Action = "reject"
)

// PIIInterceptorConfig holds configuration for PII interception
type PIIInterceptorConfig struct {
	Enabled    bool          // Enable PII interception
	Action     Action        // Action to take when PII is detected
	Scrubber   *pii.Scrubber // PII scrubber instance
	Logger     *slog.Logger  // Logger for PII detection events
	LogOnly    bool          // If true, log but don't modify requests
	StrictMode bool          // If true, fail on any detection error
}

// DefaultPIIInterceptorConfig returns a configuration with sensible defaults
func DefaultPIIInterceptorConfig() *PIIInterceptorConfig {
	return &PIIInterceptorConfig{
		Enabled:    true,
		Action:     ActionRedact,
		Scrubber:   pii.New(),
		Logger:     slog.Default(),
		LogOnly:    false,
		StrictMode: false,
	}
}

// PIIInterceptor intercepts sidecar client requests to detect and handle PII
type PIIInterceptor struct {
	config *PIIInterceptorConfig
	mu     sync.RWMutex
}

// NewPIIInterceptor creates a new PII interceptor
func NewPIIInterceptor(config *PIIInterceptorConfig) *PIIInterceptor {
	if config == nil {
		config = DefaultPIIInterceptorConfig()
	}

	return &PIIInterceptor{
		config: config,
	}
}

// isEnabled returns whether PII interception is enabled
func (i *PIIInterceptor) isEnabled() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.config.Enabled
}

// InterceptRequest intercepts a request before sending it to sidecar
// Returns potentially modified request and an error if PII is detected and action is reject
func (i *PIIInterceptor) InterceptRequest(ctx context.Context, methodName string, request interface{}) (interface{}, error) {
	if !i.isEnabled() {
		return request, nil
	}

	i.mu.RLock()
	config := i.config
	i.mu.RUnlock()

	// Scan request for PII using direct field inspection
	detections, hasPII := i.scanRequestForPIIDirect(ctx, methodName, request)

	if !hasPII {
		// No PII detected, return request as-is
		return request, nil
	}

	// Log PII detection without exposing actual PII
	i.logPIIDetection(ctx, methodName, detections, config)

	// Handle based on action
	if config.LogOnly {
		// Log only, return request unchanged
		return request, nil
	}

	if config.Action == ActionReject {
		// Reject request
		return nil, fmt.Errorf("PII detected in %s request, rejecting: %d PII instances found", methodName, len(detections))
	}

	// Redact PII from request by creating a copy
	redactedRequest := i.redactRequestDirect(request)

	config.Logger.InfoContext(ctx, "PII intercepted and redacted",
		"method", methodName,
		"detection_count", len(detections),
		"action", config.Action,
	)

	return redactedRequest, nil
}

// scanRequestForPIIDirect scans a request directly for PII
func (i *PIIInterceptor) scanRequestForPIIDirect(ctx context.Context, methodName string, request interface{}) ([]pii.Redaction, bool) {
	var allDetections []pii.Redaction

	i.mu.RLock()
	scrubber := i.config.Scrubber
	i.mu.RUnlock()

	switch req := request.(type) {
	case *UploadBlobRequest:
		// Scan content
		if req.Content != nil && i.isLikelyText(string(req.Content)) {
			detections := scrubber.Detect(string(req.Content))
			for _, detection := range detections {
				detection.Description = "UploadBlobRequest.Content: " + detection.Description
				allDetections = append(allDetections, detection)
			}
		}
		// Scan metadata fields
		allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.Provider", req.Provider, scrubber)...)
		allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.DestinationUri", req.DestinationUri, scrubber)...)
		allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.ContentType", req.ContentType, scrubber)...)
		allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.LocalFilePath", req.LocalFilePath, scrubber)...)
		// Scan provider config
		for key, value := range req.ProviderConfig {
			allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.ProviderConfig."+key, value, scrubber)...)
		}
		// Scan metadata
		if req.Metadata != nil {
			allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.Metadata.RequestId", req.Metadata.RequestId, scrubber)...)
			allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.Metadata.EphemeralToken", req.Metadata.EphemeralToken, scrubber)...)
			allDetections = append(allDetections, i.scanStringField("UploadBlobRequest.Metadata.OperationSignature", req.Metadata.OperationSignature, scrubber)...)
		}

	case *ExtractTextRequest:
		// Scan document content
		if req.DocumentContent != nil && i.isLikelyText(string(req.DocumentContent)) {
			detections := scrubber.Detect(string(req.DocumentContent))
			for _, detection := range detections {
				detection.Description = "ExtractTextRequest.DocumentContent: " + detection.Description
				allDetections = append(allDetections, detection)
			}
		}
		// Scan other fields
		allDetections = append(allDetections, i.scanStringField("ExtractTextRequest.DocumentFormat", req.DocumentFormat, scrubber)...)
		allDetections = append(allDetections, i.scanStringField("ExtractTextRequest.DocumentUri", req.DocumentUri, scrubber)...)
		// Scan options
		for key, value := range req.Options {
			allDetections = append(allDetections, i.scanStringField("ExtractTextRequest.Options."+key, value, scrubber)...)
		}
		// Scan metadata
		if req.Metadata != nil {
			allDetections = append(allDetections, i.scanStringField("ExtractTextRequest.Metadata.RequestId", req.Metadata.RequestId, scrubber)...)
			allDetections = append(allDetections, i.scanStringField("ExtractTextRequest.Metadata.EphemeralToken", req.Metadata.EphemeralToken, scrubber)...)
			allDetections = append(allDetections, i.scanStringField("ExtractTextRequest.Metadata.OperationSignature", req.Metadata.OperationSignature, scrubber)...)
		}

	case *ProcessDocumentRequest:
		// Scan input content
		if req.InputContent != nil && i.isLikelyText(string(req.InputContent)) {
			detections := scrubber.Detect(string(req.InputContent))
			for _, detection := range detections {
				detection.Description = "ProcessDocumentRequest.InputContent: " + detection.Description
				allDetections = append(allDetections, detection)
			}
		}
		// Scan other fields
		allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.InputFormat", req.InputFormat, scrubber)...)
		allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.OutputFormat", req.OutputFormat, scrubber)...)
		allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.InputUri", req.InputUri, scrubber)...)
		allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.Operation", req.Operation, scrubber)...)
		// Scan operation params
		for key, value := range req.OperationParams {
			allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.OperationParams."+key, value, scrubber)...)
		}
		// Scan metadata
		if req.Metadata != nil {
			allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.Metadata.RequestId", req.Metadata.RequestId, scrubber)...)
			allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.Metadata.EphemeralToken", req.Metadata.EphemeralToken, scrubber)...)
			allDetections = append(allDetections, i.scanStringField("ProcessDocumentRequest.Metadata.OperationSignature", req.Metadata.OperationSignature, scrubber)...)
		}
	}

	return allDetections, len(allDetections) > 0
}

// scanStringField scans a single string field for PII
func (i *PIIInterceptor) scanStringField(fieldName, value string, scrubber *pii.Scrubber) []pii.Redaction {
	if value == "" {
		return nil
	}

	detections := scrubber.Detect(value)
	for _, detection := range detections {
		detection.Description = fieldName + ": " + detection.Description
	}
	return detections
}

// redactRequestDirect creates a redacted copy of the request
func (i *PIIInterceptor) redactRequestDirect(request interface{}) interface{} {
	i.mu.RLock()
	scrubber := i.config.Scrubber
	i.mu.RUnlock()

	switch req := request.(type) {
	case *UploadBlobRequest:
		// Create a copy
		redacted := &UploadBlobRequest{
			Provider:       i.scrubString(req.Provider, scrubber),
			DestinationUri: i.scrubString(req.DestinationUri, scrubber),
			ContentType:    i.scrubString(req.ContentType, scrubber),
			LocalFilePath:  i.scrubString(req.LocalFilePath, scrubber),
		}
		// Scrub content if it's text
		if req.Content != nil && i.isLikelyText(string(req.Content)) {
			scrubbed, _ := scrubber.Scrub(string(req.Content))
			redacted.Content = []byte(scrubbed)
		} else {
			redacted.Content = req.Content
		}
		// Scrub provider config
		redacted.ProviderConfig = make(map[string]string)
		for key, value := range req.ProviderConfig {
			redacted.ProviderConfig[key] = i.scrubString(value, scrubber)
		}
		// Copy metadata without scrubbing (it contains IDs/tokens)
		if req.Metadata != nil {
			redacted.Metadata = &RequestMetadata{
				RequestId:          req.Metadata.RequestId,
				EphemeralToken:     req.Metadata.EphemeralToken,
				TimestampUnix:      req.Metadata.TimestampUnix,
				OperationSignature: req.Metadata.OperationSignature,
			}
		}
		return redacted

	case *ExtractTextRequest:
		redacted := &ExtractTextRequest{
			DocumentFormat: i.scrubString(req.DocumentFormat, scrubber),
			DocumentUri:    i.scrubString(req.DocumentUri, scrubber),
		}
		// Scrub content if it's text
		if req.DocumentContent != nil && i.isLikelyText(string(req.DocumentContent)) {
			scrubbed, _ := scrubber.Scrub(string(req.DocumentContent))
			redacted.DocumentContent = []byte(scrubbed)
		} else {
			redacted.DocumentContent = req.DocumentContent
		}
		// Scrub options
		redacted.Options = make(map[string]string)
		for key, value := range req.Options {
			redacted.Options[key] = i.scrubString(value, scrubber)
		}
		// Copy metadata
		if req.Metadata != nil {
			redacted.Metadata = &RequestMetadata{
				RequestId:          req.Metadata.RequestId,
				EphemeralToken:     req.Metadata.EphemeralToken,
				TimestampUnix:      req.Metadata.TimestampUnix,
				OperationSignature: req.Metadata.OperationSignature,
			}
		}
		return redacted

	case *ProcessDocumentRequest:
		redacted := &ProcessDocumentRequest{
			InputFormat:  i.scrubString(req.InputFormat, scrubber),
			OutputFormat: i.scrubString(req.OutputFormat, scrubber),
			InputUri:     i.scrubString(req.InputUri, scrubber),
			Operation:    i.scrubString(req.Operation, scrubber),
		}
		// Scrub input content if it's text
		if req.InputContent != nil && i.isLikelyText(string(req.InputContent)) {
			scrubbed, _ := scrubber.Scrub(string(req.InputContent))
			redacted.InputContent = []byte(scrubbed)
		} else {
			redacted.InputContent = req.InputContent
		}
		// Scrub operation params
		redacted.OperationParams = make(map[string]string)
		for key, value := range req.OperationParams {
			redacted.OperationParams[key] = i.scrubString(value, scrubber)
		}
		// Copy metadata
		if req.Metadata != nil {
			redacted.Metadata = &RequestMetadata{
				RequestId:          req.Metadata.RequestId,
				EphemeralToken:     req.Metadata.EphemeralToken,
				TimestampUnix:      req.Metadata.TimestampUnix,
				OperationSignature: req.Metadata.OperationSignature,
			}
		}
		return redacted
	}

	return request
}

// scrubString scrubs a single string
func (i *PIIInterceptor) scrubString(s string, scrubber *pii.Scrubber) string {
	if s == "" {
		return s
	}
	scrubbed, _ := scrubber.Scrub(s)
	return scrubbed
}

// logPIIDetection logs PII detection without exposing actual PII
func (i *PIIInterceptor) logPIIDetection(ctx context.Context, methodName string, detections []pii.Redaction, config *PIIInterceptorConfig) {
	// Create summary of PII types detected
	summary := config.Scrubber.CreateSummary(detections)

	// Log without exposing actual PII values
	config.Logger.WarnContext(ctx, "PII detected in sidecar request",
		"method", methodName,
		"detection_count", len(detections),
		"pii_types", summary,
		"action", config.Action,
	)
}

// isLikelyText checks if a string is likely text content (not binary)
func (i *PIIInterceptor) isLikelyText(s string) bool {
	// Check if string contains mostly printable ASCII
	if len(s) == 0 {
		return false
	}

	textChars := 0
	for _, r := range s {
		if r >= 32 && r <= 126 || r == '\n' || r == '\r' || r == '\t' {
			textChars++
		}
	}

	// If more than 90% are printable ASCII, consider it text
	ratio := float64(textChars) / float64(len(s))
	return ratio > 0.9
}

// UpdateConfig updates interceptor configuration
func (i *PIIInterceptor) UpdateConfig(config *PIIInterceptorConfig) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.config = config
}

// GetConfig returns current interceptor configuration
func (i *PIIInterceptor) GetConfig() *PIIInterceptorConfig {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.config
}

// SetEnabled enables or disables PII interception
func (i *PIIInterceptor) SetEnabled(enabled bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.config.Enabled = enabled
}

// SetAction sets action to take when PII is detected
func (i *PIIInterceptor) SetAction(action Action) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.config.Action = action
}

// SetLogOnly sets whether to log only or also modify requests
func (i *PIIInterceptor) SetLogOnly(logOnly bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.config.LogOnly = logOnly
}

// GetStatistics returns statistics about PII detection
func (i *PIIInterceptor) GetStatistics() map[string]interface{} {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return map[string]interface{}{
		"enabled":       i.config.Enabled,
		"action":        i.config.Action,
		"log_only":      i.config.LogOnly,
		"strict_mode":   i.config.StrictMode,
		"pattern_count": i.config.Scrubber.GetPatternCount(),
	}
}

// ScrubRequestText scrubs PII from a text string
func (i *PIIInterceptor) ScrubRequestText(text string) (string, []pii.Redaction) {
	if !i.isEnabled() {
		return text, nil
	}

	i.mu.RLock()
	scrubber := i.config.Scrubber
	i.mu.RUnlock()

	return scrubber.Scrub(text)
}

// ContainsPII checks if text contains PII
func (i *PIIInterceptor) ContainsPII(text string) bool {
	if !i.isEnabled() {
		return false
	}

	i.mu.RLock()
	scrubber := i.config.Scrubber
	i.mu.RUnlock()

	return scrubber.ContainsPII(text)
}
