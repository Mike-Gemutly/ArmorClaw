use armorclaw_sidecar::document::{
    delete_paragraph_in_docx, extract_text_from_docx, extract_text_from_pdf, insert_paragraph_in_docx,
    merge_pdfs, replace_text_in_docx, split_pdf, validate_file_size, MAX_FILE_SIZE,
};
use armorclaw_sidecar::document::pdf::Document;
use armorclaw_sidecar::document::docx::Docx;
use docx_rs;
use lopdf::dictionary;
use std::collections::HashMap;

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

fn create_multi_page_pdf_bytes(page_count: usize) -> Vec<u8> {
    let mut doc = Document::with_version("1.5");

    let pages_id = doc.new_object_id();
    let font_id = doc.new_object_id();
    let mut page_ids = Vec::new();
    let mut content_ids = Vec::new();

    doc.add_object(font_id, dictionary! {
        "Type" => "Font",
        "Subtype" => "Type1",
        "BaseFont" => "Helvetica",
    });

    for i in 0..page_count {
        let page_id = doc.new_object_id();
        let content_id = doc.new_object_id();

        doc.add_object(page_id, dictionary! {
            "Type" => "Page",
            "Parent" => pages_id,
            "Resources" => dictionary! {
                "Font" => dictionary! {
                    "F1" => font_id,
                },
            },
            "Contents" => content_id,
        });

        doc.add_object(content_id, format!("BT /F1 12 Tf 100 700 Td (Page {} content) Tj ET", i + 1).as_bytes());

        page_ids.push(page_id);
        content_ids.push(content_id);
    }

    let mut kids = Vec::new();
    for page_id in page_ids {
        kids.push(page_id.into());
    }

    let pages_dict = dictionary! {
        "Type" => "Pages",
        "Kids" => kids,
        "Count" => page_count as i32,
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
        "Subject" => "Test Subject",
        "Keywords" => "test, pdf, integration",
        "Creator" => "Test Creator",
        "Producer" => "Test Producer",
    });

    doc.set_trailer(dictionary! {
        "Root" => catalog_id,
        "Info" => info_id,
    });

    doc.save_to_bytes().unwrap()
}

fn create_empty_pdf_bytes() -> Vec<u8> {
    let mut doc = Document::with_version("1.5");

    let pages_id = doc.new_object_id();
    let pages_dict = dictionary! {
        "Type" => "Pages",
        "Kids" => vec![],
        "Count" => 0,
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

fn create_simple_docx_bytes() -> Vec<u8> {
    let mut docx = Docx::default();

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("Hello, World!"))
    );

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("This is a test document."))
    );

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&docx, &mut cursor).unwrap();
    cursor.into_inner()
}

fn create_docx_with_metadata_bytes() -> Vec<u8> {
    let mut docx = Docx::default();

    docx.core_properties = Some(docx_rs::CoreProperties::new()
        .title("Test DOCX Title")
        .creator("Test Author")
        .subject("Test Subject")
        .description("Test Description")
        .keywords("test, docx, integration"));

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("First paragraph"))
    );

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("Second paragraph with important content"))
    );

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("Third paragraph"))
    );

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&docx, &mut cursor).unwrap();
    cursor.into_inner()
}

fn create_editable_docx_bytes() -> Vec<u8> {
    let mut docx = Docx::default();

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("First paragraph with replace_me text"))
    );

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("Second paragraph"))
    );

    docx.document.paragraphs.push(
        docx_rs::Paragraph::new().add_run(docx_rs::Run::new().add_text("Third paragraph with replace_me"))
    );

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&docx, &mut cursor).unwrap();
    cursor.into_inner()
}

fn load_docx_from_bytes(docx_bytes: &[u8]) -> Docx {
    let cursor = std::io::Cursor::new(docx_bytes);
    docx_rs::read_docx(cursor).unwrap()
}

#[tokio::test]
async fn test_pdf_extract_single_page_with_text() {
    let pdf_bytes = create_simple_pdf_bytes();
    let result = extract_text_from_pdf(&pdf_bytes).unwrap();

    assert_eq!(result.page_count, 1);
    assert!(!result.text.is_empty());
    assert!(result.text.contains("Hello"));
    assert!(result.text.contains("World"));
}

#[tokio::test]
async fn test_pdf_extract_multi_page_with_text() {
    let pdf_bytes = create_multi_page_pdf_bytes(3);
    let result = extract_text_from_pdf(&pdf_bytes).unwrap();

    assert_eq!(result.page_count, 3);
    assert!(!result.text.is_empty());
    assert!(result.text.contains("Page 1 content"));
    assert!(result.text.contains("Page 2 content"));
    assert!(result.text.contains("Page 3 content"));
}

