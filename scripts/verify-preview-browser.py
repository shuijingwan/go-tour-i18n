#!/usr/bin/env python3
"""Fail-closed rendered-surface acceptance for a complete-locale preview."""

import csv
import importlib.util
import json
import pathlib
import re
import sys
import subprocess
import time
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

def browser_plan(locale):
    """Use Go's current adoption, renderer and sparse route authority."""
    result = subprocess.run(["go", "run", "-mod=readonly", "./cmd/tour-i18n", "preview-browser-plan", "--locale", locale],
                            cwd=ROOT, capture_output=True, text=True, timeout=120, check=False)
    if result.returncode:
        fail("Site v2 plan", "current authority", "read-only plan", result.stderr.strip())
    plan = json.loads(result.stdout)
    if plan.get("schema") != "go-learning/preview-browser-plan/v1" or plan.get("locale") != locale:
        fail("Site v2 plan", "identity", locale, plan.get("schema"))
    routes = plan["canonical_routes"]
    pages = [p["route"] for p in plan["pages"]]
    if (len(routes) != len(set(routes)) or len(pages) != len(set(pages)) or
            set(pages) != {r for r in routes if not r.startswith("/tour/")}):
        fail("Site v2 plan", "page exact-set", "all activated non-Tour routes", pages)
    return plan

class PageStructure(HTMLParser):
    """Structural contract from actual Go-rendered source and response HTML."""
    def __init__(self):
        super().__init__()
        self.main_depth = 0; self.mains = 0; self.main_text = []
        self.classes = set(); self.main_classes = set(); self.class_counts = {}; self.links = set(); self.assets = set(); self.pre = 0
        self.pre_text = None; self.pre_nonempty = 0
        self.ids = set(); self.main_links = []
        self.textareas = 0; self.editor = 0; self.run = 0; self.tour = False
    def handle_starttag(self, tag, attrs):
        a = dict(attrs); classes = set(a.get("class", "").split()); self.classes.update(classes)
        if a.get("id"): self.ids.add(a["id"])
        for value in classes: self.class_counts[value] = self.class_counts.get(value, 0) + 1
        self.tour |= a.get("ng-app") == "tour" or "ng-view" in a
        if tag == "main": self.mains += 1; self.main_depth += 1
        if self.main_depth: self.main_classes.update(classes)
        if tag == "a" and a.get("href"): self.links.add(a["href"])
        if tag == "a" and a.get("href") and self.main_depth: self.main_links.append(a["href"])
        if tag in ("script", "img") and a.get("src"): self.assets.add(a["src"])
        if tag == "link" and a.get("rel") == "stylesheet": self.assets.add(a["href"])
        if tag == "pre" and self.main_depth: self.pre += 1; self.pre_text = []
        if tag == "textarea":
            self.textareas += 1
            self.editor += int("js-playgroundCodeEl" in classes)
        if tag == "button" and "js-playgroundRunEl" in classes: self.run += 1
    def handle_endtag(self, tag):
        if tag == "pre" and self.pre_text is not None:
            self.pre_nonempty += int(bool(''.join(self.pre_text).strip())); self.pre_text = None
        if tag == "main" and self.main_depth: self.main_depth -= 1
    def handle_data(self, data):
        if self.main_depth: self.main_text.append(data)
        if self.pre_text is not None: self.pre_text.append(data)

def structure(body):
    p = PageStructure(); p.feed(body); return p

def page_contract(page):
    p = structure(page["html"])
    links = page.get("required_links")
    if not isinstance(links, list) or any(not isinstance(link, str) or not link for link in links):
        fail("Site v2 plan", page["route"], "explicit source-bound required_links", links)
    # Rendered English anchors may include Linkify artifacts from ordinary URL
    # prose. Only the Go parser/template contract supplies required link targets.
    p.links = set(links)
    if (p.textareas != p.editor or p.editor != p.run or p.editor not in (0, 1) or
            "playground" in p.classes or "CodeMirror" in p.classes):
        fail("Site v2 plan", page["route"], "supported executable widget or static page", "unknown controls")
    return p

def browser_samples(plan):
    """One route per structure bucket, plus every widget and declared breadcrumb."""
    samples = {}; executable = {}
    for page in sorted(plan["pages"], key=lambda p: p["route"]):
        p = page_contract(page)
        key = (page["family"], page["format"], p.pre > 0, p.editor > 0)
        samples.setdefault(key, page)
        if p.editor: executable[page["route"]] = page
        if "SiteBreadcrumb" in p.classes: executable[page["route"]] = page
    selected = {p["route"]: p for p in samples.values()}; selected.update(executable)
    if {p["family"] for p in selected.values()} != {p["family"] for p in plan["pages"]}:
        fail("Site v2 plan", "sampling", "every activated Site family", selected.keys())
    return [selected[r] for r in sorted(selected)]

