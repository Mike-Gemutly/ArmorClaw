// Simple E2E test demonstrating library functionality
// This test uses the compiled library directly

use armorclaw_sidecar::error::{SidecarError, Result};

#[test]
fn test_library_compiles() {
    // Verify the library compiles and exports its public API
    println!("✅ Library compiles successfully");
}

#[test]
fn test_error_types() {
    // Test that error types work
    let err = SidecarError::InvalidRequest("test".to_string());
    assert!(err.to_string().contains("test"));
    println!("✅ Error types functional");
}

#[test]
fn test_security_module() {
    // Test that security module is accessible
    use armorclaw_sidecar::security::TOKEN_TTL_SECONDS;
    assert_eq!(TOKEN_TTL_SECONDS, 1800);
    println!("✅ Security module functional (TTL: {}s)", TOKEN_TTL_SECONDS);
}

// Future E2E tests (require credentials):
// - test_s3_connector_upload_download
// - test_sharepoint_connector
// - test_pdf_extraction
// - test_docx_extraction
// - test_security_token_validation
// - test_circuit_breaker
// - test_rate_limiting

#[test]
#[ignore = "Requires AWS credentials"]
fn test_s3_connector_integration() {
    // This would test actual S3 operations
    // Requires: AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY
    unimplemented!("Requires AWS credentials")
}

#[test]
#[ignore = "Requires SharePoint credentials"]
fn test_sharepoint_connector_integration() {
    // This would test actual SharePoint operations
    // Requires: SHAREPOINT_TENANT_ID, SHAREPOINT_CLIENT_ID, SHAREPOINT_CLIENT_SECRET
    unimplemented!("Requires SharePoint credentials")
}

#[test]
fn test_document_processing_stubs() {
    // Test that document processing stubs return helpful errors
    use armorclaw_sidecar::document::xlsx::extract_data_from_xlsx;
    
    let result = extract_data_from_xlsx(&[]);
    assert!(result.is_err());
    if let Err(SidecarError::InvalidRequest(msg)) = result {
        assert!(msg.contains("not yet implemented"));
        println!("✅ XLSX stub returns helpful error: {}", msg);
    }
}
