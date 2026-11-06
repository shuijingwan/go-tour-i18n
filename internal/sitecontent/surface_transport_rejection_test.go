package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestSurfaceTransportPredecessorAndCurrentGates(t *testing.T) {
	root, c, batch := surfaceCapabilityFixture(t)
	d, err := i18n.PrepareSurfaceRevisionTransport(root, c, "zh-CN", batch, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	r := surfaceCapabilityRequest(t, d, true)
	for _, name := range []string{"wrong-review", "wrong-reopen", "wrong-predecessor", "validation-attempt", "missing-current-review", "current-not-ready"} {
		t.Run(name, func(t *testing.T) {
			catalog := *c
			request := r
			path := d.Authority.Reopen.Path
			switch name {
			case "wrong-predecessor":
				path = "data/retranslation-runs/zh-CN/" + batch + "/manifest.json"
			case "validation-attempt":
				path = d.Authority.Bindings[0].Predecessor.ValidationPath
			case "missing-current-review":
				path = d.Authority.CurrentReview.Path
			}
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, before)
			var value map[string]any
			if err := json.Unmarshal(before, &value); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "wrong-review":
				value["surface_review_id"] = "other-review"
			case "wrong-reopen":
				value["reopen_id"] = "other-reopen"
			case "wrong-predecessor":
				value["units"].([]any)[0].(map[string]any)["previous_snapshot_id"] = "other-snapshot"
			case "validation-attempt":
				value["attempt"] = float64(99)
			case "missing-current-review":
				if err := os.Remove(filepath.Join(root, path)); err != nil {
					t.Fatal(err)
				}
			case "current-not-ready":
				catalog.Adapted = append([]i18n.AdaptedTranslationUnit(nil), c.Adapted...)
				b := catalog.Adapted[0]
				b.Adapter = e2eRejectingPage{b.Adapter.(PageAdapter)}
				catalog.Adapted[0] = b
				catalog.TranslationPackages = append([]i18n.TranslationPackageDescriptor(nil), c.TranslationPackages...)
				for i := range catalog.TranslationPackages {
					p := &catalog.TranslationPackages[i]
					p.Units = append([]i18n.AdaptedTranslationUnit(nil), p.Units...)
					for j := range p.Units {
						if p.Units[j].Unit.ID == b.Unit.ID {
							p.Units[j] = b
						}
					}
				}
			}
			if name != "missing-current-review" && name != "current-not-ready" {
				bad, _ := json.Marshal(value)
				writeFixture(t, root, path, append(bad, '\n'))
				if name == "wrong-predecessor" {
					request.ManifestSHA256 = digest(append(bad, '\n'))
				}
			}
			if _, err := i18n.PrepareSurfaceRevisionTransport(root, &catalog, "zh-CN", batch, &request, false); err == nil {
				t.Fatal("invalid predecessor/current authority accepted", name)
			}
			if _, err := os.Stat(filepath.Join(root, d.Path)); !os.IsNotExist(err) {
				t.Fatal("failed preflight recorded authority", err)
			}
		})
	}
	// No operation above may rewrite the batch or its selected candidate.
	b, err := readRegular(root, d.Authority.Bindings[0].Predecessor.CandidatePath)
	if err != nil || digest(b) != d.Authority.Bindings[0].Predecessor.CandidateSHA256 {
		t.Fatal("negative preflight changed candidate", err)
	}
}

func TestSurfaceInlineDescriptorSourceOwnership(t *testing.T) {
	for _, example := range []struct {
		source string
		want   int
	}{
		{"Read [`FormatDSN method`](https://example.test/api#Config.FormatDSN).", 1},
		{"Read [`sql.OpenDB function`](https://pkg.go.dev/database/sql#OpenDB).", 1},
		{"Read <code>log package</code>.", 1},
		{"Read `Unknown method`.", 0},
		{"Read [`Unknown method`](https://example.test/api#Other).", 0},
		{"Read `Unknown method` and [another link](https://example.test/api#Unknown).", 0},
		{"Read `literal value` and `identifier`.", 0},
		{"```go\n// Keep method/function comments.\nconst s = \"log package\"\n```\n\nRead this text.", 0},
	} {
		raw := []byte("# Source proof\n\n" + example.source + "\n")
		d, err := ParseWorkflowDocument(Source{Path: "_content/doc/source-proof.md", Kind: "page", SHA256: digest(raw), Route: "/doc/source-proof"}, raw, 1)
		if err != nil {
			t.Fatal(err)
		}
		a := PageAdapter{*d, raw}
		text, _, err := a.ProtectedInput(nil)
		if err != nil {
			t.Fatal(err)
		}
		p, _, err := a.SurfaceTransportProjection(a, []byte(text), nil)
		if err != nil {
			t.Fatal(err)
		}
		count := 0
		for _, r := range p.Available {
			if r.Kind == "inline-api-descriptor" {
				count++
				projected, codec, err := a.SurfaceTransportProjection(a, []byte(text), []string{r.ID})
				if err != nil {
					t.Fatal(err)
				}
				got, err := codec.Restore(projected.Template)
				if err != nil || !bytes.Equal(got, raw) {
					t.Fatal("source descriptor mapping changed bytes", err)
				}
			}
		}
		if count != example.want {
			t.Fatalf("source-owned descriptor regions=%d want=%d: %s", count, example.want, example.source)
		}
	}
}
