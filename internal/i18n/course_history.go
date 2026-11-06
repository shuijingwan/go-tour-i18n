package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

const CourseHistoryRecoverySchema = "go-learning/historical-course-seo-recovery/v1"
const CourseRefreshReceiptSchema = "go-learning/native-course-seo-refresh/v1"

// CourseSEOTransition is derived-consumer authority, never TU or Surface A.
// Recovery reconstructs a relation with maintainer authorization; it does not
// claim that an original refresh event was recorded by the old implementation.
type CourseSEOTransition struct {
	Schema              string                     `json:"schema"`
	Kind                string                     `json:"evidence_kind"`
	Reason              string                     `json:"reason"`
	Locale              string                     `json:"locale"`
	HistoricalClosure   *GlossaryArchiveReference  `json:"historical_closure,omitempty"`
	Old                 GlossaryArchiveReference   `json:"old_metadata"`
	New                 GlossaryArchiveReference   `json:"new_metadata"`
	Canonical           GlossaryArchiveReference   `json:"canonical_metadata"`
	SchemaVersion       int                        `json:"metadata_schema_version"`
	GlossarySHA         string                     `json:"glossary_sha256"`
	FullReview          GlossaryArchiveReference   `json:"full_glossary_review"`
	CurrentFinalization GlossaryArchiveReference   `json:"current_finalization"`
	TargetSuccessor     *GlossaryArchiveReference  `json:"target_successor,omitempty"`
	ContextsSHA         string                     `json:"pre_refresh_context_identity_sha256"`
	Compatibility       []GlossaryArchiveReference `json:"compatibility_chain"`
	Inputs              []GlossaryArchiveReference `json:"inputs"`
	StalePages          []string                   `json:"stale_pages"`
	Generation          CourseMetadataGeneration   `json:"generation"`
	ReconstructionSHA   string                     `json:"reconstruction_sha256"`
	Identity            string                     `json:"identity_sha256"`
}

func courseMetadataRelativePath(locale string) string {
	return "locales/" + locale + "/course-metadata.json"
}
func courseHistoryPath(locale, sha string) string {
	return "data/course-seo/history/" + locale + "/" + sha + ".json"
}
func courseTransitionPath(e *CourseSEOTransition) string {
	return "data/course-seo/transitions/" + e.Locale + "/" + e.Identity + ".json"
}
func courseTransitionIdentity(e CourseSEOTransition) string { e.Identity = ""; return sum(mustJSON(e)) }

// This is byte recovery, not current derived-consumer authorization. A caller
// must have an immutable ref to sha. Supplied bytes are invocation-local only.
func historicalCourseBytes(root, locale, sha string, supplied ...[]byte) ([]byte, error) {
	if ValidateLocaleName(locale) != nil || !validSHA256(sha) {
		return nil, fmt.Errorf("invalid Course SEO archive identity")
	}
	for _, b := range supplied {
		if b != nil && sum(b) == sha {
			return b, nil
		}
	}
	b, err := readCompatibilityFile(root, courseMetadataRelativePath(locale))
	if err == nil && sum(b) == sha {
		return b, nil
	}
	b, err = readCompatibilityFile(root, courseHistoryPath(locale, sha))
	if err != nil || sum(b) != sha {
		return nil, fmt.Errorf("missing or changed exact Course SEO archive %s", sha)
	}
	return b, nil
}

func courseLegacyReference(root, locale string) (GlossaryArchiveReference, string, error) {
	p := "locales/" + locale + "/content-scope.json"
	b, err := readCompatibilityFile(root, p)
	if err != nil {
		return GlossaryArchiveReference{}, "", err
	}
	var l contentidentity.Locale
	if err := contentidentity.StrictJSON(b, &l); err != nil {
		return GlossaryArchiveReference{}, "", err
	}
	if l.Locale != locale || l.Schema != contentidentity.LocaleSchema {
		return GlossaryArchiveReference{}, "", fmt.Errorf("wrong historical closure locale/schema")
	}
	sha := ""
	count := 0
	for _, c := range l.Packages {
		if c.Package != "tour-v1" {
			continue
		}
		count++
		if c.EvidenceKind != "legacy-tour-closure/v1" || c.State != "complete" || sum(mustJSON(c.Evidence)) != c.EvidenceIdentity {
			return GlossaryArchiveReference{}, "", fmt.Errorf("invalid historical closure")
		}
		for _, r := range c.Evidence {
			if r.Path == courseMetadataRelativePath(locale) {
				if sha != "" {
					return GlossaryArchiveReference{}, "", fmt.Errorf("duplicate historical Course SEO ref")
				}
				sha = r.SHA256
			}
		}
	}
	if count != 1 || !validSHA256(sha) {
		return GlossaryArchiveReference{}, "", fmt.Errorf("missing historical Course SEO ref")
	}
	return GlossaryArchiveReference{p, sum(b)}, sha, nil
}

