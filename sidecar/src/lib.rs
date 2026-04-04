pub mod grpc; // Re-enabled for rate limiting testing
// pub mod connectors; // TODO: Fix compilation errors
// pub mod document; // TODO: Fix compilation errors
pub mod security; // TODO: Fix compilation errors
// pub mod utils; // TODO: Fix compilation errors
// pub mod reliability; // TODO: Fix compilation errors

pub mod config;
pub mod error;

pub use config::SidecarConfig;
pub use error::{SidecarError, Result};
