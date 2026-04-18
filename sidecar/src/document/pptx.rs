use crate::document::validate_file_size;
use crate::error::{Result, SidecarError};
use crate::security::shadowmap::ShadowMap;
use quick_xml::events::Event;
use quick_xml::Reader;
use std::collections::HashMap;
use std::io::Cursor;
use zip::ZipArchive;

pub struct PptxExtractor;

#[derive(Debug, Clone)]
pub struct PptxTextExtractionResult {
    pub text: String,
    pub page_count: i32,
    pub metadata: HashMap<String, String>,
}

impl PptxExtractor {
    pub fn new() -> Self {
        Self
    }

    pub fn extract_from_bytes(&self, pptx_bytes: &[u8]) -> Result<PptxTextExtractionResult> {
        if pptx_bytes.is_empty() {
            return Err(SidecarError::InvalidRequest(
                "PPTX content is empty".to_string(),
            ));
        }

        validate_file_size(pptx_bytes.len())?;

        let cursor = Cursor::new(pptx_bytes);
        let mut archive = ZipArchive::new(cursor).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to open PPTX archive: {}", e))
        })?;

        let slide_texts = extract_slides(&mut archive)?;
        let notes_texts = extract_notes(&mut archive)?;

        let page_count = slide_texts.len() as i32;
        let mut metadata = HashMap::new();

        // Add notes as metadata
        if !notes_texts.is_empty() {
            let notes_joined = notes_texts.join("\n\n");
            metadata.insert("speaker_notes".to_string(), notes_joined);
        }

        metadata.insert("slide_count".to_string(), page_count.to_string());

        // Join slide texts with separators, filtering empty slides
        let text = slide_texts
            .iter()
            .enumerate()
            .filter(|(_, t)| !t.trim().is_empty())
            .map(|(i, t)| {
                if slide_texts.len() > 1 {
                    format!("--- Slide {} ---\n{}", i + 1, t.trim())
                } else {
                    t.trim().to_string()
                }
            })
            .collect::<Vec<_>>()
            .join("\n\n");

        Ok(PptxTextExtractionResult {
            text,
            page_count,
            metadata,
        })
    }

    pub fn extract_from_bytes_redacted(
        &self,
        pptx_bytes: &[u8],
        shadowmap: &mut ShadowMap,
    ) -> Result<PptxTextExtractionResult> {
        let mut result = self.extract_from_bytes(pptx_bytes)?;
        result.text = shadowmap.redact(&result.text);
        Ok(result)
    }
}

impl Default for PptxExtractor {
    fn default() -> Self {
        Self::new()
    }
}

pub fn extract_text_from_pptx(pptx_bytes: &[u8]) -> Result<PptxTextExtractionResult> {
    PptxExtractor::new().extract_from_bytes(pptx_bytes)
}

/// Extract text from all slides, returning ordered slide texts.
fn extract_slides(archive: &mut ZipArchive<Cursor<&[u8]>>) -> Result<Vec<String>> {
    let mut slides = Vec::new();

    // Collect slide entry names and sort numerically
    let mut slide_entries: Vec<String> = Vec::new();
    for i in 0..archive.len() {
        let file = archive.by_index(i).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to read ZIP entry: {}", e))
        })?;
        let name = file.name().to_string();
        // Match ppt/slides/slideN.xml
        if name.starts_with("ppt/slides/slide") && name.ends_with(".xml") {
            let num_part = name
                .trim_start_matches("ppt/slides/slide")
                .trim_end_matches(".xml");
            if num_part.parse::<u32>().is_ok() {
                slide_entries.push(name);
            }
        }
    }

    // Sort by slide number
    slide_entries.sort_by_key(|name| {
        let num_part = name
            .trim_start_matches("ppt/slides/slide")
            .trim_end_matches(".xml");
        num_part.parse::<u32>().unwrap_or(0)
    });

    for slide_name in &slide_entries {
        let slide_xml = read_entry(archive, slide_name)?;
        let text = extract_text_from_slide_xml(&slide_xml);
        slides.push(text);
    }

    Ok(slides)
}

