use crate::error::SidecarError;

mod config;
mod error;
mod grpc;
mod security;

#[tokio::main]
async fn main() -> Result<(), SidecarError> {
    tracing_subscriber::fmt::init();

    let config = config::SidecarConfig::from_env()?;

    if config.shared_secret.is_empty() {
        tracing::error!("ARMORCLAW_SIDECAR__SHARED_SECRET must be set");
        return Err(SidecarError::Config("shared_secret is required".to_string()));
    }

    tracing::info!(
        socket_path = %config.socket_path.display(),
        max_concurrent_requests = %config.max_concurrent_requests,
        "Starting ArmorClaw Sidecar"
    );

    grpc::server::run_server(config).await?;

    Ok(())
}
