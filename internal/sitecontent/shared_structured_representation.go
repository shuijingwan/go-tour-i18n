package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const SharedRepresentationSchema = "go-learning/shared-structured-representation-successor/v1"
const SharedRepresentationRule = "go-learning/source-role-slash-prose/v1"
const SharedSentencePunctuationRule = "go-learning/source-sentence-final-period/v1"
const SharedRepresentationRoot = "data/site-shell/shared-representation-successors/"
const SharedRepresentationPending = "data/.shared-structured-representation-pending"

// The request selects source coordinates, not target language or a finding's
// free-form prose. Generation proves that this exact representation was used.
type SharedRepresentationRequest struct {
	Schema           string    `json:"schema"`
	Package          string    `json:"package"`
	AssetID          string    `json:"asset_id"`
	Path             string    `json:"document_path"`
	SourceSHA        string    `json:"raw_source_sha256"`
	PreviousIdentity string    `json:"previous_representation_identity_sha256"`
	Generation       Reference `json:"consumed_generation"`
	Rule             string    `json:"parser_rule"`
	Slots            []string  `json:"slot_ids"`
}
type SharedRepresentationRelease struct {
	SlotID string         `json:"old_slot_id"`
	Token  ProtectedToken `json:"protected_binding"`
}
type SharedRepresentationSuccessor struct {
	Schema          string                        `json:"schema"`
	LocaleScope     string                        `json:"locale_scope"`
	Request         SharedRepresentationRequest   `json:"request"`
	SourcePackage   Package                       `json:"source_package"`
	RegistrySHA     string                        `json:"asset_registry_identity_sha256"`
	SourceAuthority []Reference                   `json:"source_authority"`
	Previous        *Reference                    `json:"previous,omitempty"`
	Old             ShellLanguageSource           `json:"old"`
	New             ShellLanguageSource           `json:"new"`
	NewIdentity     string                        `json:"prospective_representation_identity_sha256"`
	Released        []SharedRepresentationRelease `json:"released_atoms"`
	Inputs          []ShellRepresentationInput    `json:"currentness_inputs"`
	Identity        string                        `json:"identity_sha256"`
}
type SharedRepresentationPlan struct {
	Schema          string                        `json:"schema"`
	Request         Reference                     `json:"request_input"`
	Authority       SharedRepresentationSuccessor `json:"authority"`
	Receipt         Reference                     `json:"planned_receipt"`
	CurrentPath     string                        `json:"current_head_path"`
	PriorIdentity   string                        `json:"prior_identity_sha256"`
	PriorPointerSHA string                        `json:"prior_pointer_sha256"`
	State           string                        `json:"state"`
	Identity        string                        `json:"identity_sha256"`
}
type SharedRepresentationResult struct {
	Plan           *SharedRepresentationPlan `json:"plan"`
	AlreadyCurrent bool                      `json:"already_current"`
	Applied        bool                      `json:"applied"`
	Mutation       string                    `json:"mutation"`
}

var sharedRepresentationParserPaths = []string{
	"internal/sitecontent/shared_structured_representation.go",
	"internal/sitecontent/shared_structured_representation_publication.go",
	"internal/sitecontent/shared_structured_representation_lifecycle.go",
	"internal/sitecontent/semantic_role.go", "internal/sitecontent/semantic_go_packages.go",
	"internal/sitecontent/shell_language_source.go", "internal/sitecontent/linguistic.go",
	"internal/sitecontent/shell_reconstruction.go", "internal/sitecontent/shell_adoption.go",
	"internal/sitecontent/units.go", "internal/sitecontent/structured_historical_representation.go",
	"internal/sitecontent/structured_lifecycle.go",
	"docs/SHARED_STRUCTURED_REPRESENTATION_SUCCESSOR.md", "go.mod", "go.sum",
}

func sharedRepresentationHead(pkg, asset string) string {
	return SharedRepresentationRoot + "current/" + identity([]string{pkg, asset}) + ".json"
}
func sharedRepresentationReference(r SharedRepresentationSuccessor) Reference {
	b, _ := Encode(r)
	return Reference{SharedRepresentationRoot + "receipts/" + r.Identity + ".json", digest(b)}
}
func sharedRepresentationDocument(r SharedRepresentationSuccessor) Document {
	return withSharedDocumentAuthority(shellDocument(r.New), sharedRepresentationReference(r))
}

// Use the existing hash-bound dependency envelope. Adding a protocol-specific
// field to Document would change units.go, an independent frozen Homepage
// source authority. No permission to evolve that authority is implied here.
func withSharedDocumentAuthority(d Document, ref Reference) Document {
	d.Dependencies = []Dependency{{Kind: "contract", Path: ref.Path, SHA256: ref.SHA256}}
	return d
}

func hasSharedDocumentAuthority(d Document) bool {
	for _, dep := range d.Dependencies {
		if strings.HasPrefix(dep.Path, SharedRepresentationRoot) {
			return true
		}
	}
	return false
}

