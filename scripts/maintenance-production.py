#!/usr/bin/env python3
"""Resumable orchestration for an already-live production locale.

This deliberately owns no deployment or acceptance implementation.  It binds a
release to its formal identity, invokes the established commands, and records
only the successful deployment mutation needed to resume after a human gate.
"""

from __future__ import annotations

import datetime as dt
import importlib.util
import json
import os
import pathlib
import re
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent.parent
IDENTITY_SPEC = importlib.util.spec_from_file_location(
    "production_identity", ROOT / "scripts" / "production-identity.py"
)
IDENTITY = importlib.util.module_from_spec(IDENTITY_SPEC)
IDENTITY_SPEC.loader.exec_module(IDENTITY)

RECEIPT_SCHEMA = "go-tour-i18n/maintenance-production-receipt/v1"
STAGE_LABELS = {
    "deploy": "deployment",
    "machine": "machine acceptance",
    "browser": "browser acceptance",
    "visual": "visual HUMAN gate",
}


class MaintenanceProductionError(RuntimeError):
    def __init__(self, stage: str, expected: str, actual: str, next_step: str):
        super().__init__(actual)
        self.stage = stage
        self.expected = expected
        self.actual = actual
        self.next_step = next_step


def utc_now() -> str:
    return dt.datetime.now(dt.timezone.utc).replace(microsecond=0).isoformat().replace("+00:00", "Z")


def release_locale(release_dir: pathlib.Path) -> str:
    try:
        release = json.loads((release_dir / "release.json").read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise MaintenanceProductionError("preflight", "valid release.json", str(exc), "重新执行 production publish") from exc
    locale = release.get("locale") if type(release) is dict else None
    if type(locale) is not str or not locale:
        raise MaintenanceProductionError("preflight", "release.json locale", repr(locale), "重新执行 production publish")
    return locale


def safe_release_name(release_dir: pathlib.Path) -> str:
    name = release_dir.name
    if not name.startswith("go-tour-release-"):
        raise MaintenanceProductionError("preflight", "go-tour-release-<safe-name>", name, "使用 publish 生成的正式目录")
    remote = name.removeprefix("go-tour-release-")
    if not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9._-]*", remote):
        raise MaintenanceProductionError("preflight", "safe release name", remote, "使用 publish 生成的正式目录")
    return remote


def purge_instructions(profile: dict) -> tuple[str, ...]:
    hostname = profile["production_hostname"]
    common = (
        "CDN HOSTNAME PURGE HUMAN GATE",
        f"locale: {profile['locale']}",
        f"production hostname: {hostname}",
        f"CDN: {profile['cdn']}",
    )
    if profile["cdn"] == "cloudflare":
        return common + (
            f"现在在 Cloudflare 对当前 hostname 执行 Custom Purge：{hostname}",
            "只能刷新该 hostname；不得使用 Purge Everything。",
        )
    if profile["cdn"] == "edgeone":
        return common + (
            f"现在按 EdgeOne 当前正式规则执行 hostname 缓存刷新：{hostname}",
            "只刷新当前 hostname；不要扩大为其他 hostname 的缓存操作。",
        )
    raise MaintenanceProductionError("preflight", "supported formal CDN", repr(profile["cdn"]), "检查 production identity")


