#!/usr/bin/env python3
"""Top-level local publish and live Production maintenance orchestration."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import importlib.util
import json
import os
import pathlib
import re
import shlex
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
IDENTITY_PATH = ROOT / "production" / "identity.json"
STATE_SCHEMA = "go-tour-i18n/production-release-batch-state/v1"
STATE_STAGE = "post-publish"
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


def sha256_file(path):
    if not path.is_file() or path.is_symlink():
        raise ReleaseBatchError("resume-state", "identity input must be a real file: %s" % path)
    digest = hashlib.sha256()
    try:
        with path.open("rb") as source:
            for chunk in iter(lambda: source.read(1024 * 1024), b""):
                digest.update(chunk)
    except OSError as exc:
        raise ReleaseBatchError("resume-state", str(exc)) from exc
    return digest.hexdigest()


def state_identity(data):
    core = dict(data)
    core.pop("state_identity", None)
    encoded = json.dumps(core, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode("utf-8")
    return hashlib.sha256(encoded).hexdigest()


class ReleaseBatch:
    def __init__(self, output_root, all_live, locales, now=None):
        identity = IDENTITY.load_identity(IDENTITY_PATH)
        if all_live == bool(locales):
            raise ReleaseBatchError("preflight", "choose exactly one of --all-live or repeated --locale")
        selected = ([profile["locale"] for profile in identity["locales"]
                     if profile["production_state"] == "live"] if all_live else list(locales))
        profiles = self._select_profiles(identity, selected)
        path = pathlib.Path(output_root)
        if not path.is_dir() or path.is_symlink():
            raise ReleaseBatchError("preflight", "output root must be a real directory")
        output = path.resolve()
        self._require_outside_repository(output, "output root")
        clock = now or (lambda: dt.datetime.now(dt.timezone.utc))
        stamp = clock().astimezone(dt.timezone.utc).strftime("%Y%m%dT%H%M%SZ")
        stem = "go-tour-production-release-batch-%s-%d" % (stamp, os.getpid())
        self.output_root = output
        self.assets_export = output / ("go-tour-shared-assets-%s-%d" % (stamp, os.getpid()))
        self.state_path = output / (stem + ".state.json")
        for candidate, label in ((self.assets_export, "shared-assets export output"),
                                 (self.state_path, "resume state output")):
            if candidate.exists() or candidate.is_symlink():
                raise ReleaseBatchError("preflight", "%s already exists" % label)
        self.identity = identity
        self.profiles = profiles
        self.locales = selected
        self.selection_mode = "all-live" if all_live else "explicit"
        self.needs_shared_assets = any(profile["shared_assets_policy"] == "shared-cloudflare"
                                       for profile in profiles)
        self.publish_result = None
        self.state_data = None
        self.resume_available = False
        self.publish_status = "NOT_STARTED"
        self.maintenance_status = "NOT_STARTED"
        self.maintenance_started = False
        self.shared_summary = self._initial_shared_summary()

    @classmethod
    def from_state(cls, value):
        supplied = pathlib.Path(value)
        if not supplied.is_file() or supplied.is_symlink():
            raise ReleaseBatchError("resume-state", "resume state must be a real regular file")
        path = supplied.resolve()
        cls._require_outside_repository(path, "resume state")
        try:
            data = json.loads(path.read_text(encoding="utf-8"))
        except (OSError, ValueError) as exc:
            raise ReleaseBatchError("resume-state", "invalid resume state JSON: %s" % exc) from exc
        cls._validate_state_shape(data, path)
        identity = IDENTITY.load_identity(IDENTITY_PATH)
        if sha256_file(IDENTITY_PATH) != data["production_identity_sha256"]:
            raise ReleaseBatchError("resume-state", "production identity changed since publish")
        selected = data["selection"]["locales"]
        profiles = cls._select_profiles(identity, selected)
        current_shared = any(profile["shared_assets_policy"] == "shared-cloudflare" for profile in profiles)
        if current_shared != data["shared_assets"]["required"]:
            raise ReleaseBatchError("resume-state", "shared-assets requirement changed since publish")
        if data["selection"]["mode"] == "all-live":
            current_live = [profile["locale"] for profile in identity["locales"]
                            if profile["production_state"] == "live"]
            if current_live != selected:
                raise ReleaseBatchError("resume-state", "all-live locale order changed since publish")
        workflow = cls.__new__(cls)
        workflow.output_root = path.parent
        workflow.assets_export = pathlib.Path(data["shared_assets"]["export_dir"])
        workflow.state_path = path
        workflow.identity = identity
        workflow.profiles = profiles
        workflow.locales = list(selected)
        workflow.selection_mode = data["selection"]["mode"]
        workflow.needs_shared_assets = current_shared
        workflow.publish_result = {
            "repository_head": data["repository_head"],
            "releases": [{"locale": item["locale"], "published_at": item["published_at"],
                          "release_dir": item["release_dir"], "result": "PASS"}
                         for item in data["releases"]],
        }
        workflow.state_data = data
        workflow.resume_available = False
        workflow.publish_status = "PASS"
        workflow.maintenance_status = "NOT_STARTED"
        workflow.maintenance_started = False
        workflow.shared_summary = workflow._initial_shared_summary()
        return workflow

    @staticmethod
    def _require_outside_repository(path, label):
        try:
            path.relative_to(ROOT)
        except ValueError:
            return
        raise ReleaseBatchError("preflight", "%s must be outside the repository" % label)

    @staticmethod
    def _select_profiles(identity, selected):
        if not selected:
            raise ReleaseBatchError("preflight", "no live locales selected")
        if len(selected) != len(set(selected)) or any(type(value) is not str or not value for value in selected):
            raise ReleaseBatchError("preflight", "duplicate or invalid locale")
        profiles = []
        for locale in selected:
            matches = [profile for profile in identity["locales"] if profile["locale"] == locale]
            if len(matches) != 1:
                raise ReleaseBatchError("preflight", "unknown locale: %s" % locale)
            if matches[0]["production_state"] != "live":
                raise ReleaseBatchError("preflight", "locale is not live: %s" % locale)
            profiles.append(matches[0])
        return profiles

    def _initial_shared_summary(self):
        value = "NOT_STARTED" if self.needs_shared_assets else "SKIPPED"
        return {"export": "NOT_STARTED", "deploy": value, "purge": value, "verify": value}

    @staticmethod
    def _validate_state_shape(data, path):
        required = {"schema", "state_identity", "state_path", "stage", "publish_result",
                    "repository_head", "production_identity_sha256", "selection", "releases",
                    "shared_assets"}
        if type(data) is not dict or set(data) != required:
            raise ReleaseBatchError("resume-state", "resume state must contain exactly the v1 fields")
        if data.get("schema") != STATE_SCHEMA or data.get("stage") != STATE_STAGE or data.get("publish_result") != "PASS":
            raise ReleaseBatchError("resume-state", "unsupported resume state schema or stage")
        for key in ("state_identity", "repository_head", "production_identity_sha256"):
            pattern = r"(?:[0-9a-f]{40}|[0-9a-f]{64})" if key == "repository_head" else r"[0-9a-f]{64}"
            if type(data.get(key)) is not str or re.fullmatch(pattern, data[key]) is None:
                raise ReleaseBatchError("resume-state", "invalid resume state %s" % key)
        if data["state_identity"] != state_identity(data):
            raise ReleaseBatchError("resume-state", "resume state identity mismatch")
        if data.get("state_path") != str(path):
            raise ReleaseBatchError("resume-state", "resume state path identity mismatch")
        selection = data.get("selection")
        if (type(selection) is not dict or set(selection) != {"mode", "locales"} or
                selection.get("mode") not in ("all-live", "explicit") or
                type(selection.get("locales")) is not list):
            raise ReleaseBatchError("resume-state", "invalid locale selection identity")
        locales = selection["locales"]
        if not locales or len(locales) != len(set(locales)) or any(type(item) is not str or not item for item in locales):
            raise ReleaseBatchError("resume-state", "invalid or duplicate ordered locale set")
        releases = data.get("releases")
        release_keys = {"locale", "published_at", "release_dir", "release_json_sha256", "bundle_manifest_sha256"}
        if (type(releases) is not list or len(releases) != len(locales) or
                any(type(item) is not dict or set(item) != release_keys for item in releases) or
                [item.get("locale") for item in releases] != locales):
            raise ReleaseBatchError("resume-state", "release order does not match ordered locale set")
        for item in releases:
            if (any(type(item[key]) is not str for key in release_keys) or
                    re.fullmatch(r"[0-9a-f]{64}", item["release_json_sha256"]) is None or
                    re.fullmatch(r"[0-9a-f]{64}", item["bundle_manifest_sha256"]) is None or
                    re.fullmatch(r"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z", item["published_at"]) is None):
                raise ReleaseBatchError("resume-state", "invalid release identity fields")
            try:
                dt.datetime.strptime(item["published_at"], "%Y-%m-%dT%H:%M:%SZ")
            except ValueError as exc:
                raise ReleaseBatchError("resume-state", "invalid release published_at") from exc
        shared = data.get("shared_assets")
        if (type(shared) is not dict or set(shared) != {"required", "export_dir", "manifest_sha256"} or
                type(shared.get("required")) is not bool or type(shared.get("export_dir")) is not str or
                type(shared.get("manifest_sha256")) is not str or
                re.fullmatch(r"[0-9a-f]{64}", shared["manifest_sha256"]) is None):
            raise ReleaseBatchError("resume-state", "invalid shared-assets export identity")

    @staticmethod
    def _run(stage, command, timeout, capture=False):
        try:
            completed = subprocess.run([str(value) for value in command], stdin=subprocess.DEVNULL,
                                       stdout=subprocess.PIPE if capture else None,
                                       stderr=subprocess.PIPE if capture else None,
                                       text=capture, check=False, timeout=timeout)
        except (OSError, subprocess.SubprocessError) as exc:
            raise ReleaseBatchError(stage, str(exc)) from exc
        if completed.returncode:
            detail = completed.stderr.strip() if capture and completed.stderr else ""
            raise ReleaseBatchError(stage, "command failed with exit %d%s" %
                                    (completed.returncode, ": " + detail if detail else ""))
        return completed.stdout if capture else ""

    @classmethod
    def _repository_identity(cls):
        head = cls._run("resume-state", ["git", "rev-parse", "HEAD"], 30, capture=True).strip()
        dirty = cls._run("resume-state", ["git", "status", "--porcelain", "--untracked-files=normal"],
                         30, capture=True)
        if re.fullmatch(r"(?:[0-9a-f]{40}|[0-9a-f]{64})", head) is None:
            raise ReleaseBatchError("resume-state", "repository HEAD is not a full object ID")
        if dirty.strip():
            raise ReleaseBatchError("resume-state", "repository working tree must be clean before resume")
        return head

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
        releases = result.get("releases") if type(result) is dict else None
        if (type(result) is not dict or type(result.get("repository_head")) is not str or
                re.fullmatch(r"(?:[0-9a-f]{40}|[0-9a-f]{64})", result["repository_head"]) is None or
                type(releases) is not list or len(releases) != len(self.locales) or
                any(type(item) is not dict for item in releases) or
                [item.get("locale") for item in releases] != self.locales or
                any(item.get("result") != "PASS" or type(item.get("published_at")) is not str or
                    type(item.get("release_dir")) is not str for item in releases)):
            raise ReleaseBatchError("publish-summary", "publish-batch summary identity or result mismatch")
        for item in releases:
            release = pathlib.Path(item["release_dir"])
            if (not release.is_absolute() or not release.is_dir() or release.is_symlink() or
                    release.resolve().parent != self.output_root):
                raise ReleaseBatchError("publish-summary", "publish-batch release directory is invalid")
        self.publish_result = result
        self.publish_status = "PASS"

    def _release_state_item(self, item):
        release = pathlib.Path(item["release_dir"])
        try:
            manifest = json.loads((release / "release.json").read_text(encoding="utf-8"))
        except (OSError, ValueError) as exc:
            raise ReleaseBatchError("resume-state", "invalid release.json: %s: %s" % (release, exc)) from exc
        if (type(manifest) is not dict or manifest.get("locale") != item["locale"] or
                manifest.get("published_at") != item["published_at"]):
            raise ReleaseBatchError("resume-state", "release.json locale/published_at identity mismatch: %s" % release)
        head = self.publish_result["repository_head"]
        if not release.name.endswith("-%s-%s" % (item["locale"], head[:12])):
            raise ReleaseBatchError("resume-state", "release directory does not match publish HEAD/locale: %s" % release)
        return {
            "locale": item["locale"], "published_at": item["published_at"], "release_dir": str(release),
            "release_json_sha256": sha256_file(release / "release.json"),
            "bundle_manifest_sha256": sha256_file(release / "SHA256SUMS"),
        }

    def _write_state(self):
        head = self._repository_identity()
        if head != self.publish_result["repository_head"]:
            raise ReleaseBatchError("resume-state", "repository HEAD changed during batch publish")
        data = {
            "schema": STATE_SCHEMA, "state_path": str(self.state_path), "stage": STATE_STAGE,
            "publish_result": "PASS", "repository_head": head,
            "production_identity_sha256": sha256_file(IDENTITY_PATH),
            "selection": {"mode": self.selection_mode, "locales": list(self.locales)},
            "releases": [self._release_state_item(item) for item in self.publish_result["releases"]],
            "shared_assets": {"required": self.needs_shared_assets, "export_dir": str(self.assets_export),
                              "manifest_sha256": sha256_file(self.assets_export / "SHA256SUMS")},
        }
        data["state_identity"] = state_identity(data)
        self._validate_state_shape(data, self.state_path)
        temporary = self.state_path.with_name(self.state_path.name + ".tmp-%d" % os.getpid())
        try:
            temporary.write_text(json.dumps(data, ensure_ascii=False, sort_keys=True,
                                            separators=(",", ":")) + "\n", encoding="utf-8")
            os.chmod(temporary, 0o644)
            os.replace(temporary, self.state_path)
        except OSError as exc:
            raise ReleaseBatchError("resume-state", "write resume state: %s" % exc) from exc
        self.state_data = data
        self.resume_available = True
        print("[production-release-batch] post-publish resume state: %s" % self.state_path)

    def _validate_resume_state(self, allow_head_mismatch=False):
        if not self.state_path.is_file() or self.state_path.is_symlink():
            raise ReleaseBatchError("resume-state", "resume state was removed or replaced")
        try:
            current = json.loads(self.state_path.read_text(encoding="utf-8"))
        except (OSError, ValueError) as exc:
            raise ReleaseBatchError("resume-state", "resume state is unreadable: %s" % exc) from exc
        self._validate_state_shape(current, self.state_path)
        if current != self.state_data:
            raise ReleaseBatchError("resume-state", "resume state changed after it was selected")
        current_head = self._repository_identity()
        if not allow_head_mismatch and current_head != current["repository_head"]:
            raise ReleaseBatchError("resume-state", "repository HEAD does not match published batch")
        if sha256_file(IDENTITY_PATH) != current["production_identity_sha256"]:
            raise ReleaseBatchError("resume-state", "production identity changed since publish")
        export = pathlib.Path(current["shared_assets"]["export_dir"])
        if (not export.is_absolute() or not export.is_dir() or export.is_symlink() or
                export.resolve() != export or export.parent != self.output_root or export != self.assets_export):
            raise ReleaseBatchError("resume-state", "shared-assets export was removed, replaced, or moved")
        if sha256_file(export / "SHA256SUMS") != current["shared_assets"]["manifest_sha256"]:
            raise ReleaseBatchError("resume-state", "shared-assets manifest identity changed")
        for expected, item in zip(current["releases"], self.publish_result["releases"]):
            release = pathlib.Path(expected["release_dir"])
            if (not release.is_absolute() or not release.is_dir() or release.is_symlink() or
                    release.resolve() != release or release.parent != self.output_root or
                    item["release_dir"] != expected["release_dir"]):
                raise ReleaseBatchError("resume-state", "release directory was removed, replaced, or moved: %s" % release)
            if (sha256_file(release / "release.json") != expected["release_json_sha256"] or
                    sha256_file(release / "SHA256SUMS") != expected["bundle_manifest_sha256"]):
                raise ReleaseBatchError("resume-state", "release manifest identity changed: %s" % release)
            try:
                manifest = json.loads((release / "release.json").read_text(encoding="utf-8"))
            except (OSError, ValueError) as exc:
                raise ReleaseBatchError("resume-state", "invalid release.json: %s" % exc) from exc
            if (type(manifest) is not dict or manifest.get("locale") != expected["locale"] or
                    manifest.get("published_at") != expected["published_at"]):
                raise ReleaseBatchError("resume-state", "release.json locale/published_at identity mismatch: %s" % release)
        self._run("resume-assets-validate", ["go", "run", "-mod=readonly", "./cmd/tour-i18n",
                                             "assets", "validate", "--input", export], 600)
        self.shared_summary["export"] = "PASS"
        self.resume_available = True

    def _refresh_shared_summary(self):
        if not self.needs_shared_assets:
            return
        receipt = pathlib.Path(str(self.assets_export) + ".verification-receipt.json")
        if not receipt.is_file() or receipt.is_symlink():
            return
        try:
            data = json.loads(receipt.read_text(encoding="utf-8"))
        except (OSError, ValueError):
            self.shared_summary.update(deploy="UNKNOWN", purge="UNKNOWN", verify="UNKNOWN")
            return
        if (type(data) is not dict or data.get("schema") not in (
                "go-tour-i18n/shared-assets-production-receipt/v1",
                "go-tour-i18n/shared-assets-production-receipt/v2") or
                data.get("export_dir") != str(self.assets_export) or self.state_data is None or
                data.get("manifest_sha256") != self.state_data["shared_assets"]["manifest_sha256"]):
            self.shared_summary.update(deploy="UNKNOWN", purge="UNKNOWN", verify="UNKNOWN")
            return
        self.shared_summary["deploy"] = data.get("deployment_result", "UNKNOWN")
        if data["schema"].endswith("/v1"):
            self.shared_summary.update(purge="NOT_STARTED", verify="NOT_STARTED")
        else:
            self.shared_summary["purge"] = data.get("purge_result", "UNKNOWN")
            self.shared_summary["verify"] = data.get("verification_result", "UNKNOWN")

    def _read_shared_summary(self):
        self._refresh_shared_summary()
        values = (self.shared_summary["deploy"], self.shared_summary["purge"], self.shared_summary["verify"])
        if values not in (("DEPLOYED", "PASS", "PASS"), ("NO_CHANGES", "SKIPPED", "PASS")):
            raise ReleaseBatchError("shared-assets-summary", "shared-assets workflow has no complete v2 PASS receipt")

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
                                                    stages[stage].get("result") != "PASS" for stage in required)):
                raise ReleaseBatchError("maintenance-summary", "incomplete receipt: %s" % receipt_path)
            rows.append({"locale": item["locale"], "published_at": item["published_at"], "publish": "PASS",
                         "deploy": "PASS", "purge": "PASS", "machine": "PASS", "browser": "PASS",
                         "result": "SKIPPED" if item["release_dir"] in already_complete else "PASS"})
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

    def _refresh_maintenance_status(self):
        complete = 0
        seen = 0
        for item in self.publish_result["releases"]:
            release = pathlib.Path(item["release_dir"])
            receipt_path = release.parent / (release.name + ".maintenance-production-receipt.json")
            if not receipt_path.is_file() or receipt_path.is_symlink():
                continue
            seen += 1
            try:
                receipt = json.loads(receipt_path.read_text(encoding="utf-8"))
            except (OSError, ValueError):
                continue
            stages = receipt.get("stages") if type(receipt) is dict else None
            if (receipt.get("result") == "passed" and type(stages) is dict and
                    all(type(stages.get(stage)) is dict and stages[stage].get("result") == "PASS"
                        for stage in ("deploy", "purge", "machine", "browser"))):
                complete += 1
        if complete == len(self.publish_result["releases"]):
            self.maintenance_status = "COMPLETE"
        elif self.maintenance_started or seen:
            self.maintenance_status = "PARTIAL"
        else:
            self.maintenance_status = "NOT_STARTED"

    def execute(self, resume=False):
        if not resume:
            self._run("assets-export", ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "export",
                                        "--output", self.assets_export], 600)
            self.shared_summary["export"] = "PASS"
            self._run("assets-validate", ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "assets", "validate",
                                          "--input", self.assets_export], 600)
            try:
                self._publish()
            except ReleaseBatchError:
                self.publish_status = "FAILED"
                raise
            self._write_state()
        self._validate_resume_state()
        self._refresh_shared_summary()
        self._refresh_maintenance_status()
        already_complete = self._maintenance_preflight() or set()
        if self.needs_shared_assets:
            try:
                self._run("shared-assets", [ROOT / "scripts" / "shared-assets-production.sh", self.assets_export], 3600)
            finally:
                self._refresh_shared_summary()
            self._read_shared_summary()
        releases = [item["release_dir"] for item in self.publish_result["releases"]]
        self.maintenance_started = True
        try:
            self._run("maintenance", [ROOT / "scripts" / "maintenance-production-batch.sh"] + releases, 14400)
        finally:
            self._refresh_maintenance_status()
        rows = self._maintenance_rows(already_complete)
        self.maintenance_status = "COMPLETE"
        self.print_summary(rows)

    def execute_maintenance_recovery(self):
        self._validate_resume_state(allow_head_mismatch=True)
        self._refresh_shared_summary()
        if self.needs_shared_assets:
            self._read_shared_summary()
        self._refresh_maintenance_status()
        already_complete = self._maintenance_preflight() or set()
        releases = [item["release_dir"] for item in self.publish_result["releases"]]
        self.maintenance_started = True
        try:
            self._run("maintenance", [ROOT / "scripts" / "maintenance-production-batch.sh"] + releases, 14400)
        finally:
            self._refresh_maintenance_status()
        rows = self._maintenance_rows(already_complete)
        self.maintenance_status = "COMPLETE"
        self.print_summary(rows)

    def resume_command(self):
        return "scripts/production-release-batch.sh --resume %s" % shlex.quote(str(self.state_path))

    def maintenance_recovery_command(self):
        return "scripts/production-release-batch.sh --resume-maintenance %s" % shlex.quote(str(self.state_path))

    def print_failure(self, exc):
        self._refresh_shared_summary()
        if self.publish_result is not None:
            self._refresh_maintenance_status()
        print("\n[production-release-batch] FAILED", file=sys.stderr)
        print("stage: %s" % getattr(exc, "stage", "preflight"), file=sys.stderr)
        print("evidence: %s" % getattr(exc, "evidence", str(exc)), file=sys.stderr)
        print("publish: %s" % self.publish_status, file=sys.stderr)
        print("shared-assets:", file=sys.stderr)
        for stage in ("export", "deploy", "purge", "verify"):
            print("%-12s %s" % (stage, self.shared_summary[stage]), file=sys.stderr)
        print("locale maintenance: %s" % self.maintenance_status, file=sys.stderr)
        if self.resume_available:
            print("resume state: %s" % self.state_path, file=sys.stderr)
            print("resume command: %s" % self.resume_command(), file=sys.stderr)
            if getattr(exc, "stage", None) in ("maintenance", "maintenance-summary"):
                print("maintenance recovery command: %s" % self.maintenance_recovery_command(), file=sys.stderr)
        else:
            print("resume state: UNAVAILABLE", file=sys.stderr)

    def print_summary(self, rows):
        print("\nPRODUCTION RELEASE BATCH: PASS")
        print("repository_head: %s" % self.publish_result["repository_head"])
        print("publish: PASS")
        print("shared-assets:")
        for stage in ("export", "deploy", "purge", "verify"):
            print("%-12s %s" % (stage, self.shared_summary[stage]))
        print("locale maintenance: COMPLETE")
        print("resume state: %s" % self.state_path)
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
    parser.add_argument("--output-root")
    parser.add_argument("--resume")
    parser.add_argument("--resume-maintenance")
    args = parser.parse_args(argv)
    if args.resume and args.resume_maintenance:
        parser.error("--resume and --resume-maintenance are mutually exclusive")
    if args.resume or args.resume_maintenance:
        if args.all_live or args.locale or args.output_root is not None:
            parser.error("resume cannot be combined with --all-live, --locale, or --output-root")
    elif args.all_live == bool(args.locale):
        parser.error("choose exactly one of --all-live or repeated --locale")
    return args


def main(argv=None):
    args = parse_args(sys.argv[1:] if argv is None else argv)
    workflow = None
    try:
        if args.resume or args.resume_maintenance:
            workflow = ReleaseBatch.from_state(args.resume or args.resume_maintenance)
            if args.resume_maintenance:
                workflow.execute_maintenance_recovery()
            else:
                workflow.execute(resume=True)
        else:
            workflow = ReleaseBatch(args.output_root or "/tmp", args.all_live, args.locale)
            workflow.execute()
        return 0
    except (ReleaseBatchError, IDENTITY.IdentityError, OSError, ValueError, KeyError) as exc:
        if workflow is not None:
            workflow.print_failure(exc)
        else:
            print("[production-release-batch] FAILED", file=sys.stderr)
            print("stage: %s" % getattr(exc, "stage", "preflight"), file=sys.stderr)
            print("evidence: %s" % getattr(exc, "evidence", str(exc)), file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