func sharedDocumentAuthority(d Document) (*Reference, error) {
	if !hasSharedDocumentAuthority(d) {
		return nil, nil
	}
	if len(d.Dependencies) != 1 {
		return nil, fmt.Errorf("ambiguous shared representation dependency")
	}
	dep := d.Dependencies[0]
	name := strings.TrimSuffix(strings.TrimPrefix(dep.Path, SharedRepresentationRoot+"receipts/"), ".json")
	if dep.Kind != "contract" || dep.Path != SharedRepresentationRoot+"receipts/"+name+".json" || !validSHA(name) || !validSHA(dep.SHA256) {
		return nil, fmt.Errorf("unknown shared representation dependency")
	}
	return &Reference{dep.Path, dep.SHA256}, nil
}

func frozenSharedStructuredSource(d Document, raw []byte) (*ShellLanguageSource, error) {
	ref, err := sharedDocumentAuthority(d)
	if err != nil {
		return nil, err
	}
	if ref == nil || d.Contract != ShellExtractionContract || d.Kind != "data" || d.UnitScope != "" || d.SourceSHA != digest(raw) {
		return nil, fmt.Errorf("invalid shared structured frozen representation")
	}
	base, err := ExtractShellLanguageSource(d.Path, raw)
	if err != nil || base.Page != nil || len(base.Slots) > len(d.Units) {
		return nil, fmt.Errorf("shared frozen source framing changed")
	}
	frames := map[int]ShellLanguageSlot{}
	required := map[int]bool{}
	for _, slot := range base.Slots {
		frames[slot.Unit.Start], required[slot.Unit.Start] = slot, true
	}
	if len(d.Units) > len(base.Slots) {
		periods, err := shellSentencePeriods(*base, raw)
		if err != nil {
			return nil, err
		}
		for _, period := range periods {
			frames[period.Slot.Unit.Start] = period.Slot
		}
	}
	e := shellExtractor{source: *base, raw: raw, occurrences: map[string]int{}}
	e.source.Slots = []ShellLanguageSlot{}
	seen := map[int]bool{}
	for _, unit := range d.Units {
		original, ok := frames[unit.Start]
		if !ok || seen[unit.Start] {
			return nil, fmt.Errorf("unknown/duplicate shared frozen coordinate")
		}
		if original.Role == "sentence-punctuation" && (unit.Source != "." || len(unit.Protected) != 0) {
			return nil, fmt.Errorf("shared frozen punctuation source/protection changed")
		}
		seen[unit.Start] = true
		delete(required, unit.Start)
		s := original
		s.Unit = unit
		if s.Unit.Start < 0 || s.Unit.End < s.Unit.Start || s.Unit.End > len(raw) {
			return nil, fmt.Errorf("shared slot coordinates")
		}
		want, originalErr := restoreBlock(original.Unit, original.Unit.Source)
		plain, err := restoreBlock(s.Unit, s.Unit.Source)
		if err != nil || originalErr != nil || plain != want || s.Unit.Start != original.Unit.Start || s.Unit.End != original.Unit.End || s.Role != original.Role || s.Context != original.Context || s.Encoding != original.Encoding {
			return nil, fmt.Errorf("shared slot source round-trip")
		}
		if err := e.protectedSlot(s.Unit.Start, s.Unit.End, s.Unit.Source, s.Unit.Protected, s.Role, s.Context, s.Encoding); err != nil {
			return nil, err
		}
	}
	if len(required) != 0 {
		return nil, fmt.Errorf("shared frozen slot removed")
	}
	s, err := e.finish()
	if err != nil {
		return nil, fmt.Errorf("shared frozen structure/slot identity mismatch")
	}
	want := withSharedDocumentAuthority(shellDocument(*s), *ref)
	if !reflect.DeepEqual(want, d) {
		return nil, fmt.Errorf("shared frozen Document mismatch")
	}
	values := map[string]string{}
	for _, s := range s.Slots {
		values[s.Unit.ID] = s.Unit.Source
	}
	result, err := reconstructShellSlots(*s, raw, values)
	if err != nil || !bytes.Equal(result, raw) {
		return nil, fmt.Errorf("shared frozen source round-trip: %v", err)
	}
	return s, nil
}

func frozenSharedStructuredDocument(d Document, raw []byte) (*Document, error) {
	if _, err := frozenSharedStructuredSource(d, raw); err != nil {
		return nil, err
	}
	return &d, nil
}

// Keep the legacy codecs byte-identical. Only structured consumers opt into
// this explicitly authority-bearing dependency; Homepage owns neither it nor
// the mixed Document declaration in units.go.
func parseAuthorityStructuredRepresentation(d Document, raw []byte) (*Document, error) {
	if hasSharedDocumentAuthority(d) {
		return frozenSharedStructuredDocument(d, raw)
	}
	return parseStructuredRepresentation(d, raw)
}

