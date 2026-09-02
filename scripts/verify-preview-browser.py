#!/usr/bin/env python3
"""Fail-closed rendered-surface acceptance for a complete-locale preview."""

import csv
import importlib.util
import pathlib
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
import xml.etree.ElementTree as ET
from html.parser import HTMLParser

ROOT = pathlib.Path(__file__).resolve().parent.parent

def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module); return module

CORE = load("browser_acceptance_preview", ROOT / "scripts" / "browser_acceptance.py")
IDENTITY = load("production_identity_preview", ROOT / "scripts" / "production-identity.py")

def fail(stage, check, expected, actual):
    raise CORE.BrowserFailure(f"stage={stage} check={check} expected={expected} actual={actual}")

def preview_url(value):
    parsed = urllib.parse.urlsplit(value)
    if parsed.scheme != "http" or parsed.hostname not in ("127.0.0.1", "::1", "localhost") or parsed.port is None:
        fail("preview identity", "preview URL", "loopback HTTP origin with explicit port", value)
    if parsed.path != "/" or parsed.query or parsed.fragment or parsed.username or parsed.password:
        fail("preview identity", "preview URL", "origin ending in / without credentials/query/fragment", value)
    return parsed

def profile_for(identity, locale):
    profiles = [profile for profile in identity["locales"] if profile["locale"] == locale]
    if len(profiles) != 1: fail("preview identity", "locale", "one formal production identity", locale)
    return profiles[0]

def request(base, route, headers=None):
    req = urllib.request.Request(urllib.parse.urljoin(base, route.lstrip('/')), headers=headers or {})
    try:
        with urllib.request.urlopen(req, timeout=20) as response: return response.status, response.read()
    except urllib.error.HTTPError as exc: return exc.code, exc.read()
    except OSError as exc: fail("HTTP/routes", route, "reachable response", repr(exc))

class CanonicalParser(HTMLParser):
    def __init__(self):
        super().__init__(); self.canonicals = []; self.html_lang = ""; self.title_depth = 0; self.title = []
    def handle_starttag(self, tag, attrs):
        values = dict(attrs)
        if tag.lower() == "html": self.html_lang = values.get("lang", "")
        if tag.lower() == "title": self.title_depth += 1
        if tag.lower() == "link" and values.get("rel", "").lower() == "canonical":
            self.canonicals.append(values.get("href", ""))
    def handle_endtag(self, tag):
        if tag.lower() == "title" and self.title_depth: self.title_depth -= 1
    def handle_data(self, data):
        if self.title_depth: self.title.append(data)

def validate_raw_shell(body, route, canonical_path, locale, production, require_tour_shell=False):
    parser = CanonicalParser()
    try: text = body.decode("utf-8"); parser.feed(text)
    except (UnicodeError, ValueError) as exc: fail("SEO/routes", f"raw {route} shell", "valid UTF-8 HTML", repr(exc))
    expected = production + canonical_path
    if parser.canonicals != [expected]: fail("SEO/routes", f"raw {route} canonical", expected, parser.canonicals)
    if parser.html_lang != locale: fail("SEO/routes", f"raw {route} html lang", locale, parser.html_lang)
    if not ''.join(parser.title).strip(): fail("SEO/routes", f"raw {route} title", "non-empty", parser.title)
    if "localhost" in ''.join(parser.canonicals) or "127.0.0.1" in ''.join(parser.canonicals):
        fail("SEO/routes", f"raw {route} canonical", "no localhost identity", parser.canonicals)
    if require_tour_shell and not ('ng-app="tour"' in text and '<div ng-view' in text and 'class="bar top-bar"' in text):
        fail("SEO/routes", f"raw {route} shell", "valid Tour Angular shell", "required shell markers missing")

def catalog_routes():
    with (ROOT / "data" / "tour-pages.tsv").open(encoding="utf-8", newline="") as source:
        rows = list(csv.DictReader(source, delimiter="\t"))
    if len(rows) != 103: fail("SEO/routes", "catalog", "103 pages", len(rows))
    return ["/tour" + row["route"] for row in rows]

def registry():
    source = (ROOT / "internal" / "tour" / "languages.go").read_text(encoding="utf-8")
    block = re.search(r"var languageRegistry = \[\]LanguageLink\{(.*?)\n\}", source, re.S)
    if not block: fail("language selector", "registry source", "languageRegistry", "missing")
    pattern = re.compile(r'\{Locale: "([^"]+)", EnglishName: "([^"]+)", Autonym: "([^"]+)", URL: "([^"]+)"(?:, Official: true)?\}')
    result = [{"locale": m.group(1), "english_name": m.group(2), "autonym": m.group(3), "url": m.group(4)} for m in pattern.finditer(block.group(1))]
    if not result or len(result) != len({entry["locale"] for entry in result}): fail("language selector", "registry", "unique entries", result)
    return result

