package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
)

// RetryHandoff is transport only. Each member remains an ordinary single-Unit
// Generation Bundle, reconstructed byte-for-byte for the existing importer.
// It has no attempt state, receipt, provider or acceptance authority.
type RetryHandoff struct {
	Kind    string                `json:"kind"`
	Locale  string                `json:"locale"`
	BatchID string                `json:"batch_id"`
	Members []RetryHandoffMember  `json:"members"`
	Files   []TransportBundleFile `json:"files"`
}

type RetryHandoffMember struct {
	Index               int                            `json:"index"`
	UnitID              string                         `json:"unit_id"`
	Attempt             int                            `json:"attempt"`
	BundleSHA256        string                         `json:"canonical_bundle_sha256"`
	InputIdentitySHA256 string                         `json:"input_identity_sha256"`
	ManifestObject      string                         `json:"manifest_object"`
	ExpectedOutput      GenerationBundleExpectedOutput `json:"expected_output"`
}

func retryHandoffObject(data []byte) string { return "objects/" + sum(data) + ".txt" }

// InstallRetryHandoffDirectory shares the no-replace transport installation
// primitive; no formal batch files or provenance are installed by unpack.
func InstallRetryHandoffDirectory(staging, destination string) error {
	return renameGenerationDirNoReplace(staging, destination)
}

// ExportRetryHandoff deduplicates authenticated bytes, never canonical retry
// identities. Members belong to one batch and follow its immutable Unit order.
func ExportRetryHandoff(root string, catalog *Catalog, bundles [][]byte) ([]byte, RetryHandoff, error) {
	var empty RetryHandoff
	if len(bundles) == 0 || len(bundles) > MaxRetranslationExportLimit {
		return nil, empty, fmt.Errorf("retry handoff requires 1..%d members", MaxRetranslationExportLimit)
	}
	total := 0
	for _, b := range bundles {
		total += len(b)
		if total > 64<<20 {
			return nil, empty, fmt.Errorf("retry handoff reconstructed inputs exceed 64 MiB")
		}
		if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, catalog, b); err != nil {
			return nil, empty, err
		}
	}
	b, h, err := packRetryHandoff(bundles)
	if err != nil {
		return nil, h, err
	}
	if _, _, err := ReadRetryHandoff(b); err != nil {
		return nil, h, err
	}
	return b, h, nil
}

func packRetryHandoff(bundles [][]byte) ([]byte, RetryHandoff, error) {
	h := RetryHandoff{Kind: "go-tour-i18n/retry-handoff/v1"}
	objects := map[string][]byte{}
	seen := map[string]bool{}
	var kind UnitKind
	for _, b := range bundles {
		m, files, err := readGenerationBundle(b)
		if err != nil {
			return nil, h, err
		}
		if retryUnitID(m) == "" || seen[retryUnitID(m)] {
			return nil, h, fmt.Errorf("handoff requires distinct single-Unit retry Bundles")
		}
		if len(h.Members) == 0 {
			h.Locale, h.BatchID, kind = m.Locale, m.BatchID, m.UnitKind
		}
		if m.Locale != h.Locale || m.BatchID != h.BatchID || m.UnitKind != kind {
			return nil, h, fmt.Errorf("handoff must not mix locale, batch or Unit kind")
		}
		formal, err := decodeRetranslationManifest(files["formal/manifest.json"])
		if err != nil {
			return nil, h, err
		}
		index := 0
		for i, u := range formal.Units {
			if u.UnitID == retryUnitID(m) {
				index = i + 1
			}
		}
		if index == 0 {
			return nil, h, fmt.Errorf("retry target absent from formal manifest")
		}
		seen[retryUnitID(m)] = true
		for _, data := range files {
			objects[retryHandoffObject(data)] = data
		}
		h.Members = append(h.Members, RetryHandoffMember{
			Index: index, UnitID: retryUnitID(m), Attempt: m.Attempt, BundleSHA256: sum(b),
			InputIdentitySHA256: m.InputIdentitySHA256, ManifestObject: retryHandoffObject(files["manifest.json"]), ExpectedOutput: m.ExpectedOutputs[0],
		})
	}
	sort.Slice(h.Members, func(i, j int) bool { return h.Members[i].Index < h.Members[j].Index })
	entries := []TransportBundleEntry{}
	for p, b := range objects {
		h.Files = append(h.Files, NewTransportBundleFile(p, "", b))
		entries = append(entries, TransportBundleEntry{Path: p, Data: b})
	}
	sort.Slice(h.Files, func(i, j int) bool { return h.Files[i].BundlePath < h.Files[j].BundlePath })
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return nil, h, err
	}
	b, err := WriteDeterministicTransportBundle(append(data, '\n'), entries)
	return b, h, err
}

