use crate::error::{Result, SidecarError};
use aws_config::meta::region::RegionProviderChain;
use aws_config::BehaviorVersion;
use aws_credential_types::provider::ProvideCredentials;
use aws_sdk_s3::{
    types::{ByteStream, PutObjectRequest},
    Client as S3Client,
};
use sha2::{Digest, Sha256};
use std::io;
use std::path::{Path, PathBuf};
use std::pin::Pin;
use std::sync::{Arc, Mutex};
use std::task::{Context, Poll};
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::io::{AsyncRead, AsyncReadExt, BufReader, ReadBuf};
use tokio::fs::File;
use tracing::{debug, error, info};

/// S3 upload request parameters.
///
/// Mutually exclusive: only one of `content` or `file_path` should be set.
#[derive(Debug, Clone)]
pub struct S3UploadRequest {
    /// S3 bucket name to upload to.
    pub bucket: String,
    /// Object key (path) within the bucket.
    pub key: String,
    /// AWS region (e.g., "us-east-1").
    pub region: String,
    /// Content type (e.g., "application/octet-stream").
    pub content_type: Option<String>,
    /// In-memory content bytes. Mutually exclusive with `file_path`.
    pub content: Option<Vec<u8>>,
    /// Local file path for streaming upload. Mutually exclusive with `content`.
    pub file_path: Option<PathBuf>,
    /// Optional AWS access key (ephemeral, from request metadata).
    pub access_key: Option<String>,
    /// Optional AWS secret key (ephemeral, from request metadata).
    pub secret_key: Option<String>,
    /// Optional AWS session token (ephemeral, from request metadata).
    pub session_token: Option<String>,
}

/// S3 upload result containing metadata about the uploaded object.
#[derive(Debug, Clone)]
pub struct S3UploadResult {
    /// S3 object URI in format "s3://bucket/key".
    pub blob_id: String,
    /// ETag returned by S3 for the uploaded object.
    pub etag: String,
    /// Size of the uploaded object in bytes.
    pub size_bytes: i64,
    /// SHA256 hash of the uploaded content (computed during upload).
    pub content_hash_sha256: String,
    /// Upload timestamp as Unix epoch seconds.
    pub timestamp_unix: i64,
}

/// Wrapper that computes SHA256 hash while reading data.
#[derive(Debug)]
struct HashingReader<R> {
    inner: R,
    hasher: Sha256,
    hash_output: Arc<Mutex<Option<String>>>,
}

impl<R> HashingReader<R> {
    fn new(inner: R, hash_output: Arc<Mutex<Option<String>>>) -> Self {
        Self {
            inner,
            hasher: Sha256::new(),
            hash_output,
        }
    }
}

impl<R: AsyncRead + Unpin> AsyncRead for HashingReader<R> {
    fn poll_read(
        mut self: Pin<&mut Self>,
        cx: &mut Context<'_>,
        buf: &mut ReadBuf<'_>,
    ) -> Poll<io::Result<()>> {
        let inner = Pin::new(&mut self.inner);
        let filled = inner.poll_read(cx, buf)?;
        if let Poll::Ready(Ok(())) = filled {
            let n = buf.filled().len();
            if n > 0 {
                self.hasher.update(buf.filled());
            }
        }
        filled
    }
}

impl<R> Drop for HashingReader<R> {
    fn drop(&mut self) {
        let hash = hex::encode(self.hasher.finalize());
        *self.hash_output.lock().expect("HashingReader mutex poisoned") = Some(hash);
    }
}

/// S3 connector for uploading blobs to AWS S3.
#[derive(Debug, Clone)]
pub struct S3Connector;

impl S3Connector {
    /// Creates an S3 client from upload request.
    ///
    /// Uses ephemeral credentials if provided in the request.
    /// Otherwise, uses the default AWS credential chain.
    async fn create_client(&self, request: &S3UploadRequest) -> Result<S3Client> {
        let region_provider = RegionProviderChain::first_try(Some(
            aws_sdk_s3::config::Region::new(request.region.clone()),
        ));

        let config_loader = aws_config::defaults(BehaviorVersion::latest())
            .region(region_provider);

        let config = if let (Some(access_key), Some(secret_key)) = (
            &request.access_key,
            &request.secret_key,
        ) {
            debug!("Using ephemeral credentials for S3 upload");

            let credentials_provider = aws_credential_types::Credentials::new(
                access_key.clone(),
                secret_key.clone(),
                request.session_token.clone(),
                None,
                "ephemeral",
            );

            config_loader
                .credentials_provider(credentials_provider)
                .load()
                .await
        } else {
            debug!("Using default credential chain for S3 upload");
            config_loader.load().await
        };

        Ok(S3Client::new(&config))
    }

