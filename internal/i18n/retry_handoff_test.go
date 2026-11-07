package i18n

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestRetryHandoffDeduplicationOriginalIdentityAndSiblingCurrentness(t *testing.T) {
	root, c, batch, bundles, manifests := siblingRetryFixture(t)
	handoff, h, err := ExportRetryHandoff(root, c, bundles)
	if err != nil {
		t.Fatal(err)
	}
	reversed := [][]byte{bundles[2], bundles[0], bundles[1]}
	again, _, err := ExportRetryHandoff(root, c, reversed)
	if err != nil || !bytes.Equal(again, handoff) {
		t.Fatal("handoff ordering/determinism", err)
	}
	files, err := ReadTransportBundle(handoff, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	_, single, err := readGenerationBundle(bundles[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"authority/AGENTS.md", "formal/glossary.yaml", "formal/manifest.json", "formal/retry/result.json"} {
		if !bytes.Equal(files[retryHandoffObject(single[name])], single[name]) {
			t.Fatal("missing shared bytes", name)
		}
	}
	total := 0
	logicalStoreTotal := 0
	for _, b := range bundles {
		total += len(b)
		m, f, err := readGenerationBundle(b)
		if err != nil {
			t.Fatal(err)
		}
		store, err := WriteDeterministicTransportBundle(f["manifest.json"], generationCompressionEntries(m, f))
		if err != nil {
			t.Fatal(err)
		}
		logicalStoreTotal += len(store)
	}
	// Handoff retains Store. Compare dedup savings in the same encoding;
	// independent Deflate constituents can be smaller than a Store handoff.
	if len(handoff) >= logicalStoreTotal {
		t.Fatal("no transport deduplication")
	}
	_, originals, err := ReadRetryHandoff(handoff)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range originals {
		if !bytes.Equal(b, bundles[i]) || sum(b) != h.Members[i].BundleSHA256 {
			t.Fatal("original input identity changed")
		}
	}
	if _, err := VerifyCurrentRetryHandoff(root, c, handoff); err != nil {
		t.Fatal(err)
	}
	// Consume out of order, retain exact original provenance for all siblings.
	for _, i := range []int{2, 0, 1} {
		b := originals[i]
		if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, c, b); err != nil {
			t.Fatal(err)
		}
		dir := t.TempDir()
		writeGenerationFixtureOutputs(t, root, batch, dir, manifests[i])
		if _, err := ImportGenerationOutputDirectory(root, c, b, "codex", FormalGenerationModel, dir); err != nil {
			t.Fatal(err)
		}
		if _, err := VerifyCurrentRetryHandoff(root, c, handoff); err == nil {
			t.Fatal("partially consumed aggregate CURRENT")
		}
		if _, err := ProcessRetranslationRetry(root, c, RetranslationRetryOptions{Locale: "zh-CN", BatchID: batch, UnitID: manifests[i].ExpectedOutputs[0].UnitID}); err != nil {
			t.Fatal(err)
		}
		// Integrity extraction stays possible, without a CURRENT claim.
		if _, _, err := ReadRetryHandoff(handoff); err != nil {
			t.Fatal(err)
		}
		receiptPath := generationImportReceiptPath(filepath.Join("data/retranslation-runs/zh-CN", batch), manifests[i])
		receipt, err := readSnapshotReferencedFile(root, filepath.ToSlash(receiptPath))
		if err != nil {
			t.Fatal(err)
		}
		var provenance GenerationResultBundleManifest
		if err := json.Unmarshal(receipt, &provenance); err != nil || provenance.GenerationBundleSHA256 != sum(b) || provenance.InputIdentitySHA256 != manifests[i].InputIdentitySHA256 {
			t.Fatal("aggregate substituted provenance", err)
		}
	}
	t.Logf("3 canonical ZIPs=%d bytes; one deduplicated handoff=%d bytes", total, len(handoff))
}

func TestRetryHandoffStoreDeflateExactRecovery(t *testing.T) {
	root, c, _, bundles, _ := siblingRetryFixture(t)
	m, files, err := readGenerationBundle(bundles[0])
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := WriteDeterministicTransportBundle(files["manifest.json"], generationCompressionEntries(m, files))
	if err != nil {
		t.Fatal(err)
	}
	// A historical mixed handoff is readable, preserving each original SHA.
	// The Store constituent is deliberately not current under the new writer.
	originals := [][]byte{legacy, bundles[1], bundles[2]}
	handoff, h, err := packRetryHandoff(originals)
	if err != nil {
		t.Fatal(err)
	}
	_, restored, err := ReadRetryHandoff(handoff)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range restored {
		if !bytes.Equal(b, originals[i]) || sum(b) != h.Members[i].BundleSHA256 {
			t.Fatal("historical transport rebound to a different encoding", i)
		}
	}
	if _, err := VerifyCurrentRetryHandoff(root, c, handoff); err == nil {
		t.Fatal("readable Store constituent incorrectly declared CURRENT")
	}
}

func TestRetryHandoffRejectsTamperingStaleDuplicateAndMixedScope(t *testing.T) {
	root, c, _, bundles, _ := siblingRetryFixture(t)
	for _, inputs := range [][][]byte{nil, {bundles[0], bundles[0]}} {
		if _, _, err := ExportRetryHandoff(root, c, inputs); err == nil {
			t.Fatal("invalid exact set accepted")
		}
	}
	handoff, _, err := ExportRetryHandoff(root, c, bundles)
	if err != nil {
		t.Fatal(err)
	}
	h, originals, err := ReadRetryHandoff(handoff)
	if err != nil {
		t.Fatal(err)
	}
	if len(originals) != 3 {
		t.Fatal("wrong exact outputs")
	}
	for _, kind := range []string{"locale", "batch", "unit-kind"} {
		m, files, err := readGenerationBundle(bundles[1])
		if err != nil {
			t.Fatal(err)
		}
		switch kind {
		case "locale":
			m.Locale = "ja-JP"
		case "batch":
			m.BatchID = "codex-zh-CN-002"
		case "unit-kind":
			m.UnitKind = UnitKindExample
		}
		manifest, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		entries := []TransportBundleEntry{}
		for _, f := range m.Files {
			entries = append(entries, TransportBundleEntry{Path: f.BundlePath, Data: files[f.BundlePath]})
		}
		other, err := WriteDeterministicTransportBundle(append(manifest, '\n'), entries)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := packRetryHandoff([][]byte{bundles[0], other}); err == nil {
			t.Fatal("mixed scope accepted", kind)
		}
	}
	// Reseal transport inventory to ensure semantic/canonical checks still reject.
	for _, kind := range []string{"attempt", "unit", "output", "order", "sha", "input-identity", "missing-object", "extra-object"} {
		t.Run(kind, func(t *testing.T) {
			files, err := ReadTransportBundle(handoff, 8192, 64<<20)
			if err != nil {
				t.Fatal(err)
			}
			bad := h
			bad.Members = append([]RetryHandoffMember(nil), h.Members...)
			switch kind {
			case "attempt":
				bad.Members[0].Attempt++
			case "unit":
				bad.Members[0].UnitID = "unknown"
			case "output":
				bad.Members[0].ExpectedOutput.InstallPath = "retries/forged/attempt-003.article"
			case "order":
				bad.Members[0], bad.Members[1] = bad.Members[1], bad.Members[0]
			case "sha":
				bad.Members[0].BundleSHA256 = sum([]byte("forged"))
			case "input-identity":
				bad.Members[0].InputIdentitySHA256 = sum([]byte("forged"))
			case "missing-object":
				delete(files, bad.Members[0].ManifestObject)
			case "extra-object":
				files[retryHandoffObject([]byte("extra"))] = []byte("extra")
			}
			bad.Files = nil
			entries := []TransportBundleEntry{}
			for p, b := range files {
				if p != "manifest.json" {
					bad.Files = append(bad.Files, NewTransportBundleFile(p, "", b))
					entries = append(entries, TransportBundleEntry{Path: p, Data: b})
				}
			}
			m, err := json.MarshalIndent(bad, "", "  ")
			if err != nil {
				t.Fatal(err)
			}
			b, err := WriteDeterministicTransportBundle(append(m, '\n'), entries)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := ReadRetryHandoff(b); err == nil {
				t.Fatal("tampered aggregate accepted")
			}
		})
	}
	// A formal source mutation is checked constituent-by-constituent.
	c.Pages[1].Source = append(c.Pages[1].Source, []byte("changed")...)
	if _, err := VerifyCurrentRetryHandoff(root, c, handoff); err == nil {
		t.Fatal("stale constituent accepted")
	}
	if _, _, err := ExportRetryHandoff(root, c, bundles); err == nil {
		t.Fatal("stale aggregate export")
	}
}
