import unittest

from openclaw.step_config import StepConfig


def _make(config):
    return StepConfig(raw='{"type":"test"}', task="t", config=config)


class TestRetryPropertyPresent(unittest.TestCase):
    def test_returns_dict_when_valid(self):
        sc = _make({"_retry": {"attempt": 2, "previous_result": "failed"}})
        result = sc._retry
        self.assertIsInstance(result, dict)
        self.assertEqual(result["attempt"], 2)
        self.assertEqual(result["previous_result"], "failed")


class TestRetryPropertyAbsent(unittest.TestCase):
    def test_returns_none_when_missing(self):
        sc = _make({"type": "test"})
        self.assertIsNone(sc._retry)


class TestRetryPropertyInvalidAttempt(unittest.TestCase):
    def test_returns_none_when_attempt_is_zero(self):
        sc = _make({"_retry": {"attempt": 0}})
        self.assertIsNone(sc._retry)

    def test_returns_none_when_attempt_is_negative(self):
        sc = _make({"_retry": {"attempt": -1}})
        self.assertIsNone(sc._retry)

    def test_returns_none_when_attempt_is_string(self):
        sc = _make({"_retry": {"attempt": "2"}})
        self.assertIsNone(sc._retry)


class TestBlockerResponsePropertyPresent(unittest.TestCase):
    def test_returns_dict_when_valid(self):
        sc = _make({"_blocker_response": {"input": "pass123", "provided_at": 123456}})
        result = sc._blocker_response
        self.assertIsInstance(result, dict)
        self.assertEqual(result["input"], "pass123")
        self.assertEqual(result["provided_at"], 123456)


class TestBlockerResponsePropertyAbsent(unittest.TestCase):
    def test_returns_none_when_missing(self):
        sc = _make({"type": "test"})
        self.assertIsNone(sc._blocker_response)


class TestBlockerResponsePropertyInvalidInput(unittest.TestCase):
    def test_returns_none_when_input_is_empty_string(self):
        sc = _make({"_blocker_response": {"input": ""}})
        self.assertIsNone(sc._blocker_response)

    def test_returns_none_when_input_is_not_string(self):
        sc = _make({"_blocker_response": {"input": 123}})
        self.assertIsNone(sc._blocker_response)


class TestRelevantSkillsPresent(unittest.TestCase):
    def test_returns_list_when_present(self):
        skills = [{"name": "skill1"}]
        sc = _make({"relevant_skills": skills})
        result = sc.relevant_skills
        self.assertIsInstance(result, list)
        self.assertEqual(result, skills)


class TestRelevantSkillsAbsent(unittest.TestCase):
    def test_returns_none_when_missing(self):
        sc = _make({"type": "test"})
        self.assertIsNone(sc.relevant_skills)

    def test_returns_none_when_not_list(self):
        sc = _make({"relevant_skills": "not-a-list"})
        self.assertIsNone(sc.relevant_skills)


class TestAllMissing(unittest.TestCase):
    def test_all_three_return_none(self):
        sc = _make({"type": "test"})
        self.assertIsNone(sc._retry)
        self.assertIsNone(sc._blocker_response)
        self.assertIsNone(sc.relevant_skills)
