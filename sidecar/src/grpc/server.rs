use crate::config::SidecarConfig;
use crate::grpc::interceptor::SecurityInterceptor;
use crate::grpc::proto::sidecar::sidecar_service_server::{SidecarService, SidecarServiceServer};
use prometheus::{Registry, TextEncoder};
use std::net::SocketAddr;
use tokio::net::UnixListener;
use tokio::sync::broadcast;
use tonic::transport::{Server, ServerTlsConfig};
use tonic::{Request, Response, Status};
use tracing::{error, info};

#[derive(Debug)]
pub struct SidecarServiceImpl {
    shutdown_tx: broadcast::Sender<()>,
}

impl SidecarServiceImpl {
    pub fn new(shutdown_tx: broadcast::Sender<()>) -> Self {
        Self { shutdown_tx }
    }
}

#[tonic::async_trait]
impl SidecarService for SidecarServiceImpl {
    async fn health_check(
        &self,
        _request: Request<()>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }

    async fn upload_blob(
        &self,
        _request: Request<()>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }

    async fn download_blob(
        &self,
        _request: Request<()>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }

    async fn list_blobs(
        &self,
        _request: Request<()>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }

    async fn delete_blob(
        &self,
        _request: Request<()>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }

    async fn extract_text(
        &self,
        _request: Request<()>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }

    async fn process_document(
        &self,
        _request: Request<()>,
    ) -> Result<Response<()>, Status> {
        Ok(Response::new(()))
    }
}

pub async fn run_server(config: SidecarConfig) -> Result<(), Box<dyn std::error::Error>> {
    let (shutdown_tx, mut shutdown_rx) = broadcast::channel(1);

    let registry = Registry::new();
    let shared_secret_bytes = config.shared_secret.as_bytes().to_vec();

    let interceptor = SecurityInterceptor::new(shared_secret_bytes, &registry)?;

    let service = SidecarServiceImpl::new(shutdown_tx);
    let server = SidecarServiceServer::new(service)
        .interceptor(interceptor);

    let uds_path = config.socket_path.clone();

    tokio::fs::create_dir_all(
        uds_path
            .parent()
            .ok_or("Invalid socket path")?,
    )
    .await?;

    let listener = UnixListener::bind(&uds_path)?;

    info!(
        socket_path = %uds_path.display(),
        "gRPC server listening on Unix domain socket"
    );

    tokio::spawn(async move {
        if let Err(e) = Server::builder()
            .add_service(server)
            .serve_with_incoming(tokio_stream::wrappers::UnixListenerStream::new(listener))
            .await
        {
            error!("Server error: {}", e);
        }
    });

    tokio::select! {
        _ = shutdown_rx.recv() => {
            info!("Shutdown signal received");
        }
    };

    tokio::fs::remove_file(uds_path).await.ok();

    info!("Server shutdown complete");
    Ok(())
}
