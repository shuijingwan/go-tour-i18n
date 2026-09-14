#!/usr/bin/env python3
"""Fail-fast two-phase orchestration for multiple live maintenance releases."""

from __future__ import annotations

import importlib.util
import pathlib
import shlex
import subprocess
import sys
import time

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
        self.complete_at_start = [item.receipt.get("result") == "passed" for item in self.items]

    @staticmethod
    def run(command, timeout, progress=None):
        started = time.monotonic()
        if progress is not None:
            progress("START")
        try:
            completed = subprocess.run([str(value) for value in command], stdin=subprocess.DEVNULL,
                                       check=False, timeout=timeout)
        except (OSError, subprocess.TimeoutExpired) as exc:
            if progress is not None:
                progress("FAILED", time.monotonic() - started, str(exc))
            raise BatchError(str(exc)) from exc
        if completed.returncode:
            if progress is not None:
                progress("FAILED", time.monotonic() - started, "exit=%d" % completed.returncode)
            raise BatchError("command failed with exit %d" % completed.returncode)
        if progress is not None:
            progress("PASS", time.monotonic() - started)

    def progress(self, phase, index, stage, result, elapsed=None, detail=None):
        item = self.items[index]
        fields = ["[phase %s]" % phase, "locale=%s" % item.locale,
                  "%d/%d" % (index + 1, len(self.items)), "stage=%s" % stage, result]
        if elapsed is not None:
            fields.append("elapsed=%.1fs" % elapsed)
        if detail:
            fields.append(detail)
        print(" ".join(fields), flush=True)

    def phase_progress(self, phase, completed):
        print("[batch] phase=%s completed=%d/%d remaining=%d" %
              (phase, completed, len(self.items), len(self.items) - completed), flush=True)

    def preflight(self):
        # Every local receipt/identity/release check above and every remote CDN
        # authority check below completes before the first deploy/purge mutation.
        for index, item in enumerate(self.items):
            deploy = shlex.quote(str(ROOT / "scripts" / "deploy-production.sh"))
            shell = "source %s; validate_local_tools; release_name_from_path \"$1\" >/dev/null; validate_local_release \"$1\" >/dev/null" % deploy
            callback = lambda result, elapsed=None, detail=None, i=index: self.progress(
                "preflight", i, "local", result, elapsed, detail)
            self.run(["bash", "-c", shell, "maintenance-batch-preflight", item.release_dir], 120, callback)
        for index, item in enumerate(self.items):
            callback = lambda result, elapsed=None, detail=None, i=index: self.progress(
                "preflight", i, "cdn-authority", result, elapsed, detail)
            self.run([ROOT / "scripts" / "production-cdn.py", "preflight", "--locale", item.locale], 300, callback)

    def refresh_row(self, index):
        item = self.items[index]
        receipt = item.receipt
        stages = receipt.get("stages", {})
        for stage in ("deploy", "purge", "machine", "browser"):
            value = stages.get(stage)
            self.rows[index][stage] = "PASS" if type(value) is dict and value.get("result") == "PASS" else "-"
        return receipt

    def fail_item(self, index, phase, exc):
        item = self.items[index]
        item.fail(exc)
        self.refresh_row(index)
        self.rows[index]["result"] = "%s_FAILED" % phase
        stage = getattr(exc, "stage", phase.lower())
        return BatchError("locale=%s stage=%s evidence=%s (%s)" % (item.locale, stage, item.receipt_path, exc))

    def block_remaining(self, start, result):
        for index in range(start, len(self.items)):
            self.refresh_row(index)
            if self.complete_at_start[index]:
                self.rows[index]["result"] = "SKIPPED"
            elif self.rows[index]["result"] in ("PENDING", "PENDING_ACCEPTANCE"):
                self.rows[index]["result"] = result

    def execute(self):
        self.preflight()

        for index, item in enumerate(self.items):
            item.set_progress_context("A", index + 1, len(self.items))
            self.refresh_row(index)
            if self.complete_at_start[index]:
                self.rows[index]["result"] = "SKIPPED"
                self.progress("A", index, "mutation", "SKIPPED")
                self.phase_progress("A", index + 1)
                continue
            if not all(item.stage_passed(stage) for stage in ("deploy", "purge")):
                item.begin()
            try:
                item.execute_mutation()
            except CORE.MaintenanceProductionError as exc:
                failed = self.fail_item(index, "MUTATION", exc)
                self.block_remaining(index + 1, "BLOCKED_PHASE_A")
                self.print_summary()
                raise failed
            self.refresh_row(index)
            self.rows[index]["result"] = "PENDING_ACCEPTANCE"
            self.phase_progress("A", index + 1)

        for index, item in enumerate(self.items):
            item.set_progress_context("B", index + 1, len(self.items))
            if self.complete_at_start[index]:
                self.rows[index]["result"] = "SKIPPED"
                self.progress("B", index, "acceptance", "SKIPPED")
                self.phase_progress("B", index + 1)
                continue
            item.begin()
            try:
                item.execute_acceptance()
                receipt = self.refresh_row(index)
                if receipt.get("result") != "passed":
                    raise CORE.MaintenanceProductionError(
                        "acceptance", "complete PASS receipt", repr(receipt), "检查 acceptance stage 输出后重试"
                    )
                self.rows[index]["result"] = "PASS"
                self.phase_progress("B", index + 1)
            except CORE.MaintenanceProductionError as exc:
                failed = self.fail_item(index, "ACCEPTANCE", exc)
                self.block_remaining(index + 1, "BLOCKED_PHASE_B")
                self.print_summary()
                raise failed
        self.print_summary()

    def print_summary(self):
        print("\nlocale | deploy | purge | machine | browser | result", flush=True)
        for row in self.rows:
            print("{locale} | {deploy} | {purge} | {machine} | {browser} | {result}".format(**row), flush=True)
        passed = sum(row["result"] == "PASS" for row in self.rows)
        failed = sum(row["result"] in ("MUTATION_FAILED", "ACCEPTANCE_FAILED") for row in self.rows)
        skipped = sum(row["result"] == "SKIPPED" for row in self.rows)
        pending = len(self.rows) - passed - failed - skipped
        print("PASS=%d FAILED=%d SKIPPED=%d PENDING=%d" % (passed, failed, skipped, pending), flush=True)


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
