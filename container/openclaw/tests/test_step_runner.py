"""Tests for step_runner EventEmitter integration."""

import json
import os
import tempfile

import pytest

from openclaw.step_config import StepConfig
from openclaw.step_runner import HANDLERS, StepRunner


def _make_step_config(**overrides):
    defaults = {
        "raw": json.dumps({"task": "test", "config": {"handler": "_test_"}}),
        "task": "test",
        "config": {"handler": "_test_"},
    }
    defaults.update(overrides)
    return StepConfig(**defaults)


def _read_result(state_dir):
    with open(os.path.join(state_dir, "result.json")) as f:
        return json.load(f)


def _read_events(state_dir):
    events = []
    path = os.path.join(state_dir, "_events.jsonl")
    if not os.path.exists(path):
        return events
    with open(path) as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("#"):
                continue
            events.append(json.loads(line))
    return events


class TestStepRunnerWithEmitter:

    def setup_method(self):
        self._orig = HANDLERS.get("_test_")

    def teardown_method(self):
        if self._orig is None:
            HANDLERS.pop("_test_", None)
        else:
            HANDLERS["_test_"] = self._orig

    def test_emitter_events_written(self, tmp_path):
        def handler(cfg):
            emitter = cfg.config.get("_emitter_ref")
            if emitter:
                emitter.step("test_step")
            return "done"

        HANDLERS["_test_"] = handler
        sc = _make_step_config()
        runner = StepRunner(state_dir=str(tmp_path))
        exit_code = runner.run(sc)

        assert exit_code == 0
        events = _read_events(str(tmp_path))
        step_events = [e for e in events if e.get("type") == "step"]
        assert len(step_events) >= 1
        assert step_events[0]["name"] == "test_step"

        result = _read_result(str(tmp_path))
        assert result["status"] == "success"
        assert "_events_summary" in result
        assert result["_events_summary"]["total"] >= 1


class TestStepRunnerLegacyHandler:

    def setup_method(self):
        self._orig = HANDLERS.get("_test_")

    def teardown_method(self):
        if self._orig is None:
            HANDLERS.pop("_test_", None)
        else:
            HANDLERS["_test_"] = self._orig

    def test_handler_without_emitter_works(self, tmp_path):
        def handler(cfg):
            return f"legacy output: {cfg.task}"

        HANDLERS["_test_"] = handler
        sc = _make_step_config()
        runner = StepRunner(state_dir=str(tmp_path))
        exit_code = runner.run(sc)

        assert exit_code == 0
        result = _read_result(str(tmp_path))
        assert result["status"] == "success"
        assert "legacy output" in result["output"]


class TestStepRunnerBlockerViaConfig:

    def setup_method(self):
        self._orig = HANDLERS.get("_test_")

    def teardown_method(self):
        if self._orig is None:
            HANDLERS.pop("_test_", None)
        else:
            HANDLERS["_test_"] = self._orig

    def test_blockers_in_result(self, tmp_path):
        blocker = {"blocker_type": "missing_input", "message": "No API key", "suggestion": "Provide key"}

        def handler(cfg):
            cfg.config.get("_blockers", []).append(blocker)
            return "blocked"

        HANDLERS["_test_"] = handler
        sc = _make_step_config()
        runner = StepRunner(state_dir=str(tmp_path))
        exit_code = runner.run(sc)

        assert exit_code == 0
        result = _read_result(str(tmp_path))
        assert result["status"] == "success"
        assert "_blockers" in result
        assert len(result["_blockers"]) == 1
        assert result["_blockers"][0]["message"] == "No API key"


class TestStepRunnerRetryContext:

    def setup_method(self):
        self._orig = HANDLERS.get("_test_")

    def teardown_method(self):
        if self._orig is None:
            HANDLERS.pop("_test_", None)
        else:
            HANDLERS["_test_"] = self._orig

    def test_retry_observation_emitted(self, tmp_path):
        def handler(cfg):
            return "ok"

        HANDLERS["_test_"] = handler
        config = {"handler": "_test_", "_retry": {"attempt": 2}}
        sc = _make_step_config(config=config)
        runner = StepRunner(state_dir=str(tmp_path))
        exit_code = runner.run(sc)

        assert exit_code == 0
        events = _read_events(str(tmp_path))
        obs = [e for e in events if e.get("type") == "observation" and e.get("name") == "retry_context"]
        assert len(obs) == 1
        assert obs[0]["detail"]["attempt"] == 2


