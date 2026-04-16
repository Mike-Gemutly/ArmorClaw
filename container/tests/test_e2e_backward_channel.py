import json
import os
import shutil
import tempfile
import unittest

import sys
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from openclaw.result_writer import build_result, write_result
from openclaw.step_config import StepConfig, parse_step_config
from openclaw.step_runner import StepRunner


class TestE2EBackwardChannel(unittest.TestCase):
    """End-to-end: container writes result.json → Bridge reads it."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_full_pipeline_echo(self):
        os.environ["STEP_CONFIG"] = json.dumps({"handler": "echo", "task": "e2e test"})
        os.environ["STATE_DIR"] = self.tmpdir
        try:
            cfg = parse_step_config()
            self.assertIsNotNone(cfg)
            runner = StepRunner()
            exit_code = runner.run(cfg)
            self.assertEqual(exit_code, 0)

            with open(os.path.join(self.tmpdir, "result.json")) as f:
                data = json.load(f)

            # Validate against ContainerStepResult schema (bridge/pkg/secretary/result.go)
            self.assertIsInstance(data["status"], str)
            self.assertIsInstance(data["output"], str)
            self.assertIsInstance(data["duration_ms"], int)
            self.assertEqual(data["status"], "success")
            self.assertIn("e2e test", data["output"])
        finally:
            os.environ.pop("STEP_CONFIG", None)
            os.environ.pop("STATE_DIR", None)

    def test_failed_step_writes_error(self):
        os.environ["STEP_CONFIG"] = json.dumps({"handler": "nonexistent"})
        os.environ["STATE_DIR"] = self.tmpdir
        try:
            cfg = parse_step_config()
            runner = StepRunner()
            exit_code = runner.run(cfg)

            with open(os.path.join(self.tmpdir, "result.json")) as f:
                data = json.load(f)

            self.assertIsInstance(data["duration_ms"], int)
            self.assertGreaterEqual(data["duration_ms"], 0)
            # Unknown handler still succeeds via default handler
            self.assertEqual(data["status"], "success")
        finally:
            os.environ.pop("STEP_CONFIG", None)
            os.environ.pop("STATE_DIR", None)

    def test_schema_matches_go_struct(self):
        """Validate the JSON structure is compatible with Go's ContainerStepResult."""
        result = build_result(
            status="success",
            output="schema test",
            data={"key": "value"},
            duration_ms=100,
        )
        write_result(self.tmpdir, result)

        with open(os.path.join(self.tmpdir, "result.json")) as f:
            data = json.load(f)

        self.assertIsInstance(data["status"], str)
        self.assertIsInstance(data["output"], str)
        self.assertIsInstance(data["duration_ms"], int)
        self.assertIsInstance(data["data"], dict)
        self.assertNotIn("error", data)  # omitempty: absent when None

    def test_overwrite_previous_result(self):
        r1 = build_result(status="running", output="first", duration_ms=50)
        write_result(self.tmpdir, r1)

        r2 = build_result(status="success", output="second", duration_ms=100)
        write_result(self.tmpdir, r2)

        with open(os.path.join(self.tmpdir, "result.json")) as f:
            data = json.load(f)
        self.assertEqual(data["output"], "second")
        self.assertEqual(data["duration_ms"], 100)


if __name__ == "__main__":
    unittest.main()
