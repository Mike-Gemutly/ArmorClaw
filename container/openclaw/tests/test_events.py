import json
import os
import tempfile
import unittest

from container.openclaw.events import EventEmitter, EventType, PIPE_BUF


class TestEventEmitterBasic(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_header_and_valid_jsonl(self):
        em = EventEmitter(self.tmpdir)
        em.step("init", detail={"key": "value"})
        em.close()

        path = os.path.join(self.tmpdir, "_events.jsonl")
        with open(path) as f:
            lines = f.readlines()

        self.assertEqual(lines[0], "# Agent Studio execution events\n")
        # lines[1] is step event, lines[2] is _summary
        for line in lines[1:]:
            parsed = json.loads(line)
            self.assertIn("seq", parsed)
            self.assertIn("type", parsed)
            self.assertIn("name", parsed)
            self.assertIn("ts_ms", parsed)

    def test_seq_increments(self):
        em = EventEmitter(self.tmpdir)
        em.step("a")
        em.step("b")
        em.step("c")
        em.close()

        path = os.path.join(self.tmpdir, "_events.jsonl")
        with open(path) as f:
            lines = f.readlines()

        events = [json.loads(l) for l in lines[1:]]
        self.assertEqual(events[0]["seq"], 1)
        self.assertEqual(events[1]["seq"], 2)
        self.assertEqual(events[2]["seq"], 3)


class TestEventEmitterAllTypes(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_all_convenience_methods(self):
        em = EventEmitter(self.tmpdir)
        em.step("s")
        em.file_read("/a.txt", lines=10, size_bytes=100)
        em.file_write("/b.txt", changes=3, size_bytes=200)
        em.file_delete("/c.txt")
        em.command_run("ls", exit_code=0, duration_ms=50)
        em.observation("looks good", detail={"confidence": 0.9})
        em.blocker("auth", "need login", suggestion="try again", field="password")
        em.error("boom", detail={"trace": "..."})
        em.artifact("report", "/out.pdf", mime_type="application/pdf", size_bytes=4096)
        em.progress(75, "working")
        em.checkpoint("midpoint", detail={"phase": 2})
        em.close()

        path = os.path.join(self.tmpdir, "_events.jsonl")
        with open(path) as f:
            lines = f.readlines()

        events = [json.loads(l) for l in lines[1:]]

        expected_types = [
            EventType.STEP,
            EventType.FILE_READ,
            EventType.FILE_WRITE,
            EventType.FILE_DELETE,
            EventType.COMMAND_RUN,
            EventType.OBSERVATION,
            EventType.BLOCKER,
            EventType.ERROR,
            EventType.ARTIFACT,
            EventType.PROGRESS,
            EventType.CHECKPOINT,
            "_summary",
        ]
        for i, et in enumerate(expected_types):
            self.assertEqual(events[i]["type"], et)


class TestEventEmitterClose(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_summary_event(self):
        em = EventEmitter(self.tmpdir)
        em.step("a")
        em.step("b")
        em.step("c")
        summary = em.close()

        self.assertEqual(summary.type, "_summary")
        self.assertEqual(summary.detail["total_events"], 3)
        self.assertIn("total_ms", summary.detail)
        self.assertGreaterEqual(summary.detail["total_ms"], 0)


class TestEventEmitterPipeBufEnforcement(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_large_detail_truncated(self):
        em = EventEmitter(self.tmpdir)
        big_detail = {"data": "x" * 10000}
        em.step("big_event", detail=big_detail)
        em.close()

        path = os.path.join(self.tmpdir, "_events.jsonl")
        with open(path) as f:
            lines = f.readlines()

        for line in lines[1:]:
            self.assertLessEqual(len(line.encode("utf-8")), PIPE_BUF)
            parsed = json.loads(line)
            self.assertIn("seq", parsed)

        event = json.loads(lines[1])
        self.assertTrue(event["detail"].get("_truncated"))
        self.assertGreater(event["detail"]["_original_size"], PIPE_BUF)


class TestEventEmitterPipeBufEdgeCase(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_exactly_at_pipebuf(self):
        em = EventEmitter(self.tmpdir)
        # Build a detail that makes the line exactly PIPE_BUF or just under
        base_line = json.dumps({
            "seq": 1, "type": "step", "name": "x", "ts_ms": 0,
            "detail": {}, "duration_ms": None,
        }) + "\n"
        pad_needed = PIPE_BUF - len(base_line.encode("utf-8")) - 14  # room for "detail":{"k":"..."}
        detail = {"k": "x" * max(pad_needed, 0)}
        em.step("x", detail=detail)
        em.close()

        path = os.path.join(self.tmpdir, "_events.jsonl")
        with open(path) as f:
            lines = f.readlines()

        for line in lines[1:]:
            self.assertLessEqual(len(line.encode("utf-8")), PIPE_BUF)
            json.loads(line)


class TestEventEmitterPipeBufExtreme(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_very_long_name_and_detail(self):
        em = EventEmitter(self.tmpdir)
        huge_name = "n" * 5000
        huge_detail = {"d": "x" * 10000}
        em.emit("step", huge_name, detail=huge_detail)
        em.close()

        path = os.path.join(self.tmpdir, "_events.jsonl")
        with open(path) as f:
            lines = f.readlines()

        for line in lines[1:]:
            encoded = line.encode("utf-8")
            self.assertLessEqual(len(encoded), PIPE_BUF,
                                 f"Line exceeds PIPE_BUF: {len(encoded)} bytes")
            parsed = json.loads(line)
            self.assertIn("seq", parsed)


class TestEventEmitterNoPartialWrites(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_every_line_is_valid_json(self):
        em = EventEmitter(self.tmpdir)
        for i in range(50):
            em.step(f"step_{i}", detail={"payload": "data" * i})
        em.close()

        path = os.path.join(self.tmpdir, "_events.jsonl")
        with open(path) as f:
            lines = f.readlines()

        # Skip header comment
        for line in lines[1:]:
            parsed = json.loads(line)
            self.assertIn("seq", parsed)
            self.assertIn("type", parsed)


if __name__ == "__main__":
    unittest.main()
