import json
import os
import shutil
import tempfile
import unittest

import sys
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from openclaw.step_config import StepConfig
from openclaw.step_runner import StepRunner


class TestStepRunnerEcho(unittest.TestCase):
    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def _cfg(self, handler="echo", task="test message", **extra):
        config = {"handler": handler}
        config.update(extra)
        return StepConfig(
            raw=json.dumps(config),
            task=task,
            config=config,
        )

    def test_echo_handler_success(self):
        runner = StepRunner(state_dir=self.tmpdir)
        cfg = self._cfg(task="hello from echo")
        exit_code = runner.run(cfg)
        with open(os.path.join(self.tmpdir, "result.json")) as f:
            data = json.load(f)
        self.assertEqual(exit_code, 0)
        self.assertEqual(data["status"], "success")
        self.assertIn("hello from echo", data["output"])
        self.assertGreaterEqual(data["duration_ms"], 0)

    def test_unknown_handler_default(self):
        runner = StepRunner(state_dir=self.tmpdir)
        cfg = self._cfg(handler="nonexistent", task="fallback test")
        exit_code = runner.run(cfg)
        with open(os.path.join(self.tmpdir, "result.json")) as f:
            data = json.load(f)
        self.assertEqual(exit_code, 0)
        self.assertEqual(data["status"], "success")
        self.assertIn("fallback test", data["output"])

    def test_duration_ms_populated(self):
        runner = StepRunner(state_dir=self.tmpdir)
        cfg = self._cfg(task="duration check")
        runner.run(cfg)
        with open(os.path.join(self.tmpdir, "result.json")) as f:
            data = json.load(f)
        self.assertIsInstance(data["duration_ms"], int)
        self.assertGreaterEqual(data["duration_ms"], 0)
        self.assertLess(data["duration_ms"], 10000)

    def test_creates_state_dir_if_missing(self):
        nested = os.path.join(self.tmpdir, "sub", "dir")
        runner = StepRunner(state_dir=nested)
        cfg = self._cfg(task="mkdir test")
        exit_code = runner.run(cfg)
        self.assertEqual(exit_code, 0)
        self.assertTrue(os.path.exists(os.path.join(nested, "result.json")))


if __name__ == "__main__":
    unittest.main()
