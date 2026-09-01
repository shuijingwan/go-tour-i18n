#!/usr/bin/env python3
"""Automated Chrome acceptance for a production locale.

Uses Chrome DevTools Protocol directly so the repository keeps its existing
Chrome-only browser baseline and adds no browser framework dependency.
"""

from __future__ import annotations

import base64
import hashlib
import importlib.util
import json
import os
import pathlib
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
        self.temp = pathlib.Path(tempfile.mkdtemp(prefix="go-tour-production-browser-"))
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

    def evaluate(self, expression, await_promise=False):
        result = self.call("Runtime.evaluate", {
            "expression": expression, "returnByValue": True,
            "awaitPromise": await_promise, "userGesture": True,
        }, timeout=60)
        if result.get("exceptionDetails"):
            raise BrowserFailure(f"browser JavaScript failed: {result['exceptionDetails'].get('text')}")
        return result.get("result", {}).get("value")

    def navigate(self, url, width, height):
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


def browser_ad_gate(snapshot):
    """Filled and unfilled are both valid; mount, loader and slot remain mandatory."""
    return snapshot.get("mount") == 1 and snapshot.get("ad") == 1 and bool(snapshot.get("loader"))


def page_identity(chrome, base, locale, path, width, height):
    url = urllib.parse.urljoin(base, path.lstrip("/"))
    chrome.navigate(url, width, height)
    identity = chrome.evaluate("""(() => ({
      lang: document.documentElement.lang,
      href: location.href,
      origin: location.origin,
      canonical: document.querySelector('link[rel="canonical"]')?.href || '',
      title: document.title,
      description: document.querySelector('meta[name="description"]')?.content || '',
      overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth
    }))()""")
    assert_true(identity["lang"] == locale, f"{path}: html lang mismatch")
    assert_true(identity["origin"] == base.rstrip("/"), f"{path}: production hostname mismatch")
    assert_true(identity["canonical"] == identity["href"], f"{path}: canonical mismatch")
    assert_true(identity["title"] and identity["description"], f"{path}: SEO metadata missing")
    if width <= 480:
        assert_true(identity["overflow"] <= 2, f"{path}: unexpected page-level horizontal overflow")


def acceptance(base, locale, profile, shared):
    chrome = Chrome()
    try:
        for path in ("/", "/tour/", "/tour/list", "/tour/welcome/1", "/tour/basics/11"):
            page_identity(chrome, base, locale, path, 1280, 800)
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
          const original = cm.getValue();
          const malformed = 'package main\\nfunc main(){println("browser acceptance")}\\n';
          window.__productionAcceptanceOriginal = original;
          window.__productionAcceptanceMalformed = malformed;
          cm.setValue(malformed);
          return true;
        })()""")
        assert_true(edit, "could not prepare editor interaction")
        chrome.evaluate("document.querySelector('#format').click(); true")
        time.sleep(3)
        formatted = chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.getValue()")
        assert_true(formatted != chrome.evaluate("window.__productionAcceptanceMalformed") and "func main() {" in formatted, "Format did not update source")
        chrome.evaluate("document.querySelector('#reset').click(); true")
        time.sleep(1)
        reset_value = chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.getValue()")
        assert_true(reset_value == chrome.evaluate("window.__productionAcceptanceOriginal"), "Reset did not restore source")
        chrome.evaluate("document.querySelector('.CodeMirror').CodeMirror.setValue('package main\\nfunc main(){println(\\\"BROWSER_ACCEPTANCE_OK\\\")}\\n'); true")
        chrome.evaluate("document.querySelector('#run').click(); true")
        time.sleep(8)
        runtime = chrome.evaluate("""(() => ({
          output: [...document.querySelectorAll('.output')].map(x => x.innerText).join('\\n'),
          mount: document.querySelectorAll('[data-go-dev-course-ad]').length,
          path: location.pathname
        }))()""")
        requests = chrome.network_requests()
        urls = [request.get("url", "") for request in requests]
        assert_true("BROWSER_ACCEPTANCE_OK" in runtime["output"], "Run produced no browser-visible expected result")
        playground = shared["playground_public_origin"].rstrip("/") + "/"
        assert_true(any(url.startswith(playground + "compile") for url in urls), "Run did not use the formal Playground endpoint")
        assert_true(any(url.startswith(playground + "fmt") for url in urls), "Format did not use the formal Playground endpoint")
        playground_requests = [request for request in requests if request.get("method") == "POST" and request.get("url", "").startswith(playground)]
        expected_origin = base.rstrip("/")
        origins = [{key.lower(): value for key, value in request.get("headers", {}).items()}.get("origin") for request in playground_requests]
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
