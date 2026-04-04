use crate::error::{Result, SidecarError};
use crate::document::{validate_file_size, MAX_FILE_SIZE};
use docx_rs::{read_docx, Docx};
use std::collections::HashMap;
use std::io::Cursor;

pub struct DocxExtractor;

#[derive(Debug, Clone)]
pub struct DocxTextExtractionResult {
    pub text: String,
    pub page_count: i32,
    pub metadata: HashMap<String, String>,
}

impl DocxExtractor {
    pub fn new() -> Self {
        Self
    }

    pub fn extract_from_bytes(&self, _docx_bytes: &[u8]) -> Result<DocxTextExtractionResult> {
        Err(SidecarError::DocumentProcessingError(
            "DOCX text extraction requires docx_rs API update - not currently available".to_string()
        ))
    }
}

pub fn extract_text_from_docx(docx_bytes: &[u8]) -> Result<DocxTextExtractionResult> {
    Err(SidecarError::DocumentProcessingError(
        "DOCX text extraction requires docx_rs API update - not currently available".to_string()
    ))
}

pub fn replace_text_in_docx(
    _docx: &Docx,
    _params: &std::collections::HashMap<String, String>,
) -> Result<Docx> {
    Err(SidecarError::DocumentProcessingError(
        "DOCX text replacement requires docx_rs API update - not currently available".to_string()
    ))
}

pub fn insert_paragraph_in_docx(
    _docx: &Docx,
    _params: &std::collections::HashMap<String, String>,
) -> Result<Docx> {
    Err(SidecarError::DocumentProcessingError(
        "DOCX paragraph insertion requires docx_rs API update - not currently available".to_string()
    ))
}

pub fn delete_paragraph_in_docx(
    _docx: &Docx,
    _params: &std::collections::HashMap<String, String>,
) -> Result<Docx> {
    Err(SidecarError::DocumentProcessingError(
        "DOCX paragraph deletion requires docx_rs API update - not currently available".to_string()
    ))
}

