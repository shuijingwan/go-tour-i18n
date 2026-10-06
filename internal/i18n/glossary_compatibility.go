package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"unicode"
)

const GlossaryCompatibilitySchema = "go-learning/glossary-compatibility/v1"

type GlossaryCompatibilityScope struct {
	ID              string   `json:"id"`
	ContextIdentity string   `json:"context_identity_sha256"`
	Reasons         []string `json:"reasons"`
}

type GlossaryCompatibilityEvidence struct {
	Schema                 string                         `json:"schema"`
	EvidenceKind           string                         `json:"evidence_kind"`
	Locale                 string                         `json:"locale"`
	OldGlossary            GlossaryArchiveReference       `json:"old_glossary"`
	NewGlossary            GlossaryArchiveReference       `json:"new_glossary"`
	NewFullReview          GlossaryArchiveReference       `json:"new_full_review"`
	NormalizationVersion   string                         `json:"normalization_version"`
	ImpactAlgorithmVersion string                         `json:"impact_algorithm_version"`
	Delta                  []GlossarySemanticDelta        `json:"normalized_semantic_delta"`
	Contexts               []GlossaryCompatibilityContext `json:"context_inventory"`
	Affected               []GlossaryCompatibilityScope   `json:"affected_scopes"`
	Compatible             []GlossaryCompatibilityScope   `json:"compatible_scopes"`
	Lineage                []GlossaryArchiveReference     `json:"lineage"`
	Identity               string                         `json:"evidence_identity_sha256"`
}

type GlossaryCompatibilityResolution struct {
	Status string                     `json:"status"`
	Reason string                     `json:"reason"`
	Chain  []GlossaryArchiveReference `json:"chain"`
}

func classifyGlossaryContext(context GlossaryCompatibilityContext, delta []GlossarySemanticDelta) (bool, []string) {
	if len(delta) == 0 {
		return true, []string{"semantic_delta_empty"}
	}
	// Removal of constraints needs no source-to-target language judgment.
	allRemoved := true
	for _, d := range delta {
		if d.New != nil {
			allRemoved = false
		}
	}
	if allRemoved {
		return true, []string{"constraint_removed"}
	}
	source, target, err := compatibilityVisiblePair(context)
	if err != nil {
		return false, []string{"unknown_parser"}
	}
	reasons := map[string]bool{}
	compatible := true
	for _, d := range delta {
		if d.New == nil {
			reasons["constraint_removed"] = true
			continue
		}
		switch d.Category {
		case "mandatory", "preferred", "keep":
			if compatibilitySourceTermPresent(source, d.Term) {
				compatible = false
				reasons[d.Category+"_source_term_present"] = true
			} else {
				reasons["changed_source_term_absent"] = true
			}
		case "forbidden":
			if compatibilityForbiddenTermPresent(target, d.Term) {
				compatible = false
				reasons["new_forbidden_target_present"] = true
			} else {
				reasons["new_forbidden_target_absent"] = true
			}
		default:
			compatible = false
			reasons["unknown_parser"] = true
		}
	}
	ordered := []string{}
	for reason := range reasons {
		ordered = append(ordered, reason)
	}
	sort.Strings(ordered)
	return compatible, ordered
}

// Visibility is supplied by the formal parser above. Do not run present
// markup parsing again on plain UI/metadata and accidentally hide literal text.
func compatibilityForbiddenTermPresent(text, term string) bool {
	if term == "" {
		return false
	}
	for offset := 0; offset < len(text); {
		relative := strings.Index(text[offset:], term)
		if relative < 0 {
			return false
		}
		start := offset + relative
		end := start + len(term)
		if forbiddenTermBoundaries(text, start, end) {
			return true
		}
		offset = start + 1
	}
	return false
}