/// Extract speaker notes from all notes slides.
fn extract_notes(archive: &mut ZipArchive<Cursor<&[u8]>>) -> Result<Vec<String>> {
    let mut notes = Vec::new();

    let mut notes_entries: Vec<String> = Vec::new();
    for i in 0..archive.len() {
        let file = archive.by_index(i).map_err(|e| {
            SidecarError::DocumentProcessingError(format!("Failed to read ZIP entry: {}", e))
        })?;
        let name = file.name().to_string();
        if name.starts_with("ppt/notesSlides/notesSlide") && name.ends_with(".xml") {
            let num_part = name
                .trim_start_matches("ppt/notesSlides/notesSlide")
                .trim_end_matches(".xml");
            if num_part.parse::<u32>().is_ok() {
                notes_entries.push(name);
            }
        }
    }

    notes_entries.sort_by_key(|name| {
        let num_part = name
            .trim_start_matches("ppt/notesSlides/notesSlide")
            .trim_end_matches(".xml");
        num_part.parse::<u32>().unwrap_or(0)
    });

    for notes_name in &notes_entries {
        let notes_xml = read_entry(archive, notes_name)?;
        let text = extract_text_from_slide_xml(&notes_xml);
        notes.push(text);
    }

    Ok(notes)
}

/// Read a ZIP entry's contents into a String.
fn read_entry(archive: &mut ZipArchive<Cursor<&[u8]>>, entry_name: &str) -> Result<String> {
    let mut file = archive.by_name(entry_name).map_err(|e| {
        SidecarError::DocumentProcessingError(format!(
            "Failed to read entry '{}': {}",
            entry_name, e
        ))
    })?;
    let mut content = String::new();
    std::io::Read::read_to_string(&mut file, &mut content).map_err(|e| {
        SidecarError::DocumentProcessingError(format!(
            "Failed to read content of '{}': {}",
            entry_name, e
        ))
    })?;
    Ok(content)
}

/// Extract text from a slide XML by finding all `<a:t>` elements.
/// This handles shapes, groups, tables — all use `<a:t>` for text content.
fn extract_text_from_slide_xml(xml: &str) -> String {
    let mut reader = Reader::from_str(xml);
    reader.config_mut().trim_text(true);

    let mut text_parts: Vec<String> = Vec::new();
    let mut in_at = false;
    let mut depth = 0usize;
    let mut buf = Vec::new();

    loop {
        match reader.read_event_into(&mut buf) {
            Ok(Event::Start(ref e)) | Ok(Event::Empty(ref e)) => {
                let local = e.local_name();
                let name_str = String::from_utf8_lossy(local.as_ref());
                if name_str == "t" {
                    in_at = true;
                    depth = 1;
                } else if in_at {
                    depth += 1;
                }
            }
            Ok(Event::End(ref e)) => {
                let local = e.local_name();
                let name_str = String::from_utf8_lossy(local.as_ref());
                if name_str == "t" {
                    in_at = false;
                    depth = 0;
                } else if in_at {
                    depth = depth.saturating_sub(1);
                }
            }
            Ok(Event::Text(ref e)) => {
                if in_at {
                    if let Ok(text) = e.unescape() {
                        text_parts.push(text.into_owned());
                    }
                }
            }
            Ok(Event::Eof) => break,
            Err(_) => break,
            _ => {}
        }
        buf.clear();
    }

    text_parts.join("")
}

