# Deployment Skills & VPS Tests Review

**Review Date:** 2026-04-05  
**Reviewer:** Sisyphus  
**Scope:** Deployment skills (`.skills/`) and VPS test infrastructure (`tests/ssh/`)

---

## Executive Summary

✅ **Overall Status:** Well-structured and comprehensive  
⚠️ **Areas for Improvement:** Documentation gaps, test-to-skill alignment, error handling

The deployment skills infrastructure is solid with good cross-platform support and clear automation levels. The VPS test suite is comprehensive but could be better integrated with the deployment skills workflow.

---

## 1. Deployment Skills Review

### Skills Inventory

| Skill | Lines | Purpose | Status |
|-------|-------|---------|--------|
| **deploy.yaml** | 280 | Deploy ArmorClaw to VPS | ✅ Complete |
| **status.yaml** | 210 | Health checking | ✅ Complete |
| **cloudflare.yaml** | 339 | HTTPS setup | ✅ Complete |
| **provision.yaml** | 87 | Mobile provisioning | ✅ Complete |
| **TEMPLATE.yaml** | 36 | Schema template | ✅ Complete |

### Strengths

1. **Cross-Platform Support** ✅
   - All skills support: Linux, macOS, Windows (PowerShell, Git Bash), WSL
   - Proper OS detection logic in each skill
   - Platform-specific command handling

2. **Clear Automation Levels** ✅
   ```yaml
   - auto:     Execute immediately (health checks, OS detection)
   - confirm:  Ask user first (SSH connection, installer)
   - guide:    Show instructions (account creation, DNS setup)
   ```

3. **Structured Schema** ✅
   - Consistent parameter definitions
   - Required vs optional parameters clearly marked
   - Default values provided
   - Command templates with variable interpolation

4. **Documentation** ✅
   - Each skill has detailed SKILL.md
   - Examples for different deployment modes
   - Quick reference tables
   - Platform support matrices

### Weaknesses

1. **Error Handling** ⚠️
   ```yaml
   # Missing: rollback procedures when deployment fails
   # Missing: retry logic for transient failures
   # Missing: detailed error messages for common issues
   ```

2. **Validation Gaps** ⚠️
   - No pre-flight checks for VPS requirements (RAM, disk, Docker version)
   - No validation of domain ownership before deployment
   - No Cloudflare API token validation before use

3. **Skill Interdependencies** ⚠️
   - `deploy` → `status` → `provision` flow not explicit
   - No skill orchestration workflow defined
   - Manual intervention required between skills

4. **Testing Coverage** ⚠️
   - No unit tests for skill YAML validation
   - No integration tests for skill execution
   - Only one test file (`test-deployment-skills.sh`) checks structure

### Recommendations

#### High Priority

1. **Add Pre-flight Validation** to `deploy.yaml`:
   ```yaml
   - name: "validate_vps_requirements"
     automation: "auto"
     description: "Check VPS meets minimum requirements"
     command: |
       # Check RAM (2GB minimum)
       RAM=$(ssh ${ssh_user}@${vps_ip} "free -m | awk '/Mem:/{print \$2}'")
       if [ "$RAM" -lt 2048 ]; then
         echo "ERROR: VPS needs at least 2GB RAM (has ${RAM}MB)"
         exit 1
       fi
       
       # Check disk (10GB minimum)
       DISK=$(ssh ${ssh_user}@${vps_ip} "df -BG / | awk 'NR==2 {print \$4}' | tr -d 'G'")
       if [ "$DISK" -lt 10 ]; then
         echo "ERROR: VPS needs at least 10GB disk (has ${DISK}GB)"
         exit 1
       fi
   ```

2. **Add Rollback Procedures** to `deploy.yaml`:
   ```yaml
   - name: "rollback_on_failure"
     automation: "auto"
     description: "Rollback deployment if verification fails"
     on_failure: true
     command: |
       echo "Deployment failed, rolling back..."
       ssh ${ssh_user}@${vps_ip} "docker-compose -f /opt/armorclaw/docker-compose.yml down -v"
       ssh ${ssh_user}@${vps_ip} "rm -rf /opt/armorclaw"
       echo "Rollback complete. Please check logs and retry."
   ```

3. **Add Skill Orchestration** workflow:
   ```yaml
   # New file: .skills/workflow.yaml
   name: "full_deployment"
   description: "Complete deployment workflow"
   steps:
     - skill: "deploy"
       params: ["vps_ip", "domain", "mode"]
     - skill: "status"
       params: ["vps_ip", "domain"]
       on_success: continue
       on_failure: rollback
     - skill: "provision"
       params: ["vps_ip"]
       on_success: complete
   ```

#### Medium Priority

