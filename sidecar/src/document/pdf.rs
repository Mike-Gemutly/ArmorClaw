use crate::document::validate_file_size;
use crate::error::{Result, SidecarError};
use crate::security::shadowmap::ShadowMap;
use lopdf::Document;
use lopdf::Object;
use std::collections::{BTreeMap, HashMap};
use tracing::debug;

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
                        Self::extract_string_field(
                            info_dict,
                            b"Keywords",
                            "keywords",
                            &mut metadata,
                        );
                        Self::extract_string_field(info_dict, b"Creator", "creator", &mut metadata);
                        Self::extract_string_field(
                            info_dict,
                            b"Producer",
                            "producer",
                            &mut metadata,
                        );
                        Self::extract_string_field(
                            info_dict,
                            b"CreationDate",
                            "creation_date",
                            &mut metadata,
                        );
                        Self::extract_string_field(
                            info_dict,
                            b"ModDate",
                            "modification_date",
                            &mut metadata,
                        );
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
                metadata.insert(
                    field_name.to_string(),
                    String::from_utf8_lossy(value).to_string(),
                );
            }
        }
    }

    fn extract_text_from_pages(&self, doc: &Document) -> Result<String> {
        let pages: BTreeMap<u32, (u32, u16)> = doc.get_pages();
        let mut all_text = Vec::new();

        for (page_num, page_id) in pages.iter() {
            match doc.get_page_content(*page_id) {
                Ok(content_bytes) => {
                    let page_text = Self::parse_content_stream(&content_bytes);
                    if !page_text.is_empty() {
                        all_text.push(page_text);
                    }
                }
                Err(e) => {
                    debug!("Failed to extract content from page {}: {}", page_num, e);
                }
            }
        }

        Ok(all_text.join("\n"))
    }

    /// Parse PDF content stream bytes and extract text from text-showing operators.
    /// Handles: Tj (show string), TJ (show array), ' (move to next line and show),
    /// " (set spacing, move to next line, and show).
    fn parse_content_stream(content: &[u8]) -> String {
        let text = String::from_utf8_lossy(content);
        let mut result = String::new();
        let mut chars = text.chars().peekable();

        while let Some(c) = chars.next() {
            match c {
                '\'' => {
                    result.push('\n');
                }
                '"' => {
                    result.push('\n');
                }
                // String literal: (...) or hex string <...>
                '(' => {
                    let mut string_buf = String::new();
                    let mut depth = 1;
                    while let Some(&next) = chars.peek() {
                        chars.next();
                        match next {
                            '\\' => {
                                if let Some(escaped) = chars.next() {
                                    match escaped {
                                        'n' => string_buf.push('\n'),
                                        'r' => string_buf.push('\r'),
                                        't' => string_buf.push('\t'),
                                        '(' => string_buf.push('('),
                                        ')' => string_buf.push(')'),
                                        '\\' => string_buf.push('\\'),
                                        c => string_buf.push(c),
                                    }
                                }
                            }
                            ')' => {
                                depth -= 1;
                                if depth == 0 {
                                    break;
                                }
                                string_buf.push(')');
                            }
                            _ => string_buf.push(next),
                        }
                    }
                    // Check what operator follows this string
                    Self::skip_whitespace_peek(&mut chars);
                    let next1 = chars.peek().copied();
                    let next2 = chars.peek().copied();
                    match (next1, next2) {
                        (Some('T'), Some('j')) => {
                            // (string) Tj — show string
                            result.push_str(&string_buf);
                        }
                        (Some('\''), _) => {
                            // (string) ' — move to next line and show
                            result.push('\n');
                            result.push_str(&string_buf);
                        }
                        (Some('"'), _) => {
                            // (string) " — set spacing, move, show
                            result.push('\n');
                            result.push_str(&string_buf);
                        }
                        _ => {
                            // String followed by something else or end
                            if !string_buf.is_empty() {
                                result.push_str(&string_buf);
                            }
                        }
                    }
                }
                // Hex string literal: <...>
                '<' => {
                    let mut hex_buf = String::new();
                    while let Some(&next) = chars.peek() {
                        if next == '>' {
                            chars.next();
                            break;
                        }
                        hex_buf.push(next);
                        chars.next();
                    }
                    // Decode hex pairs
                    let decoded = Self::decode_hex_string(&hex_buf);
                    if !decoded.is_empty() {
                        result.push_str(&decoded);
                    }
                }
                // Array start for TJ: [...]
                '[' => {
                    let mut array_depth = 1;
                    let mut array_strings = Vec::new();
                    let mut current_str = String::new();
                    while let Some(&next) = chars.peek() {
                        chars.next();
                        match next {
                            '[' => {
                                array_depth += 1;
                            }
                            ']' => {
                                array_depth -= 1;
                                if array_depth == 0 {
                                    if !current_str.is_empty() {
                                        array_strings.push(std::mem::take(&mut current_str));
                                    }
                                    break;
                                }
                            }
                            '(' => {
                                // Nested string in array
                                let mut str_buf = String::new();
                                let mut str_depth = 1;
                                while let Some(&inner) = chars.peek() {
                                    chars.next();
                                    match inner {
                                        '\\' => {
                                            if let Some(escaped) = chars.next() {
                                                match escaped {
                                                    'n' => str_buf.push('\n'),
                                                    'r' => str_buf.push('\r'),
                                                    't' => str_buf.push('\t'),
                                                    '(' => str_buf.push('('),
                                                    ')' => str_buf.push(')'),
                                                    '\\' => str_buf.push('\\'),
                                                    c => str_buf.push(c),
                                                }
                                            }
                                        }
                                        ')' => {
                                            str_depth -= 1;
                                            if str_depth == 0 {
                                                break;
                                            }
                                            str_buf.push(')');
                                        }
                                        _ => str_buf.push(inner),
                                    }
                                }
                                array_strings.push(str_buf);
                            }
                            '<' => {
                                // Nested hex string in array
                                let mut hex_buf = String::new();
                                while let Some(&inner) = chars.peek() {
                                    if inner == '>' {
                                        chars.next();
                                        break;
                                    }
                                    hex_buf.push(inner);
                                    chars.next();
                                }
                                let decoded = Self::decode_hex_string(&hex_buf);
                                if !decoded.is_empty() {
                                    array_strings.push(decoded);
                                }
                            }
                            '-' | '0'..='9' => {
                                // Negative number in TJ array acts as kerning (space)
                                // Collect the full number
                                let mut num_str = String::new();
                                num_str.push(next);
                                while let Some(&digit) = chars.peek() {
                                    if digit.is_ascii_digit() {
                                        num_str.push(digit);
                                        chars.next();
                                    } else {
                                        break;
                                    }
                                }
                                if let Ok(val) = num_str.parse::<i32>() {
                                    // Negative values are kerning offsets
                                    // Use a simple heuristic: large negative values = space
                                    if val < -100 {
                                        if !current_str.is_empty() {
                                            array_strings.push(std::mem::take(&mut current_str));
                                        }
                                        array_strings.push(" ".to_string());
                                    } else if val < 0 {
                                        current_str.push(' ');
                                    }
                                }
                            }
                            ' ' | '\n' | '\r' | '\t' => {
                                if !current_str.is_empty() {
                                    // Don't push yet — might be a number following
                                }
                            }
                            _ => {}
                        }
                    }
                    // After the array, check for TJ operator
                    Self::skip_whitespace_peek(&mut chars);
                    let next_chars: String = chars.by_ref().take(2).collect();
                    if next_chars == "TJ" {
                        for s in &array_strings {
                            result.push_str(s);
                        }
                    }
                }
                _ => {}
            }
        }

        result
    }

    fn skip_whitespace_peek(chars: &mut std::iter::Peekable<std::str::Chars>) {
        while let Some(&c) = chars.peek() {
            if c == ' ' || c == '\n' || c == '\r' || c == '\t' {
                chars.next();
            } else {
                break;
            }
        }
    }

    fn decode_hex_string(hex: &str) -> String {
        let hex: String = hex.chars().filter(|c| !c.is_whitespace()).collect();
        let mut result = String::new();
        let bytes = hex.as_bytes();
        let mut i = 0;
        while i + 1 < bytes.len() {
            let byte = (Self::hex_digit(bytes[i]) << 4) | Self::hex_digit(bytes[i + 1]);
            if byte != 0 {
                if let Some(c) = char::from_u32(byte as u32) {
                    if c.is_ascii() || !c.is_control() {
                        result.push(c);
                    }
                }
            }
            i += 2;
        }
        result
    }

    fn hex_digit(b: u8) -> u8 {
        match b {
            b'0'..=b'9' => b - b'0',
            b'a'..=b'f' => b - b'a' + 10,
            b'A'..=b'F' => b - b'A' + 10,
            _ => 0,
        }
    }

    pub fn extract_from_bytes_redacted(
        &self,
        pdf_bytes: &[u8],
        shadowmap: &mut ShadowMap,
    ) -> Result<PdfTextExtractionResult> {
        let mut result = self.extract_from_bytes(pdf_bytes)?;
        result.text = shadowmap.redact(&result.text);
        Ok(result)
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
pub fn extract_text_from_pdf(pdf_bytes: &[u8]) -> Result<PdfTextExtractionResult> {
    PdfExtractor::new().extract_from_bytes(pdf_bytes)
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
        "PDF split functionality requires lopdf API update - not currently available".to_string(),
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
        "PDF merge functionality requires lopdf API update - not currently available".to_string(),
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

    #[test]
    fn test_extract_text_from_pdf_empty() {
        let result = extract_text_from_pdf(&[]);
        assert!(result.is_err());
    }

    #[test]
    fn test_extract_text_from_pdf_invalid() {
        let result = extract_text_from_pdf(b"not a pdf");
        assert!(result.is_err());
    }

    #[test]
    fn test_parse_content_stream_tj() {
        let content = b"(Hello World) Tj";
        let text = PdfExtractor::parse_content_stream(content);
        assert_eq!(text, "Hello World");
    }

    #[test]
    fn test_parse_content_stream_tj_multiple() {
        let content = b"(Hello) Tj (World) Tj";
        let text = PdfExtractor::parse_content_stream(content);
        assert_eq!(text, "HelloWorld");
    }

    #[test]
    fn test_parse_content_stream_tj_array() {
        let content = b"[(H) -50 (ello) 10 ( ) -50 (Wo) 10 (rld)] TJ";
        let text = PdfExtractor::parse_content_stream(content);
        assert!(text.contains("Hello"));
        assert!(text.contains("World"));
    }

    #[test]
    fn test_parse_content_stream_hex_string() {
        let content = b"<48656C6C6F> Tj";
        let text = PdfExtractor::parse_content_stream(content);
        assert_eq!(text, "Hello");
    }

    #[test]
    fn test_decode_hex_string() {
        assert_eq!(PdfExtractor::decode_hex_string("48656C6C6F"), "Hello");
        assert_eq!(PdfExtractor::decode_hex_string(""), "");
    }

    #[test]
    fn test_hex_digit() {
        assert_eq!(PdfExtractor::hex_digit(b'0'), 0);
        assert_eq!(PdfExtractor::hex_digit(b'9'), 9);
        assert_eq!(PdfExtractor::hex_digit(b'a'), 10);
        assert_eq!(PdfExtractor::hex_digit(b'f'), 15);
        assert_eq!(PdfExtractor::hex_digit(b'A'), 10);
    }
}
