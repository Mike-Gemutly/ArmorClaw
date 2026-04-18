# Document Processing Pipeline (Sidecar)

> Part of the [ArmorClaw System Documentation](armorclaw.md)

## Overview

The document processing pipeline handles file ingestion, text extraction, encryption, and split-storage for RAG across multiple codebases: a Rust sidecar (data plane), a Python MarkItDown sidecar (legacy Office format support), a Go gRPC client with 3-layer routing (control plane bridge), and a YARA content scanner. Together they form the secure document path from cloud storage to chunked, encrypted storage with provenance tracking.

**Not to be confused with `rust-vault/`.** The vault handles secrets and credential storage. The sidecar handles documents: extracting text, encrypting chunks, scanning for malware, and maintaining a provenance chain. They share no code.

The Rust sidecar is a high-performance data plane component. It does the heavy lifting: cloud storage I/O, document parsing, AEAD encryption, and chunking. The Go Bridge is the control plane that owns security decisions, audit logging, PII interception, and request queuing. They communicate over a Unix domain socket via gRPC.

For the full sidecar API reference, compilation status, test coverage, and deployment instructions, see [sidecar/README.md](../sidecar/README.md). This document covers the pipeline as a whole, including the Go and YARA components that the sidecar README does not address.

## Architecture

```
                        Go Bridge (Control Plane)
                       ┌────────────────────────────────────────┐
                       │                                        │
                       │  bridge/pkg/sidecar/                   │
                       │  ┌──────────┐  ┌──────────────────┐   │
Agent request ────────▶│  │ Client   │  │ PIIInterceptor   │   │
                       │  │ (gRPC)   │  │ (redact/reject)  │   │
                       │  └────┬─────┘  └────────┬─────────┘   │
                       │       │                  │              │
                       │  ┌────▼─────┐  ┌────────▼─────────┐   │
                       │  │ Queue    │  │ AuditClient      │   │
                       │  │ Manager  │  │ (audit.db log)   │   │
                       │  └────┬─────┘  └──────────────────┘   │
                       │       │                                │
                       │  ┌────▼─────┐                          │
                       │  │ Token    │ HMAC-SHA256, 30 min TTL  │
                       │  │ Generator│                          │
                       │  └────┬─────┘                          │
                       └───────┼────────────────────────────────┘
                               │ gRPC over Unix Socket
                               │ (0600 permissions)
                       ┌───────▼────────────────────────────────┐
                       │  Rust Sidecar (Data Plane)              │
                       │  sidecar/                               │
                       │                                        │
                       │  ┌────────────┐  ┌───────────────┐     │
                       │  │ Connectors │  │ Document      │     │
                        │  │ S3, SP     │  │ PDF, DOCX,    │     │
                        │  │            │  │ XLSX, PPTX,   │     │
                        │  │            │  │ OCR           │     │
                       │  └──────┬─────┘  └───────┬───────┘     │
                       │         │                │             │
                       │  ┌──────▼────────────────▼───────┐     │
                       │  │ Split-Storage Manager          │     │
                       │  │ Encrypt chunks (XChaCha20)     │     │
                       │  │ Provenance signing (HMAC-SHA256)│     │
                       │  └────────────────────────────────┘     │
                       └────────────────────────────────────────┘

                       ┌────────────────────────────────────────┐
                       │  YARA Scanner (bridge/pkg/yara/)       │
                       │  Content disarm and reconstruction     │
                       │  Scans files before sidecar processing │
                       └────────────────────────────────────────┘
```

### Data Flow

1. Agent sends a document request to the Go Bridge.
2. The Bridge runs PII detection on the request payload. If PII is found and the interceptor is set to `redact`, the payload is scrubbed before forwarding. If set to `reject`, the request is denied.
3. The Bridge generates an ephemeral HMAC-SHA256 token (30 minute TTL) and attaches it as request metadata.
4. The YARA scanner (`bridge/pkg/yara/`) checks the file for known malware signatures before the sidecar touches it.
5. The request is forwarded to the Rust sidecar over a Unix domain socket via gRPC.
6. The sidecar extracts text, chunks it, encrypts each chunk with XChaCha20-Poly1305, and signs the result with a provenance signature.
7. Results flow back through the Bridge, which logs the full operation to its audit database.