// Source rules are conservatively case-insensitive and tolerate whitespace
// and English word suffixes. A visible prefix collision is affected, never a
// claimed proof of absence. Forbidden targets retain the validator's exact
// Unicode/apostrophe boundary semantics.
func compatibilitySourceTermPresent(text, term string) bool {
	normalize := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
	text, term = normalize(text), normalize(term)
	if term == "" {
		return true
	}
	for offset := 0; offset < len(text); {
		position := strings.Index(text[offset:], term)
		if position < 0 {
			break
		}
		start := offset + position
		if forbiddenBoundaryBefore(text, start) {
			return true
		}
		offset = start + 1
	}
	// Punctuation and inflection are conservative collision checks. They can
	// increase affected scopes, but never establish compatibility by guessing
	// that a near spelling is unrelated (libraries/library, SQL-transaction).
	words := func(s string) []string {
		return strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	}
	stem := func(s string) string {
		if len(s) > 4 && strings.HasSuffix(s, "ies") {
			return strings.TrimSuffix(s, "ies") + "y"
		}
		for _, suffix := range []string{"ing", "ed", "es", "s"} {
			if len(s) > len(suffix)+2 && strings.HasSuffix(s, suffix) {
				return strings.TrimSuffix(s, suffix)
			}
		}
		return s
	}
	a, b := words(text), words(term)
	if len(b) == 0 {
		return true
	}
	for i := 0; i+len(b) <= len(a); i++ {
		match := true
		for j := range b {
			left, right := stem(a[i+j]), stem(b[j])
			if !strings.HasPrefix(left, right) && !strings.HasPrefix(right, left) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func glossaryCompatibilityEvidencePath(locale, identity string) (string, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return "", err
	}
	if !validSHA256(identity) {
		return "", fmt.Errorf("invalid compatibility identity")
	}
	return "data/glossary-compatibility/" + locale + "/" + identity + ".json", nil
}

func currentGlossaryReviewReference(root, locale, sha string) (GlossaryArchiveReference, error) {
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		return GlossaryArchiveReference{}, err
	}
	receipts, err := currentGlossaryReviewReceipts(root, locale, glossaryReviewGlossaryPath(locale), sha)
	if err != nil || len(receipts) != 1 || receipts[0].Decision != "passed" {
		return GlossaryArchiveReference{}, fmt.Errorf("compatibility requires a full independent current Glossary Review receipt")
	}
	path := "data/glossary-reviews/" + locale + "/" + receipts[0].ReviewID + ".review.json"
	data, err := readCompatibilityFile(root, path)
	if err != nil {
		return GlossaryArchiveReference{}, err
	}
	return GlossaryArchiveReference{path, sum(data)}, nil
}

func validateCompatibilityReview(root, locale, sha string, reference GlossaryArchiveReference) error {
	data, err := readCompatibilityFile(root, reference.Path)
	if err != nil {
		return err
	}
	if sum(data) != reference.SHA256 {
		return fmt.Errorf("compatibility Review hash mismatch")
	}
	var receipt GlossaryReviewReceipt
	if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &receipt); err != nil {
		return err
	}
	want := "data/glossary-reviews/" + locale + "/" + receipt.ReviewID + ".review.json"
	if reference.Path != want || receipt.GlossarySHA256 != sha || receipt.Decision != "passed" {
		return fmt.Errorf("compatibility full Review identity mismatch")
	}
	return validateGlossaryReviewReceipt(receipt, locale, glossaryReviewGlossaryPath(locale), filepath.Base(want))
}