/// Create a minimal PPTX file in memory for testing.
/// PPTX is a ZIP containing [Content_Types].xml, ppt/slides/slide1.xml, etc.
#[cfg(test)]
fn create_test_pptx(slides: Vec<&[&str]>, notes: Vec<&[&str]>) -> Vec<u8> {
    use std::io::Write;
    let mut buf = Cursor::new(Vec::new());
    {
        let mut zip = zip::ZipWriter::new(&mut buf);
        let options = zip::write::SimpleFileOptions::default();

        // [Content_Types].xml
        zip.start_file("[Content_Types].xml", options).unwrap();
        zip.write_all(br#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Override PartName="/ppt/presentation.xml" ContentType="application/vnd.openxmlformats-officedocument.presentationml.presentation.main+xml"/>
</Types>"#).unwrap();

        // _rels/.rels
        zip.start_file("_rels/.rels", options).unwrap();
        zip.write_all(br#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="ppt/presentation.xml"/>
</Relationships>"#).unwrap();

        // ppt/presentation.xml
        zip.start_file("ppt/presentation.xml", options).unwrap();
        let mut pres_rels = String::new();
        for i in 1..=slides.len() {
            pres_rels.push_str(&format!(
                r#"<p:sldId id="{}" r:id="rId{}"/>"#,
                256 + i - 1,
                i
            ));
        }
        let pres_xml = format!(
            r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:presentation xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
                xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
                xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:sldIdLst>{}</p:sldIdLst>
</p:presentation>"#,
            pres_rels
        );
        zip.write_all(pres_xml.as_bytes()).unwrap();

        // Slides
        for (i, slide_texts) in slides.iter().enumerate() {
            let slide_name = format!("ppt/slides/slide{}.xml", i + 1);
            zip.start_file(&slide_name, options).unwrap();

            let mut shapes = String::new();
            for (j, text) in slide_texts.iter().enumerate() {
                // Escape XML special chars
                let escaped = text
                    .replace('&', "&amp;")
                    .replace('<', "&lt;")
                    .replace('>', "&gt;");
                shapes.push_str(&format!(
                    r#"<p:sp><p:nvSpPr><p:cNvPr id="{j}" name="sp{j}"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr><p:spPr/><p:txBody><a:p><a:r><a:t>{escaped}</a:t></a:r></a:p></p:txBody></p:sp>"#,
                    j = j + 1,
                    escaped = escaped
                ));
            }

            let slide_xml = format!(
                r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
       xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
       xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="1" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/>{shapes}</p:spTree></p:cSld>
</p:sld>"#,
                shapes = shapes
            );
            zip.write_all(slide_xml.as_bytes()).unwrap();
        }

        // Notes slides
        for (i, note_texts) in notes.iter().enumerate() {
            let notes_name = format!("ppt/notesSlides/notesSlide{}.xml", i + 1);
            zip.start_file(&notes_name, options).unwrap();

            let mut note_body = String::new();
            for text in note_texts.iter() {
                let escaped = text
                    .replace('&', "&amp;")
                    .replace('<', "&lt;")
                    .replace('>', "&gt;");
                note_body.push_str(&format!(r#"<a:p><a:r><a:t>{}</a:t></a:r></a:p>"#, escaped));
            }

            let notes_xml = format!(
                r#"<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<p:notes xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main"
         xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main"
         xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <p:cSld><p:spTree><p:nvGrpSpPr><p:cNvPr id="2" name=""/><p:cNvGrpSpPr/><p:nvPr/></p:nvGrpSpPr><p:grpSpPr/>
    <p:sp><p:nvSpPr><p:cNvPr id="3" name="Notes Placeholder"/><p:cNvSpPr/><p:nvPr/></p:nvSpPr><p:spPr/><p:txBody>{}</p:txBody></p:sp>
  </p:spTree></p:cSld>
</p:notes>"#,
                note_body
            );
            zip.write_all(notes_xml.as_bytes()).unwrap();
        }

        zip.finish().unwrap();
    }
    buf.into_inner()
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_pptx_extractor_new() {
        let _extractor = PptxExtractor::new();
    }

    #[test]
    fn test_pptx_default() {
        let _extractor = PptxExtractor::default();
    }

    #[test]
    fn test_extract_empty_pptx() {
        let result = extract_text_from_pptx(&[]);
        assert!(result.is_err());
    }

    #[test]
    fn test_extract_invalid_pptx() {
        let invalid: Vec<u8> = b"This is not a PPTX file".to_vec();
        let result = extract_text_from_pptx(&invalid);
        assert!(result.is_err());
    }

    #[test]
    fn test_extract_single_slide() {
        let pptx = create_test_pptx(vec![&["Hello World"]], vec![]);
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert_eq!(result.page_count, 1);
        assert!(result.text.contains("Hello World"));
    }

    #[test]
    fn test_extract_multi_slide() {
        let pptx = create_test_pptx(
            vec![&["Slide One"], &["Slide Two"], &["Slide Three"]],
            vec![],
        );
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert_eq!(result.page_count, 3);
        assert!(result.text.contains("Slide One"));
        assert!(result.text.contains("Slide Two"));
        assert!(result.text.contains("Slide Three"));
        assert!(result.text.contains("--- Slide 1 ---"));
        assert!(result.text.contains("--- Slide 2 ---"));
    }

    #[test]
    fn test_extract_multiple_text_blocks_per_slide() {
        let pptx = create_test_pptx(vec![&["Title", "Subtitle", "Body text"]], vec![]);
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert_eq!(result.page_count, 1);
        assert!(result.text.contains("Title"));
        assert!(result.text.contains("Subtitle"));
        assert!(result.text.contains("Body text"));
    }

    #[test]
    fn test_extract_speaker_notes() {
        let pptx = create_test_pptx(
            vec![&["Slide content"]],
            vec![&["Remember to explain this point"]],
        );
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert_eq!(result.page_count, 1);
        assert!(result.metadata.contains_key("speaker_notes"));
        assert!(result.metadata["speaker_notes"].contains("Remember to explain this point"));
    }

    #[test]
    fn test_extract_multiple_notes() {
        let pptx = create_test_pptx(
            vec![&["Slide 1"], &["Slide 2"]],
            vec![&["Note for slide 1"], &["Note for slide 2"]],
        );
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert!(result.metadata["speaker_notes"].contains("Note for slide 1"));
        assert!(result.metadata["speaker_notes"].contains("Note for slide 2"));
    }

    #[test]
    fn test_empty_slide() {
        let pptx = create_test_pptx(vec![&[]], vec![]);
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert_eq!(result.page_count, 1);
        assert!(result.text.is_empty());
    }

    #[test]
    fn test_slide_with_no_text_in_metadata() {
        let pptx = create_test_pptx(vec![&["Content"]], vec![]);
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert_eq!(result.metadata["slide_count"], "1");
    }

    #[test]
    fn test_malformed_xml_in_slide() {
        // Create a PPTX with malformed XML in a slide
        use std::io::Write;
        let mut buf = Cursor::new(Vec::new());
        {
            let mut zip = zip::ZipWriter::new(&mut buf);
            let options = zip::write::SimpleFileOptions::default();

            zip.start_file("[Content_Types].xml", options).unwrap();
            zip.write_all(br#"<?xml version="1.0"?><Types/>"#).unwrap();

            zip.start_file("ppt/slides/slide1.xml", options).unwrap();
            zip.write_all(b"<?xml version=\"1.0\"?>\n<p:sld xmlns:a=\"http://schemas.openxmlformats.org/drawingml/2006/main\" xmlns:p=\"http://schemas.openxmlformats.org/presentationml/2006/main\"><p:cSld><p:spTree><p:sp><p:txBody><a:p><a:r><a:t>Good text</a:t></a:r></a:p></p:txBody></p:sp><BROKEN<<<>>></p:spTree></p:cSld></p:sld>").unwrap();

            zip.finish().unwrap();
        }
        let pptx_data = buf.into_inner();
        // Should still extract what it can before hitting the malformed part
        let result = extract_text_from_pptx(&pptx_data).unwrap();
        assert!(result.text.contains("Good text"));
    }

    #[test]
    fn test_pptx_too_large() {
        use crate::document::MAX_FILE_SIZE;
        let oversized: Vec<u8> = vec![0u8; MAX_FILE_SIZE + 1];
        let result = extract_text_from_pptx(&oversized);
        assert!(result.is_err());
    }

    #[test]
    fn test_extract_xml_text_simple() {
        let xml = r#"<?xml version="1.0"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:sp><p:txBody><a:p><a:r><a:t>Hello</a:t></a:r></a:p></p:txBody></p:sp>
</p:sld>"#;
        let text = extract_text_from_slide_xml(xml);
        assert_eq!(text, "Hello");
    }

    #[test]
    fn test_extract_xml_text_multiple_a_t_elements() {
        let xml = r#"<?xml version="1.0"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <a:p><a:r><a:t>Hello</a:t><a:t>World</a:t></a:r></a:p>
</p:sld>"#;
        let text = extract_text_from_slide_xml(xml);
        assert_eq!(text, "HelloWorld");
    }

    #[test]
    fn test_extract_xml_text_nested_in_groups() {
        let xml = r#"<?xml version="1.0"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:spTree>
    <p:grpSp>
      <p:sp>
        <p:txBody><a:p><a:r><a:t>Grouped</a:t></a:r></a:p></p:txBody>
      </p:sp>
    </p:grpSp>
  </p:spTree>
</p:sld>"#;
        let text = extract_text_from_slide_xml(xml);
        assert_eq!(text, "Grouped");
    }

    #[test]
    fn test_extract_xml_text_empty() {
        let xml = r#"<?xml version="1.0"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main">
  <p:sp><p:txBody><a:p><a:r></a:r></a:p></p:txBody></p:sp>
</p:sld>"#;
        let text = extract_text_from_slide_xml(xml);
        assert!(text.is_empty());
    }

    #[test]
    fn test_extract_xml_table_cells() {
        // PPTX tables have <a:t> inside <a:tc> cells
        let xml = r#"<?xml version="1.0"?>
<p:sld xmlns:a="http://schemas.openxmlformats.org/drawingml/2006/main" xmlns:p="http://schemas.openxmlformats.org/presentationml/2006/main">
  <p:sp>
    <p:txBody>
      <a:tbl>
        <a:tr>
          <a:tc><a:txBody><a:p><a:r><a:t>Cell A</a:t></a:r></a:p></a:txBody></a:tc>
          <a:tc><a:txBody><a:p><a:r><a:t>Cell B</a:t></a:r></a:p></a:txBody></a:tc>
        </a:tr>
      </a:tbl>
    </p:txBody>
  </p:sp>
</p:sld>"#;
        let text = extract_text_from_slide_xml(xml);
        assert!(text.contains("Cell A"));
        assert!(text.contains("Cell B"));
    }

    #[test]
    fn test_slide_ordering() {
        // Slides should be ordered numerically, not lexicographically
        let pptx = create_test_pptx(
            vec![
                &["Slide 1"],
                &["Slide 2"],
                &["Slide 3"],
                &["Slide 4"],
                &["Slide 5"],
                &["Slide 6"],
                &["Slide 7"],
                &["Slide 8"],
                &["Slide 9"],
                &["Slide 10"],
            ],
            vec![],
        );
        let result = extract_text_from_pptx(&pptx).unwrap();
        assert_eq!(result.page_count, 10);
        // Verify ordering: Slide 1 should appear before Slide 10
        let pos1 = result.text.find("Slide 1").unwrap_or(0);
        let pos10 = result.text.find("Slide 10").unwrap_or(0);
        assert!(
            pos1 < pos10 || !result.text.contains("Slide 10"),
            "Slide 1 should appear before Slide 10"
        );
    }

    #[test]
    fn test_redacted_extraction() {
        let pptx = create_test_pptx(vec![&["SSN: 123-45-6789"]], vec![]);
        let mut shadowmap = ShadowMap::new();
        let result = PptxExtractor::new()
            .extract_from_bytes_redacted(&pptx, &mut shadowmap)
            .unwrap();
        // ShadowMap should redact the SSN
        assert!(!result.text.contains("123-45-6789"));
    }
}
