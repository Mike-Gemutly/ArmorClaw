use thiserror::Error;

pub type Result<T> = std::result::Result<T, SidecarError>;

#[derive(Debug, Error)]
pub enum SidecarError {
    #[error("Configuration error: {0}")]
    Config(#[from] String),
    
    #[error("Invalid request: {0}")]
    InvalidRequest(String),
    
    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),
    
    #[error("Storage operation failed: {0}")]
    StorageError(String),
    
    #[error("Document processing failed: {0}")]
    DocumentProcessingError(String),
    
    #[error("IO error: {0}")]
    Io(#[from] std::io::Error),
    
    #[error("gRPC error: {0}")]
    Grpc(#[from] tonic::transport::Error),
    
    #[error("Circuit breaker open: {0}")]
    CircuitBreakerOpen(String),
    
    #[error("Rate limit exceeded: {0}")]
    RateLimitExceeded(String),
}
