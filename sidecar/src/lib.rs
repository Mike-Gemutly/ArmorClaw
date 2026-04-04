pub mod grpc;
pub mod security;
pub mod config;
pub mod error;

// Disabled modules - pending fixes
// pub mod connectors; // TODO: AWS SDK v2 API changes require 73 fixes
// pub mod document; // TODO: Implement document processing
// pub mod utils; // TODO: Implement utilities
// pub mod reliability; // TODO: Implement reliability wrappers

pub use config::SidecarConfig;
pub use error::{SidecarError, Result};
