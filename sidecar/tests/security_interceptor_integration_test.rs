use armorclaw_sidecar::grpc::interceptor::SecurityInterceptor;
use armorclaw_sidecar::security::token::parse_token;
use hmac::{Hmac, Mac};
use prometheus::Registry;
use sha2::Sha256;
use std::time::{SystemTime, UNIX_EPOCH};
use tonic::metadata::MetadataMap;

type HmacSha256 = Hmac<Sha256>;

#[tokio::test]
async fn test_interceptor_rejects_request_without_token() {
    let shared_secret = b"test-secret-key-32-bytes-long!";
    let registry = Registry::new();
    let mut interceptor = SecurityInterceptor::new(shared_secret.to_vec(), &registry).unwrap();

    let metadata = MetadataMap::new();
    let request = tonic::Request::new(());

    let mut request_with_metadata = request;
    *request_with_metadata.metadata_mut() = metadata;

    let result = interceptor.call(request_with_metadata);

    assert!(result.is_err());
    let status = result.unwrap_err();
    assert_eq!(status.code(), tonic::Code::Unauthenticated);
    assert!(status.message().contains("missing authentication token"));
}

#[tokio::test]
async fn test_interceptor_rejects_invalid_signature() {
    let shared_secret = b"test-secret-key-32-bytes-long!";
    let registry = Registry::new();
    let mut interceptor = SecurityInterceptor::new(shared_secret.to_vec(), &registry).unwrap();

    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let token = format!("{}:{}:{}:{}", "test-request-id", now, "test-operation", "invalid-signature");

    let mut metadata = MetadataMap::new();
    metadata.insert("token", token.parse().unwrap());

    let request = tonic::Request::new(());

    let mut request_with_metadata = request;
    *request_with_metadata.metadata_mut() = metadata;

    let result = interceptor.call(request_with_metadata);

    assert!(result.is_err());
    let status = result.unwrap_err();
    assert_eq!(status.code(), tonic::Code::Unauthenticated);
    assert!(status.message().contains("invalid token signature"));
}

#[tokio::test]
async fn test_interceptor_rejects_expired_token() {
    let shared_secret = b"test-secret-key-32-bytes-long!";
    let registry = Registry::new();
    let mut interceptor = SecurityInterceptor::new(shared_secret.to_vec(), &registry).unwrap();

    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let timestamp = now - 2000;
    let request_id = "test-request-id";
    let operation = "test-operation";

    let data_to_sign = format!("{}{}{}", request_id, timestamp, operation);
    let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
    mac.update(data_to_sign.as_bytes());
    let signature = hex::encode(mac.finalize().into_bytes());

    let token = format!("{}:{}:{}:{}", request_id, timestamp, operation, signature);

    let mut metadata = MetadataMap::new();
    metadata.insert("token", token.parse().unwrap());

    let request = tonic::Request::new(());

    let mut request_with_metadata = request;
    *request_with_metadata.metadata_mut() = metadata;

    let result = interceptor.call(request_with_metadata);

    assert!(result.is_err());
    let status = result.unwrap_err();
    assert_eq!(status.code(), tonic::Code::Unauthenticated);
    assert!(status.message().contains("expired"));
}

#[tokio::test]
async fn test_interceptor_rejects_token_too_old() {
    let shared_secret = b"test-secret-key-32-bytes-long!";
    let registry = Registry::new();
    let mut interceptor = SecurityInterceptor::new(shared_secret.to_vec(), &registry).unwrap();

    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let timestamp = now - 400;
    let request_id = "test-request-id";
    let operation = "test-operation";

    let data_to_sign = format!("{}{}{}", request_id, timestamp, operation);
    let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
    mac.update(data_to_sign.as_bytes());
    let signature = hex::encode(mac.finalize().into_bytes());

    let token = format!("{}:{}:{}:{}", request_id, timestamp, operation, signature);

    let mut metadata = MetadataMap::new();
    metadata.insert("token", token.parse().unwrap());

    let request = tonic::Request::new(());

    let mut request_with_metadata = request;
    *request_with_metadata.metadata_mut() = metadata;

    let result = interceptor.call(request_with_metadata);

    assert!(result.is_err());
    let status = result.unwrap_err();
    assert_eq!(status.code(), tonic::Code::Unauthenticated);
    assert!(status.message().contains("too old"));
}