func buildGlossaryCompatibilityEvidence(root, locale, oldSHA, newSHA string, review GlossaryArchiveReference, contexts []GlossaryCompatibilityContext, lineage []GlossaryArchiveReference) (*GlossaryCompatibilityEvidence, error) {
	oldBytes, err := readArchivedGlossary(root, locale, oldSHA)
	if err != nil {
		return nil, err
	}
	newBytes, err := readArchivedGlossary(root, locale, newSHA)
	if err != nil {
		return nil, err
	}
	if oldSHA == newSHA {
		return nil, fmt.Errorf("compatibility requires different glossary byte identities")
	}
	if err := validateCompatibilityReview(root, locale, newSHA, review); err != nil {
		return nil, err
	}
	delta, err := glossarySemanticDelta(locale, oldBytes, newBytes)
	if err != nil {
		return nil, err
	}
	oldPath, _ := glossaryArchivePath(locale, oldSHA)
	newPath, _ := glossaryArchivePath(locale, newSHA)
	e := &GlossaryCompatibilityEvidence{Schema: GlossaryCompatibilitySchema, EvidenceKind: "glossary-downstream-compatibility", Locale: locale, OldGlossary: GlossaryArchiveReference{oldPath, oldSHA}, NewGlossary: GlossaryArchiveReference{newPath, newSHA}, NewFullReview: review, NormalizationVersion: glossaryNormalizationVersion, ImpactAlgorithmVersion: glossaryImpactAlgorithmVersion, Delta: delta, Contexts: contexts, Affected: []GlossaryCompatibilityScope{}, Compatible: []GlossaryCompatibilityScope{}, Lineage: lineage}
	if e.Lineage == nil {
		e.Lineage = []GlossaryArchiveReference{}
	}
	for _, context := range contexts {
		ok, reasons := classifyGlossaryContext(context, delta)
		scope := GlossaryCompatibilityScope{context.Scope, context.Identity, reasons}
		if ok {
			e.Compatible = append(e.Compatible, scope)
		} else {
			e.Affected = append(e.Affected, scope)
		}
	}
	e.Identity = sum(mustJSON(e))
	return e, nil
}

// AssessGlossaryCompatibility builds immutable mechanical evidence only after
// the complete new glossary has its own current passed independent Review.
func AssessGlossaryCompatibility(root, locale, oldSHA string, catalog *Catalog) (*GlossaryCompatibilityEvidence, string, error) {
	newRef, err := ArchiveCurrentGlossary(root, locale)
	if err != nil {
		return nil, "", err
	}
	review, err := currentGlossaryReviewReference(root, locale, newRef.SHA256)
	if err != nil {
		return nil, "", err
	}
	contexts, err := currentGlossaryCompatibilityContexts(root, locale, catalog)
	if err != nil {
		return nil, "", err
	}
	previous, err := readGlossaryCompatibilityEvidenceSet(root, locale)
	if err != nil {
		return nil, "", err
	}
	lineage := []GlossaryArchiveReference{}
	var predecessor *GlossaryCompatibilityEvidence
	priorOldSHA := ""
	for _, e := range previous {
		if e.NewGlossary.SHA256 != oldSHA {
			continue
		}
		if priorOldSHA != "" && priorOldSHA != e.OldGlossary.SHA256 {
			return nil, "", fmt.Errorf("ambiguous prior compatibility lineage")
		}
		priorOldSHA = e.OldGlossary.SHA256
		if verifyGlossaryCompatibilityEvidence(root, locale, e, contexts, map[string]bool{}) == nil {
			if len(lineage) != 0 {
				return nil, "", fmt.Errorf("ambiguous current compatibility predecessor")
			}
			path, _ := glossaryCompatibilityEvidencePath(locale, e.Identity)
			data, err := readCompatibilityFile(root, path)
			if err != nil {
				return nil, "", err
			}
			lineage = []GlossaryArchiveReference{{path, sum(data)}}
		}
		if predecessor == nil {
			predecessor = e
		}
	}
	if predecessor != nil && len(lineage) == 0 {
		ref, err := rebuildHistoricalCompatibilityLineage(root, locale, predecessor, contexts, map[string]bool{})
		if err != nil {
			return nil, "", err
		}
		lineage = []GlossaryArchiveReference{ref}
	}
	e, err := buildGlossaryCompatibilityEvidence(root, locale, oldSHA, newRef.SHA256, review, contexts, lineage)
	if err != nil {
		return nil, "", err
	}
	path, _ := glossaryCompatibilityEvidencePath(locale, e.Identity)
	data, err := marshalGlossaryReviewJSON(e)
	if err != nil {
		return nil, "", err
	}
	if err := writeCompatibilityImmutable(root, path, data); err != nil {
		return nil, "", err
	}
	return e, path, nil
}

