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

// PreparedGlossaryCompatibility owns the immutable compatibility world for one
// repository operation. It is intentionally invocation-local: callers that
// resolve many scopes share the expensive current Review, context inventory,
// evidence decoding, and evidence verification without introducing persistent
// cache invalidation semantics.
type PreparedGlossaryCompatibility struct {
	// Historical TU events retain full recorded inventories, but only TU
	// contexts are re-bound to repository bytes. Ordinary consumers remain full.
	tuOnly                 bool
	historicalEvidenceRefs map[string]string
	root                   string
	locale                 string
	catalog                *Catalog

	prepared       bool
	prepareStatus  string
	prepareReason  string
	terminalSHA    string
	edges          map[string]preparedGlossaryCompatibilityEdge
	evidenceByPath map[string]preparedGlossaryCompatibilityEvidence
	resolutions    map[glossaryCompatibilityResolutionKey]GlossaryCompatibilityResolution
	stats          *glossaryCompatibilityPrepareStats
	// Exact accepted candidates may have an older invocation glossary. Reuse
	// their sealed history and the shared current assessment, once per operation.
	candidateAssessments map[string]*CurrentTranslationAuthorization
	// Used only while rebuilding a historical finalized judgment. Never supplied
	// to SEO, Surface, or other derived-consumer freshness gates.
	historicalContexts        []GlossaryCompatibilityContext
	historicalTargets         map[string][]byte
	historicalBaselineTargets map[string][]byte
	historicalSHA             string
	historicalCourse          []byte
	disableSuccessor          bool
	successorChecked          bool
	successorCompatibility    *PreparedGlossaryCompatibility
}

type preparedGlossaryCompatibilityEdge struct {
	branches map[string]bool
	valid    []preparedGlossaryCompatibilityEvidence
	invalid  bool
}

type preparedGlossaryCompatibilityEvidence struct {
	evidence   *GlossaryCompatibilityEvidence
	reference  GlossaryArchiveReference
	compatible map[string]bool
	data       []byte
}

type glossaryCompatibilityResolutionKey struct {
	oldSHA string
	newSHA string
	scope  string
}

// glossaryCompatibilityPrepareStats is package-private test instrumentation.
// It counts expensive preparation stages, never workflow semantics.
type glossaryCompatibilityPrepareStats struct {
	PrepareCalls           int
	FullReviewChecks       int
	ContextInventoryBuilds int
	EvidenceSetReads       int
	EvidenceVerifications  int
	EdgePreparations       int
}

// PrepareGlossaryCompatibility creates an invocation-local resolver. Expensive
// preparation is lazy so exact-SHA transitions retain their historical cheap
// shortcut.
func PrepareGlossaryCompatibility(root, locale string, catalog *Catalog) *PreparedGlossaryCompatibility {
	return newPreparedGlossaryCompatibility(root, locale, catalog, nil)
}