def normalized_links(links, base, route):
    return {urllib.parse.urljoin(urllib.parse.urljoin(base, route), link) for link in links}

def same_document_fragment(href, base, route):
    """Match the browser's resolved origin/path and decoded fragment contract."""
    current = urllib.parse.urlsplit(urllib.parse.urljoin(base, route))
    target = urllib.parse.urlsplit(urllib.parse.urljoin(current.geturl(), href))
    def origin(url):
        return (url.scheme.lower(), url.hostname, url.port or {"http": 80, "https": 443}.get(url.scheme))
    if origin(current) != origin(target) or current.path != target.path or not target.fragment:
        return None
    if re.search(r"%(?![0-9a-fA-F]{2})", target.fragment):
        fail("Site fragment integrity", route, "valid percent-encoded fragment", href)
    try:
        return urllib.parse.unquote(target.fragment, errors="strict")
    except UnicodeDecodeError:
        fail("Site fragment integrity", route, "valid UTF-8 fragment", href)

def validate_fragment_targets(actual, page, base):
    # Required source-bound links and every rendered main link are checked on
    # every activated Site Page, before the sampled real-browser checks.
    for href in page["required_links"] + actual.main_links:
        fragment = same_document_fragment(href, base, page["route"])
        # Native top-of-document navigation needs no element with id="top".
        # Match ASCII case only; ordinary fragments still require exact IDs.
        if fragment is not None and fragment not in actual.ids and re.fullmatch(r"[tT][oO][pP]", fragment) is None:
            fail("Site fragment integrity", page["route"], "DOM target for " + href, "missing id=" + fragment)
    return sum(same_document_fragment(href, base, page["route"]) is not None for href in actual.main_links)

class BreadcrumbStructure(HTMLParser):
    def __init__(self):
        super().__init__(); self.containers = []; self.current = self.item = self.anchor = None
    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if "SiteBreadcrumb" in attrs.get("class", "").split():
            self.current = []; self.containers.append(self.current)
        if self.current is not None and tag == "li":
            self.item = {"text": "", "anchors": []}; self.current.append(self.item)
        if self.item is not None and tag == "a":
            self.anchor = {"text": "", "href": attrs.get("href", "")}; self.item["anchors"].append(self.anchor)
    def handle_data(self, text):
        if self.item is not None: self.item["text"] += text
        if self.anchor is not None: self.anchor["text"] += text
    def handle_endtag(self, tag):
        if tag == "a": self.anchor = None
        if tag == "li": self.item = None
        if tag == "ol": self.current = self.item = self.anchor = None


def validate_breadcrumbs(body, page, base):
    expected = BreadcrumbStructure(); expected.feed(page["html"])
    actual = BreadcrumbStructure(); actual.feed(body)
    if len(actual.containers) != len(expected.containers):
        fail("Site breadcrumbs", page["route"], "declared breadcrumb containers", len(actual.containers))
    for want, got in zip(expected.containers, actual.containers):
        if not got or any(not " ".join(item["text"].split()) or not item["anchors"] for item in got):
            fail("Site breadcrumbs", page["route"], "non-empty breadcrumb items", got)
        anchors = [a for item in got for a in item["anchors"]]
        if any(not " ".join(a["text"].split()) or not a["href"].strip() for a in anchors):
            fail("Site breadcrumbs", page["route"], "non-empty breadcrumb labels/hrefs", anchors)
        # The Go plan uses the same Resolver and current sparse coverage. Compare
        # ordered targets; a go.dev fallback cannot masquerade as a local route.
        resolve = lambda href: urllib.parse.urljoin(urllib.parse.urljoin(base, page["route"]), href)
        wanted = [resolve(a["href"]) for item in want for a in item["anchors"]]
        if [resolve(a["href"]) for a in anchors] != wanted:
            fail("Site breadcrumbs", page["route"], wanted, [a["href"] for a in anchors])


def check_visible_breadcrumbs(chrome):
    CORE.assert_true(chrome.evaluate("""(() => {
      const visible=n=>{const r=n.getBoundingClientRect(),s=getComputedStyle(n);return r.width>0&&r.height>0&&s.visibility==='visible'&&s.display!=='none'&&Number(s.opacity)>0};
      return [...document.querySelectorAll('.SiteBreadcrumb')].every(nav=>{
        const items=[...nav.querySelectorAll('li')];
        return visible(nav)&&items.length>0&&items.every(li=>visible(li)&&li.innerText.trim()&&
          li.querySelectorAll('a').length>0&&[...li.querySelectorAll('a')].every(a=>visible(a)&&a.innerText.trim()&&a.getAttribute('href')?.trim()));
      });
    })()""", check="visible breadcrumb labels/hrefs"), "empty or invisible breadcrumb item/anchor")


