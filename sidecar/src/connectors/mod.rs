pub mod aws_s3;
pub mod sharepoint;
pub mod connector;

pub use aws_s3::{S3Connector, S3UploadRequest, S3UploadResult};
pub use sharepoint::{SharePointConnector, SharePointConfig, CloudConnector};