class Orchestrator:
    def __init__(self, release_dir: pathlib.Path):
        if not release_dir.is_dir() or release_dir.is_symlink():
            raise MaintenanceProductionError("preflight", "real release directory", str(release_dir), "重新执行 production publish")
        self.release_dir = release_dir.resolve()
        self.release_name = safe_release_name(self.release_dir)
        self.locale = release_locale(self.release_dir)
        self.identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        profiles = [p for p in self.identity["locales"] if p["locale"] == self.locale]
        if len(profiles) != 1:
            raise MaintenanceProductionError("preflight", "one formal production identity", self.locale, "先在 production/identity.json 建立并审核 identity")
        self.profile = profiles[0]
        if self.profile["production_state"] != "live":
            raise MaintenanceProductionError(
                "preflight", "production_state=live", f"production_state={self.profile['production_state']}",
                "该 locale 尚属首次 production；请使用 scripts/first-production.sh <release-dir>",
            )
        self.receipt_path = self.release_dir.parent / f"{self.release_dir.name}.maintenance-production-receipt.json"
        self.receipt = self._load_or_new_receipt()

    def _new_receipt(self) -> dict:
        return {
            "schema": RECEIPT_SCHEMA,
            "locale": self.locale,
            "hostname": self.profile["production_hostname"],
            "cdn": self.profile["cdn"],
            "public_url": self.profile["production_public_url"],
            "release": self.release_name,
            "started_at": utc_now(),
            "completed_at": None,
            "cdn_purge_confirmed_at": None,
            "result": "running",
            "stages": {},
        }

    def _load_or_new_receipt(self) -> dict:
        base = self._new_receipt()
        if not self.receipt_path.exists():
            return base
        try:
            previous = json.loads(self.receipt_path.read_text(encoding="utf-8"))
        except (OSError, json.JSONDecodeError) as exc:
            raise MaintenanceProductionError("preflight", "valid existing receipt", str(self.receipt_path), "移走损坏 receipt 后重试") from exc
        expected = (RECEIPT_SCHEMA, self.locale, self.profile["production_hostname"], self.profile["cdn"], self.profile["production_public_url"], self.release_name)
        actual = tuple(previous.get(key) for key in ("schema", "locale", "hostname", "cdn", "public_url", "release"))
        if actual != expected:
            raise MaintenanceProductionError("preflight", repr(expected), repr(actual), "不要复用其他 locale、hostname 或 release 的 receipt")
        stages = previous.get("stages")
        if type(stages) is not dict or set(stages) - set(STAGE_LABELS):
            raise MaintenanceProductionError("preflight", "known receipt stages", repr(stages), "移走无效 receipt 并检查 production identity")
        for stage, value in stages.items():
            if type(value) is not dict or value.get("result") != "PASS" or type(value.get("completed_at")) is not str:
                raise MaintenanceProductionError("preflight", "valid passed receipt stage", repr({stage: value}), "移走无效 receipt 并检查 production identity")
        if previous.get("result") == "passed":
            if not all(stage in stages for stage in STAGE_LABELS) or type(previous.get("cdn_purge_confirmed_at")) is not str:
                raise MaintenanceProductionError("preflight", "complete passed receipt", repr(stages), "移走无效 receipt 并检查 production identity")
            print("[maintenance-production] 已有同一 release 的完整 PASS receipt；不会重复 deployment mutation。")
            return previous
        if previous.get("result") not in ("running", "failed"):
            raise MaintenanceProductionError("preflight", "running or failed resumable receipt", repr(previous.get("result")), "移走无效 receipt 并检查 production identity")
        base["started_at"] = previous.get("started_at") or base["started_at"]
        base["stages"] = previous["stages"]
        print(f"[maintenance-production] resume receipt 已识别：{self.receipt_path}")
        return base

    def write_receipt(self, result: str | None = None) -> None:
        if result is not None:
            self.receipt["result"] = result
            if result in ("passed", "failed"):
                self.receipt["completed_at"] = utc_now()
        temporary = self.receipt_path.with_name(self.receipt_path.name + f".tmp-{os.getpid()}")
        temporary.write_text(json.dumps(self.receipt, ensure_ascii=False, sort_keys=True, separators=(",", ":")) + "\n", encoding="utf-8")
        os.chmod(temporary, 0o644)
        os.replace(temporary, self.receipt_path)

    def stage_passed(self, stage: str) -> bool:
        value = self.receipt.get("stages", {}).get(stage)
        return type(value) is dict and value.get("result") == "PASS"

    def record(self, stage: str) -> None:
        self.receipt["stages"][stage] = {"completed_at": utc_now(), "result": "PASS"}
        self.write_receipt()
        print(f"[maintenance-production] {STAGE_LABELS[stage]}: PASS")

    def run_command(self, stage: str, command: list[object], timeout: int) -> None:
        try:
            completed = subprocess.run([str(value) for value in command], check=False, timeout=timeout)
        except (OSError, subprocess.TimeoutExpired) as exc:
            raise MaintenanceProductionError(stage, "command completed", str(exc), "检查本机工具和网络后重试") from exc
        if completed.returncode:
            raise MaintenanceProductionError(stage, "command exit 0", f"exit {completed.returncode}", "按该 stage 的输出检查后重试")

    def confirm_purge(self) -> None:
        for line in purge_instructions(self.profile):
            print(f"[maintenance-production] {line}")
        answer = input("完成上述 hostname purge 后输入 PURGED 继续：").strip()
        if answer != "PURGED":
            raise MaintenanceProductionError("cdn-purge", "explicit confirmation PURGED", repr(answer), "完成 hostname purge 后重新执行同一命令并输入 PURGED")
        # This receipt event is audit information only.  Every invocation asks
        # again; a receipt can never automatically satisfy a HUMAN GATE.
        self.receipt["cdn_purge_confirmed_at"] = utc_now()
        self.write_receipt()
        print("[maintenance-production] CDN hostname purge: HUMAN CONFIRMED")

    def confirm_visual(self) -> None:
        print("[maintenance-production] VISUAL HUMAN GATE")
        print("[maintenance-production] Desktop：确认 editor、广告区域和整体布局无明显异常。")
        print("[maintenance-production] Mobile：打开 /tour/moretypes/1，确认无整页横向 overflow、广告/footer 正常，并点击一次下一页。")
        answer = input("上述 visual acceptance 通过后输入 VISUAL-PASS：").strip()
        if answer != "VISUAL-PASS":
            raise MaintenanceProductionError("visual", "explicit confirmation VISUAL-PASS", repr(answer), "完成最小 visual acceptance 后重新执行同一命令")
        self.record("visual")

    def print_summary(self) -> None:
        print("\nMAINTENANCE PRODUCTION: PASS")
        print(f"release: {self.release_name}")
        print(f"locale: {self.locale}")
        print(f"hostname: {self.profile['production_hostname']}")
        print("CDN hostname purge: HUMAN CONFIRMED")
        print("machine acceptance: PASS")
        print("browser acceptance: PASS")
        print("visual HUMAN gate: PASS")
        print(f"receipt: {self.receipt_path}")

    def execute(self) -> None:
        self.write_receipt()
        if self.receipt.get("result") == "passed":
            self.print_summary()
            return
        if not self.stage_passed("deploy"):
            self.run_command("deploy", [ROOT / "scripts" / "deploy-production.sh", self.release_dir], 1800)
            self.record("deploy")
        else:
            print("[maintenance-production] deployment: RESUME（不会重复 deployment mutation）")
        self.confirm_purge()
        self.run_command("machine", [ROOT / "scripts" / "verify-production.sh", self.release_dir], 1800)
        self.record("machine")
        self.run_command("browser", [ROOT / "scripts" / "verify-production-browser.py", self.profile["production_public_url"], self.locale], 900)
        self.record("browser")
        self.confirm_visual()
        self.write_receipt("passed")
        self.print_summary()


def usage() -> None:
    print("usage: maintenance-production.sh <release-dir>", file=sys.stderr)


def main() -> int:
    if len(sys.argv) != 2:
        usage()
        return 2
    orchestrator = None
    try:
        orchestrator = Orchestrator(pathlib.Path(sys.argv[1]))
        orchestrator.execute()
        return 0
    except (MaintenanceProductionError, IDENTITY.IdentityError) as exc:
        if orchestrator is not None and orchestrator.receipt.get("result") != "passed":
            orchestrator.receipt["failure"] = {"stage": getattr(exc, "stage", "identity"), "completed_at": utc_now(), "result": "failed"}
            orchestrator.write_receipt("failed")
        print("\n[maintenance-production] FAILED", file=sys.stderr)
        print(f"stage: {getattr(exc, 'stage', 'identity')}", file=sys.stderr)
        print(f"expected: {getattr(exc, 'expected', 'valid formal identity')}", file=sys.stderr)
        print(f"actual: {getattr(exc, 'actual', str(exc))}", file=sys.stderr)
        print(f"下一步：{getattr(exc, 'next_step', '修复 production identity 后重试')}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
