use config::{Config, ConfigError};
use serde::Deserialize;
use std::path::PathBuf;

#[derive(Debug, Clone, Deserialize)]
pub struct SidecarConfig {
    pub socket_path: PathBuf,
    pub socket_permissions: String,
    pub max_concurrent_requests: usize,
    pub rate_limit_per_second: u32,
    pub request_timeout_seconds: u64,
    pub temp_directory: PathBuf,
    pub max_file_size_bytes: u64,
    pub log_level: String,
    pub metrics_port: u16,
    pub shared_secret: String,

    // Circuit breaker configuration
    pub circuit_breaker_failure_threshold: u32,
    pub circuit_breaker_recovery_timeout_secs: u64,

    // Rate limiting configuration
    pub rate_limit_max_requests_per_second: u32,
}

impl SidecarConfig {
    pub fn from_env() -> Result<Self, ConfigError> {
        let config = Config::builder()
            .set_default("socket_path", "/run/armorclaw/sidecar.sock")?
            .set_default("socket_permissions", "0")?
            .set_default("max_concurrent_requests", "50")?
            .set_default("rate_limit_per_second", "100")?
            .set_default("request_timeout_seconds", "300")?
            .set_default("temp_directory", "/tmp/armorclaw")?
            .set_default("max_file_size_bytes", "5368709120")?
            .set_default("log_level", "info")?
            .set_default("metrics_port", "9090")?
            .set_default("shared_secret", "")?
            .set_default("circuit_breaker_failure_threshold", "5")?
            .set_default("circuit_breaker_recovery_timeout_secs", "30")?
            .set_default("rate_limit_max_requests_per_second", "100")?
            .add_source(
                config::Environment::with_prefix("ARMORCLAW_SIDECAR")
                    .separator("__")
                    .try_parsing(true),
            )
            .build()?;

        config.try_deserialize()
    }
}
