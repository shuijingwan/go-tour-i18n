// Copyright 2026 The go-tour-i18n Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tour

import (
	"html"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCourseAdLayoutProtectionInBrowser(t *testing.T) {
	if os.Getenv("GO_TOUR_RUN_BROWSER_TESTS") != "1" {
		t.Skip("set GO_TOUR_RUN_BROWSER_TESTS=1 to run the Chrome integration test")
	}
	chrome, err := exec.LookPath("google-chrome")
	if err != nil {
		t.Skip("google-chrome is not installed")
	}
	adScript, err := fs.ReadFile(contentTour, "tour/static/go-dev/course-ad.js")
	if err != nil {
		t.Fatal(err)
	}

	const editor = `<div id="editor-container" style="height:auto!important;min-height:0px!important;color:red">
<div class="relative-content" style="height:auto!important;min-height:0px!important;color:red">
<div id="left-side" style="height:auto!important;min-height:0px!important;color:red">
<div class="relative-content" style="height:auto!important;min-height:0px!important;color:red">
<div class="bar module-bar"></div><div class="go-dev-course-ad" data-go-dev-course-ad></div>
</div></div></div></div>`

	document := `<!doctype html><html><body>` + editor + `<script>` + string(adScript) + `</script><script>
(async function() {
    function assert(condition, message) {
        if (!condition) throw new Error(message);
    }
    function tick() {
        return new Promise(function(resolve) { setTimeout(resolve, 0); });
    }
    function nodes(editor) {
        var editorContent = editor.children[0];
        var leftSide = editorContent.children[0];
        return [editor, editorContent, leftSide, leftSide.children[0]];
    }
    function newEditor() {
        var wrapper = document.createElement('div');
        wrapper.innerHTML = ` + "`" + editor + "`" + `;
        return wrapper.firstElementChild;
    }

    try {
        var firstEditor = document.querySelector('#editor-container');
        var firstNodes = nodes(firstEditor);
        assert(window.adsbygoogle.length === 1, 'initial ad request count');
        assert(firstEditor.querySelectorAll('ins.adsbygoogle').length === 1, 'initial ad count');
        firstNodes.forEach(function(node) {
            assert(node.style.getPropertyValue('height') === '', 'initial exact height was not removed');
            assert(node.style.getPropertyValue('min-height') === '', 'initial exact min-height was not removed');
            assert(node.style.getPropertyValue('color') === 'red', 'unrelated initial style was removed');
        });

        firstNodes.forEach(function(node) {
            node.style.setProperty('height', 'auto', 'important');
            node.style.setProperty('min-height', '0px', 'important');
            node.style.setProperty('width', '37px', 'important');
        });
        await tick();
        firstNodes.forEach(function(node) {
            assert(node.style.getPropertyValue('height') === '', 'observed exact height was not removed');
            assert(node.style.getPropertyValue('min-height') === '', 'observed exact min-height was not removed');
            assert(node.style.getPropertyValue('width') === '37px', 'unrelated observed style was removed');
        });

        firstNodes[0].style.setProperty('height', 'auto');
        firstNodes[1].style.setProperty('height', '100px', 'important');
        firstNodes[2].style.setProperty('min-height', '0px');
        firstNodes[3].style.setProperty('min-height', '1px', 'important');
        await tick();
        assert(firstNodes[0].style.getPropertyValue('height') === 'auto', 'non-important height was removed');
        assert(firstNodes[1].style.getPropertyValue('height') === '100px', 'non-exact height was removed');
        assert(firstNodes[2].style.getPropertyValue('min-height') === '0px', 'non-important min-height was removed');
        assert(firstNodes[3].style.getPropertyValue('min-height') === '1px', 'non-exact min-height was removed');

        var firstMount = firstEditor.querySelector('[data-go-dev-course-ad]');
        var firstLeftContent = firstNodes[3];
        firstLeftContent.removeChild(firstMount);
        firstLeftContent.appendChild(firstMount);
        await tick();
        assert(window.adsbygoogle.length === 1, 'filled mount requested twice');
        assert(firstMount.querySelectorAll('ins.adsbygoogle').length === 1, 'filled ad mounted twice');

        firstMount.querySelector('ins').setAttribute('data-ad-status', 'unfilled');
        firstLeftContent.removeChild(firstMount);
        firstLeftContent.appendChild(firstMount);
        await tick();
        assert(window.adsbygoogle.length === 1, 'unfilled mount requested twice');
        assert(firstMount.querySelectorAll('ins.adsbygoogle').length === 1, 'unfilled ad mounted twice');

        var secondEditor = newEditor();
        firstEditor.remove();
        document.body.appendChild(secondEditor);
        await tick();
        var secondNodes = nodes(secondEditor);
        assert(window.adsbygoogle.length === 2, 'SPA editor did not request exactly one new ad');
        assert(secondEditor.querySelectorAll('ins.adsbygoogle').length === 1, 'SPA editor ad mount count');
        secondNodes.forEach(function(node) {
            assert(node.style.getPropertyValue('height') === '', 'SPA initial exact height was not removed');
            assert(node.style.getPropertyValue('min-height') === '', 'SPA initial exact min-height was not removed');
            assert(node.style.getPropertyValue('color') === 'red', 'SPA unrelated initial style was removed');
        });

        firstNodes[0].style.setProperty('height', 'auto', 'important');
        secondNodes[0].style.setProperty('height', 'auto', 'important');
        secondNodes[3].style.setProperty('min-height', '0px', 'important');
        await tick();
        assert(firstNodes[0].style.getPropertyValue('height') === 'auto', 'old editor observer stayed active');
        assert(secondNodes[0].style.getPropertyValue('height') === '', 'new editor height was not protected');
        assert(secondNodes[3].style.getPropertyValue('min-height') === '', 'new editor min-height was not protected');

        document.body.setAttribute('data-course-ad-test', 'PASS');
    } catch (error) {
        document.body.setAttribute('data-course-ad-test', 'FAIL: ' + error.message);
    }
}());
</script></body></html>`

	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "course-ad-test.html")
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(chrome,
		"--headless=new",
		"--no-sandbox",
		"--disable-gpu",
		"--disable-dev-shm-usage",
		"--disable-breakpad",
		"--disable-crash-reporter",
		"--noerrdialogs",
		"--user-data-dir="+filepath.Join(tempDir, "chrome-profile"),
		"--run-all-compositor-stages-before-draw",
		"--virtual-time-budget=2000",
		"--dump-dom",
		"file://"+path,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("google-chrome: %v\n%s", err, output)
	}
	got := string(output)
	if !strings.Contains(got, `data-course-ad-test="PASS"`) {
		t.Fatalf("course ad browser test failed:\n%s", html.UnescapeString(got))
	}
}
