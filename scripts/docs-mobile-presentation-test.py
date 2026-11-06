#!/usr/bin/env python3
"""Focused real-Chrome regression for the project-owned Docs mobile CSS."""

import functools
import importlib.util
import pathlib
import threading
import unittest
from http.server import SimpleHTTPRequestHandler, ThreadingHTTPServer

ROOT = pathlib.Path(__file__).resolve().parent.parent
spec = importlib.util.spec_from_file_location("docs_browser", ROOT / "scripts/browser_acceptance.py")
CORE = importlib.util.module_from_spec(spec); spec.loader.exec_module(CORE)
CSS = (ROOT / "internal/site/docs_mobile.css").read_text()
TOKEN = 'example.com/mymodules/module1/package1'
CODE = 'import "example.com/mymodule/v2/package1"\n\t' + 'exact_preformatted_line_' * 12
CELLS = ''.join('<td class="DocTable-cell">Column-%d</td>' % i for i in range(18))
BODY = '''<main><article class="Doc Article">
<ol class="SiteBreadcrumb">
<li class="BreadcrumbNav-li"><a class="BreadcrumbNav-link" href="https://go.dev/doc">Documentation</a></li>
<li class="BreadcrumbNav-li"><a class="BreadcrumbNav-link" href="/doc/security/">Security</a></li>
<li class="BreadcrumbNav-li"><a class="BreadcrumbNav-link" href="/doc/security/fuzz/">Go fuzzing</a></li>
<li class="BreadcrumbNav-li"><a class="BreadcrumbNav-link" aria-current="page" href="/doc/security/fuzz/technical">Go fuzzing technical details</a></li>
</ol>
<h1>Responsive Docs presentation fixture</h1>
<p id="normal">Use <code>fmt.Println()</code> normally.</p>
<p>Import <code id="inline">import "example.com/mymodule/v2/package1"</code> here.</p>
<ul><li><ul><li>Nested module <code id="nested">TOKEN</code></li></ul></li></ul>
<h2 id="heading">Using golang.org/x/example/hello/reverse</h2>
<pre id="pre"><code>CODE</code></pre>
<table class="DocTable" id="table"><tbody><tr class="DocTable-row">CELLS</tr><tr class="DocTable-row">CELLS</tr></tbody></table>
</article></main>'''.replace('TOKEN', TOKEN).replace('CODE', CODE).replace('CELLS', CELLS)


class Handler(SimpleHTTPRequestHandler):
    def log_message(self, *args): pass

    def do_GET(self):
        if self.path == '/favicon.ico':
            self.send_response(204); self.end_headers(); return
        if self.path.startswith('/fixture'):
            style = '' if self.path.endswith('?baseline') else '<style id="docs-mobile-presentation">' + CSS + '</style>'
            body = ('<!doctype html><html><head><meta name="viewport" content="width=device-width, initial-scale=1">'
                    '<link rel="stylesheet" href="/css/styles.css">' + style + '</head><body>' + BODY + '</body></html>').encode()
            self.send_response(200); self.send_header('Content-Type', 'text/html; charset=utf-8')
            self.send_header('Content-Length', str(len(body))); self.end_headers(); self.wfile.write(body)
            return
        super().do_GET()


class DocsMobilePresentationTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.server = ThreadingHTTPServer(('127.0.0.1', 0), functools.partial(Handler, directory=str(ROOT / '_content')))
        cls.worker = threading.Thread(target=cls.server.serve_forever, daemon=True); cls.worker.start()
        cls.base = 'http://127.0.0.1:%d' % cls.server.server_port

    @classmethod
    def tearDownClass(cls):
        cls.server.shutdown(); cls.server.server_close(); cls.worker.join()

    def setUp(self):
        self.chrome = CORE.Chrome()

    def tearDown(self):
        self.chrome.close()

    def navigate(self, width, baseline=False):
        self.chrome.navigate(self.base + '/fixture' + ('?baseline' if baseline else ''), width, 812)
        def loaded():
            CORE.assert_true(self.chrome.evaluate("document.readyState==='complete'"), 'fixture stylesheet not loaded')
        CORE.wait_for_condition(loaded, 'Docs fixture loaded')
        self.chrome.evaluate('(async()=>{await document.fonts.ready;return true})()', await_promise=True)

    def test_mobile_breadcrumb_and_inline_heading_list_tokens(self):
        self.navigate(375)
        state = self.chrome.evaluate('''(() => {
          const nav=document.querySelector('.SiteBreadcrumb');
          const fits=n=>{const r=n.getBoundingClientRect();return r.left>=0&&r.right<=375};
          const lines=n=>{const r=document.createRange();r.selectNodeContents(n);return [...r.getClientRects()].filter(r=>r.width>0&&r.height>0)};
          return {overflow:document.documentElement.scrollWidth-document.documentElement.clientWidth,
            breadcrumb:{client:nav.clientWidth,scroll:nav.scrollWidth,links:[...nav.querySelectorAll('a')].map(a=>({text:a.textContent,href:a.getAttribute('href'),current:a.getAttribute('aria-current'),fits:fits(a)}))},
            tokens:['inline','nested','heading'].map(id=>({id,lines:lines(document.getElementById(id)).map(r=>({left:r.left,right:r.right,top:r.top}))})),
            normal:getComputedStyle(document.querySelector('#normal code')).wordSpacing};})()''')
        self.assertEqual(state['overflow'], 0, state)
        self.assertLessEqual(state['breadcrumb']['scroll'], state['breadcrumb']['client'], state)
        self.assertEqual([x['href'] for x in state['breadcrumb']['links']],
                         ['https://go.dev/doc', '/doc/security/', '/doc/security/fuzz/', '/doc/security/fuzz/technical'])
        self.assertEqual([x['text'] for x in state['breadcrumb']['links']],
                         ['Documentation', 'Security', 'Go fuzzing', 'Go fuzzing technical details'])
        self.assertEqual(state['breadcrumb']['links'][-1]['current'], 'page')
        self.assertTrue(all(x['fits'] for x in state['breadcrumb']['links']), state)
        for token in state['tokens']:
            self.assertGreater(len({r['top'] for r in token['lines']}), 1, token)
            self.assertTrue(all(r['left']>=0 and r['right']<=375 for r in token['lines']), token)
        baseline = self.chrome.evaluate("(()=>{document.getElementById('docs-mobile-presentation').remove();return getComputedStyle(document.querySelector('#normal code')).wordSpacing})()")
        self.assertEqual(state['normal'], baseline)

    def test_preformatted_and_table_keep_usable_local_scroll(self):
        self.navigate(375)
        state = self.chrome.evaluate('''(() => {
          const pre=document.getElementById('pre'),table=document.getElementById('table'),code=pre.querySelector('code');
          const snapshot=n=>({client:n.clientWidth,scroll:n.scrollWidth,overflow:getComputedStyle(n).overflowX});
          const result={overflow:document.documentElement.scrollWidth-document.documentElement.clientWidth,
            pre:snapshot(pre),table:snapshot(table),text:code.textContent,whiteSpace:getComputedStyle(code).whiteSpace,
            wrap:getComputedStyle(code).overflowWrap,cells:table.querySelectorAll('td').length};
          pre.scrollLeft=50;table.scrollLeft=50;
          return {...result,preScroll:pre.scrollLeft,tableScroll:table.scrollLeft};})()''')
        self.assertEqual(state['overflow'], 0, state)
        self.assertEqual(state['text'], CODE)
        self.assertEqual(state['whiteSpace'], 'pre')
        self.assertEqual(state['wrap'], 'normal')
        self.assertEqual(state['cells'], 36)
        for name in ('pre', 'table'):
            self.assertEqual(state[name]['overflow'], 'auto', state)
            self.assertGreater(state[name]['scroll'], state[name]['client'], state)
            self.assertGreater(state[name+'Scroll'], 0, state)

    def test_desktop_styles_and_geometry_are_unchanged(self):
        expression = '''[...document.querySelectorAll('article,article *')].map(n=>{
          const s=getComputedStyle(n),r=n.getBoundingClientRect();return {tag:n.tagName,id:n.id,
            text:n.textContent,href:n.getAttribute('href'),rect:[r.x,r.y,r.width,r.height],
            display:s.display,whiteSpace:s.whiteSpace,wrap:s.overflowWrap,overflow:s.overflowX,
            font:s.font,wordSpacing:s.wordSpacing,margin:s.margin,padding:s.padding};})'''
        self.navigate(1280, baseline=True)
        baseline = self.chrome.evaluate(expression)
        self.navigate(1280)
        self.assertEqual(self.chrome.evaluate(expression), baseline)


if __name__ == '__main__':
    unittest.main()