4. **Improve Error Messages**:
   ```yaml
   - name: "validate_ssh"
     automation: "auto"
     description: "Validate SSH connection with helpful errors"
     command: |
       if ! ssh -i "$SSH_KEY" -o ConnectTimeout=10 ${ssh_user}@${vps_ip} "echo ok" 2>&1; then
         echo "ERROR: SSH connection failed. Common causes:"
         echo "  1. VPS IP is incorrect (${vps_ip})"
         echo "  2. SSH key not authorized (add to ~/.ssh/authorized_keys on VPS)"
         echo "  3. Firewall blocking SSH (port 22)"
         echo "  4. SSH service not running on VPS"
         echo ""
         echo "Debug commands:"
         echo "  ping ${vps_ip}"
         echo "  nc -zv ${vps_ip} 22"
         echo "  ssh -vvv -i $SSH_KEY ${ssh_user}@${vps_ip}"
         exit 1
       fi
   ```

5. **Add Integration Tests** for skills:
   ```bash
   # New file: tests/integration/test-deploy-skill.sh
   #!/bin/bash
   # Test deploy skill execution
   
   # Test 1: Native mode deployment
   test_native_deploy() {
     ./skills/exec deploy.yaml vps_ip=192.168.1.100 mode=native
     assert_container_running "armorclaw-bridge"
     assert_socket_exists "/run/armorclaw/bridge.sock"
   }
   
   # Test 2: Sentinel mode deployment
   test_sentinel_deploy() {
     ./skills/exec deploy.yaml vps_ip=5.183.11.149 domain=test.example.com mode=sentinel
     assert_container_running "armorclaw-bridge"
     assert_https_accessible "https://test.example.com"
   }
   ```

6. **Document Skill Dependencies** in each SKILL.md:
   ```markdown
   ## Prerequisites
   
   - **deploy** skill must be run first
   - VPS must have Docker 24.0+ installed
   - Domain must point to VPS IP (for Sentinel mode)
   - Cloudflare API token with Zone.DNS permissions (for Cloudflare mode)
   ```

---

## 2. VPS Tests Review

### Test Suite Inventory

| Test File | Lines | Purpose | Coverage |
|-----------|-------|---------|----------|
| **run_all_tests.sh** | 304 | Test runner CLI | ✅ All categories |
| **test_connectivity.sh** | 350 | SSH validation | ✅ Comprehensive |
| **test_command_execution.sh** | 405 | Remote commands | ✅ Good |
| **test_container_health.sh** | 128 | Docker status | ⚠️ Basic |
| **test_api_endpoints.sh** | 439 | Bridge/Matrix APIs | ✅ Comprehensive |
| **test_integration.sh** | 542 | Cross-component | ✅ Excellent |
| **test_security.sh** | 579 | Hardening checks | ✅ Comprehensive |
| **test_deployment_modes.sh** | 213 | Mode detection | ✅ Good |
| **test_ssl_tls.sh** | 170 | Certificates | ✅ Good |
| **test_performance.sh** | 180 | Benchmarks | ⚠️ Basic |

**Total:** 3,410 lines across 10 test files

### Strengths

1. **Comprehensive Coverage** ✅
   - 10 test categories covering all aspects
   - CLI interface for running specific tests
   - JSON output support for CI/CD integration

2. **Good Test Organization** ✅
   ```bash
   # Clear test structure
   - connectivity: SSH validation, timeout, retry
   - command: Remote execution, exit codes, stderr
   - health: Container status, logs, resources
   - api: Bridge RPC, Matrix client
   - integration: Cross-component flows
   - security: Firewall, hardening, isolation
   - deployment: Mode detection
   - ssl: Certificates, HTTPS
   - performance: Benchmarks
   ```

3. **Evidence Collection** ✅
   ```bash
   EVIDENCE_DIR="$PROJECT_DIR/.sisyphus/evidence"
   mkdir -p "$EVIDENCE_DIR"
   ```

4. **Environment Validation** ✅
   ```bash
   if [ -f "$PROJECT_DIR/.env" ]; then
       source "$PROJECT_DIR/.env"
   else
       echo -e "${RED}Error: .env file not found${NC}"
       exit 2
   fi
   ```

### Weaknesses

1. **Test-to-Skill Alignment** ⚠️
   - Tests don't validate deployment skills execution
   - No tests for skill YAML structure validation
   - Missing tests for skill parameter validation

2. **Hardcoded Paths** ⚠️
   ```bash
   PROJECT_DIR="/home/mink/src/armorclaw-omo"  # Hardcoded!
   # Should be: PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
   ```

3. **Limited Performance Tests** ⚠️
   - Only 180 lines for performance testing
   - No baseline metrics defined
   - No regression detection

4. **No Test Isolation** ⚠️
   - Tests share state via `.env` file
   - No test cleanup procedures
   - Tests can interfere with each other

### Recommendations

#### High Priority

1. **Fix Hardcoded Paths** in all test files:
   ```bash
   # Before
   PROJECT_DIR="/home/mink/src/armorclaw-omo"
   
   # After
   SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
   PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
   ```

2. **Add Skill Validation Tests**:
   ```bash
   # New file: tests/ssh/test_deployment_skills.sh
   #!/bin/bash
   # Test deployment skills execution and validation
   
   test_deploy_skill_yaml() {
     # Validate YAML syntax
     yamllint .skills/deploy.yaml
     
     # Validate required parameters
     grep -q "vps_ip" .skills/deploy.yaml
     grep -q "ssh_user" .skills/deploy.yaml
     grep -q "ssh_key" .skills/deploy.yaml
   }
   
   test_deploy_skill_execution() {
     # Test skill can be parsed
     ./skills/parse deploy.yaml --validate
     
     # Test skill execution (dry-run)
     ./skills/exec deploy.yaml --dry-run vps_ip=test.local
   }
   ```