// A repaired context requires new immutable evidence. Re-assessment can
// re-certify the same historical edges against the complete current inventory;
// historical complete Review receipts still prove each archived intermediate
// glossary, while only the terminal glossary is CURRENT. Old evidence and
// reviewer ratings never change, and candidate identity still prevents a
// modified candidate from inheriting an old A.
func rebuildHistoricalCompatibilityLineage(root, locale string, e *GlossaryCompatibilityEvidence, contexts []GlossaryCompatibilityContext, seen map[string]bool) (GlossaryArchiveReference, error) {
	if seen[e.Identity] {
		return GlossaryArchiveReference{}, fmt.Errorf("compatibility lineage cycle")
	}
	seen[e.Identity] = true
	defer delete(seen, e.Identity)
	if err := verifyGlossaryCompatibilityEvidence(root, locale, e, e.Contexts, map[string]bool{}); err != nil {
		return GlossaryArchiveReference{}, err
	}
	refs := []GlossaryArchiveReference{}
	for _, ref := range e.Lineage {
		data, err := readCompatibilityFile(root, ref.Path)
		if err != nil || sum(data) != ref.SHA256 {
			return GlossaryArchiveReference{}, fmt.Errorf("historical predecessor unavailable")
		}
		var prior GlossaryCompatibilityEvidence
		if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &prior); err != nil {
			return GlossaryArchiveReference{}, err
		}
		next, err := rebuildHistoricalCompatibilityLineage(root, locale, &prior, contexts, seen)
		if err != nil {
			return GlossaryArchiveReference{}, err
		}
		refs = append(refs, next)
	}
	rebuilt, err := buildGlossaryCompatibilityEvidence(root, locale, e.OldGlossary.SHA256, e.NewGlossary.SHA256, e.NewFullReview, contexts, refs)
	if err != nil {
		return GlossaryArchiveReference{}, err
	}
	path, _ := glossaryCompatibilityEvidencePath(locale, rebuilt.Identity)
	data, err := marshalGlossaryReviewJSON(rebuilt)
	if err != nil {
		return GlossaryArchiveReference{}, err
	}
	if err := writeCompatibilityImmutable(root, path, data); err != nil {
		return GlossaryArchiveReference{}, err
	}
	return GlossaryArchiveReference{path, sum(data)}, nil
}

func readGlossaryCompatibilityEvidenceSet(root, locale string) ([]*GlossaryCompatibilityEvidence, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	directory := "data/glossary-compatibility/" + locale
	path, err := compatibilityPath(root, directory+"/.inventory", false)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	all := []*GlossaryCompatibilityEvidence{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			return nil, fmt.Errorf("unknown compatibility evidence member %s", entry.Name())
		}
		data, err := readCompatibilityFile(root, directory+"/"+entry.Name())
		if err != nil {
			return nil, err
		}
		var e GlossaryCompatibilityEvidence
		if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &e); err != nil {
			return nil, err
		}
		want, err := glossaryCompatibilityEvidencePath(locale, e.Identity)
		if err != nil || want != directory+"/"+entry.Name() {
			return nil, fmt.Errorf("compatibility evidence path/identity mismatch")
		}
		if err := verifyGlossaryCompatibilityEvidence(root, locale, &e, e.Contexts, map[string]bool{}); err != nil {
			return nil, err
		}
		all = append(all, &e)
	}
	return all, nil
}

