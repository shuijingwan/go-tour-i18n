package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func surfaceCapabilityFixture(t *testing.T) (string, *i18n.Catalog, string) {
	return surfaceCapabilityFixtureSize(t, 1)
}

func surfaceCapabilityFixtureSize(t *testing.T, count int) (string, *i18n.Catalog, string) {
	t.Helper()
	root, _, c, _ := replacementPageFixtureSize(t, count, count, func(int, []byte) []byte {
		return []byte("# Surface transport\n\nRead the [`sql.OpenDB function`](https://pkg.go.dev/database/sql#OpenDB) now.\n\nUnrelated prose remains unchanged.\n\n```go\npackage p\n//go:generate echo keep\nconst s = \"literal\"\n```\n")
	})
	f := surfaceStart(t, root, c, c.TranslationScope)
	build := surfaceBuilder(t, root, c, &f)
	attachIntegratedSurfaceAuthority(c, build)
	closure := surfaceDescriptorClosure(t, root, c, f.SnapshotID)
	findings := []SurfaceFinding{}
	texts := []string{}
	for _, x := range f.Units {
		u := x.Snapshot
		text := "Correct the inline API descriptor in this exact paragraph."
		if count > 1 {
			text += " " + u.SourcePath
		}
		findings = append(findings, SurfaceFinding{ID: u.SourcePath, Finding: text, Class: i18n.SurfaceTranslationUnitLanguage, Artifacts: []i18n.SurfaceCandidateArtifact{{SnapshotID: f.SnapshotID, Unit: u}}})
		texts = append(texts, text)
	}
	r, _ := surfaceReceipt(t, root, c, build, []Reference{closure}, "surface-capability-failed", "failed", findings)
	o := i18n.QualityCheckSurfaceReopenOptions{Locale: "zh-CN", ReopenID: "surface-capability-reopen", PreviousSnapshotID: f.SnapshotID, SurfaceReviewID: r.ReviewID, Finding: texts[0], UnitIDs: c.TranslationScope, IntegratedSurface: true}
	if count > 1 {
		o.Finding = ""
		o.Findings = texts
	}
	if _, _, err := i18n.RecordQualityCheckSurfaceReopen(root, c, o); err != nil {
		t.Fatal(err)
	}
	e, err := i18n.ExportRetranslationBatch(root, c, i18n.RetranslationExportOptions{Locale: o.Locale, Generator: i18n.RetranslationGeneratorChatGPT, UnitIDs: o.UnitIDs, PreviousSnapshotID: o.PreviousSnapshotID, SurfaceReopenID: o.ReopenID, AllowReexport: true, UnitKind: i18n.UnitKindPage})
	if err != nil {
		t.Fatal(err)
	}
	return root, c, e.BatchID
}

func surfaceCapabilityRequest(t *testing.T, p *i18n.SurfaceTransportPlan, descriptor bool) i18n.SurfaceTransportRequest {
	t.Helper()
	r := p.Authority.Request
	for _, b := range p.Authority.Bindings {
		q := i18n.SurfaceTransportUnitRequest{UnitID: b.UnitID, FindingSHA256: b.FindingSHA256}
		for _, region := range b.Projection.Available {
			if (region.Kind == "inline-api-descriptor") == descriptor {
				q.Regions = append(q.Regions, region.ID)
				break
			}
		}
		if len(q.Regions) == 0 {
			t.Fatal("no requested source-owned coordinate")
		}
		r.Units = append(r.Units, q)
	}
	return r
}

