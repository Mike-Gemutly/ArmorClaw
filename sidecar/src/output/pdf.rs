use crate::error::{Result, SidecarError};
use lopdf::content::{Content, Operation};
use lopdf::{dictionary, Document, Object, Stream};
use std::path::Path;
use tracing::debug;

#[derive(Debug, Clone)]
pub struct PdfMetadata {
    pub title: String,
    pub author: String,
    pub subject: String,
    pub creator: String,
}

impl Default for PdfMetadata {
    fn default() -> Self {
        Self {
            title: "ArmorClaw Document".to_string(),
            author: "ArmorClaw".to_string(),
            subject: String::new(),
            creator: "ArmorClaw PDF Generator v6.0".to_string(),
        }
    }
}

const FONT_SIZE: i64 = 12;
const LINE_HEIGHT: i64 = 16;
const MARGIN_LEFT: i64 = 50;
const MARGIN_TOP: i64 = 800;
const MAX_CHARS_PER_LINE: usize = 80;
const PAGE_WIDTH: i64 = 595;
const PAGE_HEIGHT: i64 = 842;

/// Generates a text-searchable PDF. Uses Type1 Helvetica (Latin + Latin-1 Supplement).
/// CJK/Arabic require embedded fonts (future work).
pub fn generate_pdf(text: &str, metadata: Option<PdfMetadata>, output_path: &Path) -> Result<()> {
    if text.trim().is_empty() {
        return Err(SidecarError::InvalidRequest(
            "Cannot generate PDF from empty text".to_string(),
        ));
    }

    let meta = metadata.unwrap_or_default();
    let mut doc = Document::with_version("1.5");
    let pages_id = doc.new_object_id();

    let font_id = doc.add_object(dictionary! {
        "Type" => "Font",
        "Subtype" => "Type1",
        "BaseFont" => "Helvetica",
    });

    let resources_id = doc.add_object(dictionary! {
        "Font" => dictionary! {
            "F1" => font_id,
        },
    });

    let pages = build_pages(text, &mut doc, pages_id);
    let page_ids: Vec<Object> = pages.iter().map(|&(id, _)| id.into()).collect();

    let pages_dict = dictionary! {
        "Type" => "Pages",
        "Kids" => page_ids,
        "Count" => pages.len() as i64,
        "Resources" => resources_id,
    };
    doc.objects.insert(pages_id, Object::Dictionary(pages_dict));

    let catalog_id = doc.add_object(dictionary! {
        "Type" => "Catalog",
        "Pages" => pages_id,
    });
    doc.trailer.set("Root", catalog_id);

    let info_id = doc.add_object(dictionary! {
        "Title" => Object::string_literal(meta.title),
        "Author" => Object::string_literal(meta.author),
        "Subject" => Object::string_literal(meta.subject),
        "Creator" => Object::string_literal(meta.creator),
        "Producer" => Object::string_literal("ArmorClaw PDF Generator"),
    });
    doc.trailer.set("Info", info_id);

    doc.save(output_path)
        .map_err(|e| SidecarError::DocumentProcessingError(format!("Failed to save PDF: {}", e)))?;

    debug!("Generated PDF at {:?}", output_path);
    Ok(())
}

fn build_pages(
    text: &str,
    doc: &mut Document,
    pages_id: lopdf::ObjectId,
) -> Vec<(lopdf::ObjectId, lopdf::ObjectId)> {
    let lines = wrap_text(text, MAX_CHARS_PER_LINE);
    let mut pages = Vec::new();
    let lines_per_page = ((MARGIN_TOP - 50) / LINE_HEIGHT) as usize;
    let mut line_idx = 0;

    while line_idx < lines.len() {
        let end = std::cmp::min(line_idx + lines_per_page, lines.len());
        let page_lines = &lines[line_idx..end];

        let mut operations: Vec<Operation> = Vec::new();
        operations.push(Operation::new("BT", vec![]));
        operations.push(Operation::new("Tf", vec!["F1".into(), FONT_SIZE.into()]));

        let mut y = MARGIN_TOP;
        for line in page_lines {
            operations.push(Operation::new("Td", vec![MARGIN_LEFT.into(), y.into()]));
            operations.push(Operation::new(
                "Tj",
                vec![Object::string_literal(line.as_str())],
            ));
            y -= LINE_HEIGHT;
        }

        operations.push(Operation::new("ET", vec![]));

        let content = Content { operations };
        let content_stream = Stream::new(dictionary! {}, content.encode().unwrap());
        let content_id = doc.add_object(content_stream);

        let page_id = doc.add_object(dictionary! {
            "Type" => "Page",
            "Parent" => pages_id,
            "Contents" => content_id,
            "MediaBox" => vec![0.into(), 0.into(), PAGE_WIDTH.into(), PAGE_HEIGHT.into()],
        });

        pages.push((page_id, content_id));
        line_idx = end;
    }

    pages
}

