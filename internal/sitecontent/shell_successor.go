package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const ShellSuccessorSchema = "go-learning/shell-source-successor/v1"
const ShellAffectedSchema = "go-learning/shell-successor-affected-scope/v1"
const ShellSuccessorRoot = "data/site-shell/successor/"

type ShellSuccessorAuthority struct {
	ParserSemanticIdentity     string                `json:"parser_semantic_identity_sha256,omitempty"`
	Schema                     string                `json:"schema"`
	Package                    string                `json:"package"`
	State                      string                `json:"state"`
	ExtractionContract         string                `json:"extraction_contract"`
	FrozenManifestIdentity     string                `json:"frozen_manifest_identity_sha256"`
	PredecessorPackageIdentity string                `json:"predecessor_package_identity_sha256"`
	PredecessorSourceIdentity  string                `json:"predecessor_source_identity_sha256"`
	HistoricalHome             Reference             `json:"historical_home_source"`
	TranslationBody            Document              `json:"translation_body"`
	Sources                    []ShellLanguageSource `json:"sources"`
	ParserReferences           []Reference           `json:"parser_references"`
	Identity                   string                `json:"identity_sha256"`
}

type ShellSlotDestination struct {
	Path    string `json:"path"`
	SlotID  string `json:"slot_id"`
	Role    string `json:"role"`
	Context string `json:"context"`
}
type ShellSlotMapping struct {
	OldSlot     Unit                   `json:"old_slot"`
	Candidates  []ShellSlotDestination `json:"new_source_candidates"`
	Destination *ShellSlotDestination  `json:"carry_destination,omitempty"`
	State       string                 `json:"state"`
	Reason      string                 `json:"reason"`
	Evidence    []Reference            `json:"predecessor_evidence"`
	TargetSHA   string                 `json:"predecessor_target_sha256"`
}
type ShellScopeUnit struct {
	Path            string      `json:"source_path"`
	Role            string      `json:"role"`
	Route           string      `json:"canonical_route,omitempty"`
	UnitID          string      `json:"formal_unit_id,omitempty"`
	SourceSHA       string      `json:"source_sha256"`
	TargetSHA       string      `json:"accepted_target_sha256,omitempty"`
	State           string      `json:"state"`
	GenerationSlots []string    `json:"generation_required_slots"`
	CarrySlots      []string    `json:"exact_carry_slots"`
	Evidence        []Reference `json:"evidence"`
}
type ShellTerminologyImpact struct {
	CurrentCorpus   Reference                  `json:"current_corpus"`
	CurrentIdentity string                     `json:"current_corpus_identity_sha256"`
	Affected        bool                       `json:"affected"`
	NewSources      int                        `json:"new_contributor_source_count"`
	NewContexts     int                        `json:"new_context_count"`
	Contributors    []i18n.LanguageContributor `json:"proposed_contributors"`
	ChangedSources  []string                   `json:"changed_contributor_sources"`
	RetiredSources  []string                   `json:"retired_contributor_sources"`
}
type ShellAffectedScope struct {
	Schema            string                 `json:"schema"`
	Package           string                 `json:"package"`
	State             string                 `json:"state"`
	Locale            string                 `json:"locale"`
	SourceAuthority   Reference              `json:"successor_source_authority"`
	SourceIdentity    string                 `json:"successor_source_identity_sha256"`
	CurrentCompletion Reference              `json:"unchanged_current_completion"`
	OldHome           Document               `json:"historical_home_document"`
	OldHomeMappings   []ShellSlotMapping     `json:"old_home_slot_mappings"`
	Units             []ShellScopeUnit       `json:"language_units"`
	Terminology       ShellTerminologyImpact `json:"terminology_impact"`
	Identity          string                 `json:"identity_sha256"`
}

