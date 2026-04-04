use crate::config::VaultConfig;
use crate::error::VaultError;
use crate::grpc::middleware::MtlsInterceptor;
use prometheus::Registry;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use tokio::net::UnixListener;

pub struct GrpcServer {
    config: VaultConfig,
    registry: Registry,
}

impl GrpcServer {
    pub fn new(config: VaultConfig) -> Result<Self, VaultError> {
        if config.use_tls && config.tls.is_none() {
            return Err(VaultError::Config("TLS config required when use_tls is true".to_string()));
        }

        let registry = Registry::new();

        Ok(Self {
            config,
            registry,
        })
    }

    pub fn socket_path(&self) -> PathBuf {
        self.config.keystore_socket_path.clone()
    }

    pub fn mtls_interceptor(&self) -> Result<Option<MtlsInterceptor>, VaultError> {
        if !self.config.use_tls {
            return Ok(None);
        }

        let tls = self.config.tls.as_ref()
            .ok_or_else(|| VaultError::Config("TLS config required when use_tls is true".to_string()))?;

        let interceptor = MtlsInterceptor::new(tls.clone(), &self.registry)
            .map_err(|e| VaultError::Config(format!("Failed to create mTLS interceptor: {}", e)))?;

        Ok(Some(interceptor))
    }

    pub fn start(&mut self) -> Result<(), VaultError> {
        let rt = tokio::runtime::Runtime::new()
            .map_err(|e| VaultError::Grpc(format!("Failed to create runtime: {}", e)))?;

        rt.block_on(async {
            if self.config.use_tls {
                Err(VaultError::Grpc("TLS server not yet implemented - use Unix socket".to_string()))
            } else {
                self.start_unix_server().await
            }
        })
    }

    async fn start_unix_server(&mut self) -> Result<(), VaultError> {
        let socket_path = self.socket_path();

        if let Some(parent) = socket_path.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| VaultError::Grpc(format!("Failed to create socket directory: {}", e)))?;
        }

        if socket_path.exists() {
            fs::remove_file(&socket_path)
                .map_err(|e| VaultError::Grpc(format!("Failed to remove existing socket: {}", e)))?;
        }

        let _listener = UnixListener::bind(&socket_path)
            .map_err(|e| VaultError::Grpc(format!("Failed to bind Unix socket: {}", e)))?;

        let mut permissions = fs::metadata(&socket_path)
            .map_err(|e| VaultError::Grpc(format!("Failed to get socket metadata: {}", e)))?
            .permissions();

        permissions.set_mode(0o600);
        fs::set_permissions(&socket_path, permissions)
            .map_err(|e| VaultError::Grpc(format!("Failed to set socket permissions: {}", e)))?;

        tracing::info!(
            socket_path = %socket_path.display(),
            "gRPC server listening on Unix domain socket"
        );

        Ok(())
    }

    pub fn shutdown(&mut self) -> Result<(), VaultError> {
        let socket_path = self.socket_path();
        if socket_path.exists() {
            fs::remove_file(&socket_path)
                .map_err(|e| VaultError::Grpc(format!("Failed to remove socket: {}", e)))?;
        }
        Ok(())
    }

    pub fn registry(&self) -> &Registry {
        &self.registry
    }
}
