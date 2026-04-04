use rust_vault::config::{TlsConfig, VaultConfig};
use rust_vault::grpc::GrpcServer;
use rust_vault::grpc::middleware::MtlsInterceptor;
use std::path::PathBuf;
use prometheus::Registry;
use tracing_test::traced_test;

fn get_test_cert_path(name: &str) -> PathBuf {
    let mut path = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    path.push("tests/certs");
    path.push(name);
    path
}

#[tokio::test]
#[traced_test]
async fn test_mtls_interceptor_with_valid_client() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec!["test-client".to_string()],
        enforce_expiration: true,
    };

    let registry = Registry::new();
    let interceptor = MtlsInterceptor::new(tls_config.clone(), &registry).unwrap();

    let cert_info = rust_vault::grpc::middleware::ClientCertInfo {
        common_name: "test-client".to_string(),
        subject_alt_names: vec![],
        serial_number: "12345".to_string(),
        issuer: "CN=Test CA".to_string(),
        not_before: "2025-01-01T00:00:00Z".to_string(),
        not_after: "2027-12-31T23:59:59Z".to_string(),
    };

    let result = interceptor.validate_client_cert(&cert_info);
    assert!(result.is_ok());
}

#[tokio::test]
#[traced_test]
async fn test_mtls_interceptor_rejects_unallowed_cn() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec!["allowed-client".to_string()],
        enforce_expiration: true,
    };

    let registry = Registry::new();
    let interceptor = MtlsInterceptor::new(tls_config.clone(), &registry).unwrap();

    let cert_info = rust_vault::grpc::middleware::ClientCertInfo {
        common_name: "test-client".to_string(),
        subject_alt_names: vec![],
        serial_number: "12345".to_string(),
        issuer: "CN=Test CA".to_string(),
        not_before: "2025-01-01T00:00:00Z".to_string(),
        not_after: "2027-12-31T23:59:59Z".to_string(),
    };

    let result = interceptor.validate_client_cert(&cert_info);
    assert!(result.is_err());
    assert_eq!(result.unwrap_err().code(), tonic::Code::PermissionDenied);
}

#[tokio::test]
#[traced_test]
async fn test_mtls_interceptor_rejects_expired_cert() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec!["test-client".to_string()],
        enforce_expiration: true,
    };

    let registry = Registry::new();
    let interceptor = MtlsInterceptor::new(tls_config.clone(), &registry).unwrap();

    let cert_info = rust_vault::grpc::middleware::ClientCertInfo {
        common_name: "test-client".to_string(),
        subject_alt_names: vec![],
        serial_number: "12345".to_string(),
        issuer: "CN=Test CA".to_string(),
        not_before: "2023-01-01T00:00:00Z".to_string(),
        not_after: "2024-12-31T23:59:59Z".to_string(),
    };

    let result = interceptor.validate_client_cert(&cert_info);
    assert!(result.is_err());
    assert!(result.unwrap_err().message().contains("expired"));
}

#[tokio::test]
#[traced_test]
async fn test_mtls_interceptor_allows_future_valid_cert() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec!["test-client".to_string()],
        enforce_expiration: true,
    };

    let registry = Registry::new();
    let interceptor = MtlsInterceptor::new(tls_config.clone(), &registry).unwrap();

    let one_year_from_now = chrono::Utc::now() + chrono::Duration::days(365);
    let two_years_from_now = chrono::Utc::now() + chrono::Duration::days(730);

    let cert_info = rust_vault::grpc::middleware::ClientCertInfo {
        common_name: "test-client".to_string(),
        subject_alt_names: vec![],
        serial_number: "12345".to_string(),
        issuer: "CN=Test CA".to_string(),
        not_before: one_year_from_now.to_rfc3339(),
        not_after: two_years_from_now.to_rfc3339(),
    };

    let result = interceptor.validate_client_cert(&cert_info);
    assert!(result.is_err());
    assert!(result.unwrap_err().message().contains("not yet valid"));
}