func reconstructAuthorityStructuredRepresentation(d Document, raw []byte, targets map[string]string) ([]byte, error) {
	if hasSharedDocumentAuthority(d) {
		source, err := frozenSharedStructuredSource(d, raw)
		if err != nil {
			return nil, err
		}
		return reconstructShellSlots(*source, raw, targets)
	}
	return reconstructStructuredRepresentation(d, raw, targets)
}

// Only selected old atoms can change. The fresh semantic protector supplies
// positive URL/path/API/markup authority; no alphabetic/slash exception list.
func sharedRepresentationDelta(old ShellLanguageSource, raw []byte, ids []string) (*ShellLanguageSource, []SharedRepresentationRelease, error) {
	if old.Page != nil || old.SourceSHA != digest(raw) || len(ids) == 0 {
		return nil, nil, fmt.Errorf("exact shared source and nonempty slot set required")
	}
	wanted := map[string]bool{}
	for i, id := range ids {
		if !validSHA(id) || wanted[id] || (i > 0 && ids[i-1] >= id) {
			return nil, nil, fmt.Errorf("ordered unique exact slot IDs required")
		}
		wanted[id] = true
	}
	changes := []SharedRepresentationRelease{}
	e := shellExtractor{source: old, raw: raw, occurrences: map[string]int{}}
	e.source.Slots = []ShellLanguageSlot{}
	for _, s := range old.Slots {
		text, locked := s.Unit.Source, s.Unit.Protected
		if wanted[s.Unit.ID] {
			delete(wanted, s.Unit.ID)
			if s.Role == "teaching-comment" || (s.Encoding != "html-text" && s.Encoding != "html-attribute") {
				return nil, nil, fmt.Errorf("unsupported source prose coordinate")
			}
			plain, err := restoreBlock(s.Unit, s.Unit.Source)
			if err != nil {
				return nil, nil, err
			}
			semantic, protected, err := protectBlockPolicy([]byte(plain), "html/"+s.Role, true, nil, false)
			if err != nil {
				return nil, nil, err
			}
			locked = []ProtectedToken{}
			count := len(changes)
			for _, p := range s.Unit.Protected {
				release := p.Role == "atom" && p.Binding == nil && strings.Contains(p.Raw, "/") && !strings.ContainsAny(p.Raw, "<>[]{} `\t\n\r") && strings.Contains(semantic, p.Raw)
				// Unknown slash spellings in an explicit identity declaration are
				// ambiguous, not prose evidence. Reuse the established source-role
				// positions, conservatively retaining every such occurrence.
				for at := 0; release && at < len(plain); {
					i := strings.Index(plain[at:], p.Raw)
					if i < 0 {
						break
					}
					i += at
					before, after := plain[max(0, i-96):i], plain[i+len(p.Raw):min(len(plain), i+len(p.Raw)+48)]
					if identifierPosition.MatchString(before) || enumPosition.MatchString(before) || enumFollowing.MatchString(after) {
						release = false
					}
					at = i + len(p.Raw)
				}
				for _, q := range protected {
					if strings.Contains(q.Raw, p.Raw) {
						release = false
					}
				}
				if release {
					text = strings.ReplaceAll(text, p.Token, p.Raw)
					changes = append(changes, SharedRepresentationRelease{s.Unit.ID, p})
				} else {
					locked = append(locked, p)
				}
			}
			if len(changes) == count {
				return nil, nil, fmt.Errorf("selected slot has no source-role-proven slash prose delta")
			}
		}
		if err := e.protectedSlot(s.Unit.Start, s.Unit.End, text, locked, s.Role, s.Context, s.Encoding); err != nil {
			return nil, nil, err
		}
	}
	if len(wanted) != 0 {
		return nil, nil, fmt.Errorf("unknown requested slot")
	}
	next, err := e.finish()
	if err != nil {
		return nil, nil, err
	}
	values := map[string]string{}
	for _, s := range next.Slots {
		values[s.Unit.ID] = s.Unit.Source
	}
	round, err := reconstructShellSlots(*next, raw, values)
	if err != nil || !bytes.Equal(round, raw) {
		return nil, nil, fmt.Errorf("prospective shared source round-trip: %v", err)
	}
	return next, changes, nil
}

func sharedRepresentationRuleKnown(rule string) bool {
	return rule == SharedRepresentationRule || rule == SharedSentencePunctuationRule
}

