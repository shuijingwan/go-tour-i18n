#!/usr/bin/env python3
"""Focused real-Chrome test of shared chrome plus the frozen Site drawer."""
import importlib.util
import json
import os
import pathlib
import subprocess
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

ROOT = pathlib.Path(__file__).resolve().parent.parent
SPEC = importlib.util.spec_from_file_location("keyboard_preview", ROOT / "scripts/verify-preview-browser.py")
PREVIEW = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(PREVIEW)
CORE = PREVIEW.CORE

STATE = """(() => {
  const d=document.querySelector('[data-project-languages]'),s=d.querySelector('summary'),
    drawer=document.querySelector('.NavigationDrawer.js-header,.ProjectChromeDrawer'),
    toggle=document.querySelector('.js-headerMenuButton,.ProjectChromeToggle');
  return {open:d.open,summary:document.activeElement===s,drawer:!!drawer&&drawer.classList.contains('is-active'),
    toggle:document.activeElement===toggle,expanded:toggle?.getAttribute('aria-expanded')||null,
    link:document.activeElement.matches('[data-project-languages] a[href]')};
})()"""


@unittest.skipUnless(os.environ.get("GO_TOUR_RUN_BROWSER_TESTS") == "1", "real Chrome opt-in")
class SharedChromeKeyboardBrowserTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.files = tempfile.TemporaryDirectory(prefix="shared-chrome-keyboard-")
        env = dict(os.environ, GO_TOUR_SHARED_CHROME_FIXTURE_DIR=cls.files.name)
        result = subprocess.run(["go", "test", "-mod=readonly", "./internal/tour", "-run",
                                 "^TestSharedChromeKeyboardFixture$", "-count=1"],
                                cwd=ROOT, env=env, capture_output=True, text=True, timeout=120)
        if result.returncode:
            cls.files.cleanup()
            raise RuntimeError(result.stdout + result.stderr)
        env["GO_TOUR_PROJECT_PRESENTATION_FIXTURE_DIR"] = cls.files.name
        project = subprocess.run(["go", "test", "-mod=readonly", "./internal/site", "-run",
                                  "^TestTranslationProjectPresentation$", "-count=1"],
                                 cwd=ROOT, env=env, capture_output=True, text=True, timeout=120)
        if project.returncode:
            cls.files.cleanup()
            raise RuntimeError(project.stdout + project.stderr)
        directory = pathlib.Path(cls.files.name)

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self):
                if self.path in ("/site.html", "/tour.html", "/project.html"):
                    body = (directory / self.path[1:]).read_bytes()
                    content_type = "text/html; charset=utf-8"
                elif self.path in ("/site.css", "/tour.css"):
                    body = (directory / self.path[1:]).read_bytes()
                    content_type = "text/css"
                elif self.path == "/dl/?mode=json":
                    body = json.dumps([{"version": "go1.26.0", "stable": True, "files": []}]).encode()
                    content_type = "application/json"
                elif self.path == "/favicon.ico":
                    self.send_response(204); self.end_headers(); return
                else:
                    self.send_error(404); return
                self.send_response(200)
                self.send_header("Content-Type", content_type)
                self.send_header("Content-Length", str(len(body)))
                self.end_headers(); self.wfile.write(body)

            def log_message(self, *args):
                pass

        cls.server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.base = "http://127.0.0.1:%d" % cls.server.server_port
        cls.chrome = CORE.Chrome()

    @classmethod
    def tearDownClass(cls):
        cls.chrome.close()
        cls.server.shutdown(); cls.server.server_close(); cls.thread.join()
        cls.files.cleanup()

    def key(self, key, phase=None, **extra):
        for kind in ([phase] if phase else ["keyDown", "keyUp"]):
            self.chrome.call("Input.dispatchKeyEvent", dict(type=kind, key=key, code=key,
                             windowsVirtualKeyCode={"Escape": 27, "Tab": 9}[key], **extra))

    def open_menu(self, mobile):
        if mobile:
            self.chrome.evaluate("document.querySelector('.js-headerMenuButton,.ProjectChromeToggle').click();true")
        self.chrome.evaluate("const s=document.querySelector('[data-project-languages] summary');s.focus();s.click();true")
        self.assertTrue(self.chrome.evaluate(STATE)["open"])
        self.key("Tab")
        self.assertTrue(self.chrome.evaluate(STATE)["link"], "Tab must reach an actual language link")

    def test_real_escape_cycle_first_middle_last_both_shells_and_viewports(self):
        for shell in ("site", "tour"):
            for width in (1280, 375):
                for position in ("first", "middle", "last"):
                    with self.subTest(shell=shell, width=width, position=position):
                        self.chrome.navigate(self.base + "/" + shell + ".html", width, 812)
                        mobile = width == 375 and shell == "site"
                        if shell == "tour":
                            self.assertTrue(self.chrome.evaluate("!document.querySelector('.ProjectChromeToggle,.ProjectChromeDrawer')&&!!document.querySelector('.top-bar [data-project-nav]')"))
                        self.open_menu(mobile)
                        self.chrome.evaluate("""(() => {const a=[...document.querySelectorAll('[data-project-languages] a[href]')];
                            a[%s].focus();return true})()""" % {"first": "0", "middle": "Math.floor(a.length/2)", "last": "a.length-1"}[position])
                        for phase in ("keyDown", "keyUp"):
                            self.key("Escape", phase)
                            state = self.chrome.evaluate(STATE)
                            self.assertFalse(state["open"], (phase, state))
                            self.assertTrue(state["summary"], (phase, state))
                            self.assertEqual(state["drawer"], mobile, (phase, state))
                            self.assertEqual(state["expanded"], str(mobile).lower() if shell == "site" else None)
                        if mobile:
                            self.key("Escape")
                            state = self.chrome.evaluate(STATE)
                            self.assertFalse(state["drawer"], state)
                            self.assertTrue(state["toggle"], state)
                            self.assertEqual(state["expanded"], "false")
                        PREVIEW.assert_browser_events(self.chrome, self.base)

    def test_outside_focus_unrelated_escape_and_ordinary_tab(self):
        for shell in ("site", "tour"):
            for width in (1280, 375):
                with self.subTest(shell=shell, width=width):
                    self.chrome.navigate(self.base + "/" + shell + ".html", width, 812)
                    self.open_menu(width == 375 and shell == "site")
                    self.chrome.evaluate("document.querySelector('[data-outside]').focus();true")
                    self.key("Escape")
                    state = self.chrome.evaluate(STATE)
                    self.assertFalse(state["open"])
                    self.assertTrue(state["summary"])
                    if width == 375 and shell == "site":
                        self.key("Escape")
                    self.chrome.evaluate("""document.querySelector('[data-outside]').focus();
                        window.escapePhases=[];window.addEventListener('keydown',e=>{if(e.key==='Escape')escapePhases.push('down')});
                        window.addEventListener('keyup',e=>{if(e.key==='Escape')escapePhases.push('up')});true""")
                    self.key("Escape")
                    self.assertEqual(self.chrome.evaluate("escapePhases"), ["down", "up"])
                    self.assertTrue(self.chrome.evaluate("document.activeElement.matches('[data-outside]')"))
                    self.key("Tab")
                    self.assertTrue(self.chrome.evaluate("document.activeElement.matches('[data-next]')"))
                    PREVIEW.assert_browser_events(self.chrome, self.base)

    def test_host_native_presentation_and_language_rows(self):
        for shell in ("site", "tour"):
            for width in (1280, 375):
                with self.subTest(shell=shell, width=width):
                    self.chrome.navigate(self.base + "/" + shell + ".html", width, 812)
                    self.open_menu(width == 375 and shell == "site")
                    result = self.chrome.evaluate("""(() => {
                      const nav=document.querySelector('[data-project-nav]'),f=document.querySelector('[data-project-footer]'),
                        about=nav.querySelector('[data-project-about]'),summary=nav.querySelector('summary'),list=nav.querySelector('ul'),
                        reference=document.querySelector('.Header-menuItem>a'),style=n=>getComputedStyle(n),r=list.getBoundingClientRect();
                      return {about:new URL(about.href).pathname,
                        singleLine:[...list.querySelectorAll('li>a,li>span')].every(n=>style(n).whiteSpace==='nowrap'),
                        left:[...list.querySelectorAll('li>a,li>span')].every(n=>style(n).textAlign==='left'),
                        withinViewport:r.left>=0&&r.right<=innerWidth,
                        footerAlign:style(f).textAlign,underline:style(f.querySelector('a')).textDecorationLine,
                        headerNative:!reference||[about,summary].every(n=>style(n).fontSize===style(reference).fontSize&&style(n).lineHeight===style(reference).lineHeight),
                        noTourDrawer:!document.querySelector('.ProjectChromeToggle,.ProjectChromeDrawer')};
                    })()""")
                    self.assertEqual(result["about"], "/translation/")
                    self.assertTrue(result["singleLine"], result)
                    self.assertTrue(result["left"], result)
                    self.assertTrue(result["withinViewport"], result)
                    self.assertEqual(result["footerAlign"], "left" if shell == "site" else "center")
                    self.assertEqual(result["underline"], "none")
                    if shell == "site" and width == 1280:
                        self.assertTrue(result["headerNative"], result)
                    if shell == "tour":
                        self.assertTrue(result["noTourDrawer"], result)
                    PREVIEW.assert_browser_events(self.chrome, self.base)

    def test_held_escape_does_not_dismiss_second_layer(self):
        for shell in ("site", "tour"):
            with self.subTest(shell=shell):
                self.chrome.navigate(self.base + "/" + shell + ".html", 375, 812)
                self.open_menu(shell == "site")
                self.key("Escape", "keyDown")
                self.key("Escape", "keyDown", autoRepeat=True)
                self.key("Escape", "keyUp")
                state = self.chrome.evaluate(STATE)
                self.assertFalse(state["open"])
                self.assertTrue(state["summary"])
                self.assertEqual(state["drawer"], shell == "site")
                PREVIEW.assert_browser_events(self.chrome, self.base)

    def test_tour_mobile_globe_and_desktop_language_trigger(self):
        for shell in ("tour", "site"):
            for width in (375, 1280):
                with self.subTest(shell=shell, width=width):
                    self.chrome.navigate(self.base + "/" + shell + ".html", width, 812)
                    if shell == "site" and width == 375:
                        self.chrome.evaluate("document.querySelector('.js-headerMenuButton').click();true")
                    state = self.chrome.evaluate("""(() => {
                      const nav=document.querySelector('[data-project-nav]'), s=nav.querySelector('summary'),
                        label=s.querySelector('[data-project-current-language]'),
                        arrow=s.querySelector('[data-project-language-arrow]'), r=s.getBoundingClientRect();
                      return {label:getComputedStyle(label).display,arrow:getComputedStyle(arrow).display,
                        globe:s.textContent.includes('🌐'),name:s.getAttribute('aria-label'),
                        count:nav.querySelectorAll('details').length,x:r.x+r.width/2,y:r.y+r.height/2};
                    })()""")
                    mobile_tour = shell == "tour" and width == 375
                    self.assertEqual(state["label"] == "none", mobile_tour, state)
                    self.assertEqual(state["arrow"] == "none", mobile_tour, state)
                    self.assertTrue(state["globe"] and state["name"], state)
                    self.assertEqual(state["count"], 1)
                    self.chrome.call("Input.dispatchMouseEvent", dict(type="mousePressed", x=state["x"],
                                     y=state["y"], button="left", clickCount=1))
                    self.chrome.call("Input.dispatchMouseEvent", dict(type="mouseReleased", x=state["x"],
                                     y=state["y"], button="left", clickCount=1))
                    self.assertTrue(self.chrome.evaluate(STATE)["open"])
                    self.key("Escape")
                    self.assertTrue(self.chrome.evaluate(STATE)["summary"])
                    self.assertFalse(self.chrome.evaluate(STATE)["open"])
                    PREVIEW.assert_browser_events(self.chrome, self.base)

    def test_project_body_sections_and_responsive_directory(self):
        for width in (375, 1280):
            with self.subTest(width=width):
                self.chrome.navigate(self.base + "/project.html", width, 812)
                result = self.chrome.evaluate("""(() => {
                  const page=document.querySelector('.ProjectPage');
                  return {sections:page.querySelectorAll('[data-project-section]').length,
                    headers:page.querySelectorAll('h1').length,cards:page.querySelectorAll('.ProjectCard').length,
                    overflow:document.documentElement.scrollWidth-innerWidth,
                    uncovered:!!page.querySelector('a[href="/doc/modules/"]'),
                    child:!!page.querySelector('a[href="/doc/modules/developing"]'),
                    footer:document.querySelectorAll('[data-project-footer]').length,
                    heroSize:parseFloat(getComputedStyle(page.querySelector('h1')).fontSize)};
                })()""")
                self.assertEqual(result["sections"], 9, result)
                self.assertEqual(result["headers"], 1, result)
                self.assertGreater(result["cards"], 0, result)
                self.assertLessEqual(result["overflow"], 1, result)
                self.assertFalse(result["uncovered"], result)
                self.assertTrue(result["child"], result)
                self.assertEqual(result["footer"], 1, result)
                self.assertGreaterEqual(result["heroSize"], 32, result)
                PREVIEW.assert_browser_events(self.chrome, self.base)


if __name__ == "__main__":
    unittest.main()
