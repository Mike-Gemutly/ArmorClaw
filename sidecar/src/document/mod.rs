pub mod pdf;

pub use pdf::{
    PdfExtractor, 
    PdfTextExtractionResult, 
    extract_text_from_pdf,
    split_pdf,
    merge_pdfs,
};
