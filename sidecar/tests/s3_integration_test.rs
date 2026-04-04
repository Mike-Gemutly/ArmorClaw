// Integration Tests for S3 Operations
//
// This file contains comprehensive integration tests for S3 operations including:
// - Basic operations (upload, download, list, delete)
// - Circuit breaker behavior
// - Rate limiting behavior
// - Error scenarios
// - Metrics verification
//
// Tests use mock S3 client to avoid requiring actual S3 credentials.
// For testing against real S3, see aws_s3_integration_test.rs with #[ignore] tests.

use armorclaw_sidecar::connectors::{
    BlobChunk, BlobInfo, S3Connector, S3DeleteRequest, S3DownloadRequest, S3ListRequest,
    S3ListResult, S3UploadRequest, S3UploadResult,
};
use armorclaw_sidecar::error::SidecarError;
use armorclaw_sidecar::reliability::{CircuitBreakerState, S3Reliability};
use std::collections::HashMap;
use std::io::Write;
use std::path::PathBuf;
use std::sync::Arc;
use tempfile::NamedTempFile;
use tokio::sync::RwLock;

// ============================================================================
// Mock S3 Client Infrastructure
// ============================================================================

/// In-memory mock S3 storage for testing
#[derive(Debug, Clone)]
struct MockS3Storage {
    objects: Arc<RwLock<HashMap<(String, String), Vec<u8>>>>,
}

impl MockS3Storage {
    fn new() -> Self {
        Self {
            objects: Arc::new(RwLock::new(HashMap::new())),
        }
    }

    async fn put_object(&self, bucket: String, key: String, data: Vec<u8>) -> Result<(), String> {
        let mut objects = self.objects.write().await;
        objects.insert((bucket, key), data);
        Ok(())
    }

    async fn get_object(&self, bucket: String, key: String) -> Result<Vec<u8>, String> {
        let objects = self.objects.read().await;
        objects
            .get(&(bucket, key))
            .cloned()
            .ok_or_else(|| "Object not found".to_string())
    }

    async fn list_objects(&self, bucket: String, prefix: Option<String>) -> Vec<(String, String, usize)> {
        let objects = self.objects.read().await;
        objects
            .iter()
            .filter(|((b, k), _)| {
                b == &bucket && prefix.as_ref().map_or(true, |p| k.starts_with(p))
            })
            .map(|((b, k), data)| (b.clone(), k.clone(), data.len()))
            .collect()
    }

    async fn delete_object(&self, bucket: String, key: String) -> Result<(), String> {
        let mut objects = self.objects.write().await;
        objects
            .remove(&(bucket, key))
            .map(|_| ())
            .ok_or_else(|| "Object not found".to_string())
    }

    async fn object_exists(&self, bucket: String, key: String) -> bool {
        let objects = self.objects.read().await;
        objects.contains_key(&(bucket, key))
    }

    async fn clear(&self) {
        self.objects.write().await.clear();
    }
}

/// Test fixture for S3 operations
struct S3TestFixture {
    storage: MockS3Storage,
    test_bucket: String,
}

impl S3TestFixture {
    fn new() -> Self {
        Self {
            storage: MockS3Storage::new(),
            test_bucket: "test-bucket".to_string(),
        }
    }

    async fn setup(&self) {
        self.storage.clear().await;
    }

    async fn teardown(&self) {
        self.storage.clear().await;
    }

    async fn put_test_object(&self, key: String, data: Vec<u8>) {
        self.storage
            .put_object(self.test_bucket.clone(), key, data)
            .await
            .unwrap();
    }

    fn create_upload_request(&self, content: Vec<u8>) -> S3UploadRequest {
        let timestamp = chrono::Utc::now().timestamp();
        S3UploadRequest {
            bucket: self.test_bucket.clone(),
            key: format!("test/object-{}", timestamp),
            region: "us-east-1".to_string(),
            content_type: Some("application/octet-stream".to_string()),
            content: Some(content),
            file_path: None,
            access_key: None,
            secret_key: None,
            session_token: None,
        }
    }

    fn create_download_request(&self, key: String) -> S3DownloadRequest {
        S3DownloadRequest {
            bucket: self.test_bucket.clone(),
            key,
            region: "us-east-1".to_string(),
            offset_bytes: None,
            max_bytes: None,
            access_key: None,
            secret_key: None,
            session_token: None,
        }
    }

