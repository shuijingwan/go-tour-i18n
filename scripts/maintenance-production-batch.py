#!/usr/bin/env python3
"""Fail-fast serial orchestration for multiple live maintenance releases."""

from __future__ import annotations

import importlib.util
import json
import pathlib
import shlex
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location("maintenance_production_batch_core", ROOT / "scripts" / "maintenance-production.py")
CORE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(CORE)


class BatchError(RuntimeError):
    pass


class Batch:
    def __init__(self, release_dirs):
        if not release_dirs:
            raise BatchError("at least one release directory is required")
        self.items = [CORE.Orchestrator(pathlib.Path(value)) for value in release_dirs]
        locales = [item.locale for item in self.items]
        if len(locales) != len(set(locales)):
            raise BatchError("duplicate locale in batch")
        self.rows = [{"locale": item.locale, "deploy": "-", "purge": "-", "machine": "-", "browser": "-", "result": "PENDING"} for item in self.items]

    @staticmethod
    def run(command, timeout):
        try:
            completed = subprocess.run([str(value) for value in command], stdin=subprocess.DEVNULL,
                                       check=False, timeout=timeout)
        except (OSError, subprocess.TimeoutExpired) as exc:
            raise BatchError(str(exc)) from exc
        if completed.returncode:
            raise BatchError("command failed with exit %d" % completed.returncode)

    def preflight(self):
        # Every local receipt/identity/release check above and every remote CDN
        # authority check below completes before the first deploy/purge mutation.
        for item in self.items:
            deploy = shlex.quote(str(ROOT / "scripts" / "deploy-production.sh"))
            shell = "source %s; validate_local_tools; release_name_from_path \"$1\" >/dev/null; validate_local_release \"$1\" >/dev/null" % deploy
            self.run(["bash", "-c", shell, "maintenance-batch-preflight", item.release_dir], 120)
        for item in self.items:
            self.run([ROOT / "scripts" / "production-cdn.py", "preflight", "--locale", item.locale], 300)

    def refresh_row(self, index):
        item = self.items[index]
        try:
            refreshed = CORE.Orchestrator(item.release_dir)
            receipt = refreshed.receipt
        except Exception:
            receipt = item.receipt
        stages = receipt.get("stages", {})
        for stage in ("deploy", "purge", "machine", "browser"):
            self.rows[index][stage] = "PASS" if stage in stages else "-"
        return receipt

    def execute(self):
        self.preflight()
        failed = None
        for index, item in enumerate(self.items):
            if failed is not None:
                self.rows[index]["result"] = "SKIPPED"
                continue
            if item.receipt.get("result") == "passed":
                self.refresh_row(index)
                self.rows[index]["result"] = "SKIPPED"
                continue
            try:
                self.run([ROOT / "scripts" / "maintenance-production.sh", item.release_dir], 3600)
                receipt = self.refresh_row(index)
                if receipt.get("result") != "passed":
                    raise BatchError("maintenance returned without a complete PASS receipt")
                self.rows[index]["result"] = "PASS"
            except BatchError as exc:
                self.refresh_row(index)
                self.rows[index]["result"] = "FAILED"
                try:
                    raw_receipt = json.loads(item.receipt_path.read_text(encoding="utf-8"))
                except (OSError, ValueError):
                    raw_receipt = {}
                failure = raw_receipt.get("failure") if type(raw_receipt) is dict else None
                stage = failure.get("stage", "maintenance") if type(failure) is dict else "maintenance"
                failed = (item.locale, stage, "%s (%s)" % (item.receipt_path, exc))
        self.print_summary()
        if failed is not None:
            locale, stage, evidence = failed
            raise BatchError("locale=%s stage=%s evidence=%s" % (locale, stage, evidence))

    def print_summary(self):
        print("\nlocale | deploy | purge | machine | browser | result")
        for row in self.rows:
            print("{locale} | {deploy} | {purge} | {machine} | {browser} | {result}".format(**row))
        totals = {name: sum(row["result"] == name for row in self.rows) for name in ("PASS", "FAILED", "SKIPPED")}
        print("PASS=%d FAILED=%d SKIPPED=%d" % (totals["PASS"], totals["FAILED"], totals["SKIPPED"]))


def main():
    try:
        batch = Batch(sys.argv[1:])
        batch.execute()
        return 0
    except (BatchError, CORE.MaintenanceProductionError, CORE.IDENTITY.IdentityError) as exc:
        print("[maintenance-production-batch] FAILED: %s" % exc, file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
