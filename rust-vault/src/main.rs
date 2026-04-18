use rust_vault::config::VaultConfig;
use rust_vault::grpc::governance::governance::governance_server::GovernanceServer;
use rust_vault::grpc::governance_service::VaultGovernanceService;
use rust_vault::grpc::server::GrpcServer;
use tonic::transport::Server;

fn main() {
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::try_from_default_env()
                .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new("info")),
        )
        .json()
        .init();

    let config = VaultConfig::default();
    let socket_path = config.keystore_socket_path.display().to_string();

    tracing::info!(socket_path = %socket_path, "Starting ArmorClaw Vault keystore");

    let mut grpc_server = GrpcServer::new(config).expect("Failed to create gRPC server");
    let router =
        Server::builder().add_service(GovernanceServer::new(VaultGovernanceService::new()));

    grpc_server
        .start_serving(router)
        .expect("Vault server failed");
}
