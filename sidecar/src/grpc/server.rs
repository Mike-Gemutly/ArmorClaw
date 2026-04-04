use crate::config::SidecarConfig;
use crate::error::Result;

pub async fn run_server(config: SidecarConfig) -> Result<()> {
    let addr = format!("unix://{}", config.socket_path.display());
    
    tracing::info!("Would start gRPC server on {}", addr);
    tracing::warn!("gRPC server disabled - proto generation requires protoc installation");
    
    Ok(())
}
