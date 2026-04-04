use std::path::PathBuf;

mod config;
mod error;
mod grpc;
mod connectors;
mod document;
mod security;
mod utils;
mod reliability;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    tracing_subscriber::fmt::init();
    
    let config = config::SidecarConfig::from_env()?;
    
    tracing::info!(
        socket_path = %config.socket_path.display(),
        max_concurrent_requests = %config.max_concurrent_requests,
        "Starting ArmorClaw Sidecar"
    );
    
    grpc::server::run_server(config).await?;
}
