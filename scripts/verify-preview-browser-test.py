#!/usr/bin/env python3

import contextlib
import importlib.util
import io
import json
import os
import pathlib
import shutil
import subprocess
import sys
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import xml.etree.ElementTree as ET
from unittest import mock

ROOT = pathlib.Path(__file__).resolve().parent.parent

def load(name, path):
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec); spec.loader.exec_module(module); return module

PREVIEW = load("verify_preview_browser_tested", ROOT / "scripts" / "verify-preview-browser.py")
PRODUCTION = load("verify_production_browser_tested", ROOT / "scripts" / "verify-production-browser.py")
CORE = PREVIEW.CORE

class PreviewBrowserTest(unittest.TestCase):
    def site_shell(self, route, content):
        return ('<html lang="zh-CN"><head><title>Shell</title>'
                '<link rel="canonical" href="https://locale.example%s"></head><body>%s</body></html>' % (route, content)).encode()

    def test_activated_site_shell_rejects_legacy_homepage_and_tour_fallback(self):
        source = '<main><a href="/translation/">Project</a><a href="/learn/">Learn</a></main>'
        coverage = {"/", "/translation/"}
        for content in ('<main class="site-home">Legacy project landing</main>',
                        '<div ng-view></div>', '<main><a href="/tour/welcome/1">Tour</a></main>'):
            with self.subTest(content=content), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.validate_site_shell(self.site_shell("/", content), source, "http://127.0.0.1:3999/", "/", "zh-CN", "https://locale.example", coverage)
        valid = '<main><a href="/translation/">项目</a><a href="https://go.dev/learn/">学习</a></main>'
        PREVIEW.validate_site_shell(self.site_shell("/", valid), source, "http://127.0.0.1:3999/", "/", "zh-CN", "https://locale.example", coverage)
        for wrong in ('http://127.0.0.1:3999/learn/', 'https://locale.example/translation/'):
            with self.subTest(wrong=wrong), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.validate_site_shell(self.site_shell("/", valid.replace('/translation/' if 'translation' in wrong else 'https://go.dev/learn/', wrong)), source,
                                            "http://127.0.0.1:3999/", "/", "zh-CN", "https://locale.example", coverage)

    def test_translation_shell_requires_its_own_canonical(self):
        source = '<main><a href="/">Home</a></main>'
        for route in ("/translation/", "/tour/", "/tour/welcome/1"):
            body = self.site_shell(route, source)
            if route == "/translation/":
                PREVIEW.validate_site_shell(body, source, "http://127.0.0.1:3999/", "/translation/", "zh-CN", "https://locale.example", {"/", "/translation/"})
            else:
                with self.assertRaises(CORE.BrowserFailure):
                    PREVIEW.validate_site_shell(body, source, "http://127.0.0.1:3999/", "/translation/", "zh-CN", "https://locale.example", {"/", "/translation/"})

    def test_activated_shell_coverage_uses_sparse_content_scope(self):
        with tempfile.TemporaryDirectory() as directory, mock.patch.object(PREVIEW, "ROOT", pathlib.Path(directory)):
            path = pathlib.Path(directory) / "locales" / "zh-CN" / "content-scope.json"
            path.parent.mkdir(parents=True)
            self.assertIsNone(PREVIEW.activated_shell_coverage("zh-CN"))
            scope = {"schema": "go-learning/locale-content-scope/v1", "locale": "zh-CN", "packages": []}
            path.write_text(json.dumps(scope), encoding="utf-8")
            self.assertIsNone(PREVIEW.activated_shell_coverage("zh-CN"))
            scope["packages"] = [{"package": "site-v2-shell", "state": "complete", "local_canonical_routes": ["/", "/translation/"]}]
            path.write_text(json.dumps(scope), encoding="utf-8")
            self.assertEqual(PREVIEW.activated_shell_coverage("zh-CN"), {"/", "/translation/"})
            scope["packages"][0]["local_canonical_routes"] = ["/"]
            path.write_text(json.dumps(scope), encoding="utf-8")
            with self.assertRaises(CORE.BrowserFailure): PREVIEW.activated_shell_coverage("zh-CN")

    def test_activated_shell_checks_redirects_and_missing_local_routes(self):
        profile = {"locale": "zh-CN", "production_public_url": "https://locale.example/"}
        coverage = {"/", "/translation/"}
        def response(base, route, **kwargs):
            self.assertFalse(kwargs["follow_redirects"])
            return (200, self.site_shell(route, '<main></main>')) if route in coverage else (404, b'Not found')
        with mock.patch.object(PREVIEW, "activated_shell_coverage", return_value=coverage), \
                mock.patch.object(PREVIEW, "validate_site_shell"), mock.patch.object(PREVIEW, "request", side_effect=response) as request:
            PREVIEW.activated_shell_acceptance("http://127.0.0.1:3999/", profile)
            self.assertEqual(request.call_count, 5)
            for rejected_route, status in (("/translation/", 302), ("/translation/", 404), ("/__preview_missing_site_route__", 200),
                                           ("/translation/__preview_missing_site_route__", 302), ("/tour/__preview_missing_site_route__", 200)):
                def wrong(base, route, **kwargs):
                    return (status, b'fallback') if route == rejected_route else response(base, route, **kwargs)
                request.side_effect = wrong
                with self.subTest(route=rejected_route, status=status), self.assertRaises(CORE.BrowserFailure):
                    PREVIEW.activated_shell_acceptance("http://127.0.0.1:3999/", profile)

    def test_site_shell_guard_runs_before_legacy_machine_acceptance(self):
        with mock.patch.object(PREVIEW, "activated_shell_acceptance", side_effect=CORE.BrowserFailure("wrong renderer")) as guard, \
                mock.patch.object(PREVIEW, "catalog_routes") as catalog:
            with self.assertRaises(CORE.BrowserFailure): PREVIEW.machine_acceptance("http://127.0.0.1:3999/", {"locale": "zh-CN"})
            guard.assert_called_once(); catalog.assert_not_called()

    def test_shell_request_preserves_redirect_status_for_rejection(self):
        error = PREVIEW.urllib.error.HTTPError("http://127.0.0.1:3999/translation/", 302, "Found", {}, io.BytesIO(b"redirect"))
        opener = mock.Mock(); opener.open.side_effect = error
        with mock.patch.object(PREVIEW.urllib.request, "build_opener", return_value=opener) as build, \
                mock.patch.object(PREVIEW.urllib.request, "urlopen") as follow:
            self.assertEqual(PREVIEW.request("http://127.0.0.1:3999/", "/translation/", follow_redirects=False), (302, b"redirect"))
            self.assertIsInstance(build.call_args.args[0], PREVIEW.NoRedirect)
            follow.assert_not_called()
        self.assertIsNone(PREVIEW.NoRedirect().redirect_request(None, None, 302, "Found", {}, "/tour/welcome/1"))

    def test_chrome_command_keeps_default_without_proxy(self):
        command = CORE.chrome_command("google-chrome", "/tmp/chrome-profile")

        self.assertEqual(command, [
            "google-chrome", "--headless=new", "--no-sandbox", "--disable-gpu",
            "--disable-dev-shm-usage", "--disable-breakpad", "--disable-crash-reporter",
            "--noerrdialogs", "--no-first-run", "--remote-debugging-address=127.0.0.1",
            "--remote-debugging-port=0", "--user-data-dir=/tmp/chrome-profile", "about:blank",
        ])

    def test_chrome_command_adds_explicit_proxy(self):
        proxy_server = "socks5://127.0.0.1:49152"
        command = CORE.chrome_command("google-chrome", "/tmp/chrome-profile", proxy_server)

        self.assertIn(f"--proxy-server={proxy_server}", command)
        self.assertEqual(command[-1], "about:blank")

    def chrome_for_navigation(self, readiness):
        chrome = CORE.Chrome.__new__(CORE.Chrome)
        chrome.current_route = "about:blank"
        chrome.events = []
        chrome.render_readiness = mock.Mock(side_effect=readiness)
        chrome.call = mock.Mock(return_value={})
        return chrome

    def test_navigation_retries_render_readiness_then_passes(self):
        chrome = self.chrome_for_navigation([
            (False, {"readyState": "loading", "bodyTextLength": 0, "location": "https://it.example/"}),
            (True, {"readyState": "interactive", "bodyTextLength": 42, "location": "https://it.example/"}),
        ])
        with mock.patch.object(CORE.time, "sleep"):
            chrome.navigate("https://it.example/", 1280, 800)
        self.assertEqual([call.args[0] for call in chrome.call.call_args_list].count("Page.navigate"), 2)
        self.assertEqual(chrome.render_readiness.call_count, 2)

    def test_navigation_retries_devtools_navigation_timeout_then_passes(self):
        chrome = self.chrome_for_navigation([
            (True, {"readyState": "interactive", "bodyTextLength": 42, "location": "https://it.example/"}),
        ])
        page_calls = 0

        def call(method, *args, **kwargs):
            nonlocal page_calls
            if method == "Page.navigate":
                page_calls += 1
                if page_calls == 1:
                    raise CORE.BrowserFailure("DevTools Page.navigate timed out")
            return {}

        chrome.call = mock.Mock(side_effect=call)
        with mock.patch.object(CORE.time, "sleep"):
            chrome.navigate("https://it.example/", 1280, 800)
        self.assertEqual(page_calls, 2)
        self.assertEqual(chrome.render_readiness.call_count, 1)

    def test_cdp_socket_timeout_becomes_bounded_call_timeout_state(self):
        chrome = CORE.Chrome.__new__(CORE.Chrome)
        chrome.next_id = 1
        chrome.session_id = None
        chrome.events = []
        chrome.ws = mock.Mock()
        chrome.ws.receive.side_effect = [CORE.socket.timeout(), {"id": 1, "result": {"ok": True}}]
        with mock.patch.object(CORE.time, "monotonic", return_value=0):
            self.assertEqual(chrome.call("Page.navigate", {"url": "https://it.example/"}, timeout=30), {"ok": True})
        self.assertEqual(chrome.ws.receive.call_count, 2)

    def test_navigation_fails_closed_after_bounded_render_attempts(self):
        chrome = self.chrome_for_navigation([
            (False, {"readyState": "loading", "bodyTextLength": 0, "location": "https://it.example/"}),
        ] * CORE.RENDER_ATTEMPTS)
        with mock.patch.object(CORE.time, "sleep"), self.assertRaises(CORE.BrowserFailure) as caught:
            chrome.navigate("https://it.example/", 1280, 800)
        evidence = str(caught.exception)
        self.assertEqual([call.args[0] for call in chrome.call.call_args_list].count("Page.navigate"), CORE.RENDER_ATTEMPTS)
        for expected in ("url='https://it.example/'", "route='/'", "attempt=3/3", "readyState='loading'",
                         "bodyTextLength=0", "location='https://it.example/'"):
            self.assertIn(expected, evidence)

    def test_semantic_assertion_failure_does_not_trigger_navigation_retry(self):
        chrome = self.chrome_for_navigation([
            (True, {"readyState": "interactive", "bodyTextLength": 42, "location": "https://it.example/tour/list"}),
        ])
        chrome.evaluate = mock.Mock(return_value={
            "lang": "wrong", "href": "https://it.example/tour/list", "origin": "https://it.example",
            "path": "/tour/list", "renderedRoute": "", "heading": "", "canonical": "https://it.example/tour/list",
            "title": "List", "description": "Description", "overflow": 0,
        })
        with mock.patch.object(CORE.time, "sleep"), self.assertRaises(CORE.BrowserFailure):
            CORE.page_identity(chrome, "https://it.example/", "it-IT", "/tour/list", 1280, 800)
        self.assertEqual([call.args[0] for call in chrome.call.call_args_list].count("Page.navigate"), 1)

    def test_project_network_transients_are_narrow_and_ignore_third_party(self):
        chrome = CORE.Chrome.__new__(CORE.Chrome)
        chrome.events = [
            {"method": "Network.requestWillBeSent", "params": {"requestId": "lesson", "type": "XHR",
                "request": {"url": "https://ja.example/tour/lesson/"}}},
            {"method": "Network.responseReceived", "params": {"requestId": "lesson", "type": "XHR",
                "response": {"url": "https://ja.example/tour/lesson/", "status": 522}}},
            {"method": "Network.requestWillBeSent", "params": {"requestId": "ad", "type": "Script",
                "request": {"url": "https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js"}}},
            {"method": "Network.loadingFailed", "params": {"requestId": "ad", "type": "Script",
                "errorText": "net::ERR_TIMED_OUT"}},
            {"method": "Network.requestWillBeSent", "params": {"requestId": "missing", "type": "XHR",
                "request": {"url": "https://ja.example/tour/static/partials/editor.html"}}},
            {"method": "Network.requestWillBeSent", "params": {"requestId": "script", "type": "Script",
                "request": {"url": "https://ja.example/tour/script.js"}}},
            {"method": "Network.loadingFailed", "params": {"requestId": "script", "type": "Script",
                "errorText": "net::ERR_TIMED_OUT"}},
            {"method": "Network.requestWillBeSent", "params": {"requestId": "server", "type": "XHR",
                "request": {"url": "https://ja.example/tour/server-error"}}},
            {"method": "Network.responseReceived", "params": {"requestId": "server", "type": "XHR",
                "response": {"url": "https://ja.example/tour/server-error", "status": 500}}},
        ]
        evidence = chrome.project_network_transients(("https://ja.example/", "https://assets.example/"))
        self.assertEqual(evidence, [
            "HTTP 522 XHR https://ja.example/tour/lesson/",
            "net::ERR_TIMED_OUT Script https://ja.example/tour/script.js",
            "pending XHR https://ja.example/tour/static/partials/editor.html",
        ])

    def test_project_page_retry_requires_transport_evidence(self):
        chrome = mock.Mock()
        chrome.project_network_transients.side_effect = [["HTTP 525 Script https://ja.example/tour/script.js"]]
        check = mock.Mock(side_effect=[CORE.BrowserFailure("editor not hydrated"), "PASS"])
        output = io.StringIO()
        with mock.patch.object(CORE.time, "sleep"), contextlib.redirect_stderr(output):
            self.assertEqual(CORE.run_project_page_check(
                chrome, "https://ja.example/tour/", 1280, 800, ("https://ja.example/",), check), "PASS")
        self.assertEqual(chrome.navigate.call_count, 2)
        self.assertIn("attempt=1/3", output.getvalue())
        self.assertIn("next=retry backoff=1s", output.getvalue())
        self.assertIn("attempt=2/3 PASS", output.getvalue())

        chrome.reset_mock()
        chrome.project_network_transients = mock.Mock(return_value=[])
        with self.assertRaisesRegex(CORE.BrowserFailure, "canonical mismatch"):
            CORE.run_project_page_check(
                chrome, "https://ja.example/tour/", 1280, 800, ("https://ja.example/",),
                mock.Mock(side_effect=CORE.BrowserFailure("canonical mismatch")))
        self.assertEqual(chrome.navigate.call_count, 1)

    def test_cdp_exception_diagnostics_include_action_route_and_stack(self):
        chrome = CORE.Chrome.__new__(CORE.Chrome)
        chrome.current_route = "/tour/welcome/1"
        chrome.call = lambda *args, **kwargs: {
            "result": {"type": "object", "subtype": "error", "description": "TypeError: broken selector"},
            "exceptionDetails": {"text": "Uncaught", "url": "http://127.0.0.1/tour/script.js",
                "lineNumber": 41, "columnNumber": 7, "stackTrace": {"callFrames": [{
                    "functionName": "inspect", "url": "http://127.0.0.1/tour/script.js",
                    "lineNumber": 41, "columnNumber": 7}]}}
        }
        with self.assertRaises(CORE.BrowserFailure) as caught:
            chrome.evaluate("broken()", check="course identity snapshot")
        evidence = str(caught.exception)
        for expected in ("course identity snapshot", "/tour/welcome/1", "TypeError: broken selector",
                         "tour/script.js", "line=42", "column=8", "inspect@"):
            self.assertIn(expected, evidence)

    def test_preview_editor_source_expression_keeps_javascript_newline_escapes(self):
        source = (ROOT / "scripts" / "browser_acceptance.py").read_text(encoding="utf-8")
        self.assertIn("window.__previewMalformed='package main\\\\nfunc main()", source)
        self.assertIn('check="prepare malformed editor source"', source)

    def test_playground_compile_endpoint_semantics(self):
        origin = "http://127.0.0.1:38573"
        CORE.validate_playground_endpoint(origin + "/_/compile?backend=", origin, "/_/compile", "compile")
        rejected = (origin + "/_/compile", origin + "/_/compile?backend=&foo=bar",
                    origin + "/_/wrong?backend=", "http://127.0.0.1:38574/_/compile?backend=")
        for url in rejected:
            with self.subTest(url=url), self.assertRaises(CORE.BrowserFailure):
                CORE.validate_playground_endpoint(url, origin, "/_/compile", "compile")

    def test_playground_fmt_endpoint_semantics(self):
        origin = "http://127.0.0.1:38573"
        CORE.validate_playground_endpoint(origin + "/_/fmt", origin, "/_/fmt", "fmt")
        for url in ("http://127.0.0.1:38574/_/fmt", origin + "/_/wrong", origin + "/_/fmt?backend="):
            with self.subTest(url=url), self.assertRaises(CORE.BrowserFailure):
                CORE.validate_playground_endpoint(url, origin, "/_/fmt", "fmt")

    def test_production_playground_paths_keep_formal_origin(self):
        origin = "https://play.example.test:8443"
        CORE.validate_playground_endpoint(origin + "/compile?backend=", origin, "/compile", "compile")
        CORE.validate_playground_endpoint(origin + "/fmt", origin, "/fmt", "fmt")

    def test_browser_ad_gate_is_policy_aware(self):
        standard = {"mount": 1, "ad": 1, "loader": True, "helper": True, "empty_mount": False}
        ad_request = [{"url": "https://pagead2.googlesyndication.com/pagead/js/adsbygoogle.js"}]
        self.assertTrue(CORE.browser_ad_gate(standard, ad_request, True))
        self.assertFalse(CORE.browser_ad_gate({"mount": 1, "ad": 1, "loader": False, "helper": True, "empty_mount": False}, ad_request, True))
        empty_host = {"mount": 1, "ad": 0, "loader": False, "helper": False, "empty_mount": True}
        self.assertTrue(CORE.browser_ad_gate(empty_host, [], False))
        self.assertFalse(CORE.browser_ad_gate({**empty_host, "ad": 1, "empty_mount": False}, [], False))
        self.assertFalse(CORE.browser_ad_gate({**empty_host, "loader": True}, [], False))
        self.assertFalse(CORE.browser_ad_gate(empty_host, ad_request, False))
        self.assertFalse(CORE.browser_ad_gate(standard, ad_request, False))

    def test_browser_policy_bridge_uses_go_authority(self):
        result = mock.Mock(returncode=0, stdout='{"locale":"example","publication":"go-local","advertising":"go-local","tour_ads_enabled":false}\n', stderr="")
        with mock.patch.object(CORE.subprocess, "run", return_value=result) as run:
            policy = CORE.publication_policy("example")
        self.assertEqual(policy["publication"], "go-local")
        self.assertEqual(policy["advertising"], "go-local")
        self.assertFalse(policy["tour_ads_enabled"])
        self.assertEqual(run.call_args.args[0], ["go", "run", "-mod=readonly", "./cmd/tour-i18n", "policy", "publication", "--locale", "example"])

    def reset_state(self, original="original", displayed="original", content="original"):
        return {"displayed": displayed, "content": content, "model": content, "view": displayed,
                "original": original, "hash": "hash", "stored": original}

    def test_reset_requires_model_and_codemirror_to_restore_original(self):
        CORE.validate_editor_modified(self.reset_state(displayed="changed", content="changed"), "changed")
        CORE.validate_reset_state(self.reset_state(), "original")
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_reset_state(self.reset_state(displayed="changed"), "original")
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_reset_state(self.reset_state(content="changed"), "original")
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_reset_state(self.reset_state(), "wrong original")

    def test_reset_wait_uses_condition_not_fixed_delay(self):
        states = [self.reset_state(displayed="changed", content="changed"), self.reset_state(), self.reset_state()]
        chrome = mock.Mock()
        chrome.evaluate.side_effect = states
        with mock.patch.object(CORE.time, "sleep") as sleep:
            CORE.wait_for_editor_reset(chrome, "original", timeout=1)
        self.assertEqual(chrome.evaluate.call_count, 3)
        self.assertTrue(sleep.called)
    def test_preview_url_requires_loopback_http(self):
        for accepted in ("http://127.0.0.1:38573/", "http://localhost:38573/", "http://[::1]:38573/"):
            PREVIEW.preview_url(accepted)
        for rejected in ("https://127.0.0.1:38573/", "http://example.com:38573/", "http://127.0.0.1/", "http://127.0.0.1:38573/tour/"):
            with self.assertRaises(CORE.BrowserFailure): PREVIEW.preview_url(rejected)

    def test_unknown_locale_fails_closed(self):
        with self.assertRaises(CORE.BrowserFailure): PREVIEW.profile_for({"locales": []}, "xx-XX")

    def test_preview_entrypoint_uses_formal_identity_without_ads_argument(self):
        captured = []
        original = sys.argv
        try:
            sys.argv = ["verify-preview-browser.py", "http://127.0.0.1:38573/", "ko-KR"]
            with mock.patch.object(PREVIEW, "machine_acceptance"), mock.patch.object(PREVIEW, "registry", return_value=[]), \
                    mock.patch.object(PREVIEW, "browser_plan", return_value={"pages": [], "families": ["/tour/**"]}), \
                    contextlib.redirect_stdout(io.StringIO()), \
                    mock.patch.object(PREVIEW.CORE, "preview_acceptance", side_effect=lambda *args, **kwargs: captured.append(args)):
                self.assertEqual(PREVIEW.main(), 0)
        finally: sys.argv = original
        self.assertEqual(captured[0][2]["production_public_url"], "https://ko-go-dev.shuijingwanwq.com/")
        self.assertEqual(len(captured[0]), 7)

    def test_preview_machine_contract_uses_catalog_and_socket(self):
        source = (ROOT / "scripts" / "verify-preview-browser.py").read_text(encoding="utf-8")
        for evidence in ("tour-pages.tsv", "sitemap URL set", "Upgrade /socket", "production_public_url"):
            self.assertIn(evidence, source)

    def test_sitemap_course_order_is_not_a_contract(self):
        production = "https://ko-go-dev.shuijingwanwq.com"
        routes = PREVIEW.catalog_routes()
        reordered = list(reversed(routes))
        urls = [production + "/", production + "/tour/list", *(production + route for route in reordered)]
        document = ET.Element("urlset", xmlns="http://www.sitemaps.org/schemas/sitemap/0.9")
        for url in urls:
            node = ET.SubElement(document, "url")
            ET.SubElement(node, "loc").text = url
        parsed = PREVIEW.validate_sitemap(ET.tostring(document), production, {"/", "/tour/list", *routes})
        self.assertEqual(parsed[2], production + reordered[0])
        self.assertEqual(set(parsed), set(urls))

    def test_sitemap_exact_activated_coverage_and_package_append(self):
        scope = json.loads((ROOT / "locales/zh-CN/content-scope.json").read_text(encoding="utf-8"))
        routes = {r for p in scope["packages"] for r in p["local_canonical_routes"]}
        self.assertEqual(PREVIEW.activated_canonical_routes("zh-CN"), routes)
        self.assertEqual(len(routes), 156)
        production = "https://locale.example"
        def sitemap(paths):
            document = ET.Element("urlset", xmlns="http://www.sitemaps.org/schemas/sitemap/0.9")
            for route in paths:
                ET.SubElement(ET.SubElement(document, "url"), "loc").text = production + route
            return ET.tostring(document)
        PREVIEW.validate_sitemap(sitemap(routes), production, routes)
        appended = routes | {"/future/"}
        PREVIEW.validate_sitemap(sitemap(appended), production, appended)
        for wrong in (routes - {"/translation/"}, (routes - {"/translation/"}) | {"/fake/"}, list(routes) + ["/"]):
            with self.subTest(wrong_count=len(wrong)), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.validate_sitemap(sitemap(wrong), production, routes)

    def test_raw_tour_shell_is_self_canonical(self):
        shell = b'<html lang="ko-KR" ng-app="tour"><head><title>Tour</title><link rel="canonical" href="https://ko-go-dev.shuijingwanwq.com/tour/"></head><body><div class="bar top-bar"></div><div ng-view></div></body></html>'
        PREVIEW.validate_raw_shell(shell, "/tour/", "/tour/", "ko-KR", "https://ko-go-dev.shuijingwanwq.com", True)
        with self.assertRaises(CORE.BrowserFailure):
            PREVIEW.validate_raw_shell(
                b'<link rel="canonical" href="https://ko-go-dev.shuijingwanwq.com/tour/welcome/1">',
                "/tour/", "/tour/", "ko-KR", "https://ko-go-dev.shuijingwanwq.com", True)

    def test_raw_list_and_course_shell_contracts(self):
        list_metadata = {"title": "강의 목록 — Go 언어 투어", "description": "목록 설명", "heading": "Go 언어 투어에 오신 것을 환영합니다"}
        template = '<html lang="ko-KR" ng-app="tour"><head><title>%s</title><meta name="description" content="%s"><link rel="canonical" href="%%s"></head><body><div class="bar top-bar"></div><div ng-view></div></body></html>' % (list_metadata["title"], list_metadata["description"])
        PREVIEW.validate_raw_shell((template % "https://ko-go-dev.shuijingwanwq.com/tour/list").encode(),
                                   "/tour/list", "/tour/list", "ko-KR", "https://ko-go-dev.shuijingwanwq.com", True, list_metadata)
        PREVIEW.validate_raw_shell((template % "https://ko-go-dev.shuijingwanwq.com/tour/").encode(),
                                   "/tour/welcome/1", "/tour/", "ko-KR", "https://ko-go-dev.shuijingwanwq.com", True)
        wrong_description = (template % "https://ko-go-dev.shuijingwanwq.com/tour/list").replace("목록 설명", "Tour")
        with self.assertRaises(CORE.BrowserFailure):
            PREVIEW.validate_raw_shell(wrong_description.encode(), "/tour/list", "/tour/list", "ko-KR",
                                       "https://ko-go-dev.shuijingwanwq.com", True, list_metadata)

    def test_list_metadata_is_complete_for_all_formal_locales(self):
        identity = PREVIEW.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        for locale in (profile["locale"] for profile in identity["locales"]):
            with self.subTest(locale=locale):
                metadata = PREVIEW.formal_list_metadata(locale)
                self.assertTrue(all(metadata.values()))
                self.assertEqual(metadata, CORE.locale_list_metadata(locale))

    def test_production_identity_profiles_match_community_registry(self):
        registry = {entry["locale"]: entry["url"] for entry in PREVIEW.registry()}
        identity = PREVIEW.IDENTITY.load_identity(ROOT / "production" / "identity.json")
        for profile in identity["locales"]:
            with self.subTest(locale=profile["locale"]):
                self.assertEqual(registry[profile["locale"]], profile["production_public_url"])
        self.assertNotIn("en", {profile["locale"] for profile in identity["locales"]})

    def test_rendered_list_requires_exact_article_and_page_routes(self):
        page_routes = CORE.formal_course_routes()
        article_routes = sorted({route.rsplit('/', 1)[0] for route in page_routes})
        chrome = mock.Mock()
        chrome.evaluate.return_value = {
            "wrappers": 1, "heading": "Directory", "modules": 5,
            "articleRoutes": article_routes, "pageRoutes": page_routes,
        }
        CORE.validate_rendered_list(chrome, {"heading": "Directory"}, page_routes)
        chrome.evaluate.return_value = {
            "wrappers": 1, "heading": "Directory", "modules": 5,
            "articleRoutes": article_routes, "pageRoutes": page_routes[:-1],
        }
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_rendered_list(chrome, {"heading": "Directory"}, page_routes)
        self.assertIn(".toc .toc-page a", chrome.evaluate.call_args.args[0])

    def course_layout(self, direction="ltr"):
        if direction == "rtl":
            lesson = {"left": 640, "right": 1280, "top": 48, "bottom": 800,
                      "width": 640, "height": 752, "visibleWidth": 640, "visibleHeight": 752}
            editor = {"left": 0, "right": 640, "top": 48, "bottom": 800,
                      "width": 640, "height": 752, "visibleWidth": 640, "visibleHeight": 752}
            divider = {"left": 635, "right": 640}
        else:
            lesson = {"left": 0, "right": 640, "top": 48, "bottom": 800,
                      "width": 640, "height": 752, "visibleWidth": 640, "visibleHeight": 752}
            editor = {"left": 640, "right": 1280, "top": 48, "bottom": 800,
                      "width": 640, "height": 752, "visibleWidth": 640, "visibleHeight": 752}
            divider = {"left": 640, "right": 645}
        return {
            "missing": [], "direction": direction, "viewportWidth": 1280,
            "container": {"left": 0, "right": 1280, "width": 1280},
            "lesson": lesson, "editor": editor, "divider": divider,
            "overlapWidth": 0, "gapWidth": 0, "lessonText": "Visible lesson",
            "lessonTextRect": {"visibleWidth": 180, "visibleHeight": 32},
            "lessonTextHit": True, "editorHit": True, "splitterReady": True, "codeDirection": "ltr",
        }

    def test_desktop_course_layout_accepts_rtl_and_ltr_logical_panes(self):
        for direction in ("rtl", "ltr"):
            with self.subTest(direction=direction):
                chrome = mock.Mock()
                chrome.evaluate.return_value = self.course_layout(direction)
                self.assertEqual(CORE.validate_desktop_course_layout(chrome)["direction"], direction)
                self.assertIn("elementFromPoint", chrome.evaluate.call_args.args[0])

    def test_desktop_course_layout_rejects_human_gate_failure_geometry(self):
        for failure in ("overlap", "covered_text", "offscreen_editor"):
            with self.subTest(failure=failure):
                snapshot = self.course_layout("rtl")
                if failure == "overlap":
                    snapshot["lesson"] = dict(snapshot["editor"])
                    snapshot["overlapWidth"] = snapshot["lesson"]["width"]
                elif failure == "covered_text":
                    snapshot["lessonTextHit"] = False
                else:
                    snapshot["editor"].update(left=-640, right=0, visibleWidth=0)
                chrome = mock.Mock()
                chrome.evaluate.return_value = snapshot
                with self.assertRaises(CORE.BrowserFailure):
                    CORE.validate_desktop_course_layout(chrome)

    def mobile_course_editor_layout(self, direction="ltr"):
        return {
            "missing": [], "direction": direction, "viewportWidth": 375, "documentOverflow": 0,
            "pane": {"left": 0, "right": 375, "width": 375, "visibleWidth": 375},
            "explorer": {"left": 0, "right": 375, "top": 657, "bottom": 745,
                         "width": 375, "visibleWidth": 375},
            "parent": {"left": 1, "right": 374, "top": 748, "bottom": 1025,
                       "width": 373, "visibleWidth": 373},
            "parentClientWidth": 373,
            "controls": [
                {"left": 275, "right": 343, "top": 657, "bottom": 685,
                 "width": 68, "visibleWidth": 68, "float": "none"},
                {"left": 221, "right": 343, "top": 687, "bottom": 715,
                 "width": 122, "visibleWidth": 122, "float": "none"},
                {"left": 242, "right": 343, "top": 717, "bottom": 745,
                 "width": 101, "visibleWidth": 101, "float": "none"},
            ],
            "syntaxFloat": "none", "importsFloat": "none",
            "file": {"left": 1, "right": 374, "width": 373, "visibleWidth": 373},
            "code": {"left": 1, "right": 374, "width": 373, "visibleWidth": 373},
            "fileDirection": "ltr", "codeDirection": "ltr",
        }

    def test_mobile_course_editor_layout_accepts_rtl_and_ltr_full_width(self):
        for direction in ("rtl", "ltr"):
            with self.subTest(direction=direction):
                chrome = mock.Mock()
                chrome.evaluate.return_value = self.mobile_course_editor_layout(direction)
                self.assertEqual(CORE.validate_mobile_course_editor_layout(chrome)["direction"], direction)
                self.assertIn("#top-part > .relative-content", chrome.evaluate.call_args.args[0])

    def test_mobile_course_editor_layout_rejects_narrow_or_offscreen_surfaces(self):
        for failure in ("narrow_file", "narrow_code", "offscreen_code", "overflow", "code_rtl",
                        "floating_toggle", "short_explorer", "same_row", "editor_intrusion"):
            with self.subTest(failure=failure):
                snapshot = self.mobile_course_editor_layout("rtl")
                if failure == "narrow_file":
                    snapshot["file"].update(left=231, width=143, visibleWidth=143)
                elif failure == "narrow_code":
                    snapshot["code"].update(left=231, width=143, visibleWidth=143)
                elif failure == "offscreen_code":
                    snapshot["code"].update(left=-200, right=173, visibleWidth=173)
                elif failure == "overflow":
                    snapshot["documentOverflow"] = 30
                elif failure == "code_rtl":
                    snapshot["codeDirection"] = "rtl"
                elif failure == "floating_toggle":
                    snapshot["syntaxFloat"] = "left"
                elif failure == "short_explorer":
                    snapshot["explorer"]["bottom"] = 685
                elif failure == "same_row":
                    snapshot["controls"][1].update(top=657, bottom=685)
                else:
                    snapshot["parent"]["top"] = 700
                chrome = mock.Mock()
                chrome.evaluate.return_value = snapshot
                with self.assertRaises(CORE.BrowserFailure):
                    CORE.validate_mobile_course_editor_layout(chrome)

    def rendered(self, path, canonical):
        return {"path": path, "href": "http://127.0.0.1:38573" + path,
                "origin": "http://127.0.0.1:38573", "lang": "ko-KR",
                "canonical": canonical, "title": "Lesson — Tour", "description": "description",
                "renderedRoute": path, "heading": "Lesson"}

    def test_tour_redirect_semantics_converge_without_another_navigation(self):
        stale = self.rendered("/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/")
        stale["renderedRoute"] = "/tour/"
        converged = self.rendered(
            "/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/welcome/1")
        chrome = mock.Mock()
        chrome.evaluate.side_effect = [stale, converged]
        with mock.patch.object(CORE.time, "sleep"):
            result = CORE.wait_for_rendered_identity(
                chrome, "http://127.0.0.1:38573/", "ko-KR", "/tour/",
                "https://ko-go-dev.shuijingwanwq.com", "/tour/welcome/1", "/tour/welcome/1", timeout=1)
        self.assertEqual(result, converged)
        chrome.navigate.assert_not_called()

    def test_permanent_canonical_mismatch_fails_after_bounded_convergence(self):
        stale = self.rendered("/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/")
        chrome = mock.Mock()
        chrome.evaluate.return_value = stale
        with self.assertRaises(CORE.BrowserFailure) as caught:
            CORE.wait_for_rendered_identity(
                chrome, "http://127.0.0.1:38573/", "ko-KR", "/tour/",
                "https://ko-go-dev.shuijingwanwq.com", "/tour/welcome/1", "/tour/welcome/1", timeout=0)
        self.assertIn("semantic convergence timed out after 0s", str(caught.exception))
        self.assertIn("canonical mismatch", str(caught.exception))
        self.assertEqual(chrome.evaluate.call_count, 1)

    def test_wrong_lang_and_origin_fail_without_convergence_retry(self):
        for field, value in (("lang", "wrong"), ("origin", "https://wrong.example")):
            with self.subTest(field=field):
                identity = self.rendered(
                    "/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/welcome/1")
                identity[field] = value
                chrome = mock.Mock()
                chrome.evaluate.return_value = identity
                with self.assertRaises(CORE.PermanentBrowserFailure):
                    CORE.wait_for_rendered_identity(
                        chrome, "http://127.0.0.1:38573/", "ko-KR", "/tour/welcome/1",
                        "https://ko-go-dev.shuijingwanwq.com", "/tour/welcome/1",
                        "/tour/welcome/1", timeout=100)
                self.assertEqual(chrome.evaluate.call_count, 1)

    def test_identity_mismatch_includes_sanitized_main_document_evidence(self):
        for field, value, label in (("lang", "zh", "html lang mismatch"),
                                    ("origin", "https://wrong.example", "production hostname mismatch")):
            with self.subTest(field=field):
                identity = {
                    "lang": "it-IT", "href": "https://it.example/tour/welcome/1",
                    "origin": "https://it.example", "path": "/tour/welcome/1",
                    "canonical": "https://it.example/tour/welcome/1", "title": "Tour italiano",
                    "description": "Descrizione", "renderedRoute": "/tour/welcome/1",
                    "heading": "Benvenuti",
                }
                identity[field] = value
                chrome = CORE.Chrome.__new__(CORE.Chrome)
                chrome.current_navigation = {
                    "requested_url": "https://it.example/tour/welcome/1",
                    "attempt": 1, "loaderId": "loader-main", "frameId": "frame-main",
                }
                chrome.events = [
                    {"method": "Network.requestWillBeSent", "params": {
                        "requestId": "document-main", "loaderId": "loader-main", "frameId": "frame-main",
                        "type": "Document", "request": {
                            "url": "https://it.example/tour/welcome/1",
                            "headers": {"Cookie": "request-secret", "Authorization": "Bearer request-secret"},
                        },
                    }},
                    {"method": "Network.responseReceived", "params": {
                        "requestId": "document-main", "loaderId": "loader-main", "frameId": "frame-main",
                        "type": "Document", "response": {
                            "url": "https://it.example/tour/welcome/1", "status": 200,
                            "headers": {
                                "CF-Cache-Status": "HIT", "Age": "42", "CF-Ray": "ray-test",
                                "Content-Type": "text/html; charset=utf-8", "Cache-Control": "public, max-age=60",
                                "Set-Cookie": "response-secret", "Authorization": "Bearer response-secret",
                                "Cookie": "response-cookie-secret",
                            },
                        },
                    }},
                ]
                chrome.evaluate = mock.Mock(return_value=identity)
                with self.assertRaises(CORE.PermanentBrowserFailure) as caught:
                    CORE.wait_for_rendered_identity(
                        chrome, "https://it.example/", "it-IT", "/tour/welcome/1",
                        "https://it.example", "/tour/welcome/1", "/tour/welcome/1", timeout=100)
                self.assertEqual(chrome.evaluate.call_count, 1)
                evidence = str(caught.exception)
                for expected in (
                        label, "requested_url", "https://it.example/tour/welcome/1", "final_response_url",
                        "http_status': 200", "request_id': 'document-main", "loader_id': 'loader-main",
                        "cf-cache-status': 'HIT", "age': '42", "cf-ray': 'ray-test",
                        "content-type': 'text/html; charset=utf-8", "cache-control': 'public, max-age=60",
                        "canonical': 'https://it.example/tour/welcome/1", "title': 'Tour italiano'",
                        "description': 'Descrizione'", "heading': 'Benvenuti'"):
                    self.assertIn(expected, evidence)
                for forbidden in ("Cookie", "Authorization", "Set-Cookie", "request-secret",
                                  "response-secret", "response-cookie-secret"):
                    self.assertNotIn(forbidden, evidence)

    def test_matching_identity_does_not_collect_failure_evidence(self):
        chrome = mock.Mock()
        identity = {"lang": "it-IT", "origin": "https://it.example"}
        CORE.validate_identity_invariants(identity, "https://it.example/", "it-IT", "/", chrome)
        chrome.main_document_response_evidence.assert_not_called()

    def test_spa_transition_waits_for_route_and_canonical_together(self):
        stale = {"lang": "ko-KR", "origin": "http://127.0.0.1:38573",
                 "path": "/tour/basics/12", "canonical": "https://ko.example/tour/basics/11",
                 "renderedRoute": "/tour/basics/11", "header": 1, "footer": 1,
                 "next": True, "body": "x" * 30, "mounts": 1}
        converged = dict(stale, canonical="https://ko.example/tour/basics/12",
                         renderedRoute="/tour/basics/12")
        chrome = mock.Mock()
        chrome.evaluate.side_effect = [stale, converged]
        with mock.patch.object(CORE.time, "sleep"):
            result = CORE.wait_for_spa_transition(
                chrome, "/tour/basics/11", "http://127.0.0.1:38573/", "ko-KR",
                "https://ko.example", timeout=1)
        self.assertEqual(result, converged)
        chrome.navigate.assert_not_called()

    def test_rendered_tour_redirect_is_exact(self):
        CORE.validate_rendered_identity(
            self.rendered("/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/welcome/1"),
            "http://127.0.0.1:38573/", "ko-KR", "/tour/", "/tour/welcome/1",
            "https://ko-go-dev.shuijingwanwq.com", "/tour/welcome/1", "description")
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_rendered_identity(
                self.rendered("/tour/basics/1", "https://ko-go-dev.shuijingwanwq.com/tour/basics/1"),
                "http://127.0.0.1:38573/", "ko-KR", "/tour/", "/tour/welcome/1",
                "https://ko-go-dev.shuijingwanwq.com", "/tour/welcome/1", "description")

    def test_ordinary_rendered_route_cannot_redirect(self):
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_rendered_identity(
                self.rendered("/tour/basics/2", "https://ko-go-dev.shuijingwanwq.com/tour/basics/2"),
                "http://127.0.0.1:38573/", "ko-KR", "/tour/basics/1", "/tour/basics/1",
                "https://ko-go-dev.shuijingwanwq.com")

    def test_rendered_canonical_must_follow_final_route(self):
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_rendered_identity(
                self.rendered("/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/"),
                "http://127.0.0.1:38573/", "ko-KR", "/tour/", "/tour/welcome/1",
                "https://ko-go-dev.shuijingwanwq.com")

    def test_rendered_course_requires_marker_and_formal_description(self):
        wrong_marker = self.rendered("/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/welcome/1")
        wrong_marker["renderedRoute"] = "/tour/"
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_rendered_identity(wrong_marker, "http://127.0.0.1:38573/", "ko-KR",
                "/tour/welcome/1", "/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com",
                "/tour/welcome/1", "description")
        wrong_description = self.rendered("/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com/tour/welcome/1")
        with self.assertRaises(CORE.BrowserFailure):
            CORE.validate_rendered_identity(wrong_description, "http://127.0.0.1:38573/", "ko-KR",
                "/tour/welcome/1", "/tour/welcome/1", "https://ko-go-dev.shuijingwanwq.com",
                "/tour/welcome/1", "formal description")

    def test_mode_specific_browser_contracts_remain_distinct(self):
        source = (ROOT / "scripts" / "browser_acceptance.py").read_text(encoding="utf-8")
        self.assertIn('playground_requests(requests, origin, "/_/compile", "/_/fmt")', source)
        self.assertIn('shared["playground_public_origin"]', source)
        self.assertIn("browser_ad_gate(snapshot, chrome.network_requests(), policy[\"tour_ads_enabled\"])", source)
        preview_body = source.split("def preview_acceptance", 1)[1]
        self.assertIn('if not policy["tour_ads_enabled"]:', preview_body)
        self.assertIn("browser_ad_gate(editor, chrome.network_requests(), False)", preview_body)
        self.assertIn("fetch('/socket')", preview_body)
        self.assertIn("wait_for_spa_transition(chrome, before, base, locale, canonical_origin)", preview_body)
        self.assertIn("validate_mobile_course_editor_layout(chrome)", preview_body)
        production_body = source.split("def acceptance", 1)[1].split("def preview_acceptance", 1)[0]
        self.assertIn("post_identity_check=lambda: validate_mobile_course_editor_layout(chrome)", production_body)

    def test_production_still_requires_https_formal_identity(self):
        original = sys.argv
        try:
            sys.argv = ["verify-production-browser.py", "http://127.0.0.1:38573/", "ko-KR"]
            self.assertEqual(PRODUCTION.main(), 1)
        finally: sys.argv = original

