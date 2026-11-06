package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const StructuredInputReopenSchema = "go-learning/structured-input-reopen/v1"
const semanticInputExpansion = "semantic-input-expansion"
const inputExpansionReason = "verified semantic input expansion"

// This immutable machine authorization schedules a whole new asset invocation.
// It grants neither target bytes nor language/Surface acceptance. Semantic atoms
// retain exact source coordinates, field context and complete protection roles.
type StructuredInputReopen struct {
	Schema            string    `json:"schema"`
	Locale            string    `json:"locale"`
	Package           string    `json:"package"`
	PackageSHA        string    `json:"package_sha256"`
	AssetID           string    `json:"asset_id"`
	Reason            string    `json:"reason"`
	Closure           Reference `json:"baseline_closure"`
	Generation        Reference `json:"baseline_generation"`
	Surface           Reference `json:"baseline_surface"`
	Glossary          Reference `json:"current_glossary"`
	Review            Reference `json:"current_unified_review"`
	SourcePath        string    `json:"source_path"`
	TargetPath        string    `json:"target_path"`
	OldSourceSHA      string    `json:"old_asset_source_sha256"`
	CurrentSourceSHA  string    `json:"current_asset_source_sha256"`
	OldRawSHA         string    `json:"old_raw_source_sha256"`
	CurrentRawSHA     string    `json:"current_raw_source_sha256"`
	OldStructureSHA   string    `json:"old_structure_sha256"`
	StructureSHA      string    `json:"current_structure_sha256"`
	OldDocumentID     string    `json:"old_document_identity_sha256"`
	DocumentID        string    `json:"current_document_identity_sha256"`
	OldUnitIDs        []string  `json:"old_unit_ids"`
	CurrentUnitIDs    []string  `json:"current_unit_ids"`
	AddedUnitIDs      []string  `json:"added_unit_ids"`
	RemovedUnitIDs    []string  `json:"removed_unit_ids"`
	ChangedUnitIDs    []string  `json:"changed_unit_ids"`
	OldAtomCount      int       `json:"old_semantic_atom_count"`
	CurrentAtomCount  int       `json:"current_semantic_atom_count"`
	AddedAtoms        []Unit    `json:"added_semantic_atoms"`
	RemovedAtoms      []Unit    `json:"removed_semantic_atoms"`
	AcceptedTargetSHA string    `json:"accepted_target_sha256"`
	Identity          string    `json:"identity_sha256"`
}

type StructuredInputReopenResult struct {
	Ready              bool                  `json:"ready"`
	Applied            bool                  `json:"applied"`
	Reason             string                `json:"reason"`
	RawSourceChanged   bool                  `json:"raw_source_changed"`
	GenerationRequired bool                  `json:"generation_required"`
	OldUnits           int                   `json:"old_units"`
	CurrentUnits       int                   `json:"current_units"`
	Added              int                   `json:"added"`
	Removed            int                   `json:"removed"`
	Changed            int                   `json:"changed"`
	CurrentState       string                `json:"current_asset_state"`
	Receipt            StructuredInputReopen `json:"receipt"`
	Reference          Reference             `json:"receipt_reference"`
	ProjectedState     string                `json:"projected_asset_state"`
	ProjectedExpected  []string              `json:"projected_expected"`
}

func inputReopenDirectory(locale, pkg, asset string) string {
	return "data/locale-language/" + locale + "/structured-input-reopens/" + identity([]string{pkg, asset})
}
func inputReopenPath(r StructuredInputReopen) string {
	return inputReopenDirectory(r.Locale, r.Package, r.AssetID) + "/" + r.Identity + ".json"
}
func inputReopenReference(r StructuredInputReopen) Reference {
	b, _ := Encode(r)
	return Reference{inputReopenPath(r), digest(b)}
}
func validateInputReopenIdentity(r StructuredInputReopen, ref Reference) error {
	x := r
	x.Identity = ""
	if r.Schema != StructuredInputReopenSchema || r.Reason != semanticInputExpansion || !component(r.Locale) || !component(r.Package) || r.Identity != identity(x) || ref != inputReopenReference(r) {
		return fmt.Errorf("invalid structured input reopen identity/canonical path")
	}
	return nil
}

