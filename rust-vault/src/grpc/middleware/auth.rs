use crate::config::TlsConfig;
use prometheus::{Histogram, IntCounter, Registry};
#[cfg(test)]
use std::path::PathBuf;
use std::sync::Arc;
use std::time::Instant;
use tonic::{
    metadata::MetadataMap,
    service::Interceptor,
    Request, Status, Code,
};
use tracing::{debug, info, warn};

const METADATA_CLIENT_CN_KEY: &str = "x-client-cn";
const METADATA_CLIENT_SAN_KEY: &str = "x-client-san";

#[derive(Debug, Clone)]
pub struct ClientCertInfo {
    pub common_name: String,
    pub subject_alt_names: Vec<String>,
    pub serial_number: String,
    pub issuer: String,
    pub not_before: String,
    pub not_after: String,
}

#[derive(Clone)]
pub struct MtlsInterceptor {
    tls_config: Arc<TlsConfig>,
    requests_total: IntCounter,
    auth_failures_total: IntCounter,
    auth_successes_total: IntCounter,
    request_duration: Histogram,
}

impl MtlsInterceptor {
    pub fn new(tls_config: TlsConfig, registry: &Registry) -> Result<Self, prometheus::Error> {
        let requests_total = IntCounter::new(
            "armorclaw_vault_mtls_requests_total",
            "Total number of requests with mTLS authentication",
        )?;

        let auth_failures_total = IntCounter::new(
            "armorclaw_vault_mtls_auth_failures_total",
            "Total number of mTLS authentication failures",
        )?;

        let auth_successes_total = IntCounter::new(
            "armorclaw_vault_mtls_auth_successes_total",
            "Total number of mTLS authentication successes",
        )?;

        let request_duration = Histogram::with_opts(
            prometheus::HistogramOpts::new(
                "armorclaw_vault_mtls_request_duration_seconds",
                "Request duration in seconds with mTLS",
            )
            .buckets(vec![0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0]),
        )?;

        registry.register(Box::new(requests_total.clone()))?;
        registry.register(Box::new(auth_failures_total.clone()))?;
        registry.register(Box::new(auth_successes_total.clone()))?;
        registry.register(Box::new(request_duration.clone()))?;

        Ok(Self {
            tls_config: Arc::new(tls_config),
            requests_total,
            auth_failures_total,
            auth_successes_total,
            request_duration,
        })
    }

    pub fn validate_client_cert(
        &self,
        cert_info: &ClientCertInfo,
    ) -> Result<(), Status> {
        if self.tls_config.verify_client {
            debug!(
                cn = %cert_info.common_name,
                serial = %cert_info.serial_number,
                issuer = %cert_info.issuer,
                "Validating client certificate"
            );
        }

        if !self.tls_config.allowed_client_cns.is_empty() {
            if !self.tls_config.allowed_client_cns.contains(&cert_info.common_name) {
                warn!(
                    cn = %cert_info.common_name,
                    allowed_cns = ?self.tls_config.allowed_client_cns,
                    "Client certificate CN not in allowlist"
                );
                return Err(Status::new(
                    Code::PermissionDenied,
                    format!(
                        "Client certificate CN '{}' not authorized",
                        cert_info.common_name
                    ),
                ));
            }
            debug!(
                cn = %cert_info.common_name,
                "Client CN is in allowlist"
            );
        }

        if self.tls_config.enforce_expiration {
            if let Err(e) = self.check_cert_expiration(cert_info) {
                warn!(
                    cn = %cert_info.common_name,
                    error = %e,
                    "Certificate expired or not yet valid"
                );
                return Err(Status::new(Code::PermissionDenied, e));
            }
            debug!(
                cn = %cert_info.common_name,
                not_after = %cert_info.not_after,
                "Certificate expiration check passed"
            );
        }

        Ok(())
    }