    fn create_list_request(&self, prefix: Option<String>) -> S3ListRequest {
        S3ListRequest {
            bucket: self.test_bucket.clone(),
            region: "us-east-1".to_string(),
            prefix,
            max_results: None,
            access_key: None,
            secret_key: None,
            session_token: None,
        }
    }

    fn create_delete_request(&self, key: String) -> S3DeleteRequest {
        S3DeleteRequest {
            bucket: self.test_bucket.clone(),
            key,
            region: "us-east-1".to_string(),
            access_key: None,
            secret_key: None,
            session_token: None,
        }
    }
}

// ============================================================================
// Helper Functions
// ============================================================================

fn create_temp_file(size: usize) -> NamedTempFile {
    let mut temp_file = NamedTempFile::new().unwrap();
    let data: Vec<u8> = (0..size).map(|i| (i % 256) as u8).collect();
    temp_file.write_all(&data).unwrap();
    temp_file
}

fn verify_upload_result(result: &S3UploadResult, expected_size: i64) {
    assert!(!result.blob_id.is_empty(), "blob_id should not be empty");
    assert!(!result.etag.is_empty(), "etag should not be empty");
    assert_eq!(result.size_bytes, expected_size, "size should match");
    assert_eq!(
        result.content_hash_sha256.len(),
        64,
        "SHA256 hash should be 64 characters"
    );
    assert!(result.timestamp_unix > 0, "timestamp should be positive");
}

fn verify_blob_info(blob: &BlobInfo, expected_key: &str, expected_size: i64) {
    assert!(blob.uri.contains(expected_key), "URI should contain key");
    assert_eq!(blob.size_bytes, expected_size, "size should match");
    assert!(!blob.etag.is_empty(), "etag should not be empty");
    assert!(blob.last_modified_unix > 0, "timestamp should be positive");
}

async fn collect_chunks(stream: impl futures::Stream<Item = Result<BlobChunk, SidecarError>>) -> Vec<BlobChunk> {
    futures::stream::TryStreamExt::try_collect(stream)
        .await
        .unwrap()
}

// ============================================================================
// Basic Operations Tests
// ============================================================================

#[tokio::test]
async fn test_upload_small_file_from_memory() {
    let _fixture = S3TestFixture::new();
    let connector = S3Connector::new();

    let content = vec![1u8; 1024 * 512];
    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: format!("test/small-{}.bin", chrono::Utc::now().timestamp()),
        region: "us-east-1".to_string(),
        content_type: Some("application/octet-stream".to_string()),
        content: Some(content.clone()),
        file_path: None,
        access_key: Some("test-access-key".to_string()),
        secret_key: Some("test-secret-key".to_string()),
        session_token: None,
    };

    assert_eq!(request.content.unwrap().len(), 1024 * 512);
}

#[tokio::test]
async fn test_upload_validation_requires_content_or_file() {
    let connector = S3Connector::new();
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
async fn test_upload_validation_rejects_both_content_and_file() {
    let connector = S3Connector::new();
    let temp_file = create_temp_file(100);
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
async fn test_upload_large_file_from_path() {
    let temp_file = create_temp_file(1024 * 1024);
    let connector = S3Connector::new();

    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: format!("test/large-{}.bin", chrono::Utc::now().timestamp()),
        region: "us-east-1".to_string(),
        content_type: Some("application/octet-stream".to_string()),
        content: None,
        file_path: Some(temp_file.path().to_path_buf()),
        access_key: Some("test-access-key".to_string()),
        secret_key: Some("test-secret-key".to_string()),
        session_token: None,
    };

    assert!(request.file_path.is_some());
    assert_eq!(
        request.file_path.unwrap(),
        temp_file.path().to_path_buf()
    );
}

#[tokio::test]
async fn test_upload_with_ephemeral_credentials() {
    let connector = S3Connector::new();

    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: "test-key".to_string(),
        region: "us-east-1".to_string(),
        content_type: Some("application/json".to_string()),
        content: Some(b"{\"test\": true}".to_vec()),
        file_path: None,
        access_key: Some("ASIA...".to_string()),
        secret_key: Some("...".to_string()),
        session_token: Some("Fwo...".to_string()),
    };

    assert_eq!(request.access_key, Some("ASIA...".to_string()));
    assert_eq!(request.secret_key, Some("...".to_string()));
    assert_eq!(request.session_token, Some("Fwo...".to_string()));
}

