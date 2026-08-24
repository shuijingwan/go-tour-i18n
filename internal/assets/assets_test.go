package assets

import (
	"reflect"
	"testing"
)

func TestSharedPathsFirstVersionAllowlist(t *testing.T) {
	want := []string{
		"images/go-logo-white.svg",
		"images/icons/brightness_2_gm_grey_24dp.svg",
		"images/icons/brightness_6_gm_grey_24dp.svg",
		"images/icons/light_mode_gm_grey_24dp.svg",
		"images/site-logo-32.png",
		"images/site-logo.png",
		"tour/static/css/app.css",
		"tour/static/img/gopher.png",
		"tour/static/lib/codemirror/lib/codemirror.css",
	}
	if !reflect.DeepEqual(SharedPaths(), want) {
		t.Fatalf("SharedPaths = %v, want %v", SharedPaths(), want)
	}
}

func TestURL(t *testing.T) {
	const asset = "tour/static/css/app.css"
	for _, test := range []struct {
		name, locale string
		development  bool
		want         string
	}{
		{"zh development", "zh-CN", true, "/" + asset},
		{"zh production", "zh-CN", false, "/" + asset},
		{"ja preview", "ja-JP", true, "/" + asset},
		{"ja production", "ja-JP", false, BaseURL + "/" + asset},
		{"future production", "ko", false, BaseURL + "/" + asset},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := URL(test.locale, test.development, asset)
			if err != nil || got != test.want {
				t.Fatalf("URL() = %q, %v; want %q", got, err, test.want)
			}
		})
	}
	for _, invalid := range []string{"", ".", "../secret", "tour/../secret", "//host/path"} {
		if _, err := URL("ja-JP", false, invalid); err == nil {
			t.Errorf("URL(%q) accepted unsafe path", invalid)
		}
	}
	for _, excluded := range []string{"tour/script.js", "tour/static/partials/editor.html", "tour/static/img/tree.png"} {
		if _, err := URL("ja-JP", false, excluded); err == nil {
			t.Errorf("URL(%q) accepted excluded asset", excluded)
		}
	}
}