def validate_site_page(body, page, base, profile):
    route = page["route"]
    validate_breadcrumbs(body.decode("utf-8"), page, base)
    validate_raw_shell(body, route, route, profile["locale"], profile["production_public_url"].rstrip("/"))
    actual = structure(body.decode("utf-8")); expected = page_contract(page)
    if (actual.tour or "site-home" in actual.classes or actual.mains != 1 or
            len(''.join(actual.main_text).strip()) < 20):
        fail("Site v2 renderer", route, "one substantive Site main", "missing content or wrong renderer")
    chrome = {"Site-header", "Site-footer"} if "Site-header" in expected.classes else {"top-bar", "site-footer"}
    if not chrome <= actual.classes or any(actual.class_counts[name] != 1 for name in chrome):
        fail("Site v2 renderer", route, sorted(chrome), sorted(actual.classes))
    if not expected.main_classes <= actual.main_classes:
        fail("Site v2 renderer", route, sorted(expected.main_classes-actual.main_classes), "missing main structure")
    for name in ("pre", "textareas", "editor", "run"):
        if getattr(actual, name) != getattr(expected, name):
            fail("Site v2 renderer", f"{route} {name}", getattr(expected, name), getattr(actual, name))
    if not expected.editor and actual.pre_nonempty != expected.pre_nonempty:
        fail("Site v2 renderer", f"{route} non-empty code blocks", expected.pre_nonempty, actual.pre_nonempty)
    metadata = CanonicalParser(); metadata.feed(page["html"])
    if metadata.descriptions:
        rendered = CanonicalParser(); rendered.feed(body.decode("utf-8"))
        if len(rendered.descriptions) != 1 or not rendered.descriptions[0].strip():
            fail("Site v2 renderer", f"{route} description", "one non-empty description", rendered.descriptions)
    for name in ("links", "assets"):
        want = normalized_links(getattr(expected, name), base, route)
        got = normalized_links(getattr(actual, name), base, route)
        if not want <= got:
            fail("Site v2 navigation/assets", route, sorted(want-got), "missing source-bound " + name)
    validate_fragment_targets(actual, page, base)
    return actual

def site_machine_acceptance(base, profile, plan):
    markdown_pages = fragment_links = 0
    for page in plan["pages"]:
        status, body = request(base, page["route"], follow_redirects=False)
        if status != 200: fail("Site v2 routes", page["route"], "HTTP 200 without redirect", status)
        actual = validate_site_page(body, page, base, profile)
        if page["format"] == "md":
            markdown_pages += 1
            fragment_links += sum(same_document_fragment(href, base, page["route"]) is not None for href in actual.main_links)
    # One non-covered path per family, including Learn/Docs: no Tour fallback.
    for family in plan["families"]:
        prefix = family.removesuffix("**")
        route = prefix.rstrip("/") + "/__preview_missing_site_route__"
        status, _ = request(base, route, follow_redirects=False)
        if status != 404: fail("Site v2 routes", route, "HTTP 404 without redirect", status)
    print("[preview-browser] Markdown fragment integrity: MARKDOWN_PAGES_CHECKED=%d SAME_DOCUMENT_FRAGMENT_LINKS=%d BROKEN_FRAGMENT_TARGETS=0" %
          (markdown_pages, fragment_links), flush=True)

def assert_browser_events(chrome, base):
    """No console/runtime/dialog/transport/HTTP failures, including external assets."""
    for event in chrome.events:
        method = event.get("method"); p = event.get("params", {})
        if (method in ("Runtime.exceptionThrown", "Page.javascriptDialogOpening", "Network.loadingFailed", "Network.webSocketFrameError") or
                method == "Network.webSocketCreated" and urllib.parse.urlsplit(p.get("url", "")).path.startswith("/socket") or
                method == "Runtime.consoleAPICalled" and p.get("type") in ("error", "assert") or
                method == "Log.entryAdded" and p.get("entry", {}).get("level") == "error"):
            # The legacy Tour explicitly probes this Fetch after asserting no
            # editor request used /socket. Only its expected HTTP 404 is allowed.
            entry = p.get("entry", {})
            if (method == "Log.entryAdded" and entry.get("source") == "network" and
                    entry.get("url") == base.rstrip("/") + "/socket" and "404" in entry.get("text", "")):
                continue
            fail("browser errors", chrome.current_route, "no browser/runtime/network error", (method, p))
        if method == "Network.responseReceived":
            r = p.get("response", {}); status = r.get("status", 0)
            intentional_socket = p.get("type") == "Fetch" and r.get("url") == base.rstrip("/") + "/socket" and status == 404
            if status >= 400 and not intentional_socket:
                fail("browser network", chrome.current_route, "HTTP <400", (r.get("url"), status))
        if method == "Network.webSocketHandshakeResponseReceived" and p.get("response", {}).get("status", 0) >= 400:
            fail("browser network", chrome.current_route, "successful WebSocket handshake", p.get("response", {}).get("status"))

