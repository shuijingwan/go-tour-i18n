#!/usr/bin/env python3
"""Strict reader for the repository production identity.

The shell interface deliberately emits ordered values, not shell source. Callers
must read the tab-separated fields without eval.
"""

import argparse
import ipaddress
import json
import pathlib
import re
import sys
import urllib.parse


SCHEMA = "go-tour-i18n/production-identity/v1"
LOCALE_FIELDS = (
    "locale", "production_state", "production_hostname", "cdn", "origin_ssh_alias", "origin_ip",
    "data_root", "releases_root", "current", "deployment_lock",
    "systemd_service", "service_user", "loopback_port",
    "localhost_health_url", "environment_file", "nginx_vhost_path",
    "tls_certificate_path", "tls_key_path", "playground_allowed_origin",
    "shared_assets_policy", "production_public_url", "cache_header",
)
SHARED_FIELDS = (
    "aliyun_ssh_alias", "zgocloud_ssh_alias", "origin_ip",
    "playground_vhost_path", "playground_public_origin",
    "cloudflare_zone_name", "cloudflare_secret_file", "nginx_test_command",
    "nginx_reload_command", "shared_assets_origin_root",
    "shared_assets_public_origin",
)
ABSOLUTE_FIELDS = {
    "data_root", "releases_root", "current", "deployment_lock",
    "environment_file", "nginx_vhost_path", "tls_certificate_path",
    "tls_key_path", "playground_vhost_path", "cloudflare_secret_file",
    "shared_assets_origin_root",
}
UNIQUE_FIELDS = (
    "locale", "production_hostname", "data_root", "releases_root", "current",
    "deployment_lock", "systemd_service", "loopback_port", "nginx_vhost_path",
    "tls_certificate_path", "tls_key_path", "playground_allowed_origin",
    "production_public_url",
)


class IdentityError(ValueError):
    pass


def fail(message):
    raise IdentityError(message)


def exact_keys(obj, fields, context):
    if type(obj) is not dict:
        fail(f"{context} must be an object")
    missing = sorted(set(fields) - set(obj))
    extra = sorted(set(obj) - set(fields))
    if missing or extra:
        fail(f"{context} keys invalid: missing={missing} extra={extra}")


def safe_text(value, context):
    if type(value) is not str or not value or any(c in value for c in "\0\r\n\t"):
        fail(f"{context} must be a non-empty single-line string")
    return value


def https_url(value, context, trailing_slash=None):
    safe_text(value, context)
    parsed = urllib.parse.urlsplit(value)
    if parsed.scheme != "https" or not parsed.hostname or parsed.username or parsed.password:
        fail(f"{context} must be an https URL without credentials")
    if parsed.query or parsed.fragment:
        fail(f"{context} must not contain query or fragment")
    if trailing_slash is True and parsed.path != "/":
        fail(f"{context} must end at the origin root")
    if trailing_slash is False and parsed.path not in ("", "/"):
        fail(f"{context} must be an origin URL")
    return parsed


def validate_path(value, context):
    safe_text(value, context)
    path = pathlib.PurePosixPath(value)
    if not path.is_absolute() or ".." in path.parts or str(path) != value:
        fail(f"{context} must be a normalized absolute path")