func sharedRepresentationRuleDelta(old ShellLanguageSource, raw []byte, ids []string, rule string) (*ShellLanguageSource, []SharedRepresentationRelease, error) {
	if rule == SharedRepresentationRule {
		return sharedRepresentationDelta(old, raw, ids)
	}
	if rule != SharedSentencePunctuationRule || len(ids) == 0 {
		return nil, nil, fmt.Errorf("known rule and nonempty exact slot set required")
	}
	wanted := map[string]bool{}
	for i, id := range ids {
		if !validSHA(id) || wanted[id] || (i > 0 && ids[i-1] >= id) {
			return nil, nil, fmt.Errorf("ordered unique exact slot IDs required")
		}
		wanted[id] = true
	}
	periods, err := shellSentencePeriods(old, raw)
	if err != nil {
		return nil, nil, err
	}
	e := shellExtractor{source: old, raw: raw, occurrences: map[string]int{}}
	e.source.Slots = append([]ShellLanguageSlot{}, old.Slots...)
	for _, period := range periods {
		if !wanted[period.Prefix] {
			continue
		}
		for _, slot := range old.Slots {
			if slot.Unit.Start <= period.Slot.Unit.Start && slot.Unit.End > period.Slot.Unit.Start {
				return nil, nil, fmt.Errorf("selected sentence punctuation already editable")
			}
		}
		delete(wanted, period.Prefix)
		s := period.Slot
		if err := e.protectedSlot(s.Unit.Start, s.Unit.End, s.Unit.Source, s.Unit.Protected, s.Role, s.Context, s.Encoding); err != nil {
			return nil, nil, err
		}
	}
	if len(wanted) != 0 {
		return nil, nil, fmt.Errorf("selected slot has no bounded sentence-final period")
	}
	next, err := e.finish()
	if err != nil {
		return nil, nil, err
	}
	values := map[string]string{}
	for _, slot := range next.Slots {
		values[slot.Unit.ID] = slot.Unit.Source
	}
	round, err := reconstructShellSlots(*next, raw, values)
	if err != nil || !bytes.Equal(round, raw) {
		return nil, nil, fmt.Errorf("prospective punctuation source round-trip: %v", err)
	}
	return next, []SharedRepresentationRelease{}, nil
}

// Reconstruct this rule from frozen Old and source syntax, never target text.
// It may add only the selected punctuation slots and release no old binding.
func checkSharedSentencePunctuation(r SharedRepresentationSuccessor, raw []byte) error {
	next, releases, err := sharedRepresentationRuleDelta(r.Old, raw, r.Request.Slots, SharedSentencePunctuationRule)
	if err != nil || !reflect.DeepEqual(next, &r.New) || !reflect.DeepEqual(releases, r.Released) {
		return fmt.Errorf("unbounded shared sentence punctuation topology: %v", err)
	}
	return nil
}

func sharedRepresentationReady(root string, owned bool) error {
	if err := shellRepresentationReady(root, owned); err != nil {
		return err
	}
	return sharedRepresentationPublicationGuard(root)
}

func sharedRepresentationPublicationGuard(root string) error {
	for _, path := range []string{SharedRepresentationPending, SharedReattestationPending} {
		if _, err := os.Lstat(pathJoin(root, path)); !os.IsNotExist(err) {
			return fmt.Errorf("mutation=unknown: pending shared representation publication: %s", path)
		}
	}
	return nil
}
func readSharedRepresentationHead(root, pkg, asset string, v *structuredSuccessorValidation) (*Reference, string, error) {
	name := sharedRepresentationHead(pkg, asset)
	if err := v.guard.optional(name); err != nil {
		return nil, "", err
	}
	if v.guard.absent[name] {
		return nil, digest(nil), nil
	}
	b, err := v.guard.readRegular(root, name)
	if err != nil {
		return nil, "", err
	}
	var ref Reference
	if err := StrictJSON(b, &ref); err != nil || !validSHA(ref.SHA256) || !validPath(ref.Path) {
		return nil, "", fmt.Errorf("invalid/ambiguous shared representation head")
	}
	canonical, _ := Encode(ref)
	if !bytes.Equal(canonical, b) {
		return nil, "", fmt.Errorf("noncanonical shared representation head")
	}
	return &ref, digest(b), nil
}

