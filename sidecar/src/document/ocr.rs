use crate::error::{Result, SidecarError};
use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// OCR result
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcrResult {
    pub text: String,
    pub confidence: f32,
    pub language: String,
    pub metadata: HashMap<String, String>,
}

/// OCR configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct OcrConfig {
    pub language: String,
    pub dpi: Option<u32>,
    pub psm: Option<u32>, // Page segmentation mode
}

impl Default for OcrConfig {
    fn default() -> Self {
        Self {
            language: "eng".to_string(),
            dpi: Some(300),
            psm: Some(3),
        }
    }
}

/// OCR extractor using Tesseract subprocess
pub struct OcrExtractor {
    config: OcrConfig,
}

impl OcrExtractor {
    pub fn new(config: OcrConfig) -> Self {
        Self { config }
    }

    /// Extract text from image using OCR
    pub fn extract(&self, image_data: &[u8]) -> Result<OcrResult> {
        // TODO: Implement Tesseract subprocess invocation
        Err(SidecarError::InvalidRequest(
            "OCR extraction not yet implemented. Task 33-34 pending.".to_string()
        ))
    }
}

impl Default for OcrExtractor {
    fn default() -> Self {
        Self::new(OcrConfig::default())
    }
}

/// Extract text with OCR
pub fn extract_text_with_ocr(image_data: &[u8], config: Option<OcrConfig>) -> Result<OcrResult> {
    let extractor = OcrExtractor::new(config.unwrap_or_default());
    extractor.extract(image_data)
}

/// Validate language code
pub fn validate_language(lang: &str) -> Result<()> {
    let supported = get_supported_languages();
    if supported.contains(&lang.to_string()) {
        Ok(())
    } else {
        Err(SidecarError::InvalidRequest(format!(
            "Unsupported language: {}. Supported: {:?}",
            lang, supported
        )))
    }
}

/// Get supported languages
pub fn get_supported_languages() -> Vec<String> {
    vec![
        "eng".to_string(),
        "spa".to_string(),
        "fra".to_string(),
        "deu".to_string(),
        "ita".to_string(),
        "por".to_string(),
        "rus".to_string(),
        "chi_sim".to_string(),
        "chi_tra".to_string(),
        "jpn".to_string(),
        "kor".to_string(),
        "ara".to_string(),
        "hin".to_string(),
        "nld".to_string(),
        "pol".to_string(),
        "tur".to_string(),
    ]
}

/// Detect language from text (simple heuristic)
pub fn detect_language_from_text(text: &str) -> String {
    // TODO: Implement proper language detection
    // For now, default to English
    if text.is_empty() {
        "eng".to_string()
    } else {
        "eng".to_string()
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_ocr_extractor_creation() {
        let extractor = OcrExtractor::default();
        assert!(extractor.extract(&[]).is_err()); // Stub implementation
    }

    #[test]
    fn test_validate_language() {
        assert!(validate_language("eng").is_ok());
        assert!(validate_language("invalid").is_err());
    }

    #[test]
    fn test_supported_languages() {
        let langs = get_supported_languages();
        assert!(langs.contains(&"eng".to_string()));
        assert!(langs.contains(&"spa".to_string()));
        assert!(langs.len() >= 10);
    }
}
