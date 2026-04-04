use rust_vault::config::VaultConfig;
use rust_vault::grpc::server::GrpcServer;
use std::fs;
use std::path::PathBuf;
use std::os::unix::fs::{FileTypeExt, PermissionsExt};

#[test]
fn test_grpc_server_can_be_created() {
    let config = VaultConfig::default();
    let socket_path = config.keystore_socket_path.clone();

    let server = GrpcServer::new(config.clone());

    assert!(server.is_ok(), "Failed to create gRPC server: {:?}", server.err());

    let server = server.unwrap();
    assert_eq!(server.socket_path(), socket_path);
}

#[test]
fn test_grpc_server_creates_unix_socket() {
    let mut config = VaultConfig::default();
    let socket_path = PathBuf::from("/tmp/test_keystore.sock");

    config.keystore_socket_path = socket_path.clone();

    let server_result = GrpcServer::new(config.clone());
    assert!(server_result.is_ok(), "Failed to create gRPC server: {:?}", server_result.err());

    let mut server = server_result.unwrap();

    let start_result = server.start();
    assert!(start_result.is_ok(), "Failed to start gRPC server: {:?}", start_result.err());

    assert!(socket_path.exists(), "Socket file should exist at path: {:?}", socket_path);

    let metadata = fs::metadata(&socket_path).expect("Failed to get socket metadata");
    assert!(metadata.file_type().is_socket(), "File should be a Unix socket");

    let shutdown_result = server.shutdown();
    assert!(shutdown_result.is_ok(), "Failed to shutdown gRPC server: {:?}", shutdown_result.err());

    assert!(!socket_path.exists(), "Socket file should be removed after shutdown");
}

#[test]
fn test_grpc_server_socket_permissions_0600() {
    let mut config = VaultConfig::default();
    let socket_path = PathBuf::from("/tmp/test_permissions.sock");

    config.keystore_socket_path = socket_path.clone();

    let server_result = GrpcServer::new(config.clone());
    assert!(server_result.is_ok(), "Failed to create gRPC server: {:?}", server_result.err());

    let mut server = server_result.unwrap();

    let start_result = server.start();
    assert!(start_result.is_ok(), "Failed to start gRPC server: {:?}", start_result.err());

    let metadata = fs::metadata(&socket_path).expect("Failed to get socket metadata");
    let permissions = metadata.permissions();
    let mode = permissions.mode();

    assert_eq!(mode & 0o777, 0o600, "Socket should have 0600 permissions, got: {:03o}", mode & 0o777);

    let shutdown_result = server.shutdown();
    assert!(shutdown_result.is_ok(), "Failed to shutdown gRPC server: {:?}", shutdown_result.err());
}

#[test]
fn test_grpc_server_socket_cleanup_on_shutdown() {
    let mut config = VaultConfig::default();
    let socket_path = PathBuf::from("/tmp/test_cleanup.sock");

    config.keystore_socket_path = socket_path.clone();

    let server_result = GrpcServer::new(config.clone());
    assert!(server_result.is_ok(), "Failed to create gRPC server: {:?}", server_result.err());

    let mut server = server_result.unwrap();

    let start_result = server.start();
    assert!(start_result.is_ok(), "Failed to start gRPC server: {:?}", start_result.err());

    assert!(socket_path.exists(), "Socket should exist after start");

    let shutdown_result = server.shutdown();
    assert!(shutdown_result.is_ok(), "Failed to shutdown gRPC server: {:?}", shutdown_result.err());

    assert!(!socket_path.exists(), "Socket should not exist after shutdown");
}