def load_identity(path):
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"cannot read identity: {exc}")
    exact_keys(data, ("schema", "shared", "locales"), "identity")
    if data["schema"] != SCHEMA:
        fail(f"unsupported identity schema: {data['schema']!r}")
    exact_keys(data["shared"], SHARED_FIELDS, "shared")
    for field in SHARED_FIELDS:
        safe_text(data["shared"][field], f"shared.{field}")
        if field in ABSOLUTE_FIELDS:
            validate_path(data["shared"][field], f"shared.{field}")
    try:
        ipaddress.ip_address(data["shared"]["origin_ip"])
    except ValueError:
        fail("shared.origin_ip must be an IP address")
    https_url(data["shared"]["playground_public_origin"], "shared.playground_public_origin")
    https_url(data["shared"]["shared_assets_public_origin"], "shared.shared_assets_public_origin")
    if not re.fullmatch(r"[a-z0-9.-]+", data["shared"]["cloudflare_zone_name"]):
        fail("shared.cloudflare_zone_name is invalid")
    if type(data["locales"]) is not list or not data["locales"]:
        fail("locales must be a non-empty array")
    for index, profile in enumerate(data["locales"]):
        context = f"locales[{index}]"
        exact_keys(profile, LOCALE_FIELDS, context)
        for field in LOCALE_FIELDS:
            value = profile[field]
            if field == "loopback_port":
                if type(value) is not int or not 1024 <= value <= 65535:
                    fail(f"{context}.loopback_port must be an unprivileged TCP port")
            else:
                safe_text(value, f"{context}.{field}")
            if field in ABSOLUTE_FIELDS:
                validate_path(value, f"{context}.{field}")
        if not re.fullmatch(r"[a-z]{2,3}-[A-Z]{2}", profile["locale"]):
            fail(f"{context}.locale is not canonical")
        if profile["production_state"] not in ("first-production", "live"):
            fail(f"{context}.production_state is unsupported")
        if not re.fullmatch(r"[a-z0-9.-]+", profile["production_hostname"]):
            fail(f"{context}.production_hostname is invalid")
        if profile["cdn"] not in ("cloudflare", "edgeone"):
            fail(f"{context}.cdn is unsupported")
        if profile["shared_assets_policy"] not in ("same-origin", "shared-cloudflare"):
            fail(f"{context}.shared_assets_policy is unsupported")
        try:
            ipaddress.ip_address(profile["origin_ip"])
        except ValueError:
            fail(f"{context}.origin_ip must be an IP address")
        if profile["origin_ip"] != data["shared"]["origin_ip"]:
            fail(f"{context}.origin_ip conflicts with shared.origin_ip")
        if profile["origin_ssh_alias"] != data["shared"]["aliyun_ssh_alias"]:
            fail(f"{context}.origin_ssh_alias conflicts with shared.aliyun_ssh_alias")
        hostname = profile["production_hostname"]
        public = https_url(profile["production_public_url"], f"{context}.production_public_url", True)
        playground = https_url(profile["playground_allowed_origin"], f"{context}.playground_allowed_origin", False)
        if public.hostname != hostname or playground.hostname != hostname:
            fail(f"{context} URL hostname does not match production_hostname")
        health = urllib.parse.urlsplit(profile["localhost_health_url"])
        if (health.scheme, health.hostname, health.port, health.path) != (
            "http", "127.0.0.1", profile["loopback_port"], "/"
        ) or health.query or health.fragment:
            fail(f"{context}.localhost_health_url does not match loopback_port")
        root = pathlib.PurePosixPath(profile["data_root"])
        if pathlib.PurePosixPath(profile["releases_root"]) != root / "releases" \
                or pathlib.PurePosixPath(profile["current"]) != root / "current" \
                or pathlib.PurePosixPath(profile["deployment_lock"]) != root / ".deploy.lock":
            fail(f"{context} data-root boundaries are inconsistent")
        expected_cache = "CF-Cache-Status" if profile["cdn"] == "cloudflare" else "EO-Cache-Status"
        if profile["cache_header"] != expected_cache:
            fail(f"{context}.cache_header does not match CDN")
    for field in UNIQUE_FIELDS:
        seen = {}
        for index, profile in enumerate(data["locales"]):
            value = profile[field]
            if value in seen:
                fail(f"duplicate/conflicting {field}: locales[{seen[value]}] and locales[{index}]")
            seen[value] = index
    return data


def emit(values):
    for value in values:
        print(str(value))


def main():
    repository = pathlib.Path(__file__).resolve().parent.parent
    parser = argparse.ArgumentParser(description="读取并严格校验 production identity")
    parser.add_argument("--identity", type=pathlib.Path, default=repository / "production" / "identity.json")
    subparsers = parser.add_subparsers(dest="command", required=True)
    subparsers.add_parser("validate", help="校验完整 identity")
    list_parser = subparsers.add_parser("list", help="按 identity 顺序列出 production profile")
    list_parser.add_argument("--state", choices=("first-production", "live"), help="仅列出指定 lifecycle state")
    locale_parser = subparsers.add_parser("locale", help="输出 locale 的有序字段")
    locale_parser.add_argument("locale")
    subparsers.add_parser("shared", help="输出共享 production identity 的有序字段")
    args = parser.parse_args()
    try:
        identity = load_identity(args.identity)
        if args.command == "validate":
            print(f"production identity: PASS ({len(identity['locales'])} locales)")
        elif args.command == "shared":
            emit(identity["shared"][field] for field in SHARED_FIELDS)
        elif args.command == "list":
            for profile in identity["locales"]:
                if args.state is None or profile["production_state"] == args.state:
                    print("\t".join(str(profile[field]) for field in (
                        "locale", "production_state", "production_public_url", "cdn", "systemd_service"
                    )))
        else:
            matches = [profile for profile in identity["locales"] if profile["locale"] == args.locale]
            if len(matches) != 1:
                fail(f"unknown production locale: {args.locale}")
            emit(matches[0][field] for field in LOCALE_FIELDS)
    except IdentityError as exc:
        print(f"production identity error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
