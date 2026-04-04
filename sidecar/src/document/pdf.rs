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
}