// The existing global/package authority is the historical predecessor. It is
// deliberately not upgraded: no pending successor can authorize completion.
func BuildShellSuccessorAuthority(root string) (*ShellSuccessorAuthority, error) {
	if _, err := os.Lstat(pathJoin(root, sourceMigrationLock)); !os.IsNotExist(err) {
		return nil, fmt.Errorf("source migration pending; mutation=unknown")
	}
	m, err := loadSourceManifest(root)
	if err != nil {
		return nil, err
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	p, err := packageByID(g, "site-v2-shell")
	if err != nil {
		return nil, err
	}
	a := &ShellSuccessorAuthority{Schema: ShellSuccessorSchema, Package: p.ID, State: "pending-source-authority", ExtractionContract: ShellExtractionContract, FrozenManifestIdentity: m.Identity, PredecessorPackageIdentity: p.Identity, PredecessorSourceIdentity: p.SourceIdentity, Sources: []ShellLanguageSource{}, ParserReferences: []Reference{}}
	for _, name := range []string{"_content/index.md", "_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"} {
		var selected *FrozenSourceFile
		for i := range m.Files {
			if m.Files[i].Path == name {
				selected = &m.Files[i]
				break
			}
		}
		if selected == nil || selected.Classification != "upstream-exact" {
			return nil, fmt.Errorf("frozen shell source authority missing: %s", name)
		}
		b, err := readAdoptedSource(root, *selected)
		if err != nil {
			return nil, err
		}
		source, err := ExtractShellLanguageSource(name, b)
		if err != nil {
			return nil, err
		}
		a.Sources = append(a.Sources, *source)
	}
	for _, s := range g.Sources {
		if s.Package != "site-v2-shell" {
			continue
		}
		b, err := readRegular(root, s.Path)
		if err != nil || digest(b) != s.SHA256 {
			return nil, fmt.Errorf("historical shell source mismatch: %s", s.Path)
		}
		switch s.Path {
		case "data/site-shell/home.html":
			a.HistoricalHome = Reference{s.Path, s.SHA256}
		case "data/site-shell/translation.html":
			d, err := ParseDocument(s, b)
			if err != nil {
				return nil, err
			}
			a.TranslationBody = *d
		default:
			return nil, fmt.Errorf("unexpected shell predecessor source")
		}
	}
	if a.HistoricalHome.Path == "" || a.TranslationBody.Path == "" {
		return nil, fmt.Errorf("incomplete shell predecessor")
	}
	for _, name := range []string{"internal/sitecontent/shell_language_source.go", "internal/sitecontent/shell_successor.go", "internal/sitecontent/shellyaml/parse.go", "internal/sitecontent/formal_page.go", "internal/sitecontent/units.go", "internal/sitecontent/linguistic.go", "internal/sitecontent/structural_binding.go", "internal/sitecontent/teaching.go", "internal/i18n/go_example_protect.go", "go.mod", "go.sum"} {
		r, err := ReferenceFile(root, name)
		if err != nil {
			return nil, err
		}
		a.ParserReferences = append(a.ParserReferences, r)
	}
	sort.Slice(a.ParserReferences, func(i, j int) bool { return a.ParserReferences[i].Path < a.ParserReferences[j].Path })
	a.ParserSemanticIdentity = shellParserSemanticIdentity(a.Sources)
	a.Identity = identity(*a)
	return a, nil
}

func shellAuthorityReference(a *ShellSuccessorAuthority) (Reference, error) {
	b, err := Encode(a)
	return Reference{ShellSuccessorRoot + "sources/" + a.Identity + ".json", digest(b)}, err
}

// Whitespace normalization is the sole source normalization. Protected
// identity, role and context are separate checks; label similarity never carries.
func normalizedShellSource(u Unit) string {
	return strings.Join(strings.Fields(u.Source), " ")
}

func compatibleShellSlot(old ShellLanguageSlot, next ShellLanguageSlot, target string) bool {
	if !hasLanguage(target) || strings.ContainsAny(target, "\x00\r\ufeff") {
		return false
	}
	if normalizedShellSource(old.Unit) != normalizedShellSource(next.Unit) || old.Role != next.Role || old.Context != next.Context || old.Encoding != next.Encoding {
		return false
	}
	if !reflect.DeepEqual(old.Unit.Protected, next.Unit.Protected) {
		return false
	}
	_, err := restoreBlock(next.Unit, target)
	if err != nil {
		return false
	}
	plain := target
	for _, p := range next.Unit.Protected {
		plain = strings.ReplaceAll(plain, p.Token, "word")
	}
	_, added, err := protectBlockMode([]byte(plain), "html/"+next.Role, false)
	return err == nil && len(added) == 0
}

// Legacy shell block roles are intentionally retained, not inferred from Chinese.
// A bundled nav block is not a collection of independently reviewed labels.
func legacyShellSlot(u Unit) ShellLanguageSlot {
	_, loc, _ := strings.Cut(u.Locator, "#")
	context, _, _ := strings.Cut(loc, ":")
	role := "visible-text"
	if strings.Contains(context, "/attribute/") {
		role = "attribute/" + context[strings.LastIndex(context, "/")+1:]
	}
	return ShellLanguageSlot{u, role, "legacy-shell/" + context, "html-text"}
}

func mapOldShellHome(old Document, sources []ShellLanguageSource, targets map[string]string, evidence []Reference) []ShellSlotMapping {
	out := []ShellSlotMapping{}
	for _, u := range old.Units {
		mapping := ShellSlotMapping{OldSlot: u, Candidates: []ShellSlotDestination{}, State: "affected", Reason: "no exact normalized source match; Generation required for successor destinations", Evidence: evidence}
		target, hasTarget := targets[u.ID]
		if hasTarget {
			mapping.TargetSHA = digest([]byte(target))
		}
		for _, s := range sources {
			for _, slot := range s.Slots {
				if normalizedShellSource(u) != normalizedShellSource(slot.Unit) {
					continue
				}
				d := ShellSlotDestination{s.Path, slot.Unit.ID, slot.Role, slot.Context}
				mapping.Candidates = append(mapping.Candidates, d)
				if hasTarget && len(evidence) > 0 && compatibleShellSlot(legacyShellSlot(u), slot, target) {
					copy := d
					mapping.Destination = &copy
				}
			}
		}
		if len(mapping.Candidates) > 0 {
			mapping.Reason = "source matches but role/context or protected structure is incompatible; no target/provenance-safe carry"
		}
		if len(mapping.Candidates) == 1 && mapping.Destination != nil {
			mapping.State = "carry"
			mapping.Reason = "exact normalized source, role/context, protection and valid target with accepted predecessor provenance"
		} else {
			mapping.Destination = nil
		}
		out = append(out, mapping)
	}
	return out
}

func shellTerminologyImpact(root string, sources []ShellLanguageSource) (ShellTerminologyImpact, error) {
	c, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return ShellTerminologyImpact{}, err
	}
	ref, err := ReferenceFile(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return ShellTerminologyImpact{}, err
	}
	impact := ShellTerminologyImpact{CurrentCorpus: ref, CurrentIdentity: c.Identity, Contributors: []i18n.LanguageContributor{}, ChangedSources: []string{}, RetiredSources: []string{"data/site-shell/home.html"}}
	old := map[string]i18n.LanguageContributor{}
	for _, contributor := range c.Contributors {
		old[contributor.Path] = contributor
	}
	if _, ok := old["data/site-shell/home.html"]; !ok {
		return impact, fmt.Errorf("current corpus lacks historical homepage contributor")
	}
	for _, source := range sources {
		contributor, err := ProjectShellSuccessorContributor(source)
		if err != nil {
			return impact, err
		}
		impact.Contributors = append(impact.Contributors, contributor)
		previous, exists := old[source.Path]
		if !exists {
			impact.NewSources++
			impact.NewContexts += len(contributor.Contexts)
		} else if !reflect.DeepEqual(previous.Contexts, contributor.Contexts) {
			impact.ChangedSources = append(impact.ChangedSources, source.Path)
		}
	}
	impact.Affected = impact.NewSources > 0 || len(impact.ChangedSources) > 0 || len(impact.RetiredSources) > 0
	sort.Slice(impact.Contributors, func(i, j int) bool { return impact.Contributors[i].ID < impact.Contributors[j].ID })
	return impact, nil
}

