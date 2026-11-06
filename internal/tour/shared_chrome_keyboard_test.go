package tour

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The real-Chrome regression consumes the actual shared renderer and the frozen
// Site drawer runtime, including its bubbling Escape keyup listener.
func TestSharedChromeKeyboardFixture(t *testing.T) {
	dir := os.Getenv("GO_TOUR_SHARED_CHROME_FIXTURE_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	siteJS, err := os.ReadFile("../../_content/js/site.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, shell := range []string{"site", "tour"} {
		cssPath := "../../_content/tour/static/css/app.css"
		if shell == "site" {
			cssPath = "../../_content/css/styles.css"
		}
		css, err := os.ReadFile(cssPath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, shell+".css"), css, 0600); err != nil {
			t.Fatal(err)
		}
		header := `<div class="bar top-bar"><div class="right"></div></div>`
		footer := `<footer class="site-footer"></footer>`
		runtime := ""
		if shell == "site" {
			header = `<header class="Site-header"><div class="Header-rightContent"><ul class="Header-menu"><li class="Header-menuItem"><a href="#fixture">Docs</a></li></ul><button class="js-headerMenuButton">Menu</button></div></header>
<aside class="NavigationDrawer js-header"><nav class="NavigationDrawer-nav"><div class="NavigationDrawer-header"><a href="#fixture">Go</a></div><ul class="NavigationDrawer-list"><li><a href="#fixture">Fixture</a></li></ul></nav></aside><div class="js-scrim"></div>`
			footer = `<footer class="Site-footer"><div class="Footer-links">Navigation</div><div class="Footer-bottom">Legal</div></footer>`
			runtime = `<script>document.cookie='cookie-consent=true';</script><script>` + string(siteJS) + `</script>`
		}
		input := `<html><head><link rel="stylesheet" href="/` + shell + `.css"><meta name="viewport" content="width=device-width, initial-scale=1"><style>.NavigationDrawer{display:none}.NavigationDrawer.is-active{display:block}</style></head><body>` + header +
			`<main id="fixture"><button data-outside>Outside</button><button data-next>Next</button></main>` + footer + runtime + `</body></html>`
		output, err := ApplySharedChrome([]byte(input), "zh-CN", "https://go-dev.shuijingwanwq.com")
		if err != nil {
			t.Fatal(shell, err)
		}
		if !strings.Contains(string(output), `id="project-chrome-runtime"`) {
			t.Fatal(shell, "shared runtime missing")
		}
		if err := os.WriteFile(filepath.Join(dir, shell+".html"), output, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
