"""
result_writer — Atomic result.json writer for ArmorClaw backward channel.

Writes a ContainerStepResult to result.json inside the bind-mounted state
directory using an atomic write (temp file + os.rename) so the Bridge never
reads a partial file.

Schema must match bridge/pkg/secretary/result.go ContainerStepResult:
  status      string            required
  output      string            required
  data        map[string]any    omitempty
  error       string            omitempty
  duration_ms int64             required
"""

import json
import os


def build_result(status, output, error=None, data=None, duration_ms=0):
    """Build a result dict matching the ContainerStepResult schema."""
    result = {
        "status": status,
        "output": output,
        "duration_ms": duration_ms,
    }
    if error is not None:
        result["error"] = error
    if data is not None:
        result["data"] = data
    return result


def write_result(state_dir, result_dict):
    """Write result_dict as result.json in state_dir atomically.

    1. Serialize to JSON (indent=2)
    2. Write to result.json.tmp in state_dir
    3. os.rename() to result.json (atomic on same filesystem)

    Raises OSError/PermissionError if state_dir is not writable.
    """
    target = os.path.join(state_dir, "result.json")
    tmp = os.path.join(state_dir, "result.json.tmp")

    payload = json.dumps(result_dict, indent=2) + "\n"

    with open(tmp, "w") as f:
        f.write(payload)
        f.flush()
        os.fsync(f.fileno())

    os.rename(tmp, target)
    return True