// Both event-time sides are verified using their persisted representation.
// Current readers additionally require today's canonical parser. No remap is
// inferred: every existing Unit, including ID and protection, must be exact.
func semanticExpansion(old, current Document, oldRaw, currentRaw []byte) ([]Unit, error) {
	if old.Contract != UnitContract || current.Contract != UnitContract || old.UnitScope != "" || current.UnitScope != "" || !bytes.Equal(oldRaw, currentRaw) || digest(oldRaw) != old.SourceSHA || digest(currentRaw) != current.SourceSHA {
		return nil, fmt.Errorf("semantic-input-expansion requires unchanged raw source and supported structured parser")
	}
	if err := verifyFrozenStructuredRepresentation(old, oldRaw); err != nil {
		return nil, err
	}
	if err := verifyFrozenStructuredRepresentation(current, currentRaw); err != nil {
		return nil, err
	}
	a, b := old, current
	a.Units, b.Units, a.StructureSHA, b.StructureSHA = nil, nil, "", ""
	if !reflect.DeepEqual(a, b) {
		return nil, fmt.Errorf("semantic-input-expansion source/parser/dependency context changed")
	}
	anchor := func(u Unit) (string, error) {
		end := strings.LastIndex(u.Locator, ":")
		if end < 0 {
			return "", fmt.Errorf("ambiguous semantic atom locator")
		}
		base := strings.TrimSuffix(u.Locator[:end], ":"+u.SourceSHA)
		return identity(struct {
			Locator    string
			Start, End int
		}{base, u.Start, u.End}), nil
	}
	before, after := map[string]Unit{}, map[string]Unit{}
	for _, doc := range []Document{old, current} {
		seen := map[string]bool{}
		for _, u := range doc.Units {
			key, err := anchor(u)
			if err != nil || seen[key] || !hasLanguage(u.Source) {
				return nil, fmt.Errorf("ambiguous/non-localizable semantic atom")
			}
			seen[key] = true
		}
	}
	for _, u := range old.Units {
		before[u.ID] = u
	}
	for _, u := range current.Units {
		after[u.ID] = u
	}
	if len(before) != len(old.Units) || len(after) != len(current.Units) {
		return nil, fmt.Errorf("duplicate structured Unit identity")
	}
	for _, u := range old.Units {
		if next, ok := after[u.ID]; !ok || !reflect.DeepEqual(u, next) {
			return nil, fmt.Errorf("semantic-input-expansion removed/changed existing Unit: %s", u.ID)
		}
	}
	added := []Unit{}
	for _, u := range current.Units {
		if _, ok := before[u.ID]; !ok {
			added = append(added, u)
		}
	}
	if len(added) == 0 || old.StructureSHA == current.StructureSHA {
		return nil, fmt.Errorf("semantic-input-expansion requires a strict semantic superset; no ID-only churn")
	}
	return added, nil
}

