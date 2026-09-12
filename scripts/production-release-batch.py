#!/usr/bin/env python3
"""Top-level local publish and live Production maintenance orchestration."""

from __future__ import annotations

import argparse
import datetime as dt
import importlib.util
import json
import os
import pathlib
import re
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
IDENTITY_SPEC = importlib.util.spec_from_file_location("production_release_identity", ROOT / "scripts" / "production-identity.py")
IDENTITY = importlib.util.module_from_spec(IDENTITY_SPEC)
IDENTITY_SPEC.loader.exec_module(IDENTITY)
MAINTENANCE_SPEC = importlib.util.spec_from_file_location(
    "production_release_maintenance_batch", ROOT / "scripts" / "maintenance-production-batch.py")
MAINTENANCE = importlib.util.module_from_spec(MAINTENANCE_SPEC)
MAINTENANCE_SPEC.loader.exec_module(MAINTENANCE)


class ReleaseBatchError(RuntimeError):
    def __init__(self, stage, evidence):
        super().__init__(evidence)
        self.stage = stage
        self.evidence = evidence


class ReleaseBatch:
    def __init__(self, output_root, all_live, locales, now=None):
        identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        if all_live == bool(locales):
            raise ReleaseBatchError("preflight", "choose exactly one of --all-live or repeated --locale")
        selected = [profile["locale"] for profile in identity["locales"] if profile["production_state"] == "live"] if all_live else list(locales)
        if len(selected) != len(set(selected)):
            raise ReleaseBatchError("preflight", "duplicate locale")
        profiles = []
        for locale in selected:
            matches = [profile for profile in identity["locales"] if profile["locale"] == locale]
            if len(matches) != 1:
                raise ReleaseBatchError("preflight", "unknown locale: %s" % locale)
            if matches[0]["production_state"] != "live":
                raise ReleaseBatchError("preflight", "locale is not live: %s" % locale)
            profiles.append(matches[0])
        if not profiles:
            raise ReleaseBatchError("preflight", "no live locales selected")
        path = pathlib.Path(output_root)
        if not path.is_dir() or path.is_symlink():
            raise ReleaseBatchError("preflight", "output root must be a real directory")
        self.output_root = path.resolve()
        try:
            self.output_root.relative_to(ROOT)
        except ValueError:
            pass
        else:
            raise ReleaseBatchError("preflight", "output root must be outside the repository")
        self.identity = identity
        self.profiles = profiles
        self.locales = selected
        self.needs_shared_assets = any(profile["shared_assets_policy"] == "shared-cloudflare" for profile in profiles)
        clock = now or (lambda: dt.datetime.now(dt.timezone.utc))
        stamp = clock().astimezone(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        self.assets_export = self.output_root / ("go-tour-shared-assets-%s-%d" % (stamp, os.getpid()))
        if self.assets_export.exists() or self.assets_export.is_symlink():
            raise ReleaseBatchError("preflight", "shared-assets export output already exists")
        self.publish_result = None
        self.shared_summary = {"export": "-", "deploy": "SKIPPED", "purge": "SKIPPED", "verify": "SKIPPED"}

    @staticmethod
    def _run(stage, command, timeout, capture=False):
        try:
            completed = subprocess.run([str(value) for value in command], stdin=subprocess.DEVNULL,
                                       stdout=subprocess.PIPE if capture else None,
                                       text=capture, check=False, timeout=timeout)
        except (OSError, subprocess.SubprocessError) as exc:
            raise ReleaseBatchError(stage, str(exc)) from exc
        if completed.returncode:
            raise ReleaseBatchError(stage, "command failed with exit %d" % completed.returncode)
        return completed.stdout if capture else ""

    def _publish(self):
        command = ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "publish-batch",
                   "--output-root", self.output_root, "--json"]
        for locale in self.locales:
            command.extend(["--locale", locale])
        raw = self._run("publish", command, 7200, capture=True)
        try:
            result = json.loads(raw)
        except (TypeError, ValueError) as exc:
            raise ReleaseBatchError("publish-summary", "invalid publish-batch JSON: %s" % exc) from exc
        if type(result) is not dict:
            raise ReleaseBatchError("publish-summary", "publish-batch summary must be an object")
        releases = result.get("releases")
        if (type(result.get("repository_head")) is not str or
                re.fullmatch(r"[0-9a-f]{40,64}", result["repository_head"]) is None or
                type(releases) is not list or
                len(releases) != len(self.locales) or any(type(item) is not dict for item in releases) or
                [item.get("locale") for item in releases] != self.locales or
                any(item.get("result") != "PASS" or type(item.get("published_at")) is not str or
                    type(item.get("release_dir")) is not str for item in releases)):
            raise ReleaseBatchError("publish-summary", "publish-batch summary identity or result mismatch")
        for item in releases:
            release = pathlib.Path(item.get("release_dir", ""))
            if not release.is_absolute() or not release.is_dir() or release.is_symlink():
                raise ReleaseBatchError("publish-summary", "publish-batch release directory is invalid")
            if release.parent != self.output_root:
                raise ReleaseBatchError("publish-summary", "publish-batch release escaped output root")
        self.publish_result = result

    def _read_shared_summary(self):
        receipt = pathlib.Path(str(self.assets_export) + ".verification-receipt.json")
        try:
            data = json.loads(receipt.read_text(encoding="utf-8"))
        except (OSError, ValueError) as exc:
            raise ReleaseBatchError("shared-assets-summary", str(exc)) from exc
        deployment = data.get("deployment_result") if type(data) is dict else None
        purge = data.get("purge_result") if type(data) is dict else None
        if (type(data) is not dict or
                data.get("schema") != "go-tour-i18n/shared-assets-production-receipt/v2" or
                data.get("export_dir") != str(self.assets_export) or
                data.get("verification_result") != "PASS" or
                (deployment, purge) not in (("DEPLOYED", "PASS"), ("NO_CHANGES", "SKIPPED"))):
            raise ReleaseBatchError("shared-assets-summary", "shared-assets workflow has no complete v2 PASS receipt")
        self.shared_summary.update(deploy=deployment, purge=purge, verify="PASS")

    def _maintenance_rows(self, already_complete):
        rows = []
        for item in self.publish_result["releases"]:
            release = pathlib.Path(item["release_dir"])
            receipt_path = release.parent / (release.name + ".maintenance-production-receipt.json")
            try:
                receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
            except (OSError, ValueError) as exc:
                raise ReleaseBatchError("maintenance-summary", "%s: %s" % (receipt_path, exc)) from exc
            stages = receipt.get("stages") if type(receipt) is dict else None
            required = ("deploy", "purge", "machine", "browser")
            if (type(receipt) is not dict or receipt.get("result") != "passed" or
                    type(stages) is not dict or any(type(stages.get(stage)) is not dict or
                                                    stages[stage].get("result") != "PASS"
                                                    for stage in required)):
                raise ReleaseBatchError("maintenance-summary", "incomplete receipt: %s" % receipt_path)
            rows.append({
                "locale": item["locale"], "published_at": item["published_at"], "publish": "PASS",
                "deploy": "PASS", "purge": "PASS", "machine": "PASS", "browser": "PASS",
                "result": "SKIPPED" if item["release_dir"] in already_complete else "PASS",
            })
        return rows

    def _maintenance_preflight(self):
        releases = [item["release_dir"] for item in self.publish_result["releases"]]
        try:
            batch = MAINTENANCE.Batch(releases)
            batch.preflight()
        except (MAINTENANCE.BatchError, MAINTENANCE.CORE.MaintenanceProductionError,
                MAINTENANCE.CORE.IDENTITY.IdentityError) as exc:
            raise ReleaseBatchError("maintenance-preflight", str(exc)) from exc
        return {str(item.release_dir) for item in batch.items if item.receipt.get("result") == "passed"}

    def execute(self):
        self._run("assets-export", ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "export",
                                    "--output", self.assets_export], 600)
        self.shared_summary["export"] = "PASS"
        self._run("assets-validate", ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "validate",
                                      "--input", self.assets_export], 600)
        self._publish()
        # Reuse the existing maintenance batch's complete local and read-only
        # authority preflight before the first shared-assets/locale mutation.
        already_complete = self._maintenance_preflight() or set()
        if self.needs_shared_assets:
            self._run("shared-assets", [ROOT / "scripts" / "shared-assets-production.sh", self.assets_export], 3600)
            self._read_shared_summary()
        releases = [item["release_dir"] for item in self.publish_result["releases"]]
        self._run("maintenance", [ROOT / "scripts" / "maintenance-production-batch.sh"] + releases, 14400)
        rows = self._maintenance_rows(already_complete)
        self.print_summary(rows)

    def print_summary(self, rows):
        print("\nPRODUCTION RELEASE BATCH: PASS")
        print("repository_head: %s" % self.publish_result["repository_head"])
        print("shared-assets:")
        for stage in ("export", "deploy", "purge", "verify"):
            print("%-12s %s" % (stage, self.shared_summary[stage]))
        print("locale | published_at | publish | deploy | purge | machine | browser | result")
        for row in rows:
            print("{locale} | {published_at} | {publish} | {deploy} | {purge} | {machine} | {browser} | {result}".format(**row))
        passed = sum(row["result"] == "PASS" for row in rows)
        skipped = sum(row["result"] == "SKIPPED" for row in rows)
        print("PASS=%d FAILED=0 SKIPPED=%d" % (passed, skipped))


def parse_args(argv):
    parser = argparse.ArgumentParser()
    parser.add_argument("--all-live", action="store_true")
    parser.add_argument("--locale", action="append", default=[])
    parser.add_argument("--output-root", default="/tmp")
    return parser.parse_args(argv)


def main(argv=None):
    args = parse_args(sys.argv[1:] if argv is None else argv)
    try:
        ReleaseBatch(args.output_root, args.all_live, args.locale).execute()
        return 0
    except (ReleaseBatchError, IDENTITY.IdentityError, OSError, ValueError, KeyError) as exc:
        print("[production-release-batch] FAILED", file=sys.stderr)
        print("stage: %s" % getattr(exc, "stage", "preflight"), file=sys.stderr)
        print("evidence: %s" % getattr(exc, "evidence", str(exc)), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