func verifyGlossaryCompatibilityEvidence(root, locale string, e *GlossaryCompatibilityEvidence, contexts []GlossaryCompatibilityContext, seen map[string]bool) error {
	if seen[e.Identity] {
		return fmt.Errorf("compatibility lineage cycle")
	}
	seen[e.Identity] = true
	defer delete(seen, e.Identity)
	if e.Locale != locale || len(contexts) == 0 || !reflect.DeepEqual(e.Contexts, contexts) {
		return fmt.Errorf("compatibility context exact-set is stale")
	}
	for i, c := range contexts {
		if i > 0 && contexts[i-1].Scope >= c.Scope {
			return fmt.Errorf("context scope not sorted/unique")
		}
		copy := c
		copy.Identity = ""
		if !validSHA256(c.Identity) || c.Identity != sum(mustJSON(copy)) || len(c.References) == 0 {
			return fmt.Errorf("context identity invalid")
		}
		if c.Parser != "plain-visible/v1" && c.Parser != "present-visible/v1" && c.Parser != "go-comment-visible/v1" && c.Parser != "unknown" {
			return fmt.Errorf("unknown context parser identity")
		}
		for j, r := range c.References {
			if !validSHA256(r.SHA256) || validateGenerationInstallPath(r.Path) != nil || (j > 0 && c.References[j-1].Path >= r.Path) {
				return fmt.Errorf("invalid context reference")
			}
		}
	}
	want, err := buildGlossaryCompatibilityEvidence(root, locale, e.OldGlossary.SHA256, e.NewGlossary.SHA256, e.NewFullReview, contexts, e.Lineage)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(e, want) {
		return fmt.Errorf("compatibility schema/delta/classification/self identity mismatch")
	}
	if len(e.Lineage) > 1 {
		return fmt.Errorf("ambiguous compatibility predecessor")
	}
	for _, ref := range e.Lineage {
		data, err := readCompatibilityFile(root, ref.Path)
		if err != nil {
			return err
		}
		if sum(data) != ref.SHA256 {
			return fmt.Errorf("compatibility predecessor hash mismatch")
		}
		var prior GlossaryCompatibilityEvidence
		if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &prior); err != nil {
			return err
		}
		path, _ := glossaryCompatibilityEvidencePath(locale, prior.Identity)
		if path != ref.Path || prior.NewGlossary.SHA256 != e.OldGlossary.SHA256 {
			return fmt.Errorf("discontinuous compatibility lineage")
		}
		if err := verifyGlossaryCompatibilityEvidence(root, locale, &prior, contexts, seen); err != nil {
			return err
		}
	}
	return nil
}

// ResolveGlossaryCompatibility is the shared fail-closed API. A scope of "*"
// requires every context, including otherwise unmapped surfaces, compatible.
func ResolveGlossaryCompatibility(root, locale, oldSHA, newSHA, scope string, catalog *Catalog) GlossaryCompatibilityResolution {
	fail := func(status, reason string) GlossaryCompatibilityResolution {
		return GlossaryCompatibilityResolution{Status: status, Reason: reason, Chain: []GlossaryArchiveReference{}}
	}
	if !validSHA256(oldSHA) || !validSHA256(newSHA) || ValidateLocaleName(locale) != nil {
		return fail("stale evidence", "invalid_identity")
	}
	if oldSHA == newSHA {
		return fail("exact", "exact_current_glossary")
	}
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil || RequireCurrentGlossaryReview(root, locale) != nil {
		return fail("stale evidence", "full_review_not_current")
	}
	contexts, err := currentGlossaryCompatibilityContexts(root, locale, catalog)
	if err != nil {
		return fail("stale evidence", "context_changed")
	}
	all, err := readGlossaryCompatibilityEvidenceSet(root, locale)
	if err != nil {
		return fail("stale evidence", "invalid_evidence")
	}
	chain := []GlossaryArchiveReference{}
	seen := map[string]bool{}
	sha := oldSHA
	var priorRef *GlossaryArchiveReference
	terminalSHA := sum(current)
	requestedChainLength := -1
	for sha != terminalSHA {
		if seen[sha] {
			return fail("stale evidence", "ambiguous_lineage")
		}
		seen[sha] = true
		var selected *GlossaryCompatibilityEvidence
		branch := ""
		for _, e := range all {
			if e.OldGlossary.SHA256 == sha {
				if branch != "" && branch != e.NewGlossary.SHA256 {
					return fail("stale evidence", "ambiguous_lineage")
				}
				branch = e.NewGlossary.SHA256
				if verifyGlossaryCompatibilityEvidence(root, locale, e, contexts, map[string]bool{}) != nil {
					continue
				}
				if selected != nil {
					return fail("stale evidence", "ambiguous_lineage")
				}
				selected = e
			}
		}
		if selected == nil {
			if branch != "" {
				return fail("stale evidence", "invalid_evidence")
			}
			return fail("missing evidence", "missing_lineage")
		}
		if err := verifyGlossaryCompatibilityEvidence(root, locale, selected, contexts, map[string]bool{}); err != nil {
			return fail("stale evidence", "invalid_evidence")
		}
		if priorRef != nil && (len(selected.Lineage) != 1 || selected.Lineage[0] != *priorRef) {
			return fail("stale evidence", "missing_lineage")
		}
		ok := scope == "*" && len(selected.Affected) == 0
		for _, c := range selected.Compatible {
			if c.ID == scope {
				ok = true
			}
		}
		if !ok {
			return fail("affected", "affected_scope")
		}
		path, _ := glossaryCompatibilityEvidencePath(locale, selected.Identity)
		data, err := readCompatibilityFile(root, path)
		if err != nil {
			return fail("stale evidence", "invalid_evidence")
		}
		ref := GlossaryArchiveReference{path, sum(data)}
		chain = append(chain, ref)
		priorRef = &chain[len(chain)-1]
		sha = selected.NewGlossary.SHA256
		if sha == newSHA {
			requestedChainLength = len(chain)
		}
	}
	if requestedChainLength < 0 {
		return fail("stale evidence", "discontinuous_lineage")
	}
	chain = chain[:requestedChainLength]
	return GlossaryCompatibilityResolution{Status: "compatible", Reason: "verified_lineage", Chain: chain}
}

