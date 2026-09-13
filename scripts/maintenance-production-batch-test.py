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

    def mark_stages(self, item, *stages):
        for stage in stages:
            item.receipt["stages"][stage] = {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"}
        item.write_receipt("failed")

    @staticmethod
    def stage_calls(calls):
        result = []
        for command, _ in calls:
            name = pathlib.Path(command[0]).name
            if name in ("bash",):
                continue
            if name == "production-cdn.py":
                if command[1] == "preflight":
                    continue
                result.append(("purge", command[-1]))
            elif name == "deploy-production.sh":
                result.append(("deploy", json.loads((pathlib.Path(command[1]) / "release.json").read_text())["locale"]))
            elif name == "verify-production.sh":
                result.append(("machine", json.loads((pathlib.Path(command[1]) / "release.json").read_text())["locale"]))
            elif name == "verify-production-browser.py":
                result.append(("browser", command[-1]))
        return result

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

    def test_all_authority_preflight_precedes_two_phase_stage_order_and_no_stdin(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR")])
        calls = []
        def run(command, **kwargs):
            calls.append((command, kwargs))
            return subprocess.CompletedProcess(command, 0)
        output = io.StringIO()
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run), redirect_stdout(output): batch.execute()
        self.assertEqual(
            [(pathlib.Path(command[0]).name, command[1] if pathlib.Path(command[0]).name == "production-cdn.py" else None)
             for command, _ in calls[:4]],
            [("bash", None), ("bash", None), ("production-cdn.py", "preflight"),
             ("production-cdn.py", "preflight")],
        )
        self.assertEqual(self.stage_calls(calls), [
            ("deploy", "de-DE"), ("purge", "de-DE"),
            ("deploy", "fr-FR"), ("purge", "fr-FR"),
            ("machine", "de-DE"), ("browser", "de-DE"),
            ("machine", "fr-FR"), ("browser", "fr-FR"),
        ])
        self.assertTrue(all(call[1]["stdin"] is subprocess.DEVNULL for call in calls))
        self.assertIn("locale | deploy | purge | machine | browser | result", output.getvalue())
        self.assertIn("PASS=2 FAILED=0 SKIPPED=0 PENDING=0", output.getvalue())

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

    def test_three_locale_success_is_strictly_two_phase(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR"), self.release("it-IT")])
        calls = []
        def run(command, **kwargs):
            calls.append((command, kwargs))
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run):
            batch.execute()
        self.assertEqual(self.stage_calls(calls), [
            ("deploy", "de-DE"), ("purge", "de-DE"),
            ("deploy", "fr-FR"), ("purge", "fr-FR"),
            ("deploy", "it-IT"), ("purge", "it-IT"),
            ("machine", "de-DE"), ("browser", "de-DE"),
            ("machine", "fr-FR"), ("browser", "fr-FR"),
            ("machine", "it-IT"), ("browser", "it-IT"),
        ])

    def test_phase_a_purge_failure_blocks_all_acceptance_and_later_mutation(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR"), self.release("it-IT")])
        calls = []
        def run(command, **kwargs):
            name = pathlib.Path(command[0]).name
            calls.append((command, kwargs))
            if name == "production-cdn.py" and command[1] == "purge" and command[-1] == "fr-FR":
                return subprocess.CompletedProcess(command, 7)
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run):
            with self.assertRaisesRegex(BATCH.BatchError, "locale=fr-FR stage=purge evidence="):
                batch.execute()
        self.assertEqual(self.stage_calls(calls), [
            ("deploy", "de-DE"), ("purge", "de-DE"),
            ("deploy", "fr-FR"), ("purge", "fr-FR"),
        ])
        self.assertEqual([row["result"] for row in batch.rows],
                         ["PENDING_ACCEPTANCE", "MUTATION_FAILED", "BLOCKED_PHASE_A"])
        self.assertEqual(set(batch.items[0].receipt["stages"]), {"deploy", "purge"})
        self.assertEqual(set(batch.items[1].receipt["stages"]), {"deploy"})
        self.assertEqual(batch.items[2].receipt["stages"], {})

    def test_phase_a_failure_preserves_prior_acceptance_failure_evidence(self):
        releases = [self.release("de-DE"), self.release("fr-FR")]
        initial = BATCH.Batch(releases)
        self.mark_stages(initial.items[0], "deploy", "purge", "machine")
        initial.items[0].receipt["failure"] = {
            "stage": "browser", "result": "failed", "completed_at": "2026-09-11T00:01:00Z",
        }
        initial.items[0].write_receipt("failed")
        batch = BATCH.Batch(releases)
        def run(command, **kwargs):
            if pathlib.Path(command[0]).name == "production-cdn.py" and command[1] == "purge" and command[-1] == "fr-FR":
                return subprocess.CompletedProcess(command, 7)
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run), self.assertRaises(BATCH.BatchError):
            batch.execute()
        persisted = json.loads(batch.items[0].receipt_path.read_text(encoding="utf-8"))
        self.assertEqual(persisted["result"], "failed")
        self.assertEqual(persisted["failure"]["stage"], "browser")
        self.assertEqual(set(persisted["stages"]), {"deploy", "purge", "machine"})

    def test_phase_b_browser_failure_is_fail_fast_without_repeating_mutation(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR"), self.release("it-IT")])
        calls = []
        def run(command, **kwargs):
            calls.append((command, kwargs))
            if pathlib.Path(command[0]).name == "verify-production-browser.py" and command[-1] == "fr-FR":
                return subprocess.CompletedProcess(command, 7)
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run), \
                self.assertRaisesRegex(BATCH.BatchError, "locale=fr-FR stage=browser evidence="):
            batch.execute()
        self.assertEqual(self.stage_calls(calls), [
            ("deploy", "de-DE"), ("purge", "de-DE"),
            ("deploy", "fr-FR"), ("purge", "fr-FR"),
            ("deploy", "it-IT"), ("purge", "it-IT"),
            ("machine", "de-DE"), ("browser", "de-DE"),
            ("machine", "fr-FR"), ("browser", "fr-FR"),
        ])
        self.assertEqual([row["result"] for row in batch.rows],
                         ["PASS", "ACCEPTANCE_FAILED", "BLOCKED_PHASE_B"])
        for item in batch.items:
            self.assertTrue(item.stage_passed("deploy"))
            self.assertTrue(item.stage_passed("purge"))

    def test_partial_receipt_resume_matches_live_zh_ja_de_fr_shape(self):
        releases = [self.release(locale) for locale in ("zh-CN", "ja-JP", "de-DE", "fr-FR")]
        initial = BATCH.Batch(releases)
        self.mark_passed(initial.items[0])
        self.mark_passed(initial.items[1])
        self.mark_stages(initial.items[2], "deploy", "purge", "machine")
        batch = BATCH.Batch(releases)
        calls = []
        def run(command, **kwargs):
            calls.append((command, kwargs))
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run):
            batch.execute()
        self.assertEqual(self.stage_calls(calls), [
            ("deploy", "fr-FR"), ("purge", "fr-FR"),
            ("browser", "de-DE"),
            ("machine", "fr-FR"), ("browser", "fr-FR"),
        ])
        self.assertEqual([row["result"] for row in batch.rows], ["SKIPPED", "SKIPPED", "PASS", "PASS"])

    def test_complete_receipt_skips_both_phases(self):
        batch = BATCH.Batch([self.release("de-DE"), self.release("fr-FR")])
        self.mark_passed(batch.items[0])
        batch = BATCH.Batch([item.release_dir for item in batch.items])
        calls = []
        def run(command, **kwargs):
            calls.append((command, kwargs))
            return subprocess.CompletedProcess(command, 0)
        with mock.patch.object(BATCH.subprocess, "run", side_effect=run):
            batch.execute()
        self.assertEqual(self.stage_calls(calls), [
            ("deploy", "fr-FR"), ("purge", "fr-FR"),
            ("machine", "fr-FR"), ("browser", "fr-FR"),
        ])
        self.assertEqual([row["result"] for row in batch.rows], ["SKIPPED", "PASS"])

if __name__ == "__main__": unittest.main()
