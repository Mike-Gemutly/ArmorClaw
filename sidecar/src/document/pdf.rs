use crate::error::{Result, SidecarError};
use lopdf::dictionary;
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

        if let Ok(info) = doc.get_info() {
            Self::extract_string_field(&info, b"Title", "title", &mut metadata);
            Self::extract_string_field(&info, b"Author", "author", &mut metadata);
            Self::extract_string_field(&info, b"Subject", "subject", &mut metadata);
            Self::extract_string_field(&info, b"Keywords", "keywords", &mut metadata);
            Self::extract_string_field(&info, b"Creator", "creator", &mut metadata);
            Self::extract_string_field(&info, b"Producer", "producer", &mut metadata);
            Self::extract_string_field(&info, b"CreationDate", "creation_date", &mut metadata);
            Self::extract_string_field(&info, b"ModDate", "modification_date", &mut metadata);
        }

        metadata
    }

    fn extract_string_field(
        info: &lopdf::Dictionary,
        pdf_key: &[u8],
        field_name: &str,
        metadata: &mut HashMap<String, String>,
    ) {
        if let Some(Object::String(value, _)) = info.get(pdf_key) {
            metadata.insert(field_name.to_string(), value.to_utf8_lossy().to_string());
        }
    }

    fn extract_text_from_pages(&self, doc: &Document) -> Result<String> {
        let pages = doc.get_pages();
        let mut extracted_text = String::new();

        for (page_num, page_id) in pages.iter() {
            let page = doc.get_object(*page_id).map_err(|e| {
                SidecarError::DocumentProcessingError(format!(
                    "Failed to get page {}: {}",
                    page_num, e
                ))
            })?;

            match self.extract_page_text(doc, page) {
                Ok(text_content) => {
                    if !text_content.is_empty() {
                        if !extracted_text.is_empty() {
                            extracted_text.push('\n');
                        }
                        extracted_text.push_str(&text_content);
                    }
                }
                Err(e) => {
                    warn!("Failed to extract text from page {}: {}", page_num, e);
                }
            }
        }

        Ok(extracted_text)
    }

    fn extract_page_text(&self, doc: &Document, page: &Object) -> Result<String> {
        let mut page_text = String::new();

        match doc.get_page_contents(page) {
            Ok(contents) => {
                for content_id in contents {
                    match doc.get_object(content_id) {
                        Ok(content) => {
                            match content.get_content() {
                                Ok(operations) => {
                                    for operation in operations {
                                        match operation.operator.as_str() {
                                            "Tj" => {
                                                self.extract_tj_operation(&operation, &mut page_text);
                                            }
                                            "TJ" => {
                                                self.extract_tj_array_operation(&operation, &mut page_text);
                                            }
                                            _ => {}
                                        }
                                    }
                                }
                                Err(e) => {
                                    debug!("Failed to get content operations: {}", e);
                                }
                            }
                        }
                        Err(e) => {
                            debug!("Failed to get content object: {}", e);
                        }
                    }
                }
            }
            Err(e) => {
                debug!("Failed to get page contents: {}", e);
            }
        }

        Ok(page_text)
    }

    fn decode_pdf_text(text: &[u8]) -> String {
        String::from_utf8_lossy(text).to_string()
    }

    fn extract_tj_operation(&self, operation: &lopdf::Operation, page_text: &mut String) {
        if let Some(Object::String(text, _)) = operation.operands.first() {
            let decoded_text = Self::decode_pdf_text(text);
            if !decoded_text.is_empty() {
                if !page_text.is_empty() {
                    page_text.push(' ');
                }
                page_text.push_str(&decoded_text);
            }
        }
    }

    fn extract_tj_array_operation(&self, operation: &lopdf::Operation, page_text: &mut String) {
        for operand in &operation.operands {
            if let Object::String(text, _) = operand {
                let decoded_text = Self::decode_pdf_text(text);
                if !decoded_text.is_empty() {
                    page_text.push_str(&decoded_text);
                }
            }
        }
    }
}

pub fn extract_text_from_pdf(pdf_bytes: &[u8]) -> Result<PdfTextExtractionResult> {
    let extractor = PdfExtractor::new();
    extractor.extract_from_bytes(pdf_bytes)
}