#[tokio::test]
async fn test_pdf_extract_with_metadata() {
    let pdf_bytes = create_multi_page_pdf_bytes(2);
    let result = extract_text_from_pdf(&pdf_bytes).unwrap();

    assert!(result.metadata.contains_key("title"));
    assert_eq!(result.metadata.get("title").unwrap(), "Test PDF Title");

    assert!(result.metadata.contains_key("author"));
    assert_eq!(result.metadata.get("author").unwrap(), "Test Author");

    assert!(result.metadata.contains_key("subject"));
    assert_eq!(result.metadata.get("subject").unwrap(), "Test Subject");

    assert!(result.metadata.contains_key("keywords"));
    assert_eq!(result.metadata.get("keywords").unwrap(), "test, pdf, integration");
}

#[tokio::test]
async fn test_pdf_extract_empty_pdf() {
    let pdf_bytes = create_empty_pdf_bytes();
    let result = extract_text_from_pdf(&pdf_bytes);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("no pages") || error_msg.contains("0"));
}

#[tokio::test]
async fn test_pdf_extract_corrupted_pdf() {
    let corrupted_pdf: Vec<u8> = b"This is not a valid PDF file".to_vec();
    let result = extract_text_from_pdf(&corrupted_pdf);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("Failed to load PDF"));
}

#[tokio::test]
async fn test_pdf_split_single_page() {
    let pdf_bytes = create_multi_page_pdf_bytes(3);
    let result = split_pdf(&pdf_bytes, "1").unwrap();

    assert!(!result.is_empty());

    let split_doc = Document::load_mem(&result).unwrap();
    assert_eq!(split_doc.get_pages().len(), 1);
}

#[tokio::test]
async fn test_pdf_split_page_range() {
    let pdf_bytes = create_multi_page_pdf_bytes(5);
    let result = split_pdf(&pdf_bytes, "1-3").unwrap();

    assert!(!result.is_empty());

    let split_doc = Document::load_mem(&result).unwrap();
    assert_eq!(split_doc.get_pages().len(), 3);
}

#[tokio::test]
async fn test_pdf_split_multiple_ranges() {
    let pdf_bytes = create_multi_page_pdf_bytes(5);
    let result = split_pdf(&pdf_bytes, "1,3,5").unwrap();

    assert!(!result.is_empty());

    let split_doc = Document::load_mem(&result).unwrap();
    assert_eq!(split_doc.get_pages().len(), 3);
}

#[tokio::test]
async fn test_pdf_split_complex_ranges() {
    let pdf_bytes = create_multi_page_pdf_bytes(10);
    let result = split_pdf(&pdf_bytes, "1-3,5-7,9").unwrap();

    assert!(!result.is_empty());

    let split_doc = Document::load_mem(&result).unwrap();
    assert_eq!(split_doc.get_pages().len(), 6);
}

#[tokio::test]
async fn test_pdf_merge_two_pdfs() {
    let pdf1 = create_simple_pdf_bytes();
    let pdf2 = create_multi_page_pdf_bytes(2);

    let pdf_list = vec![pdf1.as_slice(), pdf2.as_slice()];
    let result = merge_pdfs(&pdf_list).unwrap();

    assert!(!result.is_empty());

    let merged_doc = Document::load_mem(&result).unwrap();
    assert_eq!(merged_doc.get_pages().len(), 3);
}

#[tokio::test]
async fn test_pdf_merge_multiple_pdfs() {
    let pdf1 = create_simple_pdf_bytes();
    let pdf2 = create_multi_page_pdf_bytes(2);
    let pdf3 = create_multi_page_pdf_bytes(3);

    let pdf_list = vec![pdf1.as_slice(), pdf2.as_slice(), pdf3.as_slice()];
    let result = merge_pdfs(&pdf_list).unwrap();

    assert!(!result.is_empty());

    let merged_doc = Document::load_mem(&result).unwrap();
    assert_eq!(merged_doc.get_pages().len(), 6);
}

#[tokio::test]
async fn test_pdf_split_invalid_page_range() {
    let pdf_bytes = create_multi_page_pdf_bytes(3);
    let result = split_pdf(&pdf_bytes, "1-2-3");

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("Invalid page range format"));
}

#[tokio::test]
async fn test_pdf_split_page_out_of_bounds() {
    let pdf_bytes = create_multi_page_pdf_bytes(3);
    let result = split_pdf(&pdf_bytes, "99");

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("exceeds total pages"));
}

