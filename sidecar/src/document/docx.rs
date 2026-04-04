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

    pub fn extract_from_bytes(&self, docx_bytes: &[u8]) -> Result<DocxTextExtractionResult> {
        if docx_bytes.is_empty() {
            return Err(SidecarError::InvalidRequest(
                "DOCX content is empty".to_string(),
            ));
        }

        validate_file_size(docx_bytes.len())?;

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

pub fn replace_text_in_docx(
    docx: &Docx,
    params: &std::collections::HashMap<String, String>,
) -> Result<Docx> {
    let find = params.get("find").ok_or_else(|| {
        SidecarError::InvalidRequest("Missing 'find' parameter".to_string())
    })?;

    if find.is_empty() {
        return Err(SidecarError::InvalidRequest(
            "'find' parameter cannot be empty".to_string(),
        ));
    }

    let replace = params.get("replace").unwrap_or(&String::new());

    let mut modified_docx = docx.clone();

    for paragraph in &mut modified_docx.document.paragraphs {
        for run in &mut paragraph.runs {
            if let Some(run_text) = &run.text {
                let new_text = run_text.replace(find, replace);
                run.text = Some(new_text);
            }
        }
    }

    Ok(modified_docx)
}

pub fn insert_paragraph_in_docx(
    docx: &Docx,
    params: &std::collections::HashMap<String, String>,
) -> Result<Docx> {
    let text = params.get("text").ok_or_else(|| {
        SidecarError::InvalidRequest("Missing 'text' parameter".to_string())
    })?;

    let position_str = params.get("position").unwrap_or(&"0".to_string());
    let position: usize = position_str.parse().map_err(|_| {
        SidecarError::InvalidRequest(format!("Invalid position: {}", position_str))
    })?;

    let mut modified_docx = docx.clone();

    if position > modified_docx.document.paragraphs.len() {
        return Err(SidecarError::InvalidRequest(format!(
            "Position {} out of bounds (0-{})",
            position,
            modified_docx.document.paragraphs.len()
        )));
    }

    let new_paragraph = docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text(text));
    modified_docx.document.paragraphs.insert(position, new_paragraph);

    Ok(modified_docx)
}

pub fn delete_paragraph_in_docx(
    docx: &Docx,
    params: &std::collections::HashMap<String, String>,
) -> Result<Docx> {
    let index_str = params.get("index").ok_or_else(|| {
        SidecarError::InvalidRequest("Missing 'index' parameter".to_string())
    })?;

    let index: usize = index_str.parse().map_err(|_| {
        SidecarError::InvalidRequest(format!("Invalid index: {}", index_str))
    })?;

    if index >= docx.document.paragraphs.len() {
        return Err(SidecarError::InvalidRequest(format!(
            "Index {} out of bounds (0-{})",
            index,
            docx.document.paragraphs.len().saturating_sub(1)
        )));
    }

    let mut modified_docx = docx.clone();
    modified_docx.document.paragraphs.remove(index);

    Ok(modified_docx)
}

fn extract_text_from_docx_internal(docx: &Docx) -> String {
    let extractor = DocxExtractor::new();
    extractor.extract_text(docx)
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
