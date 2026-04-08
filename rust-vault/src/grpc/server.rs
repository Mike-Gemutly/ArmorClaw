use crate::config::VaultConfig;
use crate::error::VaultError;
use crate::grpc::middleware::MtlsInterceptor;
use prometheus::Registry;
use std::fs;
use std::os::unix::fs::PermissionsExt;
use std::path::PathBuf;
use std::pin::Pin;
use std::task::{Context, Poll};
use tokio::net::{UnixListener, UnixStream};
use tokio::sync::watch;

struct UnixListenerStream {
    listener: UnixListener,
}

impl UnixListenerStream {
    fn new(listener: UnixListener) -> Self {
        Self { listener }
    }
}

impl tokio_stream::Stream for UnixListenerStream {
    type Item = Result<UnixStream, std::io::Error>;

    fn poll_next(self: Pin<&mut Self>, cx: &mut Context<'_>) -> Poll<Option<Self::Item>> {
        match self.listener.poll_accept(cx) {
            Poll::Ready(Ok((stream, _))) => Poll::Ready(Some(Ok(stream))),
            Poll::Ready(Err(e)) => Poll::Ready(Some(Err(e))),
            Poll::Pending => Poll::Pending,
        }
    }
}

pub struct GrpcServer {
    config: VaultConfig,
    registry: Registry,
    shutdown_tx: Option<watch::Sender<bool>>,
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
            shutdown_tx: None,
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

    async fn bind_unix_socket(&self) -> Result<UnixListener, VaultError> {
        let socket_path = self.socket_path();

        if let Some(parent) = socket_path.parent() {
            fs::create_dir_all(parent)
                .map_err(|e| VaultError::Grpc(format!("Failed to create socket directory: {}", e)))?;
        }

        if socket_path.exists() {
            fs::remove_file(&socket_path)
                .map_err(|e| VaultError::Grpc(format!("Failed to remove existing socket: {}", e)))?;
        }

        let listener = UnixListener::bind(&socket_path)
            .map_err(|e| VaultError::Grpc(format!("Failed to bind Unix socket: {}", e)))?;

        let mut permissions = fs::metadata(&socket_path)
            .map_err(|e| VaultError::Grpc(format!("Failed to get socket metadata: {}", e)))?
            .permissions();

        permissions.set_mode(0o600);
        fs::set_permissions(&socket_path, permissions)
            .map_err(|e| VaultError::Grpc(format!("Failed to set socket permissions: {}", e)))?;

        tracing::info!(
            socket_path = %socket_path.display(),
            "gRPC server bound on Unix domain socket"
        );

        Ok(listener)
    }

    /// Backward-compatible synchronous start.
    ///
    /// Creates the Unix socket so callers relying on the old `start()` → `shutdown()`
    /// pattern still work. Does **not** serve gRPC traffic — use [`serve`] or
    /// [`start_serving`] for that.
    pub fn start(&mut self) -> Result<(), VaultError> {
        let rt = tokio::runtime::Runtime::new()
            .map_err(|e| VaultError::Grpc(format!("Failed to create runtime: {}", e)))?;

        rt.block_on(async {
            if self.config.use_tls {
                return Err(VaultError::Grpc("TLS server not yet implemented - use Unix socket".to_string()));
            }
            let _listener = self.bind_unix_socket().await?;
            Ok(())
        })
    }

    /// Serve gRPC requests on the Unix domain socket.
    ///
    /// The caller supplies a fully-built `tonic::transport::server::Router` with all
    /// services registered. Returns when gracefully shut down via [`shutdown`].
    pub async fn serve(
        &mut self,
        router: tonic::transport::server::Router,
    ) -> Result<(), VaultError> {
        if self.config.use_tls {
            return Err(VaultError::Grpc("TLS server not yet implemented - use Unix socket".to_string()));
        }

        let listener = self.bind_unix_socket().await?;
        let stream = UnixListenerStream::new(listener);

        let (shutdown_tx, mut shutdown_rx) = watch::channel(false);
        self.shutdown_tx = Some(shutdown_tx);

        tracing::info!("gRPC server is serving requests");

        let shutdown_signal = async move {
            let _ = shutdown_rx.changed().await;
        };

        router
            .serve_with_incoming_shutdown(stream, shutdown_signal)
            .await
            .map_err(|e| VaultError::Grpc(format!("Server error: {}", e)))
    }

    /// Blocking convenience wrapper around [`serve`].
    pub fn start_serving(
        &mut self,
        router: tonic::transport::server::Router,
    ) -> Result<(), VaultError> {
        let rt = tokio::runtime::Runtime::new()
            .map_err(|e| VaultError::Grpc(format!("Failed to create runtime: {}", e)))?;

        rt.block_on(async { self.serve(router).await })
    }

    pub fn shutdown(&mut self) -> Result<(), VaultError> {
        if let Some(tx) = self.shutdown_tx.take() {
            let _ = tx.send(true);
        }

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
