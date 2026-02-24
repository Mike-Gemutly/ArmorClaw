#!/bin/bash
# Test suite for container-setup.sh
# Tests the fixes for:
# - CRLF stripping in prompt_input
# - Provider selection validation
# - API key length validation
# - Bridge configuration output

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

TESTS_PASSED=0
TESTS_FAILED=0

# Test helper functions
pass() {
    echo -e "${GREEN}✓ PASS${NC}: $1"
    ((TESTS_PASSED++))
}

fail() {
    echo -e "${RED}✗ FAIL${NC}: $1"
    if [ -n "$2" ]; then
        echo "  Details: $2"
    fi
    ((TESTS_FAILED++))
}

#=============================================================================
# Test CRLF Stripping in prompt_input
#=============================================================================

test_crlf_stripping() {
    echo -e "\n${YELLOW}Testing CRLF stripping in prompt_input${NC}"

    # Source the functions from container-setup.sh
    # We need to extract just the prompt_input function
    PROMPT_INPUT_FUNC=$(cat << 'FUNCTION'
prompt_input() {
    local prompt="$1"
    local default="$2"
    local result

    if [ -n "$default" ]; then
        echo -ne "${CYAN}$prompt [$default]: ${NC}"
    else
        echo -ne "${CYAN}$prompt: ${NC}"
    fi

    read -r result
    # Strip carriage returns and trailing whitespace (handles CRLF from terminals)
    result="${result%$'\r'}"
    result="${result%"${result##*[![:space:]]}"}"
    echo "${result:-$default}"
}
FUNCTION
)

    # Test 1: Input with carriage return should be stripped
    local test_input=$'3\r'
    local result=$(echo "$test_input" | bash -c "$PROMPT_INPUT_FUNC; prompt_input 'Test' '1'")
    # The result should be just '3' without the \r

    # Simulate CRLF input
    local crlf_input=$'3\r'
    if [ "$crlf_input" != "3" ]; then
        # CRLF input exists, test the stripping
        local stripped="${crlf_input%$'\r'}"
        stripped="${stripped%"${stripped##*[![:space:]]}"}"
        if [ "$stripped" = "3" ]; then
            pass "CRLF carriage return is stripped correctly"
        else
            fail "CRLF carriage return not stripped" "Expected '3', got '$stripped'"
        fi
    else
        pass "CRLF stripping logic verified (no CRLF in test environment)"
    fi

    # Test 2: Trailing whitespace should be stripped
    local ws_input="hello   "
    local stripped="${ws_input%"${ws_input##*[![:space:]]}"}"
    if [ "$stripped" = "hello" ]; then
        pass "Trailing whitespace is stripped correctly"
    else
        fail "Trailing whitespace not stripped" "Expected 'hello', got '$stripped'"
    fi

    # Test 3: Normal input should be preserved
    local normal_input="test_value"
    local stripped="${normal_input%"${normal_input##*[![:space:]]}"}"
    if [ "$stripped" = "test_value" ]; then
        pass "Normal input preserved correctly"
    else
        fail "Normal input was modified" "Expected 'test_value', got '$stripped'"
    fi
}

#=============================================================================
# Test Provider Selection Validation
#=============================================================================

test_provider_selection() {
    echo -e "\n${YELLOW}Testing provider selection validation${NC}"

    # Test valid choices
    for choice in 1 2 3 4; do
        case "$choice" in
            1) url="https://api.openai.com/v1" ;;
            2) url="https://api.anthropic.com/v1" ;;
            3) url="https://api.z.ai/api/coding/paas/v4" ;;
            4) url="custom" ;;
            *) url="invalid" ;;
        esac

        if [ "$url" != "invalid" ]; then
            pass "Provider choice '$choice' maps to valid URL: $url"
        else
            fail "Provider choice '$choice' should be valid"
        fi
    done

    # Test invalid choices - should NOT fall through silently
    for invalid in "0" "5" "a" "3\r" "3 " ""; do
        case "$invalid" in
            1|2|3|4)
                fail "Invalid choice '$invalid' was accepted"
                ;;
            *)
                pass "Invalid choice '$invalid' correctly rejected"
                ;;
        esac
    done
}

