// Package pii provides PHI detection and scrubbing for HIPAA compliance.
//
// Resolves: Gap - PHI in Media Attachments
//
// Extends text-based PHI detection to include OCR extraction from
// images and PDFs, ensuring PHI embedded in media is detected and quarantined.
package pii

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// MediaType represents the type of media being processed
type MediaType string

const (
	MediaTypeImage MediaType = "image"
	MediaTypePDF   MediaType = "pdf"
	MediaTypeAudio MediaType = "audio"  // Future: Voice-to-text
	MediaTypeVideo MediaType = "video"  // Future: Frame extraction
)

// MediaAttachment represents a media file to be scanned for PHI
type MediaAttachment struct {
	ID          string
	Filename    string
	MimeType    string
	MediaType   MediaType
	Data        []byte
	Size        int64
	Caption     string // User-provided caption (also scanned)
	Source      string // Platform source (slack, discord, etc.)
}

// ScanResult represents the result of a media PHI scan
type ScanResult struct {
	AttachmentID    string
	MediaType       MediaType
	PHIDetected     bool
	PHITypes        []string   // Types of PHI found (SSN, MRN, etc.)
	Confidence      float64    // OCR confidence (0-1)
	ExtractedText   string     // Text extracted via OCR
	RedactedText    string     // Text with PHI redacted
	Quarantined     bool       // Whether media was quarantined
	QuarantinePath  string     // Path to quarantined file (if quarantined)
	ReplacementURL  string     // URL to placeholder image
	ScanDuration    time.Duration
	Errors          []string
}

// MediaScanner interface for scanning media files
type MediaScanner interface {
	// Scan analyzes a media file for PHI
	Scan(ctx context.Context, attachment *MediaAttachment) (*ScanResult, error)

	// Supports returns true if the scanner supports this media type
	Supports(mediaType MediaType) bool
}

// OCRProvider interface for OCR extraction
type OCRProvider interface {
	// ExtractText extracts text from image data
	ExtractText(ctx context.Context, imageData []byte) (string, float64, error)

	// ExtractTextFromPDF extracts text from PDF data
	ExtractTextFromPDF(ctx context.Context, pdfData []byte) (string, error)
}

// QuarantineStore interface for quarantining files
type QuarantineStore interface {
	// Store saves a quarantined file
	Store(ctx context.Context, attachment *MediaAttachment, reason string) (string, error)

	// GetPlaceholderURL returns URL to a placeholder image
	GetPlaceholderURL(reason string) string
}

// MediaPHIScanner coordinates PHI scanning for media attachments
type MediaPHIScanner struct {
	logger          *slog.Logger
	ocrProvider     OCRProvider
	quarantineStore QuarantineStore
	textScanner     *HIPAAScrubber // HIPAA-compliant text scanner
	config          MediaScanConfig
}

// MediaScanConfig configures the media scanner
type MediaScanConfig struct {
	// EnableOCR enables OCR processing
	EnableOCR bool
	// EnablePDF enables PDF text extraction
	EnablePDF bool
	// QuarantineOnPHI quarantines files when PHI detected
	QuarantineOnPHI bool
	// MaxFileSize limits file size for OCR (bytes)
	MaxFileSize int64
	// OCRTimeout timeout for OCR operations
	OCRTimeout time.Duration
	// MinConfidence minimum OCR confidence to trust results
	MinConfidence float64
}

// DefaultMediaScanConfig returns default configuration
func DefaultMediaScanConfig() MediaScanConfig {
	return MediaScanConfig{
		EnableOCR:       true,
		EnablePDF:       true,
		QuarantineOnPHI: true,
		MaxFileSize:     10 * 1024 * 1024, // 10MB
		OCRTimeout:      30 * time.Second,
		MinConfidence:   0.6,
	}
}

// NewMediaPHIScanner creates a new media PHI scanner
func NewMediaPHIScanner(
	logger *slog.Logger,
	ocrProvider OCRProvider,
	quarantineStore QuarantineStore,
	textScanner *HIPAAScrubber,
	config MediaScanConfig,
) *MediaPHIScanner {
	if logger == nil {
		logger = slog.Default().With("component", "media_phi_scanner")
	}

	return &MediaPHIScanner{
		logger:          logger,
		ocrProvider:     ocrProvider,
		quarantineStore: quarantineStore,
		textScanner:     textScanner,
		config:          config,
	}
}