def settle_site_page(chrome, base, timeout=25):
    deadline = time.monotonic() + timeout
    last_count = -1; quiet_since = time.monotonic()
    while True:
        ready = chrome.evaluate("document.readyState === 'complete'", check="Site assets loaded")
        assert_browser_events(chrome, base)
        started = {e["params"]["requestId"] for e in chrome.events if e.get("method") == "Network.requestWillBeSent"}
        finished = {e["params"]["requestId"] for e in chrome.events if e.get("method") in ("Network.loadingFinished", "Network.loadingFailed")}
        if len(chrome.events) != last_count: last_count = len(chrome.events); quiet_since = time.monotonic()
        if ready and not started-finished and time.monotonic()-quiet_since >= .5: return
        if time.monotonic() >= deadline:
            fail("browser network", chrome.current_route, "complete document and settled requests", sorted(started-finished))
        time.sleep(.1)

def settle_site_images(chrome, base, timeout=25):
    """Observe unloaded images before requiring decoded pixels, including lazy ones."""
    position = chrome.evaluate("[scrollX, scrollY]", check="Site image scroll position")
    snapshot = """[...document.querySelectorAll('main img')].map((i,index)=>{
      const r=i.getBoundingClientRect();return {index,src:i.src,currentSrc:i.currentSrc,
        loading:i.loading,complete:i.complete,naturalWidth:i.naturalWidth,naturalHeight:i.naturalHeight,
        rect:{top:r.top,bottom:r.bottom,width:r.width,height:r.height}}})"""
    try:
        images = chrome.evaluate(snapshot, check="Site image inventory")
        for image in images:
            if not image["complete"]:
                index = image["index"]
                chrome.evaluate(f"document.querySelectorAll('main img')[{index}].scrollIntoView({{block:'center',behavior:'instant'}});true",
                                check="Site image observation")
                deadline = time.monotonic() + timeout
                while True:
                    current = chrome.evaluate(snapshot, check="Site image settlement")
                    assert_browser_events(chrome, base)
                    if len(current) != len(images):
                        fail("browser images", chrome.current_route, "stable main image inventory", current)
                    image = current[index]
                    if image["complete"]: break
                    if time.monotonic() >= deadline:
                        fail("browser images", chrome.current_route, "observed image loads within timeout", image)
                    time.sleep(.1)
            if not image["currentSrc"] or image["naturalWidth"] <= 0 or image["naturalHeight"] <= 0:
                fail("browser images", chrome.current_route, "loaded image with nonzero natural size", image)
    finally:
        chrome.evaluate(f"window.scrollTo({{left:{json.dumps(position[0])},top:{json.dumps(position[1])},behavior:'instant'}});true",
                        check="Site image scroll restoration")
    # Drain resource/network events triggered by observation and restoration.
    settle_site_page(chrome, base)

