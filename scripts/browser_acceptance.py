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


class PermanentBrowserFailure(BrowserFailure):
    """A shell identity error that must not become a navigation retry."""


# Initial navigation may be repeated only while the document has not reached a
# minimally inspectable DOM.  A single command may wait long enough for the
# observed 20+ second cold MISS, while attempts and post-commit readiness stay
# independently bounded.
RENDER_ATTEMPTS = 3
RENDER_ATTEMPT_TIMEOUT = 10
NAVIGATION_COMMAND_TIMEOUT = 30
SEMANTIC_CONVERGENCE_TIMEOUT = 25
PROJECT_PAGE_ATTEMPTS = 3
PROJECT_TRANSIENT_HTTP_STATUSES = {522, 525}
PROJECT_TRANSIENT_NETWORK_ERRORS = {
    "net::ERR_NAME_NOT_RESOLVED",
    "net::ERR_CONNECTION_FAILED",
    "net::ERR_CONNECTION_RESET",
    "net::ERR_CONNECTION_CLOSED",
    "net::ERR_CONNECTION_TIMED_OUT",
    "net::ERR_TIMED_OUT",
    "net::ERR_NETWORK_CHANGED",
    "net::ERR_INTERNET_DISCONNECTED",
    "net::ERR_EMPTY_RESPONSE",
    "net::ERR_SSL_PROTOCOL_ERROR",
    "net::ERR_SSL_VERSION_OR_CIPHER_MISMATCH",
    "net::ERR_PROXY_CONNECTION_FAILED",
    "net::ERR_TUNNEL_CONNECTION_FAILED",
    "net::ERR_HTTP2_PROTOCOL_ERROR",
    "net::ERR_QUIC_PROTOCOL_ERROR",
}
PROJECT_REQUIRED_RESOURCE_TYPES = {"Document", "Script", "Stylesheet", "XHR", "Fetch"}
DOCUMENT_RESPONSE_HEADERS = {"cf-cache-status", "age", "cf-ray", "content-type", "cache-control"}


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


def publication_policy(locale):
    """Read the publication decision from the Go policy authority."""
    command = ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "policy", "publication", "--locale", locale]
    try:
        result = subprocess.run(command, cwd=ROOT, text=True, capture_output=True, timeout=60, check=False)
    except (OSError, subprocess.SubprocessError) as exc:
        raise BrowserFailure(f"read publication policy for {locale}: {exc}") from exc
    if result.returncode != 0:
        raise BrowserFailure(f"read publication policy for {locale}: exit={result.returncode} stderr={result.stderr.strip()!r}")
    try:
        policy = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise BrowserFailure(f"read publication policy for {locale}: invalid JSON") from exc
    if (not isinstance(policy, dict) or set(policy) != {"locale", "publication", "tour_ads_enabled"} or
            policy["locale"] != locale or policy["publication"] not in ("standard", "go-local") or
            not isinstance(policy["tour_ads_enabled"], bool)):
        raise BrowserFailure(f"read publication policy for {locale}: invalid result {policy!r}")
    return policy


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


def chrome_command(binary, temp, proxy_server=None):
    command = [
        binary, "--headless=new", "--no-sandbox", "--disable-gpu",
        "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter",
        "--noerrdialogs", "--no-first-run", "--remote-debugging-address=127.0.0.1",
        "--remote-debugging-port=0", f"--user-data-dir={temp}",
    ]
    if proxy_server is not None:
        command.append(f"--proxy-server={proxy_server}")
    command.append("about:blank")
    return command


