package tour

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSharedChromeMetadataAndRegistry(t *testing.T) {
	for _, locale := range []string{"zh-CN", "fr-FR"} {
		origin, err := productionOriginForLocale(locale)
		if err != nil {
			t.Fatal(err)
		}
		m, err := NewSharedChromeModel(locale, origin)
		if err != nil {
			t.Fatal(err)
		}
		if m.Home != origin+"/" || m.GitHub != Project.GitHubURL || m.Copyright != "© 2026 "+Project.CopyrightHolder {
			t.Fatal("metadata binding", m)
		}
		if (m.ICP != "") != (locale == "zh-CN") {
			t.Fatal("locale legal metadata leaked", m)
		}
		if len(m.Languages) != len(languageRegistry) {
			t.Fatal("registry exact set")
		}
		current := 0
		for i, l := range m.Languages {
			want := languageRegistry[i]
			label := want.EnglishName
			if want.Autonym != want.EnglishName {
				label += " — " + want.Autonym
			}
			if l.Locale != want.Locale || l.URL != want.URL || l.Label != label || l.Official != want.Official || l.Current != (l.Locale == locale) {
				t.Fatal("registry projection", l, want)
			}
			if l.Current {
				current++
			}
		}
		if current != 1 {
			t.Fatal("current locale count", current)
		}
		if locale == "zh-CN" && (m.About != "关于此项目" || m.Unofficial != "永夜 · 非官方社区多语言翻译项目" || m.ICP != "蜀ICP备13001590号-1" || m.ICPURL != "https://beian.miit.gov.cn/") {
			t.Fatal("zh-CN authority", m)
		}
		if locale == "fr-FR" && strings.Contains(m.Home, "https://go-dev.") {
			t.Fatal("locale production root leaked")
		}
	}
	for _, origin := range []string{"http://localhost/", "https://user@example.org/", "https://example.org/?x=1"} {
		if _, err := NewSharedChromeModel("zh-CN", origin); err == nil {
			t.Fatal("bad identity accepted", origin)
		}
	}
	if _, err := NewSharedChromeModel("xx-invalid", "https://example.org"); err == nil {
		t.Fatal("missing UI catalog accepted")
	}
}

func TestSharedChromeRealTourRenderer(t *testing.T) {
	h := WithSharedChrome(productionTestHandler(t, "https://example.invalid"), "zh-CN", "https://go-dev.shuijingwanwq.com")
	for _, route := range []string{"/tour/welcome/1", "/tour/list", "/tour/basics/11"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", route, nil))
		if w.Code != 200 {
			t.Fatal(route, w.Code, w.Body.String())
		}
		body := w.Body.Bytes()
		for _, want := range []string{`data-project-nav="zh-CN"`, `data-project-footer="zh-CN"`, `data-project-home="" href="https://go-dev.shuijingwanwq.com/"`, `data-project-github="" href="` + Project.GitHubURL + `"`, `© 2026 永夜`, Project.ICPNumber} {
			if !bytes.Contains(body, []byte(want)) {
				t.Fatalf("%s missing %s", route, want)
			}
		}
		if bytes.Count(body, []byte(`data-project-footer="`)) != 1 || bytes.Count(body, []byte(`data-project-nav="`)) != 1 || bytes.Contains(body, []byte(`class="header-project-nav"`)) {
			t.Fatal("duplicate project chrome", route)
		}
		if bytes.Count(body, []byte(`data-locale="`)) != len(languageRegistry) || bytes.Count(body, []byte(`aria-current="page"`)) != 1 {
			t.Fatal("selector exact set", route)
		}
	}
}

func TestSharedChromeRejectsMalformedShell(t *testing.T) {
	if _, err := ApplySharedChrome([]byte("<main>Missing shell</main>"), "zh-CN", "https://example.org"); err == nil {
		t.Fatal("missing shell accepted")
	}
	input := []byte(`<html><head></head><body><header class="Site-header"><div class="Header-rightContent"></div></header><main>Body</main><footer class="Site-footer"></footer></body></html>`)
	if _, err := ApplySharedChrome(input, "zh-CN", "https://example.org"); err == nil {
		t.Fatal("malformed upstream footer accepted")
	}
}

func TestSharedChromeHostPresentation(t *testing.T) {
	for _, site := range []bool{false, true} {
		header, footer, host := `<div class="top-bar"><div class="right"></div></div>`, `<footer class="site-footer"></footer>`, "tour"
		if site {
			header = `<header class="Site-header"><div class="Header-rightContent"></div></header>`
			footer = `<footer class="Site-footer"><div class="Footer-links"></div><div class="Footer-bottom"></div></footer>`
			host = "site"
		}
		out, err := ApplySharedChrome([]byte(`<html><head></head><body>`+header+`<main>Fixture</main>`+footer+`</body></html>`), "zh-CN", "https://go-dev.shuijingwanwq.com")
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`href="/translation/"`, `data-project-host="` + host + `"`, `data-project-current-language=""`} {
			if !bytes.Contains(out, []byte(want)) {
				t.Fatal(host, want)
			}
		}
		if site && !bytes.Contains(out, []byte(`class="Container Container--padded"`)) {
			t.Fatal("Site footer container alignment")
		}
		if strings.Contains(string(out), "ProjectChromeDrawer") {
			t.Fatal("Tour drawer presentation recreated")
		}
	}
	for _, want := range []string{"white-space:nowrap", "text-align:left", ".ProjectFooter[data-project-host=site]{text-align:left}", ".ProjectFooter[data-project-host=tour]{text-align:center}"} {
		if !strings.Contains(sharedChromeCSS, want) {
			t.Fatal(want)
		}
	}
	if strings.Contains(sharedChromeCSS, "text-decoration:underline") {
		t.Fatal("shared footer forces underline")
	}
}
