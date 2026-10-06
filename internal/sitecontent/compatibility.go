package sitecontent

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type ParsedContext struct {
	ID           string `json:"id"`
	Source       string `json:"source"`
	Target       string `json:"target"`
	StructureSHA string `json:"structure_sha256"`
	Route        string `json:"route"`
}
type PackageCompatibility struct {
	Schema     string                       `json:"schema"`
	Locale     string                       `json:"locale"`
	Package    string                       `json:"package"`
	Parser     string                       `json:"parser"`
	Old        Reference                    `json:"old_glossary"`
	New        Reference                    `json:"new_glossary"`
	Review     Reference                    `json:"new_full_review"`
	Delta      []i18n.GlossarySemanticDelta `json:"delta"`
	Contexts   []ParsedContext              `json:"contexts"`
	Affected   []string                     `json:"affected"`
	Compatible []string                     `json:"compatible"`
	Prior      *Reference                   `json:"prior,omitempty"`
	Identity   string                       `json:"identity_sha256"`
}

func parsedContexts(values map[string]selection) []ParsedContext {
	ids := []string{}
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	contexts := []ParsedContext{}
	for _, id := range ids {
		s := values[id]
		contexts = append(contexts, ParsedContext{id, s.Unit.Source, s.Target.Text, s.Document.StructureSHA, s.Document.Route})
	}
	return contexts
}
func archivedGlossary(root, locale, sha string) (Reference, []byte, error) {
	if !component(locale) || !validSHA(sha) {
		return Reference{}, nil, fmt.Errorf("invalid glossary archive identity")
	}
	name := "data/glossary-history/" + locale + "/" + sha + ".yaml"
	b, err := readRegular(root, name)
	if err != nil {
		return Reference{}, nil, err
	}
	if digest(b) != sha {
		return Reference{}, nil, fmt.Errorf("glossary archive hash mismatch")
	}
	return Reference{name, sha}, b, nil
}
func buildPackageCompatibility(root, locale, pkg, oldSHA, newSHA string, review Reference, contexts []ParsedContext, prior *Reference) (*PackageCompatibility, error) {
	old, oldBytes, err := archivedGlossary(root, locale, oldSHA)
	if err != nil {
		return nil, err
	}
	next, newBytes, err := archivedGlossary(root, locale, newSHA)
	if err != nil {
		return nil, err
	}
	if oldSHA == newSHA {
		return nil, fmt.Errorf("compatibility edge requires changed glossary")
	}
	if err := i18n.ValidateParsedGlossaryReview(root, locale, newSHA, i18n.GlossaryArchiveReference{Path: review.Path, SHA256: review.SHA256}); err != nil {
		return nil, err
	}
	delta, err := i18n.ParsedGlossaryDelta(locale, oldBytes, newBytes)
	if err != nil {
		return nil, err
	}
	e := &PackageCompatibility{Schema: "go-learning/parsed-glossary-compatibility/v1", Locale: locale, Package: pkg, Parser: UnitContract, Old: old, New: next, Review: review, Delta: delta, Contexts: contexts, Affected: []string{}, Compatible: []string{}, Prior: prior}
	last := ""
	for _, c := range contexts {
		if c.ID <= last || !validSHA(c.ID) || !validSHA(c.StructureSHA) {
			return nil, fmt.Errorf("invalid compatibility context")
		}
		last = c.ID
		ok, _ := i18n.ParsedGlossaryImpact(c.Source, c.Target, delta)
		if ok {
			e.Compatible = append(e.Compatible, c.ID)
		} else {
			e.Affected = append(e.Affected, c.ID)
		}
	}
	if len(contexts) == 0 {
		return nil, fmt.Errorf("empty compatibility inventory")
	}
	e.Identity = identity(*e)
	return e, nil
}