func newPreparedGlossaryCompatibility(root, locale string, catalog *Catalog, stats *glossaryCompatibilityPrepareStats) *PreparedGlossaryCompatibility {
	return &PreparedGlossaryCompatibility{
		root: root, locale: locale, catalog: catalog, stats: stats,
		resolutions: map[glossaryCompatibilityResolutionKey]GlossaryCompatibilityResolution{},
	}
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
			if !compatibilitySourceTermPresent(source, d.Term) {
				reasons["changed_source_term_absent"] = true
				continue
			}
			// A newly added mapping may merely formalize language already present in
			// an accepted historical target. Keep this proof deliberately narrow:
			// preferred rules accept an exact target literal in the same formal
			// context, while mandatory rules only accept an exact source/target
			// label pair. Changed mappings and keep additions remain affected.
			if d.Old == nil && d.New != nil {
				switch d.Category {
				case "preferred":
					sourceStillVisible := d.Term != *d.New && compatibilityExactSurfaceTermPresent(target, d.Term)
					if compatibilityTargetLiteralPresent(target, *d.New) && !sourceStillVisible {
						reasons["preferred_target_already_present"] = true
						continue
					}
				case "mandatory":
					if compatibilityExactLabelSatisfied(source, target, d.Term, *d.New) {
						reasons["mandatory_exact_target_already_present"] = true
						continue
					}
				}
			}
			compatible = false
			reasons[d.Category+"_source_term_present"] = true
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
func compatibilityTargetLiteralPresent(text, term string) bool {
	text = strings.Join(strings.Fields(text), " ")
	term = strings.Join(strings.Fields(term), " ")
	if term == "" {
		return false
	}
	var first, last rune
	for _, r := range term {
		if first == 0 {
			first = r
		}
		last = r
	}
	for offset := 0; offset < len(text); {
		relative := strings.Index(text[offset:], term)
		if relative < 0 {
			return false
		}
		start := offset + relative
		end := start + len(term)
		beforeOK := !compatibilityASCIIWordRune(first) || forbiddenBoundaryBefore(text, start)
		afterOK := !compatibilityASCIIWordRune(last) || forbiddenBoundaryAfter(text, end)
		if beforeOK && afterOK {
			return true
		}
		offset = start + 1
	}
	return false
}

func compatibilityASCIIWordRune(r rune) bool {
	return r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
}

func compatibilityExactLabelSatisfied(source, target, term, want string) bool {
	return strings.TrimSpace(source) == term && strings.TrimSpace(target) == want
}

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

// Source rules are conservatively case-insensitive and tolerate punctuation
// between phrase tokens plus a narrow set of English inflections. Forbidden
// targets retain the validator's exact Unicode/apostrophe boundary semantics.
func compatibilitySourceTermPresent(text, term string) bool {
	normalize := func(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
	text, term = normalize(text), normalize(term)
	if term == "" {
		return true
	}
	if compatibilityExactSurfaceTermPresent(text, term) {
		return true
	}
	// Tokenization preserves punctuation collisions such as SQL-transaction and
	// SQL transaction without treating arbitrary string prefixes as word forms.
	words := func(s string) []string {
		return strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	}
	a, b := words(text), words(term)
	if len(b) == 0 {
		return true
	}
	for i := 0; i+len(b) <= len(a); i++ {
		match := true
		for j := range b {
			if !compatibilityEnglishInflectionEquivalent(a[i+j], b[j]) {
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

func compatibilityExactSurfaceTermPresent(text, term string) bool {
	text = strings.Join(strings.Fields(strings.ToLower(text)), " ")
	term = strings.Join(strings.Fields(strings.ToLower(term)), " ")
	if term == "" {
		return true
	}
	for offset := 0; offset < len(text); {
		position := strings.Index(text[offset:], term)
		if position < 0 {
			return false
		}
		start := offset + position
		end := start + len(term)
		if forbiddenTermBoundaries(text, start, end) {
			return true
		}
		offset = start + 1
	}
	return false
}

func compatibilityEnglishInflectionEquivalent(left, right string) bool {
	if left == right {
		return true
	}
	return compatibilityEnglishInflectionOf(left, right) || compatibilityEnglishInflectionOf(right, left)
}

func compatibilityEnglishInflectionOf(base, form string) bool {
	if !compatibilityASCIILetters(base) || !compatibilityASCIILetters(form) || len(base) < 2 {
		return false
	}
	forms := []string{base + "s", base + "ed", base + "ing"}
	if strings.HasSuffix(base, "y") && len(base) > 1 && compatibilityASCIIConsonant(base[len(base)-2]) {
		forms = append(forms, base[:len(base)-1]+"ies")
	}
	if strings.HasSuffix(base, "s") || strings.HasSuffix(base, "x") || strings.HasSuffix(base, "z") || strings.HasSuffix(base, "ch") || strings.HasSuffix(base, "sh") {
		forms = append(forms, base+"es")
	}
	if strings.HasSuffix(base, "e") {
		forms = append(forms, base+"d", base[:len(base)-1]+"ing")
	}
	if compatibilityEndsInDoublingPattern(base) {
		last := base[len(base)-1:]
		forms = append(forms, base+last+"ed", base+last+"ing")
	}
	for _, candidate := range forms {
		if form == candidate {
			return true
		}
	}
	return false
}

func compatibilityASCIILetters(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}
	return s != ""
}

func compatibilityASCIIConsonant(b byte) bool {
	return b >= 'a' && b <= 'z' && !strings.ContainsRune("aeiou", rune(b))
}

func compatibilityEndsInDoublingPattern(s string) bool {
	if len(s) < 3 {
		return false
	}
	a, b, c := s[len(s)-3], s[len(s)-2], s[len(s)-1]
	return compatibilityASCIIConsonant(a) && strings.ContainsRune("aeiou", rune(b)) && compatibilityASCIIConsonant(c) && !strings.ContainsRune("wxy", rune(c))
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
	if rs, err := currentUnifiedReviews(root, locale); err != nil {
		return GlossaryArchiveReference{}, err
	} else if len(rs) > 0 {
		return RequireUnifiedGlossaryReview(root, locale)
	}
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
	if strings.HasPrefix(reference.Path, "data/unified-glossary-reviews/") {
		var r UnifiedGlossaryReview
		if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &r); err != nil {
			return err
		}
		if err := validateUnifiedReview(r, locale); err != nil {
			return err
		}
		if err := validateUnifiedReviewBundle(root, r); err != nil {
			return err
		}
		if r.GlossarySHA != sha || r.Decision != "passed" || reference.Path != UnifiedGlossaryReviewPath(locale, r.ReviewID) {
			return fmt.Errorf("unified compatibility Review mismatch")
		}
		return nil
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
		identity := e.Identity
		e.Identity = ""
		if identity != sum(mustJSON(&e)) {
			return nil, fmt.Errorf("compatibility evidence self identity mismatch")
		}
		e.Identity = identity
		all = append(all, &e)
	}
	return all, nil
}

type glossaryCompatibilityVerificationResult struct {
	done bool
	err  error
}

func verifyGlossaryCompatibilityEvidence(root, locale string, e *GlossaryCompatibilityEvidence, contexts []GlossaryCompatibilityContext, seen map[string]bool) error {
	return verifyGlossaryCompatibilityEvidenceMemoized(root, locale, e, contexts, seen, nil)
}

func verifyGlossaryCompatibilityEvidenceMemoized(root, locale string, e *GlossaryCompatibilityEvidence, contexts []GlossaryCompatibilityContext, seen map[string]bool, memo map[string]glossaryCompatibilityVerificationResult) (resultErr error) {
	return verifyGlossaryCompatibilityEvidenceInWorld(root, locale, e, contexts, seen, memo, nil)
}

// Historical TU verification rebinds every immutable edge in its own recorded
// candidate world. Ordinary consumers still require one complete current world
// across the entire chain.
func verifyGlossaryCompatibilityEvidenceInWorld(root, locale string, e *GlossaryCompatibilityEvidence, contexts []GlossaryCompatibilityContext, seen map[string]bool, memo map[string]glossaryCompatibilityVerificationResult, historicalTU []GlossaryCompatibilityContext) (resultErr error) {
	if seen[e.Identity] {
		return fmt.Errorf("compatibility lineage cycle")
	}
	if cached, ok := memo[e.Identity]; ok && cached.done {
		return cached.err
	}
	seen[e.Identity] = true
	defer delete(seen, e.Identity)
	if memo != nil {
		defer func() {
			memo[e.Identity] = glossaryCompatibilityVerificationResult{done: true, err: resultErr}
		}()
	}
	if historicalTU != nil {
		if err := verifyHistoricalTUCompatibilityContexts(root, locale, e.Contexts, historicalTU); err != nil {
			return err
		}
		contexts = e.Contexts
	}
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
		if err := verifyGlossaryCompatibilityEvidenceInWorld(root, locale, &prior, contexts, seen, memo, historicalTU); err != nil {
			return err
		}
	}
	return nil
}

func verifyHistoricalTUCompatibilityContexts(root, locale string, recorded, current []GlossaryCompatibilityContext) error {
	index := 0
	for _, context := range recorded {
		if !strings.HasPrefix(context.Scope, "tu:") {
			continue
		}
		if index >= len(current) {
			return fmt.Errorf("historical TU context membership changed")
		}
		unit := current[index]
		index++
		if context.Scope != unit.Scope || context.Parser != unit.Parser || context.Source != unit.Source || len(context.References) != 2 {
			return fmt.Errorf("historical TU context source/membership changed")
		}
		// Source identity uses the current formal source bytes. The candidate
		// identity uses this edge's immutable selected path, never latest-batch
		// selection or current canonical bytes.
		var sourceRef GlossaryArchiveReference
		for _, ref := range unit.References {
			if !strings.HasPrefix(ref.Path, "data/retranslation-runs/"+locale+"/") && !strings.HasPrefix(ref.Path, "data/translation-unit-reconciliations/"+locale+"/") {
				sourceRef = ref
			}
		}
		foundSource, foundTarget := false, false
		for _, ref := range context.References {
			if ref == sourceRef {
				foundSource = true
				continue
			}
			if (!strings.HasPrefix(ref.Path, "data/retranslation-runs/"+locale+"/") && !strings.HasPrefix(ref.Path, "data/translation-unit-reconciliations/"+locale+"/")) || foundTarget {
				return fmt.Errorf("historical TU candidate reference changed")
			}
			candidate, err := readCompatibilityFile(root, ref.Path)
			if err != nil || sum(candidate) != ref.SHA256 || string(candidate) != context.Target {
				return fmt.Errorf("historical TU selected candidate changed: %s", ref.Path)
			}
			foundTarget = true
		}
		if !foundSource || !foundTarget {
			return fmt.Errorf("historical TU reference exact-set changed")
		}
	}
	if index != len(current) {
		return fmt.Errorf("historical TU context membership changed")
	}
	return nil
}

func compatibilityResolutionFailure(status, reason string) GlossaryCompatibilityResolution {
	return GlossaryCompatibilityResolution{Status: status, Reason: reason, Chain: []GlossaryArchiveReference{}}
}

func (p *PreparedGlossaryCompatibility) prepare() {
	if p.prepared {
		return
	}
	p.prepared = true
	if p.stats != nil {
		p.stats.PrepareCalls++
	}
	current, err := readCompatibilityFile(p.root, glossaryReviewGlossaryPath(p.locale))
	if p.historicalSHA != "" {
		current, err = readArchivedGlossary(p.root, p.locale, p.historicalSHA)
	}
	if p.stats != nil {
		p.stats.FullReviewChecks++
	}
	if err != nil || (p.historicalSHA == "" && RequireCurrentGlossaryReview(p.root, p.locale) != nil) {
		p.prepareStatus, p.prepareReason = "stale evidence", "full_review_not_current"
		return
	}
	if p.stats != nil {
		p.stats.ContextInventoryBuilds++
	}
	contexts, err := glossaryCompatibilityContextsWithCourse(p.root, p.locale, p.catalog, p.historicalCourse, p.tuOnly)
	if err == nil && p.historicalTargets != nil {
		contexts = historicalCanonicalContexts(contexts, p.historicalTargets)
	}
	if p.historicalContexts != nil {
		contexts, err = p.historicalContexts, nil
	}
	if err != nil {
		p.prepareStatus, p.prepareReason = "stale evidence", "context_changed"
		return
	}
	if p.stats != nil {
		p.stats.EvidenceSetReads++
	}
	all, err := readGlossaryCompatibilityEvidenceSet(p.root, p.locale)
	if err != nil {
		p.prepareStatus, p.prepareReason = "stale evidence", "invalid_evidence"
		return
	}
	p.terminalSHA = sum(current)
	p.edges = map[string]preparedGlossaryCompatibilityEdge{}
	p.evidenceByPath = map[string]preparedGlossaryCompatibilityEvidence{}
	verificationMemo := map[string]glossaryCompatibilityVerificationResult{}
	for _, evidence := range all {
		edge := p.edges[evidence.OldGlossary.SHA256]
		if edge.branches == nil {
			edge.branches = map[string]bool{}
		}
		edge.branches[evidence.NewGlossary.SHA256] = true
		if p.tuOnly && len(p.historicalEvidenceRefs) > 0 {
			path, _ := glossaryCompatibilityEvidencePath(p.locale, evidence.Identity)
			if _, bound := p.historicalEvidenceRefs[path]; !bound {
				p.edges[evidence.OldGlossary.SHA256] = edge
				continue
			}
		}
		if p.stats != nil {
			p.stats.EvidenceVerifications++
		}
		var historicalTU []GlossaryCompatibilityContext
		if p.tuOnly {
			historicalTU = contexts
		}
		verificationErr := verifyGlossaryCompatibilityEvidenceInWorld(p.root, p.locale, evidence, contexts, map[string]bool{}, verificationMemo, historicalTU)
		if verificationErr == nil && p.tuOnly {
			if len(p.historicalEvidenceRefs) > 0 {
				path, _ := glossaryCompatibilityEvidencePath(p.locale, evidence.Identity)
				data, readErr := readCompatibilityFile(p.root, path)
				if readErr != nil || p.historicalEvidenceRefs[path] != sum(data) {
					verificationErr = fmt.Errorf("compatibility context exact-set is stale: unbound historical inventory")
				}
			}
		}
		if verificationErr == nil {
			path, pathErr := glossaryCompatibilityEvidencePath(p.locale, evidence.Identity)
			data, readErr := readCompatibilityFile(p.root, path)
			if pathErr == nil && readErr == nil {
				compatible := make(map[string]bool, len(evidence.Compatible))
				for _, item := range evidence.Compatible {
					compatible[item.ID] = true
				}
				prepared := preparedGlossaryCompatibilityEvidence{
					evidence:   evidence,
					reference:  GlossaryArchiveReference{Path: path, SHA256: sum(data)},
					compatible: compatible,
					data:       append([]byte(nil), data...),
				}
				edge.valid = append(edge.valid, prepared)
				p.evidenceByPath[path] = prepared
			}
		} else if !strings.Contains(verificationErr.Error(), "compatibility context exact-set is stale") {
			// A superseded exact-set is expected after a legitimate context
			// reassessment. Integrity failures within an otherwise selectable
			// branch remain fail-closed even when another file looks valid.
			edge.invalid = true
		}
		p.edges[evidence.OldGlossary.SHA256] = edge
	}
	if p.stats != nil {
		p.stats.EdgePreparations = len(p.edges)
	}
}

func (p *PreparedGlossaryCompatibility) evidence(reference GlossaryArchiveReference) (*GlossaryCompatibilityEvidence, []byte, bool) {
	p.prepare()
	prepared, ok := p.evidenceByPath[reference.Path]
	if !ok || prepared.reference != reference {
		return nil, nil, false
	}
	return prepared.evidence, append([]byte(nil), prepared.data...), true
}

// Resolve uses the prepared invocation-local compatibility world. A scope of
// "*" requires every context, including otherwise unmapped surfaces,
// compatible.
func (p *PreparedGlossaryCompatibility) Resolve(oldSHA, newSHA, scope string) GlossaryCompatibilityResolution {
	if p != nil && p.tuOnly && !strings.HasPrefix(scope, "tu:") {
		return compatibilityResolutionFailure("stale evidence", "TU_event_does_not_authorize_derived_consumers")
	}
	result := p.resolve(oldSHA, newSHA, scope)
	if p != nil && !p.disableSuccessor && strings.HasPrefix(scope, "tu:") && result.Status != "exact" && result.Status != "compatible" {
		if !p.successorChecked {
			p.successorChecked = true
			if proof, err := findCurrentTargetSuccessor(p.root, p.locale, p.catalog); err == nil && proof != nil {
				p.successorCompatibility = proof.compatibility
			}
		}
		if p.successorCompatibility != nil {
			return p.successorCompatibility.Resolve(oldSHA, newSHA, scope)
		}
	}
	return result
}

func (p *PreparedGlossaryCompatibility) resolve(oldSHA, newSHA, scope string) GlossaryCompatibilityResolution {
	fail := func(status, reason string) GlossaryCompatibilityResolution {
		return compatibilityResolutionFailure(status, reason)
	}
	if p == nil || !validSHA256(oldSHA) || !validSHA256(newSHA) || ValidateLocaleName(p.locale) != nil {
		return fail("stale evidence", "invalid_identity")
	}
	if oldSHA == newSHA {
		return fail("exact", "exact_current_glossary")
	}
	key := glossaryCompatibilityResolutionKey{oldSHA: oldSHA, newSHA: newSHA, scope: scope}
	if cached, ok := p.resolutions[key]; ok {
		cached.Chain = append([]GlossaryArchiveReference(nil), cached.Chain...)
		return cached
	}
	p.prepare()
	if p.prepareStatus != "" {
		return fail(p.prepareStatus, p.prepareReason)
	}
	chain := []GlossaryArchiveReference{}
	seen := map[string]bool{}
	sha := oldSHA
	var priorRef *GlossaryArchiveReference
	requestedChainLength := -1
	for sha != p.terminalSHA {
		if seen[sha] {
			return fail("stale evidence", "ambiguous_lineage")
		}
		seen[sha] = true
		edge, exists := p.edges[sha]
		if len(edge.branches) > 1 {
			return fail("stale evidence", "ambiguous_lineage")
		}
		if edge.invalid {
			return fail("stale evidence", "invalid_evidence")
		}
		if len(edge.valid) == 0 {
			if exists && len(edge.branches) != 0 {
				return fail("stale evidence", "invalid_evidence")
			}
			return fail("missing evidence", "missing_lineage")
		}
		if len(edge.valid) > 1 {
			return fail("stale evidence", "ambiguous_lineage")
		}
		selected := edge.valid[0]
		if priorRef != nil && (len(selected.evidence.Lineage) != 1 || selected.evidence.Lineage[0] != *priorRef) {
			return fail("stale evidence", "missing_lineage")
		}
		ok := scope == "*" && len(selected.evidence.Affected) == 0
		ok = ok || selected.compatible[scope]
		if !ok {
			return fail("affected", "affected_scope")
		}
		chain = append(chain, selected.reference)
		priorRef = &chain[len(chain)-1]
		sha = selected.evidence.NewGlossary.SHA256
		if sha == newSHA {
			requestedChainLength = len(chain)
		}
	}
	if requestedChainLength < 0 {
		return fail("stale evidence", "discontinuous_lineage")
	}
	chain = chain[:requestedChainLength]
	result := GlossaryCompatibilityResolution{Status: "compatible", Reason: "verified_lineage", Chain: chain}
	p.resolutions[key] = result
	result.Chain = append([]GlossaryArchiveReference(nil), result.Chain...)
	return result
}

// ResolveGlossaryCompatibility is the single-call compatibility wrapper.
// Bulk consumers must share PrepareGlossaryCompatibility across their scope.
func ResolveGlossaryCompatibility(root, locale, oldSHA, newSHA, scope string, catalog *Catalog) GlossaryCompatibilityResolution {
	return PrepareGlossaryCompatibility(root, locale, catalog).Resolve(oldSHA, newSHA, scope)
}

// Scope reasons remain enums; callers must never infer acceptance from detail.
func glossaryScopeCompatible(root, locale, oldSHA, newSHA, scope string, catalog *Catalog) bool {
	r := ResolveGlossaryCompatibility(root, locale, oldSHA, newSHA, scope, catalog)
	return r.Status == "exact" || r.Status == "compatible"
}

func courseGlossaryCompatibility(root, locale string, catalog *Catalog, currentSHA string) func(CoursePageMetadata) bool {
	return courseGlossaryCompatibilityWithStats(root, locale, catalog, currentSHA, nil)
}

func courseGlossaryCompatibilityWithStats(root, locale string, catalog *Catalog, currentSHA string, stats *glossaryCompatibilityPrepareStats) func(CoursePageMetadata) bool {
	prepared := newPreparedGlossaryCompatibility(root, locale, catalog, stats)
	return func(entry CoursePageMetadata) bool {
		first := prepared.Resolve(entry.GlossarySHA256, currentSHA, "seo:"+entry.PageID)
		second := prepared.Resolve(entry.GlossarySHA256, currentSHA, "seo-page:"+entry.PageID)
		return (first.Status == "exact" || first.Status == "compatible") && (second.Status == "exact" || second.Status == "compatible")
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
