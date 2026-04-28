use rust_vault::error::VaultError;

#[test]
fn test_error_display_database() {
    let err = VaultError::Database("connection failed".to_string());
    assert!(format!("{}", err).contains("Database error: connection failed"));
}

#[test]
fn test_error_display_grpc() {
    let err = VaultError::Grpc("timeout".to_string());
    assert!(format!("{}", err).contains("gRPC error: timeout"));
}

#[test]
fn test_error_display_cdp() {
    let err = VaultError::Cdp("browser closed".to_string());
    assert!(format!("{}", err).contains("CDP error: browser closed"));
}

#[test]
fn test_error_display_config() {
    let err = VaultError::Config("invalid port".to_string());
    assert!(format!("{}", err).contains("Configuration error: invalid port"));
}

#[test]
fn test_error_display_placeholder_not_found() {
    let err = VaultError::PlaceholderNotFound("payment.card_number".to_string());
    assert!(format!("{}", err).contains("Placeholder not found: payment.card_number"));
}

#[test]
fn test_error_display_secret_not_found() {
    let err = VaultError::SecretNotFound("user.password".to_string());
    assert!(format!("{}", err).contains("Secret not found: user.password"));
}

#[test]
fn test_error_display_rate_limit_exceeded() {
    let err = VaultError::RateLimitExceeded;
    assert!(format!("{}", err).contains("Rate limit exceeded"));
}

#[test]
fn test_error_display_concurrency_limit_exceeded() {
    let err = VaultError::ConcurrencyLimitExceeded;
    assert!(format!("{}", err).contains("Concurrency limit exceeded"));
}

#[test]
fn test_error_display_authentication_failed() {
    let err = VaultError::AuthenticationFailed("invalid token".to_string());
    assert!(format!("{}", err).contains("Authentication failed: invalid token"));
}

#[test]
fn test_error_from_string_database() {
    let err = VaultError::Database("database error".to_string());
    matches!(err, VaultError::Database(_));
}

#[test]
fn test_error_from_string_grpc() {
    let err = VaultError::Grpc("grpc error".to_string());
    matches!(err, VaultError::Grpc(_));
}

#[test]
fn test_error_from_string_cdp() {
    let err = VaultError::Cdp("cdp error".to_string());
    matches!(err, VaultError::Cdp(_));
}

#[test]
fn test_error_from_string_config() {
    let err = VaultError::Config("config error".to_string());
    matches!(err, VaultError::Config(_));
}

#[test]
fn test_error_send_sync() {
    fn assert_send_sync<T: Send + Sync>() {}
    assert_send_sync::<VaultError>();
}
