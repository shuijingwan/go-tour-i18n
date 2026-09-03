#!/usr/bin/env python3
"""Shared Chrome/HTTP acceptance for production and complete-locale preview.

Uses Chrome DevTools Protocol directly so the repository keeps its existing
Chrome-only browser baseline and adds no browser framework dependency.
"""

from __future__ import annotations

import base64
import csv
import hashlib
import importlib.util
import json
import os
import pathlib
import re
import shutil
import socket
import struct
import subprocess
import sys
import tempfile
import time
import urllib.parse
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parent.parent
IDENTITY_SPEC = importlib.util.spec_from_file_location(
    "production_identity_browser", ROOT / "scripts" / "production-identity.py"
)
IDENTITY = importlib.util.module_from_spec(IDENTITY_SPEC)
IDENTITY_SPEC.loader.exec_module(IDENTITY)


class BrowserFailure(RuntimeError):
    pass


def locale_list_metadata(locale):
    try:
        catalog = json.loads((ROOT / "internal" / "tour" / "ui" / f"{locale}.json").read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        raise BrowserFailure(f"cannot load list SEO catalog for {locale}: {exc}") from exc
    messages = catalog.get("messages", {})
    values = {}
    for name, key in (("title", "tour.list_title"), ("description", "tour.list_description"), ("heading", "tour.list_heading")):
        entry = messages.get(key, {})
        if entry.get("kind") != "plain" or not entry.get("text"):
            raise BrowserFailure(f"list SEO catalog {locale} has invalid {key}: {entry!r}")
        values[name] = entry["text"]
    return values


def formal_course_routes():
    try:
        with (ROOT / "data" / "tour-pages.tsv").open(encoding="utf-8", newline="") as source:
            routes = tuple("/tour" + row["route"] for row in csv.DictReader(source, delimiter="\t"))
    except OSError as exc:
        raise BrowserFailure(f"cannot read formal course route catalog: {exc}") from exc
    if len(routes) != 103 or len(set(routes)) != len(routes):
        raise BrowserFailure(f"formal course route catalog must contain 103 unique routes, got {len(routes)}")
    return routes


class WebSocket:
    def __init__(self, url):
        parsed = urllib.parse.urlsplit(url)
        self.sock = socket.create_connection((parsed.hostname, parsed.port), timeout=10)
        key = base64.b64encode(os.urandom(16)).decode()
        request = (
            f"GET {parsed.path}?{parsed.query} HTTP/1.1\r\n"
            f"Host: {parsed.hostname}:{parsed.port}\r\nUpgrade: websocket\r\n"
            f"Connection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
        )
        self.sock.sendall(request.encode())
        response = self._until(b"\r\n\r\n")
        if not response.startswith(b"HTTP/1.1 101"):
            raise BrowserFailure(f"DevTools websocket handshake failed: {response[:120]!r}")
        expected = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()).digest())
        headers = response.lower()
        if expected.lower() not in headers:
            raise BrowserFailure("DevTools websocket accept identity mismatch")

    def _until(self, marker):
        data = b""
        while marker not in data:
            chunk = self.sock.recv(4096)
            if not chunk:
                raise BrowserFailure("DevTools websocket closed")
            data += chunk
        return data

    def send(self, message):
        payload = json.dumps(message, separators=(",", ":")).encode()
        mask = os.urandom(4)
        length = len(payload)
        header = bytearray([0x81])
        if length < 126:
            header.append(0x80 | length)
        elif length < 65536:
            header.append(0x80 | 126); header.extend(struct.pack("!H", length))
        else:
            header.append(0x80 | 127); header.extend(struct.pack("!Q", length))
        header.extend(mask)
        header.extend(bytes(value ^ mask[index % 4] for index, value in enumerate(payload)))
        self.sock.sendall(header)

    def receive(self):
        first = self._read_exact(2)
        opcode = first[0] & 0x0F
        length = first[1] & 0x7F
        if length == 126:
            length = struct.unpack("!H", self._read_exact(2))[0]
        elif length == 127:
            length = struct.unpack("!Q", self._read_exact(8))[0]
        if first[1] & 0x80:
            mask = self._read_exact(4)
        else:
            mask = None
        payload = self._read_exact(length)
        if mask:
            payload = bytes(value ^ mask[index % 4] for index, value in enumerate(payload))
        if opcode == 8:
            raise BrowserFailure("DevTools websocket closed")
        if opcode == 9:
            self._send_control(10, payload)
            return self.receive()
        if opcode != 1:
            return self.receive()
        return json.loads(payload)

    def _read_exact(self, count):
        result = b""
        while len(result) < count:
            chunk = self.sock.recv(count - len(result))
            if not chunk:
                raise BrowserFailure("DevTools websocket closed")
            result += chunk
        return result

    def _send_control(self, opcode, payload):
        mask = os.urandom(4)
        self.sock.sendall(bytes([0x80 | opcode, 0x80 | len(payload)]) + mask + bytes(v ^ mask[i % 4] for i, v in enumerate(payload)))

    def close(self):
        self.sock.close()