def formal_descriptions(locale):
    import json
    metadata = json.loads((ROOT / "locales" / locale / "course-metadata.json").read_text(encoding="utf-8"))
    if metadata.get("locale") != locale: fail("SEO/routes", "course metadata locale", locale, metadata.get("locale"))
    return {page["route"]: page["description"] for page in metadata["pages"]}

def validate_sitemap(sitemap, production, formal_course_routes):
    try: document = ET.fromstring(sitemap)
    except ET.ParseError as exc: fail("SEO/routes", "sitemap XML", "valid XML", repr(exc))
    ns = "{http://www.sitemaps.org/schemas/sitemap/0.9}"
    urls = [(node.text or '').strip() for node in document.findall(f"{ns}url/{ns}loc")]
    if len(urls) != 105: fail("SEO/routes", "sitemap URL count", 105, len(urls))
    if len(urls) != len(set(urls)): fail("SEO/routes", "sitemap duplicates", 0, len(urls)-len(set(urls)))
    expected = {production + "/", production + "/tour/list", *(production + route for route in formal_course_routes)}
    actual = set(urls)
    if actual != expected:
        fail("SEO/routes", "sitemap URL set", f"exact formal 105-URL set; missing={sorted(expected-actual)}",
             f"extra={sorted(actual-expected)}")
    for url in urls:
        parsed = urllib.parse.urlsplit(url)
        if parsed.scheme + "://" + parsed.netloc != production or parsed.query or parsed.fragment:
            fail("SEO/routes", "sitemap production origin", production, url)
    return urls

def machine_acceptance(base, profile):
    production = profile["production_public_url"].rstrip('/')
    formal_course_routes = catalog_routes()
    raw_routes = [("/", "/", False), ("/tour/", "/tour/", True), ("/tour/list", "/tour/list", True)]
    raw_routes.extend((route, "/tour/", True) for route in formal_course_routes)
    for route, canonical_path, require_tour_shell in raw_routes:
        status, body = request(base, route)
        if status != 200: fail("HTTP/routes", route, "HTTP 200", status)
        validate_raw_shell(body, route, canonical_path, profile["locale"], production, require_tour_shell)
    for route in ["/robots.txt", "/sitemap.xml"]:
        status, _ = request(base, route)
        if status != 200: fail("HTTP/routes", route, "HTTP 200", status)
    for headers, name in (({}, "GET /socket"), ({"Connection":"Upgrade","Upgrade":"websocket"}, "Upgrade /socket")):
        status, _ = request(base, "/socket", headers)
        if status != 404: fail("socket boundary", name, "HTTP 404", status)
    _, robots = request(base, "/robots.txt")
    expected_sitemap = production + "/sitemap.xml"
    if expected_sitemap not in robots.decode('utf-8'): fail("SEO/routes", "robots sitemap", expected_sitemap, robots.decode('utf-8','replace'))
    _, sitemap = request(base, "/sitemap.xml")
    validate_sitemap(sitemap, production, formal_course_routes)

def main():
    if len(sys.argv) != 3:
        print(f"usage: {pathlib.Path(sys.argv[0]).name} <preview-url> <locale>", file=sys.stderr); return 2
    base, locale = sys.argv[1:]
    try:
        preview_url(base)
        identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        profile = profile_for(identity, locale)
        machine_acceptance(base, profile)
        CORE.preview_acceptance(base, locale, profile, identity["shared"], registry(), formal_descriptions(locale))
    except (CORE.BrowserFailure, IDENTITY.IdentityError, OSError, KeyError, TypeError) as exc:
        detail = str(exc)
        if not detail.startswith("stage="):
            detail = f"stage=browser check=rendered/interaction expected=PASS actual={detail}"
        print(f"[preview-browser] FAILED: {detail}", file=sys.stderr); return 1
    for stage in ("preview identity", "SEO/routes", "desktop rendered surface", "editor Run / Format / Reset", "SPA", "mobile /tour/moretypes/1"):
        print(f"[preview-browser] {stage}: PASS")
    print("PREVIEW SURFACE ACCEPTANCE: PASS")
    return 0

if __name__ == "__main__": raise SystemExit(main())