func readSharedRepresentationReceipt(root string, ref Reference, raw []byte, v *structuredSuccessorValidation, seen map[Reference]bool) (*SharedRepresentationSuccessor, error) {
	if len(seen) >= 32 || seen[ref] {
		return nil, fmt.Errorf("shared representation ancestry limit/cycle")
	}
	seen[ref] = true
	defer delete(seen, ref)
	var r SharedRepresentationSuccessor
	if err := v.guard.readReference(root, ref, &r); err != nil {
		return nil, err
	}
	x := r
	x.Identity = ""
	if r.Schema != SharedRepresentationSchema || r.LocaleScope != "all-locales" || r.Identity != identity(x) || ref != sharedRepresentationReference(r) || r.Request.Schema != SharedRepresentationSchema+"/request" || !sharedRepresentationRuleKnown(r.Request.Rule) || r.Request.AssetID != "source:"+r.Request.Path || r.Request.Package != r.SourcePackage.ID || r.Old.Page != nil || r.New.Page != nil || r.Old.Path != r.Request.Path || r.New.Path != r.Request.Path || r.Request.SourceSHA != digest(raw) || r.Old.SourceSHA != digest(raw) || r.New.SourceSHA != digest(raw) || r.Request.PreviousIdentity != identity(r.Old) || (r.Request.Rule == SharedRepresentationRule && len(r.Released) == 0) {
		return nil, fmt.Errorf("shared representation identity/scope mismatch")
	}
	if r.NewIdentity != identity(r.New) {
		return nil, fmt.Errorf("shared prospective representation identity mismatch")
	}
	if r.Previous != nil {
		parent, err := readSharedRepresentationReceipt(root, *r.Previous, raw, v, seen)
		if err != nil {
			return nil, err
		}
		if parent.Request.Package != r.Request.Package || parent.Request.AssetID != r.Request.AssetID || !reflect.DeepEqual(parent.New, r.Old) {
			return nil, fmt.Errorf("shared predecessor mismatch")
		}
	} else {
		base, err := ExtractShellLanguageSource(r.Request.Path, raw)
		if err != nil || !reflect.DeepEqual(base, &r.Old) {
			return nil, fmt.Errorf("shared initial frozen predecessor mismatch")
		}
	}
	// Recheck the sealed consumption evidence without replaying today's
	// package eligibility or language classification through a current head.
	var consumed StructuredGeneration
	if err := v.guard.readReference(root, r.Request.Generation, &consumed); err != nil {
		return nil, err
	}
	gc := consumed
	gc.Identity = ""
	pc := consumed.Plan
	pc.Identity = ""
	if consumed.Schema != StructuredSchema || consumed.Identity != identity(gc) || consumed.Plan.Identity != identity(pc) || r.Request.Generation.Path != "data/locale-language/"+consumed.Plan.Locale+"/structured-generation/"+consumed.Identity+".json" {
		return nil, fmt.Errorf("shared consumed Generation identity mismatch")
	}
	if err := v.guard.bind(consumed.Bundle); err != nil {
		return nil, err
	}
	expectedOld := shellDocument(r.Old)
	if r.Previous != nil {
		expectedOld = withSharedDocumentAuthority(expectedOld, *r.Previous)
	}
	bound := false
	for _, asset := range consumed.Plan.Assets {
		if asset.ID != r.Request.AssetID || asset.Package != r.Request.Package || asset.SourceSHA != r.RegistrySHA || !reflect.DeepEqual(asset.Document, &expectedOld) || asset.TargetPath != "locales/"+consumed.Plan.Locale+"/structured/"+r.Request.Path {
			continue
		}
		for _, output := range consumed.Output.Outputs {
			bound = bound || output.Path == asset.TargetPath
		}
	}
	if !bound {
		return nil, fmt.Errorf("shared predecessor is not the consumed registered representation")
	}
	if r.Request.Rule == SharedSentencePunctuationRule {
		if err := checkSharedSentencePunctuation(r, raw); err != nil {
			return nil, err
		}
		if _, err := frozenSharedStructuredDocument(sharedRepresentationDocument(r), raw); err != nil {
			return nil, err
		}
		return &r, nil
	}
	// Frozen lineage validates the exact recorded release, not today's semantic
	// judgment or current head. Current readers independently rebuild the delta.
	newByCoordinate := map[int]ShellLanguageSlot{}
	for _, s := range r.New.Slots {
		newByCoordinate[s.Unit.Start] = s
	}
	if len(r.New.Slots) != len(r.Old.Slots) {
		return nil, fmt.Errorf("shared slot inventory changed")
	}
	remaining := map[string]ProtectedToken{}
	for _, d := range r.Released {
		key := d.SlotID + "/" + d.Token.Token
		if _, ok := remaining[key]; ok {
			return nil, fmt.Errorf("duplicate shared release")
		}
		remaining[key] = d.Token
	}
	for _, o := range r.Old.Slots {
		n, ok := newByCoordinate[o.Unit.Start]
		if !ok || n.Unit.End != o.Unit.End || n.Role != o.Role || n.Context != o.Context || n.Encoding != o.Encoding {
			return nil, fmt.Errorf("shared slot framing changed")
		}
		text := o.Unit.Source
		locked := []ProtectedToken{}
		for _, p := range o.Unit.Protected {
			key := o.Unit.ID + "/" + p.Token
			if q, yes := remaining[key]; yes {
				if !reflect.DeepEqual(p, q) || p.Role != "atom" || p.Binding != nil || !strings.Contains(p.Raw, "/") {
					return nil, fmt.Errorf("invalid shared released binding")
				}
				delete(remaining, key)
				text = strings.ReplaceAll(text, p.Token, p.Raw)
			} else {
				locked = append(locked, p)
			}
		}
		if n.Unit.Source != text || !reflect.DeepEqual(n.Unit.Protected, locked) {
			return nil, fmt.Errorf("unbounded shared protected topology change")
		}
	}
	if len(remaining) != 0 {
		return nil, fmt.Errorf("unknown shared release")
	}
	d := sharedRepresentationDocument(r)
	if _, err := frozenSharedStructuredDocument(d, raw); err != nil {
		return nil, err
	}
	return &r, nil
}

