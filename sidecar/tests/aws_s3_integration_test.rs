use armorclaw_sidecar::connectors::{S3Connector, S3UploadRequest};
use std::io::Write;
use tempfile::NamedTempFile;

#[tokio::test]
#[ignore]
async fn integration_test_small_upload_from_memory() {
    let connector = S3Connector;

    let content = vec![1u8; 1024 * 1024];

    let request = S3UploadRequest {
        bucket: std::env::var("S3_TEST_BUCKET").unwrap_or_else(|_| "test-bucket".to_string()),
        key: format!("test/small-upload-{}.bin", chrono::Utc::now().timestamp()),
        region: std::env::var("S3_TEST_REGION").unwrap_or_else(|_| "us-east-1".to_string()),
        content_type: Some("application/octet-stream".to_string()),
        content: Some(content),
        file_path: None,
        access_key: std::env::var("AWS_ACCESS_KEY_ID").ok(),
        secret_key: std::env::var("AWS_SECRET_ACCESS_KEY").ok(),
        session_token: std::env::var("AWS_SESSION_TOKEN").ok(),
    };

    let result = connector.upload(request).await.unwrap();

    assert!(!result.blob_id.is_empty());
    assert!(!result.etag.is_empty());
    assert_eq!(result.size_bytes, 1024 * 1024);
    assert!(result.content_hash_sha256.len() == 64);
    assert!(result.timestamp_unix > 0);

    println!("Small upload from memory: {}", result.blob_id);
}

#[tokio::test]
#[ignore]
async fn integration_test_large_upload_streaming() {
    let connector = S3Connector;

    let mut temp_file = NamedTempFile::new().unwrap();

    let large_size = 1024 * 1024 * 1024;

    for i in 0..large_size {
        temp_file.write_all(&[i as u8]).unwrap();
    }

    let request = S3UploadRequest {
        bucket: std::env::var("S3_TEST_BUCKET").unwrap_or_else(|_| "test-bucket".to_string()),
        key: format!("test/large-upload-{}.bin", chrono::Utc::now().timestamp()),
        region: std::env::var("S3_TEST_REGION").unwrap_or_else(|_| "us-east-1".to_string()),
        content_type: Some("application/octet-stream".to_string()),
        content: None,
        file_path: Some(temp_file.path().to_path_buf()),
        access_key: std::env::var("AWS_ACCESS_KEY_ID").ok(),
        secret_key: std::env::var("AWS_SECRET_ACCESS_KEY").ok(),
        session_token: std::env::var("AWS_SESSION_TOKEN").ok(),
    };

    let result = connector.upload(request).await.unwrap();

    assert!(!result.blob_id.is_empty());
    assert!(!result.etag.is_empty());
    assert_eq!(result.size_bytes, large_size as i64);
    assert!(result.content_hash_sha256.len() == 64);
    assert!(result.timestamp_unix > 0);

    println!("Large upload streaming: {}", result.blob_id);
}

#[tokio::test]
async fn integration_test_validation_both_content_and_file() {
    let connector = S3Connector;

    let mut temp_file = NamedTempFile::new().unwrap();
    writeln!(temp_file, "test content").unwrap();

    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: "test-key".to_string(),
        region: "us-east-1".to_string(),
        content_type: None,
        content: Some(vec![1, 2, 3]),
        file_path: Some(temp_file.path().to_path_buf()),
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    let result = connector.upload(request).await;

    assert!(result.is_err());
    assert!(result
        .unwrap_err()
        .to_string()
        .contains("Only one of content or file_path can be provided"));
}

#[tokio::test]
async fn integration_test_validation_neither_content_nor_file() {
    let connector = S3Connector;

    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: "test-key".to_string(),
        region: "us-east-1".to_string(),
        content_type: None,
        content: None,
        file_path: None,
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    let result = connector.upload(request).await;

    assert!(result.is_err());
    assert!(result
        .unwrap_err()
        .to_string()
        .contains("Either content or file_path must be provided"));
}

#[tokio::test]
#[ignore]
async fn integration_test_upload_with_content_type() {
    let connector = S3Connector;

    let content = b"hello world".to_vec();

    let request = S3UploadRequest {
        bucket: std::env::var("S3_TEST_BUCKET").unwrap_or_else(|_| "test-bucket".to_string()),
        key: format!("test/text-upload-{}.txt", chrono::Utc::now().timestamp()),
        region: std::env::var("S3_TEST_REGION").unwrap_or_else(|_| "us-east-1".to_string()),
        content_type: Some("text/plain".to_string()),
        content: Some(content),
        file_path: None,
        access_key: std::env::var("AWS_ACCESS_KEY_ID").ok(),
        secret_key: std::env::var("AWS_SECRET_ACCESS_KEY").ok(),
        session_token: std::env::var("AWS_SESSION_TOKEN").ok(),
    };

    let result = connector.upload(request).await.unwrap();

    assert!(!result.blob_id.is_empty());
    assert_eq!(result.size_bytes, 11);

    println!("Upload with content type: {}", result.blob_id);
}