// Baseline data comes exclusively from the shared verified completion graph.
// The same deterministic constructor is used by preflight, current readers and
// frozen Generation verification; it performs no I/O and grants no target.
func buildInputReopen(p StructuredPlan, a StructuredAsset, b *structuredBaseline, raw []byte, packageSHA string) (*StructuredInputReopen, error) {
	if a.Document == nil || a.Package != b.Closure.Package || p.Locale != b.Closure.Locale || packageSHA != b.Closure.PackageSHA || a.ID != "source:"+a.Document.Path || !reflect.DeepEqual(a.SourcePaths, []string{a.Document.Path}) || a.TargetPath != "locales/"+p.Locale+"/structured/"+a.Document.Path {
		return nil, fmt.Errorf("structured input reopen exact locale/package/asset/source/target scope required")
	}
	gen := b.Owners[a.Document.Path]
	r := b.Generations[gen.Path]
	if r == nil || b.Surface.Path == "" || r.Plan.GlossarySHA != p.GlossarySHA || r.Plan.Review != p.Review {
		return nil, fmt.Errorf("structured input reopen baseline/glossary/Unified Review drift")
	}
	if x := a.Assessment; x == nil || x.Closure != b.ClosureRef || x.Generation != gen || x.Surface != b.Surface || x.SourceSHA != a.SourceSHA || x.TargetSHA != digest(b.Files[a.Document.Path]) || x.OldGlossarySHA != r.Plan.GlossarySHA || x.GlossarySHA != p.GlossarySHA || x.Review != p.Review {
		return nil, fmt.Errorf("structured input reopen baseline assessment binding mismatch")
	}
	var old *StructuredAsset
	var output *StructuredTarget
	for i := range r.Plan.Assets {
		if r.Plan.Assets[i].ID == a.ID {
			old = &r.Plan.Assets[i]
		}
	}
	for i := range r.Output.Outputs {
		if r.Output.Outputs[i].Path == a.TargetPath {
			output = &r.Output.Outputs[i]
		}
	}
	if old == nil || output == nil || old.Document == nil || old.Package != a.Package || old.SourceSHA != a.SourceSHA || old.TargetPath != a.TargetPath || !reflect.DeepEqual(old.SourcePaths, a.SourcePaths) {
		return nil, fmt.Errorf("structured input reopen contributor/baseline asset drift")
	}
	oldRaw := b.Inputs[gen.Path]["source/"+a.Document.Path]
	added, err := semanticExpansion(*old.Document, *a.Document, oldRaw, raw)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, s := range output.Slots {
		if _, ok := values[s.ID]; ok {
			return nil, fmt.Errorf("duplicate predecessor target slot")
		}
		values[s.ID] = s.Text
	}
	accepted, err := reconstructFrozenStructuredRepresentation(*old.Document, oldRaw, values)
	if err != nil || !bytes.Equal(accepted, b.Files[a.Document.Path]) || len(values) != len(old.Document.Units) || len(values) >= len(a.Document.Units) {
		return nil, fmt.Errorf("structured input reopen accepted target/old output coverage mismatch")
	}
	x := &StructuredInputReopen{
		Schema: StructuredInputReopenSchema, Locale: p.Locale, Package: a.Package,
		PackageSHA: packageSHA, AssetID: a.ID, Reason: semanticInputExpansion,
		Closure: b.ClosureRef, Generation: gen, Surface: b.Surface,
		Glossary: Reference{"locales/" + p.Locale + "/glossary.yaml", p.GlossarySHA}, Review: p.Review,
		SourcePath: a.Document.Path, TargetPath: a.TargetPath,
		OldSourceSHA: old.SourceSHA, CurrentSourceSHA: a.SourceSHA,
		OldRawSHA: old.Document.SourceSHA, CurrentRawSHA: a.Document.SourceSHA,
		OldStructureSHA: old.Document.StructureSHA, StructureSHA: a.Document.StructureSHA,
		OldDocumentID: identity(*old.Document), DocumentID: identity(*a.Document),
		OldUnitIDs: []string{}, CurrentUnitIDs: []string{}, AddedUnitIDs: []string{},
		RemovedUnitIDs: []string{}, ChangedUnitIDs: []string{},
		OldAtomCount: len(old.Document.Units), CurrentAtomCount: len(a.Document.Units),
		AddedAtoms: added, RemovedAtoms: []Unit{}, AcceptedTargetSHA: digest(accepted),
	}
	for _, u := range old.Document.Units {
		x.OldUnitIDs = append(x.OldUnitIDs, u.ID)
	}
	for _, u := range a.Document.Units {
		x.CurrentUnitIDs = append(x.CurrentUnitIDs, u.ID)
	}
	for _, u := range added {
		x.AddedUnitIDs = append(x.AddedUnitIDs, u.ID)
	}
	x.Identity = identity(*x)
	return x, nil
}

func applyInputReopenAssessment(x *StructuredAssessment, r StructuredInputReopen) {
	ref := inputReopenReference(r)
	x.Classification, x.Reason, x.InputReopen, x.Identity = "affected", inputExpansionReason, &ref, ""
	x.Identity = identity(*x)
}

