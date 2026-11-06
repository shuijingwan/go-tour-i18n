package sitecontent

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const StructuredTransportProfile = "go-learning/structured-generation-projection/v1"
const structuredTransportContract = "docs/STRUCTURED_GENERATION_TRANSPORT.md"

// Only these two exact archive families are opaque machine-validation inputs.
// Receipts, current corpus/glossary/Review, source and previous target bytes
// remain self-contained Generation inputs. This is not a general ZIP filter.
func structuredValidationArchive(locale string, ref Reference) bool {
	if !validSHA(ref.SHA256) {
		return false
	}
	for _, dir := range []string{"data/locale-language/" + locale + "/structured-bundles/", "data/unified-glossary-reviews/" + locale + "-bundles/"} {
		if ref.Path == dir+ref.SHA256+".zip" {
			return true
		}
	}
	return false
}

// allowed comes only from the fully validated historical Generation/Review
// nodes above, not from filename discovery. Every selected archive is freshly
// observed by the caller's dependency guard and bound by its exact raw SHA.
func projectStructuredArchiveTransport(m *StructuredBundle, entries []i18n.TransportBundleEntry, allowed map[Reference]bool) ([]i18n.TransportBundleEntry, error) {
	byPath := map[string][]byte{}
	for _, e := range entries {
		if _, ok := byPath[e.Path]; ok {
			return nil, fmt.Errorf("duplicate structured payload: %s", e.Path)
		}
		byPath[e.Path] = e.Data
	}
	kept := []i18n.TransportBundleFile{}
	omitted := map[string]bool{}
	refs := []Reference{}
	for _, f := range m.Files {
		ref := Reference{f.RepositoryPath, f.SHA256}
		if !allowed[ref] {
			kept = append(kept, f)
			continue
		}
		data, ok := byPath[f.BundlePath]
		if !structuredValidationArchive(m.Plan.Locale, ref) || f.BundlePath != "inputs/"+ref.Path || !ok || int64(len(data)) != f.Size || digest(data) != ref.SHA256 || omitted[f.BundlePath] {
			return nil, fmt.Errorf("invalid structured archive projection: %s", f.BundlePath)
		}
		omitted[f.BundlePath] = true
		refs = append(refs, ref)
	}
	if len(refs) != len(allowed) {
		return nil, fmt.Errorf("structured archive projection exact set mismatch")
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	if len(refs) == 0 {
		refs = nil
	}
	out := []i18n.TransportBundleEntry{}
	for _, e := range entries {
		if !omitted[e.Path] {
			out = append(out, e)
		}
	}
	m.TransportProfile, m.ValidationInputs, m.Files = StructuredTransportProfile, refs, kept
	return out, nil
}

func validateStructuredTransportProjection(m *StructuredBundle, files map[string][]byte) error {
	if m.TransportProfile == "" && len(m.ValidationInputs) == 0 {
		return nil // Immutable legacy transport retains its existing reader.
	}
	if m.TransportProfile != StructuredTransportProfile || len(files["inputs/"+structuredTransportContract]) == 0 {
		return fmt.Errorf("unknown/incomplete structured transport projection")
	}
	prior := ""
	for _, ref := range m.ValidationInputs {
		if !structuredValidationArchive(m.Plan.Locale, ref) || ref.Path <= prior {
			return fmt.Errorf("noncanonical structured validation input set")
		}
		if _, present := files["inputs/"+ref.Path]; present {
			return fmt.Errorf("structured validation archive duplicated as payload")
		}
		prior = ref.Path
	}
	found := false
	for _, f := range m.Files {
		found = found || (f.BundlePath == "inputs/"+structuredTransportContract && f.RepositoryPath == structuredTransportContract)
	}
	if !found {
		return fmt.Errorf("missing structured transport contract inventory")
	}
	return nil
}

// An audited legacy→projection transition preserves every semantic/context
// member and binds exactly the same archive hashes. It is not an exception for
// arbitrary missing payload or changed machine evidence. Oversized legacy ZIPs
// still fail the original 32 MiB gate before this comparison can be reached.
func compatibleStructuredTransportProjection(old, current StructuredBundle) (map[string]bool, error) {
	omitted := map[string]bool{}
	if old.TransportProfile == current.TransportProfile {
		if !reflect.DeepEqual(old.ValidationInputs, current.ValidationInputs) {
			return nil, fmt.Errorf("structured validated repository inputs drift")
		}
		return omitted, nil
	}
	if old.TransportProfile != "" || len(old.ValidationInputs) != 0 || current.TransportProfile != StructuredTransportProfile {
		return nil, fmt.Errorf("unknown structured transport transition")
	}
	for _, ref := range current.ValidationInputs {
		matches := 0
		for _, f := range old.Files {
			if f.BundlePath == "inputs/"+ref.Path && f.RepositoryPath == ref.Path && f.SHA256 == ref.SHA256 && structuredValidationArchive(current.Plan.Locale, ref) {
				matches++
				omitted[f.BundlePath] = true
			}
		}
		if matches != 1 {
			return nil, fmt.Errorf("legacy archive is not exact projected validation input: %s", ref.Path)
		}
	}
	// Non-archive old members remain in the ordinary exact semantic comparison.
	for name := range omitted {
		if !strings.HasPrefix(name, "inputs/") {
			return nil, fmt.Errorf("invalid projected archive path")
		}
	}
	return omitted, nil
}

// Source-owned content replacement keeps the canonical source coordinate in
// the Bundle, while the repository binding is its immutable successor body.
// The exact receipt and SHA authorize that alias; both sides must still have
// identical repository bindings and payload in the semantic comparison.
func structuredSourcePayloadBinding(m StructuredBundle, files map[string][]byte, f i18n.TransportBundleFile) bool {
	for _, a := range m.Plan.Assets {
		if a.Document == nil || a.Assessment == nil || a.Assessment.SourceSuccessor == nil || f.BundlePath != "source/"+a.Document.Path || f.SHA256 != a.Document.SourceSHA {
			continue
		}
		ref := *a.Assessment.SourceSuccessor
		raw := files["inputs/"+ref.Path]
		if digest(raw) != ref.SHA256 {
			return false
		}
		var authority StructuredSourceSuccessor
		if StrictJSON(raw, &authority) != nil {
			return false
		}
		return authority.Previous.Path == a.Document.Path && authority.Body.Path == f.RepositoryPath && authority.Body.SHA256 == f.SHA256
	}
	return false
}