#[tokio::test]
async fn test_pdf_split_zero_indexed() {
    let pdf_bytes = create_multi_page_pdf_bytes(3);
    let result = split_pdf(&pdf_bytes, "0");

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("1-indexed"));
}

#[tokio::test]
async fn test_pdf_extract_file_size_at_limit() {
    let small_pdf = create_simple_pdf_bytes();
    let padding_size = MAX_FILE_SIZE - small_pdf.len();
    let mut padded_pdf = small_pdf;
    padded_pdf.resize(MAX_FILE_SIZE, 0u8);

    let result = extract_text_from_pdf(&padded_pdf);

    assert!(result.is_err());
}

#[tokio::test]
async fn test_pdf_extract_file_size_exceeds_limit() {
    let oversized_pdf: Vec<u8> = vec![0u8; MAX_FILE_SIZE + 1];
    let result = extract_text_from_pdf(&oversized_pdf);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("exceeds maximum allowed size"));
    assert!(error_msg.contains("5GB"));
}

#[tokio::test]
async fn test_pdf_split_file_size_at_limit() {
    let small_pdf = create_simple_pdf_bytes();
    let padding_size = MAX_FILE_SIZE - small_pdf.len();
    let mut padded_pdf = small_pdf;
    padded_pdf.resize(MAX_FILE_SIZE, 0u8);

    let result = split_pdf(&padded_pdf, "1");

    assert!(result.is_err());
}

#[tokio::test]
async fn test_pdf_split_file_size_exceeds_limit() {
    let oversized_pdf: Vec<u8> = vec![0u8; MAX_FILE_SIZE + 1];
    let result = split_pdf(&oversized_pdf, "1");

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("exceeds maximum allowed size"));
}

#[tokio::test]
async fn test_pdf_merge_file_size_exceeds_limit() {
    let pdf1 = create_simple_pdf_bytes();
    let oversized_pdf: Vec<u8> = vec![0u8; MAX_FILE_SIZE + 1];

    let pdf_list = vec![pdf1.as_slice(), oversized_pdf.as_slice()];
    let result = merge_pdfs(&pdf_list);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("exceeds maximum allowed size"));
}

#[tokio::test]
async fn test_validate_file_size_valid() {
    assert!(validate_file_size(1024).is_ok());
    assert!(validate_file_size(1024 * 1024).is_ok());
    assert!(validate_file_size(MAX_FILE_SIZE).is_ok());
}

#[tokio::test]
async fn test_validate_file_size_exceeds() {
    assert!(validate_file_size(MAX_FILE_SIZE + 1).is_err());
    assert!(validate_file_size(MAX_FILE_SIZE + 1024).is_err());
}

#[tokio::test]
async fn test_docx_extract_simple_with_text() {
    let docx_bytes = create_simple_docx_bytes();
    let result = extract_text_from_docx(&docx_bytes).unwrap();

    assert!(!result.text.is_empty());
    assert!(result.text.contains("Hello"));
    assert!(result.text.contains("World"));
    assert!(result.text.contains("test document"));
}

#[tokio::test]
async fn test_docx_extract_with_metadata() {
    let docx_bytes = create_docx_with_metadata_bytes();
    let result = extract_text_from_docx(&docx_bytes).unwrap();

    assert!(result.metadata.contains_key("title"));
    assert_eq!(result.metadata.get("title").unwrap(), "Test DOCX Title");

    assert!(result.metadata.contains_key("author"));
    assert_eq!(result.metadata.get("author").unwrap(), "Test Author");

    assert!(result.metadata.contains_key("subject"));
    assert_eq!(result.metadata.get("subject").unwrap(), "Test Subject");

    assert!(result.metadata.contains_key("keywords"));
    assert_eq!(result.metadata.get("keywords").unwrap(), "test, docx, integration");
}

#[tokio::test]
async fn test_docx_extract_empty_docx() {
    let empty_docx: Vec<u8> = vec![];
    let result = extract_text_from_docx(&empty_docx);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("empty"));
}

#[tokio::test]
async fn test_docx_extract_invalid_docx() {
    let invalid_docx: Vec<u8> = b"This is not a valid DOCX file".to_vec();
    let result = extract_text_from_docx(&invalid_docx);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("Failed to load DOCX"));
}