#[tokio::test]
async fn test_interceptor_accepts_valid_token() {
    let shared_secret = b"test-secret-key-32-bytes-long!";
    let registry = Registry::new();
    let mut interceptor = SecurityInterceptor::new(shared_secret.to_vec(), &registry).unwrap();

    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let request_id = "test-request-id";
    let operation = "test-operation";

    let data_to_sign = format!("{}{}{}", request_id, now, operation);
    let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
    mac.update(data_to_sign.as_bytes());
    let signature = hex::encode(mac.finalize().into_bytes());

    let token = format!("{}:{}:{}:{}", request_id, now, operation, signature);

    let mut metadata = MetadataMap::new();
    metadata.insert("token", token.parse().unwrap());
    metadata.insert("request-id", request_id.parse().unwrap());
    metadata.insert("operation", operation.parse().unwrap());

    let request = tonic::Request::new(());

    let mut request_with_metadata = request;
    *request_with_metadata.metadata_mut() = metadata;

    let result = interceptor.call(request_with_metadata);

    assert!(result.is_ok());
}

#[tokio::test]
async fn test_interceptor_increments_metrics_on_success() {
    let shared_secret = b"test-secret-key-32-bytes-long!";
    let registry = Registry::new();
    let mut interceptor = SecurityInterceptor::new(shared_secret.to_vec(), &registry).unwrap();

    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs() as i64;

    let request_id = "test-request-id";
    let operation = "test-operation";

    let data_to_sign = format!("{}{}{}", request_id, now, operation);
    let mut mac = HmacSha256::new_from_slice(shared_secret).unwrap();
    mac.update(data_to_sign.as_bytes());
    let signature = hex::encode(mac.finalize().into_bytes());

    let token = format!("{}:{}:{}:{}", request_id, now, operation, signature);

    let mut metadata = MetadataMap::new();
    metadata.insert("token", token.parse().unwrap());

    let request = tonic::Request::new(());

    let mut request_with_metadata = request;
    *request_with_metadata.metadata_mut() = metadata;

    let requests_total_before = registry
        .gather()
        .iter()
        .find(|m| m.get_name() == "armorclaw_sidecar_requests_total")
        .and_then(|m| m.get_metric().get_sample_count());

    let _ = interceptor.call(request_with_metadata);

    let requests_total_after = registry
        .gather()
        .iter()
        .find(|m| m.get_name() == "armorclaw_sidecar_requests_total")
        .and_then(|m| m.get_metric().get_sample_count());

    assert!(requests_total_after > requests_total_before.unwrap_or(0));
}

#[tokio::test]
async fn test_interceptor_increments_error_metrics_on_failure() {
    let shared_secret = b"test-secret-key-32-bytes-long!";
    let registry = Registry::new();
    let mut interceptor = SecurityInterceptor::new(shared_secret.to_vec(), &registry).unwrap();

    let metadata = MetadataMap::new();
    let request = tonic::Request::new(());

    let mut request_with_metadata = request;
    *request_with_metadata.metadata_mut() = metadata;

    let errors_total_before = registry
        .gather()
        .iter()
        .find(|m| m.get_name() == "armorclaw_sidecar_request_errors_total")
        .and_then(|m| m.get_metric().get_sample_count());

    let _ = interceptor.call(request_with_metadata);

    let errors_total_after = registry
        .gather()
        .iter()
        .find(|m| m.get_name() == "armorclaw_sidecar_request_errors_total")
        .and_then(|m| m.get_metric().get_sample_count());

    assert!(errors_total_after > errors_total_before.unwrap_or(0));
}