fn wrap_text(text: &str, max_chars_per_line: usize) -> Vec<String> {
    let mut result = Vec::new();
    for paragraph in text.split('\n') {
        if paragraph.is_empty() {
            result.push(String::new());
            continue;
        }
        let mut current = String::new();
        for word in paragraph.split_whitespace() {
            if current.is_empty() {
                current = word.to_string();
            } else if current.len() + 1 + word.len() <= max_chars_per_line {
                current.push(' ');
                current.push_str(word);
            } else {
                result.push(std::mem::take(&mut current));
                current = word.to_string();
            }
        }
        if !current.is_empty() {
            result.push(current);
        }
    }
    result
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::fs;

    #[test]
    fn test_generate_pdf() {
        let dir = tempfile::tempdir().unwrap();
        let output_path = dir.path().join("test.pdf");

        let result = generate_pdf("Hello World", None, &output_path);
        assert!(
            result.is_ok(),
            "generate_pdf should succeed: {:?}",
            result.err()
        );
        assert!(output_path.exists(), "PDF file should exist on disk");
    }

    #[test]
    fn test_magic_bytes() {
        let dir = tempfile::tempdir().unwrap();
        let output_path = dir.path().join("magic.pdf");

        generate_pdf("Magic bytes test", None, &output_path).unwrap();
        let bytes = fs::read(&output_path).unwrap();

        assert!(
            bytes.starts_with(b"%PDF-"),
            "PDF should start with %PDF- magic bytes, got: {:?}",
            &bytes[..8]
        );
    }

    #[test]
    fn test_non_empty() {
        let dir = tempfile::tempdir().unwrap();
        let output_path = dir.path().join("nonempty.pdf");

        generate_pdf("Some content for size check", None, &output_path).unwrap();
        let metadata = fs::metadata(&output_path).unwrap();

        assert!(
            metadata.len() > 100,
            "PDF file should be larger than 100 bytes, got: {} bytes",
            metadata.len()
        );
    }

    #[test]
    fn test_metadata_set() {
        let dir = tempfile::tempdir().unwrap();
        let output_path = dir.path().join("metadata.pdf");

        let pdf_meta = PdfMetadata {
            title: "Test Title Document".to_string(),
            author: "Test Author".to_string(),
            subject: "Test Subject".to_string(),
            creator: "Test Creator".to_string(),
        };

        generate_pdf("Content with metadata", Some(pdf_meta), &output_path).unwrap();

        let doc = Document::load(output_path).expect("Should load generated PDF");
        if let Ok(info_ref) = doc.trailer.get(b"Info") {
            if let Object::Reference(obj_id) = info_ref {
                if let Ok(info) = doc.get_object(*obj_id) {
                    if let Object::Dictionary(info_dict) = info {
                        if let Ok(title_obj) = info_dict.get(b"Title") {
                            if let Object::String(title_bytes, _) = title_obj {
                                let title = String::from_utf8_lossy(title_bytes);
                                assert_eq!(
                                    title, "Test Title Document",
                                    "PDF title metadata should match"
                                );
                            } else {
                                panic!("Title field is not a string");
                            }
                        } else {
                            panic!("Title field not found in Info dictionary");
                        }

                        if let Ok(author_obj) = info_dict.get(b"Author") {
                            if let Object::String(author_bytes, _) = author_obj {
                                let author = String::from_utf8_lossy(author_bytes);
                                assert_eq!(
                                    author, "Test Author",
                                    "PDF author metadata should match"
                                );
                            }
                        }

                        if let Ok(creator_obj) = info_dict.get(b"Creator") {
                            if let Object::String(creator_bytes, _) = creator_obj {
                                let creator = String::from_utf8_lossy(creator_bytes);
                                assert_eq!(
                                    creator, "Test Creator",
                                    "PDF creator metadata should match"
                                );
                            }
                        }
                    } else {
                        panic!("Info is not a dictionary");
                    }
                } else {
                    panic!("Could not dereference Info object");
                }
            } else {
                panic!("Info is not a reference");
            }
        } else {
            panic!("Info not found in trailer");
        }
    }

    #[test]
    fn test_unicode_latin() {
        let dir = tempfile::tempdir().unwrap();
        let output_path = dir.path().join("unicode.pdf");

        let unicode_text = "Café résumé naïve über Ångström";
        let result = generate_pdf(unicode_text, None, &output_path);

        assert!(
            result.is_ok(),
            "Should handle Latin accented characters: {:?}",
            result.err()
        );

        let bytes = fs::read(&output_path).unwrap();
        assert!(
            bytes.starts_with(b"%PDF-"),
            "Unicode PDF should have valid magic bytes"
        );
    }

    #[test]
    fn test_write_to_file() {
        let dir = tempfile::tempdir().unwrap();
        let specific_path = dir.path().join("subdir").join("specific_output.pdf");

        fs::create_dir_all(specific_path.parent().unwrap()).unwrap();

        generate_pdf("Path-specific content", None, &specific_path).unwrap();

        assert!(
            specific_path.exists(),
            "PDF should be written to the exact specified path"
        );
        let bytes = fs::read(&specific_path).unwrap();
        assert!(
            bytes.starts_with(b"%PDF-"),
            "File at specified path should be valid PDF"
        );
    }

    #[test]
    fn test_empty_text_error() {
        let dir = tempfile::tempdir().unwrap();
        let output_path = dir.path().join("empty.pdf");

        let result = generate_pdf("", None, &output_path);
        assert!(result.is_err(), "Empty text should produce error");
    }

    #[test]
    fn test_default_metadata() {
        let meta = PdfMetadata::default();
        assert_eq!(meta.title, "ArmorClaw Document");
        assert_eq!(meta.author, "ArmorClaw");
        assert_eq!(meta.creator, "ArmorClaw PDF Generator v6.0");
    }

    #[test]
    fn test_wrap_text() {
        let lines = wrap_text("hello world this is a very long line that should wrap", 20);
        assert!(
            lines.len() > 1,
            "Long text should be wrapped into multiple lines"
        );
        for line in &lines {
            assert!(
                line.len() <= 20,
                "No line should exceed max_chars_per_line: '{}' is {} chars",
                line,
                line.len()
            );
        }
    }
}
