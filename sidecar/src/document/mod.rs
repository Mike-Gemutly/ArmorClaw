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