class PreviewSiteV2Test(unittest.TestCase):
    base = "http://127.0.0.1:3999/"
    profile = {"locale": "fr-FR", "production_public_url": "https://locale.example/"}

    def page(self, route="/learn/", family="/learn/**", code=True, executable=False, format="md"):
        content = '<h1>A meaningful fixture page</h1><a href="/tour/">Tour</a>'
        if code: content += '<pre><code>package main</code></pre>'
        if executable:
            content += '<textarea class="js-playgroundCodeEl"></textarea><button class="js-playgroundRunEl">Run</button><div class="js-playgroundOutputEl"><pre></pre></div>'
        html = ('<html lang="fr-FR"><head><title>Fixture</title><link rel="canonical" href="https://locale.example%s">'
                '<link rel="stylesheet" href="https://go.dev/css/styles.css"></head><body><header class="Site-header"></header>'
                '<main>%s</main><footer class="Site-footer"></footer></body></html>') % (route, content)
        return {"route": route, "family": family, "format": format, "html": html, "required_links": ["/tour/"]}

    def test_bare_quoted_url_localization_is_not_link_loss(self):
        url = "https://example.invalid/item"
        for href in (url, url + "%22"):
            p = self.page()
            source = '<p>For example, "<a href="%s">%s"</a>.</p>' % (href, url)
            p["html"] = p["html"].replace("</main>", source + "</main>")
            target = p["html"].replace(source, '<p>Exemple : « %s ».</p>' % url)
            with self.subTest(href=href):
                self.assertIn(href, PREVIEW.structure(p["html"]).links)
                PREVIEW.validate_site_page(target.encode(), p, self.base, self.profile)

    def test_explicit_source_bound_link_loss_still_fails(self):
        # Targets supplied by the Go Markdown/reference/HTML/template parser
        # contract stay required even when ordinary prose repeats the same URL.
        for href in ("/learn/", "#fixed", "https://example.invalid/quoted%22", "https://example.invalid/item", "https://example.invalid/reference",
                     "https://example.invalid/angle", "https://example.invalid/html"):
            p = self.page()
            anchor = '<a href="%s">Translated label</a>' % href
            p["html"] = p["html"].replace("</main>", anchor + "</main>")
            if href == "#fixed": p["html"] = p["html"].replace("</main>", '<h2 id="fixed">Section</h2></main>')
            p["required_links"].append(href)
            PREVIEW.validate_site_page(p["html"].encode(), p, self.base, self.profile)
            target = p["html"].replace(anchor, '<p>Localized plain URL: %s</p>' % href)
            with self.subTest(href=href), self.assertRaisesRegex(CORE.BrowserFailure, "missing source-bound links"):
                PREVIEW.validate_site_page(target.encode(), p, self.base, self.profile)

    def test_missing_link_authority_fails_closed(self):
        for value in (None, "all rendered anchors", [None], [""]):
            p = self.page()
            p["required_links"] = value
            with self.subTest(value=value), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.page_contract(p)
        del p["required_links"]
        with self.assertRaises(CORE.BrowserFailure): PREVIEW.page_contract(p)

    def test_required_image_script_and_stylesheet_loss_still_fails(self):
        p = self.page()
        p["html"] = p["html"].replace("</main>", '<img src="/image.svg"><script src="/script.js"></script></main>')
        for asset in ("/image.svg", "/script.js", "https://go.dev/css/styles.css"):
            target = p["html"].replace(asset, "/missing-asset")
            with self.subTest(asset=asset), self.assertRaisesRegex(CORE.BrowserFailure, "missing source-bound assets"):
                PREVIEW.validate_site_page(target.encode(), p, self.base, self.profile)
        with self.assertRaisesRegex(CORE.BrowserFailure, "missing source-bound assets"):
            PREVIEW.validate_site_page(p["html"].replace('<img src="/image.svg">', '').encode(), p, self.base, self.profile)

    def test_current_activated_coverage_and_executable_static_distinction(self):
        # Read-only Go route/source/renderer plan; no HTTP, Chrome, language or
        # completion artifact mutation and no formal browser acceptance.
        plan = PREVIEW.browser_plan("zh-CN")
        scope = PREVIEW.activated_content_scope("zh-CN")
        routes = {r for p in scope["packages"] for r in p["local_canonical_routes"]}
        self.assertEqual(set(plan["canonical_routes"]), routes)
        pages = {p["route"]: p for p in plan["pages"]}
        # Real source/renderer lifecycle: prose linkification consumes a quote;
        # canonical explicit link authority excludes that incidental anchor.
        database = pages["/doc/security/vuln/database"]
        artifact = "https://pkg.go.dev/vuln/GO-2023-1621%22"
        self.assertIn(artifact, PREVIEW.structure(database["html"]).links)
        self.assertNotIn(artifact, database["required_links"])
        self.assertNotIn(artifact.removesuffix("%22"), database["required_links"])
        expected_pages = {r for p in scope["packages"] if p["package"] == "learn-docs-v1" for r in p["local_canonical_routes"]}
        self.assertEqual(len(expected_pages), 49)
        self.assertEqual(set(pages), expected_pages | {"/", "/translation/"})
        self.assertEqual([p["route"] for p in plan["pages"] if PREVIEW.page_contract(p).editor], ["/"])
        for route in ("/doc/modules/release-workflow", "/doc/security/", "/doc/tutorial/compile-install", "/doc/tutorial/generics"):
            self.assertEqual(PREVIEW.page_contract(pages[route]).editor, 0)
        self.assertGreater(PREVIEW.page_contract(pages["/doc/tutorial/compile-install"]).pre, 0)
        self.assertGreater(PREVIEW.page_contract(pages["/doc/tutorial/generics"]).pre, 0)
        self.assertEqual(PREVIEW.page_contract(pages["/doc/security/"]).pre, 0)
        sample = PREVIEW.browser_samples(plan)
        self.assertEqual({p["family"] for p in sample}, set(plan["families"])-{"/tour/**"})
        print("CURRENT_PLAN families=%s routes=%d site_pages=%d samples=%s static_pre_pages=%d" %
              (plan["families"], len(routes), len(pages), [p["route"] for p in sample],
               sum(bool(PREVIEW.page_contract(p).pre) and not PREVIEW.page_contract(p).editor for p in plan["pages"])))

    def test_sampling_is_sparse_locale_neutral_and_includes_every_executable(self):
        pages = [self.page("/learn/z"), self.page("/learn/a"), self.page("/learn/no-code", code=False),
                 self.page("/doc/tutorial/x", "/doc/tutorial/**", format="html"),
                 self.page("/learn/widget-z", executable=True), self.page("/learn/widget-a", executable=True)]
        selected = PREVIEW.browser_samples({"pages": pages})
        self.assertEqual([p["route"] for p in selected], ["/doc/tutorial/x", "/learn/a", "/learn/no-code", "/learn/widget-a", "/learn/widget-z"])

    def test_raw_renderer_navigation_code_and_assets_fail_closed(self):
        p = self.page()
        PREVIEW.validate_site_page(p["html"].encode(), p, self.base, self.profile)
        for before, after in (("<main>", '<main ng-app="tour">'), ("<main>", "<section>"),
                              ('href="/tour/"', 'href="https://go.dev/tour/"'),
                              ('https://locale.example/learn/', 'http://127.0.0.1:3999/learn/'),
                              ('<pre><code>package main</code></pre>', ''),
                              ('https://go.dev/css/styles.css', '/missing.css'),
                              ('A meaningful fixture page', ''), ('Site-footer', 'missing-footer')):
            with self.subTest(before=before), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.validate_site_page(p["html"].replace(before, after).encode(), p, self.base, self.profile)

    def test_unknown_executable_is_rejected_and_static_never_runs(self):
        p = self.page()
        self.assertEqual(PREVIEW.page_contract(p).editor, 0)
        p["html"] += '<textarea class="future-widget"></textarea>'
        with self.assertRaises(CORE.BrowserFailure): PREVIEW.page_contract(p)

    def test_plan_identity_and_exact_route_set_fail_closed(self):
        valid = {"schema": "go-learning/preview-browser-plan/v1", "locale": "fr-FR", "canonical_routes": ["/learn/"], "pages": [self.page()]}
        for field, value in (("schema", "unknown"), ("locale", "zh-CN"), ("pages", []), ("canonical_routes", ["/learn/", "/learn/"])):
            result = mock.Mock(returncode=0, stdout=json.dumps(dict(valid, **{field: value})))
            with mock.patch.object(PREVIEW.subprocess, "run", return_value=result), self.subTest(field=field), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.browser_plan("fr-FR")
        with mock.patch.object(PREVIEW.subprocess, "run", return_value=mock.Mock(returncode=1, stderr="stale authority")), self.assertRaises(CORE.BrowserFailure):
            PREVIEW.browser_plan("fr-FR")

    def test_machine_visits_every_page_and_family_negative_without_redirects(self):
        pages = [self.page(), self.page("/doc/tutorial/x", "/doc/tutorial/**")]
        plan = {"pages": pages, "families": ["/learn/**", "/doc/tutorial/**", "/tour/**"]}
        def response(base, route, **kwargs):
            self.assertFalse(kwargs["follow_redirects"])
            p = next((p for p in pages if p["route"] == route), None)
            return (200, p["html"].encode()) if p else (404, b"missing")
        with mock.patch.object(PREVIEW, "request", side_effect=response) as request:
            PREVIEW.site_machine_acceptance(self.base, self.profile, plan)
            self.assertEqual(request.call_count, 5)
            for bad in (302, 404, 500):
                request.side_effect = lambda *a, **kw: (bad, b"bad")
                with self.subTest(status=bad), self.assertRaises(CORE.BrowserFailure):
                    PREVIEW.site_machine_acceptance(self.base, self.profile, plan)
            request.side_effect = lambda *a, **kw: (200, pages[0]["html"].encode()) if "__preview_missing" in a[1] else response(*a, **kw)
            with self.assertRaises(CORE.BrowserFailure): PREVIEW.site_machine_acceptance(self.base, self.profile, plan)

    def test_canceled_network_requests_have_no_telemetry_exemption(self):
        # Preview removes the source GTM injection. If any such traffic still
        # appears, even its genuine canceled Fetch shape remains a failure.
        for url, kind, method, canceled, error in (
            (self.base + "_/compile", "Fetch", "POST", True, "net::ERR_ABORTED"),
            (self.base + "doc/play/hello.go", "Fetch", "GET", True, "net::ERR_ABORTED"),
            ("https://example.invalid/required.js", "Script", "GET", True, "net::ERR_ABORTED"),
            ("https://ad.doubleclick.net/ccm/s/collect?test=1", "Fetch", "POST", True, "net::ERR_ABORTED"),
            ("https://www.google.com/ccm/collect?test=1", "Fetch", "POST", True, "net::ERR_ABORTED"),
            ("https://www.google.com/required.js", "Script", "GET", True, "net::ERR_ABORTED"),
            ("https://ad.doubleclick.net/unexpected", "Fetch", "POST", True, "net::ERR_ABORTED"),
            ("https://www.google.com/ccm/collect?test=1", "Fetch", "POST", False, "net::ERR_FAILED"),
        ):
            events = [
                {"method": "Network.requestWillBeSent", "params": {"requestId": "request", "type": kind,
                 "request": {"url": url, "method": method}, "initiator": {"type": "script", "stack": {"callFrames": [
                     {"url": "https://www.googletagmanager.com/gtm.js?id=GTM-TEST"}]}}}},
                {"method": "Network.loadingFailed", "params": {"requestId": "request", "type": kind,
                 "errorText": error, "canceled": canceled}},
            ]
            with self.subTest(url=url, canceled=canceled), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.assert_browser_events(mock.Mock(events=events, current_route="/"), self.base)

    def test_browser_errors_and_network_failures_are_not_silent(self):
        events = [
            {"method": "Runtime.exceptionThrown", "params": {"exceptionDetails": {"text": "broken"}}},
            {"method": "Runtime.consoleAPICalled", "params": {"type": "error"}},
            {"method": "Runtime.consoleAPICalled", "params": {"type": "assert"}},
            {"method": "Log.entryAdded", "params": {"entry": {"level": "error"}}},
            {"method": "Network.loadingFailed", "params": {"errorText": "net::ERR_FAILED"}},
            {"method": "Network.webSocketCreated", "params": {"url": "ws://127.0.0.1:3999/socket"}},
            {"method": "Network.webSocketFrameError", "params": {"errorMessage": "connection closed"}},
            {"method": "Network.webSocketHandshakeResponseReceived", "params": {"response": {"status": 500}}},
            {"method": "Network.responseReceived", "params": {"response": {"url": "https://go.dev/js/index.js", "status": 404}}},
            {"method": "Page.javascriptDialogOpening", "params": {"message": "Server error; try again."}},
        ]
        for event in events:
            with self.subTest(event=event), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.assert_browser_events(mock.Mock(events=[event], current_route="/"), self.base)
        deliberate_probe = {"method": "Network.responseReceived", "params": {"type": "Fetch", "response": {"url": self.base+"socket", "status": 404}}}
        PREVIEW.assert_browser_events(mock.Mock(events=[deliberate_probe]), self.base)
        deliberate_probe["params"]["response"]["status"] = 500
        with self.assertRaises(CORE.BrowserFailure): PREVIEW.assert_browser_events(mock.Mock(events=[deliberate_probe], current_route="/tour/basics/11"), self.base)

    def test_navigation_retry_cannot_erase_browser_errors(self):
        def init(chrome):
            chrome.events = []; chrome.current_route = "/"
        with mock.patch.object(CORE.Chrome, "__init__", init), mock.patch.object(CORE.Chrome, "call"):
            chrome = PREVIEW.preview_chrome_factory(self.base)()
            chrome.events.append({"method": "Runtime.consoleAPICalled", "params": {"type": "error"}})
            with self.assertRaises(CORE.BrowserFailure): chrome.events.clear()
            self.assertEqual(len(chrome.events), 1)

    def test_pending_resources_fail_and_completed_resources_settle(self):
        chrome = mock.Mock(events=[{"method": "Network.requestWillBeSent", "params": {"requestId": "asset"}}], current_route="/learn/")
        chrome.evaluate.return_value = True
        with self.assertRaises(CORE.BrowserFailure): PREVIEW.settle_site_page(chrome, self.base, timeout=0)
        chrome.events.append({"method": "Network.loadingFinished", "params": {"requestId": "asset"}})
        with mock.patch.object(PREVIEW.time, "sleep"), mock.patch.object(PREVIEW.time, "monotonic", side_effect=[0, 0, 0, 1]):
            PREVIEW.settle_site_page(chrome, self.base)

    def test_browser_samples_visit_both_viewports_static_code_is_not_executed(self):
        p = self.page(); chrome = mock.Mock(events=[])
        chrome.evaluate.side_effect = [p["html"], {"path": "/learn/", "origin": self.base.rstrip('/'), "text": "x"*30, "width": 100, "overflow": 0, "images": True, "anchors": True},
                                       p["html"], {"path": "/learn/", "origin": self.base.rstrip('/'), "text": "x"*30, "width": 100, "overflow": 0, "images": True, "anchors": True}, True, True, True, True]
        with mock.patch.object(PREVIEW, "settle_site_page"), mock.patch.object(PREVIEW, "settle_site_images") as images, mock.patch.object(PREVIEW, "run_site_example") as run:
            PREVIEW.site_browser_acceptance(chrome, self.base, self.profile, {"pages": [p]})
            run.assert_not_called()
            self.assertEqual(images.call_args_list, [mock.call(chrome, self.base)] * 2)
        self.assertEqual(chrome.navigate.call_args_list, [mock.call(self.base+"learn/", 1280, 800), mock.call(self.base+"learn/", 375, 812)])
        if shutil.which("node"):
            expressions = [c.args[0] for c in chrome.evaluate.call_args_list]
            expression = next(c.args[0] for c in chrome.evaluate.call_args_list if c.kwargs.get("check") == "Site visible content/assets/overflow")
            script = """const assert=require('node:assert/strict');
              const data=JSON.parse(require('node:fs').readFileSync(0,'utf8'));
              for(const e of data.expressions) new Function(e);
              globalThis.location={origin:'http://127.0.0.1:3999',pathname:'/learn/'};
              const main={innerText:'Visible main fixture content',getBoundingClientRect:()=>({width:375}),
                querySelectorAll:()=>[{href:'http://127.0.0.1:3999/learn/#section'},{href:'https://go.dev/elsewhere#external'}]};
              globalThis.document={querySelector:()=>main,documentElement:{scrollWidth:375,clientWidth:375},
                querySelectorAll:()=>[{complete:true,naturalWidth:20,naturalHeight:10}],getElementById:id=>id==='section'?{}:null};
              assert.equal(eval(data.expression).images,true);
              document.querySelectorAll=()=>[{complete:true,naturalWidth:0,naturalHeight:0}];
              assert.equal(eval(data.expression).images,false);
              assert.equal(eval(data.expression).anchors,true);
              document.getElementById=()=>null;
              assert.equal(eval(data.expression).anchors,false);
            """
            result = subprocess.run(["node", "-e", script], input=json.dumps({"expression": expression, "expressions": expressions}), text=True, capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_real_widget_click_output_completion_response_and_failure_contract(self):
        chrome = mock.Mock(events=[{"method": "Network.responseReceived", "params": {"requestId": "compile", "response": {"url": self.base+"_/compile", "status": 200}}}], current_route="/")
        chrome.evaluate.side_effect = [True, True, {"stdout": "SITE_V2_BROWSER_ACCEPTANCE_OK\n", "system": 1, "stderr": 0}, True, {"error": True, "text": "syntax error: unexpected {"}]
        chrome.network_requests.side_effect = [[], [{"method": "POST", "url": self.base+"_/compile?backend="}, {"method": "POST", "url": self.base+"_/fmt?backend="}]]
        chrome.call.return_value = {"body": json.dumps({"Status": 0, "Events": [{"Kind": "stdout", "Message": "SITE_V2_BROWSER_ACCEPTANCE_OK\n"}]})}
        with mock.patch.object(PREVIEW, "settle_site_page"):
            PREVIEW.run_site_example(chrome, self.base)
        self.assertIn(".click()", chrome.evaluate.call_args_list[1].args[0])
        self.assertIn('package main\\nimport "fmt"\\n', chrome.evaluate.call_args_list[1].args[0])
        self.assertNotIn('package main\nimport "fmt"\n', chrome.evaluate.call_args_list[1].args[0])
        self.assertIn(".js-playgroundRunEl", chrome.evaluate.call_args_list[3].args[0])
        chrome.call.assert_called_once_with("Network.getResponseBody", {"requestId": "compile"})

    def test_widget_is_exercised_once_and_mobile_overflow_cannot_pass(self):
        p = self.page("/", "/", executable=True)
        for overflow in (0, 20):
            chrome = mock.Mock(events=[])
            snapshot = {"path": "/", "origin": self.base.rstrip('/'), "text": "x"*30, "width": 100, "overflow": 0, "images": True, "anchors": True}
            chrome.evaluate.side_effect = [p["html"], snapshot, p["html"], dict(snapshot, overflow=overflow), True, True, True, True]
            with mock.patch.object(PREVIEW, "settle_site_page"), mock.patch.object(PREVIEW, "settle_site_images"), mock.patch.object(PREVIEW, "run_site_example") as run, self.subTest(overflow=overflow):
                if overflow:
                    with self.assertRaises(CORE.BrowserFailure): PREVIEW.site_browser_acceptance(chrome, self.base, self.profile, {"pages": [p]})
                else:
                    PREVIEW.site_browser_acceptance(chrome, self.base, self.profile, {"pages": [p]})
                run.assert_called_once_with(chrome, self.base)

    def test_compile_failure_cannot_pass_with_matching_visible_stdout(self):
        chrome = mock.Mock(events=[{"method": "Network.responseReceived", "params": {"requestId": "compile", "response": {"url": self.base+"_/compile", "status": 200}}}], current_route="/")
        chrome.evaluate.side_effect = [True, True, {"stdout": "SITE_V2_BROWSER_ACCEPTANCE_OK", "system": 1, "stderr": 0}]
        chrome.network_requests.side_effect = [[], [{"method": "POST", "url": self.base+"_/compile?backend="}, {"method": "POST", "url": self.base+"_/fmt?backend="}]]
        chrome.call.return_value = {"body": json.dumps({"Status": 1, "Events": [{"Kind": "stdout", "Message": "SITE_V2_BROWSER_ACCEPTANCE_OK\n"}]})}
        with mock.patch.object(PREVIEW, "settle_site_page"), self.assertRaises(CORE.BrowserFailure): PREVIEW.run_site_example(chrome, self.base)

    def test_incomplete_widget_output_cannot_pass(self):
        for value in ({"stdout": "", "system": 1, "stderr": 0},
                      {"stdout": "SITE_V2_BROWSER_ACCEPTANCE_OK", "system": 0, "stderr": 0},
                      {"stdout": "SITE_V2_BROWSER_ACCEPTANCE_OK", "system": 1, "stderr": 1}):
            chrome = mock.Mock(events=[], current_route="/")
            chrome.network_requests.return_value = []
            chrome.evaluate.side_effect = [True, True, value]
            # Exercise the actual completion assertion once; no 25-second wait.
            with mock.patch.object(CORE, "wait_for_condition", side_effect=lambda check, label: check()), self.subTest(value=value), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.run_site_example(chrome, self.base)

class PreviewHeadingFragmentTest(unittest.TestCase):
    base = "https://locale.example/"

    def page(self, href="#input-minimization", target="input-minimization"):
        return {"route": "/doc/security/fuzz/technical", "required_links": [href], "format": "md",
                "html": '<main><a href="%s">最小化输入</a><a href="%s">下文讨论</a><h2 id="%s">输入最小化</h2></main>' % (href, href, target)}

    def test_special_top_without_id_is_ascii_case_insensitive(self):
        for href in ("#top", "#TOP", "#ToP", "#%74%4F%50"):
            with self.subTest(href=href):
                page = self.page(href, "unrelated")
                self.assertEqual(PREVIEW.validate_fragment_targets(PREVIEW.structure(page["html"]), page, self.base), 2)

    def test_ordinary_fragments_still_require_exact_targets(self):
        for href, target, valid in (("#ordinary", "ordinary", True), ("#ordinary", "other", False),
                                    ("#ordinary", "ORDINARY", False), ("#topmost", "other", False),
                                    ("#top%0A", "other", False), ("#ｔｏｐ", "other", False)):
            page = self.page(href, target)
            with self.subTest(href=href, target=target):
                if valid:
                    self.assertEqual(PREVIEW.validate_fragment_targets(PREVIEW.structure(page["html"]), page, self.base), 2)
                else:
                    with self.assertRaisesRegex(CORE.BrowserFailure, "missing id="):
                        PREVIEW.validate_fragment_targets(PREVIEW.structure(page["html"]), page, self.base)

    def test_fuzz_back_to_top_without_synthetic_target(self):
        page = {"route": "/doc/tutorial/fuzz", "required_links": ["#top"], "format": "md"}
        actual = PREVIEW.structure('<main><p>Tutorial content</p><a href="#top">Back to top</a></main>')
        self.assertNotIn("top", actual.ids)
        self.assertEqual(PREVIEW.validate_fragment_targets(actual, page, self.base), 1)

    def test_browser_anchor_predicate_preserves_top_and_exact_target_semantics(self):
        fixture = PreviewSiteV2Test(); page = fixture.page(code=False)
        snapshot = {"path": page["route"], "origin": fixture.base.rstrip('/'), "text": "x"*30,
                    "width": 100, "overflow": 0, "images": True, "anchors": True}
        chrome = mock.Mock(events=[])
        def evaluate(expression, **kwargs):
            if kwargs.get("check") == "Site rendered DOM": return page["html"]
            if kwargs.get("check") == "Site visible content/assets/overflow": return snapshot
            return True
        chrome.evaluate.side_effect = evaluate
        with mock.patch.object(PREVIEW, "settle_site_page"), mock.patch.object(PREVIEW, "settle_site_images"), \
                mock.patch.object(PREVIEW, "check_visible_breadcrumbs"):
            PREVIEW.site_browser_acceptance(chrome, fixture.base, fixture.profile, {"pages": [page]})
        expression = next(call.args[0] for call in chrome.evaluate.call_args_list
                          if call.kwargs.get("check") == "Site visible content/assets/overflow")
        # Execute the exact browser gate, not a test copy of its fragment rule.
        script = """const expression=JSON.parse(require('fs').readFileSync(0,'utf8'));
          global.location=new URL('https://locale.example/learn/');
          for(const [href,ids,want] of [
            ['#top',[],true],['#TOP',[],true],['#ToP',[],true],['#%74%4F%50',[],true],
            ['#ordinary',[],false],['#ordinary',['ordinary'],true],['#ordinary',['ORDINARY'],false],
            ['#input-minimization',['heading-5'],false],['#topmost',[],false],['#top%0A',[],false],['#ｔｏｐ',[],false]]){
            const main={innerText:'Substantive content',getBoundingClientRect:()=>({width:100}),
              querySelectorAll:()=>[{href:new URL(href,location).href}]};
            global.document={querySelector:()=>main,documentElement:{scrollWidth:1280,clientWidth:1280},
              querySelectorAll:()=>[],getElementById:id=>ids.includes(id)?{}:null};
            if(eval(expression).anchors!==want)throw Error(JSON.stringify({href,ids,want}));
          }"""
        result = subprocess.run(["node", "-e", script], input=json.dumps(expression), text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_source_fragment_and_duplicate_links_have_localized_target(self):
        page = self.page()
        self.assertEqual(PREVIEW.validate_fragment_targets(PREVIEW.structure(page["html"]), page, self.base), 2)
        for target in ("heading-5", "", "INPUT-MINIMIZATION"):
            with self.subTest(target=target), self.assertRaisesRegex(CORE.BrowserFailure, "missing id=input-minimization"):
                broken = self.page(target=target)
                PREVIEW.validate_fragment_targets(PREVIEW.structure(broken["html"]), page, self.base)

    def test_resolved_same_page_fragments_and_unicode(self):
        for href in ("#caf%C3%A9", "#café", "/doc/security/fuzz/technical#café",
                     "https://locale.example/doc/security/fuzz/technical?mode=exact#caf%C3%A9"):
            with self.subTest(href=href):
                page = self.page(href, "café")
                self.assertEqual(PREVIEW.validate_fragment_targets(PREVIEW.structure(page["html"]), page, self.base), 2)

    def test_external_and_cross_document_fragments_are_not_local_targets(self):
        for href in ("https://clang.llvm.org/docs/SanitizerCoverage.html#inline-8bit-counters",
                     "/doc/other#section", "https://other.invalid/doc/security/fuzz/technical#section", "#"):
            page = self.page(href, "unrelated")
            self.assertEqual(PREVIEW.validate_fragment_targets(PREVIEW.structure(page["html"]), page, self.base), 0)

    def test_required_source_fragment_cannot_disappear_with_anchor(self):
        page = self.page()
        with self.assertRaisesRegex(CORE.BrowserFailure, "missing id=input-minimization"):
            PREVIEW.validate_fragment_targets(PREVIEW.structure('<main>Localized body</main>'), page, self.base)

    def test_additional_rendered_fragment_must_also_resolve(self):
        page = self.page()
        body = page["html"].replace('</main>', '<a href="#missing">Missing</a></main>')
        with self.assertRaisesRegex(CORE.BrowserFailure, "missing id=missing"):
            PREVIEW.validate_fragment_targets(PREVIEW.structure(body), page, self.base)

    def test_malformed_fragment_fails_closed(self):
        for href in ("#bad%ZZ", "#bad%FF"):
            page = self.page(href, href[1:])
            with self.subTest(href=href), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.validate_fragment_targets(PREVIEW.structure(page["html"]), page, self.base)

    def test_html_and_shell_explicit_targets_remain_usable(self):
        page = self.page("#shell-id", "article-id"); page["format"] = "html"
        body = '<header id="shell-id"></header>' + page["html"]
        self.assertEqual(PREVIEW.validate_fragment_targets(PREVIEW.structure(body), page, self.base), 2)

    def test_all_activated_pages_are_scanned_before_browser(self):
        pages = [self.page(), self.page("#feedback", "feedback")]
        pages[1]["route"] = "/doc/security/vuln/"
        profile = {"locale": "fixture"}
        def validate(body, page, base, profile):
            actual = PREVIEW.structure(body.decode()); PREVIEW.validate_fragment_targets(actual, page, base); return actual
        with mock.patch.object(PREVIEW, "request", side_effect=[(200, p["html"].encode()) for p in pages]), \
                mock.patch.object(PREVIEW, "validate_site_page", side_effect=validate), contextlib.redirect_stdout(io.StringIO()) as log:
            PREVIEW.site_machine_acceptance(self.base, profile, {"pages": pages, "families": []})
        self.assertIn("MARKDOWN_PAGES_CHECKED=2 SAME_DOCUMENT_FRAGMENT_LINKS=4 BROKEN_FRAGMENT_TARGETS=0", log.getvalue())

class PreviewBreadcrumbTest(unittest.TestCase):
    def page(self):
        return {"route": "/doc/tutorial/getting-started", "family": "/doc/tutorial/**", "format": "html", "required_links": [],
                "html": '<main><ol class="SiteBreadcrumb"><li><a href="https://go.dev/doc">Documentation</a></li>'
                        '<li><a href="/doc/tutorial/">本地教程</a></li><li><a href="/doc/tutorial/getting-started">当前页面</a></li></ol></main>'}

    def test_nonempty_breadcrumbs_and_coverage_targets(self):
        page = self.page()
        PREVIEW.validate_breadcrumbs(page["html"], page, "http://127.0.0.1:3999/")
        for before, after in [('Documentation', ' \n\t '), ('本地教程', '&nbsp;'),
                              ('href="/doc/tutorial/"', 'href=""'), ('href="/doc/tutorial/"', 'href=" "'),
                              ('https://go.dev/doc', '/doc'), ('/doc/tutorial/', 'https://go.dev/doc/tutorial/'),
                              ('/doc/tutorial/', '/doc/tutorial'), ('/doc/tutorial/', 'https://wrong.invalid/tutorial/'),
                              ('SiteBreadcrumb', 'missing'), ('<li><a href="https://go.dev/doc">Documentation</a></li>', '<li> </li>')]:
            with self.subTest(before=before, after=after), self.assertRaises(CORE.BrowserFailure):
                PREVIEW.validate_breadcrumbs(page["html"].replace(before, after), page, "http://127.0.0.1:3999/")

    def test_no_breadcrumb_required_when_source_has_none(self):
        page = {"html": '<main><article>Security</article></main>', "route": "/doc/security/"}
        PREVIEW.validate_breadcrumbs(page["html"], page, "http://127.0.0.1:3999/")
        with self.assertRaises(CORE.BrowserFailure):
            PREVIEW.validate_breadcrumbs('<ol class="SiteBreadcrumb"></ol>', page, "http://127.0.0.1:3999/")

    def test_visible_breadcrumb_gate_is_mandatory(self):
        chrome = mock.Mock(); chrome.evaluate.return_value = False
        with self.assertRaisesRegex(CORE.BrowserFailure, "breadcrumb"):
            PREVIEW.check_visible_breadcrumbs(chrome)
        expression = chrome.evaluate.call_args.args[0]
        # Execute the actual verifier expression, including its visibility test.
        script = """const expression=JSON.parse(require('fs').readFileSync(0,'utf8'));
          for(const defect of ['none','empty-item','empty-anchor','empty-href','hidden']){
            const a={innerText:defect==='empty-anchor'?' ': 'Local title',getAttribute:()=>defect==='empty-href'?'':'/doc/tutorial/',getBoundingClientRect:()=>({width:100,height:20})};
            const li={innerText:defect==='empty-item'?' ':a.innerText,getBoundingClientRect:()=>({width:100,height:20}),querySelectorAll:()=>[a]};
            global.document={querySelectorAll:()=>[{getBoundingClientRect:()=>({width:100,height:20}),querySelectorAll:()=>[li]}]};
            global.getComputedStyle=()=>({display:'block',visibility:defect==='hidden'?'hidden':'visible',opacity:'1'});
            if(Boolean(eval(expression))!==(defect==='none'))throw Error(defect);
          }"""
        result = subprocess.run(["node", "-e", script], input=json.dumps(expression), text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_real_tutorial_representatives_are_selected(self):
        pages = []
        for route in ('/doc/tutorial/getting-started', '/doc/tutorial/handle-errors'):
            page = self.page(); page['route'] = route; pages.append(page)
        self.assertEqual([p['route'] for p in PREVIEW.browser_samples({'pages': pages})], [p['route'] for p in pages])

class PreviewSharedChromeTest(unittest.TestCase):
    def contract(self, locale="zh-CN"):
        identity = PREVIEW.IDENTITY.load_identity(ROOT / "production/identity.json")
        profile = PREVIEW.profile_for(identity, locale)
        return CORE.shared_chrome_contract(locale, profile["production_public_url"], PREVIEW.registry())

    def state(self, contract):
        state = {key: contract[key] for key in ("root", "github", "copyright", "unofficial", "home_label", "github_label", "legal", "legal_url")}
        state.update(nav_count=1, footer_count=1, landmarks=1, about=contract["about"], about_path="/translation/", rows=["identity", "legal"],
                     row_binding=True, placement=True, footer_visible=True, header_placement=True,
                     trigger=next(l["autonym"] for l in contract["registry"] if l["locale"] == contract["locale"]),
                     languages=[dict(locale=l["locale"], label=l["english_name"]+(" — "+l["autonym"] if l["autonym"] != l["english_name"] else ""),
                                     href=l["url"] if l["locale"] != contract["locale"] else "", current=l["locale"] == contract["locale"]) for l in contract["registry"]])
        return state

    def test_shared_registry_and_locale_metadata_contract(self):
        for locale in ("zh-CN", "fr-FR"):
            contract = self.contract(locale)
            CORE.validate_shared_chrome(self.state(contract), contract)
            self.assertEqual(bool(contract["legal"]), locale == "zh-CN")
        self.assertNotEqual(self.contract("fr-FR")["root"], self.contract()["root"])

    def test_missing_duplicate_reordered_wrong_label_url_or_current_fails(self):
        contract = self.contract()
        for mutation in ("missing", "duplicate", "order", "label", "url", "current", "official"):
            state = self.state(contract)
            if mutation == "missing": state["languages"].pop()
            if mutation == "duplicate": state["languages"].append(dict(state["languages"][0]))
            if mutation == "order": state["languages"].reverse()
            if mutation == "label": state["languages"][0]["label"] = "wrong label"
            if mutation == "url": state["languages"][0]["href"] = "https://example.invalid/"
            if mutation == "current": state["languages"][0]["current"] = True
            if mutation == "official": next(l for l in state["languages"] if l["locale"] == "en")["href"] = "https://example.invalid/tour/"
            with self.subTest(mutation=mutation), self.assertRaises(CORE.BrowserFailure): CORE.validate_shared_chrome(state, contract)

    def test_header_footer_metadata_presence_rows_and_placement_fail_closed(self):
        contract = self.contract()
        for key, value in (("nav_count", 0), ("footer_count", 0), ("footer_count", 2), ("landmarks", 2),
                           ("trigger", "wrong locale"), ("about", ""), ("root", "https://example.invalid/"),
                           ("github", "https://example.invalid/"), ("copyright", "© 2024–2026 永夜"),
                           ("legal", ""), ("legal_url", "/fake-icp"), ("unofficial", ""),
                           ("rows", ["identity"]), ("row_binding", False), ("placement", False),
                           ("footer_visible", False), ("header_placement", False)):
            state = self.state(contract); state[key] = value
            with self.subTest(key=key, value=value), self.assertRaises(CORE.BrowserFailure): CORE.validate_shared_chrome(state, contract)

    def test_desktop_and_existing_mobile_drawer_keyboard_checks_are_required(self):
        contract = self.contract()
        for mobile in (False, True):
            chrome = mock.Mock()
            chrome.evaluate.side_effect = [self.state(contract), mobile] + [True] * (8 if mobile else 4)
            CORE.check_shared_chrome(chrome, contract)
            self.assertEqual(chrome.call.call_count, 6 if mobile else 4)
            self.assertTrue(any("document.activeElement.matches" in c.args[0] for c in chrome.evaluate.call_args_list))
            # The exact browser expression must be valid JavaScript.
            expression = chrome.evaluate.call_args_list[0].args[0]
            result = subprocess.run(["node", "-e", "new Function(require('node:fs').readFileSync(0,'utf8'))"], input=expression, text=True, capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)

    def test_language_escape_open_menu_or_missing_focus_still_fails(self):
        contract = self.contract()
        for defect in ("menu remains open", "summary not focused"):
            chrome = mock.Mock()
            chrome.evaluate.side_effect = [self.state(contract), False, True, True, True, False]
            with self.subTest(defect=defect), self.assertRaisesRegex(CORE.BrowserFailure, "Escape/focus"):
                CORE.check_shared_chrome(chrome, contract)
            expression = chrome.evaluate.call_args.args[0]
            # Both predicates remain mandatory; neither failure may pass alone.
            script = """const expression=JSON.parse(require('fs').readFileSync(0,'utf8'));
              const summary={};
              for(const [open,focused] of [[true,true],[false,false]]){
                global.document={activeElement:focused?summary:{},querySelector:s=>s.endsWith('summary')?summary:{open}};
                if(eval(expression)!==false)throw Error('broken Escape contract accepted');
              }"""
            result = subprocess.run(["node", "-e", script], input=json.dumps(expression), text=True, capture_output=True, timeout=10)
            self.assertEqual(result.returncode, 0, result.stderr)

class PreviewLearnNavigationTest(unittest.TestCase):
    def expression(self):
        chrome = mock.Mock()
        chrome.evaluate.return_value = True
        PREVIEW.check_learn_navigation(chrome)
        self.assertEqual(chrome.evaluate.call_args.kwargs["check"], "Learn navigation anchor click")
        return chrome.evaluate.call_args.args[0]

    def run_cases(self, cases):
        # Execute the production verifier expression with the native URL parser.
        # href forms and click effects are independent; rejected anchors cannot
        # acquire a fake click or target merely by matching a raw prefix.
        script = """const assert=require('node:assert/strict');
          const p=JSON.parse(require('node:fs').readFileSync(0,'utf8'));
          for(const c of p.cases){
            globalThis.location=new URL(c.location||'https://locale.example/learn/');
            let clicks=0;
            const anchors=c.links.map(href=>({href,click(){
              clicks++;
              if(c.throws)throw new Error('genuine click failure');
              if(!c.prevented){location=new URL(href,location.href);if(c.wrongHash)location.hash='#wrong'}
            }}));
            const target={innerText:c.empty?'   ':'Substantive rendered section',
              getBoundingClientRect:()=>({width:c.zeroWidth?0:200,height:c.zeroHeight?0:80})};
            globalThis.getComputedStyle=()=>({display:c.hidden?'none':'block',
              visibility:c.invisible?'hidden':'visible',opacity:c.transparent?'0':'1'});
            globalThis.document={querySelectorAll(selector){assert.equal(selector,'.LearnNav a[href]');return anchors},
              getElementById(id){return !c.missing&&id===(c.id||'section')?target:null}};
            if(c.throws)assert.throws(()=>eval(p.expression),/genuine click failure/);
            else assert.equal(eval(p.expression),c.pass,JSON.stringify(c));
            assert.equal(clicks,c.clicks,JSON.stringify(c));
          }
        """
        result = subprocess.run(["node", "-e", script], input=json.dumps({"expression": self.expression(), "cases": cases}),
                                text=True, capture_output=True, timeout=10)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_fragment_path_and_absolute_same_page_links(self):
        self.run_cases([
            {"links": [href], "pass": True, "clicks": 1} for href in
            ("#section", "/learn/#section", "https://locale.example/learn/#section")
        ] + [
            {"links": ["https://example.invalid/#section", "/doc/#section", "/learn/#section"], "pass": True, "clicks": 1},
            {"links": ["#section"], "location": "https://locale.example/future/", "pass": True, "clicks": 1},
            {"links": ["/learn/#caf%C3%A9"], "id": "café", "pass": True, "clicks": 1},
        ])

    def test_cross_page_external_empty_and_invalid_anchors_are_not_selected(self):
        self.run_cases([
            {"links": [href], "pass": False, "clicks": 0} for href in
            ("/doc/#section", "https://example.invalid/learn/#section", "https://locale.example:444/learn/#section",
             "/learn/", "#", "#%20", "http://[invalid", "#%E0%A4")
        ] + [{"links": [], "pass": False, "clicks": 0}])

    def test_missing_target_blocked_click_and_nonrendered_targets_fail(self):
        self.run_cases([
            {"links": ["/learn/#missing"], "pass": False, "clicks": 1},
            *[{"links": ["#section"], "pass": False, "clicks": 1, failure: True} for failure in
              ("missing", "prevented", "wrongHash", "empty", "zeroWidth", "zeroHeight", "hidden", "invisible", "transparent")],
            {"links": ["#section"], "location": "https://locale.example/learn/#section", "pass": False, "clicks": 1},
            {"links": ["#section"], "throws": True, "clicks": 1},
        ])

    def test_no_anchor_and_javascript_evaluation_errors_fail_closed(self):
        chrome = mock.Mock()
        chrome.evaluate.return_value = False
        with self.assertRaisesRegex(CORE.BrowserFailure, "Learn navigation did not reach a real section"):
            PREVIEW.check_learn_navigation(chrome)
        # Exercise Chrome.evaluate itself: an evaluation exception is never a
        # boolean success and must retain its original diagnostic.
        chrome = object.__new__(CORE.Chrome)
        chrome.current_route = "/fixture/"
        chrome.call = mock.Mock(return_value={"exceptionDetails": {"text": "Uncaught"},
                                             "result": {"description": "genuine evaluation failure"}})
        with self.assertRaisesRegex(CORE.BrowserFailure, "genuine evaluation failure"):
            PREVIEW.check_learn_navigation(chrome)

    def test_navigation_check_is_required_in_both_viewports(self):
        fixture = PreviewSiteV2Test()
        page = fixture.page()
        page["html"] = page["html"].replace('<main>', '<main class="LearnNav">')
        snapshot = {"path": "/learn/", "origin": fixture.base.rstrip('/'), "text": "x"*30,
                    "width": 100, "overflow": 0, "images": True, "anchors": True}
        for failure in (None, 0, 1):
            chrome = mock.Mock(events=[])
            chrome.evaluate.side_effect = [page["html"], snapshot, page["html"], snapshot, True, True, True, True]
            outcomes = [None, None]
            if failure is not None: outcomes[failure] = CORE.BrowserFailure("Learn navigation failed")
            with self.subTest(failure=failure), mock.patch.object(PREVIEW, "settle_site_page"), \
                    mock.patch.object(PREVIEW, "settle_site_images"), \
                    mock.patch.object(PREVIEW, "check_learn_navigation", side_effect=outcomes) as check:
                if failure is None:
                    PREVIEW.site_browser_acceptance(chrome, fixture.base, fixture.profile, {"pages": [page]})
                else:
                    with self.assertRaisesRegex(CORE.BrowserFailure, "Learn navigation failed"):
                        PREVIEW.site_browser_acceptance(chrome, fixture.base, fixture.profile, {"pages": [page]})
                self.assertEqual(check.call_count, 1 if failure == 0 else 2)

class PreviewImageAcceptanceTest(unittest.TestCase):
    base = "http://127.0.0.1:3999/"

    def image(self, **changes):
        return dict(index=0, src=self.base+"image.svg", currentSrc=self.base+"image.svg",
                    loading="eager", complete=True, naturalWidth=20, naturalHeight=10, **changes)

    def test_loaded_eager_image_does_not_scroll_into_view(self):
        chrome = mock.Mock(events=[], current_route="/")
        chrome.evaluate.side_effect = [[0, 50], [self.image()], True]
        with mock.patch.object(PREVIEW, "settle_site_page") as settle:
            PREVIEW.settle_site_images(chrome, self.base)
        self.assertFalse(any("scrollIntoView" in c.args[0] for c in chrome.evaluate.call_args_list))
        self.assertIn("top:50", chrome.evaluate.call_args.args[0])
        settle.assert_called_once_with(chrome, self.base)

    def test_visible_complete_broken_image_fails(self):
        for field, value in (("naturalWidth", 0), ("naturalHeight", 0), ("currentSrc", "")):
            image = self.image(); image[field] = value
            chrome = mock.Mock(events=[], current_route="/")
            chrome.evaluate.side_effect = [[0, 0], [image], True]
            with self.subTest(field=field), self.assertRaisesRegex(CORE.BrowserFailure, "nonzero natural size"):
                PREVIEW.settle_site_images(chrome, self.base)

    def test_observed_lazy_image_cannot_remain_unrequested_forever(self):
        image = self.image(); image.update(loading="lazy", complete=False, currentSrc="", naturalWidth=0, naturalHeight=0)
        chrome = mock.Mock(events=[], current_route="/")
        chrome.evaluate.side_effect = [[0, 0], [image], True, [image], True]
        with self.assertRaisesRegex(CORE.BrowserFailure, "observed image loads within timeout"):
            PREVIEW.settle_site_images(chrome, self.base, timeout=0)
        self.assertIn("scrollIntoView", chrome.evaluate.call_args_list[2].args[0])

    @unittest.skipUnless(os.environ.get("GO_TOUR_RUN_BROWSER_TESTS") == "1", "explicit real Chrome image regression")
    def test_real_lazy_and_eager_images_in_both_viewports(self):
        # A far offscreen lazy image is deliberately unrequested initially.
        # Observation must trigger its real load, and still reject 404, invalid
        # image bytes (HTTP 200), and connection failure after observation.
        class Handler(BaseHTTPRequestHandler):
            def log_message(self, *args): pass
            def do_GET(self):
                if self.path == "/favicon.ico":
                    self.send_response(204); self.end_headers(); return
                if self.path.startswith("/page?"):
                    mode = self.path.split("?", 1)[1]
                    loading = "eager" if mode == "eager" else "lazy"
                    body = (f'<main><h1>Deterministic image acceptance fixture</h1><div style="height:20000px"></div>'
                            f'<img loading="{loading}" width="40" height="30" src="/{mode}.svg"></main>').encode()
                    status, kind = 200, "text/html"
                elif self.path == "/network.svg":
                    self.connection.close(); return
                elif self.path == "/broken.svg":
                    status, kind, body = 404, "text/plain", b"missing image"
                elif self.path == "/invalid.svg":
                    status, kind, body = 200, "image/svg+xml", b"invalid image bytes"
                else:
                    status, kind, body = 200, "image/svg+xml", b'<svg xmlns="http://www.w3.org/2000/svg" width="40" height="30"><rect width="40" height="30"/></svg>'
                self.send_response(status); self.send_header("Content-Type", kind)
                self.send_header("Content-Length", str(len(body))); self.end_headers(); self.wfile.write(body)

        server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        worker = threading.Thread(target=server.serve_forever, daemon=True); worker.start()
        base = f"http://127.0.0.1:{server.server_port}/"
        try:
            for width, height in ((1280, 800), (375, 812)):
                for mode in ("lazy", "eager", "broken", "invalid", "network"):
                    with self.subTest(width=width, mode=mode):
                        chrome = CORE.Chrome()
                        try:
                            chrome.navigate(base+"page?"+mode, width, height)
                            PREVIEW.settle_site_page(chrome, base)
                            before = chrome.evaluate("(() => {const i=document.querySelector('img');return {complete:i.complete,width:i.naturalWidth,top:i.getBoundingClientRect().top}})()")
                            if mode != "eager":
                                self.assertFalse(before["complete"])
                                self.assertEqual(before["width"], 0)
                                self.assertGreater(before["top"], height)
                                self.assertFalse(any(r["url"] == base+mode+".svg" for r in chrome.network_requests()))
                            if mode in ("lazy", "eager"):
                                PREVIEW.settle_site_images(chrome, base)
                                self.assertTrue(chrome.evaluate("(() => {const i=document.querySelector('img');return i.complete&&i.naturalWidth===40&&i.naturalHeight===30&&scrollY===0})()"))
                            else:
                                with self.assertRaises(CORE.BrowserFailure): PREVIEW.settle_site_images(chrome, base)
                            self.assertTrue(any(r["url"] == base+mode+".svg" for r in chrome.network_requests()))
                        finally: chrome.close()
        finally:
            server.shutdown(); worker.join(); server.server_close()

if __name__ == "__main__": unittest.main()
