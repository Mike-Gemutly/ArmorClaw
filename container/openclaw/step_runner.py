"""
step_runner — Execute a single step and write result.json before exit.

Step mode is triggered when STEP_CONFIG env var is present (set by
bridge/pkg/studio/factory.go:103).  The runner parses the config,
dispatches to a built-in handler, and writes an atomic result.json
to the bind-mounted state directory.
"""

import os
import sys
import time

from openclaw.result_writer import build_result, write_result
from openclaw.step_config import StepConfig

HANDLERS = {}


def register_handler(name):
    def decorator(fn):
        HANDLERS[name] = fn
        return fn
    return decorator


@register_handler("echo")
def _echo(cfg):
    return cfg.task or "echo: no task description"


@register_handler("transform")
def _transform(cfg):
    return str(cfg.config)


def _default_handler(cfg):
    return f"step received: {cfg.task or 'unnamed'}"


class StepRunner:
    def __init__(self, state_dir=None):
        self.state_dir = state_dir or os.environ.get(
            "STATE_DIR", "/home/claw/.openclaw"
        )

    def run(self, step_config):
        start = time.monotonic()
        exit_code = 0
        try:
            output = self._execute(step_config)
            result = build_result(
                status="success",
                output=output,
                duration_ms=self._elapsed_ms(start),
            )
        except Exception as exc:
            exit_code = 1
            result = build_result(
                status="failed",
                output="",
                error=str(exc),
                duration_ms=self._elapsed_ms(start),
            )
            print(f"[step_runner] step failed: {exc}", file=sys.stderr)

        try:
            os.makedirs(self.state_dir, exist_ok=True)
            write_result(self.state_dir, result)
        except Exception as write_exc:
            print(f"[step_runner] failed to write result: {write_exc}", file=sys.stderr)
            exit_code = 1

        return exit_code

    def _execute(self, step_config):
        handler_name = step_config.config.get("handler", "") if step_config.config else ""
        handler = HANDLERS.get(handler_name, _default_handler)
        return handler(step_config)

    @staticmethod
    def _elapsed_ms(start):
        return int((time.monotonic() - start) * 1000)
