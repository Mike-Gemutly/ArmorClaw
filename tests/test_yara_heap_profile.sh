#!/usr/bin/env bash
set -euo pipefail

# test_yara_heap_profile.sh — Heap profiling test for YARA disk-based scanner
# Asserts that ScanFile (disk-based) does NOT load file content into heap.
#
# Requires: CGO_ENABLED=1, libyara-dev, go tool pprof

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BRIDGE_DIR="${PROJECT_ROOT}/bridge"
TESTDATA_DIR="${BRIDGE_DIR}/pkg/yara/testdata"
RULES_FILE="${BRIDGE_DIR}/configs/yara_rules.yar"
TMPDIR_BASE=""
HARNESS_DIR=""
FAILED=0

# ── Cleanup ──────────────────────────────────────────────────────────────
cleanup() {
    rm -rf "${HARNESS_DIR}" 2>/dev/null || true
    rm -rf "${TMPDIR_BASE}" 2>/dev/null || true
}
trap cleanup EXIT

# ── Pre-flight checks ────────────────────────────────────────────────────
skip_unless_available() {
    # CGO_ENABLED must be 1
    if [[ "${CGO_ENABLED:-}" == "0" ]]; then
        echo "SKIP: CGO_ENABLED=0, YARA requires CGO"
        exit 0
    fi

    # go tool must exist
    if ! command -v go &>/dev/null; then
        echo "SKIP: go not found in PATH"
        exit 0
    fi

    # libyara headers (CGO will fail without them)
    if ! pkg-config --exists yara 2>/dev/null; then
        echo "SKIP: libyara-dev not installed (pkg-config yara)"
        exit 0
    fi

    # Rules file must exist
    if [[ ! -f "${RULES_FILE}" ]]; then
        echo "SKIP: YARA rules file not found at ${RULES_FILE}"
        exit 0
    fi

    # go tool pprof
    if ! go tool pprof --help &>/dev/null; then
        echo "SKIP: go tool pprof not available"
        exit 0
    fi
}

# ── Build test harness ───────────────────────────────────────────────────
# We create a minimal Go main that imports the yara package and scans files
# passed as CLI arguments.  Heap profile is written via runtime/pprof.
build_harness() {
    HARNESS_DIR="$(mktemp -d "${TMPDIR:-/tmp}/yara-harness-XXXXXX")"

    cat > "${HARNESS_DIR}/main.go" << 'GOEOF'
package main

import (
	"fmt"
	"os"
	"runtime/pprof"
	"strconv"

	"github.com/armorclaw/bridge/pkg/yara"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: harness <ruleFile> <heapProfileOut> <file1> [file2...]")
		os.Exit(1)
	}

	ruleFile := os.Args[1]
	profileOut := os.Args[2]
	files := os.Args[3:]

	if err := yara.InitYARA(ruleFile); err != nil {
		fmt.Fprintf(os.Stderr, "InitYARA failed: %v\n", err)
		os.Exit(1)
	}

	f, err := os.Create(profileOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create profile: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		fmt.Fprintf(os.Stderr, "write heap profile: %v\n", err)
		os.Exit(1)
	}
	f.Close()

	for _, file := range files {
		_, err := yara.ScanFileForMalware(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "scan %s: %v\n", file, err)
			os.Exit(1)
		}
	}

	// Write post-scan heap profile (reuse same flag so pprof can diff,
	// but we just overwrite — we only care about the peak)
	f2, err := os.Create(profileOut)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create profile2: %v\n", err)
		os.Exit(1)
	}
	if err := pprof.WriteHeapProfile(f2); err != nil {
		fmt.Fprintf(os.Stderr, "write heap profile2: %v\n", err)
		os.Exit(1)
	}
	f2.Close()

	// Exit code 0 = all scans succeeded
	_ = strconv.Itoa(len(files))
}
GOEOF

    cd "${BRIDGE_DIR}"
    CGO_ENABLED=1 go build -o "${HARNESS_DIR}/yara-harness" "${HARNESS_DIR}/main.go"
}