func validateCourseTransitionIdentity(e *CourseSEOTransition, path string) error {
	validKind := e.Schema == CourseHistoryRecoverySchema && e.Kind == "historical-course-seo-recovery" && e.Reason == "missing-original-refresh-receipt" && e.HistoricalClosure != nil
	validKind = validKind || e.Schema == CourseRefreshReceiptSchema && e.Kind == "native-course-seo-refresh" && e.Reason == "formal-course-seo-refresh" && e.HistoricalClosure == nil
	if !validKind || ValidateLocaleName(e.Locale) != nil || !validSHA256(e.Identity) || courseTransitionIdentity(*e) != e.Identity || courseTransitionPath(e) != path || e.Old.Path != courseHistoryPath(e.Locale, e.Old.SHA256) || e.New.Path != courseHistoryPath(e.Locale, e.New.SHA256) || e.Canonical.Path != courseMetadataRelativePath(e.Locale) || e.Canonical.SHA256 != e.New.SHA256 || e.Old.SHA256 == e.New.SHA256 {
		return fmt.Errorf("invalid Course SEO transition schema/kind/identity")
	}
	return nil
}

// Reconstruct the exact pre-refresh world, then reuse the production refresh
// implementation. Missing/stale evidence is never reinterpreted as affected.
func buildCourseTransition(root, locale string, cat *Catalog, old, current []byte, recovery bool) (*CourseSEOTransition, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if err := CheckCourseRefreshTransactionState(root, locale); err != nil {
		return nil, err
	}
	if err := CheckPromotionTransactionState(root, locale); err != nil {
		return nil, err
	}
	if err := RequireCurrentGlossaryReview(root, locale); err != nil {
		return nil, err
	}
	base, err := decodeCourseMetadata(old)
	if err != nil {
		return nil, err
	}
	if err := contentidentity.StrictJSON(old, base); err != nil {
		return nil, err
	}
	result, err := decodeCourseMetadata(current)
	if err != nil {
		return nil, err
	}
	if err := contentidentity.StrictJSON(current, result); err != nil {
		return nil, err
	}
	if base.Locale != locale || result.Locale != locale || base.SchemaVersion != result.SchemaVersion {
		return nil, fmt.Errorf("Course SEO transition locale/schema mismatch")
	}
	contract, _, err := courseMetadataContract(base.SchemaVersion)
	if err != nil {
		return nil, err
	}
	if base.GeneratorContract != contract || result.GeneratorContract != contract {
		return nil, fmt.Errorf("unsupported Course SEO transition contract")
	}
	e := &CourseSEOTransition{Schema: CourseRefreshReceiptSchema, Kind: "native-course-seo-refresh", Reason: "formal-course-seo-refresh", Locale: locale, SchemaVersion: base.SchemaVersion, Old: GlossaryArchiveReference{courseHistoryPath(locale, sum(old)), sum(old)}, New: GlossaryArchiveReference{courseHistoryPath(locale, sum(current)), sum(current)}, Canonical: GlossaryArchiveReference{courseMetadataRelativePath(locale), sum(current)}, Compatibility: []GlossaryArchiveReference{}, Inputs: []GlossaryArchiveReference{}}
	if recovery {
		r, sha, err := courseLegacyReference(root, locale)
		if err != nil {
			return nil, err
		}
		if sum(old) != sha {
			return nil, fmt.Errorf("supplied old Course SEO bytes do not match immutable closure SHA")
		}
		e.Schema, e.Kind, e.Reason = CourseHistoryRecoverySchema, "historical-course-seo-recovery", "missing-original-refresh-receipt"
		e.HistoricalClosure = &r
		if _, _, err := historicalCanonicalTargets(root, locale, cat, old); err != nil {
			return nil, err
		}
	}
	contexts, err := glossaryCompatibilityContextsWithCourse(root, locale, cat, old)
	if err != nil {
		return nil, err
	}
	e.ContextsSHA = sum(mustJSON(contexts))
	glossary, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return nil, err
	}
	e.GlossarySHA = sum(glossary)
	e.FullReview, err = currentGlossaryReviewReference(root, locale, e.GlossarySHA)
	if err != nil {
		return nil, err
	}
	p := PrepareGlossaryCompatibility(root, locale, cat)
	p.disableSuccessor = true
	p.historicalContexts = contexts
	chains := map[string]GlossaryArchiveReference{}
	compat := func(entry CoursePageMetadata) bool {
		a := p.Resolve(entry.GlossarySHA256, e.GlossarySHA, "seo:"+entry.PageID)
		b := p.Resolve(entry.GlossarySHA256, e.GlossarySHA, "seo-page:"+entry.PageID)
		for _, r := range append(a.Chain, b.Chain...) {
			chains[r.Path] = r
		}
		return (a.Status == "exact" || a.Status == "compatible") && (b.Status == "exact" || b.Status == "compatible")
	}
	for _, entry := range base.Pages {
		for _, scope := range []string{"seo:" + entry.PageID, "seo-page:" + entry.PageID} {
			r := p.Resolve(entry.GlossarySHA256, e.GlossarySHA, scope)
			if r.Status != "exact" && r.Status != "compatible" && r.Status != "affected" {
				return nil, fmt.Errorf("pre-refresh compatibility %s: %s (%s)", scope, r.Status, r.Reason)
			}
		}
	}
	// The real changes supply descriptions/provenance, never their identities.
	if _, err := courseMetadataBaseIndex(result, cat); err != nil {
		return nil, err
	}
	replacements := courseDescriptionsFile{Pages: []courseDescriptionEntry{}}
	baseByID, err := courseMetadataBaseIndex(base, cat)
	if err != nil {
		return nil, err
	}
	haveGeneration := false
	for _, entry := range result.Pages {
		if reflect.DeepEqual(entry, baseByID[entry.PageID]) {
			continue
		}
		replacements.Pages = append(replacements.Pages, courseDescriptionEntry{entry.PageID, entry.Description})
		if !haveGeneration {
			e.Generation = entry.Generation
			haveGeneration = true
		} else if e.Generation != entry.Generation {
			return nil, fmt.Errorf("ambiguous Course SEO replacement provenance")
		}
	}
	if !haveGeneration {
		return nil, fmt.Errorf("no changed Course SEO entries")
	}
	options := CourseMetadataRefreshOptions{SchemaVersion: base.SchemaVersion, Locale: locale, Provider: e.Generation.Provider, Model: e.Generation.Model, GeneratedAt: e.Generation.GeneratedAt, Descriptions: mustJSON(replacements)}
	reconstructed, stale, err := refreshCourseMetadataFromBase(root, cat, options, old, compat, nil)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(reconstructed, current) {
		return nil, fmt.Errorf("Course SEO is not the exact deterministic stale-subset refresh")
	}
	e.StalePages = stale
	e.ReconstructionSHA = sum(reconstructed)
	for _, r := range chains {
		e.Compatibility = append(e.Compatibility, r)
	}
	sort.Slice(e.Compatibility, func(i, j int) bool { return e.Compatibility[i].Path < e.Compatibility[j].Path })
	// Exact canonical targets need independent finalized TU authorization. A
	// Course SEO recovery cannot itself authorize a Page target.
	proof, err := findCurrentTargetSuccessor(root, locale, cat, old)
	if err != nil {
		return nil, err
	}
	var oldTargets map[string][]byte
	if proof != nil {
		path, _ := successorPath(proof.evidence)
		r, err := successorReference(root, path)
		if err != nil {
			return nil, err
		}
		e.TargetSuccessor = &r
		oldTargets = proof.oldTargets
	}
	ids, err := qualityCheckFinalizedSnapshotIDs(root, locale)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		s, err := readQualityCheckSnapshot(root, locale, id, false)
		if err != nil {
			return nil, err
		}
		if s.GlossarySHA256 != e.GlossarySHA {
			continue
		}
		p, _ := historicalSuccessorCompatibility(root, locale, cat, oldTargets)
		p.historicalCourse = old
		f, err := verifySuccessorFinalization(root, locale, id, cat, p)
		if err != nil {
			return nil, fmt.Errorf("current finalized TU authority: %w", err)
		}
		match := len(f.Units) > 0
		for _, u := range f.Units {
			unit, err := cat.Unit(u.UnitID)
			if err != nil {
				return nil, err
			}
			selected, err := readCompatibilityFile(root, u.Snapshot.CandidatePath)
			if err != nil {
				return nil, err
			}
			if unit.Kind == UnitKindPage {
				selected = canonicalizeCandidateEOF(selected)
			}
			path, _ := canonicalTranslationUnitCandidatePath(locale, unit)
			b, err := readCompatibilityFile(root, path)
			if err != nil || !bytes.Equal(b, selected) {
				match = false
				break
			}
			r := PrepareGlossaryCompatibility(root, locale, cat).Resolve(s.GlossarySHA256, e.GlossarySHA, "tu:"+u.UnitID)
			if r.Status != "exact" && r.Status != "compatible" {
				match = false
				break
			}
		}
		if match {
			e.CurrentFinalization, err = successorReference(root, filepath.ToSlash(qualityCheckFinalizationPath("", locale, id)))
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if e.CurrentFinalization.Path == "" {
		return nil, fmt.Errorf("missing current finalized canonical TU authority")
	}
	inputPaths := map[string]bool{glossaryReviewGlossaryPath(locale): true}
	for _, c := range contexts {
		for _, r := range c.References {
			if r.Path != courseMetadataRelativePath(locale) {
				inputPaths[r.Path] = true
			}
		}
	}
	if base.SchemaVersion == 2 {
		inputPaths[courseSourceDescriptionsRelativePath] = true
	}
	keys := []string{}
	for path := range inputPaths {
		keys = append(keys, path)
	}
	sort.Strings(keys)
	for _, path := range keys {
		r, err := successorReference(root, path)
		if err != nil {
			return nil, err
		}
		e.Inputs = append(e.Inputs, r)
	}
	e.Identity = courseTransitionIdentity(*e)
	return e, nil
}

