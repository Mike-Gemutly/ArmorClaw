"""
step_runner — Execute a single step and write result.json before exit.

Step mode is triggered when STEP_CONFIG env var is present (set by
bridge/pkg/studio/factory.go:103).  The runner parses the config,
dispatches to a built-in handler, and writes an atomic result.json
to the bind-mounted state directory.

v2: Integrates EventEmitter for structured event logging. Injects
emitter, comments, and blockers into the config dict so handlers
can emit events without importing events.py themselves.
"""

import json
import os
import sys
import time

from openclaw.events import EventEmitter
from openclaw.result_writer import build_result, write_result, write_enriched_result
from openclaw.step_config import StepConfig

HANDLERS = {}

EVENTS_FILE = "_events.jsonl"


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


def _extract_blockers_from_events(state_dir):
    """Read _events.jsonl and return list of blocker event detail dicts."""
    blockers = []
    path = os.path.join(state_dir, EVENTS_FILE)
    if not os.path.exists(path):
        return blockers
    try:
        with open(path, "r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue
                if event.get("type") == "blocker":
                    blockers.append(event.get("detail", {}))
    except OSError:
        pass
    return blockers


def _summarize_events(state_dir):
    """Read _events.jsonl and return a summary dict of event counts by type."""
    path = os.path.join(state_dir, EVENTS_FILE)
    if not os.path.exists(path):
        return None
    counts = {}
    total = 0
    try:
        with open(path, "r", encoding="utf-8") as fh:
            for line in fh:
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                try:
                    event = json.loads(line)
                except json.JSONDecodeError:
                    continue
                etype = event.get("type", "unknown")
                counts[etype] = counts.get(etype, 0) + 1
                total += 1
    except OSError:
        return None
    if total == 0:
        return None
    return {"total": total, "types": counts}


class StepRunner:
    def __init__(self, state_dir=None):
        self.state_dir = state_dir or os.environ.get(
            "STATE_DIR", "/home/claw/.openclaw"
        )

    def run(self, step_config):
        start = time.monotonic()
        exit_code = 0
        emitter = None
        try:
            emitter = EventEmitter(self.state_dir)

            if not isinstance(step_config.config, dict):
                step_config.config = {}

            step_config.config["_emitter_ref"] = emitter
            step_config.config["_comments"] = []
            step_config.config["_blockers"] = []

            retry = step_config._retry
            if retry is not None:
                emitter.observation("retry_context", {"attempt": retry.get("attempt")})

            blocker_resp = step_config._blocker_response
            if blocker_resp is not None:
                emitter.observation("blocker_response_received", blocker_resp)

            skills = step_config.relevant_skills
            if skills is not None:
                emitter.observation("skills_injected", {"count": len(skills)})

            output = self._execute(step_config)

            comments = step_config.config.get("_comments", [])
            config_blockers = step_config.config.get("_blockers", [])
            event_blockers = _extract_blockers_from_events(self.state_dir)
            merged_blockers = config_blockers + event_blockers

            events_summary = _summarize_events(self.state_dir)

            os.makedirs(self.state_dir, exist_ok=True)
            write_enriched_result(
                self.state_dir,
                status="success",
                output=output,
                duration_ms=self._elapsed_ms(start),
                comments=comments or None,
                blockers=merged_blockers or None,
                events_summary=events_summary,
            )
        except Exception as exc:
            exit_code = 1
            if emitter is not None:
                try:
                    emitter.error(str(exc))
                except Exception:
                    pass
            error_msg = str(exc)
            events_summary = _summarize_events(self.state_dir)
            try:
                os.makedirs(self.state_dir, exist_ok=True)
                write_enriched_result(
                    self.state_dir,
                    status="failed",
                    output="",
                    error=error_msg,
                    duration_ms=self._elapsed_ms(start),
                    events_summary=events_summary,
                )
            except Exception as write_exc:
                print(f"[step_runner] failed to write result: {write_exc}", file=sys.stderr)
                exit_code = 1
            print(f"[step_runner] step failed: {exc}", file=sys.stderr)
        finally:
            if emitter is not None:
                try:
                    emitter.close()
                except Exception:
                    pass

        return exit_code

    def _execute(self, step_config):
        handler_name = step_config.config.get("handler", "") if step_config.config else ""
        handler = HANDLERS.get(handler_name, _default_handler)
        return handler(step_config)

    @staticmethod
    def _elapsed_ms(start):
        return int((time.monotonic() - start) * 1000)