/// Splits a PDF by extracting specified page ranges
///
/// # Arguments
/// * `pdf_bytes` - The source PDF bytes
/// * `page_ranges` - Comma-separated page ranges (e.g., "1-3,5-7,9"). Pages are 1-indexed.
///
/// # Returns
/// A new PDF containing only the specified pages
///
/// # Errors
/// Returns error if:
/// - PDF is empty or corrupted
/// - Page range format is invalid
/// - Page numbers are out of bounds
pub fn split_pdf(pdf_bytes: &[u8], page_ranges: &str) -> Result<Vec<u8>> {
    if pdf_bytes.is_empty() {
        return Err(SidecarError::InvalidRequest(
            "PDF content is empty".to_string(),
        ));
    }

    let mut doc = Document::load_mem(pdf_bytes).map_err(|e| {
        SidecarError::DocumentProcessingError(format!("Failed to load PDF: {}", e))
    })?;

    let pages = doc.get_pages();
    let total_pages = pages.len();

    if total_pages == 0 {
        return Err(SidecarError::DocumentProcessingError(
            "PDF has no pages".to_string(),
        ));
    }

    let page_indices = parse_page_ranges(page_ranges, total_pages)?;

    if page_indices.is_empty() {
        return Err(SidecarError::InvalidRequest(
            "No valid pages selected".to_string(),
        ));
    }

    let mut selected_pages = Vec::new();
    for (page_num, _) in &pages {
        if page_indices.contains(page_num) {
            if let Some(&page_id) = pages.get(page_num) {
                selected_pages.push(page_id);
            }
        }
    }

    let mut new_doc = Document::with_version("1.5");
    
    for page_id in &selected_pages {
        if let Ok(page_obj) = doc.get_object(*page_id) {
            new_doc.insert_object(page_obj.clone());
        }
    }

    if let Ok(info) = doc.get_info() {
        let info_id = new_doc.insert_object(info.clone());
        new_doc.set_info(info_id);
    }

    new_doc.save_to_bytes().map_err(|e| {
        SidecarError::DocumentProcessingError(format!("Failed to save split PDF: {}", e))
    })
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
pub fn merge_pdfs(pdf_bytes_list: &[&[u8]]) -> Result<Vec<u8>> {
    if pdf_bytes_list.is_empty() {
        return Err(SidecarError::InvalidRequest(
            "No PDFs provided for merge".to_string(),
        ));
    }

    let mut merged_doc = Document::with_version("1.5");

    let first_pdf = &pdf_bytes_list[0];
    if first_pdf.is_empty() {
        return Err(SidecarError::InvalidRequest(
            "First PDF is empty".to_string(),
        ));
    }

    let first_doc = Document::load_mem(first_pdf).map_err(|e| {
        SidecarError::DocumentProcessingError(format!(
            "Failed to load first PDF: {}",
            e
        ))
    })?;

    if let Ok(info) = first_doc.get_info() {
        let info_id = merged_doc.insert_object(info.clone());
        merged_doc.set_info(info_id);
    }

    for (idx, pdf_bytes) in pdf_bytes_list.iter().enumerate() {
        if pdf_bytes.is_empty() {
            return Err(SidecarError::InvalidRequest(format!(
                "PDF at index {} is empty",
                idx
            )));
        }

        let doc = Document::load_mem(pdf_bytes).map_err(|e| {
            SidecarError::DocumentProcessingError(format!(
                "Failed to load PDF at index {}: {}",
                idx, e
            ))
        })?;

        let pages = doc.get_pages();
        for (_page_num, page_id) in pages.iter() {
            if let Ok(page_obj) = doc.get_object(*page_id) {
                merged_doc.insert_object(page_obj.clone());
            }
        }
    }

    merged_doc.save_to_bytes().map_err(|e| {
        SidecarError::DocumentProcessingError(format!("Failed to save merged PDF: {}", e))
    })
}

/// Parses page range string and returns set of page numbers (1-indexed)
///
/// # Arguments
/// * `page_ranges` - Comma-separated ranges (e.g., "1-3,5-7,9")
/// * `total_pages` - Total number of pages in the PDF
///
/// # Returns
/// Sorted set of page numbers
///
/// # Errors
/// Returns error if:
/// - Range format is invalid
/// - Page numbers are out of bounds
fn parse_page_ranges(page_ranges: &str, total_pages: usize) -> Result<std::collections::HashSet<u32>> {
    let mut pages = std::collections::HashSet::new();

    for range_str in page_ranges.split(',') {
        let range_str = range_str.trim();
        
        if range_str.is_empty() {
            continue;
        }

        if range_str.contains('-') {
            let parts: Vec<&str> = range_str.split('-').collect();
            if parts.len() != 2 {
                return Err(SidecarError::InvalidRequest(format!(
                    "Invalid page range format: {}",
                    range_str
                )));
            }

            let start: u32 = parts[0].trim().parse().map_err(|_| {
                SidecarError::InvalidRequest(format!(
                    "Invalid start page number: {}",
                    parts[0]
                ))
            })?;

            let end: u32 = parts[1].trim().parse().map_err(|_| {
                SidecarError::InvalidRequest(format!(
                    "Invalid end page number: {}",
                    parts[1]
                ))
            })?;

            if start == 0 || end == 0 {
                return Err(SidecarError::InvalidRequest(
                    "Page numbers must be 1-indexed (not 0)".to_string(),
                ));
            }

            if start > end {
                return Err(SidecarError::InvalidRequest(format!(
                    "Start page {} cannot be greater than end page {}",
                    start, end
                )));
            }

            if end > total_pages as u32 {
                return Err(SidecarError::InvalidRequest(format!(
                    "End page {} exceeds total pages {}",
                    end, total_pages
                )));
            }

            for page in start..=end {
                pages.insert(page);
            }
        } else {
            let page_num: u32 = range_str.parse().map_err(|_| {
                SidecarError::InvalidRequest(format!(
                    "Invalid page number: {}",
                    range_str
                ))
            })?;

            if page_num == 0 {
                return Err(SidecarError::InvalidRequest(
                    "Page numbers must be 1-indexed (not 0)".to_string(),
                ));
            }

            if page_num > total_pages as u32 {
                return Err(SidecarError::InvalidRequest(format!(
                    "Page {} exceeds total pages {}",
                    page_num, total_pages
                )));
            }

            pages.insert(page_num);
        }
    }

    Ok(pages)
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_simple_pdf_bytes() -> Vec<u8> {
        let mut doc = Document::with_version("1.5");
        
        let pages_id = doc.new_object_id();
        let font_id = doc.new_object_id();
        let page1_id = doc.new_object_id();
        let content_id = doc.new_object_id();

        doc.add_object(font_id, dictionary! {
            "Type" => "Font",
            "Subtype" => "Type1",
            "BaseFont" => "Helvetica",
        });

        doc.add_object(page1_id, dictionary! {
            "Type" => "Page",
            "Parent" => pages_id,
            "Resources" => dictionary! {
                "Font" => dictionary! {
                    "F1" => font_id,
                },
            },
            "Contents" => content_id,
        });

        doc.add_object(content_id, "BT /F1 12 Tf 100 700 Td (Hello, World!) Tj ET".as_bytes());

        let pages_dict = dictionary! {
            "Type" => "Pages",
            "Kids" => vec![page1_id.into()],
            "Count" => 1,
        };

        doc.add_object(pages_id, pages_dict);

        let catalog_id = doc.new_object_id();
        doc.add_object(catalog_id, dictionary! {
            "Type" => "Catalog",
            "Pages" => pages_id,
        });

        doc.set_trailer(dictionary! {
            "Root" => catalog_id,
        });

        doc.save_to_bytes().unwrap()
    }

    fn create_multi_page_pdf_bytes() -> Vec<u8> {
        let mut doc = Document::with_version("1.5");
        
        let pages_id = doc.new_object_id();
        let font_id = doc.new_object_id();
        let page1_id = doc.new_object_id();
        let page2_id = doc.new_object_id();
        let content1_id = doc.new_object_id();
        let content2_id = doc.new_object_id();

        doc.add_object(font_id, dictionary! {
            "Type" => "Font",
            "Subtype" => "Type1",
            "BaseFont" => "Helvetica",
        });

        doc.add_object(page1_id, dictionary! {
            "Type" => "Page",
            "Parent" => pages_id,
            "Resources" => dictionary! {
                "Font" => dictionary! {
                    "F1" => font_id,
                },
            },
            "Contents" => content1_id,
        });

        doc.add_object(content1_id, "BT /F1 12 Tf 100 700 Td (Page 1 content) Tj ET".as_bytes());

        doc.add_object(page2_id, dictionary! {
            "Type" => "Page",
            "Parent" => pages_id,
            "Resources" => dictionary! {
                "Font" => dictionary! {
                    "F1" => font_id,
                },
            },
            "Contents" => content2_id,
        });

        doc.add_object(content2_id, "BT /F1 12 Tf 100 700 Td (Page 2 content) Tj ET".as_bytes());

        let pages_dict = dictionary! {
            "Type" => "Pages",
            "Kids" => vec![page1_id.into(), page2_id.into()],
            "Count" => 2,
        };

        doc.add_object(pages_id, pages_dict);

        let catalog_id = doc.new_object_id();
        doc.add_object(catalog_id, dictionary! {
            "Type" => "Catalog",
            "Pages" => pages_id,
        });

        let info_id = doc.new_object_id();
        doc.add_object(info_id, dictionary! {
            "Title" => "Test PDF Title",
            "Author" => "Test Author",
        });

        doc.set_trailer(dictionary! {
            "Root" => catalog_id,
            "Info" => info_id,
        });

        doc.save_to_bytes().unwrap()
    }

    #[test]
    fn test_extract_text_from_simple_pdf() {
        let pdf_bytes = create_simple_pdf_bytes();
        let result = extract_text_from_pdf(&pdf_bytes).unwrap();

        assert_eq!(result.page_count, 1);
        assert!(result.text.contains("Hello"));
        assert!(result.text.contains("World"));
    }

    #[test]
    fn test_extract_text_from_multi_page_pdf() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = extract_text_from_pdf(&pdf_bytes).unwrap();

        assert_eq!(result.page_count, 2);
        assert!(result.text.contains("Page 1 content"));
        assert!(result.text.contains("Page 2 content"));
    }

    #[test]
    fn test_extract_pdf_with_metadata() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = extract_text_from_pdf(&pdf_bytes).unwrap();

        assert!(result.metadata.contains_key("title"));
        assert_eq!(result.metadata.get("title").unwrap(), "Test PDF Title");
        assert!(result.metadata.contains_key("author"));
        assert_eq!(result.metadata.get("author").unwrap(), "Test Author");
    }

    #[test]
    fn test_extract_empty_pdf() {
        let empty_pdf: Vec<u8> = vec![];
        let result = extract_text_from_pdf(&empty_pdf);

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("empty"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_extract_invalid_pdf() {
        let invalid_pdf: Vec<u8> = b"This is not a PDF file".to_vec();
        let result = extract_text_from_pdf(&invalid_pdf);

        assert!(result.is_err());
        match result {
            Err(SidecarError::DocumentProcessingError(msg)) => {
                assert!(msg.contains("Failed to load PDF"));
            }
            _ => panic!("Expected DocumentProcessingError"),
        }
    }

    #[test]
    fn test_pdf_extractor_new() {
        let extractor = PdfExtractor::new();
        let pdf_bytes = create_simple_pdf_bytes();
        let result = extractor.extract_from_bytes(&pdf_bytes).unwrap();

        assert_eq!(result.page_count, 1);
    }

    #[test]
    fn test_extracted_text_not_empty() {
        let pdf_bytes = create_simple_pdf_bytes();
        let result = extract_text_from_pdf(&pdf_bytes).unwrap();

        assert!(!result.text.is_empty());
    }

    #[test]
    fn test_split_pdf_single_page() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "1").unwrap();

        assert!(!result.is_empty());
        
        let split_doc = Document::load_mem(&result).unwrap();
        assert_eq!(split_doc.get_pages().len(), 1);
    }

    #[test]
    fn test_split_pdf_page_range() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "1-2").unwrap();

        assert!(!result.is_empty());
        
        let split_doc = Document::load_mem(&result).unwrap();
        assert_eq!(split_doc.get_pages().len(), 2);
    }

    #[test]
    fn test_split_pdf_multiple_ranges() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "1,2").unwrap();

        assert!(!result.is_empty());
        
        let split_doc = Document::load_mem(&result).unwrap();
        assert_eq!(split_doc.get_pages().len(), 2);
    }

    #[test]
    fn test_split_pdf_empty_ranges() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "");

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("No valid pages selected"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_split_pdf_invalid_page_number() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "99");

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("exceeds total pages"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_split_pdf_zero_indexed() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "0");

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("1-indexed"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_split_pdf_invalid_range_format() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "1-2-3");

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("Invalid page range format"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_split_pdf_start_greater_than_end() {
        let pdf_bytes = create_multi_page_pdf_bytes();
        let result = split_pdf(&pdf_bytes, "5-3");

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("cannot be greater than"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_split_pdf_empty_input() {
        let empty_pdf: Vec<u8> = vec![];
        let result = split_pdf(&empty_pdf, "1");

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("empty"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_split_pdf_invalid_pdf() {
        let invalid_pdf: Vec<u8> = b"Not a PDF".to_vec();
        let result = split_pdf(&invalid_pdf, "1");

        assert!(result.is_err());
        match result {
            Err(SidecarError::DocumentProcessingError(msg)) => {
                assert!(msg.contains("Failed to load PDF"));
            }
            _ => panic!("Expected DocumentProcessingError"),
        }
    }

    #[test]
    fn test_merge_pdfs_two_pdfs() {
        let pdf1 = create_simple_pdf_bytes();
        let pdf2 = create_multi_page_pdf_bytes();
        
        let pdf_list = vec![pdf1.as_slice(), pdf2.as_slice()];
        let result = merge_pdfs(&pdf_list).unwrap();

        assert!(!result.is_empty());
        
        let merged_doc = Document::load_mem(&result).unwrap();
        assert_eq!(merged_doc.get_pages().len(), 3);
    }

    #[test]
    fn test_merge_pdfs_single_pdf() {
        let pdf1 = create_simple_pdf_bytes();
        
        let pdf_list = vec![pdf1.as_slice()];
        let result = merge_pdfs(&pdf_list).unwrap();

        assert!(!result.is_empty());
        
        let merged_doc = Document::load_mem(&result).unwrap();
        assert_eq!(merged_doc.get_pages().len(), 1);
    }

    #[test]
    fn test_merge_pdfs_empty_list() {
        let pdf_list: Vec<&[u8]> = vec![];
        let result = merge_pdfs(&pdf_list);

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("No PDFs provided"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_merge_pdfs_empty_pdf_in_list() {
        let pdf1 = create_simple_pdf_bytes();
        let empty_pdf: Vec<u8> = vec![];
        
        let pdf_list = vec![pdf1.as_slice(), empty_pdf.as_slice()];
        let result = merge_pdfs(&pdf_list);

        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("empty"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_merge_pdfs_invalid_pdf_in_list() {
        let pdf1 = create_simple_pdf_bytes();
        let invalid_pdf: Vec<u8> = b"Not a PDF".to_vec();
        
        let pdf_list = vec![pdf1.as_slice(), invalid_pdf.as_slice()];
        let result = merge_pdfs(&pdf_list);

        assert!(result.is_err());
        match result {
            Err(SidecarError::DocumentProcessingError(msg)) => {
                assert!(msg.contains("Failed to load PDF"));
            }
            _ => panic!("Expected DocumentProcessingError"),
        }
    }

    #[test]
    fn test_merge_pdfs_preserves_metadata() {
        let pdf1 = create_multi_page_pdf_bytes();
        let pdf2 = create_simple_pdf_bytes();
        
        let pdf_list = vec![pdf1.as_slice(), pdf2.as_slice()];
        let result = merge_pdfs(&pdf_list).unwrap();

        assert!(!result.is_empty());
        
        let merged_doc = Document::load_mem(&result).unwrap();
        let info = merged_doc.get_info().unwrap();
        
        assert!(info.contains_key(b"Title"));
    }
}
