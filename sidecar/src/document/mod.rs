use crate::error::{Result, SidecarError};

pub const MAX_FILE_SIZE: usize = 5 * 1024 * 1024 * 1024; // 5GB in bytes

pub fn validate_file_size(size: usize) -> Result<()> {
    if size > MAX_FILE_SIZE {
        return Err(SidecarError::InvalidRequest(format!(
            "File size {} bytes exceeds maximum allowed size {} bytes (5GB)",
            size, MAX_FILE_SIZE
        )));
    }
    Ok(())
}

pub mod pdf;
pub mod docx;

pub use pdf::{
    PdfExtractor,
    PdfTextExtractionResult,
    extract_text_from_pdf,
    split_pdf,
    merge_pdfs,
};

pub use docx::{
    DocxExtractor,
    DocxTextExtractionResult,
    extract_text_from_docx,
};

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_validate_file_size_within_limit() {
        let result = validate_file_size(1024);
        assert!(result.is_ok());
    }

    #[test]
    fn test_validate_file_size_at_limit() {
        let result = validate_file_size(MAX_FILE_SIZE);
        assert!(result.is_ok());
    }

    #[test]
    fn test_validate_file_size_exceeds_limit() {
        let result = validate_file_size(MAX_FILE_SIZE + 1);
        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => {
                assert!(msg.contains("exceeds maximum allowed size"));
                assert!(msg.contains("5GB"));
            }
            _ => panic!("Expected InvalidRequest error"),
        }
    }

    #[test]
    fn test_validate_file_size_zero() {
        let result = validate_file_size(0);
        assert!(result.is_ok());
    }

    #[test]
    fn test_validate_file_size_one_byte_over_limit() {
        let result = validate_file_size(MAX_FILE_SIZE + 1);
        assert!(result.is_err());
    }
}
