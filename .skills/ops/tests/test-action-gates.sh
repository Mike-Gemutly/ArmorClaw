#!/usr/bin/env bash
# test-action-gates.sh — Verify action gate patterns in ops.yaml steps
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
YAML_FILE="$(cd "$SCRIPT_DIR/../.." && pwd)/ops.yaml"
FAIL=0

pass() { echo "  PASS  $1"; }
fail() { echo "  FAIL  $1"; FAIL=1; }

echo "=== Action Gate Tests ==="
echo "File: $YAML_FILE"
echo ""

# --- 1. Each action has at least one step with a gate ---
echo "[1/3] Action gates for 6 actions"
actions="deploy health redeploy logs backup restore"
for action in $actions; do
    # Python sees the YAML command text literally.
    # In ops.yaml the gate text is: [[ "$action" != "deploy" ]]
    # After YAML parsing, Python holds: [[ "$action" != "deploy" ]]
    # We search for the literal substring containing the action name.
    found=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
target = sys.argv[2]
for s in d.get('steps', []):
    cmd = s.get('command', '')
    # Look for gate pattern: 'action' variable compared to target action name
    # Patterns: != \"ACTION\"  or  == \"ACTION\"
    if ('!= \"' + target + '\"') in cmd or ('== \"' + target + '\"') in cmd:
        if 'action' in cmd:
            print('yes')
            sys.exit(0)
print('no')
" "$YAML_FILE" "$action")
    if [[ "$found" == "yes" ]]; then
        pass "action '$action' has a gate"
    else
        fail "action '$action' has NO gate"
    fi
done

# --- 2. detect_platform and validate_ssh have NO action gate ---
echo "[2/3] detect_platform and validate_ssh run for ALL actions"
for step_name in detect_platform validate_ssh; do
    has_gate=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
for s in d.get('steps', []):
    if s['name'] == sys.argv[2]:
        cmd = s.get('command', '')
        # Gate = comparison of action variable to a literal action string
        if 'action' in cmd and ('!=\"' in cmd or '==\"' in cmd):
            print('yes')
        else:
            print('no')
        break
" "$YAML_FILE" "$step_name")
    if [[ "$has_gate" == "no" ]]; then
        pass "'$step_name' has NO action gate (runs for all)"
    else
        fail "'$step_name' has an action gate (should run for all)"
    fi
done

# --- 3. detect_topology skips for deploy ---
echo "[3/3] detect_topology skips for deploy"
topology_skips=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
for s in d.get('steps', []):
    if s['name'] == 'detect_topology':
        cmd = s.get('command', '')
        if '== \"deploy\"' in cmd and 'action' in cmd:
            print('yes')
        else:
            print('no')
        break
" "$YAML_FILE")
if [[ "$topology_skips" == "yes" ]]; then
    pass "detect_topology skips for deploy (== \"deploy\" gate)"
else
    fail "detect_topology does NOT skip for deploy"
fi

echo ""
if [[ $FAIL -eq 0 ]]; then
    echo "=== ALL GATE TESTS PASSED ==="
else
    echo "=== SOME GATE TESTS FAILED ==="
fi
exit $FAIL
