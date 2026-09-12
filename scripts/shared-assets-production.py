#!/usr/bin/env python3
"""Resumable deploy, exact-URL purge, and verification for shared assets."""

from __future__ import annotations

import importlib.util
import json
import os
import pathlib
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location("production_cdn_shared_assets", ROOT / "scripts" / "production-cdn.py")
CDN = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CDN)


class SharedAssetsProductionError(RuntimeError):
    def __init__(self, stage, evidence):
        super().__init__(evidence)
        self.stage = stage
        self.evidence = evidence


class Workflow:
    def __init__(self, export_dir):
        path = pathlib.Path(export_dir)
        if not path.is_dir() or path.is_symlink():
            raise SharedAssetsProductionError("preflight", "export must be a real directory")
        self.export_dir = path.resolve()
        self.receipt_path = pathlib.Path(str(self.export_dir) + ".verification-receipt.json")
        self.identity = CDN._load_identity()
        self.shared = self.identity["shared"]
        self.receipt = None
        if self.receipt_path.exists():
            self.receipt, _ = CDN.load_shared_assets_receipt(self.receipt_path, self.shared)
            if self.receipt["schema"] == CDN.SHARED_ASSETS_RECEIPT_V1:
                self._upgrade_receipt()
            if self.receipt["purge_result"] == "ATTEMPTED":
                raise SharedAssetsProductionError(
                    "purge", "previous exact-URL purge attempt is uncertain; refusing duplicate mutation")

    @staticmethod
    def _run(stage, command, timeout):
        try:
            completed = subprocess.run([str(value) for value in command], stdin=subprocess.DEVNULL,
                                       check=False, timeout=timeout)
        except (OSError, subprocess.SubprocessError) as exc:
            raise SharedAssetsProductionError(stage, str(exc)) from exc
        if completed.returncode:
            raise SharedAssetsProductionError(stage, "command failed with exit %d" % completed.returncode)

    def _write(self):
        temporary = self.receipt_path.with_name(self.receipt_path.name + ".tmp-%d" % os.getpid())
        temporary.write_text(json.dumps(self.receipt, ensure_ascii=False, sort_keys=True,
                                        separators=(",", ":")) + "\n", encoding="utf-8")
        os.chmod(temporary, 0o644)
        os.replace(temporary, self.receipt_path)
        self.receipt, _ = CDN.load_shared_assets_receipt(self.receipt_path, self.shared)

    def _upgrade_receipt(self):
        self.receipt["schema"] = CDN.SHARED_ASSETS_RECEIPT_V2
        self.receipt["purge_result"] = "PENDING" if self.receipt["deployment_result"] == "DEPLOYED" else "SKIPPED"
        self.receipt["verification_result"] = "PENDING"
        self._write()

    def execute(self):
        preflight_complete = False
        if self.receipt is None:
            self._run("cdn-preflight", [ROOT / "scripts" / "production-cdn.py", "shared-assets-preflight"], 300)
            preflight_complete = True
            self._run("deploy", [ROOT / "scripts" / "deploy-shared-assets.sh", self.export_dir], 1800)
            try:
                self.receipt, _ = CDN.load_shared_assets_receipt(self.receipt_path, self.shared)
            except CDN.CDNError as exc:
                raise SharedAssetsProductionError("deploy-receipt", str(exc)) from exc
            self._upgrade_receipt()
        if self.receipt["verification_result"] == "PASS":
            self.print_summary()
            return
        if self.receipt["purge_result"] == "PENDING":
            if not preflight_complete:
                self._run("cdn-preflight", [ROOT / "scripts" / "production-cdn.py", "shared-assets-preflight"], 300)
            self._run("purge", [ROOT / "scripts" / "production-cdn.py", "purge-shared-assets",
                                "--receipt", self.receipt_path], 300)
            self.receipt, _ = CDN.load_shared_assets_receipt(self.receipt_path, self.shared)
            if self.receipt["purge_result"] != "PASS":
                raise SharedAssetsProductionError("purge", "purge command returned without a PASS receipt")
        self._run("verify", [ROOT / "scripts" / "verify-shared-assets-production.sh", self.receipt_path], 1800)
        self.receipt["verification_result"] = "PASS"
        self._write()
        self.print_summary()

    def print_summary(self):
        print("\nSHARED ASSETS PRODUCTION: PASS")
        print("export: %s" % self.export_dir)
        print("deploy: %s" % self.receipt["deployment_result"])
        print("purge: %s" % self.receipt["purge_result"])
        print("verify: %s" % self.receipt["verification_result"])
        print("receipt: %s" % self.receipt_path)


def main(argv=None):
    argv = list(sys.argv[1:] if argv is None else argv)
    if len(argv) != 1:
        print("usage: shared-assets-production.sh <assets-export-dir>", file=sys.stderr)
        return 2
    try:
        Workflow(argv[0]).execute()
        return 0
    except (SharedAssetsProductionError, CDN.CDNError, OSError, ValueError, KeyError) as exc:
        print("[shared-assets-production] FAILED", file=sys.stderr)
        print("stage: %s" % getattr(exc, "stage", "preflight"), file=sys.stderr)
        print("evidence: %s" % getattr(exc, "evidence", str(exc)), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
