package sitecontent

import (
	"bytes"
	"fmt"
)

// Corpus consumers receive an already hash/canonical-verified foundation.
// Byte-current dependencies retain the pre-adoption path. Any drift requires
// the current, canonical adoption of this exact immutable foundation, whose
// verifier reconstructs the full source/slot/protection and corpus projection.
// No path or arbitrary changed hash grants compatibility by itself.
func verifyShellParserDependenciesThroughAdoption(root string, foundation Reference, a ShellSuccessorAuthority) error {
	expected, err := shellAuthorityReference(&a)
	copy := a
	copy.Identity = ""
	if err != nil || expected != foundation || a.Identity != identity(copy) {
		return fmt.Errorf("shell parser foundation binding mismatch")
	}
	drift := ""
	for _, ref := range a.ParserReferences {
		raw, err := readRegular(root, ref.Path)
		if err != nil {
			return err
		}
		if digest(raw) != ref.SHA256 && drift == "" {
			drift = ref.Path
		}
	}
	if drift == "" {
		return checkShellParserDependencies(root, a)
	}
	b, err := readRegular(root, ShellAdoptionCurrent)
	if err != nil {
		return fmt.Errorf("shell parser authority stale: %s: current adoption required: %w", drift, err)
	}
	var ref Reference
	if err := StrictJSON(b, &ref); err != nil {
		return err
	}
	canonical, err := Encode(ref)
	if err != nil || !bytes.Equal(b, canonical) {
		return fmt.Errorf("shell adoption current pointer noncanonical")
	}
	adoption, err := CheckShellAdoption(root, ref)
	if err != nil {
		return err
	}
	if adoption.Foundation != foundation {
		return fmt.Errorf("shell adoption does not cover exact parser foundation")
	}
	b, err = readRegularReference(root, ref)
	if err != nil {
		return err
	}
	canonical, err = Encode(adoption)
	if err != nil || !bytes.Equal(b, canonical) {
		return fmt.Errorf("shell adoption evidence noncanonical")
	}
	return nil
}

// The source foundation consumes the extractor's exact finite source/slot
// projection. It does not consume every possible future Page binding reader.
// Current validator/reconstruction authority remains separately hashed by the
// TU adapters. Module/library dependencies remain byte exact.
func shellParserSemanticIdentity(sources []ShellLanguageSource) string {
	return identity(struct {
		Contract string
		Sources  []ShellLanguageSource
	}{ShellExtractionContract, sources})
}

func checkShellParserDependencies(root string, a ShellSuccessorAuthority) error {
	if a.ParserSemanticIdentity != "" && a.ParserSemanticIdentity != shellParserSemanticIdentity(a.Sources) {
		return fmt.Errorf("shell parser semantic dependency identity changed")
	}
	projected := make([]ShellLanguageSource, 0, len(a.Sources))
	for _, s := range a.Sources {
		raw, err := readRegular(root, s.Path)
		if err != nil || digest(raw) != s.SourceSHA {
			return fmt.Errorf("shell parser semantic source changed: %s", s.Path)
		}
		p, err := ExtractShellLanguageSource(s.Path, raw)
		if err != nil {
			return err
		}
		projected = append(projected, *p)
	}
	if shellParserSemanticIdentity(projected) != shellParserSemanticIdentity(a.Sources) {
		return fmt.Errorf("shell parser semantic source/slot/protection drift")
	}
	for _, ref := range a.ParserReferences {
		switch ref.Path {
		case "internal/sitecontent/shell_language_source.go", "internal/sitecontent/shell_successor.go", "internal/sitecontent/shellyaml/parse.go", "internal/sitecontent/formal_page.go", "internal/sitecontent/units.go", "internal/sitecontent/linguistic.go", "internal/sitecontent/structural_binding.go", "internal/sitecontent/teaching.go", "internal/i18n/go_example_protect.go":
			// The exact projection above replaces only these declared source
			// parser dependencies. Recorded file SHA values remain provenance,
			// including for legacy foundations without the additive field.
			if _, err := readRegular(root, ref.Path); err != nil {
				return err
			}
		default:
			if _, err := readRegularReference(root, ref); err != nil {
				return err
			}
		}
	}
	return nil
}