func BuildShellSuccessorAffectedScope(root, locale string, a *ShellSuccessorAuthority) (*ShellAffectedScope, error) {
	current, err := BuildShellSuccessorAuthority(root)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(a, current) {
		return nil, fmt.Errorf("shell successor source authority stale")
	}
	if !component(locale) {
		return nil, fmt.Errorf("invalid locale")
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, err
	}
	b, err := readRegular(root, LocalePath(locale))
	if err != nil {
		return nil, err
	}
	var l Locale
	if err = StrictJSON(b, &l); err != nil {
		return nil, err
	}
	if l.Locale != locale {
		return nil, fmt.Errorf("locale mismatch")
	}
	if err = ValidateLocale(g, l); err != nil {
		return nil, err
	}
	accepted := map[string]map[string][]byte{}
	completions := map[string]Completion{}
	for _, c := range l.Packages {
		if c.Package != "site-v2-shell" && c.Package != "learn-docs-v1" {
			continue
		}
		if err = checkPackageCompletion(root, g, locale, c); err != nil {
			return nil, err
		}
		_, files, err := CheckPackageClosure(root, c.Evidence[0])
		if err != nil {
			return nil, err
		}
		accepted[c.Package], completions[c.Package] = files, c
	}
	if accepted["site-v2-shell"] == nil || accepted["learn-docs-v1"] == nil {
		return nil, fmt.Errorf("accepted shell and Learn/Docs predecessor completion required")
	}
	ref, err := shellAuthorityReference(a)
	if err != nil {
		return nil, err
	}
	scope := &ShellAffectedScope{Schema: ShellAffectedSchema, Package: "site-v2-shell", State: "pending-language-work", Locale: locale, SourceAuthority: ref, SourceIdentity: a.Identity, CurrentCompletion: Reference{LocalePath(locale), digest(b)}, Units: []ShellScopeUnit{}}
	oldBytes, err := readRegular(root, a.HistoricalHome.Path)
	if err != nil {
		return nil, err
	}
	old, err := ParseDocument(Source{Path: a.HistoricalHome.Path, Kind: "page", SHA256: a.HistoricalHome.SHA256, Route: "/", Dependencies: []Dependency{}}, oldBytes)
	if err != nil {
		return nil, err
	}
	scope.OldHome = *old
	targets := map[string]string{}
	// Read the exact accepted structured output, not a mutable locale directory.
	var closure PackageClosure
	if err = readReference(root, completions["site-v2-shell"].Evidence[0], &closure); err != nil {
		return nil, err
	}
	for _, r := range closure.Structured {
		var generation StructuredGeneration
		if err = readReference(root, r, &generation); err != nil {
			return nil, err
		}
		for _, t := range generation.Output.Outputs {
			if t.Path == "locales/"+locale+"/structured/"+a.HistoricalHome.Path {
				for _, slot := range t.Slots {
					if _, ok := targets[slot.ID]; ok {
						return nil, fmt.Errorf("ambiguous old homepage slot provenance")
					}
					targets[slot.ID] = slot.Text
				}
			}
		}
	}
	if len(targets) != len(old.Units) {
		return nil, fmt.Errorf("old homepage accepted slot exact-set mismatch")
	}
	for _, u := range old.Units {
		target, ok := targets[u.ID]
		if !ok {
			return nil, fmt.Errorf("missing accepted old homepage slot")
		}
		if _, err := restoreBlock(u, target); err != nil {
			return nil, err
		}
	}
	scope.OldHomeMappings = mapOldShellHome(*old, a.Sources, targets, completions["site-v2-shell"].Evidence)
	for _, source := range a.Sources {
		u := ShellScopeUnit{Path: source.Path, Role: source.Role, SourceSHA: source.SourceSHA, State: "generation-required", GenerationSlots: []string{}, CarrySlots: []string{}, Evidence: []Reference{}}
		if source.Page != nil {
			u.Route = "/"
			u.UnitID = source.Page.Units[0].ID
		}
		for _, slot := range source.Slots {
			carry := false
			for _, m := range scope.OldHomeMappings {
				if m.State == "carry" && m.Destination.Path == source.Path && m.Destination.SlotID == slot.Unit.ID {
					carry = true
				}
			}
			if carry {
				u.CarrySlots = append(u.CarrySlots, slot.Unit.ID)
			} else {
				u.GenerationSlots = append(u.GenerationSlots, slot.Unit.ID)
			}
		}
		if len(u.GenerationSlots) == 0 {
			u.State = "carry"
		}
		scope.Units = append(scope.Units, u)
	}
	body := a.TranslationBody
	bodyTarget, ok := accepted["site-v2-shell"][body.Path]
	if !ok {
		return nil, fmt.Errorf("translation body accepted target missing")
	}
	bodyCarry := ShellScopeUnit{Path: body.Path, Role: "translation-body", Route: "/translation/", SourceSHA: body.SourceSHA, TargetSHA: digest(bodyTarget), State: "carry", GenerationSlots: []string{}, CarrySlots: []string{}, Evidence: completions["site-v2-shell"].Evidence}
	for _, u := range body.Units {
		bodyCarry.CarrySlots = append(bodyCarry.CarrySlots, u.ID)
	}
	scope.Units = append(scope.Units, bodyCarry)
	docs, _, err := WorkflowDocuments(root, g, "learn-docs-v1")
	if err != nil {
		return nil, err
	}
	expectedPages := 0
	for _, s := range g.Sources {
		if s.Package == "learn-docs-v1" && s.Kind == "page" {
			expectedPages++
		}
	}
	if expectedPages == 0 || len(docs) != expectedPages {
		return nil, fmt.Errorf("Learn/Docs exact Page set changed")
	}
	for _, d := range docs {
		target, ok := accepted["learn-docs-v1"][d.Path]
		if !ok {
			return nil, fmt.Errorf("Learn/Docs accepted target missing: %s", d.Path)
		}
		scope.Units = append(scope.Units, ShellScopeUnit{Path: d.Path, Role: "learn-docs-page", Route: d.Route, UnitID: d.Units[0].ID, SourceSHA: d.SourceSHA, TargetSHA: digest(target), State: "carry", GenerationSlots: []string{}, CarrySlots: []string{d.Units[0].ID}, Evidence: completions["learn-docs-v1"].Evidence})
	}
	scope.Terminology, err = shellTerminologyImpact(root, a.Sources)
	if err != nil {
		return nil, err
	}
	scope.Identity = identity(*scope)
	return scope, nil
}

