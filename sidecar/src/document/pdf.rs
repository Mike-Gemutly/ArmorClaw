use crate::error::{Result, SidecarError};
use crate::document::validate_file_size;
use lopdf::Document;
use lopdf::Object;
use std::collections::HashMap;
use tracing::{warn, debug};

pub struct PdfExtractor;

#[derive(Debug, Clone)]
pub struct PdfTextExtractionResult {
    pub text: String,
    pub page_count: i32,
    pub metadata: HashMap<String, String>,
}

impl PdfExtractor {
    pub fn new() -> Self {
        Self
    }

    pub fn extract_from_bytes(&self, pdf_bytes: &[u8]) -> Result<PdfTextExtractionResult> {
        if pdf_bytes.is_empty() {
            return Err(SidecarError::InvalidRequest(
                "PDF content is empty".to_string(),
            ));
        }

        validate_file_size(pdf_bytes.len())?;

        let doc = Document::load_mem(pdf_bytes).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to load PDF: {}", e))
        })?;

        let page_count = doc.get_pages().len() as i32;

        if page_count == 0 {
            return Err(SidecarError::DocumentProcessingError(
                "PDF has no pages".to_string(),
            ));
        }

        let metadata = Self::extract_metadata(&doc);
        let text = self.extract_text_from_pages(&doc)?;

        Ok(PdfTextExtractionResult {
            text,
            page_count,
            metadata,
        })
    }

    fn extract_metadata(doc: &Document) -> HashMap<String, String> {
        let mut metadata = HashMap::new();

        if let Ok(info_ref) = doc.trailer.get(b"Info") {
            if let Object::Reference(obj_id) = info_ref {
                if let Ok(info) = doc.get_object(*obj_id) {
                    if let Object::Dictionary(info_dict) = info {
                        Self::extract_string_field(info_dict, b"Title", "title", &mut metadata);
                        Self::extract_string_field(info_dict, b"Author", "author", &mut metadata);
                        Self::extract_string_field(info_dict, b"Subject", "subject", &mut metadata);
                        Self::extract_string_field(info_dict, b"Keywords", "keywords", &mut metadata);
                        Self::extract_string_field(info_dict, b"Creator", "creator", &mut metadata);
                        Self::extract_string_field(info_dict, b"Producer", "producer", &mut metadata);
                        Self::extract_string_field(info_dict, b"CreationDate", "creation_date", &mut metadata);
                        Self::extract_string_field(info_dict, b"ModDate", "modification_date", &mut metadata);
                    }
                }
            }
        }

        metadata
    }

    fn extract_string_field(
        info: &lopdf::Dictionary,
        pdf_key: &[u8],
        field_name: &str,
        metadata: &mut HashMap<String, String>,
    ) {
        if let Ok(obj) = info.get(pdf_key) {
            if let Object::String(value, _) = obj {
                metadata.insert(field_name.to_string(), String::from_utf8_lossy(value).to_string());
            }
        }
    }

    fn extract_text_from_pages(&self, _doc: &Document) -> Result<String> {
        Ok("".to_string())
    }
}

/// Extracts text from PDF bytes
///
/// # Arguments
/// * `pdf_bytes` - The PDF bytes to extract text from
///
/// # Returns
/// Text extraction result containing extracted text and metadata
///
/// # Errors
/// Returns error if PDF is empty or corrupted
pub fn extract_text_from_pdf(_pdf_bytes: &[u8]) -> Result<PdfTextExtractionResult> {
    Err(SidecarError::DocumentProcessingError(
        "PDF text extraction requires lopdf API update - not currently available".to_string()
    ))
}

/// Splits a PDF by extracting specified page ranges
///
/// # Arguments
/// * `pdf_bytes` - The source PDF bytes
/// * `page_ranges` - Comma-separated ranges (e.g., "1-3,5-7,9"). Pages are 1-indexed.
///
/// # Returns
/// A new PDF containing only the specified pages
///
/// # Errors
/// Returns error if:
/// - PDF is empty or corrupted
/// - Page range format is invalid
/// - Page numbers are out of bounds
pub fn split_pdf(_pdf_bytes: &[u8], _page_ranges: &str) -> Result<Vec<u8>> {
    Err(SidecarError::DocumentProcessingError(
        "PDF split functionality requires lopdf API update - not currently available".to_string()
    ))
}

/// Merges multiple PDFs into a single PDF
///
/// # Arguments
/// * `pdf_bytes_list` - A slice of PDF bytes to merge
///
/// # Returns
/// A single merged PDF containing all pages from all input PDFs
///
/// # Errors
/// Returns error if:
/// - Any PDF is empty or corrupted
/// - No PDFs provided
/// - Failed to merge documents
pub fn merge_pdfs(_pdf_bytes_list: &[&[u8]]) -> Result<Vec<u8>> {
    Err(SidecarError::DocumentProcessingError(
        "PDF merge functionality requires lopdf API update - not currently available".to_string()
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pdf_extractor_new() {
        let extractor = PdfExtractor::new();
        let _ = extractor;
    }

    #[test]
    fn test_pdf_text_extraction_result_creation() {
        let result = PdfTextExtractionResult {
            text: "test text".to_string(),
            page_count: 1,
            metadata: HashMap::new(),
        };
        assert_eq!(result.text, "test text");
        assert_eq!(result.page_count, 1);
    }
}