// ReadRetryHandoff verifies transport integrity and recovers original ZIPs.
// This does NOT declare them current. It remains usable after sibling success
// so pending outputs can retain the actual model input SHA without rebinding.
func ReadRetryHandoff(data []byte) (RetryHandoff, [][]byte, error) {
	var h RetryHandoff
	files, err := ReadTransportBundle(data, 8192, 64<<20)
	if err != nil {
		return h, nil, err
	}
	if err := decodeStrictBundleJSON(files["manifest.json"], &h); err != nil {
		return h, nil, err
	}
	if h.Kind != "go-tour-i18n/retry-handoff/v1" || len(h.Members) == 0 || len(h.Members) > MaxRetranslationExportLimit {
		return h, nil, fmt.Errorf("invalid retry handoff")
	}
	if err := ValidateTransportBundleInventory(files, h.Files, true); err != nil {
		return h, nil, err
	}
	bundles := [][]byte{}
	total := int64(0)
	for _, member := range h.Members {
		manifestBytes, ok := files[member.ManifestObject]
		if !ok || retryHandoffObject(manifestBytes) != member.ManifestObject {
			return h, nil, fmt.Errorf("missing canonical retry manifest")
		}
		var m GenerationBundleManifest
		if err := decodeStrictBundleJSON(manifestBytes, &m); err != nil {
			return h, nil, err
		}
		if len(m.Files)+1 > TranslationUnitGenerationBundleMaxFiles {
			return h, nil, fmt.Errorf("retry constituent has too many files")
		}
		size := int64(len(manifestBytes))
		for _, f := range m.Files {
			if f.Size < 0 || f.Size > TranslationUnitGenerationBundleMaxBytes-size {
				return h, nil, fmt.Errorf("retry constituent exceeds transport limit")
			}
			size += f.Size
		}
		if total+size > 64<<20 {
			return h, nil, fmt.Errorf("retry handoff reconstructed inputs exceed 64 MiB")
		}
		entries := []TransportBundleEntry{}
		for _, f := range m.Files {
			b, ok := files["objects/"+f.SHA256+".txt"]
			if !ok {
				return h, nil, fmt.Errorf("missing retry input %s", f.BundlePath)
			}
			entries = append(entries, TransportBundleEntry{Path: f.BundlePath, Data: b})
		}
		b, err := WriteDeterministicTransportBundle(manifestBytes, entries)
		if err != nil {
			return h, nil, err
		}
		if sum(b) != member.BundleSHA256 {
			// Historical handoffs have no encoding field. Restore only the exact
			// recorded Store or Deflate-5 ZIP, never substitute a new invocation.
			b, err = writeTranslationUnitGenerationBundle(manifestBytes, entries)
			if err != nil {
				return h, nil, err
			}
			if sum(b) != member.BundleSHA256 {
				return h, nil, fmt.Errorf("canonical retry ZIP SHA mismatch")
			}
		}
		total += int64(len(b))
		if total > 64<<20 {
			return h, nil, fmt.Errorf("retry handoff reconstructed inputs exceed 64 MiB")
		}
		if _, _, err := readGenerationBundle(b); err != nil {
			return h, nil, err
		}
		bundles = append(bundles, b)
	}
	canonical, _, err := packRetryHandoff(bundles)
	if err != nil {
		return h, nil, err
	}
	if !bytes.Equal(canonical, data) {
		return h, nil, fmt.Errorf("non-canonical retry handoff")
	}
	return h, bundles, nil
}

// Aggregate currentness is all-members-current. Once any member is imported
// the aggregate is stale; remaining original single-Unit identities can still
// be current according to the existing authenticated sibling-history rule.
func VerifyCurrentRetryHandoff(root string, catalog *Catalog, data []byte) (RetryHandoff, error) {
	h, bundles, err := ReadRetryHandoff(data)
	if err != nil {
		return h, err
	}
	for i, b := range bundles {
		if _, err := VerifyCurrentTranslationUnitGenerationBundle(root, catalog, b); err != nil {
			return h, fmt.Errorf("retry handoff member %s is stale: %w", h.Members[i].UnitID, err)
		}
	}
	return h, nil
}