    fn check_cert_expiration(&self, cert_info: &ClientCertInfo) -> Result<String, String> {
        let now = chrono::Utc::now();
        let not_after = cert_info
            .not_after
            .parse::<chrono::DateTime<chrono::Utc>>()
            .map_err(|e| format!("Failed to parse not_after: {}", e))?;

        if now > not_after {
            return Err(format!(
                "Certificate expired on {}",
                cert_info.not_after
            ));
        }

        let not_before = cert_info
            .not_before
            .parse::<chrono::DateTime<chrono::Utc>>()
            .map_err(|e| format!("Failed to parse not_before: {}", e))?;

        if now < not_before {
            return Err(format!(
                "Certificate not yet valid (valid from {})",
                cert_info.not_before
            ));
        }

        Ok(cert_info.not_after.clone())
    }

    fn extract_cert_info_from_metadata(
        metadata: &MetadataMap,
    ) -> Result<ClientCertInfo, Status> {
        let common_name = metadata
            .get(METADATA_CLIENT_CN_KEY)
            .and_then(|v| v.to_str().ok())
            .ok_or_else(|| {
                Status::new(Code::Unauthenticated, "Missing client CN in metadata")
            })?;

        let subject_alt_names = metadata
            .get(METADATA_CLIENT_SAN_KEY)
            .and_then(|v| v.to_str().ok())
            .map(|s| s.split(',').map(String::from).collect())
            .unwrap_or_default();

        let serial_number = metadata
            .get("x-client-serial")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("unknown")
            .to_string();

        let issuer = metadata
            .get("x-client-issuer")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("unknown")
            .to_string();

        let not_before = metadata
            .get("x-client-not-before")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("unknown")
            .to_string();

        let not_after = metadata
            .get("x-client-not-after")
            .and_then(|v| v.to_str().ok())
            .unwrap_or("unknown")
            .to_string();

        Ok(ClientCertInfo {
            common_name: common_name.to_string(),
            subject_alt_names,
            serial_number,
            issuer,
            not_before,
            not_after,
        })
    }
}

