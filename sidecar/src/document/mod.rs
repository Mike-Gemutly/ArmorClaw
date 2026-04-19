use crate::error::{Result, SidecarError};

pub use crate::security::shadowmap::{PiiCategory, ShadowMap};

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
pub mod xlsx;
pub mod pptx;
pub mod ocr;
pub mod diff;
pub mod rag;
pub mod html_diff;
pub mod docx_diff;
pub mod embeddings;
pub mod qdrant;
pub mod convert;

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
    replace_text_in_docx,
    insert_paragraph_in_docx,
    delete_paragraph_in_docx,
};

pub use xlsx::{
    XlsxExtractor,
    XlsxExtractionResult,
    SheetData,
    extract_data_from_xlsx,
};

pub use pptx::{
    PptxExtractor,
    PptxTextExtractionResult,
    extract_text_from_pptx,
};

pub use ocr::{
    OcrExtractor,
    OcrResult,
    OcrConfig,
    extract_text_with_ocr,
    validate_language,
    get_supported_languages,
    detect_language_from_text,
};

pub use diff::{
    DiffEngine,
    DiffResult,
    DiffOp,
    compute_text_diff,
};

pub use rag::{
    TextChunker,
    TextChunk,
    ChunkingStrategy,
};

pub use html_diff::{
    HtmlDiffGenerator,
};

pub use docx_diff::{
    DocxDiffGenerator,
    generate_redline_document,
};

pub use embeddings::{
    EmbeddingGenerator,
    generate_text_embedding,
    Embedder,
    OpenAIEmbedder,
};

pub use convert::{
    convert_docx_to_pdf,
    convert_xlsx_to_csv,
    convert_pptx_to_pdf,
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
