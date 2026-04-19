use crate::error::{Result, SidecarError};

pub fn convert_docx_to_pdf(docx_bytes: &[u8]) -> Result<Vec<u8>> {
    use printpdf::*;
    use crate::document::extract_text_from_docx;

    if docx_bytes.is_empty() {
        return Err(SidecarError::InvalidRequest("DOCX content is empty".to_string()));
    }

    let extraction = extract_text_from_docx(docx_bytes)?;
    let text = extraction.text;

    let line_height = 14.0_f32;
    let font_size = 12.0_f32;
    let margin_left = Mm(20.0);
    let margin_bottom = 20.0_f32;
    let page_height = Mm(297.0);
    let page_width = Mm(210.0);
    let top_margin = 20.0_f32;
    let usable_bottom = margin_bottom + line_height;

    let make_header_ops = |y: f32| -> Vec<Op> {
        vec![
            Op::StartTextSection,
            Op::SetTextCursor { pos: Point::new(margin_left, Mm(y)) },
            Op::SetFontSizeBuiltinFont { size: Pt(font_size), font: BuiltinFont::Helvetica },
            Op::SetLineHeight { lh: Pt(line_height) },
        ]
    };

    let mut pages: Vec<PdfPage> = Vec::new();
    let mut current_ops = make_header_ops(page_height.0 - top_margin);
    let mut y_cursor = page_height.0 - top_margin;

    for line in text.lines() {
        if y_cursor - line_height < usable_bottom {
            current_ops.push(Op::EndTextSection);
            pages.push(PdfPage::new(page_width, page_height, std::mem::take(&mut current_ops)));
            y_cursor = page_height.0 - top_margin;
            current_ops = make_header_ops(y_cursor);
        }

        let sanitized: String = line.chars().filter(|c| c.is_ascii_graphic() || *c == ' ').collect();
        if sanitized.is_empty() {
            y_cursor -= line_height;
            continue;
        }

        current_ops.push(Op::SetTextCursor { pos: Point::new(margin_left, Mm(y_cursor)) });
        current_ops.push(Op::WriteTextBuiltinFont {
            items: vec![TextItem::Text(sanitized)],
            font: BuiltinFont::Helvetica,
        });
        y_cursor -= line_height;
    }

    if !current_ops.is_empty() {
        if matches!(current_ops.first(), Some(Op::StartTextSection)) {
            current_ops.push(Op::EndTextSection);
        }
        pages.push(PdfPage::new(page_width, page_height, current_ops));
    }

    if pages.is_empty() {
        let empty_ops = vec![
            Op::StartTextSection,
            Op::SetTextCursor { pos: Point::new(margin_left, Mm(page_height.0 - 40.0)) },
            Op::SetFontSizeBuiltinFont { size: Pt(font_size), font: BuiltinFont::Helvetica },
            Op::WriteTextBuiltinFont {
                items: vec![TextItem::Text("(empty document)".to_string())],
                font: BuiltinFont::Helvetica,
            },
            Op::EndTextSection,
        ];
        pages.push(PdfPage::new(page_width, page_height, empty_ops));
    }

    let mut doc = PdfDocument::new("Converted Document");
    let pdf_bytes = doc
        .with_pages(pages)
        .save(&PdfSaveOptions::default(), &mut Vec::new());

    Ok(pdf_bytes)
}

pub fn convert_xlsx_to_csv(xlsx_bytes: &[u8]) -> Result<Vec<u8>> {
    use crate::document::extract_data_from_xlsx;

    if xlsx_bytes.is_empty() {
        return Err(SidecarError::InvalidRequest("XLSX content is empty".to_string()));
    }

    let extraction = extract_data_from_xlsx(xlsx_bytes)?;
    let mut csv_lines: Vec<String> = Vec::new();

    for sheet in &extraction.sheets {
        for row in &sheet.rows {
            let cells: Vec<String> = row
                .iter()
                .map(|cell| {
                    match cell {
                        Some(value) => {
                            if value.contains(',') || value.contains('"') || value.contains('\n') {
                                format!("\"{}\"", value.replace('"', "\"\""))
                            } else {
                                value.clone()
                            }
                        }
                        None => String::new(),
                    }
                })
                .collect();
            csv_lines.push(cells.join(","));
        }
        csv_lines.push(String::new());
    }

    Ok(csv_lines.join("\n").into_bytes())
}

pub fn convert_pptx_to_pdf(_pptx_bytes: &[u8]) -> Result<Vec<u8>> {
    Err(SidecarError::DocumentProcessingError(
        "PPTX→PDF conversion is not yet supported".to_string(),
    ))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_convert_docx_to_pdf_empty_input() {
        let result = convert_docx_to_pdf(&[]);
        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => assert!(msg.contains("empty")),
            _ => panic!("Expected InvalidRequest"),
        }
    }

    #[test]
    fn test_convert_docx_to_pdf_invalid_input() {
        let result = convert_docx_to_pdf(b"not a docx");
        assert!(result.is_err());
    }

    #[test]
    fn test_convert_xlsx_to_csv_empty_input() {
        let result = convert_xlsx_to_csv(&[]);
        assert!(result.is_err());
        match result {
            Err(SidecarError::InvalidRequest(msg)) => assert!(msg.contains("empty")),
            _ => panic!("Expected InvalidRequest"),
        }
    }

    #[test]
    fn test_convert_pptx_to_pdf_returns_unsupported() {
        let result = convert_pptx_to_pdf(&[1, 2, 3]);
        assert!(result.is_err());
        match result {
            Err(SidecarError::DocumentProcessingError(msg)) => {
                assert!(msg.contains("not yet supported"));
            }
            _ => panic!("Expected DocumentProcessingError"),
        }
    }
}
