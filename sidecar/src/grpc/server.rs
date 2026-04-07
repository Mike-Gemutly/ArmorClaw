use crate::config::SidecarConfig;
use crate::error::{Result, SidecarError};
use crate::grpc::interceptor::SecurityInterceptor;
use crate::grpc::proto::sidecar_service_server::{SidecarService, SidecarServiceServer};
use crate::grpc::proto::{
    HealthCheckRequest, HealthCheckResponse, UploadBlobRequest, UploadBlobResponse,
    DownloadBlobRequest, BlobChunk, ListBlobsRequest, ListBlobsResponse,
    DeleteBlobRequest, DeleteBlobResponse, ExtractTextRequest, ExtractTextResponse,
    ProcessDocumentRequest, ProcessDocumentResponse, sidecar_service_server::SidecarService as SidecarServiceTrait,
};
use prometheus::Registry;
use std::path::Path;
use tokio::net::UnixListener;
use tokio::signal;
use tokio_stream::wrappers::UnixListenerStream;
use tonic::{Request, Response, Status};
use tracing::{info, error, warn};

#[derive(Debug, Default)]
pub struct SidecarServiceImpl {}

type DownloadBlobStream = tokio_stream::wrappers::ReceiverStream<std::result::Result<BlobChunk, tonic::Status>>;

#[tonic::async_trait]
impl SidecarServiceTrait for SidecarServiceImpl {
    type DownloadBlobStream = DownloadBlobStream;
    async fn health_check(
        &self,
        _request: Request<HealthCheckRequest>,
    ) -> std::result::Result<Response<HealthCheckResponse>, Status> {
        info!("HealthCheck called");

        let response = HealthCheckResponse {
            status: "healthy".to_string(),
            uptime_seconds: 0,
            active_requests: 0,
            memory_used_bytes: 0,
            version: env!("CARGO_PKG_VERSION").to_string(),
        };

        Ok(Response::new(response))
    }

    async fn upload_blob(
        &self,
        _request: Request<UploadBlobRequest>,
    ) -> std::result::Result<Response<UploadBlobResponse>, Status> {
        warn!("UploadBlob called - placeholder implementation");

        let response = UploadBlobResponse {
            blob_id: "placeholder-id".to_string(),
            etag: "placeholder-etag".to_string(),
            size_bytes: 0,
            content_hash_sha256: "placeholder-hash".to_string(),
            timestamp_unix: chrono::Utc::now().timestamp(),
        };

        Ok(Response::new(response))
    }

    async fn download_blob(
        &self,
        _request: Request<DownloadBlobRequest>,
    ) -> std::result::Result<Response<DownloadBlobStream>, Status> {
        warn!("DownloadBlob called - placeholder implementation");

        let (tx, rx) = tokio::sync::mpsc::channel(4);

        let chunk = BlobChunk {
            data: vec![],
            offset: 0,
            is_last: true,
        };

        tokio::spawn(async move {
            let _ = tx.send(Ok(chunk)).await;
        });

        Ok(Response::new(DownloadBlobStream::new(rx)))
    }

    async fn list_blobs(
        &self,
        _request: Request<ListBlobsRequest>,
    ) -> std::result::Result<Response<ListBlobsResponse>, Status> {
        warn!("ListBlobs called - placeholder implementation");

        let response = ListBlobsResponse {
            blobs: vec![],
            continuation_token: String::new(),
        };

        Ok(Response::new(response))
    }

    async fn delete_blob(
        &self,
        _request: Request<DeleteBlobRequest>,
    ) -> std::result::Result<Response<DeleteBlobResponse>, Status> {
        warn!("DeleteBlob called - placeholder implementation");

        let response = DeleteBlobResponse {
            success: true,
            message: "placeholder".to_string(),
        };

        Ok(Response::new(response))
    }

    async fn extract_text(
        &self,
        _request: Request<ExtractTextRequest>,
    ) -> std::result::Result<Response<ExtractTextResponse>, Status> {
        warn!("ExtractText called - placeholder implementation");

        let response = ExtractTextResponse {
            text: String::new(),
            page_count: 0,
            metadata: std::collections::HashMap::new(),
        };

        Ok(Response::new(response))
    }

    async fn process_document(
        &self,
        _request: Request<ProcessDocumentRequest>,
    ) -> std::result::Result<Response<ProcessDocumentResponse>, Status> {
        warn!("ProcessDocument called - placeholder implementation");

        let response = ProcessDocumentResponse {
            output_uri: String::new(),
            output_content: vec![],
            output_format: String::new(),
            metadata: std::collections::HashMap::new(),
        };

        Ok(Response::new(response))
    }
}

pub async fn run_server(config: SidecarConfig) -> Result<()> {
    let socket_path = &config.socket_path;

    info!(
        socket_path = %socket_path.display(),
        "Starting gRPC server"
    );

    let registry = Registry::new();

    let shared_secret_bytes = config.shared_secret.as_bytes().to_vec();
    let security_interceptor = SecurityInterceptor::new(shared_secret_bytes, &registry)
        .map_err(|e| SidecarError::Config(format!("Failed to create security interceptor: {}", e)))?;

    info!("Security interceptor initialized with metrics");

    let impl_service = SidecarServiceImpl::default();

    let addr = format!("unix://{}", socket_path.display());

    let listener = UnixListener::bind(socket_path).map_err(|e| {
        if e.kind() == std::io::ErrorKind::AddrInUse {
            SidecarError::Config(format!(
                "Socket path {} already in use. Is another instance running?",
                socket_path.display()
            ))
        } else {
            SidecarError::Config(format!("Failed to bind to socket: {}", e))
        }
    })?;

    #[cfg(unix)]
    {
        use std::os::unix::fs::PermissionsExt;
        std::fs::set_permissions(socket_path, std::fs::Permissions::from_mode(0o600))
            .map_err(|e| SidecarError::Config(format!("Failed to set socket permissions: {}", e)))?;
    }

    info!(
        socket_path = %socket_path.display(),
        "Server listening on Unix socket with 0600 permissions"
    );

    let server = tonic::transport::Server::builder()
        .add_service(SidecarServiceServer::with_interceptor(
            impl_service,
            security_interceptor,
        ))
        .serve_with_incoming(UnixListenerStream::new(listener));

    let ctrl_c = async {
        signal::ctrl_c()
            .await
            .expect("Failed to install CTRL+C handler");
        info!("Received CTRL+C, shutting down gracefully");
    };

    #[cfg(unix)]
    let terminate = async {
        signal::unix::signal(signal::unix::SignalKind::terminate())
            .expect("Failed to install SIGTERM handler")
            .recv()
            .await;
        info!("Received SIGTERM, shutting down gracefully");
    };

    #[cfg(not(unix))]
    let terminate = std::future::pending::<()>();

    tokio::select! {
        _ = ctrl_c => {
            info!("Shutdown initiated via CTRL+C");
        }
        _ = terminate => {
            info!("Shutdown initiated via SIGTERM");
        }
        result = server => {
            if let Err(e) = result {
                error!("Server error: {}", e);
                return Err(SidecarError::Grpc(e));
            }
        }
    }

    info!("Server shutdown complete");

    if Path::new(socket_path).exists() {
        std::fs::remove_file(socket_path)
            .map_err(|e| error!("Failed to remove socket file: {}", e));
    }

    Ok(())
}