// AssessPackageCompatibility shares V2-B normalization and semantic impact;
// its parser-derived exact inventory is isolated per locale/package.
func AssessPackageCompatibility(root, locale, pkg, oldSHA string, refs []Reference, prior *Reference) (*PackageCompatibility, string, error) {
	_, _, docs, _, glossary, err := currentPackage(root, locale, pkg)
	if err != nil {
		return nil, "", err
	}
	archive, err := i18n.ArchiveCurrentGlossary(root, locale)
	if err != nil {
		return nil, "", err
	}
	if archive.SHA256 != digest(glossary) {
		return nil, "", fmt.Errorf("glossary changed during assess")
	}
	review, err := i18n.CurrentParsedGlossaryReview(root, locale, archive.SHA256)
	if err != nil {
		return nil, "", err
	}
	values, err := selections(root, locale, pkg, refs, docs)
	if err != nil {
		return nil, "", err
	}
	if len(values) != len(unitsByID(docs)) {
		return nil, "", fmt.Errorf("compatibility inventory must contain exact complete package")
	}
	e, err := buildPackageCompatibility(root, locale, pkg, oldSHA, archive.SHA256, Reference{review.Path, review.SHA256}, parsedContexts(values), prior)
	if err != nil {
		return nil, "", err
	}
	if prior != nil {
		var previous PackageCompatibility
		if err := readReference(root, *prior, &previous); err != nil {
			return nil, "", err
		}
		if previous.New.SHA256 != oldSHA || previous.Locale != locale || previous.Package != pkg {
			return nil, "", fmt.Errorf("discontinuous prior lineage")
		}
	}
	name := workflowPath(locale, pkg, "compatibility", e.Identity)
	if err := saveImmutable(root, name, e); err != nil {
		return nil, "", err
	}
	return e, name, nil
}

func requirePackageGlossaryCompatibility(root, locale, pkg, oldSHA, newSHA string, s selection, refs []Reference) error {
	if oldSHA == newSHA {
		return nil
	}
	byOld := map[string]*PackageCompatibility{}
	seenRefs := map[string]bool{}
	var load func(Reference) error
	load = func(ref Reference) error {
		if seenRefs[ref.Path] {
			return nil
		}
		seenRefs[ref.Path] = true
		var e PackageCompatibility
		if err := readReference(root, ref, &e); err != nil {
			return err
		}
		if e.Locale != locale || e.Package != pkg || ref.Path != workflowPath(locale, pkg, "compatibility", e.Identity) {
			return fmt.Errorf("compatibility scope/path mismatch")
		}
		want, err := buildPackageCompatibility(root, locale, pkg, e.Old.SHA256, e.New.SHA256, e.Review, e.Contexts, e.Prior)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(e, *want) {
			return fmt.Errorf("compatibility identity/classification mismatch")
		}
		if previous := byOld[e.Old.SHA256]; previous != nil && previous.Identity != e.Identity {
			return fmt.Errorf("ambiguous glossary lineage")
		}
		byOld[e.Old.SHA256] = &e
		if e.Prior != nil {
			if err := load(*e.Prior); err != nil {
				return err
			}
			var prior PackageCompatibility
			if err := readReference(root, *e.Prior, &prior); err != nil {
				return err
			}
			if prior.New.SHA256 != e.Old.SHA256 {
				return fmt.Errorf("discontinuous glossary lineage")
			}
		}
		return nil
	}
	for _, ref := range refs {
		if err := load(ref); err != nil {
			return err
		}
	}
	visited := map[string]bool{}
	for oldSHA != newSHA {
		if visited[oldSHA] {
			return fmt.Errorf("glossary lineage cycle")
		}
		visited[oldSHA] = true
		e := byOld[oldSHA]
		if e == nil {
			return fmt.Errorf("missing glossary compatibility evidence")
		}
		found := false
		for _, c := range e.Contexts {
			if c.ID == s.Unit.ID {
				if c.Source != s.Unit.Source || c.Target != s.Target.Text || c.StructureSHA != s.Document.StructureSHA || c.Route != s.Document.Route {
					return fmt.Errorf("compatibility context stale")
				}
				found = true
			}
		}
		if !found {
			return fmt.Errorf("missing exact compatibility context")
		}
		compatible := false
		for _, id := range e.Compatible {
			if id == s.Unit.ID {
				compatible = true
			}
		}
		if !compatible {
			return fmt.Errorf("affected unit cannot carry historical A")
		}
		oldSHA = e.New.SHA256
	}
	return nil
}