def run_site_example(chrome, base):
    # Run through the real bound UI handler. No injected transport or mocked API.
    token = "SITE_V2_BROWSER_ACCEPTANCE_OK"
    CORE.assert_true(chrome.evaluate("""(() => {const run=document.querySelector('.js-playgroundRunEl');
      run.scrollIntoView({block:'center'});const r=run.getBoundingClientRect();
      return r.width>0&&r.height>0&&!run.disabled})()""", check="Site Playground visible Run control"),
      "Site Playground Run is hidden or disabled")
    request_start = len(chrome.network_requests())
    chrome.evaluate("""(() => {const code=document.querySelector('.js-playgroundCodeEl');
      code.value='package main\\nimport "fmt"\\nfunc main(){fmt.Println("SITE_V2_BROWSER_ACCEPTANCE_OK")}\\n';
      code.dispatchEvent(new Event('input',{bubbles:true}));
      document.querySelector('.js-playgroundOutputEl pre').replaceChildren();
      document.querySelector('.js-playgroundRunEl').click();return true})()""", check="Site Playground Run")
    def completed():
        assert_browser_events(chrome, base)
        value = chrome.evaluate("""(() => {const out=document.querySelector('.js-playgroundOutputEl');return {
          text:out.innerText,stdout:[...out.querySelectorAll('.stdout')].map(n=>n.innerText).join(''),
          stderr:out.querySelectorAll('.stderr,.error').length,system:out.querySelectorAll('.system').length}})()""",
          check="Site Playground deterministic completion")
        CORE.assert_true(value["stdout"].strip() == token and value["system"] > 0 and value["stderr"] == 0,
                         f"Site Playground missing successful output/end: {value}")
        return value
    CORE.wait_for_condition(completed, "Site Playground output and completion")
    settle_site_page(chrome, base)
    requests = chrome.network_requests()[request_start:]
    for operation in ("compile", "fmt"):
        matches = [r for r in requests if urllib.parse.urlsplit(r.get("url", "")).path == "/_/" + operation]
        CORE.assert_true(len(matches) == 1 and matches[0]["method"] == "POST", f"Site {operation} request mismatch: {matches}")
        url = urllib.parse.urlsplit(matches[0]["url"])
        CORE.assert_true(url.scheme + "://" + url.netloc == base.rstrip("/") and url.query == "backend=",
                         f"Site {operation} runtime endpoint mismatch: {matches}")
    responses = [e["params"] for e in chrome.events if e.get("method") == "Network.responseReceived" and
                 urllib.parse.urlsplit(e["params"]["response"]["url"]).path == "/_/compile"]
    CORE.assert_true(bool(responses), "Site compile response missing")
    raw = chrome.call("Network.getResponseBody", {"requestId": responses[-1]["requestId"]})
    CORE.assert_true(not raw.get("base64Encoded"), "Site compile response unexpectedly binary")
    result = json.loads(raw["body"])
    CORE.assert_true(not result.get("Errors") and not result.get("VetErrors") and result.get("Status", 0) == 0 and
                     ''.join(e["Message"] for e in result.get("Events", []) if e["Kind"] == "stdout").strip() == token,
                     f"Site compile success response mismatch: {result}")
    # A syntax error is an application failure state with a successful HTTP
    # response, not permission to ignore console/transport errors.
    chrome.evaluate("""(() => {document.querySelector('.js-playgroundCodeEl').value='package main\\nfunc main( {';
      document.querySelector('.js-playgroundOutputEl pre').replaceChildren();
      document.querySelector('.js-playgroundRunEl').click();return true})()""", check="Site Playground syntax failure")
    def rejected():
        assert_browser_events(chrome, base)
        value = chrome.evaluate("""(() => {const out=document.querySelector('.js-playgroundOutputEl');return {
          error:!!out.querySelector('.error,.stderr'), text:out.innerText}})()""", check="Site Playground failure state")
        CORE.assert_true(value["error"] and value["text"].strip() and token not in value["text"],
                         f"Site Playground missing syntax failure state: {value}")
    CORE.wait_for_condition(rejected, "Site Playground syntax error")
    settle_site_page(chrome, base)

def check_learn_navigation(chrome):
    CORE.assert_true(chrome.evaluate("""(() => {
      let selected;
      for (const a of document.querySelectorAll('.LearnNav a[href]')) {
        let url, id;
        try {
          url=new URL(a.href,location.href);
          id=decodeURIComponent(url.hash.slice(1));
        } catch { continue; }
        if (url.origin===location.origin && url.pathname===location.pathname && id.trim()) {
          selected={a,url,id};break;
        }
      }
      if (!selected) return false;
      const {a,url,id}=selected, previousHash=location.hash;
      a.click();
      const target=document.getElementById(id);
      if (location.origin!==url.origin || location.pathname!==url.pathname ||
          location.hash!==url.hash || location.hash===previousHash || !target) return false;
      const r=target.getBoundingClientRect(),style=getComputedStyle(target);
      return r.width>0 && r.height>0 && target.innerText.trim().length>0 &&
        style.display!=='none' && style.visibility==='visible' && Number(style.opacity)>0;
    })()""", check="Learn navigation anchor click"), "Learn navigation did not reach a real section")