class Chrome:
    def __init__(self, proxy_server=None):
        binary = shutil.which("google-chrome")
        if not binary:
            raise BrowserFailure("google-chrome is required")
        self.temp = pathlib.Path(tempfile.mkdtemp(prefix="go-tour-browser-acceptance-"))
        self.process = subprocess.Popen(chrome_command(binary, self.temp, proxy_server), stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
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
        self.current_navigation = {"requested_url": "about:blank"}
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
            try:
                message = self.ws.receive()
            except socket.timeout:
                continue
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

    def render_readiness(self, timeout=RENDER_ATTEMPT_TIMEOUT):
        """Wait only for a DOM that can be subjected to semantic acceptance.

        `complete` waits for the window load event, including third-party ads
        and analytics resources.  `interactive` plus meaningful body text is
        sufficient to begin the strict DOM and behavior checks that follow.
        """
        deadline = time.monotonic() + timeout
        latest = {"readyState": None, "bodyTextLength": None, "location": None}
        while time.monotonic() < deadline:
            try:
                latest = self.evaluate("""(() => ({
                  readyState: document.readyState,
                  bodyTextLength: document.body ? document.body.innerText.length : 0,
                  location: location.href
                }))()""", check="render readiness")
            except BrowserFailure as exc:
                latest = {"readyState": None, "bodyTextLength": None, "location": None,
                          "evaluateError": str(exc)}
            else:
                if latest["readyState"] in ("interactive", "complete") and latest["bodyTextLength"] > 20:
                    return True, latest
            time.sleep(0.25)
        return False, latest

    def navigate(self, url, width, height):
        self.current_route = urllib.parse.urlsplit(url).path or "/"
        self.call("Emulation.setDeviceMetricsOverride", {
            "width": width, "height": height, "deviceScaleFactor": 1,
            "mobile": width <= 480,
        })
        last = {"readyState": None, "bodyTextLength": None, "location": None}
        navigation = {}
        for attempt in range(1, RENDER_ATTEMPTS + 1):
            self.events.clear()
            self.current_navigation = {"requested_url": url, "attempt": attempt}
            try:
                navigation = self.call("Page.navigate", {"url": url}, timeout=NAVIGATION_COMMAND_TIMEOUT)
                self.current_navigation.update({
                    key: navigation[key] for key in ("loaderId", "frameId") if key in navigation
                })
                ready, last = self.render_readiness()
            except BrowserFailure as exc:
                ready = False
                last = {"readyState": None, "bodyTextLength": None, "location": None,
                        "navigationError": str(exc)}
            if ready:
                return
            last = {"attempt": attempt, **last}
            if attempt < RENDER_ATTEMPTS:
                time.sleep(1)
        transport = {key: navigation.get(key) for key in ("errorText", "isDownload", "loaderId", "frameId") if key in navigation}
        raise BrowserFailure(
            f"page did not render: url={url!r} route={self.current_route!r} "
            f"attempt={last['attempt']}/{RENDER_ATTEMPTS} readyState={last.get('readyState')!r} "
            f"bodyTextLength={last.get('bodyTextLength')!r} location={last.get('location')!r} "
            f"readinessError={last.get('evaluateError') or last.get('navigationError')!r} navigation={transport!r}"
        )

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

    def main_document_response_evidence(self):
        """Return bounded, allowlisted evidence from the current main Document events."""
        navigation = dict(getattr(self, "current_navigation", {}) or {})
        loader_id = navigation.get("loaderId")
        frame_id = navigation.get("frameId")
        requests = {}
        responses = []
        for event in self.events:
            method = event.get("method")
            params = event.get("params", {})
            if params.get("type") != "Document":
                continue
            if method == "Network.requestWillBeSent":
                requests[params.get("requestId")] = params
            elif method == "Network.responseReceived":
                responses.append(params)
        if loader_id:
            responses = [params for params in responses if params.get("loaderId") == loader_id]
        elif frame_id:
            responses = [params for params in responses if params.get("frameId") == frame_id]
        elif len(responses) != 1:
            responses = []
        evidence = {
            "requested_url": navigation.get("requested_url"),
            "loader_id": loader_id,
            "frame_id": frame_id,
        }
        if not responses:
            evidence["response_event"] = "missing"
            return evidence
        params = responses[-1]
        response = params.get("response", {})
        request_id = params.get("requestId")
        request = requests.get(request_id, {}).get("request", {})
        headers = {}
        for name, value in response.get("headers", {}).items():
            normalized = str(name).lower()
            if normalized in DOCUMENT_RESPONSE_HEADERS:
                headers[normalized] = str(value)[:512]
        evidence.update({
            "request_id": request_id,
            "loader_id": params.get("loaderId") or loader_id,
            "frame_id": params.get("frameId") or frame_id,
            "request_url": request.get("url"),
            "final_response_url": response.get("url"),
            "http_status": response.get("status"),
            "response_headers": headers,
        })
        return evidence

    def project_network_transients(self, origins):
        """Return required project-resource transport evidence for this page.

        Third-party ads/fonts/analytics are deliberately outside this set.
        Deterministic HTTP failures are also excluded, so they cannot trigger
        a reload that might hide an application or publication error.
        """
        expected = {
            (parsed.scheme, parsed.netloc)
            for parsed in (urllib.parse.urlsplit(value.rstrip("/")) for value in origins)
        }
        requests = {}
        finished = set()
        responses = set()
        failures = []
        for event in self.events:
            method = event.get("method")
            params = event.get("params", {})
            request_id = params.get("requestId")
            if method == "Network.requestWillBeSent":
                requests[request_id] = {
                    "url": params.get("request", {}).get("url", ""),
                    "type": params.get("type", ""),
                }
            elif method == "Network.responseReceived":
                responses.add(request_id)
                response = params.get("response", {})
                request = requests.get(request_id, {})
                url = response.get("url") or request.get("url", "")
                resource_type = params.get("type") or request.get("type", "")
                parsed = urllib.parse.urlsplit(url)
                status = response.get("status")
                if ((parsed.scheme, parsed.netloc) in expected and
                        resource_type in PROJECT_REQUIRED_RESOURCE_TYPES and
                        status in PROJECT_TRANSIENT_HTTP_STATUSES):
                    failures.append(f"HTTP {int(status)} {resource_type} {url}")
            elif method == "Network.loadingFinished":
                finished.add(request_id)
            elif method == "Network.loadingFailed":
                finished.add(request_id)
                request = requests.get(request_id, {})
                url = request.get("url", "")
                resource_type = params.get("type") or request.get("type", "")
                error_text = params.get("errorText", "")
                parsed = urllib.parse.urlsplit(url)
                if ((parsed.scheme, parsed.netloc) in expected and
                        resource_type in PROJECT_REQUIRED_RESOURCE_TYPES and
                        error_text in PROJECT_TRANSIENT_NETWORK_ERRORS):
                    failures.append(f"{error_text} {resource_type} {url}")
        for request_id, request in requests.items():
            parsed = urllib.parse.urlsplit(request["url"])
            if ((parsed.scheme, parsed.netloc) in expected and
                    request["type"] in PROJECT_REQUIRED_RESOURCE_TYPES and
                    request_id not in finished and request_id not in responses):
                failures.append(f"pending {request['type']} {request['url']}")
        return failures

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


def is_tour_ad_request(request):
    host = urllib.parse.urlsplit(request.get("url", "")).hostname or ""
    return host == "googleads.g.doubleclick.net" or host.endswith(".googlesyndication.com")


def is_course_ad_resource(request):
    path = urllib.parse.urlsplit(request.get("url", "")).path
    return path.endswith("/tour/static/go-dev/course-ad.js") or path.endswith("/tour/static/go-dev/course-ad.css")


def browser_ad_gate(snapshot, requests, tour_ads_enabled):
    """Assert the policy-specific Tour ad contract without requiring a fill."""
    request_opportunity = any(is_tour_ad_request(request) for request in requests)
    helper_requested = bool(snapshot.get("helper")) or any(is_course_ad_resource(request) for request in requests)
    present = (snapshot.get("mount") == 1 and snapshot.get("ad") == 1 and bool(snapshot.get("loader")) and
               helper_requested and request_opportunity)
    # A legacy/static directive host may remain in the DOM. It is harmless only
    # when it is empty and has none of the attributes the helper adds at mount.
    absent = (snapshot.get("mount") in (0, 1) and snapshot.get("ad") == 0 and not snapshot.get("loader") and
              not helper_requested and not request_opportunity and bool(snapshot.get("empty_mount")))
    return present if tour_ads_enabled else absent


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
        if re.match(r"^/tour/[^/]+/[1-9][0-9]*$", expected_rendered_route):
            assert_true(identity["heading"] and identity["heading"] in identity["title"],
                        f"{requested_path}: title is not route-specific: title={identity['title']} heading={identity['heading']}")
    if expected_description is not None:
        assert_true(identity["description"] == expected_description,
                    f"{requested_path}: description mismatch: expected={expected_description} actual={identity['description']}")


PAGE_IDENTITY_EXPRESSION = """(() => ({
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
}))()"""


def identity_failure_evidence(chrome, identity):
    document = {}
    if isinstance(identity, dict):
        for key in ("lang", "href", "origin", "path", "canonical", "title", "description",
                    "renderedRoute", "heading"):
            if key in identity:
                value = identity[key]
                document[key] = value[:512] if isinstance(value, str) else value
    network = {"response_event": "unavailable"}
    collector = getattr(chrome, "main_document_response_evidence", None) if chrome is not None else None
    if callable(collector):
        try:
            candidate = collector()
        except Exception as exc:
            network = {"collection_error": str(exc)[:512]}
        else:
            if isinstance(candidate, dict):
                network = candidate
    return {"network": network, "document": document}


def validate_identity_invariants(identity, base, locale, requested_path, chrome=None):
    if not isinstance(identity, dict):
        raise PermanentBrowserFailure(f"{requested_path}: page identity snapshot is not an object: {identity!r}")
    if identity.get("lang") != locale:
        raise PermanentBrowserFailure(
            f"{requested_path}: html lang mismatch: expected={locale} actual={identity.get('lang')}; "
            f"mainDocument={identity_failure_evidence(chrome, identity)!r}"
        )
    if identity.get("origin") != base.rstrip("/"):
        raise PermanentBrowserFailure(
            f"{requested_path}: production hostname mismatch: expected={base.rstrip('/')} "
            f"actual={identity.get('origin')}; mainDocument={identity_failure_evidence(chrome, identity)!r}"
        )


def wait_for_rendered_identity(chrome, base, locale, requested_path, canonical_origin=None, expected_final_path=None,
                               expected_rendered_route=None, expected_description=None, expected_title=None,
                               width=1280, timeout=SEMANTIC_CONVERGENCE_TIMEOUT, post_identity_check=None):
    """Wait for Angular route metadata to converge without reloading the page."""
    deadline = time.monotonic() + timeout
    last = None
    last_failure = "no identity snapshot"
    while True:
        try:
            last = chrome.evaluate(
                PAGE_IDENTITY_EXPRESSION,
                check=f"page identity convergence requested={requested_path} expected_final={expected_final_path}",
            )
        except BrowserFailure as exc:
            last_failure = str(exc)
        else:
            # lang/origin are shell identity, not route-hydration state.  They
            # must never become reload/retry candidates.
            validate_identity_invariants(last, base, locale, requested_path, chrome)
            try:
                validate_rendered_identity(last, base, locale, requested_path, expected_final_path,
                                           canonical_origin, expected_rendered_route, expected_description)
                if expected_title is not None:
                    assert_true(last["title"] == expected_title,
                                f"{requested_path}: title mismatch: expected={expected_title} actual={last['title']}")
                if width <= 480:
                    assert_true(last["overflow"] <= 2,
                                f"{requested_path}: unexpected page-level horizontal overflow")
                if post_identity_check is not None:
                    post_identity_check()
            except BrowserFailure as exc:
                last_failure = str(exc)
            else:
                return last
        if time.monotonic() >= deadline:
            break
        time.sleep(0.1)
    raise BrowserFailure(
        f"{requested_path}: semantic convergence timed out after {timeout}s: "
        f"lastFailure={last_failure}; lastIdentity={last!r}"
    )


def run_project_page_check(chrome, url, width, height, network_retry_origins, check):
    """Retry a page only when its failed check has project transport evidence."""
    last_failure = None
    last_transients = []
    for attempt in range(1, PROJECT_PAGE_ATTEMPTS + 1):
        chrome.navigate(url, width, height)
        try:
            result = check()
        except PermanentBrowserFailure:
            raise
        except BrowserFailure as exc:
            last_failure = exc
            last_transients = chrome.project_network_transients(network_retry_origins)
            if not last_transients or attempt == PROJECT_PAGE_ATTEMPTS:
                if last_transients:
                    raise BrowserFailure(
                        f"project page transport retry exhausted: url={url!r} "
                        f"attempt={attempt}/{PROJECT_PAGE_ATTEMPTS} transients={last_transients!r} "
                        f"lastFailure={str(exc)!r}"
                    ) from exc
                raise
            backoff = min(attempt, 2)
            summary = last_transients[0]
            if len(last_transients) > 1:
                summary += f" (+{len(last_transients) - 1} more)"
            print(
                f"[production-browser] retry stage=page-transport url={url} "
                f"attempt={attempt}/{PROJECT_PAGE_ATTEMPTS} reason={summary!r} "
                f"next=retry backoff={backoff}s",
                file=sys.stderr,
                flush=True,
            )
            time.sleep(backoff)
        else:
            if attempt > 1:
                print(
                    f"[production-browser] recovered stage=page-transport url={url} "
                    f"attempt={attempt}/{PROJECT_PAGE_ATTEMPTS} PASS",
                    file=sys.stderr,
                    flush=True,
                )
            return result
    raise BrowserFailure(
        f"project page transport retry exhausted: url={url!r} "
        f"transients={last_transients!r} lastFailure={str(last_failure)!r}"
    )


def page_identity(chrome, base, locale, requested_path, width, height, canonical_origin=None, expected_final_path=None,
                  expected_rendered_route=None, expected_description=None, expected_title=None,
                  network_retry_origins=(), post_identity_check=None):
    url = urllib.parse.urljoin(base, requested_path.lstrip("/"))
    check = lambda: wait_for_rendered_identity(
        chrome, base, locale, requested_path, canonical_origin, expected_final_path,
        expected_rendered_route, expected_description, expected_title, width,
        post_identity_check=post_identity_check,
    )
    if network_retry_origins:
        return run_project_page_check(chrome, url, width, height, network_retry_origins, check)
    chrome.navigate(url, width, height)
    return check()


def wait_for_condition(check, label, timeout=SEMANTIC_CONVERGENCE_TIMEOUT):
    deadline = time.monotonic() + timeout
    last_failure = "condition was not evaluated"
    while True:
        try:
            return check()
        except PermanentBrowserFailure:
            raise
        except BrowserFailure as exc:
            last_failure = str(exc)
        if time.monotonic() >= deadline:
            break
        time.sleep(0.1)
    raise BrowserFailure(f"{label} timed out after {timeout}s: lastFailure={last_failure}")


def wait_for_spa_transition(chrome, before, base, locale, canonical_origin, timeout=SEMANTIC_CONVERGENCE_TIMEOUT):
    """Wait for path, SEO metadata, rendered marker, and route shell together."""
    def check():
        snapshot = chrome.evaluate("""(() => ({
          lang: document.documentElement.lang, origin: location.origin, path: location.pathname,
          canonical: document.querySelector('link[rel="canonical"]')?.href || '',
          renderedRoute: document.documentElement.getAttribute('data-tour-rendered-route') || '',
          header: document.querySelectorAll('.top-bar').length,
          footer: document.querySelectorAll('.site-footer').length,
          next: !!document.querySelector('.next-page'),
          body: (document.body?.innerText || '').trim(),
          mounts: document.querySelectorAll('[data-go-dev-course-ad]').length
        }))()""", check=f"SPA semantic convergence from {before}")
        validate_identity_invariants(snapshot, base, locale, before, chrome)
        assert_true(snapshot["path"] != before, f"SPA route did not change from {before}: {snapshot}")
        assert_true(snapshot["canonical"] == canonical_origin + snapshot["path"],
                    f"SPA canonical mismatch: {snapshot}")
        assert_true(snapshot["renderedRoute"] == snapshot["path"],
                    f"SPA rendered route mismatch: {snapshot}")
        assert_true(snapshot["header"] == 1 and snapshot["footer"] == 1 and snapshot["next"] and
                    len(snapshot["body"]) > 20, f"SPA DOM/shell failed: {snapshot}")
        return snapshot

    return wait_for_condition(check, f"SPA semantic convergence from {before}", timeout)


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
    return snapshot


DESKTOP_COURSE_LAYOUT_EXPRESSION = """(() => {
  const selectors = {
    container: '#editor-container', lesson: '#left-side', content: '.slide-content',
    editor: '#right-side', code: '#right-side .CodeMirror', divider: '[vertical-slide]'
  };
  const elements = Object.fromEntries(Object.entries(selectors).map(([name, selector]) =>
    [name, document.querySelector(selector)]));
  const missing = Object.entries(elements).filter(([, element]) => !element).map(([name]) => name);
  if (missing.length) return {missing};
  const rect = element => {
    const box = element.getBoundingClientRect();
    const visibleLeft = Math.max(0, box.left), visibleTop = Math.max(0, box.top);
    const visibleRight = Math.min(innerWidth, box.right), visibleBottom = Math.min(innerHeight, box.bottom);
    return {
      left: box.left, right: box.right, top: box.top, bottom: box.bottom,
      width: box.width, height: box.height,
      visibleWidth: Math.max(0, visibleRight - visibleLeft),
      visibleHeight: Math.max(0, visibleBottom - visibleTop)
    };
  };
  const lesson = rect(elements.lesson), editor = rect(elements.editor);
  const container = rect(elements.container), divider = rect(elements.divider);
  const overlapWidth = Math.max(0, Math.min(lesson.right, editor.right) - Math.max(lesson.left, editor.left));
  const gapWidth = Math.max(0, Math.max(lesson.left, editor.left) - Math.min(lesson.right, editor.right));
  const hitWithin = (element, box) => {
    const left = Math.max(0, box.left), right = Math.min(innerWidth, box.right);
    const top = Math.max(0, box.top), bottom = Math.min(innerHeight, box.bottom);
    if (right <= left || bottom <= top) return false;
    return element.contains(document.elementFromPoint((left + right) / 2, (top + bottom) / 2));
  };
  const walker = document.createTreeWalker(elements.content, NodeFilter.SHOW_TEXT);
  let textNode = null;
  while ((textNode = walker.nextNode()) && !textNode.nodeValue.trim()) {}
  let lessonText = '', lessonTextRect = null, lessonTextHit = false;
  if (textNode) {
    lessonText = textNode.nodeValue.trim();
    const range = document.createRange();
    range.selectNodeContents(textNode);
    lessonTextRect = rect(range);
    lessonTextHit = hitWithin(elements.lesson, lessonTextRect);
  }
  return {
    missing, direction: document.documentElement.dir, viewportWidth: innerWidth,
    container, lesson, editor, divider, overlapWidth, gapWidth,
    lessonText, lessonTextRect, lessonTextHit,
    editorHit: hitWithin(elements.editor, editor),
    splitterReady: !!$(elements.divider).data('ui-draggable'),
    codeDirection: getComputedStyle(elements.code).direction
  };
})()"""


def validate_desktop_course_layout(chrome):
    """Require both desktop course panes and real lesson text to be visibly painted."""
    snapshot = chrome.evaluate(DESKTOP_COURSE_LAYOUT_EXPRESSION, check="inspect desktop course pane geometry")
    assert_true(not snapshot.get("missing"), f"desktop course layout elements missing: {snapshot}")
    direction = snapshot.get("direction")
    assert_true(direction in ("ltr", "rtl"), f"desktop course writing direction invalid: {snapshot}")
    container = snapshot["container"]
    lesson, editor = snapshot["lesson"], snapshot["editor"]
    minimum_width = container["width"] * 0.25
    for name, pane in (("lesson", lesson), ("editor", editor)):
        assert_true(pane["visibleWidth"] >= minimum_width and pane["visibleHeight"] >= 100,
                    f"desktop {name} pane has insufficient visible area: {snapshot}")
        assert_true(pane["left"] >= container["left"] - 2 and pane["right"] <= container["right"] + 2,
                    f"desktop {name} pane is outside the course viewport: {snapshot}")
    assert_true(snapshot["overlapWidth"] <= 2, f"desktop course panes overlap: {snapshot}")
    assert_true(snapshot["gapWidth"] <= 8, f"desktop course panes leave an unexpected gap: {snapshot}")
    if direction == "rtl":
        assert_true(editor["right"] <= lesson["left"] + 2,
                    f"RTL lesson/editor logical order is wrong: {snapshot}")
        boundary = (editor["right"] + lesson["left"]) / 2
    else:
        assert_true(lesson["right"] <= editor["left"] + 2,
                    f"LTR lesson/editor logical order is wrong: {snapshot}")
        boundary = (lesson["right"] + editor["left"]) / 2
    divider_center = (snapshot["divider"]["left"] + snapshot["divider"]["right"]) / 2
    assert_true(abs(divider_center - boundary) <= 8, f"desktop course divider is detached from pane boundary: {snapshot}")
    text_rect = snapshot.get("lessonTextRect") or {}
    assert_true(bool(snapshot.get("lessonText", "").strip()) and
                text_rect.get("visibleWidth", 0) > 0 and text_rect.get("visibleHeight", 0) > 0 and
                snapshot.get("lessonTextHit"), f"desktop lesson text is not visibly painted: {snapshot}")
    assert_true(snapshot.get("editorHit"), f"desktop editor is covered or outside the viewport: {snapshot}")
    assert_true(snapshot.get("splitterReady"), f"desktop course splitter is not draggable: {snapshot}")
    assert_true(snapshot.get("codeDirection") == "ltr", f"desktop code editor is not LTR: {snapshot}")
    return snapshot


MOBILE_COURSE_EDITOR_LAYOUT_EXPRESSION = """(() => {
  const selectors = {
    pane: '#right-side', explorer: '#explorer', parent: '#top-part > .relative-content',
    syntax: '#explorer .syntax-checkbox', imports: '#explorer .imports-checkbox',
    file: '#file-editor', code: '#file-editor .CodeMirror'
  };
  const elements = Object.fromEntries(Object.entries(selectors).map(([name, selector]) =>
    [name, document.querySelector(selector)]));
  const missing = Object.entries(elements).filter(([, element]) => !element).map(([name]) => name);
  if (missing.length) return {missing};
  const rect = element => {
    const box = element.getBoundingClientRect();
    const visibleLeft = Math.max(0, box.left), visibleRight = Math.min(innerWidth, box.right);
    return {
      left: box.left, right: box.right, top: box.top, bottom: box.bottom,
      width: box.width, height: box.height,
      visibleWidth: Math.max(0, visibleRight - visibleLeft)
    };
  };
  const controls = [...elements.explorer.children]
    .filter(element => element.classList.contains('menu-button'))
    .map(element => ({...rect(element), float: getComputedStyle(element).cssFloat}));
  return {
    missing, direction: document.documentElement.dir,
    viewportWidth: innerWidth,
    documentOverflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
    pane: rect(elements.pane), explorer: rect(elements.explorer), parent: rect(elements.parent),
    parentClientWidth: elements.parent.clientWidth,
    controls,
    syntaxFloat: getComputedStyle(elements.syntax).cssFloat,
    importsFloat: getComputedStyle(elements.imports).cssFloat,
    file: rect(elements.file), code: rect(elements.code),
    fileDirection: getComputedStyle(elements.file).direction,
    codeDirection: getComputedStyle(elements.code).direction
  };
})()"""


def validate_mobile_course_editor_layout(chrome):
    """Require the mobile CodeMirror surface to fill its available course width."""
    snapshot = chrome.evaluate(MOBILE_COURSE_EDITOR_LAYOUT_EXPRESSION,
                               check="inspect mobile course editor geometry")
    assert_true(not snapshot.get("missing"), f"mobile course editor elements missing: {snapshot}")
    assert_true(snapshot.get("direction") in ("ltr", "rtl"),
                f"mobile course writing direction invalid: {snapshot}")
    viewport_width = snapshot["viewportWidth"]
    assert_true(viewport_width <= 600, f"mobile course editor check used a desktop viewport: {snapshot}")
    pane, parent = snapshot["pane"], snapshot["parent"]
    assert_true(pane["visibleWidth"] >= viewport_width * 0.95,
                f"mobile editor pane does not fill the viewport: {snapshot}")
    assert_true(parent["visibleWidth"] >= pane["visibleWidth"] * 0.95,
                f"mobile editor parent has insufficient visible width: {snapshot}")
    assert_true(snapshot["syntaxFloat"] == "none" and snapshot["importsFloat"] == "none",
                f"mobile explorer toggles must not float: {snapshot}")
    explorer, controls = snapshot["explorer"], snapshot["controls"]
    assert_true(len(controls) >= 3, f"mobile explorer controls are incomplete: {snapshot}")
    for control in controls:
        assert_true(control["top"] >= explorer["top"] - 2 and control["bottom"] <= explorer["bottom"] + 2,
                    f"mobile explorer height does not contain its controls: {snapshot}")
    for previous, current in zip(controls, controls[1:]):
        assert_true(current["top"] >= previous["bottom"] - 2,
                    f"mobile explorer controls are not one per line: {snapshot}")
    assert_true(max(control["bottom"] for control in controls) <= parent["top"] + 2,
                f"mobile explorer controls intrude into the editor: {snapshot}")
    parent_width = snapshot["parentClientWidth"]
    tolerance = max(2, parent_width * 0.01)
    file_box, code_box = snapshot["file"], snapshot["code"]
    assert_true(file_box["width"] >= parent_width * 0.95 and
                abs(file_box["width"] - parent_width) <= tolerance,
                f"mobile file editor shrank below its available parent width: {snapshot}")
    assert_true(abs(code_box["width"] - file_box["width"]) <= tolerance,
                f"mobile CodeMirror width does not follow the file editor: {snapshot}")
    for name, box in (("file editor", file_box), ("CodeMirror", code_box)):
        assert_true(box["left"] >= -tolerance and box["right"] <= viewport_width + tolerance and
                    box["visibleWidth"] >= box["width"] - tolerance,
                    f"mobile {name} is outside the viewport: {snapshot}")
    assert_true(snapshot["documentOverflow"] <= 2,
                f"mobile course editor causes document horizontal overflow: {snapshot}")
    assert_true(snapshot["fileDirection"] == "ltr" and snapshot["codeDirection"] == "ltr",
                f"mobile code editor is not LTR: {snapshot}")
    return snapshot


def acceptance(base, locale, profile, shared, proxy_server=None):
    list_metadata = locale_list_metadata(locale)
    policy = publication_policy(locale)
    network_retry_origins = (base, shared["shared_assets_public_origin"])
    chrome = Chrome(proxy_server=proxy_server)
    try:
        rendered_routes = (("/", "/", None), ("/tour/", "/tour/welcome/1", "/tour/welcome/1"),
                           ("/tour/list", "/tour/list", None),
                           ("/tour/welcome/1", "/tour/welcome/1", "/tour/welcome/1"),
                           ("/tour/basics/11", "/tour/basics/11", "/tour/basics/11"))
        for path, final_path, rendered_route in rendered_routes:
            is_list = final_path == "/tour/list"
            list_check = (lambda: validate_rendered_list(chrome, list_metadata, formal_course_routes())) if is_list else None
            page_identity(chrome, base, locale, path, 1280, 800, base.rstrip("/"), final_path, rendered_route,
                          expected_description=list_metadata["description"] if is_list else None,
                          expected_title=list_metadata["title"] if is_list else None,
                          network_retry_origins=network_retry_origins, post_identity_check=list_check)
        def language_ready():
            def check():
                language = chrome.evaluate("""(() => ({
                  lang: document.documentElement.lang,
                  origin: location.origin,
                  count: document.querySelectorAll('.site-language-list li').length,
                  current: document.querySelectorAll('.site-language-list [aria-current="page"]').length,
                  links: document.querySelectorAll('.site-language-list a[href]').length
                }))()""", check="production language selector convergence")
                validate_identity_invariants(language, base, locale, "/", chrome)
                assert_true(language["count"] >= 2 and language["current"] == 1 and language["links"] >= 1,
                            f"language selector identity failed: {language}")
                return language

            return wait_for_condition(check, "production language selector convergence")

        run_project_page_check(chrome, base, 375, 812, network_retry_origins, language_ready)

        editor_url = urllib.parse.urljoin(base, "tour/basics/11")
        shared_assets = json.dumps(shared["shared_assets_public_origin"].rstrip("/") + "/")

        def editor_ready():
            def check():
                snapshot = chrome.evaluate(f"""(() => ({{
                  run: !!document.querySelector('#run'), format: !!document.querySelector('#format'),
                  reset: !!document.querySelector('#reset'), cm: !!document.querySelector('.CodeMirror')?.CodeMirror,
                  mount: document.querySelectorAll('[data-go-dev-course-ad]').length,
                  ad: document.querySelectorAll('[data-go-dev-course-ad] ins.adsbygoogle').length,
                  loader: [...document.scripts].some(s => /adsbygoogle/.test(s.src)),
                  helper: [...document.scripts].some(s => /course-ad\\.js(?:$|[?#])/.test(s.src)),
                  empty_mount: [...document.querySelectorAll('[data-go-dev-course-ad]')].every(e =>
                    e.children.length === 0 && !e.hasAttribute('role') && !e.hasAttribute('aria-label') &&
                    !e.hasAttribute('data-go-dev-course-ad-group')),
                  shared: performance.getEntriesByType('resource').some(e => e.name.startsWith({shared_assets}))
                }}))()""", check="production editor dependency convergence")
                assert_true(all(snapshot[key] for key in ("run", "format", "reset", "cm")),
                            f"editor browser identity failed: {snapshot}")
                validate_desktop_course_layout(chrome)
                assert_true(browser_ad_gate(snapshot, chrome.network_requests(), policy["tour_ads_enabled"]),
                            f"editor/ad browser identity failed for publication={policy['publication']}: editor={snapshot}")
                if profile["shared_assets_policy"] == "shared-cloudflare":
                    assert_true(snapshot["shared"], "shared assets were not requested")
                return snapshot

            return wait_for_condition(check, "production editor dependency convergence")

        run_project_page_check(
            chrome, editor_url, 1280, 800, network_retry_origins, editor_ready,
        )
        # Filled and unfilled ads are both accepted: standard requires a request
        # opportunity, while go-local requires complete absence of Tour ads.
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
        malformed = chrome.evaluate("window.__productionAcceptanceMalformed")

        def formatted_source():
            formatted = chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.getValue()",
                                        check="production Format result convergence")
            assert_true(formatted != malformed and "func main() {" in formatted,
                        f"Format did not update source: {formatted!r}")
            return formatted

        wait_for_condition(formatted_source, "production Format result convergence")
        chrome.evaluate("document.querySelector('#reset').click(); true")
        wait_for_editor_reset(chrome, original)
        chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.setValue('package main\\nfunc main(){println(\\\"BROWSER_ACCEPTANCE_OK\\\")}\\n'); true")
        chrome.evaluate("document.querySelector('#run').click(); true")

        def runtime_output():
            runtime = chrome.evaluate("""(() => ({
              output: [...document.querySelectorAll('.output')].map(x => x.innerText).join('\\n'),
              mount: document.querySelectorAll('[data-go-dev-course-ad]').length,
              path: location.pathname
            }))()""", check="production Run result convergence")
            assert_true("BROWSER_ACCEPTANCE_OK" in runtime["output"],
                        f"Run produced no browser-visible expected result: {runtime}")
            return runtime

        runtime = wait_for_condition(runtime_output, "production Run result convergence")
        requests = chrome.network_requests()
        playground = shared["playground_public_origin"].rstrip("/")
        playground_posts = playground_requests(requests, playground, "/compile", "/fmt")
        expected_origin = base.rstrip("/")
        origins = [{key.lower(): value for key, value in request.get("headers", {}).items()}.get("origin") for request in playground_posts]
        assert_true(len(origins) >= 2 and all(origin == expected_origin for origin in origins), f"Playground POST Origin mismatch: {origins}")
        socket_status = chrome.evaluate("fetch('/socket').then(r => r.status)", await_promise=True)
        assert_true(socket_status == 404, "/socket browser boundary failed")

        before = chrome.evaluate("location.pathname")
        chrome.evaluate("document.querySelector('.next-page').click(); true")
        spa = wait_for_spa_transition(chrome, before, base, locale, base.rstrip("/"))
        final_mounts = spa["mounts"]
        if policy["tour_ads_enabled"]:
            assert_true(final_mounts == 1, f"SPA course-ad mount count mismatch: expected=1 actual={final_mounts}")
        else:
            assert_true(final_mounts in (0, 1), f"SPA empty course-ad host count mismatch: actual={final_mounts}")

        page_identity(chrome, base, locale, "/tour/moretypes/1", 375, 812, base.rstrip("/"),
                      "/tour/moretypes/1", "/tour/moretypes/1",
                      network_retry_origins=network_retry_origins,
                      post_identity_check=lambda: validate_mobile_course_editor_layout(chrome))
        before = chrome.evaluate("location.pathname")
        chrome.evaluate("document.querySelector('.next-page').click(); true")
        wait_for_spa_transition(chrome, before, base, locale, base.rstrip("/"))
    finally:
        chrome.close()


