pub mod aws_s3;
pub mod sharepoint;
pub mod connector;

pub use aws_s3::{S3Connector, S3UploadRequest, S3UploadResult, BlobChunk, BlobInfo, S3DeleteRequest, S3DownloadRequest, S3ListRequest, S3ListResult, S3DeleteResult};
pub use sharepoint::{SharePointConnector, SharePointConfig, CloudConnector};