def site_browser_acceptance(chrome, base, profile, plan, shared_chrome_check=None):
    for page in browser_samples(plan):
        contract = page_contract(page)
        for width, height in ((1280, 800), (375, 812)):
            chrome.navigate(urllib.parse.urljoin(base, page["route"]), width, height)
            settle_site_page(chrome, base)
            body = chrome.evaluate("document.documentElement.outerHTML", check="Site rendered DOM")
            validate_site_page(body.encode(), page, base, profile)
            check_visible_breadcrumbs(chrome)
            settle_site_images(chrome, base)
            state = chrome.evaluate("""(() => {const m=document.querySelector('main'),r=m.getBoundingClientRect();return {
              path:location.pathname,origin:location.origin,text:m.innerText.trim(),width:r.width,
              overflow:document.documentElement.scrollWidth-document.documentElement.clientWidth,
              anchors:[...m.querySelectorAll('a[href]')].every(a=>{const u=new URL(a.href);return (
                u.origin!==location.origin||u.pathname!==location.pathname||u.hash.length<2||
                (()=>{const id=decodeURIComponent(u.hash.slice(1));return (
                  id.length===3&&/^[tT][oO][pP]$/.test(id))||!!document.getElementById(id)})())}),
              images:[...document.querySelectorAll('main img')].every(i=>i.complete&&i.naturalWidth>0&&i.naturalHeight>0)}})()""", check="Site visible content/assets/overflow")
            CORE.assert_true(state["path"] == page["route"] and state["origin"] == base.rstrip("/"), f"Site route redirected: {state}")
            CORE.assert_true(len(state["text"]) > 20 and state["width"] > 0 and state["overflow"] <= 2 and state["images"] and state["anchors"],
                             f"Site content/assets/overflow: {page['route']} width={width} {state}")
            if shared_chrome_check is not None: shared_chrome_check(chrome)
            if "LearnNav" in contract.classes:
                check_learn_navigation(chrome)
            if width == 375 and "Site-header" in contract.classes:
                chrome.evaluate("document.querySelector('.js-headerMenuButton').click();true", check="Site mobile navigation open")
                CORE.assert_true(chrome.evaluate("document.querySelector('.js-header').classList.contains('is-active')"), "Site mobile navigation did not open")
                chrome.evaluate("document.querySelector('.js-headerMenuButton').click();true", check="Site mobile navigation close")
                CORE.assert_true(chrome.evaluate("!document.querySelector('.js-header').classList.contains('is-active')"), "Site mobile navigation did not close")
            if contract.editor and width == 1280: run_site_example(chrome, base)
            assert_browser_events(chrome, base)

def preview_chrome_factory(base):
    class CheckedChrome(CORE.Chrome):
        def __init__(self):
            super().__init__(); self.call("Log.enable")
            owner = self
            class CheckedEvents(list):
                def clear(self):
                    # Chrome's bounded readiness retries must not erase a
                    # failed resource, console error or dialog from an attempt.
                    assert_browser_events(owner, base)
                    super().clear()
            self.events = CheckedEvents(self.events)
        def navigate(self, url, width, height):
            self.evaluate("true", check="flush browser errors before navigation")
            assert_browser_events(self, base)
            return super().navigate(url, width, height)
        def close(self):
            try:
                if sys.exc_info()[0] is None:
                    self.evaluate("true", check="flush final browser errors")
                    assert_browser_events(self, base)
            finally: super().close()
    return CheckedChrome

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

class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None

def request(base, route, headers=None, follow_redirects=True):
    req = urllib.request.Request(urllib.parse.urljoin(base, route.lstrip('/')), headers=headers or {})
    open_request = urllib.request.urlopen if follow_redirects else urllib.request.build_opener(NoRedirect()).open
    try:
        with open_request(req, timeout=20) as response: return response.status, response.read()
    except urllib.error.HTTPError as exc: return exc.code, exc.read()
    except OSError as exc: fail("HTTP/routes", route, "reachable response", repr(exc))

class CanonicalParser(HTMLParser):
    def __init__(self):
        super().__init__(); self.canonicals = []; self.html_lang = ""; self.title_depth = 0; self.title = []; self.descriptions = []
    def handle_starttag(self, tag, attrs):
        values = dict(attrs)
        if tag.lower() == "html": self.html_lang = values.get("lang", "")
        if tag.lower() == "title": self.title_depth += 1
        if tag.lower() == "link" and values.get("rel", "").lower() == "canonical":
            self.canonicals.append(values.get("href", ""))
        if tag.lower() == "meta" and values.get("name", "").lower() == "description":
            self.descriptions.append(values.get("content", ""))
    def handle_endtag(self, tag):
        if tag.lower() == "title" and self.title_depth: self.title_depth -= 1
    def handle_data(self, data):
        if self.title_depth: self.title.append(data)

class SiteShellParser(HTMLParser):
    def __init__(self):
        super().__init__(); self.main_depth = 0; self.mains = 0; self.links = []; self.tour = False; self.legacy = False
    def handle_starttag(self, tag, attrs):
        values = dict(attrs)
        self.tour |= values.get("ng-app") == "tour" or "ng-view" in values
        self.legacy |= "site-home" in values.get("class", "").split()
        if tag == "main": self.main_depth += 1; self.mains += 1
        if tag == "a" and self.main_depth: self.links.append(values.get("href", ""))
    def handle_endtag(self, tag):
        if tag == "main" and self.main_depth: self.main_depth -= 1

