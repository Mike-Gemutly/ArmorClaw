use crate::error::{Result, SidecarError};
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

    pub fn extract_from_bytes(&self, docx_bytes: &[u8]) -> Result<DocxTextExtractionResult> {
        if docx_bytes.is_empty() {
            return Err(SidecarError::InvalidRequest(
                "DOCX content is empty".to_string(),
            ));
        }

        let cursor = Cursor::new(docx_bytes);
        let docx = read_docx(cursor).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to load DOCX: {}", e))
        })?;

        let metadata = Self::extract_metadata(&docx);
        let text = self.extract_text(&docx);

        let paragraph_count = self.count_paragraphs(&docx);
        let page_count = Self::estimate_page_count(paragraph_count);

        if text.is_empty() {
            return Err(SidecarError::DocumentProcessingError(
                "DOCX contains no extractable text".to_string(),
            ));
        }

        Ok(DocxTextExtractionResult {
            text,
            page_count,
            metadata,
        })
    }

    fn extract_metadata(docx: &Docx) -> HashMap<String, String> {
        let mut metadata = HashMap::new();

        if let Some(core_props) = &docx.core_properties {
            if !core_props.title.is_empty() {
                metadata.insert("title".to_string(), core_props.title.clone());
            }
            if !core_props.creator.is_empty() {
                metadata.insert("author".to_string(), core_props.creator.clone());
            }
            if !core_props.subject.is_empty() {
                metadata.insert("subject".to_string(), core_props.subject.clone());
            }
            if !core_props.description.is_empty() {
                metadata.insert("description".to_string(), core_props.description.clone());
            }
            if !core_props.keywords.is_empty() {
                metadata.insert("keywords".to_string(), core_props.keywords.clone());
            }
        }

        metadata
    }

    fn extract_text(&self, docx: &Docx) -> String {
        let mut extracted_text = String::new();

        for paragraph in &docx.document.paragraphs {
            let paragraph_text = self.extract_paragraph_text(paragraph);
            if !paragraph_text.is_empty() {
                if !extracted_text.is_empty() {
                    extracted_text.push('\n');
                }
                extracted_text.push_str(&paragraph_text);
            }
        }

        extracted_text
    }

    fn extract_paragraph_text(&self, paragraph: &docx_rs::Paragraph) -> String {
        let mut text = String::new();

        for run in &paragraph.runs {
            if let Some(run_text) = &run.text {
                text.push_str(run_text);
            }
        }

        text
    }

    fn count_paragraphs(&self, docx: &Docx) -> usize {
        docx.document.paragraphs.len()
    }

    fn estimate_page_count(paragraph_count: usize) -> i32 {
        if paragraph_count == 0 {
            return 0;
        }
        ((paragraph_count + 2) / 3) as i32
    }
}

pub fn extract_text_from_docx(docx_bytes: &[u8]) -> Result<DocxTextExtractionResult> {
    let extractor = DocxExtractor::new();
    extractor.extract_from_bytes(docx_bytes)
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
}