#[tokio::test]
async fn test_docx_replace_text_in_paragraph() {
    let docx_bytes = create_editable_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let mut params = HashMap::new();
    params.insert("find".to_string(), "replace_me".to_string());
    params.insert("replace".to_string(), "REPLACED".to_string());

    let result = replace_text_in_docx(&docx, &params).unwrap();

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&result, &mut cursor).unwrap();
    let modified_bytes = cursor.into_inner();

    let extract_result = extract_text_from_docx(&modified_bytes).unwrap();
    assert!(!extract_result.text.contains("replace_me"));
    assert!(extract_result.text.contains("REPLACED"));
}

#[tokio::test]
async fn test_docx_replace_text_not_found() {
    let docx_bytes = create_editable_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let mut params = HashMap::new();
    params.insert("find".to_string(), "nonexistent".to_string());
    params.insert("replace".to_string(), "REPLACED".to_string());

    let result = replace_text_in_docx(&docx, &params).unwrap();

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&result, &mut cursor).unwrap();
    let modified_bytes = cursor.into_inner();

    let original_text = extract_text_from_docx(&docx_bytes).unwrap().text;
    let modified_text = extract_text_from_docx(&modified_bytes).unwrap().text;
    assert_eq!(original_text, modified_text);
}

#[tokio::test]
async fn test_docx_replace_text_empty_find() {
    let docx_bytes = create_editable_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let mut params = HashMap::new();
    params.insert("find".to_string(), "".to_string());
    params.insert("replace".to_string(), "REPLACED".to_string());

    let result = replace_text_in_docx(&docx, &params);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("find"));
}

#[tokio::test]
async fn test_docx_insert_paragraph_at_position() {
    let docx_bytes = create_simple_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let original_paragraph_count = docx.document.paragraphs.len();

    let mut params = HashMap::new();
    params.insert("text".to_string(), "Inserted paragraph".to_string());
    params.insert("position".to_string(), "1".to_string());

    let result = insert_paragraph_in_docx(&docx, &params).unwrap();

    assert_eq!(result.document.paragraphs.len(), original_paragraph_count + 1);
}

#[tokio::test]
async fn test_docx_insert_paragraph_at_beginning() {
    let docx_bytes = create_simple_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let mut params = HashMap::new();
    params.insert("text".to_string(), "First paragraph".to_string());
    params.insert("position".to_string(), "0".to_string());

    let result = insert_paragraph_in_docx(&docx, &params).unwrap();

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&result, &mut cursor).unwrap();
    let modified_bytes = cursor.into_inner();

    let extract_result = extract_text_from_docx(&modified_bytes).unwrap();
    let lines: Vec<&str> = extract_result.text.lines().collect();
    assert_eq!(lines[0], "First paragraph");
}

#[tokio::test]
async fn test_docx_insert_paragraph_at_end() {
    let docx_bytes = create_simple_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let original_count = docx.document.paragraphs.len();

    let mut params = HashMap::new();
    params.insert("text".to_string(), "Last paragraph".to_string());
    params.insert("position".to_string(), &original_count.to_string());

    let result = insert_paragraph_in_docx(&docx, &params).unwrap();

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&result, &mut cursor).unwrap();
    let modified_bytes = cursor.into_inner();

    let extract_result = extract_text_from_docx(&modified_bytes).unwrap();
    let lines: Vec<&str> = extract_result.text.lines().collect();
    assert_eq!(lines.last().unwrap(), "Last paragraph");
}

#[tokio::test]
async fn test_docx_insert_paragraph_invalid_position() {
    let docx_bytes = create_simple_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let mut params = HashMap::new();
    params.insert("text".to_string(), "New paragraph".to_string());
    params.insert("position".to_string(), "999".to_string());

    let result = insert_paragraph_in_docx(&docx, &params);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("out of bounds"));
}

#[tokio::test]
async fn test_docx_delete_paragraph_by_index() {
    let docx_bytes = create_editable_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let original_count = docx.document.paragraphs.len();

    let mut params = HashMap::new();
    params.insert("index".to_string(), "1".to_string());

    let result = delete_paragraph_in_docx(&docx, &params).unwrap();

    assert_eq!(result.document.paragraphs.len(), original_count - 1);
}

#[tokio::test]
async fn test_docx_delete_first_paragraph() {
    let docx_bytes = create_editable_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let mut params = HashMap::new();
    params.insert("index".to_string(), "0".to_string());

    let result = delete_paragraph_in_docx(&docx, &params).unwrap();

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&result, &mut cursor).unwrap();
    let modified_bytes = cursor.into_inner();

    let extract_result = extract_text_from_docx(&modified_bytes).unwrap();
    assert!(!extract_result.text.contains("First paragraph"));
}