def activated_content_scope(locale):
    """Read the formal sparse coverage; runtime still verifies completion evidence."""
    path = ROOT / "locales" / locale / "content-scope.json"
    if not path.exists(): return None
    scope = json.loads(path.read_text(encoding="utf-8"))
    if scope.get("schema") != "go-learning/locale-content-scope/v1" or scope.get("locale") != locale:
        fail("Site shell", "Content Scope identity", locale, scope)
    packages = scope["packages"]
    routes = []
    names = []
    for package in packages:
        names.append(package["package"])
        if package.get("state") != "complete":
            fail("SEO/routes", "Content Scope completion", "complete sparse records", package)
        routes.extend(package["local_canonical_routes"])
    if len(names) != len(set(names)) or len(routes) != len(set(routes)):
        fail("SEO/routes", "Content Scope duplicates", "unique packages and canonical routes", scope)
    return scope

def activated_canonical_routes(locale):
    scope = activated_content_scope(locale)
    if scope is None: return None
    return {route for package in scope["packages"] for route in package["local_canonical_routes"]}

def activated_shell_coverage(locale):
    """Select stricter checks; this never replaces the runtime's formal gates."""
    scope = activated_content_scope(locale)
    if scope is None: return None
    packages = scope["packages"]
    shell = [p for p in packages if p.get("package") == "site-v2-shell"]
    if not shell: return None
    if len(shell) != 1 or shell[0].get("state") != "complete" or set(shell[0].get("local_canonical_routes", [])) != {"/", "/translation/"}:
        fail("Site shell", "activation coverage", "one complete shell with both canonical routes", shell)
    return {route for package in packages for route in package["local_canonical_routes"]}

def validate_site_shell(body, source, base, route, locale, production, coverage):
    validate_raw_shell(body, route, route, locale, production)
    actual = SiteShellParser(); actual.feed(body.decode("utf-8"))
    frozen = SiteShellParser(); frozen.feed(source)
    if actual.tour or actual.legacy or actual.mains != 1:
        fail("Site shell", route, "one Site shell main without legacy homepage or Tour SPA", "wrong renderer")
    actual_links = {urllib.parse.urljoin(base, link) for link in actual.links}
    for link in frozen.links:
        parsed = urllib.parse.urlsplit(link)
        expected = link if parsed.scheme or parsed.netloc else urllib.parse.urljoin(base if parsed.path in coverage else "https://go.dev/", link)
        if expected not in actual_links:
            fail("Site shell", f"{route} navigation", expected, sorted(actual_links))

def activated_shell_acceptance(base, profile):
    coverage = activated_shell_coverage(profile["locale"])
    if coverage is None: return
    production = profile["production_public_url"].rstrip('/')
    for route, name in (("/", "home.html"), ("/translation/", "translation.html")):
        status, body = request(base, route, follow_redirects=False)
        if status != 200: fail("Site shell", route, "HTTP 200 without redirect", status)
        source = (ROOT / "data" / "site-shell" / name).read_text(encoding="utf-8")
        validate_site_shell(body, source, base, route, profile["locale"], production, coverage)
    for route in ("/__preview_missing_site_route__", "/translation/__preview_missing_site_route__", "/tour/__preview_missing_site_route__"):
        status, _ = request(base, route, follow_redirects=False)
        if status != 404: fail("Site shell", route, "HTTP 404 without redirect or Tour fallback", status)

def validate_raw_shell(body, route, canonical_path, locale, production, require_tour_shell=False, list_metadata=None):
    parser = CanonicalParser()
    try: text = body.decode("utf-8"); parser.feed(text)
    except (UnicodeError, ValueError) as exc: fail("SEO/routes", f"raw {route} shell", "valid UTF-8 HTML", repr(exc))
    expected = production + canonical_path
    if parser.canonicals != [expected]: fail("SEO/routes", f"raw {route} canonical", expected, parser.canonicals)
    if parser.html_lang != locale: fail("SEO/routes", f"raw {route} html lang", locale, parser.html_lang)
    if not ''.join(parser.title).strip(): fail("SEO/routes", f"raw {route} title", "non-empty", parser.title)
    if list_metadata is not None:
        if ''.join(parser.title).strip() != list_metadata["title"]:
            fail("SEO/routes", f"raw {route} title", list_metadata["title"], ''.join(parser.title).strip())
        if parser.descriptions != [list_metadata["description"]]:
            fail("SEO/routes", f"raw {route} description", list_metadata["description"], parser.descriptions)
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
    return CORE.shared_language_registry()

def formal_descriptions(locale):
    import json
    metadata = json.loads((ROOT / "locales" / locale / "course-metadata.json").read_text(encoding="utf-8"))
    if metadata.get("locale") != locale: fail("SEO/routes", "course metadata locale", locale, metadata.get("locale"))
    return {page["route"]: page["description"] for page in metadata["pages"]}

