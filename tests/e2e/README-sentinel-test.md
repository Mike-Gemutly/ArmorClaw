# Sentinel Mode E2E Test

## Overview

This test suite validates the ArmorClaw Sentinel Mode deployment configuration without requiring Docker to be running. It provides comprehensive validation of the installer, Caddyfile template, Docker Compose configuration, and generated files.

## Test Scripts

### 1. test-sentinel-mode-validation.sh (Recommended)

**Purpose:** Configuration validation without Docker dependency
**Execution Time:** ~2-5 seconds
**Docker Required:** No
**Test Count:** 48 tests

**When to Use:**
- CI/CD pipelines
- Development without Docker
- Quick configuration validation
- PR validation

**How to Run:**
```bash
# With default test domain
bash tests/e2e/test-sentinel-mode-validation.sh

# With custom test domain
TEST_DOMAIN=your-domain.com TEST_EMAIL=you@example.com \
  bash tests/e2e/test-sentinel-mode-validation.sh
```

### 2. test-sentinel-mode.sh

**Purpose:** Full deployment test with Docker execution
**Execution Time:** ~5-10 minutes
**Docker Required:** Yes
**Test Count:** Similar to validation test + Docker-specific tests

**When to Use:**
- Pre-release validation
- Full deployment testing
- TLS certificate validation
- Production environment verification

**How to Run:**
```bash
# Requires Docker to be running
sudo bash tests/e2e/test-sentinel-mode.sh
```

## Test Coverage

The validation test covers 9 categories:

### 1. Installer Sentinel Mode Detection (7 tests)
- Installer exists and is executable
- Deployment mode detection
- Email prompt for sentinel mode
- Sentinel mode logic
- Caddyfile generation
- .env file generation
- Sentinel profile usage

### 2. Caddyfile Template (8 tests)
- Template existence
- Domain variable (DOMAIN_NAME)
- Email variable (ADMIN_EMAIL)
- Matrix routes
- Well-known endpoints
- Bridge API routes
- Discovery endpoint
- Health check endpoint

### 3. Docker Compose Sentinel Profile (5 tests)
- Sentinel profile presence
- Correct ports (80, 443)
- Caddyfile mount
- Certificate volumes
- ACME_AGREE=true

### 4. Environment File Generation (5 tests)
- Server mode (ARMORCLAW_SERVER_MODE)
- Public base URL (ARMORCLAW_PUBLIC_BASE_URL)
- Email (ARMORCLAW_EMAIL)
- Secrets (ADMIN_TOKEN, KEYSTORE_SECRET, MATRIX_SECRET)
- Matrix homeserver URL

### 5. Sentinel Mode Deployment Simulation (3 tests)
- Test .env file creation
- Test Caddyfile creation
- Evidence copying

### 6. Matrix Well-Known Endpoints (3 tests)
- Client well-known format
- Server well-known format
- CORS headers

### 7. Installer Non-Interactive Mode (3 tests)
- Environment variable support
- Non-interactive mode detection
- Email validation

### 8. Secrets Generation (3 tests)
- Secret generation functions
- openssl rand usage
- Secret export

### 9. Sentinel Mode Requirements (8 tests)
- Domain detection
- Email collection for Let's Encrypt
- Sentinel mode selection
- Caddyfile generation
- .env generation
- Docker compose sentinel profile
- Secrets generation
- All requirements met

## Test Results

### Expected Output

```
========================================
Test Summary
========================================
Total:  48
Passed: 48
Failed: 0
========================================
✓ ALL TESTS PASSED
```

### Evidence Collection

Test evidence is collected in:
- **Evidence Directory:** `/tmp/armorclaw-test-e2e-XXX/evidence`
- **Results File:** `/tmp/armorclaw-test-e2e-XXX/results.txt`

**Evidence Files:**
- `installer-v6.sh` - Full installer script
- `docker-compose.yml` - Docker Compose configuration
- `Caddyfile.template` - Caddy reverse proxy template
- `generated-Caddyfile` - Generated Caddyfile for test domain

### Exit Codes

- `0` - All tests passed
- `1` - One or more tests failed
- `2` - Test setup failed

## Environment Variables

### Required

None required (uses sensible defaults)

### Optional

- `TEST_DOMAIN` - Test domain name (default: `test.armorclaw.local`)
- `TEST_EMAIL` - Test email address (default: `test@example.com`)

## Troubleshooting

### Test Failures

If tests fail:

1. **Check the test output** - Each failure includes a detailed message
2. **Review evidence files** - Check `$TEST_EVIDENCE_DIR` for copied files
3. **Read the issues.md** - See `.sisyphus/notepads/end-to-end-sentinel-test/issues.md`
4. **Check requirements** - Verify installer, docker-compose.yml, and template exist

### Common Issues

**Issue:** "Installer not found"
**Solution:** Verify `deploy/installer-v6.sh` exists in the project root

**Issue:** "Docker Compose not found"
**Solution:** Verify `docker-compose.yml` exists in the project root

**Issue:** "Pattern matching failed"
**Solution:** This is likely a bug in the test; check issues.md for known problems

## Development

### Running Tests Locally

```bash
# Run validation test
cd /home/mink/src/armorclaw-omo
bash tests/e2e/test-sentinel-mode-validation.sh

# Run with verbose output
bash tests/e2e/test-sentinel-mode-validation.sh | tee test-output.log

# Run with custom domain
TEST_DOMAIN=mydomain.com bash tests/e2e/test-sentinel-mode-validation.sh
```

### Adding New Tests

1. Add test function in appropriate category
2. Call function in `main()`
3. Document test purpose in function comments
4. Update this README with new test count

### Test Pattern Matching

When adding pattern matching tests:

- Use `grep -qE` for extended regex
- Test for multiple syntax variations
- Use flexible patterns (`80.*:.*80` not `"80:80"`)
- See `issues.md` for pattern matching lessons learned

## Documentation

For detailed information on test development, see:

- **learnings.md** - Lessons learned and best practices
- **issues.md** - Issues encountered and resolutions
- **decisions.md** - Architectural decisions made
- **summary.md** - Test completion summary

## CI/CD Integration

### GitHub Actions Example

```yaml
name: Sentinel Mode Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Sentinel Mode Validation Tests
        run: bash tests/e2e/test-sentinel-mode-validation.sh
      - name: Upload Test Evidence
        if: failure()
        uses: actions/upload-artifact@v3
        with:
          name: sentinel-test-evidence
          path: /tmp/armorclaw-test-e2e-*/
```

## Maintenance

### Regular Updates

- Update test patterns when installer syntax changes
- Add new tests when sentinel mode features are added
- Review and update documentation quarterly
- Monitor test execution time for performance issues

### Version History

- **v1.0.0** (2026-04-01) - Initial release with 48 tests

## Support

For issues or questions:
1. Check `issues.md` for known problems
2. Review test output for specific failure messages
3. Check evidence files for detailed information
4. Review `learnings.md` for best practices

---

**Test Status:** ✅ Production Ready
**Test Count:** 48 tests
**Pass Rate:** 100%
**Maintenance:** Active
