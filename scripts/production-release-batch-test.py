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
HEAD = "0123456789abcdef0123456789abcdef01234567"
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
            release = self.output_root / ("go-tour-release-20260911T203%d01Z-%s-%s" %
                                          (index, locale, HEAD[:12]))
            release.mkdir()
            published_at = "2026-09-11T20:3%d:01Z" % index
            (release / "release.json").write_text(json.dumps({
                "locale": locale, "published_at": published_at,
            }), encoding="utf-8")
            (release / "SHA256SUMS").write_text("formal bundle manifest\n", encoding="utf-8")
            releases.append({
                "locale": locale,
                "published_at": published_at,
                "release_dir": str(release),
                "result": "PASS",
            })
        return {"repository_head": HEAD, "releases": releases}

    def write_shared_pass(self, workflow, deployment="DEPLOYED", purge="PASS"):
        receipt = pathlib.Path(str(workflow.assets_export) + ".verification-receipt.json")
        receipt.write_text(json.dumps({
            "schema": "go-tour-i18n/shared-assets-production-receipt/v2",
            "export_dir": str(workflow.assets_export),
            "manifest_sha256": MODULE.sha256_file(workflow.assets_export / "SHA256SUMS"),
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
                (workflow.assets_export / "SHA256SUMS").write_text("formal assets manifest\n", encoding="utf-8")
            elif stage == "publish":
                result = self.create_publish_result(workflow)
                return json.dumps(result)
            elif stage == "shared-assets":
                self.write_shared_pass(workflow)
            elif stage == "maintenance":
                self.write_maintenance_passes(result)
            return ""

        return run

    def prepare_state(self, workflow):
        workflow.assets_export.mkdir()
        (workflow.assets_export / "SHA256SUMS").write_text("formal assets manifest\n", encoding="utf-8")
        workflow.shared_summary["export"] = "PASS"
        workflow.publish_result = self.create_publish_result(workflow)
        workflow.publish_status = "PASS"
        with mock.patch.object(workflow, "_repository_identity", return_value=HEAD):
            workflow._write_state()
        return workflow.state_path

    def load_state(self, path, identity=None):
        with mock.patch.object(MODULE.IDENTITY, "load_identity", return_value=identity or test_identity()):
            return MODULE.ReleaseBatch.from_state(path)

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
                mock.patch.object(workflow, "_repository_identity", return_value=HEAD), \
                mock.patch.object(workflow, "_maintenance_preflight", side_effect=lambda: calls.append(("maintenance-preflight", [], False))), \
                contextlib.redirect_stdout(io.StringIO()):
            workflow.execute()
        self.assertEqual([item[0] for item in calls], [
            "assets-export", "assets-validate", "publish", "resume-assets-validate",
            "maintenance-preflight", "shared-assets", "maintenance",
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
                (workflow.assets_export / "SHA256SUMS").write_text("formal assets manifest\n", encoding="utf-8")
            if stage == "publish":
                raise MODULE.ReleaseBatchError("publish", "locale failed")
            return ""

        with mock.patch.object(workflow, "_run", side_effect=run), \
                mock.patch.object(workflow, "_maintenance_preflight") as preflight, \
                self.assertRaises(MODULE.ReleaseBatchError):
            workflow.execute()
        self.assertEqual(calls, ["assets-export", "assets-validate", "publish"])
        self.assertFalse(workflow.state_path.exists())
        preflight.assert_not_called()

    def test_same_origin_skips_shared_assets_and_shared_failure_stops_maintenance(self):
        same_origin = self.workflow(["zh-CN"])
        calls = []
        with mock.patch.object(same_origin, "_run", side_effect=self.fake_success_runner(same_origin, calls)), \
                mock.patch.object(same_origin, "_repository_identity", return_value=HEAD), \
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
                (shared.assets_export / "SHA256SUMS").write_text("formal assets manifest\n", encoding="utf-8")
            if stage == "publish":
                return json.dumps(self.create_publish_result(shared))
            if stage == "shared-assets":
                raise MODULE.ReleaseBatchError("shared-assets", "purge failed")
            return ""

        with mock.patch.object(shared, "_run", side_effect=fail_shared), \
                mock.patch.object(shared, "_repository_identity", return_value=HEAD), \
                mock.patch.object(shared, "_maintenance_preflight"), \
                self.assertRaises(MODULE.ReleaseBatchError):
            shared.execute()
        self.assertNotIn("maintenance", stages)

    def test_final_summary_and_no_stdin_runner(self):
        workflow = self.workflow(["de-DE"])
        calls = []
        output = io.StringIO()
        with mock.patch.object(workflow, "_run", side_effect=self.fake_success_runner(workflow, calls)), \
                mock.patch.object(workflow, "_repository_identity", return_value=HEAD), \
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
                mock.patch.object(workflow, "_repository_identity", return_value=HEAD), \
                mock.patch.object(workflow, "_maintenance_preflight", side_effect=lambda: {
                    str(self.output_root / ("go-tour-release-20260911T203001Z-de-DE-%s" % HEAD[:12]))
                }), contextlib.redirect_stdout(output):
            workflow.execute()
        self.assertIn("de-DE | 2026-09-11T20:30:01Z | PASS | PASS | PASS | PASS | PASS | SKIPPED",
                      output.getvalue())
        self.assertIn("PASS=0 FAILED=0 SKIPPED=1", output.getvalue())

    def test_shared_verify_failure_keeps_state_and_never_starts_maintenance(self):
        workflow = self.workflow(["de-DE"])
        calls = []

        def run(stage, command, timeout, capture=False):
            calls.append(stage)
            if stage == "assets-export":
                workflow.assets_export.mkdir()
                (workflow.assets_export / "SHA256SUMS").write_text("formal assets manifest\n", encoding="utf-8")
            elif stage == "publish":
                return json.dumps(self.create_publish_result(workflow))
            elif stage == "shared-assets":
                receipt = pathlib.Path(str(workflow.assets_export) + ".verification-receipt.json")
                receipt.write_text(json.dumps({
                    "schema": "go-tour-i18n/shared-assets-production-receipt/v2",
                    "export_dir": str(workflow.assets_export),
                    "manifest_sha256": MODULE.sha256_file(workflow.assets_export / "SHA256SUMS"),
                    "deployment_result": "DEPLOYED", "purge_result": "PASS",
                    "verification_result": "PENDING",
                }), encoding="utf-8")
                raise MODULE.ReleaseBatchError("shared-assets", "public HTTP failed")
            return ""

        with mock.patch.object(workflow, "_run", side_effect=run), \
                mock.patch.object(workflow, "_repository_identity", return_value=HEAD), \
                mock.patch.object(workflow, "_maintenance_preflight", return_value=set()), \
                self.assertRaises(MODULE.ReleaseBatchError) as raised:
            workflow.execute()
        failure = io.StringIO()
        with contextlib.redirect_stderr(failure):
            workflow.print_failure(raised.exception)
        self.assertTrue(workflow.state_path.is_file())
        state = json.loads(workflow.state_path.read_text(encoding="utf-8"))
        self.assertEqual((state["schema"], state["stage"], state["publish_result"]),
                         (MODULE.STATE_SCHEMA, MODULE.STATE_STAGE, "PASS"))
        self.assertEqual(state["selection"]["locales"], ["de-DE"])
        self.assertEqual(state["releases"][0]["release_dir"],
                         workflow.publish_result["releases"][0]["release_dir"])
        self.assertEqual(state["shared_assets"]["export_dir"], str(workflow.assets_export))
        self.assertNotIn("maintenance", calls)
        self.assertIn("publish: PASS", failure.getvalue())
        self.assertIn("verify       PENDING", failure.getvalue())
        self.assertIn("locale maintenance: NOT_STARTED", failure.getvalue())
        self.assertIn("scripts/production-release-batch.sh --resume", failure.getvalue())

    def test_resume_skips_publish_and_preserves_exact_release_order(self):
        fresh = self.workflow(["de-DE", "zh-CN"])
        state_path = self.prepare_state(fresh)
        receipt = pathlib.Path(str(fresh.assets_export) + ".verification-receipt.json")
        receipt.write_text(json.dumps({
            "schema": "go-tour-i18n/shared-assets-production-receipt/v2",
            "export_dir": str(fresh.assets_export),
            "manifest_sha256": MODULE.sha256_file(fresh.assets_export / "SHA256SUMS"),
            "deployment_result": "DEPLOYED", "purge_result": "PASS",
            "verification_result": "PENDING",
        }), encoding="utf-8")
        resumed = self.load_state(state_path)
        calls = []
        expected = [item["release_dir"] for item in resumed.publish_result["releases"]]

        def run(stage, command, timeout, capture=False):
            calls.append((stage, [str(value) for value in command]))
            if stage == "shared-assets":
                current = json.loads(receipt.read_text(encoding="utf-8"))
                self.assertEqual((current["deployment_result"], current["purge_result"],
                                  current["verification_result"]), ("DEPLOYED", "PASS", "PENDING"))
                current["verification_result"] = "PASS"
                receipt.write_text(json.dumps(current), encoding="utf-8")
            elif stage == "maintenance":
                self.assertEqual([str(value) for value in command[1:]], expected)
                self.write_maintenance_passes(resumed.publish_result)
            return ""

        with mock.patch.object(resumed, "_run", side_effect=run), \
                mock.patch.object(resumed, "_repository_identity", return_value=HEAD), \
                mock.patch.object(resumed, "_maintenance_preflight", return_value=set()), \
                contextlib.redirect_stdout(io.StringIO()):
            resumed.execute(resume=True)
        stages = [stage for stage, _ in calls]
        self.assertEqual(stages, ["resume-assets-validate", "shared-assets", "maintenance"])
        self.assertNotIn("publish", stages)
        self.assertNotIn("assets-export", stages)

    def test_maintenance_failure_resume_reuses_both_existing_state_machines(self):
        fresh = self.workflow(["de-DE", "zh-CN"])
        state_path = self.prepare_state(fresh)
        self.write_shared_pass(fresh)
        first_release = fresh.publish_result["releases"][0]
        release = pathlib.Path(first_release["release_dir"])
        receipt = release.parent / (release.name + ".maintenance-production-receipt.json")
        receipt.write_text(json.dumps({
            "result": "passed",
            "stages": {stage: {"result": "PASS"} for stage in ("deploy", "purge", "machine", "browser")},
        }), encoding="utf-8")
        second = pathlib.Path(fresh.publish_result["releases"][1]["release_dir"])
        second_receipt = second.parent / (second.name + ".maintenance-production-receipt.json")
        second_receipt.write_text(json.dumps({
            "result": "failed", "stages": {"deploy": {"result": "PASS"}},
            "failure": {"stage": "purge", "result": "failed"},
        }), encoding="utf-8")
        resumed = self.load_state(state_path)
        calls = []

        def run(stage, command, timeout, capture=False):
            calls.append(stage)
            if stage == "maintenance":
                self.write_maintenance_passes(resumed.publish_result)
            return ""

        with mock.patch.object(resumed, "_run", side_effect=run), \
                mock.patch.object(resumed, "_repository_identity", return_value=HEAD), \
                mock.patch.object(resumed, "_maintenance_preflight", return_value={first_release["release_dir"]}), \
                contextlib.redirect_stdout(io.StringIO()):
            resumed.execute(resume=True)
        self.assertEqual(calls, ["resume-assets-validate", "shared-assets", "maintenance"])

    def test_tampered_resume_state_and_artifacts_fail_before_production_commands(self):
        cases = ("repository", "production-identity", "locale-order", "release-deleted", "release-symlink",
                 "release-json", "assets-manifest")
        for index, case in enumerate(cases):
            with self.subTest(case=case):
                root = self.output_root / ("tamper-%d" % index)
                root.mkdir()
                previous = self.output_root
                self.output_root = root
                fresh = self.workflow(["de-DE", "zh-CN"])
                state_path = self.prepare_state(fresh)
                data = json.loads(state_path.read_text(encoding="utf-8"))
                if case == "production-identity":
                    data["production_identity_sha256"] = "f" * 64
                    data["state_identity"] = MODULE.state_identity(data)
                    state_path.write_text(json.dumps(data), encoding="utf-8")
                    with self.assertRaises(MODULE.ReleaseBatchError):
                        self.load_state(state_path)
                    self.output_root = previous
                    continue
                if case == "locale-order":
                    data["selection"]["locales"].reverse()
                    data["state_identity"] = MODULE.state_identity(data)
                    state_path.write_text(json.dumps(data), encoding="utf-8")
                    with self.assertRaises(MODULE.ReleaseBatchError):
                        self.load_state(state_path)
                    self.output_root = previous
                    continue
                resumed = self.load_state(state_path)
                if case == "release-deleted":
                    release = pathlib.Path(data["releases"][0]["release_dir"])
                    release.rename(release.with_name(release.name + "-deleted"))
                elif case == "release-symlink":
                    release = pathlib.Path(data["releases"][0]["release_dir"])
                    real = release.with_name(release.name + "-real")
                    release.rename(real)
                    release.symlink_to(real, target_is_directory=True)
                elif case == "release-json":
                    release_json = pathlib.Path(data["releases"][0]["release_dir"]) / "release.json"
                    changed = json.loads(release_json.read_text(encoding="utf-8"))
                    changed["locale"] = "zh-CN"
                    release_json.write_text(json.dumps(changed), encoding="utf-8")
                    data["releases"][0]["release_json_sha256"] = MODULE.sha256_file(release_json)
                    data["state_identity"] = MODULE.state_identity(data)
                    state_path.write_text(json.dumps(data), encoding="utf-8")
                    resumed = self.load_state(state_path)
                elif case == "assets-manifest":
                    (fresh.assets_export / "SHA256SUMS").write_text("changed\n", encoding="utf-8")
                calls = []
                head = "f" * 40 if case == "repository" else HEAD
                with mock.patch.object(resumed, "_repository_identity", return_value=head), \
                        mock.patch.object(resumed, "_run", side_effect=lambda stage, *args, **kwargs: calls.append(stage)), \
                        self.assertRaises(MODULE.ReleaseBatchError):
                    resumed.execute(resume=True)
                self.assertFalse(any(stage in ("shared-assets", "maintenance") for stage in calls))
                self.output_root = previous

    def test_source_has_no_search_first_production_or_confirmation_tokens(self):
        source = (ROOT / "scripts" / "production-release-batch.py").read_text(encoding="utf-8").lower()
        for forbidden in ("indexnow", "search console", "first-production", "visual-pass", "purged"):
            self.assertNotIn(forbidden, source)


if __name__ == "__main__":
    unittest.main()
