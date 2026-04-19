#!/usr/bin/env bash
# validate.sh — Offline structural validation of ops.yaml
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
YAML_FILE="$(cd "$SCRIPT_DIR/../.." && pwd)/ops.yaml"
FAIL=0

pass() { echo "  PASS  $1"; }
fail() { echo "  FAIL  $1"; FAIL=1; }

echo "=== ops.yaml Offline Validation ==="
echo "File: $YAML_FILE"
echo ""

# --- 1. File exists and is valid YAML ---
echo "[1/13] YAML parse"
if python3 -c "import yaml,sys; yaml.safe_load(open(sys.argv[1]))" "$YAML_FILE" 2>/dev/null; then
    pass "ops.yaml parses as valid YAML"
else
    fail "ops.yaml does not parse as valid YAML"
    echo "  Cannot continue — aborting."
    exit 1
fi

# Helper: extract value via python3
pyval() {
    python3 -c "
import yaml, sys, json
d = yaml.safe_load(open(sys.argv[1]))
keys = sys.argv[2].split('.')
v = d
for k in keys:
    if isinstance(v, list):
        k = int(k)
    v = v[k]
print(json.dumps(v))
" "$YAML_FILE" "$1"
}

# --- 2. Required top-level keys ---
echo "[2/13] Required top-level keys"
for key in name version description parameters steps; do
    if python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
if '$key' not in d:
    sys.exit(1)
" "$YAML_FILE"; then
        pass "top-level key '$key' exists"
    else
        fail "missing top-level key '$key'"
    fi
done

# --- 3. name equals "armorclaw_ops" ---
echo "[3/13] Skill name"
name=$(pyval "name" | tr -d '"')
if [[ "$name" == "armorclaw_ops" ]]; then
    pass "name is 'armorclaw_ops'"
else
    fail "name is '$name', expected 'armorclaw_ops'"
fi

# --- 4. Version matches semver ---
echo "[4/13] Semver version"
version=$(pyval "version" | tr -d '"')
if [[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    pass "version '$version' matches semver"
else
    fail "version '$version' does not match semver (X.Y.Z)"
fi

# --- 5. All 10 required parameters ---
echo "[5/13] Required parameters (10)"
required_params="action vps_ip ssh_user ssh_key service mode verbose domain tail backup_path"
for p in $required_params; do
    found=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
names = [x['name'] for x in d.get('parameters', [])]
print('yes' if '$p' in names else 'no')
" "$YAML_FILE")
    if [[ "$found" == "yes" ]]; then
        pass "parameter '$p' defined"
    else
        fail "parameter '$p' missing"
    fi
done

# --- 6. Each parameter has required fields ---
echo "[6/13] Parameter fields (name, type, required, description, default)"
bad_params=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
required_fields = ['name','type','required','description','default']
missing = []
for p in d.get('parameters', []):
    for f in required_fields:
        if f not in p:
            missing.append(f\"{p.get('name','?')}.{f}\")
if missing:
    print(', '.join(missing))
" "$YAML_FILE")
if [[ -z "$bad_params" ]]; then
    pass "all parameters have required fields"
else
    fail "missing fields: $bad_params"
fi

# --- 7. All 10 required steps ---
echo "[7/13] Required steps (10)"
required_steps="detect_platform validate_ssh detect_topology deploy check_health redeploy view_logs create_backup restore_backup print_summary"
for s in $required_steps; do
    found=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
names = [x['name'] for x in d.get('steps', [])]
print('yes' if '$s' in names else 'no')
" "$YAML_FILE")
    if [[ "$found" == "yes" ]]; then
        pass "step '$s' defined"
    else
        fail "step '$s' missing"
    fi
done

# --- 8. Each step has required fields ---
echo "[8/13] Step fields (name, automation, description, command)"
bad_steps=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
required_fields = ['name','automation','description','command']
missing = []
for s in d.get('steps', []):
    for f in required_fields:
        if f not in s:
            missing.append(f\"{s.get('name','?')}.{f}\")
if missing:
    print(', '.join(missing))
" "$YAML_FILE")
if [[ -z "$bad_steps" ]]; then
    pass "all steps have required fields"
else
    fail "missing fields: $bad_steps"
fi

# --- 9. Valid automation levels ---
echo "[9/13] Automation levels (auto|confirm|guide)"
bad_auto=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
valid = {'auto','confirm','guide'}
invalid = []
for s in d.get('steps', []):
    a = s.get('automation','')
    if a not in valid:
        invalid.append(f\"{s['name']}={a}\")
if invalid:
    print(', '.join(invalid))
" "$YAML_FILE")
if [[ -z "$bad_auto" ]]; then
    pass "all automation levels valid"
else
    fail "invalid automation: $bad_auto"
fi

# --- 10. No `return 0` in commands ---
echo "[10/13] No 'return 0' in commands"
return0=$(python3 -c "
import yaml, sys, re
d = yaml.safe_load(open(sys.argv[1]))
found = []
for s in d.get('steps', []):
    cmd = s.get('command','')
    depth = 0
    for line in cmd.split('\n'):
        stripped = line.strip()
        if re.match(r'^\w+\s*\(\)', stripped):
            depth += 1
        if stripped == '}':
            depth = max(0, depth - 1)
        if depth == 0 and 'return 0' in stripped:
            found.append(s['name'])
            break
if found:
    print(', '.join(found))
" "$YAML_FILE")
if [[ -z "$return0" ]]; then
    pass "no 'return 0' in any command"
else
    fail "'return 0' found in: $return0"
fi

# --- 11. No `eval` in commands ---
echo "[11/13] No 'eval' in commands"
eval_found=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
found = []
for s in d.get('steps', []):
    cmd = s.get('command','')
    # Check for bare eval (not inside a string literal like a comment)
    import re
    if re.search(r'\beval\b', cmd):
        found.append(s['name'])
if found:
    print(', '.join(found))
" "$YAML_FILE")
if [[ -z "$eval_found" ]]; then
    pass "no 'eval' in any command"
else
    fail "'eval' found in: $eval_found"
fi

# --- 12. Platform detection covers all expected platforms ---
echo "[12/13] Platform detection block (5 platform names)"
expected_platforms="windows-gitbash wsl macos linux windows-powershell"
for plat in $expected_platforms; do
    found=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
# Check the detect_platform step command for the platform string
for s in d.get('steps', []):
    if s['name'] == 'detect_platform':
        if '$plat' in s.get('command',''):
            print('yes')
        else:
            print('no')
        break
" "$YAML_FILE")
    if [[ "$found" == "yes" ]]; then
        pass "detect_platform handles '$plat'"
    else
        fail "detect_platform missing '$plat'"
    fi
done

# --- 13. platforms list contains at least linux, macos ---
echo "[13/13] Platforms list (linux, macos)"
for plat in linux macos; do
    found=$(python3 -c "
import yaml, sys
d = yaml.safe_load(open(sys.argv[1]))
platforms = d.get('platforms', [])
print('yes' if '$plat' in platforms else 'no')
" "$YAML_FILE")
    if [[ "$found" == "yes" ]]; then
        pass "platform '$plat' in platforms list"
    else
        fail "platform '$plat' missing from platforms list"
    fi
done

echo ""
if [[ $FAIL -eq 0 ]]; then
    echo "=== ALL CHECKS PASSED ==="
else
    echo "=== SOME CHECKS FAILED ==="
fi
exit $FAIL