// Scan performs PHI detection on a media attachment
func (s *MediaPHIScanner) Scan(ctx context.Context, attachment *MediaAttachment) (*ScanResult, error) {
	start := time.Now()
	result := &ScanResult{
		AttachmentID: attachment.ID,
		MediaType:    attachment.MediaType,
		PHIDetected:  false,
	}

	defer func() {
		result.ScanDuration = time.Since(start)
	}()

	// Check file size limit
	if attachment.Size > s.config.MaxFileSize {
		s.logger.Warn("file_too_large_for_ocr",
			"attachment_id", attachment.ID,
			"size", attachment.Size,
			"limit", s.config.MaxFileSize,
		)
		result.Errors = append(result.Errors, "file exceeds size limit for OCR")
		return result, nil
	}

	// Extract text based on media type
	var extractedText string
	var confidence float64 = 1.0
	var err error

	switch attachment.MediaType {
	case MediaTypeImage:
		if !s.config.EnableOCR || s.ocrProvider == nil {
			s.logger.Debug("ocr_disabled", "attachment_id", attachment.ID)
			return result, nil
		}

		extractedText, confidence, err = s.ocrProvider.ExtractText(ctx, attachment.Data)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("OCR failed: %v", err))
			s.logger.Warn("ocr_failed", "attachment_id", attachment.ID, "error", err)
			return result, nil
		}
		result.Confidence = confidence
		result.ExtractedText = extractedText

	case MediaTypePDF:
		if !s.config.EnablePDF || s.ocrProvider == nil {
			s.logger.Debug("pdf_extraction_disabled", "attachment_id", attachment.ID)
			return result, nil
		}

		extractedText, err = s.ocrProvider.ExtractTextFromPDF(ctx, attachment.Data)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("PDF extraction failed: %v", err))
			s.logger.Warn("pdf_extraction_failed", "attachment_id", attachment.ID, "error", err)
			return result, nil
		}
		result.ExtractedText = extractedText

	default:
		// Unsupported media type
		return result, nil
	}

	// Check OCR confidence threshold
	if confidence < s.config.MinConfidence {
		s.logger.Warn("ocr_confidence_low",
			"attachment_id", attachment.ID,
			"confidence", confidence,
			"min", s.config.MinConfidence,
		)
		// Still scan what we have, but note the low confidence
	}

	// Combine extracted text with caption for comprehensive scan
	fullText := strings.Join([]string{extractedText, attachment.Caption}, "\n")

	// Scan for PHI using HIPAA-compliant text scanner
	scrubbedText, detections, err := s.textScanner.ScrubPHI(ctx, fullText, "media_scan")
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("PHI scan failed: %v", err))
		s.logger.Warn("phi_scan_failed", "attachment_id", attachment.ID, "error", err)
		return result, nil
	}

	if len(detections) > 0 {
		result.PHIDetected = true
		result.PHITypes = extractPHITypes(detections)
		result.RedactedText = scrubbedText

		s.logger.Warn("phi_detected_in_media",
			"attachment_id", attachment.ID,
			"media_type", attachment.MediaType,
			"phi_types", result.PHITypes,
			"ocr_confidence", confidence,
		)

		// Quarantine the file
		if s.config.QuarantineOnPHI && s.quarantineStore != nil {
			reason := fmt.Sprintf("PHI detected: %s", strings.Join(result.PHITypes, ", "))
			quarantinePath, err := s.quarantineStore.Store(ctx, attachment, reason)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Quarantine failed: %v", err))
				s.logger.Error("quarantine_failed", "attachment_id", attachment.ID, "error", err)
			} else {
				result.Quarantined = true
				result.QuarantinePath = quarantinePath
				result.ReplacementURL = s.quarantineStore.GetPlaceholderURL(reason)
			}
		}
	}

	return result, nil
}

// ScanCaption scans just the caption text for PHI (fast path)
func (s *MediaPHIScanner) ScanCaption(ctx context.Context, caption string) (string, []PHIDetection, error) {
	return s.textScanner.ScrubPHI(ctx, caption, "caption")
}

// ShouldQuarantine determines if an attachment should be quarantined
func (s *MediaPHIScanner) ShouldQuarantine(result *ScanResult) bool {
	return result.PHIDetected && s.config.QuarantineOnPHI
}

// GetReplacementMessage returns a user-facing message for quarantined media
func GetReplacementMessage(result *ScanResult) string {
	if !result.PHIDetected {
		return ""
	}

	phiTypes := strings.Join(result.PHITypes, ", ")
	return fmt.Sprintf("[REDACTED: %s detected in attachment]", phiTypes)
}

// extractPHITypes extracts unique PHI type strings from detections
func extractPHITypes(detections []PHIDetection) []string {
	seen := make(map[PHIType]bool)
	var types []string
	for _, d := range detections {
		if !seen[d.Type] {
			seen[d.Type] = true
			types = append(types, string(d.Type))
		}
	}
	return types
}
