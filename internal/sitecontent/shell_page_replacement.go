package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Existing shell foundations already freeze the complete historical Page and
// slot encoding inventory. Consume that authority directly, rather than ask a
// maintainer to synthesize another historical-document JSON file.
func readReplacementRepresentations(root string, raw []byte, ref i18n.GlossaryArchiveReference) (*HistoricalPageRepresentations, error) {
	var header struct {
		Schema string `json:"schema"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, err
	}
	if header.Schema != ShellSuccessorSchema {
		var frozen HistoricalPageRepresentations
		if err := StrictJSON(raw, &frozen); err != nil {
			return nil, err
		}
		return &frozen, nil
	}
	var foundation ShellSuccessorAuthority
	if err := StrictJSON(raw, &foundation); err != nil {
		return nil, err
	}
	if err := verifyShellParserDependenciesThroughAdoption(root, Reference{ref.Path, ref.SHA256}, foundation); err != nil {
		return nil, err
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	if g.ShellAdoption == nil {
		return nil, fmt.Errorf("shell replacement requires current adoption")
	}
	adoption, err := CheckShellAdoption(root, *g.ShellAdoption)
	if err != nil {
		return nil, err
	}
	if adoption.Foundation != (Reference{ref.Path, ref.SHA256}) || adoption.Package.ID != foundation.Package {
		return nil, fmt.Errorf("shell replacement foundation/adoption mismatch")
	}
	frozen := &HistoricalPageRepresentations{Package: adoption.Package.ID, PackageSHA256: adoption.Package.Identity}
	for _, s := range foundation.Sources {
		if s.Page != nil {
			frozen.Documents = append(frozen.Documents, *s.Page)
		}
	}
	return frozen, nil
}

func replacementPageAdapter(adapter i18n.TranslationUnitAdapter) (PageAdapter, error) {
	switch a := adapter.(type) {
	case PageAdapter:
		return a, nil
	case adoptedPageAdapter:
		return a.PageAdapter, nil
	case ShellPageAdapter:
		if a.Source.Page != nil {
			return PageAdapter{*a.Source.Page, a.Raw}, nil
		}
	}
	return PageAdapter{}, fmt.Errorf("replacement requires formal Page adapter")
}

func currentReplacementPage(a PageAdapter) (*Document, error) {
	if a.Document.Contract == ShellExtractionContract {
		s, err := ExtractShellPageRepresentation(a.Document.Path, a.Source)
		if err != nil {
			return nil, err
		}
		if s.Page == nil {
			return nil, fmt.Errorf("current shell replacement requires formal Page")
		}
		return s.Page, nil
	}
	return ParseWorkflowDocument(documentSource(a.Document), a.Source, a.Document.StableIndex)
}

func historicalReplacementAdapter(current i18n.TranslationUnitAdapter, d Document) (i18n.TranslationUnitAdapter, error) {
	a, err := replacementPageAdapter(current)
	if err != nil {
		return nil, err
	}
	if _, ok := current.(ShellPageAdapter); ok {
		// The frozen shell reader owns historical slot roles and encodings.
		// A Document alone must not invent them from the current representation.
		s, err := ExtractShellLanguageSource(d.Path, a.Source)
		if err != nil || s.Page == nil || !reflect.DeepEqual(*s.Page, d) {
			return nil, fmt.Errorf("historical shell Page representation mismatch")
		}
		return ShellPageAdapter{*s, bytes.Clone(a.Source)}, nil
	}
	return PageAdapter{d, bytes.Clone(a.Source)}, nil
}

func validateReplacementDocuments(docs []Document, current bool) error {
	if len(docs) == 0 {
		return fmt.Errorf("empty replacement documents")
	}
	paths, ids := map[string]bool{}, map[string]bool{}
	for _, d := range docs {
		if paths[d.Path] || len(d.Units) != 1 || ids[d.Units[0].ID] {
			return fmt.Errorf("duplicate/invalid replacement Page")
		}
		paths[d.Path], ids[d.Units[0].ID] = true, true
		if d.Contract != ShellExtractionContract {
			if err := validateDocumentRepresentations([]Document{d}, current); err != nil {
				return err
			}
			continue
		}
		raw, err := restoreBlock(d.Units[0], d.Units[0].Source)
		if err != nil || digest([]byte(raw)) != d.SourceSHA || d.UnitScope != PageContract {
			return fmt.Errorf("replacement shell Page source identity mismatch")
		}
		var source *ShellLanguageSource
		if current {
			source, err = ExtractShellPageRepresentation(d.Path, []byte(raw))
		} else {
			source, err = ExtractShellLanguageSource(d.Path, []byte(raw))
		}
		if err != nil || source.Page == nil || !reflect.DeepEqual(*source.Page, d) {
			return fmt.Errorf("replacement shell Page parser identity mismatch")
		}
	}
	return nil
}

func (a ShellPageAdapter) ReconcileRepresentation(adapter i18n.TranslationUnitAdapter, raw []byte) ([]byte, error) {
	old, ok := adapter.(ShellPageAdapter)
	if !ok || !bytes.Equal(old.Raw, a.Raw) || old.Source.Page == nil || a.Source.Page == nil {
		return nil, fmt.Errorf("shell replacement predecessor source mismatch")
	}
	before, err := old.Restore(string(raw))
	if err != nil {
		return nil, err
	}
	// Use the existing source-owned slot/structure and inline binding reader.
	// It checks exact framing, URLs/pairs, unique literal inventory and encoding;
	// it never discovers translation slots by parsing translated language.
	values, err := shellTargetValues(a.Source, a.Raw, before)
	if err != nil {
		return nil, err
	}
	u := a.Source.Page.Units[0]
	var output strings.Builder
	at := 0
	for i, slot := range a.Source.Slots {
		if at >= len(u.Protected) || u.Protected[at].Role != "boundary" {
			return nil, fmt.Errorf("shell replacement Page boundary mismatch")
		}
		output.WriteString(u.Protected[at].Token)
		at++
		pairs := []string{}
		for _, p := range slot.Unit.Protected {
			if at >= len(u.Protected) || p.Raw != u.Protected[at].Raw {
				return nil, fmt.Errorf("shell replacement slot protection mismatch")
			}
			pairs = append(pairs, p.Token, u.Protected[at].Token)
			at++
		}
		output.WriteString(strings.NewReplacer(pairs...).Replace(values[i]))
	}
	if at != len(u.Protected)-1 || u.Protected[at].Role != "boundary" {
		return nil, fmt.Errorf("shell replacement final boundary mismatch")
	}
	output.WriteString(u.Protected[at].Token)
	next := []byte(output.String())
	after, err := a.Restore(string(next))
	if err != nil || !bytes.Equal(before, after) {
		return nil, fmt.Errorf("shell replacement changes reviewed bytes: %v", err)
	}
	return next, nil
}
