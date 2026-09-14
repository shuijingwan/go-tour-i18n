#!/usr/bin/env python3

import contextlib
import importlib.util
import io
import pathlib
import sys
import unittest
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
        result = mock.Mock(returncode=0, stdout='{"locale":"example","publication":"go-local","tour_ads_enabled":false}\n', stderr="")
        with mock.patch.object(CORE.subprocess, "run", return_value=result) as run:
            policy = CORE.publication_policy("example")
        self.assertEqual(policy["publication"], "go-local")
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
                    mock.patch.object(PREVIEW.CORE, "preview_acceptance", side_effect=lambda *args: captured.append(args)):
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
        parsed = PREVIEW.validate_sitemap(ET.tostring(document), production, routes)
        self.assertEqual(parsed[2], production + reordered[0])
        self.assertEqual(set(parsed), set(urls))

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

    def test_production_still_requires_https_formal_identity(self):
        original = sys.argv
        try:
            sys.argv = ["verify-production-browser.py", "http://127.0.0.1:38573/", "ko-KR"]
            self.assertEqual(PRODUCTION.main(), 1)
        finally: sys.argv = original

if __name__ == "__main__": unittest.main()
