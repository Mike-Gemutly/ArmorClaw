import json
import os
import shutil
import tempfile

from openclaw.result_writer import build_result, write_enriched_result, write_result


def _read_result(state_dir):
    with open(os.path.join(state_dir, "result.json")) as f:
        return json.load(f)


class TestWriteEnrichedResultAllFields:
    def test_all_underscore_fields_present(self):
        state_dir = tempfile.mkdtemp()
        try:
            write_enriched_result(
                state_dir,
                status="success",
                output="done",
                duration_ms=100,
                comments=["good"],
                blockers=[{"blocker_type": "x", "message": "m", "suggestion": "s", "field": "f"}],
                skill_candidates=[{"name": "n", "description": "d", "pattern_type": "p", "pattern_data": {}, "confidence": 0.9}],
                events_summary={"total": 5, "types": {}},
            )
            result = _read_result(state_dir)
            assert result["status"] == "success"
            assert result["_comments"] == ["good"]
            assert len(result["_blockers"]) == 1
            assert len(result["_skill_candidates"]) == 1
            assert result["_events_summary"]["total"] == 5
        finally:
            shutil.rmtree(state_dir)


class TestWriteEnrichedResultOmitsEmpty:
    def test_empty_underscore_fields_absent_base_present(self):
        state_dir = tempfile.mkdtemp()
        try:
            write_enriched_result(
                state_dir,
                status="success",
                output="done",
                duration_ms=50,
                comments=None,
                blockers=None,
                skill_candidates=None,
                events_summary=None,
            )
            result = _read_result(state_dir)
            assert result["status"] == "success"
            assert result["output"] == "done"
            assert result["duration_ms"] == 50
            assert "_comments" not in result
            assert "_blockers" not in result
            assert "_skill_candidates" not in result
            assert "_events_summary" not in result
        finally:
            shutil.rmtree(state_dir)


class TestWriteEnrichedResultBackwardCompat:
    def test_existing_write_result_unchanged(self):
        state_dir = tempfile.mkdtemp()
        try:
            payload = build_result("success", "ok", error="err", data={"k": "v"}, duration_ms=10)
            assert write_result(state_dir, payload) is True
            result = _read_result(state_dir)
            assert result == {"status": "success", "output": "ok", "duration_ms": 10, "error": "err", "data": {"k": "v"}}
        finally:
            shutil.rmtree(state_dir)
