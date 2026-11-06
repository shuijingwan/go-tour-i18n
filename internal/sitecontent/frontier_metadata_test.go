package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

func TestFrozenFrontierPageMetadata(t *testing.T) {
	for _, name := range []string{"_content/doc/index.html", "_content/doc/install.html"} {
		raw, err := readRegular(repositoryRoot(t), name)
		if err != nil {
			t.Fatal(err)
		}
		d, err := ParseWorkflowDocument(Source{Path: name, Kind: "page", Route: pageRoute(name), SHA256: digest(raw)}, raw, 1)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		a := PageAdapter{*d, raw}
		restored, err := a.Restore(d.Units[0].Source)
		if err != nil || !bytes.Equal(restored, raw) {
			t.Fatalf("%s round trip: %v", name, err)
		}
		if strings.Contains(d.Units[0].Source, `"Sidebar"`) {
			t.Fatal("machine metadata exposed as language")
		}
	}
	for _, raw := range []string{`<!--{"Title":"Page","Sidebar":true}--><p>Body.</p>`, `<!--{"Title":"Page","Unknown":"bug"}--><p>Body.</p>`} {
		if _, err := parseMetadata([]byte(raw), func(int, int, string) {}); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
}
