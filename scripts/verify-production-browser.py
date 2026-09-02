#!/usr/bin/env python3
"""Automated Chrome acceptance for a formal production locale."""

import importlib.util
import pathlib
import sys

ROOT = pathlib.Path(__file__).resolve().parent.parent

def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module); return module

CORE = load("browser_acceptance_production", ROOT / "scripts" / "browser_acceptance.py")
IDENTITY = load("production_identity_browser", ROOT / "scripts" / "production-identity.py")
BrowserFailure = CORE.BrowserFailure
browser_ad_gate = CORE.browser_ad_gate

def acceptance(base, locale, profile, shared):
    return CORE.acceptance(base, locale, profile, shared)

def main():
    if len(sys.argv) != 3:
        print(f"usage: {pathlib.Path(sys.argv[0]).name} <production-public-url> <locale>", file=sys.stderr); return 2
    base, locale = sys.argv[1:]
    parsed = CORE.urllib.parse.urlsplit(base)
    if parsed.scheme != "https" or parsed.path != "/" or parsed.query or parsed.fragment:
        print("[production-browser] ERROR: public URL must be an HTTPS origin ending in /", file=sys.stderr); return 1
    try:
        identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        profiles = [profile for profile in identity["locales"] if profile["locale"] == locale]
        if len(profiles) != 1: raise BrowserFailure(f"unknown formal production locale: {locale}")
        profile = profiles[0]
        if base != profile["production_public_url"]: raise BrowserFailure("public URL does not match formal production identity")
        acceptance(base, locale, profile, identity["shared"])
    except (CORE.BrowserFailure, IDENTITY.IdentityError, OSError, KeyError, TypeError) as exc:
        print(f"[production-browser] FAILED: {exc}", file=sys.stderr); return 1
    print("[production-browser] desktop routes: PASS")
    print("[production-browser] mobile /tour/moretypes/1: PASS")
    print("[production-browser] Run / Format / Reset / SPA / ads: PASS")
    print("PRODUCTION BROWSER ACCEPTANCE: PASS")
    return 0

if __name__ == "__main__": raise SystemExit(main())