class TestStepRunnerException:

    def setup_method(self):
        self._orig = HANDLERS.get("_test_")

    def teardown_method(self):
        if self._orig is None:
            HANDLERS.pop("_test_", None)
        else:
            HANDLERS["_test_"] = self._orig

    def test_error_event_on_exception(self, tmp_path):
        def handler(cfg):
            raise RuntimeError("something broke")

        HANDLERS["_test_"] = handler
        sc = _make_step_config()
        runner = StepRunner(state_dir=str(tmp_path))
        exit_code = runner.run(sc)

        assert exit_code == 1
        events = _read_events(str(tmp_path))
        errors = [e for e in events if e.get("type") == "error"]
        assert len(errors) >= 1
        assert "something broke" in errors[0]["name"]

        result = _read_result(str(tmp_path))
        assert result["status"] == "failed"
        assert "something broke" in result["error"]


class TestStepRunnerLearnedSkills:

    def setup_method(self):
        self._orig = HANDLERS.get("_test_")

    def teardown_method(self):
        if self._orig is None:
            HANDLERS.pop("_test_", None)
        else:
            HANDLERS["_test_"] = self._orig

    def test_skills_observation_emitted(self, tmp_path):
        skills = [
            {
                "name": "db-migrate",
                "pattern_type": "command_sequence",
                "confidence": 0.7,
            }
        ]

        def handler(cfg):
            emitter = cfg.config.get("_emitter_ref")
            if cfg.relevant_skills and emitter:
                emitter.observation(
                    "skills_injected_handler",
                    {"skills": [s["name"] for s in cfg.relevant_skills]},
                )
            return "skills ack"

        HANDLERS["_test_"] = handler
        config = {"handler": "_test_", "relevant_skills": skills}
        sc = _make_step_config(config=config)
        runner = StepRunner(state_dir=str(tmp_path))
        exit_code = runner.run(sc)

        assert exit_code == 0

        # Verify _events.jsonl has a skills_injected observation from StepRunner
        events = _read_events(str(tmp_path))
        skills_obs = [
            e
            for e in events
            if e.get("type") == "observation" and e.get("name") == "skills_injected"
        ]
        assert len(skills_obs) == 1
        assert skills_obs[0]["detail"]["count"] == 1

        # Verify result.json has _events_summary with correct total
        result = _read_result(str(tmp_path))
        assert result["status"] == "success"
        assert "_events_summary" in result
        assert result["_events_summary"]["total"] >= 1


class TestCrossComponent:

    def setup_method(self):
        self._orig = HANDLERS.get("_test_")

    def teardown_method(self):
        if self._orig is None:
            HANDLERS.pop("_test_", None)
        else:
            HANDLERS["_test_"] = self._orig

    def test_result_json_matches_go_extended_contract(self, tmp_path):
        """Cross-component: result.json parseable by Go ExtendedStepResult parsing."""

        def handler(cfg):
            emitter = cfg.config.get("_emitter_ref")
            if emitter:
                emitter.step("cross_step")
                emitter.file_read("config.yaml", lines=42, size_bytes=1024)
                emitter.artifact("report.html", {"size": 2048})
            cfg.config.get("_comments", []).append("Cross-component check passed")
            cfg.config.get("_blockers", []).append(
                {
                    "blocker_type": "approval_needed",
                    "message": "Payment requires approval",
                    "suggestion": "Approve via ArmorChat",
                }
            )
            return "cross-component output"

        HANDLERS["_test_"] = handler
        sc = _make_step_config()
        runner = StepRunner(state_dir=str(tmp_path))
        exit_code = runner.run(sc)

        assert exit_code == 0

        result = _read_result(str(tmp_path))

        # Base fields (Go ContainerStepResult — always present)
        assert "status" in result
        assert "output" in result
        assert "duration_ms" in result
        assert isinstance(result["duration_ms"], int)

        # Underscore fields (Go ExtendedStepResult)
        assert "_comments" in result
        assert isinstance(result["_comments"], list)
        assert all(isinstance(c, str) for c in result["_comments"])

        assert "_blockers" in result
        assert isinstance(result["_blockers"], list)
        for blocker in result["_blockers"]:
            assert "blocker_type" in blocker

        assert "_events_summary" in result
        assert isinstance(result["_events_summary"], dict)
        assert "total" in result["_events_summary"]
        assert "types" in result["_events_summary"]

        # Verify _events_summary.total matches actual events in _events.jsonl
        # minus the _summary event appended by emitter.close() after summarize
        events = _read_events(str(tmp_path))
        non_summary_events = [e for e in events if e.get("name") != "_summary"]
        assert result["_events_summary"]["total"] == len(non_summary_events)

        # Verify each event matches Go StepEvent struct fields
        for evt in events:
            assert "seq" in evt, f"Event missing 'seq': {evt}"
            assert "type" in evt, f"Event missing 'type': {evt}"
            assert "name" in evt, f"Event missing 'name': {evt}"
            assert "ts_ms" in evt, f"Event missing 'ts_ms': {evt}"
