use serde::Deserialize;
use std::path::PathBuf;

#[derive(Debug, Clone, Deserialize)]
pub struct TlsConfig {
    /// Path to the server's TLS certificate file (PEM format)
    pub cert_path: PathBuf,
    /// Path to the server's TLS private key file (PEM format)
    pub key_path: PathBuf,
    /// Path to the CA certificate for validating client certificates (PEM format)
    pub ca_cert_path: PathBuf,
    /// Whether to require client certificate verification (mTLS)
    pub verify_client: bool,
    /// List of allowed client certificate Common Names (CN). If empty, all valid certificates are accepted.
    pub allowed_client_cns: Vec<String>,
    /// Whether to enforce certificate expiration checking
    pub enforce_expiration: bool,
}

impl Default for TlsConfig {
    fn default() -> Self {
        Self {
            cert_path: PathBuf::from("/etc/armorclaw/certs/server.crt"),
            key_path: PathBuf::from("/etc/armorclaw/certs/server.key"),
            ca_cert_path: PathBuf::from("/etc/armorclaw/certs/ca.crt"),
            verify_client: true,
            allowed_client_cns: Vec::new(),
            enforce_expiration: true,
        }
    }
}

#[derive(Debug, Clone, Deserialize)]
pub struct VaultConfig {
    pub keystore_socket_path: PathBuf,
    pub ephemeral_token_ttl_seconds: u64,
    pub keystore_request_timeout_seconds: u64,
    pub cdp_port: u16,
    pub placeholder_prefix: String,
    pub placeholder_suffix: String,
    pub rate_limit_max_requests_per_second: u32,
    pub burst_capacity: u32,
    pub concurrency_limit: usize,
    pub log_level: String,
    /// TLS configuration for mTLS authentication
    pub tls: Option<TlsConfig>,
    /// Whether to use TLS (false = Unix domain socket, true = TLS TCP)
    pub use_tls: bool,
    /// Listen address for TLS server (e.g., "0.0.0.0:8443")
    pub tls_listen_addr: String,
}

impl Default for VaultConfig {
    fn default() -> Self {
        Self {
            keystore_socket_path: PathBuf::from("/run/armorclaw/keystore.sock"),
            ephemeral_token_ttl_seconds: 1800,
            keystore_request_timeout_seconds: 5,
            cdp_port: 9222,
            placeholder_prefix: "{{secret:".to_string(),
            placeholder_suffix: "}}".to_string(),
            rate_limit_max_requests_per_second: 100,
            burst_capacity: 200,
            concurrency_limit: 50,
            log_level: "info".to_string(),
            tls: None,
            use_tls: false,
            tls_listen_addr: "0.0.0.0:8443".to_string(),
        }
    }
}
