package sitecontent

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Opt-in acceptance against immutable repository evidence; no formal writer.
func TestStructuredHistoricalReadonlyRepository(t *testing.T) {
	if os.Getenv("GO_TOUR_STRUCTURED_HISTORICAL_READONLY") != "1" {
		t.Skip("explicit repository readonly acceptance")
	}
	root := repositoryRoot(t)
	fingerprint := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, name := range []string{"data/site-shell/successor", "data/glossary-source-corpus.json", "data/quality-check-snapshots", "data/translation-authorizations", "data/locale-language", "data/site-workflow"} {
			if err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(name)), func(path string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				out[path] = digest(b)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	before := fingerprint()
	defer func() {
		if !reflect.DeepEqual(before, fingerprint()) {
			t.Error("FORMAL_ARTIFACT_MUTATION detected")
		}
	}()
	done := contentidentity.BeginMemoization()
	defer done()
	start := time.Now()
	v := newClosureValidation(root)
	defer func() {
		if err := v.verify(); err != nil {
			t.Error("read-only dependency verification", err)
		}
	}()
	refs := []Reference{
		{"data/locale-language/zh-CN/structured-generation/3aa00f3907d55604a78a411e88fe0fb478e232dd5f80cbfea66ca9e80c53bc78.json", "e4c7a8b4e9fa94cf06422d9c1a376777faf3a9e43a03f0694a90c358cebd0413"},
		{"data/locale-language/zh-CN/structured-generation/7caebba33580782e5837719efa23a8d1cf4d60f51de84fa3e5c56c8816d25cb5.json", "3bdea5f48d19a48c69964d515d87117a095015d9aa6a33e2ad4837b59a10c8c3"},
	}
	const books = "source:_content/learn/books.yaml"
	const targetSHA = "dd74e596afdf406a44c85550c37442bf4d1479407e9c8335a38f3c4feb5e3a39"
	page := Reference{"data/quality-check-snapshots/zh-CN/learn-docs-v1-qc-015/finalization.json", "530374a7aa563c103f975940b480f998fbd315541c56438e21d2557e22275b1d"}
	// Preserve the public builder's exact order, including the current shared
	// Page proof before traversal of structured predecessors. Inspect only the
	// proofs that this canonical traversal actually completed.
	if err := v.currentInputs("zh-CN"); err != nil {
		t.Fatal("current closure inputs", err)
	}
	cat, err := v.packageCatalog("learn-docs-v1")
	if err != nil {
		t.Fatal(err)
	}
	receiptRef := Reference{"data/translation-unit-reconciliations/zh-CN/learn-docs-qc014-machine-successors/receipt.json", "8f159c38611cd9538c4c71976e66f8191e6fa6007ab18583e983f213b5875324"}
	var receipt i18n.RepresentationReconciliation
	if err := v.readReference(root, receiptRef, &receipt); err != nil {
		t.Fatal(err)
	}
	if len(receipt.Units) != 42 {
		t.Fatal("machine exact set", len(receipt.Units))
	}
	verifier := i18n.NewRepresentationSuccessorVerifier(root, cat, "zh-CN")
	blockerSeen := false
	for _, u := range receipt.Units {
		p, refs, err := verifier.Verify(u.Replacement.OldUnitID, u.Replacement.CurrentUnitID)
		if err != nil || !reflect.DeepEqual(p, &u.Replacement) {
			t.Fatal("machine successor verification", u.Replacement.OldUnitID, err)
		}
		for _, ref := range refs {
			if err := v.bind(Reference{ref.Path, ref.SHA256}); err != nil {
				t.Fatal(err)
			}
		}
		if p.OldUnitID == "d8f9a1931eb5cbf1f5eae5df9648f25b122d758a04472a107fbcf4579a0c60fb" {
			if p.CurrentUnitID != "019f0fdcf18ddaffbb42b88a1cd2aae11f9e7f8dddcb0f16916113933357fa5f" || p.SourceSHA256 != "b0173a57e407fe2937c85ed434dee7037eee0a9d91f495cb4d43a4e6f7dabd35" || p.OldInputSHA256 != "68543e61f3d4d1fe6e13aa1d01abef45799c428d6d59f0b2ca49273e51883d4a" || p.CurrentInputSHA256 != "b72193e1ee213f64c93e9c53402228ec64e083e885442d2b18ba6b0a6c209704" || p.ReconciliationSHA256 != "5d3a8d71c9cfa2b5385325c9291d3eb4a88e86194fb337febe80b9b48b52221b" {
				t.Fatal("blocker exact bridge identity")
			}
			blockerSeen = true
		}
	}
	if !blockerSeen {
		t.Fatal("blocker not in formal exact set")
	}
	t.Logf("RECONCILIATION_RECEIPT_VERIFIED=PASS EXACT_SUCCESSOR_VERIFIED=PASS MACHINE_RECONCILIATION_SUCCESSORS_VERIFIED=42 elapsed=%s", time.Since(start))
	var finalization i18n.QualityCheckFinalization
	if err := v.readReference(root, page, &finalization); err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct {
		index int
		batch string
	}{{40, "chatgpt-zh-CN-047"}, {47, "chatgpt-zh-CN-048"}, {16, "chatgpt-zh-CN-022"}} {
		if len(finalization.Units) != 49 || finalization.Units[pair.index-1].Snapshot.SelectedBatchID != pair.batch {
			t.Fatal("current finalized revision lineage changed", pair)
		}
	}
	t.Log("qc-015 lineage: fips140=047 vuln/database=048 module-conclusion=022")
	c, files, buildErr := v.buildPackageClosure(root, "zh-CN", "learn-docs-v1", &page, refs)
	for i, ref := range refs {
		proof, ok := v.generations[ref]
		if !ok {
			t.Logf("HISTORICAL_GENERATION_%d=not-completed", i+1)
			continue
		}
		r := proof.generation
		for _, a := range r.Plan.Assets {
			if a.ID != books {
				continue
			}
			if a.SourceSHA != "3ba174700fb64fcd242e65fbf85831cc1f3e7b9dd17f928e1ab8ef38a4b7de60" || a.Document.SourceSHA != "929aff40ecf99c211788d4edc7bd88697e290a3f7976559f39ba6403dcc8f19a" || digest(proof.files[a.TargetPath]) != targetSHA {
				t.Fatal("books historical identity/target drift")
			}
			if i == 1 && (a.State != "generation-required" || a.Assessment == nil || a.Assessment.Classification != "affected" || a.Assessment.TargetSHA != targetSHA) {
				t.Fatal("stored books assessment changed")
			}
		}
		t.Logf("HISTORICAL_GENERATION_%d=PASS elapsed=%s; Bundle/source/Document/output integrity PASS", i+1, time.Since(start))
	}
	baselineRef := Reference{"data/site-workflow/zh-CN/learn-docs-v1/closure/8ca5ed968d28c26ac2c186514dff1cf2c50f4df29e7071c593a954390ea34ca7.json", "6c77995569df3483699c36fb65ea7fd34d497dd4495d2e54b8220a5d290f01e7"}
	if b := v.closures[baselineRef]; b != nil {
		if len(b.Files) != 56 {
			t.Fatal("baseline inventory", len(b.Files))
		}
		t.Logf("BASELINE_CLOSURE_INTEGRITY=PASS files=56 elapsed=%s; stored books assessment=affected/generation-required", time.Since(start))
	}
	if buildErr != nil {
		t.Fatalf("PACKAGE_CLOSURE_READONLY_EQUIVALENT=FAILED NEXT_BLOCKER=%s elapsed=%s", buildErr, time.Since(start))
	}
	if len(files) != 56 || len(c.Files) != 56 {
		t.Fatal("current inventory mismatch")
	}
	t.Logf("PACKAGE_CLOSURE_READONLY_EQUIVALENT=PASS files=56 elapsed=%s; no artifact saved", time.Since(start))
}