func TestSurfaceRevisionTransportCapability(t *testing.T) {
	root, c, batch := surfaceCapabilityFixture(t)
	o := i18n.GenerationBundleOptions{Locale: "zh-CN", BatchID: batch}
	if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, o); err == nil || !strings.Contains(err.Error(), "capability authority required") {
		t.Fatal("handoff did not fail before Generation", err)
	}
	discovery, err := i18n.PrepareSurfaceRevisionTransport(root, c, o.Locale, batch, nil, false)
	if err != nil || discovery.RevisionTransportCapable || discovery.Recorded {
		t.Fatal("discovery is not capability authorization", err)
	}
	if _, err := i18n.PrepareSurfaceRevisionTransport(root, c, o.Locale, batch, nil, true); err == nil {
		t.Fatal("unbounded discovery recorded")
	}
	request := surfaceCapabilityRequest(t, discovery, true)
	t.Run("saved-input-drift", func(t *testing.T) {
		b, err := readRegular(root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json")
		if err != nil {
			t.Fatal(err)
		}
		var m i18n.RetranslationBatchManifest
		if json.Unmarshal(b, &m) != nil {
			t.Fatal("manifest")
		}
		path := "data/retranslation-runs/zh-CN/" + batch + "/" + m.Units[0].InputPath
		before, err := readRegular(root, path)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, append(bytes.Clone(before), ' '))
		defer writeFixture(t, root, path, before)
		if _, err := i18n.PrepareSurfaceRevisionTransport(root, c, o.Locale, batch, &request, false); err == nil {
			t.Fatal("saved input not bound")
		}
	})
	for _, name := range []string{"wrong-unit", "wrong-finding", "wrong-batch", "wrong-manifest", "superset", "duplicate"} {
		t.Run(name, func(t *testing.T) {
			b, _ := json.Marshal(request)
			var bad i18n.SurfaceTransportRequest
			json.Unmarshal(b, &bad)
			switch name {
			case "wrong-unit":
				bad.Units[0].UnitID = "other"
			case "wrong-finding":
				bad.Units[0].FindingSHA256 = strings.Repeat("0", 64)
			case "wrong-batch":
				bad.BatchID = "chatgpt-zh-CN-999"
			case "wrong-manifest":
				bad.ManifestSHA256 = strings.Repeat("0", 64)
			case "superset":
				bad.Units[0].Regions = append(bad.Units[0].Regions, "not-a-source-region")
			case "duplicate":
				bad.Units[0].Regions = append(bad.Units[0].Regions, bad.Units[0].Regions[0])
			}
			if _, err := i18n.PrepareSurfaceRevisionTransport(root, c, o.Locale, batch, &bad, false); err == nil {
				t.Fatal("wrong capability authority accepted")
			}
		})
	}
	preflight, err := i18n.PrepareSurfaceRevisionTransport(root, c, o.Locale, batch, &request, false)
	if err != nil || !preflight.RevisionTransportCapable || preflight.Recorded {
		t.Fatal("explicit capability preflight", err)
	}
	if _, err := os.Stat(filepath.Join(root, preflight.Path)); !os.IsNotExist(err) {
		t.Fatal("readonly preflight mutated authority", err)
	}
	plan, err := i18n.PrepareSurfaceRevisionTransport(root, c, o.Locale, batch, &request, true)
	if err != nil || !plan.Recorded {
		t.Fatal("isolated capability record", err)
	}
	if _, err := i18n.PrepareSurfaceRevisionTransport(root, c, o.Locale, batch, &request, true); err == nil {
		t.Fatal("immutable receipt overwritten")
	}
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, c, o)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("cross-batch-authority", func(t *testing.T) {
		original := "data/retranslation-runs/zh-CN/" + batch
		b, err := readRegular(root, original+"/manifest.json")
		if err != nil {
			t.Fatal(err)
		}
		var m i18n.RetranslationBatchManifest
		if json.Unmarshal(b, &m) != nil {
			t.Fatal("manifest")
		}
		m.BatchID = "chatgpt-zh-CN-999"
		base := "data/retranslation-runs/zh-CN/" + m.BatchID
		defer os.RemoveAll(filepath.Join(root, base))
		b, _ = json.Marshal(m)
		writeFixture(t, root, base+"/manifest.json", b)
		for _, u := range m.Units {
			b, err := readRegular(root, original+"/"+u.InputPath)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, base+"/"+u.InputPath, b)
		}
		if _, _, err := i18n.ExportTranslationUnitGenerationBundle(root, c, i18n.GenerationBundleOptions{Locale: o.Locale, BatchID: m.BatchID}); err == nil {
			t.Fatal("capability crossed batch identity")
		}
	})
	t.Run("codec-scope-superset", func(t *testing.T) {
		before, err := readRegular(root, plan.Path)
		if err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, plan.Path, before)
		var changed i18n.SurfaceTransportAuthority
		if json.Unmarshal(before, &changed) != nil {
			t.Fatal("authority")
		}
		changed.Bindings[0].Projection.Editable = append(changed.Bindings[0].Projection.Editable, "unrequested")
		changed.Identity = ""
		changed.Identity = identity(changed)
		b, _ := json.Marshal(changed)
		writeFixture(t, root, plan.Path, append(b, '\n'))
		if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
			t.Fatal("canonical identity hid scope superset")
		}
	})
	checked, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle)
	if err != nil || !reflect.DeepEqual(bm, checked) {
		t.Fatal("independent capability/currentness rebuild", err)
	}
	files, err := i18n.ReadTransportBundle(bundle, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var context map[string]any
	if json.Unmarshal(files["generation-context.json"], &context) != nil || context["current_inputs"] != true || context["revision_transport_capable"] != true {
		t.Fatal("currentness and capability conflated")
	}
	binding := plan.Authority.Bindings[0]
	u, _ := c.Unit(binding.UnitID)
	factory := u.Adapter.(i18n.SurfaceTransportAdapter)
	adapter, err := factory.ReadSurfaceTransportProjection(binding.Projection.Codec)
	if err != nil {
		t.Fatal(err)
	}
	got, err := adapter.Restore(binding.Projection.Template)
	candidate, readErr := readRegular(root, binding.Predecessor.CandidatePath)
	if err != nil || readErr != nil || !bytes.Equal(bridgeEOF(got), candidate) {
		t.Fatal("exact restore", err, readErr)
	}
	revised := strings.Replace(binding.Projection.Template, "function", "函数", 1)
	got, err = adapter.Restore(revised)
	if err != nil || !bytes.Contains(got, []byte("sql.OpenDB 函数")) || bytes.Contains(got, []byte("sql.OpenDB function")) {
		t.Fatal("descriptor cannot be expressed", err)
	}
	if err := i18n.ValidateTranslationUnitCandidate(root, c, u.ID, o.Locale, got); err != nil {
		t.Fatal("canonical structural/source invariants changed", err)
	}
	if u.ID != binding.Predecessor.UnitID || !bytes.Contains(got, []byte("Unrelated prose remains unchanged.")) || !bytes.Contains(got, []byte("//go:generate echo keep")) || !bytes.Contains(got, []byte("https://pkg.go.dev/database/sql#OpenDB")) {
		t.Fatal("opaque scope or formal Unit changed")
	}
	for _, bad := range []string{revised + "outside", strings.Replace(revised, "函数", "`changed`", 1), strings.Replace(revised, "函数", "函数\nnew", 1)} {
		if _, err := adapter.Restore(bad); err == nil {
			t.Fatal("opaque/code boundary escaped")
		}
	}
	for name, path := range map[string]string{"receipt": plan.Path, "raw": binding.Raw.Path, "candidate": binding.Predecessor.CandidatePath, "validation": binding.Predecessor.ValidationPath, "snapshot": "data/quality-check-snapshots/zh-CN/" + requestSnapshot(t, root, batch) + "/manifest.json"} {
		t.Run("tampered-"+name, func(t *testing.T) {
			before, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, root, path, append(bytes.Clone(before), ' '))
			defer writeFixture(t, root, path, before)
			if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, c, bundle); err == nil {
				t.Fatal("tampered capability stayed CURRENT")
			}
		})
	}
	// The projection must remain a usable response decoder, rather than merely
	// a handoff report. Everything below is isolated synthetic fixture data.
	previous := requestSnapshot(t, root, batch)
	surfaceImport(t, root, c, bundle, map[string][]byte{u.ID: bridgeEOF([]byte(revised))})
	processed, err := i18n.ProcessRetranslationBatch(root, c, i18n.RetranslationProcessOptions{Locale: o.Locale, BatchID: batch})
	if err != nil || processed.ValidationPassed != 1 {
		t.Fatal("Surface projection process", processed, err)
	}
	f := surfaceFinalize(t, root, c, "surface-capability-successor", previous, []string{u.ID})
	if _, err := i18n.VerifyHistoricalQualityCheckFinalization(root, o.Locale, f.SnapshotID, c); err != nil {
		t.Fatal("Surface projection historical reconstruction", err)
	}
}