// Every entry is strict/self-bound; only this exact current Document may grant
// authority. Receipts for earlier Documents remain historical and cannot grant
// future eligibility. Two claims for today's input are a hard conflict.
func (v *closureValidation) findInputReopen(root string, p StructuredPlan, a StructuredAsset, b *structuredBaseline) (*StructuredInputReopen, error) {
	dir := inputReopenDirectory(p.Locale, a.Package, a.ID)
	if _, err := readRegular(root, dir+"/.inventory"); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	if err := v.directory(dir); err != nil {
		return nil, err
	}
	var found *StructuredInputReopen
	for _, name := range v.dirs[dir] {
		full := path.Join(dir, name)
		raw, err := v.readRegular(root, full)
		if err != nil {
			return nil, err
		}
		var r StructuredInputReopen
		if err := StrictJSON(raw, &r); err != nil {
			return nil, err
		}
		if err := validateInputReopenIdentity(r, Reference{full, digest(raw)}); err != nil {
			return nil, err
		}
		if r.Locale != p.Locale || r.Package != a.Package || r.AssetID != a.ID {
			return nil, fmt.Errorf("cross-scope structured input reopen")
		}
		if r.DocumentID != identity(*a.Document) {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("conflicting current structured input reopen receipts")
		}
		g, err := v.loadCurrent(root)
		if err != nil {
			return nil, err
		}
		pkg, err := packageByID(g, a.Package)
		if err != nil {
			return nil, err
		}
		_, raws, err := currentStructuredDocuments(root, g, []StructuredAsset{a})
		if err != nil {
			return nil, err
		}
		if err := v.checkInputReopenCurrent(root, p, a, raws[a.Document.Path]); err != nil {
			return nil, err
		}
		want, err := buildInputReopen(p, a, b, raws[a.Document.Path], pkg.Identity)
		if err != nil {
			return nil, err
		}
		if !reflect.DeepEqual(want, &r) {
			return nil, fmt.Errorf("structured input reopen baseline/current input binding mismatch")
		}
		found = &r
	}
	return found, nil
}

func (v *closureValidation) checkInputReopenCurrent(root string, p StructuredPlan, a StructuredAsset, raw []byte) error {
	gs, err := v.readRegular(root, "locales/"+p.Locale+"/glossary.yaml")
	if err != nil || digest(gs) != p.GlossarySHA {
		return fmt.Errorf("structured input reopen current glossary drift")
	}
	review, err := v.unifiedReview(p.Locale)
	if err != nil {
		return err
	}
	if p.Review != (Reference{review.Path, review.SHA256}) {
		return fmt.Errorf("structured input reopen current Unified Review drift")
	}
	actual, err := v.readRegular(root, a.Document.Path)
	if err != nil || !bytes.Equal(actual, raw) {
		return fmt.Errorf("structured input reopen actual raw source drift")
	}
	g, err := v.loadCurrent(root)
	if err != nil {
		return err
	}
	docs, _, err := CurrentPackageDocuments(root, g, a.Package)
	if err != nil {
		return err
	}
	matched := 0
	for _, d := range docs {
		if d.Path == a.Document.Path && reflect.DeepEqual(d, *a.Document) {
			matched++
		}
	}
	if matched != 1 {
		return fmt.Errorf("structured input reopen current canonical Document mismatch")
	}
	corpusRaw, err := v.readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return err
	}
	var corpus i18n.GlossarySourceCorpus
	if err := StrictJSON(corpusRaw, &corpus); err != nil {
		return err
	}
	if err := i18n.ValidateGlossaryCorpus(&corpus); err != nil {
		return err
	}
	if corpus.Identity != p.CorpusSHA || corpus.Identity != i18n.CorpusIdentity(corpus) {
		return fmt.Errorf("structured input reopen current corpus identity drift")
	}
	groups, err := structuredContributorGroups(corpus.Contributors)
	if err != nil {
		return err
	}
	cs := groups[a.ID]
	if len(cs) == 0 || cs[0].Package != a.Package || identity(cs) != a.SourceSHA {
		return fmt.Errorf("structured input reopen current contributor drift")
	}
	return nil
}