// Additive, immutable artifacts only. No corpus, completion, closure or target
// is written, and no Generation/Review/runtime action is available here.
func SaveShellSuccessorFoundation(root, locale string) (*ShellSuccessorAuthority, *ShellAffectedScope, string, error) {
	a, err := BuildShellSuccessorAuthority(root)
	if err != nil {
		return nil, nil, "", err
	}
	scope, err := BuildShellSuccessorAffectedScope(root, locale, a)
	if err != nil {
		return nil, nil, "", err
	}
	if err = saveImmutable(root, scope.SourceAuthority.Path, a); err != nil {
		return nil, nil, "", err
	}
	name := ShellSuccessorRoot + "affected/" + locale + "/" + scope.Identity + ".json"
	return a, scope, name, saveImmutable(root, name, scope)
}

func CheckShellSuccessorFoundation(root, input string) (*ShellAffectedScope, error) {
	b, err := readRegular(root, input)
	if err != nil {
		return nil, err
	}
	var scope ShellAffectedScope
	if err = StrictJSON(b, &scope); err != nil {
		return nil, err
	}
	copy := scope
	copy.Identity = ""
	if scope.Schema != ShellAffectedSchema || scope.Package != "site-v2-shell" || scope.State != "pending-language-work" || !component(scope.Locale) || !validSHA(scope.Identity) || scope.Identity != identity(copy) {
		return nil, fmt.Errorf("invalid pending successor affected identity")
	}
	wantName := ShellSuccessorRoot + "affected/" + scope.Locale + "/" + scope.Identity + ".json"
	if input != wantName {
		return nil, fmt.Errorf("shell successor affected path mismatch")
	}
	a, err := BuildShellSuccessorAuthority(root)
	if err != nil {
		return nil, err
	}
	want, err := BuildShellSuccessorAffectedScope(root, scope.Locale, a)
	if err != nil {
		return nil, err
	}
	bound, err := readRegular(root, want.SourceAuthority.Path)
	if err != nil {
		return nil, err
	}
	if err = checkShellSuccessorWire(b, bound, want, a); err != nil {
		return nil, err
	}
	return want, nil
}

func checkShellSuccessorWire(scopeBytes, sourceBytes []byte, want *ShellAffectedScope, a *ShellSuccessorAuthority) error {
	var scope ShellAffectedScope
	var source ShellSuccessorAuthority
	if err := StrictJSON(scopeBytes, &scope); err != nil {
		return err
	}
	if err := StrictJSON(sourceBytes, &source); err != nil {
		return err
	}
	canonical, err := Encode(want)
	if err != nil {
		return err
	}
	if !bytes.Equal(scopeBytes, canonical) {
		return fmt.Errorf("shell successor affected scope stale/non-canonical")
	}
	canonical, err = Encode(a)
	if err != nil {
		return err
	}
	if !bytes.Equal(sourceBytes, canonical) || digest(sourceBytes) != want.SourceAuthority.SHA256 {
		return fmt.Errorf("shell successor source authority stale/non-canonical")
	}
	return nil
}