impl Interceptor for MtlsInterceptor {
    fn call(&mut self, req: Request<()>) -> Result<Request<()>, Status> {
        let start_time = Instant::now();
        let metadata = req.metadata();

        self.requests_total.inc();

        let cert_info = match Self::extract_cert_info_from_metadata(metadata) {
            Ok(info) => info,
            Err(e) => {
                self.auth_failures_total.inc();
                warn!(
                    error = %e.message(),
                    "Failed to extract client certificate info"
                );
                return Err(e);
            }
        };

        match self.validate_client_cert(&cert_info) {
            Ok(_) => {
                self.auth_successes_total.inc();
                info!(
                    cn = %cert_info.common_name,
                    serial = %cert_info.serial_number,
                    "mTLS authentication successful"
                );

                let latency = start_time.elapsed();
                self.request_duration.observe(latency.as_secs_f64());

                debug!(
                    cn = %cert_info.common_name,
                    latency_ms = latency.as_millis(),
                    "Request completed"
                );

                Ok(req)
            }
            Err(e) => {
                self.auth_failures_total.inc();
                warn!(
                    cn = %cert_info.common_name,
                    error = %e.message(),
                    "mTLS authentication failed"
                );
                Err(e)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn create_test_tls_config() -> TlsConfig {
        TlsConfig {
            cert_path: PathBuf::from("/tmp/test.crt"),
            key_path: PathBuf::from("/tmp/test.key"),
            ca_cert_path: PathBuf::from("/tmp/ca.crt"),
            verify_client: true,
            allowed_client_cns: vec!["test-client".to_string()],
            enforce_expiration: true,
        }
    }

    fn create_test_tls_config_unallowed() -> TlsConfig {
        TlsConfig {
            cert_path: PathBuf::from("/tmp/test.crt"),
            key_path: PathBuf::from("/tmp/test.key"),
            ca_cert_path: PathBuf::from("/tmp/ca.crt"),
            verify_client: true,
            allowed_client_cns: vec!["allowed-client".to_string()],
            enforce_expiration: true,
        }
    }

    #[test]
    fn test_mtls_interceptor_creation() {
        let tls_config = create_test_tls_config();
        let registry = Registry::new();
        let interceptor = MtlsInterceptor::new(tls_config, &registry);
        assert!(interceptor.is_ok());
    }

    #[test]
    fn test_extract_cert_info_with_metadata() {
        let mut metadata = MetadataMap::new();
        metadata.insert(
            METADATA_CLIENT_CN_KEY,
            tonic::metadata::MetadataValue::try_from("test-client").unwrap(),
        );
        metadata.insert(
            METADATA_CLIENT_SAN_KEY,
            tonic::metadata::MetadataValue::try_from("DNS:test.example.com,IP:127.0.0.1").unwrap(),
        );
        metadata.insert(
            "x-client-serial",
            tonic::metadata::MetadataValue::try_from("12345").unwrap(),
        );
        metadata.insert(
            "x-client-issuer",
            tonic::metadata::MetadataValue::try_from("CN=Test CA").unwrap(),
        );
        metadata.insert(
            "x-client-not-before",
            tonic::metadata::MetadataValue::try_from("2024-01-01T00:00:00Z").unwrap(),
        );
        metadata.insert(
            "x-client-not-after",
            tonic::metadata::MetadataValue::try_from("2025-12-31T23:59:59Z").unwrap(),
        );

        let cert_info = MtlsInterceptor::extract_cert_info_from_metadata(&metadata).unwrap();
        assert_eq!(cert_info.common_name, "test-client");
        assert_eq!(cert_info.subject_alt_names.len(), 2);
        assert_eq!(cert_info.serial_number, "12345");
        assert_eq!(cert_info.issuer, "CN=Test CA");
    }

    #[test]
    fn test_extract_cert_info_missing_cn() {
        let metadata = MetadataMap::new();
        let result = MtlsInterceptor::extract_cert_info_from_metadata(&metadata);
        assert!(result.is_err());
        assert_eq!(result.unwrap_err().code(), Code::Unauthenticated);
    }

    #[test]
    fn test_validate_allowed_cn() {
        let tls_config = create_test_tls_config();
        let registry = Registry::new();
        let interceptor = MtlsInterceptor::new(tls_config, &registry).unwrap();

        let cert_info = ClientCertInfo {
            common_name: "test-client".to_string(),
            subject_alt_names: vec![],
            serial_number: "12345".to_string(),
            issuer: "CN=Test CA".to_string(),
            not_before: "2025-01-01T00:00:00Z".to_string(),
            not_after: "2027-12-31T23:59:59Z".to_string(),
        };

        assert!(interceptor.validate_client_cert(&cert_info).is_ok());
    }

    #[test]
    fn test_validate_unallowed_cn() {
        let tls_config = create_test_tls_config_unallowed();
        let registry = Registry::new();
        let interceptor = MtlsInterceptor::new(tls_config, &registry).unwrap();

        let cert_info = ClientCertInfo {
            common_name: "malicious-client".to_string(),
            subject_alt_names: vec![],
            serial_number: "12345".to_string(),
            issuer: "CN=Test CA".to_string(),
            not_before: "2025-01-01T00:00:00Z".to_string(),
            not_after: "2027-12-31T23:59:59Z".to_string(),
        };

        let result = interceptor.validate_client_cert(&cert_info);
        assert!(result.is_err());
        assert_eq!(result.unwrap_err().code(), Code::PermissionDenied);
    }

    #[test]
    fn test_validate_expired_cert() {
        let tls_config = create_test_tls_config();
        let registry = Registry::new();
        let interceptor = MtlsInterceptor::new(tls_config, &registry).unwrap();

        let cert_info = ClientCertInfo {
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

    #[test]
    fn test_validate_future_cert() {
        let tls_config = create_test_tls_config();
        let registry = Registry::new();
        let interceptor = MtlsInterceptor::new(tls_config, &registry).unwrap();

        let cert_info = ClientCertInfo {
            common_name: "test-client".to_string(),
            subject_alt_names: vec![],
            serial_number: "12345".to_string(),
            issuer: "CN=Test CA".to_string(),
            not_before: "2027-01-01T00:00:00Z".to_string(),
            not_after: "2028-12-31T23:59:59Z".to_string(),
        };

        let result = interceptor.validate_client_cert(&cert_info);
        assert!(result.is_err());
        assert!(result.unwrap_err().message().contains("not yet valid"));
    }
}