func requestSnapshot(t *testing.T, root, batch string) string {
	t.Helper()
	b, err := readRegular(root, "data/retranslation-runs/zh-CN/"+batch+"/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var m i18n.RetranslationBatchManifest
	if json.Unmarshal(b, &m) != nil {
		t.Fatal("manifest")
	}
	return m.Units[0].PreviousSnapshotID
}

func TestSurfaceOrdinaryCapabilityDoesNotRequireBridge(t *testing.T) {
	root, c, ids := surfaceFixture(t, "learn-docs", 1)
	f := surfaceStart(t, root, c, ids)
	// Reuse the existing full authority chain test, which records capability
	// below, while checking that no new ordinary representation is manufactured.
	_ = f
	u, _ := c.Unit(ids[0])
	a := u.Adapter.(PageAdapter)
	text, _, _ := a.ProtectedInput(nil)
	p, _, err := a.SurfaceTransportProjection(a, []byte(text), nil)
	if err != nil {
		t.Fatal(err)
	}
	var ordinary string
	for _, r := range p.Available {
		if r.Kind == "ordinary-prose" {
			ordinary = r.ID
			break
		}
	}
	p, codec, err := a.SurfaceTransportProjection(a, []byte(text), []string{ordinary})
	if err != nil || p.Kind != "ordinary" || p.Template != text {
		t.Fatal("ordinary transport acquired unnecessary bridge", err)
	}
	got, err := codec.Restore(text)
	if err != nil || !bytes.Equal(got, a.Source) {
		t.Fatal("ordinary restore changed", err)
	}
}
