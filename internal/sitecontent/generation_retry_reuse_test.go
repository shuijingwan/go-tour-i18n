package sitecontent

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Reuse the synthetic two-Unit approved-interval bridge fixture, including its
// distinct per-Unit predecessor identities. No real locale state is executed.
func testRegisteredBridgeRetryReuse(t *testing.T, root string, catalog *i18n.Catalog, batch *i18n.RetranslationExportResult, dependency string) {
	t.Helper()
	var m i18n.RetranslationBatchManifest
	if err := json.Unmarshal(mustReadFixture(t, root, batch.BatchPath+"/manifest.json"), &m); err != nil {
		t.Fatal(err)
	}
	for _, u := range m.Units {
		writeFixture(t, root, batch.BatchPath+"/raw-responses/"+filepath.Base(u.InputPath), []byte("broken response\n"))
	}
	c := *catalog
	AttachPageReplacementInvocation(root, &c)
	failed, err := i18n.ProcessRetranslationBatch(root, &c, i18n.RetranslationProcessOptions{Locale: m.Locale, BatchID: m.BatchID})
	if err != nil || failed.RestoreFailed != m.UnitCount {
		t.Fatal("synthetic failed batch", failed, err)
	}
	options := i18n.GenerationBundleOptions{Locale: m.Locale, BatchID: m.BatchID, UnitID: m.Units[0].UnitID}
	bundle, bm, err := i18n.ExportTranslationUnitGenerationBundle(root, &c, options)
	if err != nil {
		t.Fatal(err)
	}
	observe := func() map[string]string {
		t.Helper()
		files := map[string]string{}
		if err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err == nil && !d.IsDir() {
				b, readErr := os.ReadFile(path)
				err = readErr
				files[path] = digest(b)
			}
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return files
	}
	before := observe()
	buildCounts := []int{}
	for _, reuse := range []bool{false, true} {
		// Separate command-owned scopes. Baseline uses the same construction
		// context and fresh gates, but forces the canonical bridge build.
		v := newClosureValidation(root)
		v.metrics = map[string]int{}
		v.attachReplacementReuse(&c)
		resolve := c.BridgeResolver
		requests, builds := 0, 0
		c.BridgeResolver = func(root string, input *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
			requests++
			counted := func() (*i18n.SameUnitBridgeResolution, error) { builds++; return build() }
			if !reuse {
				return counted()
			}
			return resolve(root, input, mode, ref, counted)
		}
		start := time.Now()
		checked, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, &c, bundle)
		elapsed := time.Since(start)
		if err != nil || !reflect.DeepEqual(checked, bm) {
			t.Fatal("retry currentness/identity changed", err)
		}
		buildCounts = append(buildCounts, builds)
		t.Logf("retry current-check reuse=%t wall=%s bridge_requests=%d full_builds=%d", reuse, elapsed, requests, builds)
	}
	if buildCounts[1] >= buildCounts[0] || buildCounts[1] == 0 {
		t.Fatal("registered codec disabled bridge reuse", buildCounts)
	}
	// The combined CLI calls these same two public operations with one owner.
	// Verification keeps its fresh gates but must reuse export's full builds.
	AttachPageReplacementInvocation(root, &c)
	combinedResolver := c.BridgeResolver
	combinedBuilds := 0
	c.BridgeResolver = func(root string, input *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
		return combinedResolver(root, input, mode, ref, func() (*i18n.SameUnitBridgeResolution, error) { combinedBuilds++; return build() })
	}
	combined, cm, err := i18n.ExportTranslationUnitGenerationBundle(root, &c, options)
	if err != nil || !bytes.Equal(combined, bundle) {
		t.Fatal("combined export drift", err)
	}
	exportBuilds := combinedBuilds
	verified, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, &c, combined)
	if err != nil || !reflect.DeepEqual(cm, verified) || combinedBuilds != exportBuilds {
		t.Fatal("combined verifier repeated full reconstruction", exportBuilds, combinedBuilds, err)
	}
	t.Logf("combined bridge export_builds=%d verification_additional_builds=%d", exportBuilds, combinedBuilds-exportBuilds)
	// A separately rebuilt canonical transport is still byte-identical.
	rebuilt, _, err := i18n.ExportTranslationUnitGenerationBundle(root, &c, options)
	if err != nil || !bytes.Equal(bundle, rebuilt) {
		t.Fatal("canonical retry bytes changed", err)
	}
	for _, path := range []string{dependency, batch.BatchPath + "/result.json", batch.BatchPath + "/validation/" + m.Units[0].UnitID + ".json"} {
		t.Run("drift/"+filepath.Base(path), func(t *testing.T) {
			old := mustReadFixture(t, root, path)
			defer writeFixture(t, root, path, old)
			writeFixture(t, root, path, append(bytes.Clone(old), ' '))
			if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, &c, bundle); err == nil {
				t.Fatal("warm reconstruction hid dependency/result/validation drift", path)
			}
		})
	}
	// Another invocation must reconstruct again, even after a successful check.
	AttachPageReplacementInvocation(root, &c)
	resolve := c.BridgeResolver
	builds := 0
	c.BridgeResolver = func(root string, input *i18n.Catalog, mode string, ref i18n.GlossaryArchiveReference, build func() (*i18n.SameUnitBridgeResolution, error)) (*i18n.SameUnitBridgeResolution, error) {
		return resolve(root, input, mode, ref, func() (*i18n.SameUnitBridgeResolution, error) { builds++; return build() })
	}
	if _, err := i18n.VerifyCurrentTranslationUnitGenerationBundle(root, &c, bundle); err != nil || builds != buildCounts[1] {
		t.Fatal("reconstruction escaped invocation", builds, buildCounts, err)
	}
	if !reflect.DeepEqual(before, observe()) {
		t.Fatal("readonly retry check mutated fixture or created persistent state")
	}
}
