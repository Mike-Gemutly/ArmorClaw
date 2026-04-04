# S3 Integration Tests

Comprehensive integration tests for S3 operations including upload, download, list, delete, circuit breaker, rate limiting, error scenarios, and metrics verification.

## Test Coverage

### Basic Operations
- ✅ Upload small file (<1MB) from memory
- ✅ Upload large file (>100MB) with streaming
- ✅ Upload with ephemeral credentials
- ✅ Upload with content-type
- ✅ Download with and without range requests
- ✅ List blobs with prefix filtering
- ✅ Delete blob

### Circuit Breaker
- ✅ Closed state on success
- ✅ Opens after failure threshold
- ✅ Blocks requests when open
- ✅ Transitions to half-open after timeout
- ✅ Closes on successful retry
- ✅ Fails in half-open state

### Rate Limiting
- ✅ Allows requests within limit
- ✅ Denies requests exceeding limit
- ✅ Resets after rate limit window
- ✅ Handles concurrent requests

### Error Scenarios
- ✅ File not found errors
- ✅ Empty content validation
- ✅ Invalid region format
- ✅ Both content and file_path provided (validation error)
- ✅ Neither content nor file_path provided (validation error)

### Metrics
- ✅ Circuit breaker calls recorded
- ✅ Circuit breaker state recorded
- ✅ Circuit breaker failures recorded
- ✅ Rate limit allowed recorded
- ✅ Rate limit denied recorded
- ✅ All S3 operation metrics collected

## Running Tests

### Prerequisites
- Rust 1.70 or later
- Cargo package manager

### Run All S3 Integration Tests

```bash
cd sidecar
cargo test --test s3_integration_test
```

### Run Specific Test Categories

```bash
# Basic operations only
cargo test --test s3_integration_test test_upload

# Circuit breaker tests only
cargo test --test s3_integration_test test_circuit_breaker

# Rate limiter tests only
cargo test --test s3_integration_test test_rate_limiter

# Error scenario tests only
cargo test --test s3_integration_test test_error_scenario

# Metrics tests only
cargo test --test s3_integration_test test_metrics

# Mock infrastructure tests only
cargo test --test s3_integration_test test_mock_s3_storage
```

### Run with Output

```bash
# Show test output
cargo test --test s3_integration_test -- --nocapture

# Show test output with timestamps
cargo test --test s3_integration_test -- --nocapture --test-threads=1
```

### Run Ignored Tests (Real S3)

Some tests require actual S3 credentials and are marked with `#[ignore]`:

```bash
# Run all tests including ignored ones
cargo test --test s3_integration_test -- --ignored

# Run only ignored tests
cargo test --test s3_integration_test -- --ignored --test-threads=1
```

### Run with Environment Variables (for Real S3 Tests)

```bash
export S3_TEST_BUCKET=your-test-bucket
export S3_TEST_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_SESSION_TOKEN=your-session-token  # optional

cargo test --test s3_integration_test -- --ignored
```

## Test Infrastructure

### Mock S3 Storage

Tests use an in-memory mock S3 implementation (`MockS3Storage`) to avoid requiring actual S3 credentials:

```rust
struct MockS3Storage {
    objects: Arc<RwLock<HashMap<(String, String), Vec<u8>>>>,
}
```

### Test Fixtures

`S3TestFixture` provides helper methods for creating test requests:

```rust
let fixture = S3TestFixture::new();
let upload_request = fixture.create_upload_request(data);
let download_request = fixture.create_download_request(key);
let list_request = fixture.create_list_request(prefix);
let delete_request = fixture.create_delete_request(key);
```

### Temporary Files

Tests use `tempfile` crate for temporary file creation:

```rust
let temp_file = create_temp_file(1024 * 1024); // 1 MB
```

## CI/CD Integration

### GitHub Actions

Tests can be integrated into CI/CD pipelines:

```yaml
- name: Run S3 Integration Tests
  working-directory: ./sidecar
  run: cargo test --test s3_integration_test
```

### Docker

Run tests in Docker container:

```bash
docker run --rm -v $PWD:/work -w /work/sidecar \
  rust:1.70 cargo test --test s3_integration_test
```

## Troubleshooting

### Test Timeouts

If tests timeout, increase the timeout:

```bash
cargo test --test s3_integration_test -- --test-timeout=300
```

### Test Failures

If tests fail, run with more verbose output:

```bash
RUST_BACKTRACE=1 cargo test --test s3_integration_test -- --nocapture
```

### Rate Limit Test Flakiness

Rate limiter tests use timing and may be flaky on slow systems. If they fail:

1. Run with single thread: `--test-threads=1`
2. Increase timeouts in test code
3. Check system time synchronization

## Adding New Tests

1. Add test function with `#[tokio::test]` attribute
2. Use test fixtures and helpers from existing tests
3. Follow naming convention: `test_<category>_<feature>`
4. Add to appropriate section in test file
5. Update this README with new test description

Example:

```rust
#[tokio::test]
async fn test_upload_with_custom_metadata() {
    let fixture = S3TestFixture::new();
    let connector = S3Connector::new();

    let content = b"test data".to_vec();
    let request = fixture.create_upload_request(content);

    // ... test implementation ...
}
```

## Mock vs Real S3

### Mock S3 (Default)
- ✅ No credentials required
- ✅ Fast execution
- ✅ Deterministic behavior
- ✅ CI/CD friendly
- ❌ Doesn't test actual AWS SDK integration

### Real S3 (with #[ignore] tests)
- ✅ Tests actual AWS SDK integration
- ✅ Validates against real S3 API
- ✅ Catches AWS SDK version issues
- ❌ Requires credentials
- ❌ Slower execution
- ❌ Costs money (minimal)

Recommendation: Use mock S3 for CI/CD, real S3 for periodic validation.

## See Also

- [S3 Connector Implementation](../src/connectors/aws_s3.rs)
- [Reliability Module](../src/reliability.rs)
- [Existing Integration Tests](./aws_s3_integration_test.rs)