// RecoverCourseMetadataHistory requires explicitly supplied bytes; never Git.
func RecoverCourseMetadataHistory(root string, cat *Catalog, locale string, old []byte, apply bool) (*CourseSEOTransition, string, error) {
	current, err := readCompatibilityFile(root, courseMetadataRelativePath(locale))
	if err != nil {
		return nil, "", err
	}
	e, err := buildCourseTransition(root, locale, cat, old, current, true)
	if err != nil {
		return nil, "", err
	}
	if _, err := LoadCourseMetadata(root, locale, cat); err != nil {
		return nil, "", err
	}
	path := courseTransitionPath(e)
	if apply {
		for _, item := range []struct {
			ref  GlossaryArchiveReference
			data []byte
		}{{e.Old, old}, {e.New, current}} {
			if err := saveCourseHistory(root, item.ref, item.data); err != nil {
				return nil, "", err
			}
		}
		// Last precommit: no mutable input changed since the proof was built.
		if err := checkCourseTransitionInputs(root, e); err != nil {
			return nil, "", err
		}
		actual, err := readCompatibilityFile(root, e.Canonical.Path)
		if err != nil || sum(actual) != e.Canonical.SHA256 {
			return nil, "", fmt.Errorf("current Course SEO changed during recovery preflight")
		}
		p, err := compatibilityPath(root, path, true)
		if err != nil {
			return nil, "", err
		}
		b, _ := json.MarshalIndent(e, "", "  ")
		if err := writeNewGenerationFile(p, append(b, '\n')); err != nil {
			return nil, "", err
		}
	}
	return e, path, nil
}