func sharedRepresentationBasis(root string, g *Global, q SharedRepresentationRequest, v *structuredSuccessorValidation) (*ShellLanguageSource, Package, string, []Reference, []byte, error) {
	if q.Package != "site-v2-shell" || q.AssetID != "source:"+q.Path || !validPath(q.Path) || q.Schema != SharedRepresentationSchema+"/request" || !sharedRepresentationRuleKnown(q.Rule) {
		return nil, Package{}, "", nil, nil, fmt.Errorf("registered shared structured asset/request required")
	}
	// Membership and canonical representation are downstream projections. The
	// receipt binds the independently verified source-adoption/successor graph.
	if g.Membership != nil || g.ShellRepresentation != nil {
		var err error
		g, err = loadSourceCurrent(root, v)
		if err != nil {
			return nil, Package{}, "", nil, nil, err
		}
	}
	docs, raws, err := baseCurrentPackageDocuments(root, g, q.Package, v)
	if err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	var doc *Document
	for i := range docs {
		d := &docs[i]
		if d.Path == q.Path && d.Kind == "data" && d.Contract == ShellExtractionContract && d.UnitScope == "" {
			if doc != nil {
				return nil, Package{}, "", nil, nil, fmt.Errorf("ambiguous shared owner")
			}
			doc = d
		}
	}
	if doc == nil {
		return nil, Package{}, "", nil, nil, fmt.Errorf("asset is not a registered shared structured Document")
	}
	raw := raws[q.Path]
	if digest(raw) != q.SourceSHA {
		return nil, Package{}, "", nil, nil, fmt.Errorf("shared raw source SHA mismatch")
	}
	if err := v.guard.bind(Reference{q.Path, q.SourceSHA}); err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	corpusBytes, err := v.guard.readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	var corpus i18n.GlossarySourceCorpus
	if err := StrictJSON(corpusBytes, &corpus); err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	if err := i18n.ValidateGlossaryCorpus(&corpus); err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	groups, err := structuredContributorGroups(corpus.Contributors)
	if err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	cs := groups[q.AssetID]
	if len(cs) == 0 || cs[0].Package != q.Package || cs[0].Path != q.Path {
		return nil, Package{}, "", nil, nil, fmt.Errorf("shared structured registry owner mismatch")
	}
	s, err := ExtractShellLanguageSource(q.Path, raw)
	if err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	if !reflect.DeepEqual(shellDocument(*s), *doc) {
		return nil, Package{}, "", nil, nil, fmt.Errorf("unsupported shared source foundation")
	}
	p, err := packageByID(g, q.Package)
	if err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	refs := append([]Reference{}, g.StructuredSuccessors...)
	if g.ShellAdoption == nil {
		return nil, Package{}, "", nil, nil, fmt.Errorf("shared Shell adoption required")
	}
	refs = append(refs, *g.ShellAdoption)
	var adoption ShellAdoption
	if err := v.guard.readReference(root, *g.ShellAdoption, &adoption); err != nil {
		return nil, Package{}, "", nil, nil, err
	}
	refs = append(refs, adoption.Foundation)
	for _, r := range refs {
		if err := v.guard.bind(r); err != nil {
			return nil, Package{}, "", nil, nil, err
		}
	}
	return s, p, identity(cs), refs, raw, nil
}

func checkSharedRepresentationSource(root string, g *Global, r SharedRepresentationSuccessor, v *structuredSuccessorValidation) error {
	_, pkg, registry, refs, raw, err := sharedRepresentationBasis(root, g, r.Request, v)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(pkg, r.SourcePackage) || registry != r.RegistrySHA || !reflect.DeepEqual(refs, r.SourceAuthority) {
		return fmt.Errorf("shared source/registry authority changed")
	}
	next, releases, err := sharedRepresentationRuleDelta(r.Old, raw, r.Request.Slots, r.Request.Rule)
	if err != nil || !reflect.DeepEqual(next, &r.New) || !reflect.DeepEqual(releases, r.Released) {
		return fmt.Errorf("shared current semantic/topology proof changed: %v", err)
	}
	return nil
}

// Append-only recognition of complete canonical implementation inventories.
// Fingerprints are SHA256(json.Marshal(sorted implementation paths)); corpus
// and consumed Bundle paths are separately derived exact, mandatory inputs.
// Retain every prior fingerprint when evolving the declared creation profile.
const (
	sharedLegacyProfileSHA     = "f3d9fbe15ba2e3a474c98c5a46d04b1c3af869f8b5baa26b63a8d59dbd58f0f8"
	sharedExtendedV1ProfileSHA = "11a0c0d22ffbc1f14a3d3ec72eef11574971e46486755469feaf3e3b3bfa9292"
	sharedAuditV2ProfileSHA    = "71d3a48ee30b9570b7a0800535a865a6b3148559296645302ef88b6a0511d49b"
)