3. **Add Test Isolation**:
   ```bash
   setup_test_env() {
     export TEST_ENV=$(mktemp -d)
     cp .env "$TEST_ENV/.env"
     source "$TEST_ENV/.env"
   }
   
   teardown_test_env() {
     rm -rf "$TEST_ENV"
   }
   
   trap teardown_test_env EXIT
   ```

#### Medium Priority

4. **Expand Performance Tests**:
   ```bash
   # Add baseline metrics
   BASELINE_SSH_CONNECT_TIME=2.0  # seconds
   BASELINE_API_RESPONSE_TIME=0.5  # seconds
   BASELINE_CONTAINER_START_TIME=30  # seconds
   
   test_performance_regression() {
     ACTUAL_SSH_TIME=$(measure_ssh_connect_time)
     if (( $(echo "$ACTUAL_SSH_TIME > $BASELINE_SSH_CONNECT_TIME * 1.5" | bc -l) )); then
       echo "PERFORMANCE REGRESSION: SSH connect time degraded by 50%"
       exit 1
     fi
   }
   ```

5. **Add Test Coverage Reporting**:
   ```bash
   # New file: tests/ssh/coverage_report.sh
   #!/bin/bash
   # Generate test coverage report
   
   TOTAL_TESTS=0
   PASSED_TESTS=0
   FAILED_TESTS=0
   SKIPPED_TESTS=0
   
   generate_coverage_report() {
     echo "Test Coverage Report"
     echo "===================="
     echo "Total: $TOTAL_TESTS"
     echo "Passed: $PASSED_TESTS"
     echo "Failed: $FAILED_TESTS"
     echo "Skipped: $SKIPPED_TESTS"
     echo "Coverage: $(echo "scale=2; $PASSED_TESTS / $TOTAL_TESTS * 100" | bc)%"
   }
   ```

---

## 3. Integration Between Skills and Tests

### Current State

- ❌ No automated tests for skill execution
- ❌ No validation that tests match skill capabilities
- ❌ No CI/CD integration for skill testing
- ✅ Good separation of concerns (skills define what, tests verify how)

### Recommended Integration

```yaml
# New file: .skills/test.yaml
name: "test_deployment"
description: "Run VPS tests after deployment"
parameters:
  - name: "vps_ip"
    type: "string"
    required: true
  - name: "test_categories"
    type: "array"
    required: false
    default: ["connectivity", "health", "api"]
    
steps:
  - name: "run_tests"
    automation: "auto"
    description: "Execute VPS test suite"
    command: |
      cd tests/ssh
      ./run_all_tests.sh --${test_categories[@]} --output json
```

---

## 4. Action Items

### Immediate (This Sprint)

- [ ] Fix hardcoded paths in `tests/ssh/*.sh`
- [ ] Add pre-flight validation to `deploy.yaml`
- [ ] Add rollback procedures to `deploy.yaml`
- [ ] Create `test_deployment_skills.sh` for skill validation

### Short-term (Next Sprint)

- [ ] Add skill orchestration workflow (`workflow.yaml`)
- [ ] Expand performance tests with baselines
- [ ] Add test isolation and cleanup
- [ ] Document skill dependencies in SKILL.md files

### Long-term (Future)

- [ ] CI/CD integration for skill testing
- [ ] Automated test-to-skill alignment validation
- [ ] Performance regression detection
- [ ] Test coverage reporting

---

## 5. Metrics

### Skills Metrics

- **Total Skills:** 4 + 1 template
- **Total Lines:** 952 lines (YAML only)
- **Cross-Platform Support:** 5 platforms (Linux, macOS, Windows PS, Git Bash, WSL)
- **Documentation Coverage:** 100% (all skills have SKILL.md)

### Tests Metrics

- **Total Test Files:** 10
- **Total Lines:** 3,410 lines
- **Test Categories:** 10
- **Evidence Collection:** ✅ Yes
- **JSON Output:** ✅ Yes

### Coverage Gaps

- ❌ Skill execution tests (0%)
- ❌ Skill YAML validation tests (0%)
- ⚠️ Performance baselines (0%)
- ⚠️ Test isolation (0%)

---

## 6. Conclusion

The deployment skills and VPS tests are **well-structured and comprehensive**, but there are opportunities for improvement:

**Strengths:**
- Excellent cross-platform support
- Clear automation levels
- Comprehensive test coverage
- Good documentation

**Weaknesses:**
- Missing skill execution tests
- No rollback procedures
- Hardcoded paths in tests
- Limited performance baselines

**Priority:** Focus on adding skill execution tests and rollback procedures first, as these have the highest impact on reliability.

---

**Reviewed by:** Sisyphus  
**Date:** 2026-04-05  
**Next Review:** After implementing high-priority items