func saveCourseHistory(root string, r GlossaryArchiveReference, b []byte) error {
	if sum(b) != r.SHA256 {
		return fmt.Errorf("Course SEO archive SHA mismatch")
	}
	p, err := compatibilityPath(root, r.Path, true)
	if err != nil {
		return err
	}
	if existing, err := readCompatibilityFile(root, r.Path); err == nil {
		if bytes.Equal(existing, b) {
			return nil
		}
		return fmt.Errorf("refuse to overwrite Course SEO archive")
	} else if !os.IsNotExist(err) {
		return err
	}
	return writeNewGenerationFile(p, b)
}
func checkCourseTransitionInputs(root string, e *CourseSEOTransition) error {
	refs := append([]GlossaryArchiveReference{e.Old, e.New, e.FullReview, e.CurrentFinalization}, e.Inputs...)
	refs = append(refs, e.Compatibility...)
	if e.TargetSuccessor != nil {
		refs = append(refs, *e.TargetSuccessor)
	}
	if e.HistoricalClosure != nil {
		refs = append(refs, *e.HistoricalClosure)
	}
	for _, r := range refs {
		b, err := readCompatibilityFile(root, r.Path)
		if err != nil || sum(b) != r.SHA256 {
			return fmt.Errorf("Course SEO transition input changed %s", r.Path)
		}
	}
	return nil
}