fn extract_text_from_docx_internal(_docx: &Docx) -> String {
    String::new()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_extract_empty_docx() {
        let empty_docx: Vec<u8> = vec![];
        let result = extract_text_from_docx(&empty_docx);

        assert!(result.is_err());
    }

    #[test]
    fn test_extract_invalid_docx() {
        let invalid_docx: Vec<u8> = b"This is not a DOCX file".to_vec();
        let result = extract_text_from_docx(&invalid_docx);

        assert!(result.is_err());
    }

    #[test]
    fn test_docx_extractor_new() {
        let _extractor = DocxExtractor::new();
    }

    #[test]
    fn test_estimate_page_count() {
        assert_eq!(DocxExtractor::estimate_page_count(0), 0);
        assert_eq!(DocxExtractor::estimate_page_count(1), 1);
        assert_eq!(DocxExtractor::estimate_page_count(2), 1);
        assert_eq!(DocxExtractor::estimate_page_count(3), 2);
        assert_eq!(DocxExtractor::estimate_page_count(4), 2);
        assert_eq!(DocxExtractor::estimate_page_count(5), 2);
        assert_eq!(DocxExtractor::estimate_page_count(6), 3);
        assert_eq!(DocxExtractor::estimate_page_count(30), 10);
    }

    #[test]
    fn test_replace_text_simple() {
        let mut docx = Docx::default();
        docx.document.paragraphs.push(
            docx_rs::Paragraph::new().add_run(
                docx_rs::Run::new().add_text("Hello world")
            )
        );

        let mut params = std::collections::HashMap::new();
        params.insert("find".to_string(), "world".to_string());
        params.insert("replace".to_string(), "rust".to_string());

        let result = replace_text_in_docx(&docx, &params);
        assert!(result.is_ok());

        let modified_docx = result.unwrap();
        let modified_text = extract_text_from_docx_internal(&modified_docx);
        assert_eq!(modified_text, "Hello rust");
    }

    #[test]
    fn test_replace_text_not_found() {
        let mut docx = Docx::default();
        docx.document.paragraphs.push(
            docx_rs::Paragraph::new().add_run(
                docx_rs::Run::new().add_text("Hello world")
            )
        );

        let mut params = std::collections::HashMap::new();
        params.insert("find".to_string(), "nonexistent".to_string());
        params.insert("replace".to_string(), "rust".to_string());

        let result = replace_text_in_docx(&docx, &params);
        assert!(result.is_ok());

        let modified_docx = result.unwrap();
        let modified_text = extract_text_from_docx_internal(&modified_docx);
        assert_eq!(modified_text, "Hello world");
    }

    #[test]
    fn test_replace_text_empty_find() {
        let docx = Docx::default();
        let mut params = std::collections::HashMap::new();
        params.insert("find".to_string(), "".to_string());
        params.insert("replace".to_string(), "rust".to_string());

        let result = replace_text_in_docx(&docx, &params);
        assert!(result.is_err());
        if let Err(SidecarError::InvalidRequest(msg)) = result {
            assert!(msg.contains("find"));
        } else {
            panic!("Expected InvalidRequest error");
        }
    }

    #[test]
    fn test_insert_paragraph_at_beginning() {
        let mut docx = Docx::default();
        docx.document.paragraphs.push(
            docx_rs::Paragraph::new().add_run(
                docx_rs::Run::new().add_text("First paragraph")
            )
        );

        let mut params = std::collections::HashMap::new();
        params.insert("text".to_string(), "New paragraph".to_string());
        params.insert("position".to_string(), "0".to_string());

        let result = insert_paragraph_in_docx(&docx, &params);
        assert!(result.is_ok());

        let modified_docx = result.unwrap();
        assert_eq!(modified_docx.document.paragraphs.len(), 2);
        let modified_text = extract_text_from_docx_internal(&modified_docx);
        assert_eq!(modified_text, "New paragraph\nFirst paragraph");
    }

    #[test]
    fn test_insert_paragraph_at_end() {
        let mut docx = Docx::default();
        docx.document.paragraphs.push(
            docx_rs::Paragraph::new().add_run(
                docx_rs::Run::new().add_text("First paragraph")
            )
        );

        let mut params = std::collections::HashMap::new();
        params.insert("text".to_string(), "New paragraph".to_string());
        params.insert("position".to_string(), "1".to_string());

        let result = insert_paragraph_in_docx(&docx, &params);
        assert!(result.is_ok());

        let modified_docx = result.unwrap();
        assert_eq!(modified_docx.document.paragraphs.len(), 2);
        let modified_text = extract_text_from_docx_internal(&modified_docx);
        assert_eq!(modified_text, "First paragraph\nNew paragraph");
    }

    #[test]
    fn test_insert_paragraph_invalid_position() {
        let docx = Docx::default();
        let mut params = std::collections::HashMap::new();
        params.insert("text".to_string(), "New paragraph".to_string());
        params.insert("position".to_string(), "999".to_string());

        let result = insert_paragraph_in_docx(&docx, &params);
        assert!(result.is_err());
    }

    #[test]
    fn test_delete_paragraph() {
        let mut docx = Docx::default();
        docx.document.paragraphs.push(
            docx_rs::Paragraph::new().add_run(
                docx_rs::Run::new().add_text("First paragraph")
            )
        );
        docx.document.paragraphs.push(
            docx_rs::Paragraph::new().add_run(
                docx_rs::Run::new().add_text("Second paragraph")
            )
        );

        let mut params = std::collections::HashMap::new();
        params.insert("index".to_string(), "0".to_string());

        let result = delete_paragraph_in_docx(&docx, &params);
        assert!(result.is_ok());

        let modified_docx = result.unwrap();
        assert_eq!(modified_docx.document.paragraphs.len(), 1);
        let modified_text = extract_text_from_docx_internal(&modified_docx);
        assert_eq!(modified_text, "Second paragraph");
    }

    #[test]
    fn test_delete_paragraph_invalid_index() {
        let mut docx = Docx::default();
        docx.document.paragraphs.push(
            docx_rs::Paragraph::new().add_run(
                docx_rs::Run::new().add_text("Only paragraph")
            )
        );

        let mut params = std::collections::HashMap::new();
        params.insert("index".to_string(), "5".to_string());

        let result = delete_paragraph_in_docx(&docx, &params);
        assert!(result.is_err());
        if let Err(SidecarError::InvalidRequest(msg)) = result {
            assert!(msg.contains("index") || msg.contains("out of bounds"));
        } else {
            panic!("Expected InvalidRequest error");
        }
    }

    #[test]
    fn test_extract_docx_too_large() {
        use crate::document::MAX_FILE_SIZE;
        let oversized_docx: Vec<u8> = vec![0u8; MAX_FILE_SIZE + 1];
        let result = extract_text_from_docx(&oversized_docx);

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("exceeds maximum allowed size"));
                assert!(msg.contains("5GB"));
            }
            _ => panic!("Expected InvalidRequest error for oversized file"),
        }
    }
}
