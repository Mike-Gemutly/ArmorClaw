use thiserror::Error;

pub type Result<T> = std::result::Result<T, SidecarError>;

#[derive(Debug, Error)]
pub enum SidecarError {
    #[error("Configuration error: {0}")]
    Config(String),

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

    #[error("HTTP error: {0}")]
    HttpError(String),

    #[error("API error: {0}")]
    ApiError(String),

    #[error("Access denied")]
    AccessDenied,

    #[error("No such bucket: {0}")]
    NoSuchBucket(String),

    #[error("No such key: {0}")]
    NoSuchKey(String),

    #[error("Authentication error: {0}")]
    AuthenticationError(String),

    #[error("Cloud storage error: {0}")]
    CloudStorageError(String),

    #[error("PII redaction error: {0}")]
    PiiRedactionError(String),

    #[error("Internal error: {0}")]
    InternalError(String),
}

impl From<String> for SidecarError {
    fn from(s: String) -> Self {
        SidecarError::Config(s)
    }
}

impl From<config::ConfigError> for SidecarError {
    fn from(e: config::ConfigError) -> Self {
        SidecarError::Config(e.to_string())
    }
}
