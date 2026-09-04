#!/usr/bin/env python3

import importlib.util
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
        for locale in ("zh-CN", "ja-JP", "de-DE", "fr-FR", "ko-KR", "es-ES"):
            with self.subTest(locale=locale):
                metadata = PREVIEW.formal_list_metadata(locale)
                self.assertTrue(all(metadata.values()))
                self.assertEqual(metadata, CORE.locale_list_metadata(locale))

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
        self.assertIn("browser_ad_gate(editor)", source)
        preview_body = source.split("def preview_acceptance", 1)[1]
        self.assertNotIn("browser_ad_gate", preview_body)
        self.assertIn("fetch('/socket')", preview_body)
        self.assertIn("canonical_origin + after", preview_body)

    def test_production_still_requires_https_formal_identity(self):
        original = sys.argv
        try:
            sys.argv = ["verify-production-browser.py", "http://127.0.0.1:38573/", "ko-KR"]
            self.assertEqual(PRODUCTION.main(), 1)
        finally: sys.argv = original

if __name__ == "__main__": unittest.main()