def preview_acceptance(base, locale, profile, shared, registry, descriptions, list_metadata):
    """Run browser checks whose preview identity intentionally differs from production."""
    canonical_origin = profile["production_public_url"].rstrip("/")
    policy = publication_policy(locale)
    chrome = Chrome()
    try:
        rendered_routes = (("/", "/"), ("/tour/", "/tour/welcome/1"), ("/tour/list", "/tour/list"),
                           ("/tour/welcome/1", "/tour/welcome/1"), ("/tour/basics/11", "/tour/basics/11"))
        for path, final_path in rendered_routes:
            course_route = final_path if re.match(r"^/tour/[^/]+/[1-9][0-9]*$", final_path) else None
            is_list = final_path == "/tour/list"
            list_check = (lambda: validate_rendered_list(chrome, list_metadata, formal_course_routes())) if is_list else None
            page_identity(chrome, base, locale, path, 1280, 800, canonical_origin, final_path, course_route,
                          descriptions.get(course_route) if course_route else (list_metadata["description"] if is_list else None),
                          list_metadata["title"] if is_list else None, post_identity_check=list_check)
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
            def preview_editor_ready():
                editor = chrome.evaluate("""(() => ({run:!!document.querySelector('#run'),format:!!document.querySelector('#format'),
                  reset:!!document.querySelector('#reset'),cm:!!document.querySelector('.CodeMirror')?.CodeMirror,
                  mount:document.querySelectorAll('[data-go-dev-course-ad]').length,
                  ad:document.querySelectorAll('[data-go-dev-course-ad] ins.adsbygoogle').length,
                  loader:[...document.scripts].some(s=>/adsbygoogle/.test(s.src)),
                  helper:[...document.scripts].some(s=>/course-ad\\.js(?:$|[?#])/.test(s.src)),
                  empty_mount:[...document.querySelectorAll('[data-go-dev-course-ad]')].every(e=>
                    e.children.length===0&&!e.hasAttribute('role')&&!e.hasAttribute('aria-label')&&
                    !e.hasAttribute('data-go-dev-course-ad-group'))}))()""",
                    check="preview editor dependency convergence")
                assert_true(all(editor[key] for key in ("run", "format", "reset", "cm")),
                            f"editor controls missing: {editor}")
                if not mobile:
                    validate_desktop_course_layout(chrome)
                else:
                    validate_mobile_course_editor_layout(chrome)
                if not policy["tour_ads_enabled"]:
                    assert_true(browser_ad_gate(editor, chrome.network_requests(), False),
                                f"go-local preview retains Tour ad surface: {editor}")
                return editor

            wait_for_condition(preview_editor_ready, "preview editor dependency convergence")
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
            chrome.evaluate("document.querySelector('.next-page').click();true")
            wait_for_spa_transition(chrome, before, base, locale, canonical_origin)

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
