#!/usr/bin/env python3

import hashlib
import importlib.util
import json
import pathlib
import subprocess
import tempfile
import unittest
from unittest import mock


ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location("shared_assets_production", ROOT / "scripts" / "shared-assets-production.py")
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class SharedAssetsProductionTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.export = pathlib.Path(self.temp.name) / "formal-export"
        self.export.mkdir()
        (self.export / "SHA256SUMS").write_text("formal manifest\n", encoding="utf-8")
        self.receipt = pathlib.Path(str(self.export) + ".verification-receipt.json")

    def tearDown(self):
        self.temp.cleanup()

    def write_v1(self, result, changed):
        digest = hashlib.sha256((self.export / "SHA256SUMS").read_bytes()).hexdigest()
        self.receipt.write_text(json.dumps({
            "schema": MODULE.CDN.SHARED_ASSETS_RECEIPT_V1,
            "export_dir": str(self.export),
            "manifest_sha256": digest,
            "deployment_result": result,
            "production_base_url": "https://assets-go-dev.shuijingwanwq.com",
            "changed_paths": changed,
            "boundary_paths": MODULE.CDN.SHARED_ASSETS_BOUNDARY_PATHS,
        }), encoding="utf-8")

    def successful_run(self, stage, command, timeout):
        if stage == "deploy":
            self.write_v1("DEPLOYED", ["SHA256SUMS", "tour/static/css/app.css"])
        if stage == "purge":
            receipt = json.loads(self.receipt.read_text(encoding="utf-8"))
            receipt["purge_result"] = "PASS"
            self.receipt.write_text(json.dumps(receipt), encoding="utf-8")

    def test_deployed_purges_then_verifies_and_records_pass(self):
        workflow = MODULE.Workflow(self.export)
        calls = []
        def run(stage, command, timeout):
            calls.append((stage, [str(value) for value in command]))
            self.successful_run(stage, command, timeout)
        with mock.patch.object(MODULE.Workflow, "_run", side_effect=run):
            workflow.execute()
        self.assertEqual([stage for stage, _ in calls], ["cdn-preflight", "deploy", "purge", "verify"])
        purge = next(command for stage, command in calls if stage == "purge")
        self.assertEqual(purge[-2:], ["--receipt", str(self.receipt)])
        receipt = json.loads(self.receipt.read_text(encoding="utf-8"))
        self.assertEqual((receipt["purge_result"], receipt["verification_result"]), ("PASS", "PASS"))
        self.assertNotIn("token", json.dumps(receipt).lower())

    def test_no_changes_skips_purge_but_still_verifies(self):
        workflow = MODULE.Workflow(self.export)
        calls = []
        def run(stage, command, timeout):
            calls.append(stage)
            if stage == "deploy":
                self.write_v1("NO_CHANGES", [])
        with mock.patch.object(MODULE.Workflow, "_run", side_effect=run):
            workflow.execute()
        self.assertEqual(calls, ["cdn-preflight", "deploy", "verify"])
        receipt = json.loads(self.receipt.read_text(encoding="utf-8"))
        self.assertEqual((receipt["purge_result"], receipt["verification_result"]), ("SKIPPED", "PASS"))

    def test_purge_failure_preserves_original_paths_for_resume(self):
        workflow = MODULE.Workflow(self.export)
        calls = []
        def first(stage, command, timeout):
            calls.append(stage)
            if stage == "deploy":
                self.write_v1("DEPLOYED", ["SHA256SUMS", "tour/static/css/app.css"])
            if stage == "purge":
                raise MODULE.SharedAssetsProductionError("purge", "definite failure")
        with mock.patch.object(MODULE.Workflow, "_run", side_effect=first), self.assertRaises(MODULE.SharedAssetsProductionError):
            workflow.execute()
        pending = json.loads(self.receipt.read_text(encoding="utf-8"))
        self.assertEqual(pending["changed_paths"], ["SHA256SUMS", "tour/static/css/app.css"])
        self.assertEqual(pending["purge_result"], "PENDING")

        resumed = MODULE.Workflow(self.export)
        resumed_calls = []
        def resume(stage, command, timeout):
            resumed_calls.append(stage)
            self.successful_run(stage, command, timeout)
        with mock.patch.object(MODULE.Workflow, "_run", side_effect=resume):
            resumed.execute()
        self.assertEqual(resumed_calls, ["cdn-preflight", "purge", "verify"])
        self.assertEqual(json.loads(self.receipt.read_text(encoding="utf-8"))["changed_paths"],
                         ["SHA256SUMS", "tour/static/css/app.css"])

    def test_verify_failure_retries_without_repeating_purge(self):
        workflow = MODULE.Workflow(self.export)
        def first(stage, command, timeout):
            if stage == "deploy":
                self.write_v1("DEPLOYED", ["tour/static/css/app.css"])
            if stage == "purge":
                self.successful_run(stage, command, timeout)
            if stage == "verify":
                raise MODULE.SharedAssetsProductionError("verify", "public SHA mismatch")
        with mock.patch.object(MODULE.Workflow, "_run", side_effect=first), self.assertRaises(MODULE.SharedAssetsProductionError):
            workflow.execute()
        pending = json.loads(self.receipt.read_text(encoding="utf-8"))
        self.assertEqual((pending["purge_result"], pending["verification_result"]), ("PASS", "PENDING"))

        resumed = MODULE.Workflow(self.export)
        calls = []
        with mock.patch.object(MODULE.Workflow, "_run", side_effect=lambda stage, command, timeout: calls.append(stage)):
            resumed.execute()
        self.assertEqual(calls, ["verify"])

    def test_historical_v1_receipt_is_mechanically_migrated(self):
        self.write_v1("DEPLOYED", ["SHA256SUMS"])
        workflow = MODULE.Workflow(self.export)
        migrated = json.loads(self.receipt.read_text(encoding="utf-8"))
        self.assertEqual(migrated["schema"], MODULE.CDN.SHARED_ASSETS_RECEIPT_V2)
        self.assertEqual((migrated["purge_result"], migrated["verification_result"]), ("PENDING", "PENDING"))
        calls = []
        def migrate(stage, command, timeout):
            calls.append(stage)
            self.successful_run(stage, command, timeout)
        with mock.patch.object(MODULE.Workflow, "_run", side_effect=migrate):
            workflow.execute()
        self.assertEqual(calls, ["cdn-preflight", "purge", "verify"])

    def test_uncertain_attempt_is_never_retried(self):
        self.write_v1("DEPLOYED", ["SHA256SUMS"])
        workflow = MODULE.Workflow(self.export)
        workflow.receipt["purge_result"] = "ATTEMPTED"
        workflow._write()
        with mock.patch.object(MODULE.Workflow, "_run") as run, \
                self.assertRaisesRegex(MODULE.SharedAssetsProductionError, "refusing duplicate"):
            MODULE.Workflow(self.export)
        run.assert_not_called()

    def test_complete_v2_pass_performs_no_deploy_purge_or_verify(self):
        self.write_v1("DEPLOYED", ["SHA256SUMS"])
        workflow = MODULE.Workflow(self.export)
        workflow.receipt["purge_result"] = "PASS"
        workflow.receipt["verification_result"] = "PASS"
        workflow._write()
        resumed = MODULE.Workflow(self.export)
        with mock.patch.object(MODULE.Workflow, "_run") as run:
            resumed.execute()
        run.assert_not_called()

    def test_real_runner_never_reads_stdin(self):
        with mock.patch.object(MODULE.subprocess, "run", return_value=subprocess.CompletedProcess([], 0)) as run:
            MODULE.Workflow._run("test", ["true"], 1)
        self.assertIs(run.call_args.kwargs["stdin"], subprocess.DEVNULL)


if __name__ == "__main__":
    unittest.main()