func (v *closureValidation) verifyFrozenInputReopen(root string, p StructuredPlan, a StructuredAsset, b *structuredBaseline, fs map[string][]byte) error {
	x := a.Assessment
	ref := *x.InputReopen
	raw := fs["inputs/"+ref.Path]
	if digest(raw) != ref.SHA256 {
		return fmt.Errorf("historical structured input reopen archive mismatch")
	}
	var r StructuredInputReopen
	if err := StrictJSON(raw, &r); err != nil {
		return err
	}
	if err := validateInputReopenIdentity(r, ref); err != nil {
		return err
	}
	if err := v.bind(ref); err != nil {
		return err
	}
	want, err := buildInputReopen(p, a, b, fs["source/"+a.Document.Path], b.Closure.PackageSHA)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(want, &r) || x.Classification != "affected" || x.Reason != inputExpansionReason || a.State != "generation-required" {
		return fmt.Errorf("historical structured input reopen exact binding mismatch")
	}
	return nil
}

// ReopenStructuredInput discovers its exact scope from current registered
// contributors and the active completion. Only Apply saves the canonical receipt.
func ReopenStructuredInput(root, locale, asset string, apply bool) (*StructuredInputReopenResult, error) {
	out := &StructuredInputReopenResult{Reason: semanticInputExpansion}
	err := func() error {
		p, err := PlanStructuredAssets(root, locale)
		if err != nil {
			return err
		}
		var a *StructuredAsset
		for i := range p.Assets {
			if p.Assets[i].ID == asset {
				if a != nil {
					return fmt.Errorf("ambiguous structured asset")
				}
				a = &p.Assets[i]
			}
		}
		if a == nil || a.Document == nil || a.Assessment == nil {
			return fmt.Errorf("completed exact structured asset required")
		}
		out.CurrentState = a.State
		if a.Assessment.Classification != "stale" && a.Assessment.InputReopen == nil {
			return fmt.Errorf("structured input expansion requires stale exact source/input")
		}
		v := newClosureValidation(root)
		g, err := v.loadCurrent(root)
		if err != nil {
			return err
		}
		if _, err := v.readRegular(root, LocalePath(locale)); err != nil {
			return err
		}
		l, err := readLocaleAuthority(root, g, locale)
		if err != nil {
			return err
		}
		var baseline *structuredBaseline
		for _, c := range l.Packages {
			if c.Package == a.Package {
				baseline, err = v.historicalStructuredCompletion(root, locale, c)
				if err != nil {
					return err
				}
			}
		}
		if baseline == nil {
			return fmt.Errorf("completed package baseline required")
		}
		pkg, err := packageByID(g, a.Package)
		if err != nil {
			return err
		}
		_, raw, err := currentStructuredDocuments(root, g, []StructuredAsset{*a})
		if err != nil {
			return err
		}
		if err := v.checkInputReopenCurrent(root, *p, *a, raw[a.Document.Path]); err != nil {
			return err
		}
		r, err := buildInputReopen(*p, *a, baseline, raw[a.Document.Path], pkg.Identity)
		if err != nil {
			return err
		}
		if _, err := v.findInputReopen(root, *p, *a, baseline); err != nil {
			return err
		}
		// Project using the same assessment transition and Plan normalization as
		// the persisted consumer, without writing a receipt or exported Bundle.
		var projected StructuredPlan
		encoded, _ := json.Marshal(p)
		if err := json.Unmarshal(encoded, &projected); err != nil {
			return err
		}
		for i := range projected.Assets {
			if projected.Assets[i].ID == asset {
				applyInputReopenAssessment(projected.Assets[i].Assessment, *r)
				projected.Assets[i].State = "generation-required"
			}
		}
		finishStructuredPlan(&projected, g)
		if err := v.verify(); err != nil {
			return err
		}
		if apply {
			if err := saveImmutable(root, inputReopenPath(*r), r); err != nil {
				return err
			}
			out.Applied = true
		}
		out.Ready, out.GenerationRequired, out.Receipt, out.Reference, out.ProjectedState, out.ProjectedExpected = true, true, *r, inputReopenReference(*r), "generation-required", projected.Expected
		out.OldUnits, out.CurrentUnits, out.Added, out.Removed, out.Changed = len(r.OldUnitIDs), len(r.CurrentUnitIDs), len(r.AddedUnitIDs), len(r.RemovedUnitIDs), len(r.ChangedUnitIDs)
		return nil
	}()
	if err != nil {
		out.Reason = err.Error()
	}
	return out, err
}
