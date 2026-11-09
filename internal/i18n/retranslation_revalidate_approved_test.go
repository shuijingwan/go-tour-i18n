package i18n

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCurrentValidationRevalidateRejectsInstalledApprovedInventory(t *testing.T) {
	root, c, o, exported := importedCommentProcessFixture(t)
	result, err := ProcessRetranslationBatch(root, c, RetranslationProcessOptions{Locale: o.Locale, BatchID: exported.BatchID})
	if err != nil || result.ValidationPassed != 9 {
		t.Fatal(result, err)
	}
	dir := filepath.Join(root, "data/retranslation-runs", o.Locale, exported.BatchID)
	m, err := readRetranslationProcessManifest(dir, o.Locale, exported.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	ip := filepath.Join(dir, "generation-imports/initial-manifest.json")
	ib, err := os.ReadFile(ip)
	if err != nil {
		t.Fatal(err)
	}
	var original GenerationResultBundleManifest
	if err := json.Unmarshal(ib, &original); err != nil {
		t.Fatal(err)
	}
	archive, err := os.ReadFile(filepath.Join(root, importedGenerationArchive(o.Locale, original.GenerationBundleSHA256)))
	if err != nil {
		t.Fatal(err)
	}
	gm, files, err := readGenerationBundle(archive)
	if err != nil {
		t.Fatal(err)
	}
	before := map[string][]byte{}
	for _, path := range []string{"result.json", result.Units[0].CandidatePath, result.Units[0].ValidationPath} {
		before[path], err = os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, mode := range []string{"proof-without-selection", "extra-proof", "empty-selection", "malformed-selection", "malformed-import", "missing-archive"} {
		t.Run(mode, func(t *testing.T) {
			defer func() {
				if err := os.WriteFile(ip, ib, 0644); err != nil {
					t.Error(err)
				}
			}()
			receipt := original
			if mode == "malformed-import" {
				if err := os.WriteFile(ip, []byte("{"), 0644); err != nil {
					t.Fatal(err)
				}
			} else if mode == "missing-archive" {
				receipt.GenerationBundleSHA256 = sum([]byte("missing immutable archive"))
				if err := writeTranslationJSON(ip, receipt); err != nil {
					t.Fatal(err)
				}
			} else {
				path := approvedIntervalSelectionMember
				data := []byte(`{"schema":"unknown"}`)
				switch mode {
				case "proof-without-selection":
					path = approvedIntervalProofMember(m.Units[0].UnitID)
					data = []byte("not valid proof JSON")
				case "extra-proof":
					path = approvedIntervalProofMember("not-in-this-scope")
					data = []byte("{}")
				case "empty-selection":
					data = []byte{}
				}
				changed := gm
				changed.Files = append(append([]TransportBundleFile(nil), gm.Files...), NewTransportBundleFile(path, "", data))
				sort.Slice(changed.Files, func(i, j int) bool { return changed.Files[i].BundlePath < changed.Files[j].BundlePath })
				entries := []TransportBundleEntry{}
				var identity strings.Builder
				for _, f := range changed.Files {
					b := files[f.BundlePath]
					if f.BundlePath == path {
						b = data
					}
					entries = append(entries, TransportBundleEntry{Path: f.BundlePath, Data: b})
					identity.WriteString(f.BundlePath + "\x00" + f.SHA256 + "\x00")
				}
				changed.InputIdentitySHA256 = sum([]byte(identity.String()))
				mb, err := json.Marshal(changed)
				if err != nil {
					t.Fatal(err)
				}
				bundle, err := writeTranslationUnitGenerationBundle(mb, entries)
				if err != nil {
					t.Fatal(err)
				}
				receipt.GenerationBundleSHA256 = sum(bundle)
				receipt.InputIdentitySHA256 = changed.InputIdentitySHA256
				receipt.GenerationManifest = NewTransportBundleFile("generation-manifest.json", "", mb)
				ap := importedGenerationArchive(o.Locale, receipt.GenerationBundleSHA256)
				if err := writeCompatibilityImmutable(root, ap, bundle); err != nil {
					t.Fatal(err)
				}
				defer os.Remove(filepath.Join(root, ap))
				if err := writeTranslationJSON(ip, receipt); err != nil {
					t.Fatal(err)
				}
			}
			_, err := RevalidateRetranslationCandidate(root, c, RetranslationRevalidateOptions{Locale: o.Locale, BatchID: exported.BatchID, UnitID: m.Units[0].UnitID})
			if err == nil {
				t.Fatal("unrecognized installed transport entered current-only revalidation", mode)
			}
			if (mode == "proof-without-selection" || mode == "extra-proof") && !strings.Contains(err.Error(), "unselected approved interval proof") {
				t.Fatal("unselected proof did not reach strict decoder/proof comparison", err)
			}
			for path, want := range before {
				got, readErr := os.ReadFile(filepath.Join(dir, path))
				if readErr != nil || !bytes.Equal(got, want) {
					t.Fatal("blocked revalidation mutated candidate/result/validation", path, readErr)
				}
			}
		})
	}
}
