#!/usr/bin/env python3

import builtins
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
        name = name or f"go-tour-release-20260904-{locale}-test"
        release = self.parent / name
        release.mkdir()
        (release / "release.json").write_text(json.dumps({"locale": locale}), encoding="utf-8")
        return release

    def make(self, locale="de-DE", name=None):
        return MAINTENANCE.Orchestrator(self.release(locale, name))

    def run_results(self, *codes):
        return mock.patch.object(
            MAINTENANCE.subprocess,
            "run",
            side_effect=[subprocess.CompletedProcess([], code) for code in codes],
        )

    def test_live_locale_normal_path_and_final_pass(self):
        instance = self.make()
        with self.run_results(0, 0, 0) as run, mock.patch.object(builtins, "input", side_effect=["PURGED", "VISUAL-PASS"]):
            instance.execute()
        self.assertEqual(run.call_count, 3)
        self.assertEqual(instance.receipt["result"], "passed")
        self.assertTrue(instance.stage_passed("deploy"))
        self.assertTrue(instance.stage_passed("machine"))
        self.assertTrue(instance.stage_passed("browser"))
        self.assertTrue(instance.stage_passed("visual"))
        self.assertIn("maintenance-production-receipt", instance.receipt_path.name)

    def test_first_production_locale_is_rejected_with_formal_flow_hint(self):
        release = self.release()
        identity = MAINTENANCE.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        identity = copy.deepcopy(identity)
        identity["locales"][2]["production_state"] = "first-production"
        with mock.patch.object(MAINTENANCE.IDENTITY, "load_identity", return_value=identity):
            with self.assertRaisesRegex(MAINTENANCE.MaintenanceProductionError, "production_state=first-production") as raised:
                MAINTENANCE.Orchestrator(release)
        self.assertIn("first-production.sh", raised.exception.next_step)

    def test_unknown_and_invalid_locale_fail_closed(self):
        for index, locale in enumerate(("it-IT", "not a locale")):
            with self.subTest(locale=locale):
                with self.assertRaises(MAINTENANCE.MaintenanceProductionError) as raised:
                    MAINTENANCE.Orchestrator(self.release(locale, f"go-tour-release-20260904-invalid-{index}"))
                self.assertEqual(raised.exception.expected, "one formal production identity")

    def test_receipt_identity_mismatch_fails_closed_for_other_release_or_hostname(self):
        release = self.release()
        receipt = release.parent / f"{release.name}.maintenance-production-receipt.json"
        receipt.write_text(json.dumps({
            "schema": MAINTENANCE.RECEIPT_SCHEMA, "locale": "fr-FR",
            "hostname": "fr-go-dev.shuijingwanwq.com", "cdn": "cloudflare",
            "public_url": "https://fr-go-dev.shuijingwanwq.com/", "release": "other",
            "result": "running", "stages": {},
        }), encoding="utf-8")
        with self.assertRaises(MAINTENANCE.MaintenanceProductionError) as raised:
            MAINTENANCE.Orchestrator(release)
        self.assertIn("不要复用其他 locale", raised.exception.next_step)

    def test_deploy_failure_does_not_continue(self):
        instance = self.make()
        with self.run_results(7) as run, mock.patch.object(builtins, "input") as prompt:
            with self.assertRaisesRegex(MAINTENANCE.MaintenanceProductionError, "exit 7"):
                instance.execute()
        self.assertEqual(run.call_count, 1)
        prompt.assert_not_called()
        self.assertFalse(instance.stage_passed("deploy"))

    def test_deploy_success_enters_purge_gate_and_unconfirmed_purge_cannot_verify(self):
        instance = self.make()
        with self.run_results(0) as run, mock.patch.object(builtins, "input", return_value="no"):
            with self.assertRaises(MAINTENANCE.MaintenanceProductionError) as raised:
                instance.execute()
        self.assertEqual(raised.exception.expected, "explicit confirmation PURGED")
        self.assertEqual(run.call_count, 1)
        self.assertTrue(instance.stage_passed("deploy"))

    def test_machine_failure_stops_before_browser(self):
        instance = self.make()
        with self.run_results(0, 3) as run, mock.patch.object(builtins, "input", return_value="PURGED"):
            with self.assertRaisesRegex(MAINTENANCE.MaintenanceProductionError, "exit 3"):
                instance.execute()
        self.assertEqual(run.call_count, 2)
        self.assertFalse(instance.stage_passed("machine"))

    def test_browser_failure_stops_before_visual_gate(self):
        instance = self.make()
        with self.run_results(0, 0, 5) as run, mock.patch.object(builtins, "input", side_effect=["PURGED"]):
            with self.assertRaisesRegex(MAINTENANCE.MaintenanceProductionError, "exit 5"):
                instance.execute()
        self.assertEqual(run.call_count, 3)
        self.assertFalse(instance.stage_passed("browser"))

    def test_resume_skips_only_successful_deployment_and_requires_purge_confirmation_again(self):
        instance = self.make()
        instance.receipt["stages"]["deploy"] = {"result": "PASS", "completed_at": "2026-09-04T00:00:00Z"}
        instance.write_receipt("failed")
        resumed = MAINTENANCE.Orchestrator(instance.release_dir)
        with self.run_results(0, 0) as run, mock.patch.object(builtins, "input", side_effect=["PURGED", "VISUAL-PASS"]) as prompt:
            resumed.execute()
        self.assertEqual(run.call_count, 2)
        prompt.assert_any_call("完成上述 hostname purge 后输入 PURGED 继续：")
        self.assertEqual(resumed.receipt["result"], "passed")

    def test_incomplete_passed_receipt_fails_closed_instead_of_printing_pass(self):
        instance = self.make()
        instance.receipt["stages"]["deploy"] = {"result": "PASS", "completed_at": "2026-09-04T00:00:00Z"}
        instance.write_receipt("passed")
        with self.assertRaises(MAINTENANCE.MaintenanceProductionError) as raised:
            MAINTENANCE.Orchestrator(instance.release_dir)
        self.assertEqual(raised.exception.expected, "complete passed receipt")

    def test_cloudflare_and_edgeone_human_gate_messages_are_profile_specific(self):
        identity = MAINTENANCE.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        cloudflare = next(p for p in identity["locales"] if p["locale"] == "de-DE")
        edgeone = next(p for p in identity["locales"] if p["locale"] == "zh-CN")
        cloudflare_text = "\n".join(MAINTENANCE.purge_instructions(cloudflare))
        edgeone_text = "\n".join(MAINTENANCE.purge_instructions(edgeone))
        self.assertIn(cloudflare["production_hostname"], cloudflare_text)
        self.assertIn("Custom Purge", cloudflare_text)
        self.assertIn("不得使用 Purge Everything", cloudflare_text)
        self.assertIn(edgeone["production_hostname"], edgeone_text)
        self.assertIn("EdgeOne", edgeone_text)

    def test_receipt_and_source_have_no_credentials_or_secret_values(self):
        instance = self.make()
        rendered = json.dumps(instance.receipt)
        source = (ROOT / "scripts" / "maintenance-production.py").read_text(encoding="utf-8")
        self.assertNotIn("CF_Token", rendered)
        self.assertNotIn("TOUR_AD_HTML", rendered)
        self.assertNotIn("CF_Token", source)
        self.assertNotIn("TOUR_AD_HTML", source)


if __name__ == "__main__":
    unittest.main()
