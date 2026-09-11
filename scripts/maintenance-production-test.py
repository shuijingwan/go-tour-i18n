#!/usr/bin/env python3

import copy
import importlib.util
import json
import pathlib
import subprocess
import tempfile
import unittest
from unittest import mock

ROOT = pathlib.Path(__file__).resolve().parent.parent

def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module

MAINTENANCE = load("maintenance_production", ROOT / "scripts" / "maintenance-production.py")

class MaintenanceProductionTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.parent = pathlib.Path(self.temp.name)

    def tearDown(self):
        self.temp.cleanup()

    def release(self, locale="de-DE", name=None):
        release = self.parent / (name or "go-tour-release-20260911-%s-test" % locale)
        release.mkdir()
        (release / "release.json").write_text(json.dumps({"locale": locale}), encoding="utf-8")
        return release

    def make(self, locale="de-DE", name=None):
        return MAINTENANCE.Orchestrator(self.release(locale, name))

    def run_results(self, *codes):
        return mock.patch.object(MAINTENANCE.subprocess, "run", side_effect=[subprocess.CompletedProcess([], code) for code in codes])

    def test_live_locale_automatic_path_and_final_pass_without_stdin(self):
        instance = self.make()
        with self.run_results(0, 0, 0, 0, 0) as run, mock.patch("builtins.input") as prompt:
            instance.execute()
        self.assertEqual([pathlib.Path(call.args[0][0]).name for call in run.call_args_list], [
            "production-cdn.py", "deploy-production.sh", "production-cdn.py", "verify-production.sh", "verify-production-browser.py"])
        prompt.assert_not_called()
        self.assertEqual(instance.receipt["result"], "passed")
        self.assertEqual(set(instance.receipt["stages"]), set(MAINTENANCE.STAGE_LABELS))
        self.assertNotIn("visual", instance.receipt["stages"])
        self.assertNotIn("cdn_purge_confirmed_at", instance.receipt)

    def test_first_production_and_unknown_locale_fail_before_commands(self):
        release = self.release()
        identity = copy.deepcopy(MAINTENANCE.IDENTITY.load_identity(ROOT / "production" / "identity.json"))
        next(p for p in identity["locales"] if p["locale"] == "de-DE")["production_state"] = "first-production"
        with mock.patch.object(MAINTENANCE.IDENTITY, "load_identity", return_value=identity):
            with self.assertRaisesRegex(MAINTENANCE.MaintenanceProductionError, "production_state=first-production"):
                MAINTENANCE.Orchestrator(release)
        with self.assertRaises(MAINTENANCE.MaintenanceProductionError):
            MAINTENANCE.Orchestrator(self.release("zz-ZZ", "go-tour-release-unknown"))

    def test_receipt_identity_and_stage_order_fail_closed(self):
        instance = self.make()
        instance.receipt["locale"] = "fr-FR"
        instance.write_receipt("failed")
        with self.assertRaises(MAINTENANCE.MaintenanceProductionError):
            MAINTENANCE.Orchestrator(instance.release_dir)
        instance.receipt_path.unlink()
        instance = MAINTENANCE.Orchestrator(instance.release_dir)
        instance.receipt["stages"]["purge"] = {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"}
        instance.receipt["stages"]["browser"] = {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"}
        instance.write_receipt("failed")
        with self.assertRaises(MAINTENANCE.MaintenanceProductionError) as raised:
            MAINTENANCE.Orchestrator(instance.release_dir)
        self.assertEqual(raised.exception.expected, "ordered receipt stage prefix")

    def test_cdn_preflight_failure_prevents_deploy(self):
        instance = self.make()
        with self.run_results(7) as run:
            with self.assertRaisesRegex(MAINTENANCE.MaintenanceProductionError, "exit 7"):
                instance.execute()
        self.assertEqual(run.call_count, 1)
        self.assertFalse(instance.stage_passed("deploy"))

    def test_deploy_or_purge_failure_stops_before_acceptance(self):
        instance = self.make()
        with self.run_results(0, 7) as run:
            with self.assertRaises(MAINTENANCE.MaintenanceProductionError): instance.execute()
        self.assertEqual(run.call_count, 2)
        other = self.make(name="go-tour-release-20260911-de-DE-purge")
        with self.run_results(0, 0, 9) as run:
            with self.assertRaises(MAINTENANCE.MaintenanceProductionError): other.execute()
        self.assertEqual(run.call_count, 3)
        self.assertTrue(other.stage_passed("deploy"))
        self.assertFalse(other.stage_passed("purge"))
        self.assertFalse(other.stage_passed("machine"))

    def test_machine_and_browser_failure_stop_in_order(self):
        instance = self.make(name="go-tour-release-20260911-de-DE-machine")
        with self.run_results(0, 0, 0, 3) as run:
            with self.assertRaises(MAINTENANCE.MaintenanceProductionError): instance.execute()
        self.assertEqual(run.call_count, 4)
        self.assertTrue(instance.stage_passed("purge"))
        other = self.make(name="go-tour-release-20260911-de-DE-browser")
        with self.run_results(0, 0, 0, 0, 5) as run:
            with self.assertRaises(MAINTENANCE.MaintenanceProductionError): other.execute()
        self.assertEqual(run.call_count, 5)
        self.assertFalse(other.stage_passed("browser"))

    def test_resume_revalidates_deploy_and_reuses_passed_purge(self):
        instance = self.make()
        for stage in ("deploy", "purge"):
            instance.receipt["stages"][stage] = {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"}
        instance.write_receipt("failed")
        resumed = MAINTENANCE.Orchestrator(instance.release_dir)
        with self.run_results(0, 0, 0, 0) as run: resumed.execute()
        self.assertEqual([pathlib.Path(call.args[0][0]).name for call in run.call_args_list], [
            "production-cdn.py", "deploy-production.sh", "verify-production.sh", "verify-production-browser.py"])

    def test_failed_deploy_receipt_reruns_strict_deploy(self):
        instance = self.make(); instance.write_receipt("failed")
        resumed = MAINTENANCE.Orchestrator(instance.release_dir)
        with self.run_results(0, 0, 0, 0, 0) as run: resumed.execute()
        self.assertEqual(pathlib.Path(run.call_args_list[1].args[0][0]).name, "deploy-production.sh")

    def test_passed_receipt_skips_everything(self):
        instance = self.make()
        for stage in MAINTENANCE.STAGE_LABELS:
            instance.receipt["stages"][stage] = {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"}
        instance.write_receipt("passed")
        resumed = MAINTENANCE.Orchestrator(instance.release_dir)
        with mock.patch.object(MAINTENANCE.subprocess, "run") as run: resumed.execute()
        run.assert_not_called()

    def test_historical_complete_receipt_remains_readable_and_skips(self):
        instance = self.make()
        instance.receipt["stages"] = {stage: {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"} for stage in ("deploy", "machine", "browser", "visual")}
        instance.receipt["cdn_purge_confirmed_at"] = "2026-09-11T00:00:00Z"
        instance.write_receipt("passed")
        resumed = MAINTENANCE.Orchestrator(instance.release_dir)
        self.assertTrue(resumed.historical_pass)
        with mock.patch.object(MAINTENANCE.subprocess, "run") as run: resumed.execute()
        run.assert_not_called()

    def test_historical_in_progress_receipt_requires_automatic_purge(self):
        instance = self.make()
        instance.receipt["stages"] = {stage: {"result": "PASS", "completed_at": "2026-09-11T00:00:00Z"} for stage in ("deploy", "machine", "browser")}
        instance.receipt["cdn_purge_confirmed_at"] = "2026-09-11T00:00:00Z"
        instance.write_receipt("failed")
        resumed = MAINTENANCE.Orchestrator(instance.release_dir)
        self.assertEqual(set(resumed.receipt["stages"]), {"deploy"})
        with self.run_results(0, 0, 0, 0, 0) as run: resumed.execute()
        self.assertEqual(pathlib.Path(run.call_args_list[2].args[0][0]).name, "production-cdn.py")

    def test_source_has_no_human_tokens_or_secrets(self):
        source = (ROOT / "scripts" / "maintenance-production.py").read_text(encoding="utf-8")
        for forbidden in ("input(", "PURGED", "VISUAL-PASS", "CF_Token", "TENCENTCLOUD_SECRET"):
            self.assertNotIn(forbidden, source)

if __name__ == "__main__": unittest.main()
