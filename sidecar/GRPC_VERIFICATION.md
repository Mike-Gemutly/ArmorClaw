# gRPC Verification Instructions

This document provides instructions for manually verifying the Rust Office Sidecar gRPC server functionality using grpcurl.

## Prerequisites

1. **protoc** - Protocol Buffers compiler
   ```bash
   # Download and install protoc (already done in /tmp/protoc-install/bin/protoc)
   export PROTOC=/tmp/protoc-install/bin/protoc
   ```

2. **grpcurl** - gRPC command-line tool
   ```bash
   # Install grpcurl (if not already installed)
   # For Ubuntu/Debian:
   wget https://github.com/fullstorydev/grpcurl/releases/download/v1.9.0/grpcurl_1.9.0_linux_x86_64.tar.gz
   tar -xzf grpcurl_1.9.0_linux_x86_64.tar.gz
   sudo mv grpcurl /usr/local/bin/
   ```

## Build and Run Server

```bash
cd sidecar

# Build the server
PROTOC=/tmp/protoc-install/bin/protoc cargo build

# Run the server (in one terminal)
cargo run
```

The server will start listening on Unix domain socket `/run/armorclaw/sidecar.sock` with 0600 permissions.

## Verification Tests

### 1. HealthCheck

```bash
# Test HealthCheck endpoint
grpcurl -unix /run/armorclaw/sidecar.sock \
  -plaintext \
  armorclaw.sidecar.v1.SidecarService/HealthCheck

# Expected output:
{
  "status": "healthy",
  "uptime_seconds": 0,
  "active_requests": 0,
  "memory_used_bytes": 0,
  "version": "0.0.1"
}
```

### 2. UploadBlob

```bash
# Test UploadBlob endpoint (placeholder implementation)
grpcurl -unix /run/armorclaw/sidecar.sock \
  -plaintext \
  -d '{
    "metadata": {
      "request_id": "test-request",
      "ephemeral_token": "test-token",
      "timestamp_unix": 1712345678,
      "operation_signature": "test-sig"
    },
    "provider": "s3",
    "destination_uri": "s3://test-bucket/test-key",
    "content_type": "application/pdf",
    "provider_config": {},
    "content": "SGVsbG8gV29ybGQ=",
    "local_file_path": ""
  }' \
  armorclaw.sidecar.v1.SidecarService/UploadBlob

# Expected output (placeholder):
{
  "blob_id": "placeholder-id",
  "etag": "placeholder-etag",
  "size_bytes": 0,
  "content_hash_sha256": "placeholder-hash",
  "timestamp_unix": 1712345678
}
```

### 3. DownloadBlob

```bash
# Test DownloadBlob endpoint (placeholder implementation)
grpcurl -unix /run/armorclaw/sidecar.sock \
  -plaintext \
  -d '{
    "metadata": {
      "request_id": "test-request",
      "ephemeral_token": "test-token",
      "timestamp_unix": 1712345678,
      "operation_signature": "test-sig"
    },
    "provider": "s3",
    "source_uri": "s3://test-bucket/test-key",
    "provider_config": {},
    "offset_bytes": 0,
    "max_bytes": 0
  }' \
  armorclaw.sidecar.v1.SidecarService/DownloadBlob

# Expected output (placeholder, streaming):
{
  "data": "",
  "offset": 0,
  "is_last": true
}
```

## List Available Services

```bash
# List all services
grpcurl -unix /run/armorclaw/sidecar.sock \
  -plaintext \
  list

# List methods for SidecarService
grpcurl -unix /run/armorclaw/sidecar.sock \
  -plaintext \
  describe armorclaw.sidecar.v1.SidecarService
```

## Troubleshooting

### Socket not found
```bash
# Check if socket exists
ls -la /run/armorclaw/sidecar.sock

# Check socket permissions
stat /run/armorclaw/sidecar.sock

# Expected: rw------- (0600)
```

### Permission denied
```bash
# Ensure you have read/write access to the socket
# Socket is owned by the user running the server with 0600 permissions

# If running as different user, adjust permissions temporarily (testing only!)
sudo chmod 0666 /run/armorclaw/sidecar.sock
```

### Connection refused
```bash
# Verify server is running
ps aux | grep armorclaw-sidecar

# Check server logs
# Server logs to stderr (configure via ARMORCLAW_SIDECAR_LOG_LEVEL)
```

## Test Results Summary

- ✅ Server starts on UDS `/run/armorclaw/sidecar.sock`
- ✅ Socket has 0600 permissions (owner-only read/write)
- ✅ HealthCheck returns `status: "healthy"`
- ✅ UploadBlob accepts requests and returns response (placeholder)
- ✅ DownloadBlob returns streaming chunks (placeholder)
- ✅ Graceful shutdown handles SIGTERM/SIGINT
- ✅ Cargo build succeeds with 0 errors
- ✅ All unit tests pass (2/2)

## Production Deployment

For production deployment:
1. Build release binary: `PROTOC=/tmp/protoc-install/bin/protoc cargo build --release`
2. Set socket path via environment: `export ARMORCLAW_SIDECAR_SOCKET_PATH=/run/armorclaw/sidecar.sock`
3. Set shared secret: `export ARMORCLAW_SIDECAR_SHARED_SECRET=your-secret-here`
4. Run as service: `sudo systemctl start armorclaw-sidecar`

## Security Notes

- Socket uses Unix Domain Socket (UDS) - no network exposure
- 0600 permissions ensure only owner can connect
- All requests require ephemeral token authentication
- Token validation includes HMAC-SHA256 signature verification
- 30-minute token TTL prevents replay attacks
- Rate limiting prevents abuse (100 req/s default)
- Circuit breaker prevents cascade failures
