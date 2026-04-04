use rust_vault::config::VaultConfig;
use std::path::PathBuf;

#[test]
fn test_config_defaults() {
    let config = VaultConfig::default();
    assert_eq!(config.keystore_socket_path, PathBuf::from("/run/armorclaw/keystore.sock"));
    assert_eq!(config.ephemeral_token_ttl_seconds, 1800);
    assert_eq!(config.keystore_request_timeout_seconds, 5);
    assert_eq!(config.cdp_port, 9222);
    assert_eq!(config.placeholder_prefix, "{{secret:");
    assert_eq!(config.placeholder_suffix, "}}");
    assert_eq!(config.rate_limit_max_requests_per_second, 100);
    assert_eq!(config.burst_capacity, 200);
    assert_eq!(config.concurrency_limit, 50);
    assert_eq!(config.log_level, "info");
}

#[test]
fn test_config_clone() {
    let config = VaultConfig::default();
    let cloned = config.clone();

    assert_eq!(config.keystore_socket_path, cloned.keystore_socket_path);
    assert_eq!(config.ephemeral_token_ttl_seconds, cloned.ephemeral_token_ttl_seconds);
}

#[test]
fn test_config_debug() {
    let config = VaultConfig::default();
    let debug_str = format!("{:?}", config);
    assert!(debug_str.contains("VaultConfig"));
    assert!(debug_str.contains("keystore_socket_path"));
}

#[test]
fn test_config_custom_values() {
    let mut config = VaultConfig::default();
    config.keystore_socket_path = PathBuf::from("/custom/path.sock");
    config.cdp_port = 9333;
    config.rate_limit_max_requests_per_second = 200;

    assert_eq!(config.keystore_socket_path, PathBuf::from("/custom/path.sock"));
    assert_eq!(config.cdp_port, 9333);
    assert_eq!(config.rate_limit_max_requests_per_second, 200);
}

#[test]
fn test_config_no_secrets() {
    let config = VaultConfig::default();
    let debug_str = format!("{:?}", config);
    assert!(!debug_str.contains("password"));
    assert!(!debug_str.contains("secret_value"));
    assert!(!debug_str.contains("credential"));
    assert!(!debug_str.contains("api_key"));
}