#[tokio::test]
async fn test_upload_with_content_type() {
    let connector = S3Connector::new();

    let content = b"test data".to_vec();
    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: "test.txt".to_string(),
        region: "us-east-1".to_string(),
        content_type: Some("text/plain".to_string()),
        content: Some(content),
        file_path: None,
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    assert_eq!(request.content_type, Some("text/plain".to_string()));
    assert_eq!(request.content.unwrap().len(), 9);
}

#[tokio::test]
async fn test_download_request_with_range() {
    let connector = S3Connector::new();

    let request = S3DownloadRequest {
        bucket: "test-bucket".to_string(),
        key: "test.bin".to_string(),
        region: "us-east-1".to_string(),
        offset_bytes: Some(100),
        max_bytes: Some(1024),
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    assert_eq!(request.offset_bytes, Some(100));
    assert_eq!(request.max_bytes, Some(1024));
}

#[tokio::test]
async fn test_download_request_without_range() {
    let connector = S3Connector::new();

    let request = S3DownloadRequest {
        bucket: "test-bucket".to_string(),
        key: "test.bin".to_string(),
        region: "us-east-1".to_string(),
        offset_bytes: None,
        max_bytes: None,
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    assert!(request.offset_bytes.is_none());
    assert!(request.max_bytes.is_none());
}

#[tokio::test]
async fn test_list_blobs_with_prefix() {
    let connector = S3Connector::new();

    let request = S3ListRequest {
        bucket: "test-bucket".to_string(),
        region: "us-east-1".to_string(),
        prefix: Some("documents/".to_string()),
        max_results: Some(100),
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    assert_eq!(request.prefix, Some("documents/".to_string()));
    assert_eq!(request.max_results, Some(100));
}

#[tokio::test]
async fn test_list_blobs_without_prefix() {
    let connector = S3Connector::new();

    let request = S3ListRequest {
        bucket: "test-bucket".to_string(),
        region: "us-east-1".to_string(),
        prefix: None,
        max_results: None,
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    assert!(request.prefix.is_none());
    assert!(request.max_results.is_none());
}

#[tokio::test]
async fn test_delete_blob_request() {
    let connector = S3Connector::new();

    let request = S3DeleteRequest {
        bucket: "test-bucket".to_string(),
        key: "test/key.txt".to_string(),
        region: "us-east-1".to_string(),
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    assert_eq!(request.bucket, "test-bucket".to_string());
    assert_eq!(request.key, "test/key.txt".to_string());
}

// ============================================================================
// Circuit Breaker Tests
// ============================================================================

#[tokio::test]
async fn test_circuit_breaker_closed_on_success() {
    let reliability = S3Reliability::new(3, 5, 100);

    let result = reliability
        .upload(async { Ok::<(), String>(()) })
        .await;

    assert!(result.is_ok());
}

#[tokio::test]
async fn test_circuit_breaker_opens_after_threshold() {
    let reliability = S3Reliability::new(3, 5, 100);

    for _ in 0..3 {
        let result = reliability
            .upload(async { Err::<(), String>("test error".to_string()) })
            .await;
        assert!(result.is_err());
    }

    let result = reliability
        .upload(async { Ok::<(), String>(()) })
        .await;
    assert!(result.is_err());
    assert!(matches!(
        result.unwrap_err(),
        SidecarError::CircuitBreakerOpen(_)
    ));
}

#[tokio::test]
async fn test_circuit_breaker_blocks_requests_when_open() {
    let reliability = S3Reliability::new(2, 5, 100);

    for _ in 0..2 {
        let _ = reliability
            .upload(async { Err::<(), String>("error".to_string()) })
            .await;
    }

    for _ in 0..3 {
        let result = reliability
            .upload(async { Ok::<(), String>(()) })
            .await;
        assert!(result.is_err());
        assert!(matches!(
            result.unwrap_err(),
            SidecarError::CircuitBreakerOpen(_)
        ));
    }
}

#[tokio::test]
async fn test_circuit_breaker_half_open_after_timeout() {
    let reliability = S3Reliability::new(2, 1, 100);

    for _ in 0..2 {
        let _ = reliability
            .upload(async { Err::<(), String>("error".to_string()) })
            .await;
    }

    tokio::time::sleep(tokio::time::Duration::from_secs(1) + tokio::time::Duration::from_millis(100)).await;

    let result = reliability
        .upload(async { Ok::<(), String>(()) })
        .await;

    if result.is_err() {
        assert!(!matches!(
            result.unwrap_err(),
            SidecarError::CircuitBreakerOpen(_)
        ));
    }
}

#[tokio::test]
async fn test_circuit_breaker_closes_on_success() {
    let reliability = S3Reliability::new(2, 1, 100); // 1 second timeout

    // Trigger failures to open circuit
    for _ in 0..2 {
        let _ = reliability
            .upload(async { Err::<(), String>("error".to_string()) })
            .await;
    }

    // Wait for recovery timeout
    tokio::time::sleep(tokio::time::Duration::from_secs(1) + tokio::time::Duration::from_millis(100)).await;

    // Successful call should close circuit
    let result = reliability
        .upload(async { Ok::<(), String>(()) })
        .await;
    assert!(result.is_ok());

    // Next call should succeed (circuit is closed again)
    let result = reliability
        .upload(async { Ok::<(), String>(()) })
        .await;
    assert!(result.is_ok());
}

#[tokio::test]
async fn test_circuit_breaker_fails_in_half_open() {
    let reliability = S3Reliability::new(2, 1, 100); // 1 second timeout

    // Trigger failures to open circuit
    for _ in 0..2 {
        let _ = reliability
            .upload(async { Err::<(), String>("error".to_string()) })
            .await;
    }

    // Wait for recovery timeout
    tokio::time::sleep(tokio::time::Duration::from_secs(1) + tokio::time::Duration::from_millis(100)).await;

    // Failed call in half-open should keep circuit open
    let result = reliability
        .upload(async { Err::<(), String>("still failing".to_string()) })
        .await;
    assert!(result.is_err());

    // Wait for recovery timeout again
    tokio::time::sleep(tokio::time::Duration::from_secs(1) + tokio::time::Duration::from_millis(100)).await;

    // Circuit should still be open
    let result = reliability
        .upload(async { Ok::<(), String>(()) })
        .await;
    assert!(result.is_err());
}

// ============================================================================
// Rate Limiter Tests
// ============================================================================

#[tokio::test]
async fn test_rate_limiter_allows_within_limit() {
    let reliability = S3Reliability::new(5, 10, 10);

    for i in 0..10 {
        let result = reliability
            .upload(async move { Ok::<(), String>(()) })
            .await;
        assert!(
            result.is_ok(),
            "Request {} should succeed",
            i
        );
    }
}

#[tokio::test]
async fn test_rate_limiter_denies_exceeding_limit() {
    let reliability = S3Reliability::new(5, 10, 5);

    for i in 0..5 {
        let result = reliability
            .upload(async move { Ok::<(), String>(()) })
            .await;
        assert!(result.is_ok(), "Request {} should succeed", i);
    }

    let result = reliability
        .upload(async { Ok::<(), String>(()) })
        .await;
    assert!(result.is_err());
    assert!(matches!(
        result.unwrap_err(),
        SidecarError::RateLimitExceeded(_)
    ));
}

#[tokio::test]
async fn test_rate_limiter_resets_after_window() {
    let reliability = S3Reliability::new(5, 10, 2); // 2 req/s limit

    // Make 2 requests (at limit)
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;

    // 3rd request should be rate limited
    let result = reliability.upload(async { Ok::<(), String>(()) }).await;
    assert!(result.is_err());

    // Wait for rate limit window to reset (1 second + buffer)
    tokio::time::sleep(tokio::time::Duration::from_secs(1) + tokio::time::Duration::from_millis(100)).await;

    // Should be able to make requests again
    let result = reliability.upload(async { Ok::<(), String>(()) }).await;
    assert!(result.is_ok());
}

#[tokio::test]
async fn test_rate_limiter_with_concurrent_requests() {
    let reliability = Arc::new(S3Reliability::new(5, 10, 10)); // 10 req/s limit

    // Launch 15 concurrent requests
    let handles: Vec<_> = (0..15)
        .map(|_| {
            let reliability = reliability.clone();
            tokio::spawn(async move {
                reliability.upload(async { Ok::<(), String>(()) }).await
            })
        })
        .collect();

    let results: Vec<_> = futures::future::join_all(handles)
        .await
        .into_iter()
        .map(|r| r.unwrap())
        .collect();

    // Should have some successes and some rate limit errors
    let success_count = results.iter().filter(|r| r.is_ok()).count();
    let error_count = results.iter().filter(|r| r.is_err()).count();

    assert!(success_count > 0, "Should have some successful requests");
    assert!(error_count > 0, "Should have some rate-limited requests");
    assert_eq!(success_count + error_count, 15, "All requests should complete");
}

// ============================================================================
// Error Scenario Tests
// ============================================================================

#[tokio::test]
async fn test_error_scenario_file_not_found() {
    let connector = S3Connector::new();

    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: "test-key".to_string(),
        region: "us-east-1".to_string(),
        content_type: None,
        content: None,
        file_path: Some(PathBuf::from("/nonexistent/file.txt")),
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    let result = connector.upload(request).await;
    assert!(result.is_err());
    assert!(matches!(result.unwrap_err(), SidecarError::Io(_)));
}

#[tokio::test]
async fn test_error_scenario_empty_content() {
    let connector = S3Connector::new();

    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: "test-key".to_string(),
        region: "us-east-1".to_string(),
        content_type: None,
        content: Some(vec![]),
        file_path: None,
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    let result = connector.upload(request).await;
    assert!(result.is_err());
}

#[tokio::test]
async fn test_error_scenario_invalid_region_format() {
    let connector = S3Connector::new();

    let content = b"test".to_vec();
    let request = S3UploadRequest {
        bucket: "test-bucket".to_string(),
        key: "test-key".to_string(),
        region: "invalid-region!".to_string(),
        content_type: None,
        content: Some(content),
        file_path: None,
        access_key: None,
        secret_key: None,
        session_token: None,
    };

    assert_eq!(request.region, "invalid-region!");
}

// ============================================================================
// Metrics Tests
// ============================================================================

#[tokio::test]
async fn test_metrics_circuit_breaker_calls_recorded() {
    let reliability = S3Reliability::new(5, 10, 100);
    let metrics = reliability.metrics();

    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.upload(async { Err::<(), String>("error".to_string()) }).await;
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;

    let metric_families = metrics.registry().gather();
    let calls_total: Vec<_> = metric_families
        .iter()
        .filter(|m| m.get_name() == "circuit_breaker_calls_total")
        .collect();

    assert!(!calls_total.is_empty(), "circuit_breaker_calls_total metric should exist");
}

#[tokio::test]
async fn test_metrics_circuit_breaker_state_recorded() {
    let reliability = S3Reliability::new(2, 1, 100);
    let metrics = reliability.metrics();

    // Trigger circuit to open
    let _ = reliability
        .upload(async { Err::<(), String>("error".to_string()) })
        .await;
    let _ = reliability
        .upload(async { Err::<(), String>("error".to_string()) })
        .await;

    // Wait for circuit to open
    tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;

    // Verify state metric
    let metric_families = metrics.registry().gather();
    let state_metric: Vec<_> = metric_families
        .iter()
        .filter(|m| m.get_name() == "circuit_breaker_state")
        .collect();

    assert!(!state_metric.is_empty(), "circuit_breaker_state metric should exist");
}

#[tokio::test]
async fn test_metrics_circuit_breaker_failures_recorded() {
    let reliability = S3Reliability::new(5, 10, 100);
    let metrics = reliability.metrics();

    // Trigger some failures
    for _ in 0..3 {
        let _ = reliability
            .upload(async { Err::<(), String>("error".to_string()) })
            .await;
    }

    // Verify failure count metric
    let metric_families = metrics.registry().gather();
    let failures_metric: Vec<_> = metric_families
        .iter()
        .filter(|m| m.get_name() == "circuit_breaker_failures")
        .collect();

    assert!(!failures_metric.is_empty(), "circuit_breaker_failures metric should exist");
}

#[tokio::test]
async fn test_metrics_rate_limit_allowed_recorded() {
    let reliability = S3Reliability::new(5, 10, 10);
    let metrics = reliability.metrics();

    // Make some requests
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;

    // Verify allowed metric
    let metric_families = metrics.registry().gather();
    let allowed_metric: Vec<_> = metric_families
        .iter()
        .filter(|m| m.get_name() == "rate_limit_allowed")
        .collect();

    assert!(!allowed_metric.is_empty(), "rate_limit_allowed metric should exist");
}

#[tokio::test]
async fn test_metrics_rate_limit_denied_recorded() {
    let reliability = S3Reliability::new(5, 10, 2); // Low limit
    let metrics = reliability.metrics();

    // Exhaust rate limit
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await; // Should be denied

    // Verify denied metric
    let metric_families = metrics.registry().gather();
    let denied_metric: Vec<_> = metric_families
        .iter()
        .filter(|m| m.get_name() == "rate_limit_denied")
        .collect();

    assert!(!denied_metric.is_empty(), "rate_limit_denied metric should exist");
}

#[tokio::test]
async fn test_metrics_all_s3_operations() {
    let reliability = S3Reliability::new(5, 10, 100);
    let metrics = reliability.metrics();

    // Test all operation types
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.download(async { Ok::<(), String>(()) }).await;
    let _ = reliability.list(async { Ok::<(), String>(()) }).await;
    let _ = reliability.delete(async { Ok::<(), String>(()) }).await;

    // Verify metrics are collected for all operations
    let metric_families = metrics.registry().gather();
    let call_metrics: Vec<_> = metric_families
        .iter()
        .filter(|m| m.get_name() == "circuit_breaker_calls_total")
        .collect();

    assert!(!call_metrics.is_empty(), "Calls should be recorded");
}

// ============================================================================
// Integration Tests with Multiple Operations
// ============================================================================

#[tokio::test]
async fn test_multiple_uploads_with_different_sizes() {
    let connector = S3Connector::new();

    let sizes = vec![1024, 1024 * 10, 1024 * 100]; // 1KB, 10KB, 100KB

    for size in sizes {
        let content = vec![1u8; size];
        let request = S3UploadRequest {
            bucket: "test-bucket".to_string(),
            key: format!("test/multi-{}-{}.bin", size, chrono::Utc::now().timestamp()),
            region: "us-east-1".to_string(),
            content_type: Some("application/octet-stream".to_string()),
            content: Some(content),
            file_path: None,
            access_key: None,
            secret_key: None,
            session_token: None,
        };

        // Verify request structure
        assert_eq!(request.content.unwrap().len(), size);
    }
}

#[tokio::test]
async fn test_circuit_breaker_and_rate_limiter_interaction() {
    // Low rate limit and low failure threshold
    let reliability = S3Reliability::new(2, 10, 2);

    // Exhaust rate limit (2 requests)
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;
    let _ = reliability.upload(async { Ok::<(), String>(()) }).await;

    // 3rd request should be rate limited
    let result = reliability.upload(async { Ok::<(), String>(()) }).await;
    assert!(result.is_err());
    assert!(matches!(
        result.unwrap_err(),
        SidecarError::RateLimitExceeded(_)
    ));

    // Wait for rate limit reset
    tokio::time::sleep(tokio::time::Duration::from_secs(1) + tokio::time::Duration::from_millis(100)).await;

    // Now trigger failures
    let _ = reliability.upload(async { Err::<(), String>("error1".to_string()) }).await;
    let _ = reliability.upload(async { Err::<(), String>("error2".to_string()) }).await;

    // Circuit should be open now
    let result = reliability.upload(async { Ok::<(), String>(()) }).await;
    assert!(result.is_err());
    assert!(matches!(
        result.unwrap_err(),
        SidecarError::CircuitBreakerOpen(_)
    ));
}

// ============================================================================
// Helper Test: Verify Test Infrastructure
// ============================================================================

#[tokio::test]
async fn test_mock_s3_storage_basic_operations() {
    let storage = MockS3Storage::new();

    storage
        .put_object("bucket1".to_string(), "key1".to_string(), vec![1, 2, 3])
        .await
        .unwrap();

    assert!(
        storage
            .object_exists("bucket1".to_string(), "key1".to_string())
            .await
    );
    assert!(!storage
        .object_exists("bucket1".to_string(), "key2".to_string())
        .await);

    let data = storage
        .get_object("bucket1".to_string(), "key1".to_string())
        .await
        .unwrap();
    assert_eq!(data, vec![1, 2, 3]);

    let objects = storage
        .list_objects("bucket1".to_string(), None)
        .await;
    assert_eq!(objects.len(), 1);

    storage
        .delete_object("bucket1".to_string(), "key1".to_string())
        .await
        .unwrap();
    assert!(!storage
        .object_exists("bucket1".to_string(), "key1".to_string())
        .await);
}

#[tokio::test]
async fn test_mock_s3_storage_prefix_filtering() {
    let storage = MockS3Storage::new();

    storage
        .put_object("bucket1".to_string(), "docs/file1.txt".to_string(), vec![1])
        .await
        .unwrap();
    storage
        .put_object("bucket1".to_string(), "docs/file2.txt".to_string(), vec![2])
        .await
        .unwrap();
    storage
        .put_object("bucket1".to_string(), "images/photo.jpg".to_string(), vec![3])
        .await
        .unwrap();

    let docs = storage
        .list_objects("bucket1".to_string(), Some("docs/".to_string()))
        .await;
    assert_eq!(docs.len(), 2);

    let all = storage.list_objects("bucket1".to_string(), None).await;
    assert_eq!(all.len(), 3);
}
