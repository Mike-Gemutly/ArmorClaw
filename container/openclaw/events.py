"""
events — Structured event emitter for Agent Studio execution events.

Writes one JSON object per line to _events.jsonl in the bind-mounted state
directory. Every line respects PIPE_BUF (4096 bytes) so consumers can read
safely from a pipe without partial-line corruption.

Schema matches bridge/pkg/secretary/result.go StepEvent:
  seq         int              required
  type        string           required
  name        string           required
  ts_ms       int64            required
  detail      map[string]any   omitempty
  duration_ms *int             omitempty
"""

import json
import os
import time
from dataclasses import asdict, dataclass, field
from typing import Dict, Optional

PIPE_BUF = 4096
EVENTS_FILE = "_events.jsonl"


class EventType:
    STEP = "step"
    FILE_READ = "file_read"
    FILE_WRITE = "file_write"
    FILE_DELETE = "file_delete"
    COMMAND_RUN = "command_run"
    OBSERVATION = "observation"
    BLOCKER = "blocker"
    ERROR = "error"
    ARTIFACT = "artifact"
    PROGRESS = "progress"
    CHECKPOINT = "checkpoint"


# Field names must match Go StepEvent struct JSON tags exactly

@dataclass
class StepEvent:
    seq: int
    type: str
    name: str
    ts_ms: int
    detail: Optional[Dict] = field(default_factory=dict)
    duration_ms: Optional[int] = None


class EventEmitter:

    def __init__(self, state_dir: str) -> None:
        self._path = os.path.join(state_dir, EVENTS_FILE)
        self._fh = open(self._path, "a", encoding="utf-8")  # noqa: SIM115
        self._fh.write("# Agent Studio execution events\n")
        self._fh.flush()
        self._seq = 0
        self._start_ms = time.monotonic()

    def emit(
        self,
        event_type: str,
        name: str,
        detail: Optional[Dict] = None,
        duration_ms: Optional[int] = None,
    ) -> StepEvent:
        self._seq += 1
        event = StepEvent(
            seq=self._seq,
            type=event_type,
            name=name,
            ts_ms=int((time.monotonic() - self._start_ms) * 1000),
            detail=detail if detail is not None else {},
            duration_ms=duration_ms,
        )

        line = json.dumps(asdict(event)) + "\n"
        encoded = line.encode("utf-8")

        # PIPE_BUF enforcement
        if len(encoded) > PIPE_BUF:
            original_size = len(encoded)
            event.detail = {"_truncated": True, "_original_size": original_size}
            line = json.dumps(asdict(event)) + "\n"
            encoded = line.encode("utf-8")

            # If STILL over after truncating detail, truncate name
            if len(encoded) > PIPE_BUF:
                event.name = event.name[:64]
                line = json.dumps(asdict(event)) + "\n"
                encoded = line.encode("utf-8")

                # Last resort: drop detail entirely
                if len(encoded) > PIPE_BUF:
                    event.detail = {}
                    line = json.dumps(asdict(event)) + "\n"
                    encoded = line.encode("utf-8")

        self._fh.write(line)
        self._fh.flush()
        return event

    def step(self, name: str, detail: Optional[Dict] = None,
             duration_ms: Optional[int] = None) -> StepEvent:
        return self.emit(EventType.STEP, name, detail=detail,
                         duration_ms=duration_ms)

    def file_read(self, path: str, lines: int, size_bytes: int) -> StepEvent:
        return self.emit(EventType.FILE_READ, path,
                         detail={"lines": lines, "size_bytes": size_bytes})

    def file_write(self, path: str, changes: int, size_bytes: int) -> StepEvent:
        return self.emit(EventType.FILE_WRITE, path,
                         detail={"changes": changes, "size_bytes": size_bytes})

    def file_delete(self, path: str) -> StepEvent:
        return self.emit(EventType.FILE_DELETE, path)

    def command_run(
        self,
        command: str,
        exit_code: int,
        duration_ms: Optional[int] = None,
        truncated: bool = False,
    ) -> StepEvent:
        return self.emit(
            EventType.COMMAND_RUN, command,
            detail={"exit_code": exit_code, "truncated": truncated},
            duration_ms=duration_ms,
        )

    def observation(self, message: str,
                    detail: Optional[Dict] = None) -> StepEvent:
        return self.emit(EventType.OBSERVATION, message, detail=detail)

    def blocker(
        self,
        blocker_type: str,
        message: str,
        suggestion: str = "",
        field: str = "",
    ) -> StepEvent:
        return self.emit(
            EventType.BLOCKER, message,
            detail={
                "blocker_type": blocker_type,
                "suggestion": suggestion,
                "field": field,
            },
        )

    def error(self, message: str,
              detail: Optional[Dict] = None) -> StepEvent:
        return self.emit(EventType.ERROR, message, detail=detail)

    def artifact(
        self,
        name: str,
        path: str,
        mime_type: str = "",
        size_bytes: int = 0,
    ) -> StepEvent:
        return self.emit(
            EventType.ARTIFACT, name,
            detail={
                "path": path,
                "mime_type": mime_type,
                "size_bytes": size_bytes,
            },
        )

    def progress(self, percent: int,
                 message: str = "") -> StepEvent:
        return self.emit(
            EventType.PROGRESS, message,
            detail={"percent": percent},
        )

    def checkpoint(self, name: str,
                   detail: Optional[Dict] = None) -> StepEvent:
        return self.emit(EventType.CHECKPOINT, name, detail=detail)

    def close(self) -> StepEvent:
        elapsed_ms = int((time.monotonic() - self._start_ms) * 1000)
        summary = self.emit(
            "_summary", "_summary",
            detail={
                "total_events": self._seq,
                "total_ms": elapsed_ms,
            },
        )
        self._fh.close()
        return summary
