use crate::blindfill::placeholder::PlaceholderParseError;
use thiserror::Error;

#[derive(Debug, Error)]
pub enum VaultError {
    #[error("Database error: {0}")]
    Database(String),

    #[error("gRPC error: {0}")]
    Grpc(String),

    #[error("CDP error: {0}")]
    Cdp(String),

    #[error("Configuration error: {0}")]
    Config(String),

    #[error("Placeholder not found: {0}")]
    PlaceholderNotFound(String),

    #[error("Invalid placeholder format: {0}")]
    InvalidPlaceholderFormat(String),

    #[error("Secret not found: {0}")]
    SecretNotFound(String),

    #[error("Rate limit exceeded")]
    RateLimitExceeded,

    #[error("Concurrency limit exceeded")]
    ConcurrencyLimitExceeded,

    #[error("Authentication failed: {0}")]
    AuthenticationFailed(String),
}

impl From<rusqlite::Error> for VaultError {
    fn from(err: rusqlite::Error) -> Self {
        VaultError::Database(err.to_string())
    }
}

impl From<tonic::Status> for VaultError {
    fn from(status: tonic::Status) -> Self {
        VaultError::Grpc(format!("gRPC error: {}", status))
    }
}

impl From<PlaceholderParseError> for VaultError {
    fn from(err: PlaceholderParseError) -> Self {
        match err {
            PlaceholderParseError::InvalidFormat(msg) => {
                VaultError::InvalidPlaceholderFormat(msg)
            }
            PlaceholderParseError::SecretNotFound(secret) => {
                VaultError::SecretNotFound(secret)
            }
            PlaceholderParseError::NestedPlaceholder(msg) |
            PlaceholderParseError::ConditionalNotSupported(msg) |
            PlaceholderParseError::LoopNotSupported(msg) => {
                VaultError::InvalidPlaceholderFormat(msg)
            }
        }
    }
}
