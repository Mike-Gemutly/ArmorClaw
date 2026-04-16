import json
import os
import unittest

import sys
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from openclaw.step_config import StepConfig, parse_step_config


class TestParseStepConfig(unittest.TestCase):
    def setUp(self):
        os.environ.pop("STEP_CONFIG", None)
        os.environ.pop("TASK_DESCRIPTION", None)

    def test_valid_config(self):
        os.environ["STEP_CONFIG"] = json.dumps({"task": "research NYC", "model": "gpt-4"})
        cfg = parse_step_config()
        self.assertIsInstance(cfg, StepConfig)
        self.assertEqual(cfg.task, "research NYC")
        self.assertEqual(cfg.config.get("model"), "gpt-4")

    def test_absent_returns_none(self):
        cfg = parse_step_config()
        self.assertIsNone(cfg)

    def test_empty_string_returns_none(self):
        os.environ["STEP_CONFIG"] = ""
        cfg = parse_step_config()
        self.assertIsNone(cfg)

    def test_whitespace_only_returns_none(self):
        os.environ["STEP_CONFIG"] = "   "
        cfg = parse_step_config()
        self.assertIsNone(cfg)

    def test_invalid_json_raises(self):
        os.environ["STEP_CONFIG"] = "not-json-at-all"
        with self.assertRaises(ValueError):
            parse_step_config()

    def test_non_object_raises(self):
        os.environ["STEP_CONFIG"] = "[1,2,3]"
        with self.assertRaises(ValueError):
            parse_step_config()

    def test_scalar_raises(self):
        os.environ["STEP_CONFIG"] = "42"
        with self.assertRaises(ValueError):
            parse_step_config()

    def test_falls_back_to_task_description_env(self):
        os.environ["STEP_CONFIG"] = json.dumps({"handler": "echo"})
        os.environ["TASK_DESCRIPTION"] = "fallback task"
        cfg = parse_step_config()
        self.assertEqual(cfg.task, "fallback task")

    def test_extracts_step_name(self):
        os.environ["STEP_CONFIG"] = json.dumps({"name": "my step", "task": "do it"})
        cfg = parse_step_config()
        self.assertEqual(cfg.step_name, "my step")


if __name__ == "__main__":
    unittest.main()