// AuthorizeHistoricalCourseMetadataReference is separate from TU successor.
// Current strict validation and exact deterministic reconstruction are required
// in addition to finding an archive. No new Surface Review decision is granted.
func AuthorizeHistoricalCourseMetadataReference(root, locale, oldSHA string, cat *Catalog) ([]byte, error) {
	if err := CheckCourseRefreshTransactionState(root, locale); err != nil {
		return nil, err
	}
	old, err := historicalCourseBytes(root, locale, oldSHA)
	if err != nil {
		return nil, err
	}
	current, err := readCompatibilityFile(root, courseMetadataRelativePath(locale))
	if err != nil {
		return nil, err
	}
	if sum(current) == oldSHA {
		return old, nil
	}
	directory := "data/course-seo/transitions/" + locale
	p, err := compatibilityPath(root, directory+"/.inventory", false)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		return nil, fmt.Errorf("missing Course SEO transition: %w", err)
	}
	var found *CourseSEOTransition
	for _, entry := range entries {
		path := directory + "/" + entry.Name()
		b, err := readCompatibilityFile(root, path)
		if err != nil {
			return nil, err
		}
		var e CourseSEOTransition
		if err := contentidentity.StrictJSON(b, &e); err != nil {
			return nil, err
		}
		if err := validateCourseTransitionIdentity(&e, path); err != nil {
			return nil, err
		}
		if e.Locale != locale {
			return nil, fmt.Errorf("wrong transition locale")
		}
		if e.Old.SHA256 != oldSHA {
			continue
		}
		if found != nil {
			return nil, fmt.Errorf("ambiguous/forked Course SEO transition")
		}
		found = &e
	}
	if found == nil || found.New.SHA256 != sum(current) {
		return nil, fmt.Errorf("missing exact Course SEO successor authority")
	}
	if err := checkCourseTransitionInputs(root, found); err != nil {
		return nil, err
	}
	newBytes, err := historicalCourseBytes(root, locale, found.New.SHA256)
	if err != nil {
		return nil, err
	}
	want, err := buildCourseTransition(root, locale, cat, old, newBytes, found.Schema == CourseHistoryRecoverySchema)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(found, want) {
		return nil, fmt.Errorf("Course SEO transition proof changed")
	}
	if _, err := LoadCourseMetadata(root, locale, cat); err != nil {
		return nil, err
	}
	return old, nil
}

func CheckCourseRefreshTransactionState(root, locale string) error {
	if err := ValidateLocaleName(locale); err != nil {
		return err
	}
	p, err := compatibilityPath(root, "locales/"+locale+"/.inventory", false)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".course-refresh-staging-") {
			return fmt.Errorf("Course SEO refresh transaction pending/unrecovered: %s", e.Name())
		}
	}
	return nil
}