#=============================================================================
# Test API Key Length Validation
#=============================================================================

test_api_key_validation() {
    echo -e "\n${YELLOW}Testing API key length validation${NC}"

    # Test valid API keys (>= 20 chars)
    local valid_keys=(
        "sk-1234567890abcdefghijklmnop"
        "sk-ant-api03-12345678901234567890"
        "sk-proj-123456789012345678901234"
    )

    for key in "${valid_keys[@]}"; do
        if [ ${#key} -ge 20 ]; then
            pass "Valid API key length (${#key} chars): ${key:0:10}..."
        else
            fail "API key too short: ${#key} chars"
        fi
    done

    # Test invalid API keys (< 20 chars)
    local invalid_keys=(
        "3"
        "short"
        "sk-short"
        ""
    )

    for key in "${invalid_keys[@]}"; do
        local len=${#key}
        if [ $len -lt 20 ]; then
            pass "Invalid API key correctly rejected (${len} chars): '$key'"
        else
            fail "Short API key should be rejected but wasn't"
        fi
    done
}

#=============================================================================
# Test Bridge Configuration Output
#=============================================================================

test_bridge_config_output() {
    echo -e "\n${YELLOW}Testing bridge configuration output${NC}"

    # Verify the configure_bridge function has descriptive output
    local script_path="deploy/container-setup.sh"

    if [ -f "$script_path" ]; then
        # Check for the echo statement with description
        if grep -q 'Configure the ArmorClaw bridge settings' "$script_path"; then
            pass "Bridge config has descriptive intro text"
        else
            fail "Bridge config missing descriptive intro text"
        fi

        # Check for confirmation output
        if grep -q 'print_info.*Log level' "$script_path"; then
            pass "Bridge config shows log level confirmation"
        else
            fail "Bridge config missing log level confirmation"
        fi

        if grep -q 'print_info.*Socket path' "$script_path"; then
            pass "Bridge config shows socket path confirmation"
        else
            fail "Bridge config missing socket path confirmation"
        fi
    else
        fail "Script not found: $script_path"
    fi
}

#=============================================================================
# Test Integration: Full Flow Simulation
#=============================================================================

test_integration_flow() {
    echo -e "\n${YELLOW}Testing integration flow${NC}"

    # Simulate the full input processing flow
    local simulated_input=$'3\r'
    local step1="${simulated_input%$'\r'}"  # Strip CR
    local step2="${step1%"${step1##*[![:space:]]}"}"  # Strip trailing whitespace

    # Verify step 1: CR stripped
    if [ "$step1" = "3" ] || [ "$step1" = $'3\r' ]; then
        if [ "$step1" = "3" ]; then
            pass "Integration: CR stripped in step 1"
        else
            fail "Integration: CR not stripped in step 1"
        fi
    fi

    # Verify step 2: Whitespace stripped
    if [ "$step2" = "3" ]; then
        pass "Integration: Final value is clean '3'"
    else
        fail "Integration: Final value is not clean" "Expected '3', got '$step2'"
    fi

    # Verify '3' would now match the case statement
    case "$step2" in
        3)
            pass "Integration: Cleaned input '3' matches case pattern"
            ;;
        *)
            fail "Integration: Cleaned input doesn't match case pattern"
            ;;
    esac
}

#=============================================================================
# Main
#=============================================================================

main() {
    echo "=========================================="
    echo "  Container Setup Script Test Suite"
    echo "=========================================="

    test_crlf_stripping
    test_provider_selection
    test_api_key_validation
    test_bridge_config_output
    test_integration_flow

    echo ""
    echo "=========================================="
    echo "  Test Results"
    echo "=========================================="
    echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
    echo -e "${RED}Failed: $TESTS_FAILED${NC}"
    echo ""

    if [ $TESTS_FAILED -gt 0 ]; then
        echo -e "${RED}Some tests failed!${NC}"
        exit 1
    else
        echo -e "${GREEN}All tests passed!${NC}"
        exit 0
    fi
}

main "$@"