func sharedFrozenProfilePaths(frozen []ShellRepresentationInput, bundle string) ([]string, error) {
	paths := []string{}
	if frozen == nil {
		// Creation uses today's complete declared profile, never a caller subset.
		paths = uniqueStrings(append(append(append([]string{}, sharedRepresentationParserPaths...), sharedReattestationValidationPaths...), i18n.GlossaryCorpusPath, bundle))
	} else {
		for _, input := range frozen {
			p := input.Reference.Path
			if !validPath(p) || !validSHA(input.Reference.SHA256) || input.Absent || !os.FileMode(input.Mode).IsRegular() || (len(paths) > 0 && paths[len(paths)-1] >= p) {
				return nil, fmt.Errorf("invalid frozen shared dependency inventory: %s", p)
			}
			paths = append(paths, p)
		}
	}
	implementation := []string{}
	corpusFound, bundleFound := false, false
	for _, p := range paths {
		switch p {
		case i18n.GlossaryCorpusPath:
			corpusFound = true
		case bundle:
			bundleFound = true
		default:
			implementation = append(implementation, p)
		}
	}
	if !corpusFound || !bundleFound || bundle == i18n.GlossaryCorpusPath {
		return nil, fmt.Errorf("shared dependency exact-set mismatch: corpus/consumed Bundle")
	}
	switch identity(implementation) {
	case sharedLegacyProfileSHA, sharedExtendedV1ProfileSHA, sharedAuditV2ProfileSHA:
		return paths, nil
	default:
		return nil, fmt.Errorf("shared dependency exact-set mismatch: unrecognized canonical profile")
	}
}

func sharedRepresentationCurrentInputs(root string, r SharedRepresentationSuccessor, v *structuredSuccessorValidation) ([]ShellRepresentationInput, error) {
	if r.Schema != "" && r.Inputs == nil {
		return nil, fmt.Errorf("missing frozen shared dependency inventory")
	}
	var consumed StructuredGeneration
	if err := v.guard.readReference(root, r.Request.Generation, &consumed); err != nil {
		return nil, err
	}
	paths, err := sharedFrozenProfilePaths(r.Inputs, consumed.Bundle.Path)
	if err != nil {
		return nil, err
	}
	inputs := []ShellRepresentationInput{}
	for _, path := range paths {
		b, err := v.guard.readRegular(root, path)
		if err != nil {
			return nil, err
		}
		st, err := os.Stat(pathJoin(root, path))
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, ShellRepresentationInput{Reference: Reference{path, digest(b)}, Mode: uint32(st.Mode())})
	}
	if err := v.guard.bind(r.Request.Generation); err != nil {
		return nil, err
	}
	if err := v.guard.bind(consumed.Bundle); err != nil {
		return nil, err
	}
	if r.Inputs != nil {
		if _, err := sharedDependencyChanges(r.Inputs, inputs); err != nil {
			return nil, err
		}
	}
	return inputs, nil
}

func checkSharedRepresentationCurrent(root string, g *Global, r SharedRepresentationSuccessor, v *structuredSuccessorValidation) error {
	if err := checkSharedRepresentationSource(root, g, r, v); err != nil {
		return err
	}
	current, err := sharedRepresentationCurrentInputs(root, r, v)
	if err != nil {
		return err
	}
	return currentSharedRepresentationDependencies(root, r, current, v)
}

// This authority needs proof of consumed source representation, never a carry
// or language PASS. Keep the historical decoding checks shared with the full
// reader, without replaying unrelated package Page/Surface eligibility chains.
func sharedRepresentationConsumedGeneration(root string, ref Reference, guard *closureValidation) (*StructuredGeneration, map[string][]byte, map[string][]byte, error) {
	if err := guard.bind(ref); err != nil {
		return nil, nil, nil, err
	}
	r, m, fs, err := readStructuredGeneration(root, ref)
	if err != nil {
		return nil, nil, nil, err
	}
	if err := guard.bind(r.Bundle); err != nil {
		return nil, nil, nil, err
	}
	if r.CarryForward != nil {
		for _, dep := range []Reference{r.CarryForward.CurrentBundle, r.CarryForward.OriginalOutput} {
			if err := guard.bind(dep); err != nil {
				return nil, nil, nil, err
			}
		}
	}
	if err := guard.review(r.Plan.Locale, r.Plan.Review); err != nil {
		return nil, nil, nil, err
	}
	return guard.validateFrozenStructuredContent(root, r, m, fs)
}

