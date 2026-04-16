#!/bin/bash
# e2e_backward_channel.sh — End-to-end backward channel test
# Simulates: Bridge sets STEP_CONFIG → entrypoint detects → step_runner writes result.json
set -euo pipefail
cd "$(dirname "$0")/.."

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

export PYTHONPATH="$(pwd)"
export STEP_CONFIG='{"handler":"echo","task":"e2e shell test"}'
export STATE_DIR="$TMPDIR"
export OPENAI_API_KEY=sk-test-dummy

python3 -c "
import sys, os, json
sys.path.insert(0, '.')
from openclaw.step_config import parse_step_config
from openclaw.step_runner import StepRunner
cfg = parse_step_config()
assert cfg is not None, 'parse_step_config returned None'
runner = StepRunner()
exit_code = runner.run(cfg)
assert exit_code == 0, f'exit_code was {exit_code}'
with open(os.path.join(os.environ['STATE_DIR'], 'result.json')) as f:
    data = json.load(f)
assert data['status'] == 'success', f'status was {data[\"status\"]}'
assert 'e2e shell test' in data['output'], f'output was {data[\"output\"]}'
assert isinstance(data['duration_ms'], int), f'duration_ms was {type(data[\"duration_ms\"])}'
print(f'E2E PASS: {json.dumps(data, indent=2)}')
"

echo "---"
echo "result.json contents:"
cat "$TMPDIR/result.json"
echo ""
echo "PASS: End-to-end backward channel test completed successfully"