class Chrome:
    def __init__(self):
        binary = shutil.which("google-chrome")
        if not binary:
            raise BrowserFailure("google-chrome is required")
        self.temp = pathlib.Path(tempfile.mkdtemp(prefix="go-tour-browser-acceptance-"))
        self.process = subprocess.Popen([
            binary, "--headless=new", "--no-sandbox", "--disable-gpu",
            "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter",
            "--noerrdialogs", "--no-first-run", "--remote-debugging-address=127.0.0.1",
            "--remote-debugging-port=0", f"--user-data-dir={self.temp}", "about:blank",
        ], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        active = self.temp / "DevToolsActivePort"
        for _ in range(100):
            if active.exists():
                break
            if self.process.poll() is not None:
                raise BrowserFailure("Chrome exited before DevTools became ready")
            time.sleep(0.05)
        else:
            raise BrowserFailure("Chrome DevTools did not become ready")
        active_lines = active.read_text().splitlines()
        port = int(active_lines[0])
        self.ws = WebSocket(f"ws://127.0.0.1:{port}{active_lines[1]}")
        self.next_id = 1
        self.events = []
        self.session_id = None
        self.current_route = "about:blank"
        target = self.call("Target.createTarget", {"url": "about:blank"})["targetId"]
        self.session_id = self.call("Target.attachToTarget", {"targetId": target, "flatten": True})["sessionId"]
        self.call("Page.enable")
        self.call("Runtime.enable")
        self.call("Network.enable")

    def call(self, method, params=None, timeout=20):
        identifier = self.next_id; self.next_id += 1
        message = {"id": identifier, "method": method, "params": params or {}}
        if self.session_id:
            message["sessionId"] = self.session_id
        self.ws.send(message)
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            self.ws.sock.settimeout(max(0.1, deadline - time.monotonic()))
            message = self.ws.receive()
            if message.get("id") == identifier:
                if "error" in message:
                    raise BrowserFailure(f"DevTools {method}: {message['error']}")
                return message.get("result", {})
            self.events.append(message)
        raise BrowserFailure(f"DevTools {method} timed out")

    def evaluate(self, expression, await_promise=False, check="unspecified evaluate"):
        result = self.call("Runtime.evaluate", {
            "expression": expression, "returnByValue": True,
            "awaitPromise": await_promise, "userGesture": True,
        }, timeout=60)
        details = result.get("exceptionDetails")
        if details:
            exception = result.get("result", {})
            description = exception.get("description") or exception.get("value") or exception.get("className") or "missing"
            frames = details.get("stackTrace", {}).get("callFrames", [])
            stack = [f"{frame.get('functionName') or '<anonymous>'}@{frame.get('url') or '<evaluate>'}:{frame.get('lineNumber', -1) + 1}:{frame.get('columnNumber', -1) + 1}" for frame in frames]
            source = details.get("url") or (frames[0].get("url") if frames else "<evaluate>")
            line = details.get("lineNumber", -1) + 1
            column = details.get("columnNumber", -1) + 1
            raise BrowserFailure(
                f"browser JavaScript exception: check={check!r} route={self.current_route!r} "
                f"text={details.get('text')!r} description={description!r} source={source!r} "
                f"line={line} column={column} stack={stack!r} expression={expression[:240]!r}"
            )
        return result.get("result", {}).get("value")

    def navigate(self, url, width, height):
        self.current_route = urllib.parse.urlsplit(url).path
        self.call("Emulation.setDeviceMetricsOverride", {
            "width": width, "height": height, "deviceScaleFactor": 1,
            "mobile": width <= 480,
        })
        self.events.clear()
        self.call("Page.navigate", {"url": url})
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            if self.evaluate("document.readyState") == "complete" and self.evaluate("document.body && document.body.innerText.length > 20"):
                time.sleep(2)
                return
            time.sleep(0.25)
        raise BrowserFailure(f"page did not render: {url}")

    def network_urls(self):
        return [event.get("params", {}).get("request", {}).get("url", "") for event in self.events if event.get("method") == "Network.requestWillBeSent"]

    def network_requests(self):
        requests = {}
        extras = {}
        for event in self.events:
            params = event.get("params", {})
            if event.get("method") == "Network.requestWillBeSent":
                requests[params.get("requestId")] = dict(params.get("request", {}))
            elif event.get("method") == "Network.requestWillBeSentExtraInfo":
                extras[params.get("requestId")] = params.get("headers", {})
        for request_id, headers in extras.items():
            if request_id in requests:
                requests[request_id]["headers"] = {**requests[request_id].get("headers", {}), **headers}
        return list(requests.values())

    def close(self):
        try:
            self.ws.close()
        finally:
            self.process.terminate()
            try:
                self.process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                self.process.kill()
            shutil.rmtree(self.temp, ignore_errors=True)


def assert_true(value, message):
    if not value:
        raise BrowserFailure(message)


def validate_playground_endpoint(url, expected_origin, expected_path, operation):
    """Validate the exact semantic identity of one Playground request URL."""
    actual = urllib.parse.urlsplit(url)
    expected = urllib.parse.urlsplit(expected_origin.rstrip("/"))
    assert_true(
        actual.scheme == expected.scheme and actual.netloc == expected.netloc,
        f"{operation} Playground origin mismatch: expected={expected.scheme}://{expected.netloc} actual={url}",
    )
    assert_true(actual.path == expected_path, f"{operation} Playground path mismatch: expected={expected_path} actual={url}")
    assert_true(not actual.fragment, f"{operation} Playground URL has unexpected fragment: {url}")
    try:
        query = urllib.parse.parse_qsl(actual.query, keep_blank_values=True, strict_parsing=True)
    except ValueError as error:
        raise BrowserFailure(f"{operation} Playground query is malformed: actual={url}: {error}") from error
    expected_query = [("backend", "")] if operation == "compile" else []
    assert_true(query == expected_query,
                f"{operation} Playground query mismatch: expected={expected_query} actual={query} url={url}")


def playground_requests(requests, expected_origin, compile_path, fmt_path):
    posts = [request for request in requests if request.get("method") == "POST"]
    compile_requests = [request for request in posts if urllib.parse.urlsplit(request.get("url", "")).path == compile_path]
    fmt_requests = [request for request in posts if urllib.parse.urlsplit(request.get("url", "")).path == fmt_path]
    assert_true(len(compile_requests) == 1, f"compile Playground request count mismatch: actual={compile_requests}")
    assert_true(len(fmt_requests) == 1, f"fmt Playground request count mismatch: actual={fmt_requests}")
    validate_playground_endpoint(compile_requests[0]["url"], expected_origin, compile_path, "compile")
    validate_playground_endpoint(fmt_requests[0]["url"], expected_origin, fmt_path, "fmt")
    return compile_requests + fmt_requests


EDITOR_STATE_EXPRESSION = """(() => {
  const textarea = document.querySelector('textarea[ui-codemirror]');
  const scope = angular.element(textarea).scope();
  const lessons = scope.toc.lessons.$$v || scope.toc.lessons;
  const file = lessons[scope.lessonId].Pages[scope.curPage - 1].Files[scope.curFile];
  const model = angular.element(textarea).controller('ngModel');
  return {displayed:file && document.querySelector('.CodeMirror').CodeMirror.getValue(),
    content:file.Content, model:model.$modelValue, view:model.$viewValue,
    original:file.OrigContent, hash:file.Hash, stored:localStorage.getItem(file.Hash)};
})()"""


def validate_editor_modified(state, changed):
    assert_true(changed != state["original"], "editor test change unexpectedly equals OrigContent")
    assert_true(state["displayed"] == changed and state["content"] == changed and
                state["model"] == changed and state["view"] == changed,
                f"editor change did not synchronize display and Angular model: {state}")


def validate_reset_state(state, expected_original):
    assert_true(state["original"] == expected_original,
                "Reset expected original source does not match file.OrigContent")
    assert_true(state["content"] == expected_original and state["model"] == expected_original,
                f"Reset did not restore Angular model: {state}")
    assert_true(state["displayed"] == expected_original and state["view"] == expected_original,
                f"Reset did not restore CodeMirror view: {state}")
    assert_true(state["stored"] == expected_original, f"Reset did not persist OrigContent to localStorage: {state}")


def wait_for_editor_reset(chrome, expected_original, timeout=5):
    deadline = time.monotonic() + timeout
    matching = 0
    last = None
    while time.monotonic() < deadline:
        last = chrome.evaluate(EDITOR_STATE_EXPRESSION, check="inspect Reset synchronization")
        try:
            validate_reset_state(last, expected_original)
        except BrowserFailure:
            matching = 0
        else:
            matching += 1
            if matching == 2:
                return last
        time.sleep(0.05)
    raise BrowserFailure(f"Reset model/view synchronization timed out: expected={expected_original!r} actual={last!r}")


def browser_ad_gate(snapshot):
    """Filled and unfilled are both valid; mount, loader and slot remain mandatory."""
    return snapshot.get("mount") == 1 and snapshot.get("ad") == 1 and bool(snapshot.get("loader"))


def validate_rendered_identity(identity, base, locale, requested_path, expected_final_path=None, canonical_origin=None,
                               expected_rendered_route=None, expected_description=None):
    if expected_final_path is not None:
        assert_true(identity["path"] == expected_final_path,
                    f"{requested_path}: rendered path mismatch: expected={expected_final_path} actual={identity['path']}")
    assert_true(identity["lang"] == locale, f"{requested_path}: html lang mismatch")
    assert_true(identity["origin"] == base.rstrip("/"), f"{requested_path}: production hostname mismatch")
    expected_canonical = identity["href"] if expected_final_path is None else (canonical_origin or base.rstrip("/")) + expected_final_path
    assert_true(identity["canonical"] == expected_canonical,
                f"{requested_path}: canonical mismatch: expected={expected_canonical} actual={identity['canonical']}")
    assert_true(identity["title"] and identity["description"], f"{requested_path}: SEO metadata missing")
    if expected_rendered_route is not None:
        assert_true(identity["renderedRoute"] == expected_rendered_route,
                    f"{requested_path}: data-tour-rendered-route mismatch: expected={expected_rendered_route} actual={identity['renderedRoute']}")
        assert_true(identity["heading"] and identity["heading"] in identity["title"],
                    f"{requested_path}: title is not route-specific: title={identity['title']} heading={identity['heading']}")
    if expected_description is not None:
        assert_true(identity["description"] == expected_description,
                    f"{requested_path}: description mismatch: expected={expected_description} actual={identity['description']}")


def page_identity(chrome, base, locale, requested_path, width, height, canonical_origin=None, expected_final_path=None,
                  expected_rendered_route=None, expected_description=None, expected_title=None):
    url = urllib.parse.urljoin(base, requested_path.lstrip("/"))
    chrome.navigate(url, width, height)
    identity = chrome.evaluate("""(() => ({
      lang: document.documentElement.lang,
      href: location.href,
      origin: location.origin,
      path: location.pathname,
      renderedRoute: document.documentElement.getAttribute('data-tour-rendered-route') || '',
      heading: document.querySelector('.slide-content h2,h2')?.textContent.trim() || '',
      canonical: document.querySelector('link[rel="canonical"]')?.href || '',
      title: document.title,
      description: document.querySelector('meta[name="description"]')?.content || '',
      overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth
    }))()""", check=f"page identity snapshot requested={requested_path} expected_final={expected_final_path}")
    validate_rendered_identity(identity, base, locale, requested_path, expected_final_path, canonical_origin,
                               expected_rendered_route, expected_description)
    if expected_title is not None:
        assert_true(identity["title"] == expected_title,
                    f"{requested_path}: title mismatch: expected={expected_title} actual={identity['title']}")
    if width <= 480:
        assert_true(identity["overflow"] <= 2, f"{requested_path}: unexpected page-level horizontal overflow")


def validate_rendered_list(chrome, list_metadata, expected_page_routes):
    snapshot = chrome.evaluate("""(() => ({
      wrappers: document.querySelectorAll('.list-wrapper').length,
      heading: document.querySelector('.list-wrapper .page-header h1')?.textContent.trim() || '',
      modules: document.querySelectorAll('.list-wrapper .module').length,
      articleRoutes: [...document.querySelectorAll('.list-wrapper a.lesson-title[href^="/tour/"]')]
        .map(a => new URL(a.getAttribute('href'), location.origin).pathname),
      pageRoutes: [...document.querySelectorAll('.toc .toc-page a[href^="/tour/"]')]
        .map(a => new URL(a.getAttribute('href'), location.origin).pathname)
    }))()""", check="inspect rendered course directory")
    assert_true(snapshot["wrappers"] == 1, f"/tour/list: duplicated or missing directory body: {snapshot}")
    assert_true(snapshot["heading"] == list_metadata["heading"], f"/tour/list: list heading mismatch: {snapshot}")
    assert_true(snapshot["modules"] == 5, f"/tour/list: module directory mismatch: {snapshot}")
    expected_article_routes = sorted({route.rsplit('/', 1)[0] for route in expected_page_routes})
    assert_true(sorted(snapshot["articleRoutes"]) == expected_article_routes,
                f"/tour/list: article directory route mismatch: expected={expected_article_routes} actual={snapshot}")
    assert_true(sorted(snapshot["pageRoutes"]) == sorted(expected_page_routes),
                f"/tour/list: Page directory route mismatch: expected={len(expected_page_routes)} unique formal routes actual={snapshot}")


def acceptance(base, locale, profile, shared):
    list_metadata = locale_list_metadata(locale)
    chrome = Chrome()
    try:
        for path in ("/", "/tour/", "/tour/list", "/tour/welcome/1", "/tour/basics/11"):
            is_list = path == "/tour/list"
            page_identity(chrome, base, locale, path, 1280, 800,
                          expected_description=list_metadata["description"] if is_list else None,
                          expected_title=list_metadata["title"] if is_list else None)
            if is_list:
                validate_rendered_list(chrome, list_metadata, formal_course_routes())
        chrome.navigate(base, 375, 812)
        language = chrome.evaluate("""(() => ({
          count: document.querySelectorAll('.site-language-list li').length,
          current: document.querySelectorAll('.site-language-list [aria-current="page"]').length,
          links: document.querySelectorAll('.site-language-list a[href]').length
        }))()""")
        assert_true(language["count"] >= 2 and language["current"] == 1 and language["links"] >= 1, "language selector identity failed")

        editor_url = urllib.parse.urljoin(base, "tour/basics/11")
        chrome.navigate(editor_url, 1280, 800)
        shared_assets = json.dumps(shared["shared_assets_public_origin"].rstrip("/") + "/")
        editor = chrome.evaluate(f"""(() => ({{
          run: !!document.querySelector('#run'), format: !!document.querySelector('#format'),
          reset: !!document.querySelector('#reset'), cm: !!document.querySelector('.CodeMirror')?.CodeMirror,
          mount: document.querySelectorAll('[data-go-dev-course-ad]').length,
          ad: document.querySelectorAll('[data-go-dev-course-ad] ins.adsbygoogle').length,
          loader: [...document.scripts].some(s => /adsbygoogle/.test(s.src)),
          shared: performance.getEntriesByType('resource').some(e => e.name.startsWith({shared_assets}))
        }}))()""")
        assert_true(all(editor[key] for key in ("run", "format", "reset", "cm")) and browser_ad_gate(editor), "editor/ad browser identity failed")
        if profile["shared_assets_policy"] == "shared-cloudflare":
            assert_true(editor["shared"], "shared assets were not requested")
        # Filled and unfilled ads are both accepted: the gate is mount + loader + request opportunity.
        edit = chrome.evaluate("""(() => {
          const cm = document.querySelector('.CodeMirror').CodeMirror;
          const malformed = 'package main\\nfunc main(){println("browser acceptance")}\\n';
          window.__productionAcceptanceMalformed = malformed;
          cm.setValue(malformed);
          return true;
        })()""")
        assert_true(edit, "could not prepare editor interaction")
        initial = chrome.evaluate(EDITOR_STATE_EXPRESSION, check="inspect production editor change")
        original = initial["original"]
        validate_editor_modified(initial, chrome.evaluate("window.__productionAcceptanceMalformed"))
        chrome.evaluate("document.querySelector('#format').click(); true")
        time.sleep(3)
        formatted = chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.getValue()")
        assert_true(formatted != chrome.evaluate("window.__productionAcceptanceMalformed") and "func main() {" in formatted, "Format did not update source")
        chrome.evaluate("document.querySelector('#reset').click(); true")
        wait_for_editor_reset(chrome, original)
        chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.setValue('package main\\nfunc main(){println(\\\"BROWSER_ACCEPTANCE_OK\\\")}\\n'); true")
        chrome.evaluate("document.querySelector('#run').click(); true")
        time.sleep(8)
        runtime = chrome.evaluate("""(() => ({
          output: [...document.querySelectorAll('.output')].map(x => x.innerText).join('\\n'),
          mount: document.querySelectorAll('[data-go-dev-course-ad]').length,
          path: location.pathname
        }))()""")
        requests = chrome.network_requests()
        assert_true("BROWSER_ACCEPTANCE_OK" in runtime["output"], "Run produced no browser-visible expected result")
        playground = shared["playground_public_origin"].rstrip("/")
        playground_posts = playground_requests(requests, playground, "/compile", "/fmt")
        expected_origin = base.rstrip("/")
        origins = [{key.lower(): value for key, value in request.get("headers", {}).items()}.get("origin") for request in playground_posts]
        assert_true(len(origins) >= 2 and all(origin == expected_origin for origin in origins), f"Playground POST Origin mismatch: {origins}")
        socket_status = chrome.evaluate("fetch('/socket').then(r => r.status)", await_promise=True)
        assert_true(socket_status == 404, "/socket browser boundary failed")

        before = chrome.evaluate("location.pathname")
        chrome.evaluate("document.querySelector('.next-page').click(); true")
        time.sleep(3)
        after = chrome.evaluate("location.pathname")
        assert_true(after != before, "SPA next-page transition did not change route")
        assert_true(chrome.evaluate("document.querySelectorAll('[data-go-dev-course-ad]').length") == 1, "SPA transition lost or duplicated course-ad mount")

        page_identity(chrome, base, locale, "/tour/moretypes/1", 375, 812)
        before = chrome.evaluate("location.pathname")
        chrome.evaluate("document.querySelector('.next-page').click(); true")
        time.sleep(2)
        assert_true(chrome.evaluate("location.pathname") != before, "mobile SPA transition failed")
    finally:
        chrome.close()


def preview_acceptance(base, locale, profile, shared, registry, descriptions, list_metadata):
    """Run browser checks whose preview identity intentionally differs from production."""
    canonical_origin = profile["production_public_url"].rstrip("/")
    chrome = Chrome()
    try:
        rendered_routes = (("/", "/"), ("/tour/", "/tour/welcome/1"), ("/tour/list", "/tour/list"),
                           ("/tour/welcome/1", "/tour/welcome/1"), ("/tour/basics/11", "/tour/basics/11"))
        for path, final_path in rendered_routes:
            course_route = final_path if re.match(r"^/tour/[^/]+/[1-9][0-9]*$", final_path) else None
            is_list = final_path == "/tour/list"
            page_identity(chrome, base, locale, path, 1280, 800, canonical_origin, final_path, course_route,
                          descriptions.get(course_route) if course_route else (list_metadata["description"] if is_list else None),
                          list_metadata["title"] if is_list else None)
            if is_list:
                validate_rendered_list(chrome, list_metadata, formal_course_routes())
            shell = chrome.evaluate("""(() => ({header:document.querySelectorAll('.top-bar').length,
              footer:document.querySelectorAll('.site-footer').length, body:(document.body?.innerText||'').trim(),
              overflow:document.documentElement.scrollWidth-document.documentElement.clientWidth}))()""")
            assert_true(shell["header"] == 1 and shell["footer"] == 1 and len(shell["body"]) > 20,
                        f"{path}: rendered shell missing or duplicated: {shell}")
            assert_true(shell["overflow"] <= 2, f"{path}: document horizontal overflow={shell['overflow']}")

        chrome.navigate(base, 1280, 800)
        languages = chrome.evaluate("""(() => [...document.querySelectorAll('.site-language-list li')].map(li => {
          const n=li.querySelector('a,[aria-current="page"]')||li;
          return {text:n.textContent.trim(),href:n.href||'',current:n.getAttribute('aria-current')==='page'};}))()""")
        assert_true(len(languages) == len(registry), f"language selector count: expected={len(registry)} actual={len(languages)}")
        assert_true(sum(item["current"] for item in languages) == 1, f"language selector current identity: {languages}")
        for got, want in zip(languages, registry):
            assert_true(want["english_name"] in got["text"] and want["autonym"] in got["text"],
                        f"language selector order/label: expected={want} actual={got}")
            assert_true(got["current"] == (want["locale"] == locale), f"language selector current: expected={want} actual={got}")
            if not got["current"]:
                assert_true(got["href"] == want["url"], f"language selector URL: expected={want['url']} actual={got['href']}")

        for mobile in (False, True):
            chrome.navigate(urllib.parse.urljoin(base, "tour/basics/11"), 375 if mobile else 1280, 812 if mobile else 800)
            editor = chrome.evaluate("""(() => ({run:!!document.querySelector('#run'),format:!!document.querySelector('#format'),
              reset:!!document.querySelector('#reset'),cm:!!document.querySelector('.CodeMirror')?.CodeMirror,
              mount:document.querySelectorAll('[data-go-dev-course-ad]').length,
              ad:document.querySelectorAll('[data-go-dev-course-ad] ins.adsbygoogle').length,
              loader:[...document.scripts].some(s=>/adsbygoogle/.test(s.src))}))()""")
            assert_true(all(editor[key] for key in ("run", "format", "reset", "cm")), f"editor controls missing: {editor}")
            chrome.evaluate("(() => {const cm=document.querySelector('.CodeMirror').CodeMirror;"
                            "window.__previewMalformed='package main\\nfunc main(){println(\"browser acceptance\")}\\n';"
                            "cm.setValue(window.__previewMalformed);return true})()",
                            check="prepare malformed editor source")
            changed_state = chrome.evaluate(EDITOR_STATE_EXPRESSION, check="inspect preview editor change")
            original = changed_state["original"]
            validate_editor_modified(changed_state, chrome.evaluate("window.__previewMalformed"))
            chrome.evaluate("document.querySelector('#format').click();true"); time.sleep(3)
            formatted = chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.getValue()")
            assert_true(formatted != chrome.evaluate("window.__previewMalformed") and "func main() {" in formatted, "Format did not update source")
            chrome.evaluate("document.querySelector('#reset').click();true")
            wait_for_editor_reset(chrome, original)
            chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.setValue('package main\\nfunc main(){println(\\\"BROWSER_ACCEPTANCE_OK\\\")}\\n');true")
            chrome.evaluate("document.querySelector('#run').click();true"); time.sleep(8)
            output = chrome.evaluate("[...document.querySelectorAll('.output')].map(x=>x.innerText).join('\\n')")
            assert_true("BROWSER_ACCEPTANCE_OK" in output, "Run produced no browser-visible expected result")
            requests = chrome.network_requests()
            urls = [request.get("url", "") for request in requests]
            origin = base.rstrip("/")
            playground_requests(requests, origin, "/_/compile", "/_/fmt")
            assert_true(not any(urllib.parse.urlsplit(url).path.startswith('/socket') for url in urls), "editor used /socket")
            assert_true(chrome.evaluate("fetch('/socket').then(r=>r.status)", await_promise=True) == 404, "/socket browser boundary failed")
            before = chrome.evaluate("location.pathname")
            chrome.evaluate("document.querySelector('.next-page').click();true"); time.sleep(3)
            after = chrome.evaluate("location.pathname")
            assert_true(after != before, "SPA next-page transition did not change route")
            spa = chrome.evaluate("""(() => ({canonical:document.querySelector('link[rel="canonical"]')?.href||'',
              header:document.querySelectorAll('.top-bar').length,footer:document.querySelectorAll('.site-footer').length,
              next:!!document.querySelector('.next-page'),body:(document.body?.innerText||'').trim()}))()""")
            assert_true(spa["canonical"] == canonical_origin + after, f"SPA canonical mismatch: {spa}")
            assert_true(spa["header"] == 1 and spa["footer"] == 1 and spa["next"] and len(spa["body"]) > 20,
                        f"SPA DOM/shell failed: {spa}")

        mobile_routes = (("/", "/"), ("/tour/", "/tour/welcome/1"), ("/tour/list", "/tour/list"),
                         ("/tour/welcome/1", "/tour/welcome/1"), ("/tour/moretypes/1", "/tour/moretypes/1"))
        for path, final_path in mobile_routes:
            course_route = final_path if re.match(r"^/tour/[^/]+/[1-9][0-9]*$", final_path) else None
            is_list = final_path == "/tour/list"
            page_identity(chrome, base, locale, path, 375, 812, canonical_origin, final_path, course_route,
                          descriptions.get(course_route) if course_route else (list_metadata["description"] if is_list else None),
                          list_metadata["title"] if is_list else None)
    finally:
        chrome.close()


def main():
    if len(sys.argv) != 3:
        print(f"usage: {pathlib.Path(sys.argv[0]).name} <production-public-url> <locale>", file=sys.stderr)
        return 2
    base, locale = sys.argv[1:]
    parsed = urllib.parse.urlsplit(base)
    if parsed.scheme != "https" or parsed.path != "/" or parsed.query or parsed.fragment:
        print("[production-browser] ERROR: public URL must be an HTTPS origin ending in /", file=sys.stderr)
        return 1
    try:
        identity = IDENTITY.load_identity(ROOT / "production" / "identity.json")
        profiles = [profile for profile in identity["locales"] if profile["locale"] == locale]
        if len(profiles) != 1:
            raise BrowserFailure(f"unknown formal production locale: {locale}")
        profile = profiles[0]
        if base != profile["production_public_url"]:
            raise BrowserFailure("public URL does not match formal production identity")
        acceptance(base, locale, profile, identity["shared"])
    except (BrowserFailure, IDENTITY.IdentityError, OSError, KeyError, TypeError) as exc:
        print(f"[production-browser] FAILED: {exc}", file=sys.stderr)
        return 1
    print("[production-browser] desktop routes: PASS")
    print("[production-browser] mobile /tour/moretypes/1: PASS")
    print("[production-browser] Run / Format / Reset / SPA / ads: PASS")
    print("PRODUCTION BROWSER ACCEPTANCE: PASS")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
