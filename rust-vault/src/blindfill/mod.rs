//! Vestigial BlindFill design artifact.
//!
//! This module has **zero production callers**. It is re-exported in lib.rs
//! for test coverage only. The active CDP interception layer is Jetski
//! (`jetski/internal/cdp/proxy.go`). This code is NOT wired into any gRPC
//! server handler. See: `.sisyphus/plans/status-review-validation.md`

pub mod cdp_interceptor;
pub mod integration;
pub mod placeholder;

#[doc(hidden)]
pub use cdp_interceptor::CdpInterceptor;
#[doc(hidden)]
pub use integration::BlindFillIntegrator;
#[doc(hidden)]
pub use placeholder::{parse_placeholders, replace_placeholders, Placeholder, PlaceholderParseError};