#[tokio::test]
async fn test_docx_delete_paragraph_invalid_index() {
    let docx_bytes = create_simple_docx_bytes();
    let docx = load_docx_from_bytes(&docx_bytes);

    let mut params = HashMap::new();
    params.insert("index".to_string(), "999".to_string());

    let result = delete_paragraph_in_docx(&docx, &params);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("index") || error_msg.contains("out of bounds"));
}

#[tokio::test]
async fn test_docx_extract_file_size_exceeds_limit() {
    let oversized_docx: Vec<u8> = vec![0u8; MAX_FILE_SIZE + 1];
    let result = extract_text_from_docx(&oversized_docx);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("exceeds maximum allowed size"));
    assert!(error_msg.contains("5GB"));
}

#[tokio::test]
async fn test_pdf_workflow_extract_validate_split() {
    let pdf_bytes = create_multi_page_pdf_bytes(5);

    let extraction_result = extract_text_from_pdf(&pdf_bytes).unwrap();
    assert_eq!(extraction_result.page_count, 5);
    assert!(!extraction_result.text.is_empty());

    let split_result = split_pdf(&pdf_bytes, "1-3").unwrap();
    assert!(!split_result.is_empty());

    let split_doc = Document::load_mem(&split_result).unwrap();
    assert_eq!(split_doc.get_pages().len(), 3);

    let split_extraction = extract_text_from_pdf(&split_result).unwrap();
    assert_eq!(split_extraction.page_count, 3);
}

#[tokio::test]
async fn test_pdf_workflow_split_merge_verify() {
    let pdf1 = create_multi_page_pdf_bytes(2);
    let pdf2 = create_multi_page_pdf_bytes(3);

    let split1 = split_pdf(&pdf1, "1").unwrap();
    let split2 = split_pdf(&pdf2, "1-2").unwrap();

    let pdf_list = vec![split1.as_slice(), split2.as_slice()];
    let merged = merge_pdfs(&pdf_list).unwrap();

    let merged_doc = Document::load_mem(&merged).unwrap();
    assert_eq!(merged_doc.get_pages().len(), 3);
}

#[tokio::test]
async fn test_docx_workflow_extract_edit_verify() {
    let docx_bytes = create_editable_docx_bytes();

    let extraction_result = extract_text_from_docx(&docx_bytes).unwrap();
    assert!(!extraction_result.text.is_empty());

    let docx = load_docx_from_bytes(&docx_bytes);

    let mut replace_params = HashMap::new();
    replace_params.insert("find".to_string(), "replace_me".to_string());
    replace_params.insert("replace".to_string(), "REPLACED".to_string());

    let edited_docx = replace_text_in_docx(&docx, &replace_params).unwrap();

    let mut insert_params = HashMap::new();
    insert_params.insert("text".to_string(), "New paragraph".to_string());
    insert_params.insert("position".to_string(), "0".to_string());

    let final_docx = insert_paragraph_in_docx(&edited_docx, &insert_params).unwrap();

    let mut cursor = std::io::Cursor::new(Vec::new());
    docx_rs::write_docx(&final_docx, &mut cursor).unwrap();
    let final_bytes = cursor.into_inner();

    let final_extraction = extract_text_from_docx(&final_bytes).unwrap();
    assert!(!final_extraction.text.contains("replace_me"));
    assert!(final_extraction.text.contains("REPLACED"));
    assert!(final_extraction.text.contains("New paragraph"));
}

#[tokio::test]
async fn test_pdf_split_empty_input() {
    let empty_pdf: Vec<u8> = vec![];
    let result = split_pdf(&empty_pdf, "1");

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("empty"));
}

#[tokio::test]
async fn test_pdf_split_empty_page_range() {
    let pdf_bytes = create_multi_page_pdf_bytes(3);
    let result = split_pdf(&pdf_bytes, "");

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("No valid pages selected"));
}

#[tokio::test]
async fn test_pdf_merge_empty_list() {
    let pdf_list: Vec<&[u8]> = vec![];
    let result = merge_pdfs(&pdf_list);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("No PDFs provided"));
}

#[tokio::test]
async fn test_pdf_merge_contains_empty_pdf() {
    let pdf1 = create_simple_pdf_bytes();
    let empty_pdf: Vec<u8> = vec![];

    let pdf_list = vec![pdf1.as_slice(), empty_pdf.as_slice()];
    let result = merge_pdfs(&pdf_list);

    assert!(result.is_err());
    let error_msg = result.unwrap_err().to_string();
    assert!(error_msg.contains("empty"));
}
