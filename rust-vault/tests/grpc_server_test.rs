use rust_vault::config::VaultConfig;
use rust_vault::grpc::server::GrpcServer;
use rust_vault::grpc::governance::governance::governance_server::{Governance, GovernanceServer};
use rust_vault::grpc::governance::governance::{IssueTokenRequest, IssueTokenResponse, ConsumeTokenRequest, ConsumeTokenResponse, ZeroizeRequest, ZeroizeResponse, SubscribeRequest, VaultEventStream};
use std::fs;
use std::os::unix::fs::{FileTypeExt, PermissionsExt};
use std::path::PathBuf;
use tokio::net::UnixStream;
use tonic::transport::Server;
use tonic::{Request, Response, Status};

struct NoopGovernance;

#[tonic::async_trait]
impl Governance for NoopGovernance {
    async fn issue_ephemeral_token(
        &self,
        _request: Request<IssueTokenRequest>,
    ) -> Result<Response<IssueTokenResponse>, Status> {
        Err(Status::unimplemented("not yet implemented"))
    }

    async fn consume_ephemeral_token(
        &self,
        _request: Request<ConsumeTokenRequest>,
    ) -> Result<Response<ConsumeTokenResponse>, Status> {
        Err(Status::unimplemented("not yet implemented"))
    }

    async fn zeroize_tool_secrets(
        &self,
        _request: Request<ZeroizeRequest>,
    ) -> Result<Response<ZeroizeResponse>, Status> {
        Err(Status::unimplemented("not yet implemented"))
    }

    type SubscribeEventsStream = std::pin::Pin<Box<dyn tokio_stream::Stream<Item = Result<VaultEventStream, Status>> + Send>>;

    async fn subscribe_events(
        &self,
        _request: Request<SubscribeRequest>,
    ) -> Result<Response<Self::SubscribeEventsStream>, Status> {
        Err(Status::unimplemented("not yet implemented"))
    }
}

#[test]
fn test_grpc_server_can_be_created() {
    let config = VaultConfig::default();
    let socket_path = config.keystore_socket_path.clone();

    let server = GrpcServer::new(config.clone());

    assert!(
        server.is_ok(),
        "Failed to create gRPC server: {:?}",
        server.err()
    );

    let server = server.unwrap();
    assert_eq!(server.socket_path(), socket_path);
}

#[test]
fn test_grpc_server_creates_unix_socket() {
    let mut config = VaultConfig::default();
    let socket_path = PathBuf::from("/tmp/test_keystore.sock");

    config.keystore_socket_path = socket_path.clone();

    let server_result = GrpcServer::new(config.clone());
    assert!(
        server_result.is_ok(),
        "Failed to create gRPC server: {:?}",
        server_result.err()
    );

    let mut server = server_result.unwrap();

    let start_result = server.start();
    assert!(
        start_result.is_ok(),
        "Failed to start gRPC server: {:?}",
        start_result.err()
    );

    assert!(
        socket_path.exists(),
        "Socket file should exist at path: {:?}",
        socket_path
    );

    let metadata = fs::metadata(&socket_path).expect("Failed to get socket metadata");
    assert!(
        metadata.file_type().is_socket(),
        "File should be a Unix socket"
    );

    let shutdown_result = server.shutdown();
    assert!(
        shutdown_result.is_ok(),
        "Failed to shutdown gRPC server: {:?}",
        shutdown_result.err()
    );

    assert!(
        !socket_path.exists(),
        "Socket file should be removed after shutdown"
    );
}

#[test]
fn test_grpc_server_socket_permissions_0600() {
    let mut config = VaultConfig::default();
    let socket_path = PathBuf::from("/tmp/test_permissions.sock");

    config.keystore_socket_path = socket_path.clone();

    let server_result = GrpcServer::new(config.clone());
    assert!(
        server_result.is_ok(),
        "Failed to create gRPC server: {:?}",
        server_result.err()
    );

    let mut server = server_result.unwrap();

    let start_result = server.start();
    assert!(
        start_result.is_ok(),
        "Failed to start gRPC server: {:?}",
        start_result.err()
    );

    let metadata = fs::metadata(&socket_path).expect("Failed to get socket metadata");
    let permissions = metadata.permissions();
    let mode = permissions.mode();

    assert_eq!(
        mode & 0o777,
        0o600,
        "Socket should have 0600 permissions, got: {:03o}",
        mode & 0o777
    );

    let shutdown_result = server.shutdown();
    assert!(
        shutdown_result.is_ok(),
        "Failed to shutdown gRPC server: {:?}",
        shutdown_result.err()
    );
}

#[test]
fn test_grpc_server_socket_cleanup_on_shutdown() {
    let mut config = VaultConfig::default();
    let socket_path = PathBuf::from("/tmp/test_cleanup.sock");

    config.keystore_socket_path = socket_path.clone();

    let server_result = GrpcServer::new(config.clone());
    assert!(
        server_result.is_ok(),
        "Failed to create gRPC server: {:?}",
        server_result.err()
    );

    let mut server = server_result.unwrap();

    let start_result = server.start();
    assert!(
        start_result.is_ok(),
        "Failed to start gRPC server: {:?}",
        start_result.err()
    );

    assert!(socket_path.exists(), "Socket should exist after start");

    let shutdown_result = server.shutdown();
    assert!(
        shutdown_result.is_ok(),
        "Failed to shutdown gRPC server: {:?}",
        shutdown_result.err()
    );

    assert!(
        !socket_path.exists(),
        "Socket should not exist after shutdown"
    );
}

#[tokio::test]
async fn test_server_serves_and_accepts_connections() {
    let mut config = VaultConfig::default();
    let socket_path = PathBuf::from("/tmp/test_serving.sock");

    let _ = fs::remove_file(&socket_path);

    config.keystore_socket_path = socket_path.clone();

    let mut server = GrpcServer::new(config.clone()).expect("Failed to create gRPC server");

    let server_handle = tokio::spawn(async move {
        let router = Server::builder().add_service(GovernanceServer::new(NoopGovernance));
        server.serve(router).await
    });

    for _ in 0..50 {
        if socket_path.exists() {
            break;
        }
        tokio::time::sleep(std::time::Duration::from_millis(10)).await;
    }
    assert!(socket_path.exists(), "Socket should exist while server is running");

    let metadata = fs::metadata(&socket_path).expect("Failed to get socket metadata");
    assert!(metadata.file_type().is_socket(), "File should be a Unix socket");
    let mode = metadata.permissions().mode();
    assert_eq!(mode & 0o777, 0o600, "Socket should have 0600 permissions, got: {:03o}", mode & 0o777);

    let connect_result = UnixStream::connect(&socket_path).await;
    assert!(connect_result.is_ok(), "Server should accept connections on the Unix socket");

    server_handle.abort();

    let _ = fs::remove_file(&socket_path);
}
