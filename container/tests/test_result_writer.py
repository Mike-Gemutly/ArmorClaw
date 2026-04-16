import json
import os
import shutil
import tempfile
import unittest

import sys
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..'))

from openclaw.result_writer import build_result, write_result


class TestBuildResult(unittest.TestCase):
    def test_minimal_fields(self):
        r = build_result(status="success", output="done")
        self.assertEqual(r["status"], "success")
        self.assertEqual(r["output"], "done")
        self.assertEqual(r["duration_ms"], 0)
        self.assertNotIn("error", r)
        self.assertNotIn("data", r)

    def test_all_fields(self):
        r = build_result(status="failed", output="oops", error="bad", data={"k": 1}, duration_ms=500)
        self.assertEqual(r["error"], "bad")
        self.assertEqual(r["data"], {"k": 1})
        self.assertEqual(r["duration_ms"], 500)


class TestWriteResult(unittest.TestCase):
    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_atomic_write_creates_valid_json(self):
        r = build_result(status="success", output="hello world", duration_ms=1500)
        write_result(self.tmpdir, r)
        with open(os.path.join(self.tmpdir, "result.json")) as f:
            data = json.load(f)
        self.assertEqual(data["status"], "success")
        self.assertEqual(data["output"], "hello world")
        self.assertEqual(data["duration_ms"], 1500)

    def test_overwrite_replaces_previous(self):
        r1 = build_result(status="running", output="first")
        r2 = build_result(status="success", output="second")
        write_result(self.tmpdir, r1)
        write_result(self.tmpdir, r2)
        with open(os.path.join(self.tmpdir, "result.json")) as f:
            data = json.load(f)
        self.assertEqual(data["output"], "second")
        self.assertEqual(data["status"], "success")

    def test_no_tmp_file_left(self):
        r = build_result(status="success", output="clean")
        write_result(self.tmpdir, r)
        files = os.listdir(self.tmpdir)
        self.assertNotIn("result.json.tmp", files)
        self.assertIn("result.json", files)

    def test_readonly_dir_raises(self):
        readonly = os.path.join(self.tmpdir, "readonly")
        os.makedirs(readonly)
        os.chmod(readonly, 0o000)
        try:
            r = build_result(status="success", output="test")
            with self.assertRaises((PermissionError, OSError)):
                write_result(readonly, r)
        finally:
            os.chmod(readonly, 0o755)


if __name__ == "__main__":
    unittest.main()