# ── Parse heap profile ───────────────────────────────────────────────────
# Returns heap_inuse in KB from a heap profile using go tool pprof
extract_heap_kb() {
    local profile="$1"
    # go tool pprof -top -cum -sample_index=inuse_space outputs cumulative inuse
    # We grab the first data line which is the total
    local output
    output=$(go tool pprof -top -cum -sample_index=inuse_space "${profile}" 2>/dev/null \
        | head -5 \
        | grep -oP '^\s*[\d.]+\s+(MB|kB|GB)' \
        | head -1 || true)

    if [[ -z "$output" ]]; then
        echo "0"
        return
    fi

    # Extract number and unit
    local num unit
    num=$(echo "$output" | grep -oP '[\d.]+' | head -1)
    unit=$(echo "$output" | grep -oP '(MB|kB|GB)' | head -1)

    case "$unit" in
        GB) echo "$num" | awk '{printf "%.0f", $1 * 1048576}' ;;
        MB) echo "$num" | awk '{printf "%.0f", $1 * 1024}' ;;
        kB) echo "$num" | awk '{printf "%.0f", $1}' ;;
        *)  echo "0" ;;
    esac
}

# ── Assertion helper ─────────────────────────────────────────────────────
assert_heap_below() {
    local label="$1"
    local heap_kb="$2"
    local limit_kb="$3"

    if [[ "$heap_kb" -lt "$limit_kb" ]]; then
        echo "PASS: ${label} — heap ${heap_kb} KB < ${limit_kb} KB limit"
    else
        echo "FAIL: ${label} — heap ${heap_kb} KB >= ${limit_kb} KB limit"
        FAILED=1
    fi
}

# ── Generate test files ──────────────────────────────────────────────────
generate_test_files() {
    TMPDIR_BASE="$(mktemp -d "${TMPDIR:-/tmp}/yara-heap-test-XXXXXX")"
    local small_dir="${TMPDIR_BASE}/small"
    mkdir -p "${small_dir}"

    # 1000 small clean test files
    echo -n "Generating 1000 small test files... "
    local i
    for i in $(seq 1 1000); do
        printf "clean test content %06d\n" "$i" > "${small_dir}/file_${i}.txt"
    done
    echo "done"

    # One 100 MB sparse file
    echo -n "Creating 100 MB sparse file... "
    truncate -s 100M "${TMPDIR_BASE}/large_sparse.bin"
    echo "done"
}

# ── Main ─────────────────────────────────────────────────────────────────
main() {
    echo "=== YARA Heap Profile Test ==="
    echo ""

    skip_unless_available

    echo "Building test harness..."
    build_harness

    echo "Generating test files..."
    generate_test_files

    local harness="${HARNESS_DIR}/yara-harness"
    local heap_profile="${HARNESS_DIR}/heap.prof"
    local large_file="${TMPDIR_BASE}/large_sparse.bin"
    local small_dir="${TMPDIR_BASE}/small"

    # ── Test 1: Large single file scan ───────────────────────────────
    echo ""
    echo "--- Test 1: Large single file (100 MB sparse) ---"
    "${harness}" "${RULES_FILE}" "${heap_profile}" "${large_file}"

    local heap1
    heap1=$(extract_heap_kb "${heap_profile}")
    assert_heap_below "large file scan (100 MB sparse)" "$heap1" 5120  # 5 MB

    # ── Test 2: Batch 1000-file scan ─────────────────────────────────
    echo ""
    echo "--- Test 2: Batch scan (1000 small files) ---"

    # Build file list
    local file_list="${HARNESS_DIR}/filelist.txt"
    ls "${small_dir}"/*.txt > "${file_list}"

    # Read files into array
    readarray -t files < "${file_list}"

    "${harness}" "${RULES_FILE}" "${heap_profile}" "${files[@]}"

    local heap2
    heap2=$(extract_heap_kb "${heap_profile}")
    assert_heap_below "batch scan (1000 files)" "$heap2" 51200  # 50 MB

    # ── Summary ──────────────────────────────────────────────────────
    echo ""
    if [[ "$FAILED" -eq 0 ]]; then
        echo "=== ALL TESTS PASSED ==="
        exit 0
    else
        echo "=== SOME TESTS FAILED ==="
        exit 1
    fi
}

main "$@"
