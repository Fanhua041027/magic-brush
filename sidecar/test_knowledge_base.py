import tempfile
import threading
import unittest
from pathlib import Path

from knowledge_base import KnowledgeBase


class KnowledgeBaseConcurrencyTests(unittest.TestCase):
    def test_reload_and_search_keep_consistent_snapshots(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            document = path / "content.md"
            document.write_text("## First\nalpha alpha", encoding="utf-8")
            knowledge_base = KnowledgeBase(str(path))
            failures = []
            stop = threading.Event()

            def search_worker():
                while not stop.is_set():
                    try:
                        results = knowledge_base.search("alpha beta")
                        for result in results:
                            self.assertIn(result["header"], {"First", "Second"})
                    except Exception as exc:  # pragma: no cover - captured across thread
                        failures.append(exc)
                        stop.set()

            worker = threading.Thread(target=search_worker)
            worker.start()
            try:
                for index in range(20):
                    if index % 2:
                        document.write_text("## First\nalpha alpha", encoding="utf-8")
                    else:
                        document.write_text("## Second\nbeta beta", encoding="utf-8")
                    knowledge_base.load(str(path))
            finally:
                stop.set()
                worker.join(timeout=5)

            self.assertFalse(worker.is_alive())
            self.assertEqual(failures, [])

    def test_failed_reload_preserves_previous_snapshot(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory)
            (path / "content.md").write_text("## Stable\nsearchable", encoding="utf-8")
            knowledge_base = KnowledgeBase(str(path))

            original_limit = KnowledgeBase.MAX_FILES
            KnowledgeBase.MAX_FILES = 0
            try:
                with self.assertRaises(ValueError):
                    knowledge_base.load(str(path))
            finally:
                KnowledgeBase.MAX_FILES = original_limit

            results = knowledge_base.search("searchable")
            self.assertEqual(results[0]["header"], "Stable")


if __name__ == "__main__":
    unittest.main()
