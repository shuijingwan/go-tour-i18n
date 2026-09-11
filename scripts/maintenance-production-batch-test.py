#!/usr/bin/env python3

import importlib.util
import json
import pathlib
import subprocess
import tempfile
import unittest
import io
from contextlib import redirect_stdout
from unittest import mock

ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location("maintenance_batch", ROOT / "scripts" / "maintenance-production-batch.py")
BATCH = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(BATCH)

class BatchTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.temp.name)

    def tearDown(self): self.temp.cleanup()

    def release(self, locale):
        path = self.root / ("go-tour-release-20260911-%s-batch" % locale)
        path.mkdir()
        (path / "release.json").write_text(json.dumps({"locale": locale}), encoding="utf-8")
        return path

    def mark_passed(self, item):
        for stage in BATCH.CORE.STAGE_LABELS:
            item.receipt["stages"][stage] = {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"}
        item.write_receipt("passed")

    def test_duplicate_and_first_production_fail_before_mutation(self):
        release = self.release("de-DE")
        with self.assertRaisesRegex(BATCH.BatchError, "duplicate"):
            BATCH.Batch([release, release])
        identity = BATCH.CORE.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        profile = next(p for p in identity["locales"] if p["locale"] == "de-DE")
        profile["production_state"] = "first-production"
        with mock.patch.object(BATCH.CORE.IDENTITY, "load_identity", return_value=identity):
            with self.assertRaises(BATCH.CORE.MaintenanceProductionError):
                BATCH.Batch([release])

    def test_all_authority_preflight_happens_before_first_maintenance_and_no_stdin(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR")])
        calls = []
        def run(command, **kwargs):
            calls.append((command, kwargs))
            if pathlib.Path(command[0]).name == "maintenance-production.sh":
                item = next(item for item in batch.items if item.release_dir == pathlib.Path(command[1]))
                self.mark_passed(item)
            return subprocess.CompletedProcess(command, 0)
        output = io.StringIO()
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run), redirect_stdout(output): batch.execute()
        names = [pathlib.Path(call[0][0]).name for call in calls]
        self.assertEqual(names, ["bash", "bash", "production-cdn.py", "production-cdn.py", "maintenance-production.sh", "maintenance-production.sh"])
        self.assertTrue(all(call[1]["stdin"] is subprocess.DEVNULL for call in calls))
        self.assertIn("locale | deploy | purge | machine | browser | result", output.getvalue())
        self.assertIn("PASS=2 FAILED=0 SKIPPED=0", output.getvalue())

    def test_missing_cloudflare_or_edgeone_authority_stops_before_mutation(self):
        for locale in ("de-DE", "zh-CN"):
            with self.subTest(locale=locale):
                batch = BATCH.Batch([self.release(locale)])
                def fail_cdn(command, **kwargs):
                    return subprocess.CompletedProcess(command, 9 if pathlib.Path(command[0]).name == "production-cdn.py" else 0)
                with mock.patch.object(BATCH.subprocess, "run", side_effect=fail_cdn) as run:
                    with self.assertRaises(BATCH.BatchError): batch.execute()
                self.assertEqual(run.call_count, 2)
                self.assertEqual(pathlib.Path(run.call_args_list[-1].args[0][0]).name, "production-cdn.py")

    def test_strict_serial_fail_fast_and_no_rollback(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR"), self.release("it-IT")])
        maintenance = 0
        calls = []
        def run(command, **kwargs):
            nonlocal maintenance
            name = pathlib.Path(command[0]).name
            calls.append((name, command))
            if name == "maintenance-production.sh":
                maintenance += 1
                item = next(item for item in batch.items if item.release_dir == pathlib.Path(command[1]))
                if maintenance == 1:
                    self.mark_passed(item)
                    return subprocess.CompletedProcess(command, 0)
                item.receipt["failure"] = {"stage": "purge", "result": "failed", "completed_at": "2026-09-11T00:00:00Z"}
                item.write_receipt("failed")
                return subprocess.CompletedProcess(command, 7)
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run):
            with self.assertRaisesRegex(BATCH.BatchError, "locale=fr-FR stage=purge evidence="):
                batch.execute()
        started = [pathlib.Path(command[1]).name for name, command in calls if name == "maintenance-production.sh"]
        self.assertEqual(started, ["go-tour-release-20260911-de-DE-batch", "go-tour-release-20260911-fr-FR-batch"])
        self.assertEqual([row["result"] for row in batch.rows], ["PASS", "FAILED", "SKIPPED"])
        self.assertNotIn("rollback", " ".join(name for name, _ in calls).lower())

    def test_pass_receipt_skips_and_incomplete_uses_single_locale_resume(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR")])
        self.mark_passed(batch.items[0])
        batch = BATCH.Batch([item.release_dir for item in batch.items])
        calls = []
        def run(command, **kwargs):
            calls.append(command)
            if pathlib.Path(command[0]).name == "maintenance-production.sh":
                item = batch.items[1]
                self.mark_passed(item)
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run): batch.execute()
        maintenance = [command for command in calls if pathlib.Path(command[0]).name == "maintenance-production.sh"]
        self.assertEqual(len(maintenance), 1)
        self.assertEqual(pathlib.Path(maintenance[0][1]), batch.items[1].release_dir)
        self.assertEqual(sum(pathlib.Path(command[0]).name == "production-cdn.py" for command in calls), 2)
        self.assertEqual([row["result"] for row in batch.rows], ["SKIPPED", "PASS"])

if __name__ == "__main__": unittest.main()