// Scope reasons remain enums; callers must never infer acceptance from detail.
func glossaryScopeCompatible(root, locale, oldSHA, newSHA, scope string, catalog *Catalog) bool {
	r := ResolveGlossaryCompatibility(root, locale, oldSHA, newSHA, scope, catalog)
	return r.Status == "exact" || r.Status == "compatible"
}

func courseGlossaryCompatibility(root, locale string, catalog *Catalog, currentSHA string) func(CoursePageMetadata) bool {
	return func(entry CoursePageMetadata) bool {
		return glossaryScopeCompatible(root, locale, entry.GlossarySHA256, currentSHA, "seo:"+entry.PageID, catalog) && glossaryScopeCompatible(root, locale, entry.GlossarySHA256, currentSHA, "seo-page:"+entry.PageID, catalog)
	}
}

func CheckGlossaryCompatibilityEvidence(root, locale, path string, catalog *Catalog) error {
	data, err := readCompatibilityFile(root, path)
	if err != nil {
		return err
	}
	var e GlossaryCompatibilityEvidence
	if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &e); err != nil {
		return err
	}
	want, err := glossaryCompatibilityEvidencePath(locale, e.Identity)
	if err != nil || want != path {
		return fmt.Errorf("compatibility path mismatch")
	}
	current, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return err
	}
	if sum(current) != e.NewGlossary.SHA256 {
		return fmt.Errorf("compatibility evidence glossary is stale")
	}
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		return err
	}
	contexts, err := currentGlossaryCompatibilityContexts(root, locale, catalog)
	if err != nil {
		return err
	}
	return verifyGlossaryCompatibilityEvidence(root, locale, &e, contexts, map[string]bool{})
}

func GlossaryCompatibilityStatus(root, locale, oldSHA, scope string, catalog *Catalog) GlossaryCompatibilityResolution {
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		return GlossaryCompatibilityResolution{Status: "stale evidence", Reason: "full_review_not_current", Chain: []GlossaryArchiveReference{}}
	}
	data, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return GlossaryCompatibilityResolution{Status: "stale evidence", Reason: "invalid_identity", Chain: []GlossaryArchiveReference{}}
	}
	return ResolveGlossaryCompatibility(root, locale, oldSHA, sum(data), scope, catalog)
}