def formal_list_metadata(locale):
    import json
    catalog = json.loads((ROOT / "internal" / "tour" / "ui" / f"{locale}.json").read_text(encoding="utf-8"))
    messages = catalog.get("messages", {})
    values = {}
    for name, key in (("title", "tour.list_title"), ("description", "tour.list_description"), ("heading", "tour.list_heading")):
        entry = messages.get(key, {})
        if entry.get("kind") != "plain" or not entry.get("text"):
            fail("SEO/routes", f"list UI key {key}", "non-empty plain catalog message", entry)
        values[name] = entry["text"]
    return values

def validate_sitemap(sitemap, production, canonical_routes):
    try: document = ET.fromstring(sitemap)
    except ET.ParseError as exc: fail("SEO/routes", "sitemap XML", "valid XML", repr(exc))
    ns = "{http://www.sitemaps.org/schemas/sitemap/0.9}"
    urls = [(node.text or '').strip() for node in document.findall(f"{ns}url/{ns}loc")]
    if len(urls) != len(canonical_routes): fail("SEO/routes", "sitemap URL count", len(canonical_routes), len(urls))
    if len(urls) != len(set(urls)): fail("SEO/routes", "sitemap duplicates", 0, len(urls)-len(set(urls)))
    expected = {production + route for route in canonical_routes}
    actual = set(urls)
    if actual != expected:
        fail("SEO/routes", "sitemap URL set", f"exact activated coverage; missing={sorted(expected-actual)}",
             f"extra={sorted(actual-expected)}")
    for url in urls:
        parsed = urllib.parse.urlsplit(url)
        if parsed.scheme + "://" + parsed.netloc != production or parsed.query or parsed.fragment:
            fail("SEO/routes", "sitemap production origin", production, url)
    return urls

def machine_acceptance(base, profile, plan=None):
    if plan is None:
        activated_shell_acceptance(base, profile)
    else:
        site_machine_acceptance(base, profile, plan)
    production = profile["production_public_url"].rstrip('/')
    formal_course_routes = catalog_routes()
    list_metadata = formal_list_metadata(profile["locale"])
    raw_routes = [("/", "/", False, None), ("/tour/", "/tour/", True, None), ("/tour/list", "/tour/list", True, list_metadata)]
    raw_routes.extend((route, "/tour/", True, None) for route in formal_course_routes)
    for route, canonical_path, require_tour_shell, expected_list_metadata in raw_routes:
        status, body = request(base, route)
        if status != 200: fail("HTTP/routes", route, "HTTP 200", status)
        validate_raw_shell(body, route, canonical_path, profile["locale"], production, require_tour_shell, expected_list_metadata)
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
    coverage = activated_canonical_routes(profile["locale"])
    if coverage is None:
        fail("SEO/routes", "Content Scope", "formal activated canonical coverage", "missing")
    validate_sitemap(sitemap, production, coverage)

def main():
    if len(sys.argv) != 3:
        print(f"usage: {pathlib.Path(sys.argv[0]).name} <preview-url> <locale>", file=sys.stderr); return 2
    base, locale = sys.argv[1:]
    try:
        preview_url(base)
        identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        profile = profile_for(identity, locale)
        plan = browser_plan(locale)
        machine_acceptance(base, profile, plan)
        languages = registry()
        chrome_contract = CORE.shared_chrome_contract(locale, profile["production_public_url"], languages)
        chrome_check = lambda chrome: CORE.check_shared_chrome(chrome, chrome_contract)
        CORE.preview_acceptance(base, locale, profile, identity["shared"], languages, formal_descriptions(locale), formal_list_metadata(locale),
                                site_acceptance=(lambda chrome: site_browser_acceptance(chrome, base, profile, plan, chrome_check)) if plan["pages"] else None,
                                shared_chrome_check=chrome_check,
                                chrome_factory=preview_chrome_factory(base),
                                site_home=any(p["route"] == "/" for p in plan["pages"]))
    except (CORE.BrowserFailure, IDENTITY.IdentityError, OSError, KeyError, TypeError, ValueError, subprocess.SubprocessError) as exc:
        detail = str(exc)
        if not detail.startswith("stage="):
            detail = f"stage=browser check=rendered/interaction expected=PASS actual={detail}"
        print(f"[preview-browser] FAILED: {detail}", file=sys.stderr); return 1
    print(f"[preview-browser] Site route coverage: families={plan['families']} machine_pages={len(plan['pages'])} browser_samples={[p['route'] for p in browser_samples(plan)]}")
    for stage in ("preview identity", "SEO/routes", "Site v2 renderer/navigation/assets/code/runtime", "console/network errors", "desktop rendered surface", "editor Run / Format / Reset", "SPA", "mobile /tour/moretypes/1"):
        print(f"[preview-browser] {stage}: PASS")
    print("PREVIEW SURFACE ACCEPTANCE: PASS")
    return 0

if __name__ == "__main__": raise SystemExit(main())