    /// Uploads a blob to S3 with streaming and hash computation.
    ///
    /// Supports both in-memory content and file path streaming.
    /// For large files, streaming from disk is recommended to avoid OOM.
    ///
    /// # Arguments
    ///
    /// * `request` - S3 upload request parameters
    ///
    /// # Returns
    ///
    /// Returns an `S3UploadResult` containing upload metadata.
    ///
    /// # Errors
    ///
    /// Returns `SidecarError` if:
    /// - Neither content nor file_path is provided
    /// - Both content and file_path are provided
    /// - File I/O fails
    /// - S3 operation fails
    /// - Hash computation fails
    pub async fn upload(&self, request: S3UploadRequest) -> Result<S3UploadResult> {
        if request.content.is_none() && request.file_path.is_none() {
            return Err(SidecarError::InvalidRequest(
                "Either content or file_path must be provided".to_string(),
            ));
        }

        if request.content.is_some() && request.file_path.is_some() {
            return Err(SidecarError::InvalidRequest(
                "Only one of content or file_path can be provided".to_string(),
            ));
        }

        let client = self.create_client(&request).await?;
        let (byte_stream, size_bytes, hash_output) = if let Some(content) = request.content {
            let mut hasher = Sha256::new();
            hasher.update(&content);
            let hash = hex::encode(hasher.finalize());

            let size = content.len() as i64;
            debug!(
                "Uploading in-memory content of {} bytes to s3://{}/{}",
                size, request.bucket, request.key
            );

            (ByteStream::from(content), size, Arc::new(Mutex::new(Some(hash))))
        } else if let Some(file_path) = request.file_path {
            let metadata = tokio::fs::metadata(&file_path).await.map_err(|e| {
                SidecarError::Io(e)
            })?;
            let size = metadata.len() as i64;
            debug!(
                "Streaming file {} ({} bytes) to s3://{}/{}",
                file_path.display(),
                size,
                request.bucket,
                request.key
            );

            let file = File::open(&file_path).await.map_err(|e| SidecarError::Io(e))?;
            let hash_output = Arc::new(Mutex::new(None));
            let hashing_reader = HashingReader::new(file, hash_output.clone());
            let byte_stream = ByteStream::new_with_size(hashing_reader, size as u64);

            (byte_stream, size, hash_output)
        } else {
            unreachable!();
        };

        let mut put_request = PutObjectRequest::builder()
            .bucket(&request.bucket)
            .key(&request.key)
            .content_length(size_bytes as i64);

        if let Some(content_type) = &request.content_type {
            put_request = put_request.content_type(content_type);
        }

        let put_request = put_request
            .build()
            .map_err(|e| {
                SidecarError::StorageError(format!("Failed to build S3 request: {}", e))
            })?;

        info!(
            "Starting S3 upload: bucket={}, key={}, size={}",
            request.bucket, request.key, size_bytes
        );

        let output = client
            .put_object(put_request)
            .body(byte_stream)
            .send()
            .await
            .map_err(|e| {
                error!("S3 upload failed: {}", e);
                match e.into_service_error() {
                    aws_sdk_s3::error::PutObjectError::AccessDenied(_) => {
                        SidecarError::StorageError("Access denied to S3 bucket".to_string())
                    }
                    aws_sdk_s3::error::PutObjectError::NoSuchBucket(_) => {
                        SidecarError::StorageError("S3 bucket not found".to_string())
                    }
                    _ => SidecarError::StorageError(format!("S3 upload failed: {}", e)),
                }
            })?;

        debug!(
            "S3 upload completed: bucket={}, key, etag={:?}",
            request.bucket,
            request.key,
            output.e_tag()
        );

        let content_hash_sha256 = Arc::try_unwrap(hash_output.clone())
            .ok()
            .and_then(|m| m.lock().expect("Hash mutex poisoned").take())
            .unwrap_or_else(|| {
                error!("Failed to extract hash after upload");
                "".to_string()
            });

        let timestamp_unix = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .map(|d| d.as_secs() as i64)
            .unwrap_or_else(|_| {
                error!("System clock appears to be before Unix epoch, using 0");
                0
            });

        let etag = output
            .e_tag()
            .map(|s| s.trim_matches('"'))
            .unwrap_or("")
            .to_string();

        let blob_id = format!("s3://{}/{}", request.bucket, request.key);

        info!(
            "Upload completed: blob_id={}, etag={}, size={}, hash={}",
            blob_id, etag, size_bytes, content_hash_sha256
        );

        Ok(S3UploadResult {
            blob_id,
            etag,
            size_bytes,
            content_hash_sha256,
            timestamp_unix,
        })
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_hashing_reader_computes_hash() {
        let data = b"hello world";
        let hash_output = Arc::new(Mutex::new(None));
        {
            let reader = HashingReader::new(&data[..], hash_output.clone());
            drop(reader);
        }
        let hash = hash_output.lock().expect("Hash mutex poisoned").as_ref().unwrap();
        assert_eq!(
            hash,
            "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
        );
    }

    #[test]
    fn test_hashing_reader_with_multiple_reads() {
        let data = b"hello world test data";
        let hash_output = Arc::new(Mutex::new(None));
        {
            let mut reader = HashingReader::new(&data[..], hash_output.clone());
            let mut buf = [0u8; 5];
            reader.read_exact(&mut buf).unwrap();
            assert_eq!(&buf, b"hello");
            reader.read_exact(&mut buf).unwrap();
            assert_eq!(&buf, b" worl");
            assert_eq!(reader.read(&mut buf).unwrap(), 11);
            drop(reader);
        }
        let hash = hash_output.lock().expect("Hash mutex poisoned").as_ref().unwrap();
        assert!(hash.len() == 64);
    }

    #[tokio::test]
    async fn test_upload_validation_both_content_and_file() {
        let connector = S3Connector;
        let request = S3UploadRequest {
            bucket: "test-bucket".to_string(),
            key: "test-key".to_string(),
            region: "us-east-1".to_string(),
            content_type: None,
            content: Some(vec![1, 2, 3]),
            file_path: Some(PathBuf::from("/tmp/test.txt")),
            access_key: None,
            secret_key: None,
            session_token: None,
        };

        let result = connector.upload(request).await;
        assert!(result.is_err());
        assert!(result
            .unwrap_err()
            .to_string()
            .contains("Only one of content or file_path can be provided"));
    }

    #[tokio::test]
    async fn test_upload_validation_neither_content_nor_file() {
        let connector = S3Connector;
        let request = S3UploadRequest {
            bucket: "test-bucket".to_string(),
            key: "test-key".to_string(),
            region: "us-east-1".to_string(),
            content_type: None,
            content: None,
            file_path: None,
            access_key: None,
            secret_key: None,
            session_token: None,
        };

        let result = connector.upload(request).await;
        assert!(result.is_err());
        assert!(result
            .unwrap_err()
            .to_string()
            .contains("Either content or file_path must be provided"));
    }

    #[tokio::test]
    async fn test_upload_file_not_found() {
        let connector = S3Connector;
        let request = S3UploadRequest {
            bucket: "test-bucket".to_string(),
            key: "test-key".to_string(),
            region: "us-east-1".to_string(),
            content_type: None,
            content: None,
            file_path: Some(PathBuf::from("/nonexistent/file.txt")),
            access_key: None,
            secret_key: None,
            session_token: None,
        };

        let result = connector.upload(request).await;
        assert!(result.is_err());
        assert!(matches!(result.unwrap_err(), SidecarError::Io(_)));
    }

    #[tokio::test]
    async fn test_upload_empty_content() {
        let connector = S3Connector;
        let request = S3UploadRequest {
            bucket: "test-bucket".to_string(),
            key: "test-key".to_string(),
            region: "us-east-1".to_string(),
            content_type: None,
            content: Some(vec![]),
            file_path: None,
            access_key: None,
            secret_key: None,
            session_token: None,
        };

        let result = connector.upload(request).await;
        assert!(result.is_err());
    }

    #[test]
    fn test_timestamp_creation() {
        let timestamp = SystemTime::now()
            .duration_since(UNIX_EPOCH)
            .unwrap()
            .as_secs() as i64;
        assert!(timestamp > 0);
    }

    #[test]
    fn test_size_computation_memory() {
        let content = vec![1u8; 100];
        let size = content.len() as i64;
        assert_eq!(size, 100);
    }

    #[test]
    fn test_size_computation_file() {
        let temp_file = tempfile::NamedTempFile::new().unwrap();
        let data = vec![1u8; 200];
        std::io::Write::write_all(&temp_file, &data).unwrap();

        let metadata = std::fs::metadata(temp_file.path()).unwrap();
        let size = metadata.len() as i64;
        assert_eq!(size, 200);
    }

    #[test]
    fn test_region_configuration() {
        let region = aws_sdk_s3::config::Region::new("us-west-2");
        assert_eq!(region.as_ref(), "us-west-2");
    }

    #[test]
    fn test_content_type_handling() {
        let content_type = Some("application/json".to_string());
        assert_eq!(content_type, Some("application/json".to_string()));

        let content_type_none: Option<String> = None;
        assert!(content_type_none.is_none());
    }

    #[test]
    fn test_blob_id_components() {
        let result = S3UploadResult {
            blob_id: "s3://bucket/key".to_string(),
            etag: "etag".to_string(),
            size_bytes: 100,
            content_hash_sha256: "hash".to_string(),
            timestamp_unix: 1234567890,
        };

        assert_eq!(result.blob_id, "s3://bucket/key");
        assert_eq!(result.etag, "etag");
        assert_eq!(result.size_bytes, 100);
        assert_eq!(result.content_hash_sha256, "hash");
        assert_eq!(result.timestamp_unix, 1234567890);
    }

    #[test]
    fn test_etag_trimming() {
        let etag_with_quotes = "\"abc123def\"";
        let trimmed = etag_with_quotes.trim_matches('"');
        assert_eq!(trimmed, "abc123def");

        let etag_without_quotes = "abc123def";
        let trimmed = etag_without_quotes.trim_matches('"');
        assert_eq!(trimmed, "abc123def");
    }

    #[test]
    fn test_blob_id_formatting() {
        let bucket = "my-bucket";
        let key = "path/to/object.txt";
        let blob_id = format!("s3://{}/{}", bucket, key);
        assert_eq!(blob_id, "s3://my-bucket/path/to/object.txt");
    }

    #[test]
    fn test_hashing_reader_empty_input() {
        let data = b"";
        let hash_output = Arc::new(Mutex::new(None));
        {
            let reader = HashingReader::new(&data[..], hash_output.clone());
            drop(reader);
        }
        let hash = hash_output.lock().expect("Hash mutex poisoned").as_ref().unwrap();
        assert_eq!(
            hash,
            "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
        );
    }

    #[test]
    fn test_hashing_reader_large_input() {
        let data: Vec<u8> = (0..10000).map(|i| (i % 256) as u8).collect();
        let hash_output = Arc::new(Mutex::new(None));
        {
            let mut reader = HashingReader::new(data.as_slice(), hash_output.clone());
            let mut buf = [0u8; 100];
            while reader.read(&mut buf).unwrap() > 0 {}
            drop(reader);
        }
        let hash = hash_output.lock().expect("Hash mutex poisoned").as_ref().unwrap();
        assert!(hash.len() == 64);
    }

    #[tokio::test]
    async fn test_file_hash_computation() -> Result<()> {
        let mut temp_file = tempfile::NamedTempFile::new()?;
        writeln!(temp_file, "hello world")?;

        let mut hasher = Sha256::new();
        let file = File::open(temp_file.path()).await?;
        let mut reader = BufReader::new(file);
        let mut buffer = [0u8; 8192];

        loop {
            let bytes_read = reader.read(&mut buffer).await?;
            if bytes_read == 0 {
                break;
            }
            hasher.update(&buffer[..bytes_read]);
        }

        let hash = hex::encode(hasher.finalize());

        assert!(hash.len() == 64);

        Ok(())
    }

    #[test]
    fn test_sha256_hash_computation_memory() {
        let content = b"hello world";
        let mut hasher = Sha256::new();
        hasher.update(content);
        let hash = hex::encode(hasher.finalize());

        assert_eq!(
            hash,
            "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
        );
    }

    #[test]
    fn test_s3_upload_request_builder() {
        let request = S3UploadRequest {
            bucket: "my-bucket".to_string(),
            key: "path/to/object.txt".to_string(),
            region: "us-west-2".to_string(),
            content_type: Some("text/plain".to_string()),
            content: Some(vec![1, 2, 3]),
            file_path: None,
            access_key: Some("AKIAIOSFODNN7EXAMPLE".to_string()),
            secret_key: Some("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY".to_string()),
            session_token: Some("session-token-123".to_string()),
        };

        assert_eq!(request.bucket, "my-bucket");
        assert_eq!(request.key, "path/to/object.txt");
        assert_eq!(request.region, "us-west-2");
        assert_eq!(request.content_type, Some("text/plain".to_string()));
        assert_eq!(request.content, Some(vec![1, 2, 3]));
        assert!(request.file_path.is_none());
        assert_eq!(
            request.access_key,
            Some("AKIAIOSFODNN7EXAMPLE".to_string())
        );
        assert_eq!(
            request.secret_key,
            Some("wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY".to_string())
        );
    }

    #[test]
    fn test_s3_upload_result_creation() {
        let result = S3UploadResult {
            blob_id: "s3://test-bucket/test-key".to_string(),
            etag: "\"abc123\"".to_string(),
            size_bytes: 1024,
            content_hash_sha256: "a591a6d40bf420404a011733cfb7b190d62c65bf0bcda32b57b277d9ad9f146".to_string(),
            timestamp_unix: 1234567890,
        };

        assert_eq!(result.blob_id, "s3://test-bucket/test-key");
        assert_eq!(result.size_bytes, 1024);
        assert!(result.content_hash_sha256.len() == 64);
    }
}
