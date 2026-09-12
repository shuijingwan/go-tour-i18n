#!/usr/bin/env python3

import contextlib
import datetime as dt
import importlib.util
import io
import json
import pathlib
import subprocess
import tempfile
import unittest
from unittest import mock


ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location(
    "production_release_batch", ROOT / "scripts" / "production-release-batch.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


def test_identity(states=None):
    states = states or {"zh-CN": ("live", "same-origin"), "de-DE": ("live", "shared-cloudflare")}
    return {
        "schema": "go-tour-i18n/production-identity/v1",
        "shared": {},
        "locales": [
            {"locale": locale, "production_state": state, "shared_assets_policy": policy}
            for locale, (state, policy) in states.items()
        ],
    }


class ProductionReleaseBatchTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.output_root = pathlib.Path(self.temp.name)
        self.now = lambda: dt.datetime(2026, 9, 11, 20, 30, 1, tzinfo=dt.timezone.utc)

    def tearDown(self):
        self.temp.cleanup()

    def workflow(self, locales, identity=None, all_live=False):
        with mock.patch.object(MODULE.IDENTITY, "load_identity", return_value=identity or test_identity()):
            return MODULE.ReleaseBatch(self.output_root, all_live, locales, now=self.now)

    def create_publish_result(self, workflow):
        releases = []
        for index, locale in enumerate(workflow.locales):
            release = self.output_root / ("release-%d-%s" % (index, locale))
            release.mkdir()
            releases.append({
                "locale": locale,
                "published_at": "2026-09-11T20:3%d:01Z" % index,
                "release_dir": str(release),
                "result": "PASS",
            })
        return {"repository_head": "0123456789abcdef0123456789abcdef01234567", "releases": releases}

    def write_shared_pass(self, workflow, deployment="DEPLOYED", purge="PASS"):
        receipt = pathlib.Path(str(workflow.assets_export) + ".verification-receipt.json")
        receipt.write_text(json.dumps({
            "schema": "go-tour-i18n/shared-assets-production-receipt/v2",
            "export_dir": str(workflow.assets_export),
            "deployment_result": deployment,
            "purge_result": purge,
            "verification_result": "PASS",
        }), encoding="utf-8")

    def write_maintenance_passes(self, result):
        for item in result["releases"]:
            release = pathlib.Path(item["release_dir"])
            receipt = release.parent / (release.name + ".maintenance-production-receipt.json")
            receipt.write_text(json.dumps({
                "result": "passed",
                "stages": {stage: {"result": "PASS"} for stage in ("deploy", "purge", "machine", "browser")},
            }), encoding="utf-8")

    def fake_success_runner(self, workflow, calls):
        result = None

        def run(stage, command, timeout, capture=False):
            nonlocal result
            command = [str(value) for value in command]
            calls.append((stage, command, capture))
            if stage == "assets-export":
                workflow.assets_export.mkdir()
            elif stage == "publish":
                result = self.create_publish_result(workflow)
                return json.dumps(result)
            elif stage == "shared-assets":
                self.write_shared_pass(workflow)
            elif stage == "maintenance":
                self.write_maintenance_passes(result)
            return ""

        return run

    def test_all_live_is_derived_and_explicit_subset_preserves_order(self):
        identity = test_identity({
            "zh-CN": ("live", "same-origin"),
            "ja-JP": ("first-production", "shared-cloudflare"),
            "de-DE": ("live", "shared-cloudflare"),
        })
        self.assertEqual(self.workflow([], identity, all_live=True).locales, ["zh-CN", "de-DE"])
        self.assertEqual(self.workflow(["de-DE", "zh-CN"], identity).locales, ["de-DE", "zh-CN"])

    def test_invalid_selection_fails_before_any_command(self):
        identity = test_identity({
            "zh-CN": ("live", "same-origin"),
            "ja-JP": ("first-production", "shared-cloudflare"),
        })
        for locales in (["unknown"], ["ja-JP"], ["zh-CN", "zh-CN"]):
            with self.subTest(locales=locales), \
                    mock.patch.object(MODULE.IDENTITY, "load_identity", return_value=identity), \
                    mock.patch.object(MODULE.ReleaseBatch, "_run") as run, \
                    self.assertRaises(MODULE.ReleaseBatchError):
                MODULE.ReleaseBatch(self.output_root, False, locales, now=self.now)
            run.assert_not_called()

    def test_shared_locale_runs_only_after_all_publish_and_preflight(self):
        workflow = self.workflow(["zh-CN", "de-DE"])
        calls = []
        with mock.patch.object(workflow, "_run", side_effect=self.fake_success_runner(workflow, calls)), \
                mock.patch.object(workflow, "_maintenance_preflight", side_effect=lambda: calls.append(("maintenance-preflight", [], False))), \
                contextlib.redirect_stdout(io.StringIO()):
            workflow.execute()
        self.assertEqual([item[0] for item in calls], [
            "assets-export", "assets-validate", "publish", "maintenance-preflight", "shared-assets", "maintenance",
        ])
        maintenance = next(command for stage, command, _ in calls if stage == "maintenance")
        self.assertEqual(maintenance[1:], [item["release_dir"] for item in workflow.publish_result["releases"]])

    def test_maintenance_preflight_delegates_exact_release_dirs(self):
        workflow = self.workflow(["de-DE", "zh-CN"])
        workflow.publish_result = self.create_publish_result(workflow)
        expected = [item["release_dir"] for item in workflow.publish_result["releases"]]
        with mock.patch.object(MODULE.MAINTENANCE, "Batch") as batch:
            batch.return_value.items = []
            self.assertEqual(workflow._maintenance_preflight(), set())
        batch.assert_called_once_with(expected)
        batch.return_value.preflight.assert_called_once_with()

    def test_publish_failure_stops_all_production_work(self):
        workflow = self.workflow(["de-DE"])
        calls = []

        def run(stage, command, timeout, capture=False):
            calls.append(stage)
            if stage == "assets-export":
                workflow.assets_export.mkdir()
            if stage == "publish":
                raise MODULE.ReleaseBatchError("publish", "locale failed")
            return ""

        with mock.patch.object(workflow, "_run", side_effect=run), \
                mock.patch.object(workflow, "_maintenance_preflight") as preflight, \
                self.assertRaises(MODULE.ReleaseBatchError):
            workflow.execute()
        self.assertEqual(calls, ["assets-export", "assets-validate", "publish"])
        preflight.assert_not_called()

    def test_same_origin_skips_shared_assets_and_shared_failure_stops_maintenance(self):
        same_origin = self.workflow(["zh-CN"])
        calls = []
        with mock.patch.object(same_origin, "_run", side_effect=self.fake_success_runner(same_origin, calls)), \
                mock.patch.object(same_origin, "_maintenance_preflight"), \
                contextlib.redirect_stdout(io.StringIO()):
            same_origin.execute()
        self.assertNotIn("shared-assets", [item[0] for item in calls])

        self.output_root = self.output_root / "shared-case"
        self.output_root.mkdir()
        shared = self.workflow(["de-DE"])
        stages = []

        def fail_shared(stage, command, timeout, capture=False):
            stages.append(stage)
            if stage == "assets-export":
                shared.assets_export.mkdir()
            if stage == "publish":
                return json.dumps(self.create_publish_result(shared))
            if stage == "shared-assets":
                raise MODULE.ReleaseBatchError("shared-assets", "purge failed")
            return ""

        with mock.patch.object(shared, "_run", side_effect=fail_shared), \
                mock.patch.object(shared, "_maintenance_preflight"), \
                self.assertRaises(MODULE.ReleaseBatchError):
            shared.execute()
        self.assertNotIn("maintenance", stages)

    def test_final_summary_and_no_stdin_runner(self):
        workflow = self.workflow(["de-DE"])
        calls = []
        output = io.StringIO()
        with mock.patch.object(workflow, "_run", side_effect=self.fake_success_runner(workflow, calls)), \
                mock.patch.object(workflow, "_maintenance_preflight"), contextlib.redirect_stdout(output):
            workflow.execute()
        rendered = output.getvalue()
        self.assertIn("repository_head: 0123456789abcdef0123456789abcdef01234567", rendered)
        self.assertIn("deploy       DEPLOYED", rendered)
        self.assertIn("de-DE | 2026-09-11T20:30:01Z | PASS | PASS | PASS | PASS | PASS | PASS", rendered)
        self.assertIn("PASS=1 FAILED=0 SKIPPED=0", rendered)

        with mock.patch.object(MODULE.subprocess, "run", return_value=subprocess.CompletedProcess([], 0)) as run:
            MODULE.ReleaseBatch._run("test", ["true"], 1)
        self.assertIs(run.call_args.kwargs["stdin"], subprocess.DEVNULL)

    def test_final_summary_preserves_existing_maintenance_skip(self):
        workflow = self.workflow(["de-DE"])
        calls = []
        output = io.StringIO()
        with mock.patch.object(workflow, "_run", side_effect=self.fake_success_runner(workflow, calls)), \
                mock.patch.object(workflow, "_maintenance_preflight", side_effect=lambda: {
                    str(self.output_root / "release-0-de-DE")
                }), contextlib.redirect_stdout(output):
            workflow.execute()
        self.assertIn("de-DE | 2026-09-11T20:30:01Z | PASS | PASS | PASS | PASS | PASS | SKIPPED",
                      output.getvalue())
        self.assertIn("PASS=0 FAILED=0 SKIPPED=1", output.getvalue())

    def test_source_has_no_search_first_production_or_confirmation_tokens(self):
        source = (ROOT / "scripts" / "production-release-batch.py").read_text(encoding="utf-8").lower()
        for forbidden in ("indexnow", "search console", "first-production", "visual-pass", "purged"):
            self.assertNotIn(forbidden, source)


if __name__ == "__main__":
    unittest.main()