func buildSharedRepresentationPlan(root string, g *Global, input Reference, v *structuredSuccessorValidation, owned bool) (*SharedRepresentationPlan, error) {
	if err := sharedRepresentationReady(root, owned); err != nil {
		return nil, err
	}
	var q SharedRepresentationRequest
	if err := v.guard.readReference(root, input, &q); err != nil {
		return nil, err
	}
	base, pkg, registry, refs, raw, err := sharedRepresentationBasis(root, g, q, v)
	if err != nil {
		return nil, err
	}
	head, sha, err := readSharedRepresentationHead(root, q.Package, q.AssetID, v)
	if err != nil {
		return nil, err
	}
	old := *base
	if head != nil {
		parent, err := readSharedRepresentationReceipt(root, *head, raw, v, map[Reference]bool{})
		if err != nil {
			return nil, err
		}
		if err := checkSharedRepresentationCurrent(root, g, *parent, v); err != nil {
			return nil, err
		}
		old = parent.New
	}
	if q.PreviousIdentity != identity(old) {
		return nil, fmt.Errorf("exact predecessor representation required")
	}
	gen, _, _, err := sharedRepresentationConsumedGeneration(root, q.Generation, v.guard)
	if err != nil {
		return nil, fmt.Errorf("shared consumed Generation: %w", err)
	}
	consumed := false
	for _, a := range gen.Plan.Assets {
		if a.ID == q.AssetID && a.Package == q.Package && a.Document != nil && a.SourceSHA == registry && a.TargetPath == "locales/"+gen.Plan.Locale+"/structured/"+q.Path {
			expected := shellDocument(old)
			if head != nil {
				parent, err := readSharedRepresentationReceipt(root, *head, raw, v, map[Reference]bool{})
				if err != nil {
					return nil, err
				}
				expected = sharedRepresentationDocument(*parent)
			}
			for _, out := range gen.Output.Outputs {
				consumed = consumed || out.Path == a.TargetPath && reflect.DeepEqual(a.Document, &expected)
			}
		}
	}
	if !consumed {
		return nil, fmt.Errorf("consumed Generation does not bind exact predecessor structured owner")
	}
	if err := v.guard.bind(q.Generation); err != nil {
		return nil, err
	}
	next, releases, err := sharedRepresentationRuleDelta(old, raw, q.Slots, q.Rule)
	if err != nil {
		return nil, err
	}
	inputs, err := sharedRepresentationCurrentInputs(root, SharedRepresentationSuccessor{Request: q}, v)
	if err != nil {
		return nil, err
	}
	r := SharedRepresentationSuccessor{Schema: SharedRepresentationSchema, LocaleScope: "all-locales", Request: q, SourcePackage: pkg, RegistrySHA: registry, SourceAuthority: refs, Previous: head, Old: old, New: *next, NewIdentity: identity(*next), Released: releases, Inputs: inputs}
	r.Identity = identity(r)
	p := &SharedRepresentationPlan{Schema: SharedRepresentationSchema + "/plan", Request: input, Authority: r, Receipt: sharedRepresentationReference(r), CurrentPath: sharedRepresentationHead(q.Package, q.AssetID), PriorIdentity: identity(old), PriorPointerSHA: sha, State: "generation-required"}
	p.Identity = identity(*p)
	return p, nil
}

func selectSharedStructuredDocuments(root string, g *Global, pkg string, docs []Document, raw map[string][]byte, contexts ...*structuredSuccessorValidation) ([]Document, map[string][]byte, error) {
	v := successorValidation(root, contexts)
	if err := sharedRepresentationReady(root, v.readonlyOwner); err != nil {
		return nil, nil, err
	}
	for i, d := range docs {
		if d.Contract != ShellExtractionContract || d.Kind != "data" || d.UnitScope != "" {
			continue
		}
		head, _, err := readSharedRepresentationHead(root, pkg, "source:"+d.Path, v)
		if err != nil {
			return nil, nil, err
		}
		if head == nil {
			continue
		}
		r, err := readSharedRepresentationReceipt(root, *head, raw[d.Path], v, map[Reference]bool{})
		if err != nil {
			return nil, nil, err
		}
		if r.Request.Package != pkg || r.Request.Path != d.Path {
			return nil, nil, fmt.Errorf("wrong shared current head owner")
		}
		if err := checkSharedRepresentationCurrent(root, g, *r, v); err != nil {
			return nil, nil, err
		}
		docs[i] = sharedRepresentationDocument(*r)
	}
	if err := v.verifyObserved(); err != nil {
		return nil, nil, err
	}
	return docs, raw, nil
}

func PlanSharedStructuredRepresentation(root string, request Reference) (*SharedRepresentationPlan, error) {
	var p *SharedRepresentationPlan
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		g, err := loadSourceCurrent(root, v)
		if err != nil {
			return err
		}
		p, err = buildSharedRepresentationPlan(root, g, request, v, false)
		return err
	})
	return p, err
}