If the sidecar is down, the Bridge's queue manager buffers requests and retries with exponential backoff.

## Key Packages

### Rust Sidecar (`sidecar/`)

The Rust sidecar is organized into the following modules. The library code is production-quality but requires `protoc` (Protocol Buffers compiler) to compile due to the gRPC service definition. The binary target has outstanding compilation errors and is not needed for library use.

#### Connectors (`sidecar/src/connectors/`)

Cloud storage adapters. Each connector implements upload, download, list, and delete operations with streaming support for large files (up to 5 GB).

| Connector | File | Status |
|-----------|------|--------|
| AWS S3 | `aws_s3.rs` | Functional |
| SharePoint | `sharepoint.rs` | Functional (Microsoft Graph API) |
| Azure Blob | `azure_blob.rs.disabled` | Disabled, needs rustls migration |

The `CloudConnector` trait in `connector.rs` defines the shared interface. The `SharePointConnector` is the reference implementation.

#### Document Processing (`sidecar/src/document/`)

Extracts text from common document formats. All extractors return structured results with page counts and metadata maps.

| Format | File | Notes |
|--------|------|-------|
| PDF | `pdf.rs` | Text extraction, split, merge |
| DOCX | `docx.rs` | Text extraction, paragraph insert/delete, find/replace |
| XLSX | `xlsx.rs` | Functional — calamine-based extraction with ShadowMap redaction |
| PPTX | `pptx.rs` | ZIP-based extraction using `zip` + `quick-xml` crates (v0.6.0) |
| OCR | `ocr.rs` | Functional — Tesseract subprocess + ONNX fallback, multi-language |
| Diff | `diff.rs` | Myers algorithm for text diff |
| HTML Diff | `html_diff.rs` | HTML-aware diff generation |
| DOCX Diff | `docx_diff.rs` | Stub, redline document generation |

Additional document modules:

| Module | File | Purpose |
|--------|------|---------|
| RAG Chunking | `rag.rs` | `TextChunker` with pluggable chunking strategies |
| Embeddings | `embeddings.rs` | `EmbeddingGenerator` trait, `OpenAIEmbedder` implementation |
| Qdrant | `qdrant.rs` | Implemented — create/upsert/search (needs qdrant-client-rs v1.7 builder migration) |

The `MAX_FILE_SIZE` constant (5 GB) caps all document operations.

#### Encryption (`sidecar/src/encryption/`)

`aead.rs` implements `AeadCipher`, which wraps XChaCha20-Poly1305 with deterministic nonce derivation. Nonces are derived via HMAC-SHA256 of `key_id || blob_id` plus a fixed message, ensuring the same blob always encrypts to the same ciphertext (idempotent encryption). The cipher key is zeroized on drop. Decryption returns plaintext wrapped in `Zeroizing<Vec<u8>>` to limit plaintext exposure in memory.

Wire format: `[version: 1 byte][nonce: 24 bytes][ciphertext + Poly1305 tag]`

#### Provenance (`sidecar/src/provenance/`)

`signer.rs` implements `ProvenanceSigner`, which produces truncated 8-byte HMAC-SHA256 signatures for lightweight provenance tracking. Verification uses constant-time comparison to prevent timing attacks. The formatted output looks like:

```
[Provenance: AC-v6-Sig:a1b2c3d4e5f6a1b2 | Sess:sess-123]
```

#### Split-Storage Manager (`sidecar/src/split_storage/`)

`manager.rs` ties encryption and chunking together. `SplitStorageManager` takes text chunks, encrypts them with the AEAD cipher, and wraps the result in an `EncryptedPayload` struct (base64-encoded ciphertext, version byte, clearance level). It supports decryption and clearance-based filtering, ensuring that retrieval only returns chunks the caller is authorized to see.

#### gRPC Service (`sidecar/src/grpc/`)

The server in `server.rs` implements the `SidecarService` trait defined in the proto. It routes gRPC calls to the appropriate connector or document module. Key RPCs:

| RPC | Purpose |
|-----|---------|
| `HealthCheck` | Returns status, uptime, version |
| `UploadBlob` | Upload to S3 via `destination_uri` (s3://bucket/key) |
| `DownloadBlob` | Server-streaming download, 1 MB chunks |
| `ListBlobs` | List objects with prefix filter |
| `DeleteBlob` | Delete an object |
| `ExtractText` | Extract text from PDF, DOCX, XLSX, or images (OCR) |
| `ProcessDocument` | General document processing (extract_text, convert) |

`interceptor.rs` implements `SecurityInterceptor`, which validates ephemeral tokens on every incoming request. The server binds to a Unix domain socket with `0600` permissions and handles SIGTERM/SIGINT for graceful shutdown.

The proto definition lives in `sidecar/src/grpc/proto/sidecar.proto` and mirrors `bridge/pkg/sidecar/sidecar.proto`.

### Go Client (`bridge/pkg/sidecar/`)

The Go client is the Bridge's interface to the sidecar. It provides a layered architecture: raw client, audit wrapper, and queuing system.

#### `client.go`

The core `Client` type manages a gRPC connection over a Unix domain socket. Key design decisions:

- **Retry with exponential backoff.** Every operation runs through `withRetry()`, which reconnects and retries up to 5 times with capped backoff (max 5 seconds).
- **PII interception.** When enabled, the `PIIInterceptor` scans request payloads before forwarding them to the sidecar.
- **Streaming downloads.** `DownloadBlob` collects chunks from the server stream and reassembles them into a single byte slice.
- **Configurable message sizes.** Default max is 256 MB for both send and receive.
- **Version negotiation.** gRPC interceptors attach `x-sidecar-version` metadata to every request.

#### `audit_client.go`

`AuditClient` wraps `Client` and logs every operation to the Bridge's audit database (`audit.db`). It records:

- Operation name and duration
- Success/failure status
- File sizes
- Request/user/agent/session IDs extracted from gRPC metadata
- Custom event types: `EventSidecarHealthCheck`, `EventSidecarUploadBlob`, `EventSidecarDownloadBlob`, `EventSidecarExtractText`, `EventSidecarProcessDocument`, `EventSidecarListBlobs`, `EventSidecarDeleteBlob`

It also provides `LogQueueEvent` and `LogRetryEvent` for when the sidecar is unavailable.

#### `pii_interceptor.go`

`PIIInterceptor` scans outgoing requests for personally identifiable information before they reach the sidecar. It supports two modes:

| Action | Behavior |
|--------|----------|
| `redact` | Scrubs PII from the request, forwards the cleaned version |
| `reject` | Returns an error, does not forward the request |

A `LogOnly` mode is available for monitoring without modifying requests. The interceptor uses `bridge/pkg/pii.Scrubber` for detection and handles `UploadBlobRequest`, `ExtractTextRequest`, and `ProcessDocumentRequest`. It skips binary content using a heuristic (90% printable ASCII threshold).

#### `queue.go`

`QueueManager` buffers requests when the sidecar is down. It runs a background goroutine that periodically health-checks the sidecar and drains the queue when it comes back up. Configuration:

| Parameter | Default |
|-----------|---------|
| Max queue size | 1000 |
| Max retry attempts | 5 |
| Initial backoff | 1s |
| Max backoff | 30s |
| Backoff multiplier | 2.0 |
| Health check interval | 10s |

`QueuedClient` wraps `Client` with automatic queuing on transient errors (unavailable, deadline exceeded, resource exhausted).

#### `token.go`

`TokenGenerator` creates and validates ephemeral tokens for sidecar authentication. Token format:

```
{request_id}:{timestamp}:{operation}:{hmac_sha256_signature}
```

Constants:

| Constant | Value |
|----------|-------|
| Token TTL | 30 minutes |
| Max timestamp age | 5 minutes |

The generator uses constant-time HMAC comparison to prevent timing attacks. Request IDs are generated with `crypto/rand` (16 bytes, hex-encoded).

#### `version.go`

Client version `1.0.0`, supported server range `1.0.0` through `1.5.0`. gRPC interceptors attach version metadata to every request for compatibility negotiation.

#### `sidecar.proto`

The Protocol Buffers service definition. Defines the `SidecarService` with 7 RPCs, request/response messages, and `RequestMetadata` for authentication. The same proto file is compiled into both Rust (tonic) and Go (protoc-gen-go) stubs.

### YARA Scanner (`bridge/pkg/yara/`)

#### `scanner.go`

The YARA scanner provides content disarm and reconstruction (CDR) for files entering the pipeline. It compiles YARA rules from a file at startup (`InitYARA`) and scans files against those rules (`ScanFileForMalware`). If any rule matches, the scan returns `false` (not clean) and logs the matching rule name and file path at `SECURITY` priority.

The scanner runs in the Go Bridge, before any request reaches the Rust sidecar. This keeps malicious content out of the data plane entirely.

Test data lives in `bridge/pkg/yara/testdata/`.

## Configuration

### Rust Sidecar Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `SIDECAR_SOCKET_PATH` | `/tmp/armorclaw-sidecar.sock` | Unix socket path |
| `SIDECAR_MAX_CONCURRENT_REQUESTS` | `1000` | Concurrency limit |
| `AWS_ACCESS_KEY_ID` | | S3 credential |
| `AWS_SECRET_ACCESS_KEY` | | S3 credential |
| `AWS_REGION` | `us-east-1` | S3 region |
| `SHAREPOINT_TENANT_ID` | | SharePoint Graph API |
| `SHAREPOINT_CLIENT_ID` | | SharePoint Graph API |
| `SHAREPOINT_CLIENT_SECRET` | | SharePoint Graph API |
| `SHAREPOINT_SITE_URL` | | SharePoint Graph API |
| `SHARED_SECRET` | | HMAC key for token validation |

### Go Client Configuration

| Field | Default | Purpose |
|-------|---------|---------|
| `SocketPath` | `/run/armorclaw/sidecar.sock` | Unix socket path |
| `Timeout` | 30s | Default operation timeout |
| `MaxRetries` | 5 | Retry attempts |
| `DialTimeout` | 10s | Connection timeout |
| `IdleTimeout` | 5m | Connection idle timeout |
| `MaxMsgSize` | 256 MB | gRPC message size limit |

### SidecarConfig Struct (Rust)

```rust
pub struct SidecarConfig {
    pub socket_path: PathBuf,
    pub max_concurrent_requests: usize,
    pub rate_limit_requests_per_second: usize,
    pub rate_limit_burst_capacity: usize,
    pub circuit_breaker_failure_threshold: usize,
    pub circuit_breaker_timeout_seconds: u64,
}
```

The `SidecarConfig` struct also carries the `shared_secret` field used by the security interceptor.

## Integration Points

### Bridge to Sidecar

The Go Bridge connects to the Rust sidecar via gRPC over a Unix domain socket. The socket is created with `0600` permissions, restricting access to the Bridge process. Every request carries `RequestMetadata` containing an ephemeral token, timestamp, and operation signature. The sidecar's `SecurityInterceptor` validates these before processing.

### YARA Integration

The YARA scanner runs inside the Bridge process, before sidecar calls. Files are scanned on disk. If a YARA rule matches, the file is flagged and the sidecar request is not made. Rules are loaded from a compiled YARA rules file at Bridge startup.

### Split-Storage RAG Pipeline

Documents flow through this pipeline:

1. Agent requests document processing via Matrix.
2. Bridge downloads the document (or receives it from the agent).
3. YARA scans the file for malware.
4. Bridge sends the document to the sidecar for text extraction.
5. Sidecar extracts text, chunks it via `TextChunker`.
6. Each chunk is encrypted with `AeadCipher` (XChaCha20-Poly1305).
7. `ProvenanceSigner` attaches a signature to the chunk metadata.
8. `SplitStorageManager` wraps chunks into `EncryptedPayload` structs with clearance levels.
9. Encrypted chunks are stored separately from their embeddings (split-storage pattern).
10. At query time, chunks are decrypted and filtered by clearance before being returned to the agent.

### Matrix / ArmorChat

Agents initiate document operations through Matrix rooms. The Bridge translates these into sidecar gRPC calls. The `AuditClient` logs every operation to `audit.db`, enabling compliance review through the ArmorChat admin interface.

### Jetski Browser Sidecar

Jetski (`jetski/`) is a separate component that handles browser automation via CDP. The document sidecar does not interact with Jetski directly. They share the same Bridge control plane but operate independently. Jetski handles web pages; the document sidecar handles files.

### Python MarkItDown Sidecar (`sidecar-python/`)

The Python sidecar extends the document pipeline with Microsoft Office legacy format support via the MarkItDown library. It handles formats that the Rust sidecar does not support natively: `.xlsx`, `.msg`, `.doc`, `.xls`, and `.ppt`. PPTX was migrated to the Rust sidecar in v0.6.0.

#### Architecture

```
                        Go Bridge (Control Plane)
                        ┌────────────────────────────────────────┐
                        │  bridge/pkg/sidecar/                   │
                        │  ┌──────────────────────────────────┐  │
                        │  │ RouteExtractText()               │  │
                        │  │ Layer 0: native text bypass      │  │
                        │  │ Layer 1: compound magic+format   │  │
                        │  │ Layer 2: strict drop on mismatch │  │
                        │  └──────────┬───────────────────────┘  │
                        │             │                           │
                        │     ┌───────┴────────┐                  │
                        │     ▼                ▼                  │
                        │  ┌────────┐   ┌──────────────┐         │
                         │  │ Rust   │   │ Python       │         │
                         │  │ Sidecar│   │ MarkItDown   │         │
                         │  │ (PDF,  │   │ Sidecar      │         │
                         │  │ DOCX,  │   │ (XLSX, MSG,  │         │
                         │  │ PPTX)  │   │  DOC, XLS,   │         │
                         │  └────────┘   │  PPT)        │         │
                        │               │  XLS, PPT)   │         │
                        │               └──────────────┘         │
                        └────────────────────────────────────────┘
```

#### Routing Logic (3-Layer)

| Layer | Condition | Action |
|-------|-----------|--------|
| **Layer 0** | `text/plain`, `text/csv`, `application/json`, `text/markdown` | Decode natively in Go — no sidecar call |
| **Layer 1** | ZIP magic + xlsx format → Python; OLE magic + xls/msg/doc/ppt format → Python; ZIP magic + pptx/docx/pdf → Rust | Route to appropriate sidecar based on compound magic byte + MIME type validation |
| **Layer 2** | Magic bytes don't match declared format (e.g., ZIP magic + msg format) | **Strict drop** — return `InvalidArgument` immediately |

#### Key Design Decisions

- **Compound validation**: The Go Bridge validates both the file's magic bytes (ZIP: `PK\x03\x04` or OLE: `\xd0\xcf\x11\xe0\xa1\xb1\x1a\xe1`) AND the declared MIME type before routing. Mismatches are rejected at the gateway.
- **No HTTP/FastAPI**: The Python sidecar uses `grpc.server()` exclusively — no HTTP endpoints exposed.
- **Threshold streaming**: Files under 10 MB are converted in-memory via `BytesIO`. Files over 10 MB are written to a temp file for conversion, then cleaned up.
- **TTL recycling**: The server exits gracefully after `MAX_REQUESTS` (default: 50) to enable container restart cycling.
- **Network isolation**: Container runs with `NetworkMode: none`, `cap_drop: ALL`, read-only root filesystem, and tmpfs for `/tmp/office_worker`.

#### Python Server (`sidecar-python/worker.py`)

| Feature | Implementation |
|---------|---------------|
| **gRPC Server** | Sync `grpc.server()` with `ThreadPoolExecutor` |
| **Format Mapping** | `FORMAT_MAP` — 6 MIME types → extensions |
| **Conversion** | MarkItDown library with `StreamInfo` for in-memory path |
| **Threshold** | `_THRESHOLD_BYTES = 10 * 1024 * 1024` (10 MB) |
| **TTL** | `MAX_REQUESTS = 50` before graceful shutdown |
| **Version** | `SERVER_VERSION = "1.0.0"` in `HealthCheck` response |
| **Socket** | `SIDECAR_SOCKET` env var (default: `/run/armorclaw/sidecar-office.sock`) |

#### Token Interceptor (`sidecar-python/interceptor.py`)

HMAC-SHA256 token validation using a sync `grpc.ServerInterceptor`. Tokens carry `{request_id}:{timestamp}:{hmac_signature}` format with configurable TTL. The interceptor was originally implemented as `grpc_aio.ServerInterceptor` (async), which was incompatible with the sync `grpc.server()` in `worker.py`. This has been fixed: `interceptor.py` now uses a sync interceptor that works correctly with the sync gRPC server.

#### Supported Formats

| Format | MIME Type | Magic Bytes | Extension | Converter |
|--------|-----------|-------------|-----------|-----------|
| Excel (modern) | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` | ZIP (PK) | `.xlsx` | MarkItDown `XlsxConverter` |
| PowerPoint (modern) | `application/vnd.openxmlformats-officedocument.presentationml.presentation` | ZIP (PK) | `.pptx` | Rust PPTX Extractor (pptx.rs) |
| Outlook Email | `application/vnd.ms-outlook` | OLE (D0CF) | `.msg` | MarkItDown `OutlookMsgConverter` |
| Word (legacy) | `application/msword` | OLE (D0CF) | `.doc` | Error — XlsConverter intercepts |
| Excel (legacy) | `application/vnd.ms-excel` | OLE (D0CF) | `.xls` | MarkItDown `XlsConverter` |
| PowerPoint (legacy) | `application/vnd.ms-powerpoint` | OLE (D0CF) | `.ppt` | Error — XlsConverter intercepts |

> **Limitation**: Legacy `.doc` and `.ppt` files produce conversion errors because MarkItDown's `XlsConverter` claims OLE files before the Word/PowerPoint converters. This is a known MarkItDown library limitation.

#### PPTX Migration to Rust (v0.6.0)

PPTX text extraction has been migrated from Python MarkItDown to the Rust sidecar:

- **Extractor**: `sidecar/src/document/pptx.rs` — ZIP-based extraction using `zip` + `quick-xml` crates
- **Routing**: `bridge/pkg/sidecar/office_client.go` — `.pptx` files route to Rust sidecar (Layer 1)
- **Format support**: Multi-slide presentations, speaker notes, embedded media metadata
- **Security**: Malformed archive protection, XML bomb mitigation, size limits
- The 3-layer routing architecture is preserved — only the PPTX destination changed from Python to Rust

#### Docker Deployment (`deploy/docker-compose.sidecar-py.yml`)

```yaml
# Container hardening
network_mode: none
cap_drop: [ALL]
read_only: true
security_opt: [no-new-privileges:true]
mem_limit: 512MB
tmpfs:
  - /tmp/office_worker:size=100M
```

#### Test Coverage

| Test File | Tests | Status |
|-----------|-------|--------|
| `sidecar-python/test_worker.py` | 27 | All pass |
| `sidecar-python/test_edge_cases.py` | 16 | All pass |
| `sidecar-python/test_interceptor.py` | 12 | All pass |
| `sidecar-python/test_docker_integration.py` | 10 | Skip when no Docker |
| `bridge/pkg/sidecar/office_client_e2e_test.go` | 7 | All pass |
| `bridge/pkg/sidecar/office_client_test.go` | 18 | All pass |
| **Total** | **90** | **0 regressions** |

#### Running Tests

```bash
# Python unit + integration tests
cd sidecar-python && python -m pytest test_worker.py test_edge_cases.py test_interceptor.py -v

# Go routing + E2E tests
cd bridge && go test -v -run "TestRouteExtractText|TestE2E" ./pkg/sidecar/

# Full regression (Python + Go)
cd sidecar-python && python -m pytest -v
cd bridge && go test ./pkg/sidecar/...
```

#### Go Client Routing (`bridge/pkg/sidecar/office_client.go`)

The `RouteExtractText()` function implements the 3-layer routing:

1. **Native text bypass**: Detects `text/*` MIME types and returns decoded content immediately without any gRPC call.
2. **Compound validation**: Reads first 8 bytes for magic bytes, cross-references with `document_format` MIME type. Routes ZIP-based xlsx to Python, ZIP-based docx/pdf/pptx to Rust.
3. **Strict drop**: If magic bytes contradict the declared format (e.g., OLE magic with xlsx MIME), returns `codes.InvalidArgument` without calling any sidecar.

## References

- [sidecar/README.md](../sidecar/README.md) - Full Rust sidecar documentation (API, testing, deployment, security audit)
- [armorclaw.md](armorclaw.md) - ArmorClaw system documentation index
- `.sisyphus/audits/SECURITY_AUDIT_TASK_49.md` - Security audit results
- `.sisyphus/plans/rust-office-sidecar.md` - Rust sidecar implementation plan
- `.sisyphus/plans/markitdown-sidecar.md` - Python MarkItDown sidecar implementation plan
- `.sisyphus/plans/markitdown-sidecar-testing.md` - Python sidecar testing plan
