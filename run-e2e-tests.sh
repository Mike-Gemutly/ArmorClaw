#!/bin/bash
# Test Coverage Verification Script
# Runs all E2E tests and generates coverage report

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
OUTPUT_DIR="$PROJECT_ROOT/.sisyphus/notepads/test-coverage-ops-hardening"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Test tracking
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Test files mapping
declare -A TEST_MAPPING=(
    ["test-installation.sh"]="US-1"
    ["test-api-key-setup.sh"]="US-2"
    ["test-admin-user.sh"]="US-3"
    ["test-calendar.sh"]="US-7"
    ["test-webdav.sh"]="US-8"
    ["test-contacts.sh"]="US-9"
)

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "========================================="
echo "E2E Test Coverage Verification"
echo "========================================="
echo ""

# Header for report
cat > "$OUTPUT_DIR/test-coverage-report.md" << 'EOF'
# E2E Test Coverage Report

Generated: $(date)

## Executive Summary

- **Total User Stories**: 11 (US-1 through US-11)
- **Tests Expected**: 8 (US-1, US-2, US-3, US-7, US-8, US-9, US-10, US-11)
- **Tests Found**: 6
- **Tests Missing**: 2 (US-10, US-11)

## Test Execution Results

EOF

# Run each test
echo "Running E2E tests..."
echo ""

for test_file in tests/e2e/test-*.sh; do
    if [[ "$test_file" == *"template.sh" ]] || [[ "$test_file" == *"common.sh" ]]; then
        continue
    fi

    TEST_NAME=$(basename "$test_file")
    US_LABEL="${TEST_MAPPING[$TEST_NAME]:-UNKNOWN}"

    echo "----------------------------------------"
    echo "Running: $TEST_NAME ($US_LABEL)"
    echo "----------------------------------------"

    TOTAL_TESTS=$((TOTAL_TESTS + 1))

    LOG_FILE="$OUTPUT_DIR/${TEST_NAME%.sh}-log.txt"

    # Run the test and capture output
    if bash "$test_file" > "$LOG_FILE" 2>&1; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        echo -e "${GREEN}✓ PASSED${NC}"
        STATUS="✅ PASSED"

        # Append to report
        cat >> "$OUTPUT_DIR/test-coverage-report.md" << EOF

### $TEST_NAME ($US_LABEL)
- **Status**: ✅ PASSED
- **Log**: [${TEST_NAME%.sh}-log.txt](${TEST_NAME%.sh}-log.txt)
EOF
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        echo -e "${RED}✗ FAILED${NC}"
        STATUS="❌ FAILED"

        # Append to report
        cat >> "$OUTPUT_DIR/test-coverage-report.md" << EOF

### $TEST_NAME ($US_LABEL)
- **Status**: ❌ FAILED
- **Log**: [${TEST_NAME%.sh}-log.txt](${TEST_NAME%.sh}-log.txt)
- **Error Summary**: $(tail -20 "$LOG_FILE" | grep -i "error\|fail\|exit" | head -3 || echo "Unknown error")
EOF
    fi

    echo ""
done

# Check for missing tests
echo "========================================="
echo "Missing Tests"
echo "========================================="

cat >> "$OUTPUT_DIR/test-coverage-report.md" << EOF

## Missing Tests

The following tests are expected but not found:

| US Story | Description | Expected Test File | Status |
|----------|-------------|-------------------|--------|
| US-10    | Mobile App Connection | test-mobile-connection.sh | ❌ MISSING |
| US-11    | Three-Way Consent     | test-three-way-consent.sh | ❌ MISSING |

EOF

echo "❌ US-10 (Mobile App Connection) - test-mobile-connection.sh"
echo "❌ US-11 (Three-Way Consent) - test-three-way-consent.sh"
echo ""

# Final summary
echo "========================================="
echo "Test Execution Summary"
echo "========================================="
echo "Total Tests Run: $TOTAL_TESTS"
echo -e "${GREEN}Passed: $PASSED_TESTS${NC}"
echo -e "${RED}Failed: $FAILED_TESTS${NC}"
echo ""

# Calculate coverage percentage
TESTED_USER_STORIES=6  # US-1,2,3,7,8,9
COVERAGE=$((TESTED_USER_STORIES * 100 / 8))  # 8 is total expected tests

cat >> "$OUTPUT_DIR/test-coverage-report.md" << EOF

## Coverage Summary

| Metric | Value |
|--------|-------|
| Total User Stories | 11 |
| Tests Expected | 8 (US-1,2,3,7,8,9,10,11) |
| Tests Found | 6 |
| Tests Missing | 2 (US-10, US-11) |
| Tests Run | $TOTAL_TESTS |
| Passed | $PASSED_TESTS |
| Failed | $FAILED_TESTS |
| Pass Rate | $((PASSED_TESTS * 100 / TOTAL_TESTS))% |
| Coverage Percentage | $COVERAGE% |

## Recommendations

1. **Immediate Actions**:
   - Create test for US-10 (Mobile App Connection)
   - Create test for US-11 (Three-Way Consent)

2. **Test Stability**:
   - Fix $FAILED_TESTS failing test(s)
   - Review error logs for common patterns

3. **CI Integration**:
   - Add all E2E tests to CI workflow
   - Ensure tests run on every PR

## Next Steps

- Implement missing tests (US-10, US-11) as per Wave 3
- Fix failing tests
- Update CI workflow to include all E2E tests

---

**Report Generated**: $(date)
**Execution Time**: $SECONDS seconds
EOF

echo "Test Coverage: $COVERAGE% (6 out of 8 expected tests)"
echo "Pass Rate: $((PASSED_TESTS * 100 / TOTAL_TESTS))%"
echo ""
echo "Full report saved to: $OUTPUT_DIR/test-coverage-report.md"
echo ""

# Exit with non-zero if any tests failed
if [[ $FAILED_TESTS -gt 0 ]]; then
    exit 1
fi

exit 0
