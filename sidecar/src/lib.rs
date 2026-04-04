pub mod grpc;
pub mod security;
pub mod config;
pub mod error;
pub mod reliability;
pub mod document;
pub mod utils;

// Disabled pending AWS SDK v2 migration (21 errors)
// pub mod connectors;

pub use config::SidecarConfig;
pub use error::{SidecarError, Result};
