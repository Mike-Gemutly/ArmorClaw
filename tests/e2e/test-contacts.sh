#!/bin/bash
# E2E Test for US-9 Contacts (Rolodex)
# Tests contact management including:
# - Creating contacts
# - Searching contacts
# - Updating contacts
# - Deleting contacts
# - Verifying encryption at rest

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
source "$SCRIPT_DIR/common.sh"

# Test data
TEST_CONTACT_NAME="E2E Test Contact"
TEST_CONTACT_COMPANY="Test Corp"
TEST_CONTACT_RELATIONSHIP="client"
TEST_CONTACT_PHONE="+1-555-0123"
TEST_CONTACT_EMAIL="test@example.com"
TEST_CONTACT_ADDRESS="123 Test St, Test City, TC 12345"
TEST_CONTACT_NOTES="This is a test contact for E2E testing"

# ============================================================================
# Test: Contacts - Create Contact
# ============================================================================

test_contacts_create() {
    echo ""
    echo "Test: Contacts - Create Contact"

    cd "$PROJECT_ROOT/bridge"

    go run ../tests/e2e/helpers/rolodex_tester.go create \
        --name "$TEST_CONTACT_NAME" \
        --company "$TEST_CONTACT_COMPANY" \
        --relationship "$TEST_CONTACT_RELATIONSHIP" \
        --phone "$TEST_CONTACT_PHONE" \
        --email "$TEST_CONTACT_EMAIL" \
        --address "$TEST_CONTACT_ADDRESS" \
        --notes "$TEST_CONTACT_NOTES" \
        --db "$TEST_DIR/rolodex.db" \
        --keystore "$TEST_DIR/keystore.db"

    if [[ $? -eq 0 ]]; then
        log_result "contacts_create" "true" "Contact created successfully"
        return 0
    else
        log_result "contacts_create" "false" "Failed to create contact"
        return 1
    fi
}

test_contacts_search() {
    echo ""
    echo "Test: Contacts - Search Contacts"

    cd "$PROJECT_ROOT/bridge"

    local result
    result=$(go run ../tests/e2e/helpers/rolodex_tester.go search \
        --query "$TEST_CONTACT_NAME" \
        --db "$TEST_DIR/rolodex.db" \
        --keystore "$TEST_DIR/keystore.db" 2>&1)

    if echo "$result" | grep -q "$TEST_CONTACT_NAME"; then
        log_result "contacts_search" "true" "Contact found in search results"
        return 0
    else
        log_result "contacts_search" "false" "Contact not found in search results"
        return 1
    fi
}

test_contacts_get() {
    echo ""
    echo "Test: Contacts - Get Contact"

    cd "$PROJECT_ROOT/bridge"

    local result
    result=$(go run ../tests/e2e/helpers/rolodex_tester.go get \
        --name "$TEST_CONTACT_NAME" \
        --db "$TEST_DIR/rolodex.db" \
        --keystore "$TEST_DIR/keystore.db" 2>&1)

    if echo "$result" | grep -q "$TEST_CONTACT_EMAIL" && \
       echo "$result" | grep -q "$TEST_CONTACT_PHONE"; then
        log_result "contacts_get" "true" "Contact retrieved with all fields"
        return 0
    else
        log_result "contacts_get" "false" "Contact retrieval incomplete: $result"
        return 1
    fi
}

test_contacts_update() {
    echo ""
    echo "Test: Contacts - Update Contact"

    cd "$PROJECT_ROOT/bridge"

    local updated_notes="Updated notes for E2E test"

    go run ../tests/e2e/helpers/rolodex_tester.go update \
        --name "$TEST_CONTACT_NAME" \
        --notes "$updated_notes" \
        --db "$TEST_DIR/rolodex.db" \
        --keystore "$TEST_DIR/keystore.db"

    if [[ $? -eq 0 ]]; then
        # Verify the update
        local result
        result=$(go run ../tests/e2e/helpers/rolodex_tester.go get \
            --name "$TEST_CONTACT_NAME" \
            --db "$TEST_DIR/rolodex.db" \
            --keystore "$TEST_DIR/keystore.db" 2>&1)

        if echo "$result" | grep -q "$updated_notes"; then
            log_result "contacts_update" "true" "Contact updated successfully"
            return 0
        else
            log_result "contacts_update" "false" "Contact update not reflected"
            return 1
        fi
    else
        log_result "contacts_update" "false" "Failed to update contact"
        return 1
    fi
}

test_contacts_encryption() {
    echo ""
    echo "Test: Contacts - Verify Encryption at Rest"

    cd "$PROJECT_ROOT/bridge"

    # Verify that sensitive data is encrypted in the database
    local result
    result=$(go run ../tests/e2e/helpers/rolodex_tester.go verify-encryption \
        --name "$TEST_CONTACT_NAME" \
        --db "$TEST_DIR/rolodex.db" \
        --keystore "$TEST_DIR/keystore.db" 2>&1)

    if echo "$result" | grep -q "encryption_verified.*true"; then
        log_result "contacts_encryption" "true" "Contact data is encrypted at rest"
        return 0
    else
        log_result "contacts_encryption" "false" "Contact data is not properly encrypted: $result"
        return 1
    fi
}

test_contacts_delete() {
    echo ""
    echo "Test: Contacts - Delete Contact"

    cd "$PROJECT_ROOT/bridge"

    go run ../tests/e2e/helpers/rolodex_tester.go delete \
        --name "$TEST_CONTACT_NAME" \
        --db "$TEST_DIR/rolodex.db" \
        --keystore "$TEST_DIR/keystore.db"

    if [[ $? -eq 0 ]]; then
        # Verify the deletion
        local result
        result=$(go run ../tests/e2e/helpers/rolodex_tester.go get \
            --name "$TEST_CONTACT_NAME" \
            --db "$TEST_DIR/rolodex.db" \
            --keystore "$TEST_DIR/keystore.db" 2>&1) || true

        if echo "$result" | grep -q "not found"; then
            log_result "contacts_delete" "true" "Contact deleted successfully"
            return 0
        else
            log_result "contacts_delete" "false" "Contact still exists after deletion"
            return 1
        fi
    else
        log_result "contacts_delete" "false" "Failed to delete contact"
        return 1
    fi
}

# ============================================================================
# Main Execution
# ============================================================================

main() {
    echo "========================================"
    echo "US-9 Contacts (Rolodex) E2E Test Suite"
    echo "========================================"
    echo ""
    echo "Testing Rolodex service with encryption"
    echo ""

    setup_test_env || exit 1

    check_dependencies || exit 1

    # Verify Go is available
    if ! command -v go &>/dev/null; then
        echo -e "${RED}✗ go not found${NC}"
        exit 1
    fi

    echo ""
    echo "Running contact tests..."
    echo ""

    test_contacts_create
    test_contacts_get
    test_contacts_search
    test_contacts_update
    test_contacts_encryption
    test_contacts_delete

    test_summary
    exit $?
}

main "$@"
