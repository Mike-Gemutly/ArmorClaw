"""
step_config — Parse the STEP_CONFIG environment variable for step execution mode.

When STEP_CONFIG is present the container runs in step mode instead of the
default agent (Matrix polling) mode.  The env var holds arbitrary JSON set by
bridge/pkg/studio/factory.go from WorkflowStep.Config (json.RawMessage).
"""

import json
import os


class StepConfig:
    """Parsed representation of the STEP_CONFIG env var."""

    __slots__ = ("raw", "task", "config", "step_id", "step_name")

    def __init__(self, raw, task, config, step_id=None, step_name=None):
        self.raw = raw
        self.task = task
        self.config = config
        self.step_id = step_id
        self.step_name = step_name

    @property
    def _retry(self):
        """Retry configuration, or None if absent/invalid."""
        val = self.config.get("_retry")
        if not isinstance(val, dict):
            return None
        attempt = val.get("attempt")
        if not isinstance(attempt, int) or attempt < 1:
            return None
        return val

    @property
    def _blocker_response(self):
        """Blocker response payload, or None if absent/invalid."""
        val = self.config.get("_blocker_response")
        if not isinstance(val, dict):
            return None
        inp = val.get("input")
        if not isinstance(inp, str) or not inp:
            return None
        return val

    @property
    def relevant_skills(self):
        """List of relevant skill descriptors, or None if absent/not a list."""
        val = self.config.get("relevant_skills")
        if not isinstance(val, list):
            return None
        return val


def parse_step_config():
    """Read and validate STEP_CONFIG from the environment.

    Returns StepConfig on success, None when the env var is absent/empty
    (indicating agent mode, not step mode).

    Raises ValueError for invalid or non-object JSON.
    """
    raw = os.environ.get("STEP_CONFIG", "").strip()
    if not raw:
        return None

    parsed = json.loads(raw)

    if not isinstance(parsed, dict):
        raise ValueError(
            f"STEP_CONFIG must be a JSON object, got {type(parsed).__name__}"
        )

    task = parsed.get("task") or os.environ.get("TASK_DESCRIPTION", "")
    config = parsed.get("config") if isinstance(parsed.get("config"), dict) else parsed
    step_id = parsed.get("step_id")
    step_name = parsed.get("step_name") or parsed.get("name")

    return StepConfig(
        raw=raw,
        task=task,
        config=config if isinstance(config, dict) else {},
        step_id=step_id,
        step_name=step_name,
    )