#[tokio::test]
#[traced_test]
async fn test_mtls_interceptor_with_empty_allowed_cns() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec![],
        enforce_expiration: true,
    };

    let registry = Registry::new();
    let interceptor = MtlsInterceptor::new(tls_config.clone(), &registry).unwrap();

    let cert_info = rust_vault::grpc::middleware::ClientCertInfo {
        common_name: "any-client".to_string(),
        subject_alt_names: vec![],
        serial_number: "12345".to_string(),
        issuer: "CN=Test CA".to_string(),
        not_before: "2025-01-01T00:00:00Z".to_string(),
        not_after: "2027-12-31T23:59:59Z".to_string(),
    };

    let result = interceptor.validate_client_cert(&cert_info);
    assert!(result.is_ok());
}

#[tokio::test]
#[traced_test]
async fn test_mtls_interceptor_without_expiration_check() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec!["test-client".to_string()],
        enforce_expiration: false,
    };

    let registry = Registry::new();
    let interceptor = MtlsInterceptor::new(tls_config.clone(), &registry).unwrap();

    let cert_info = rust_vault::grpc::middleware::ClientCertInfo {
        common_name: "test-client".to_string(),
        subject_alt_names: vec![],
        serial_number: "12345".to_string(),
        issuer: "CN=Test CA".to_string(),
        not_before: "2023-01-01T00:00:00Z".to_string(),
        not_after: "2024-12-31T23:59:59Z".to_string(),
    };

    let result = interceptor.validate_client_cert(&cert_info);
    assert!(result.is_ok());
}

#[tokio::test]
#[traced_test]
async fn test_server_creates_with_tls_config() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec!["test-client".to_string()],
        enforce_expiration: true,
    };

    let config = VaultConfig {
        use_tls: true,
        tls: Some(tls_config),
        tls_listen_addr: "127.0.0.1:0".to_string(),
        ..Default::default()
    };

    let server = GrpcServer::new(config);
    assert!(server.is_ok());
}

#[tokio::test]
#[traced_test]
async fn test_server_fails_without_tls_config() {
    let config = VaultConfig {
        use_tls: true,
        tls: None,
        tls_listen_addr: "127.0.0.1:0".to_string(),
        ..Default::default()
    };

    let server = GrpcServer::new(config);
    assert!(server.is_err());
}

#[tokio::test]
#[traced_test]
async fn test_server_creates_with_unix_socket() {
    let config = VaultConfig {
        use_tls: false,
        tls: None,
        ..Default::default()
    };

    let server = GrpcServer::new(config);
    assert!(server.is_ok());
}

#[tokio::test]
#[traced_test]
async fn test_mtls_metrics_recording() {
    let tls_config = TlsConfig {
        cert_path: get_test_cert_path("server.crt"),
        key_path: get_test_cert_path("server.key"),
        ca_cert_path: get_test_cert_path("ca.crt"),
        verify_client: true,
        allowed_client_cns: vec!["test-client".to_string()],
        enforce_expiration: true,
    };

    let registry = Registry::new();
    let interceptor = MtlsInterceptor::new(tls_config.clone(), &registry).unwrap();

    let cert_info = rust_vault::grpc::middleware::ClientCertInfo {
        common_name: "test-client".to_string(),
        subject_alt_names: vec![],
        serial_number: "12345".to_string(),
        issuer: "CN=Test CA".to_string(),
        not_before: "2025-01-01T00:00:00Z".to_string(),
        not_after: "2027-12-31T23:59:59Z".to_string(),
    };

    interceptor.validate_client_cert(&cert_info).unwrap();

    let metrics = prometheus::TextEncoder::new()
        .encode_to_string(&registry.gather())
        .unwrap();

    assert!(metrics.contains("armorclaw_vault_mtls_auth_successes_total"));
}
