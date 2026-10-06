package i18n

import (
	"bytes"
	"fmt"
	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"os"
	"sort"
	"strings"
)

const UnifiedGlossaryReviewSchema = "go-learning/unified-glossary-review/v1"

type UnifiedGlossaryReview struct {
	Schema            string   `json:"schema"`
	Locale            string   `json:"locale"`
	GlossaryPath      string   `json:"glossary_path"`
	GlossarySHA       string   `json:"glossary_sha256"`
	CorpusSHA         string   `json:"corpus_identity_sha256"`
	InputSHA          string   `json:"input_identity_sha256"`
	BundleSHA         string   `json:"reviewer_bundle_sha256"`
	ReviewID          string   `json:"review_id"`
	Reviewer          string   `json:"reviewer"`
	GenerationSession string   `json:"generation_session"`
	Decision          string   `json:"decision"`
	Rubric            string   `json:"rubric"`
	Findings          []string `json:"findings"`
	Identity          string   `json:"identity_sha256"`
}

func unifiedReviewIdentity(r UnifiedGlossaryReview) string { r.Identity = ""; return sum(mustJSON(r)) }
func validateUnifiedReview(r UnifiedGlossaryReview, locale string) error {
	if r.Schema != UnifiedGlossaryReviewSchema || r.Locale != locale || r.GlossaryPath != glossaryReviewGlossaryPath(locale) || !reviewIDPattern.MatchString(r.ReviewID) || strings.TrimSpace(r.Reviewer) == "" || r.GenerationSession == "" || r.Reviewer == r.GenerationSession || r.Rubric != "unified-glossary-review/v1" || !validSHA256(r.GlossarySHA) || !validSHA256(r.CorpusSHA) || !validSHA256(r.InputSHA) || !validSHA256(r.BundleSHA) || r.Identity != unifiedReviewIdentity(r) {
		return fmt.Errorf("invalid unified Review identity/provenance")
	}
	if (r.Decision != "passed" && r.Decision != "failed") || (r.Decision == "passed" && len(r.Findings) != 0) || (r.Decision == "failed" && len(r.Findings) == 0) {
		return fmt.Errorf("invalid independent Review decision/findings")
	}
	return nil
}
func UnifiedGlossaryReviewPath(locale, id string) string {
	return "data/unified-glossary-reviews/" + locale + "/" + id + ".review.json"
}
func currentUnifiedReviews(root, locale string) ([]UnifiedGlossaryReview, error) {
	entries, err := os.ReadDir(root + "/data/unified-glossary-reviews/" + locale)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	corpus, err := LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, err
	}
	b, err := readCompatibilityFile(root, glossaryReviewGlossaryPath(locale))
	if err != nil {
		return nil, err
	}
	sha := sum(b)
	out := []UnifiedGlossaryReview{}
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".review.json") {
			return nil, fmt.Errorf("unknown unified Review artifact")
		}
		var r UnifiedGlossaryReview
		b, err := readCompatibilityFile(root, "data/unified-glossary-reviews/"+locale+"/"+entry.Name())
		if err != nil {
			return nil, err
		}
		if err := contentidentity.StrictJSON(b, &r); err != nil {
			return nil, err
		}
		if err := validateUnifiedReview(r, locale); err != nil {
			return nil, err
		}
		if err := validateUnifiedReviewBundle(root, r); err != nil {
			return nil, err
		}
		if entry.Name() != r.ReviewID+".review.json" {
			return nil, fmt.Errorf("unified Review filename mismatch")
		}
		if r.GlossarySHA == sha && r.CorpusSHA == corpus.Identity {
			out = append(out, r)
		}
	}
	return out, nil
}
func RequireUnifiedGlossaryReview(root, locale string) (GlossaryArchiveReference, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return GlossaryArchiveReference{}, err
	}
	if _, err := LoadGlossary(root, locale); err != nil {
		return GlossaryArchiveReference{}, err
	}
	rs, err := currentUnifiedReviews(root, locale)
	if err != nil {
		return GlossaryArchiveReference{}, err
	}
	if len(rs) != 1 || rs[0].Decision != "passed" {
		return GlossaryArchiveReference{}, fmt.Errorf("unified Glossary Review missing/stale/ambiguous or non-passed: %s", locale)
	}
	p := UnifiedGlossaryReviewPath(locale, rs[0].ReviewID)
	b, err := readCompatibilityFile(root, p)
	return GlossaryArchiveReference{Path: p, SHA256: sum(b)}, err
}

type UnifiedGlossaryBundle struct {
	ProvenanceContract string                `json:"provenance_contract"`
	Schema             string                `json:"schema"`
	Role               string                `json:"role"`
	Locale             string                `json:"locale"`
	CorpusSHA          string                `json:"corpus_identity_sha256"`
	GlossarySHA        string                `json:"glossary_sha256"`
	ExpectedOutputs    []string              `json:"expected_outputs"`
	Files              []TransportBundleFile `json:"files"`
	Identity           string                `json:"input_identity_sha256"`
}

func ValidateLocaleIdentityBytes(locale string, b []byte) error {
	var l Locale
	if err := contentidentity.StrictJSON(b, &l); err != nil {
		return err
	}
	if l.Locale != locale || l.LanguageName == "" || l.EnglishName == "" || l.HTMLLang == "" || l.TranslationUnit != "present.Section" {
		return fmt.Errorf("invalid unified locale identity")
	}
	return ValidateLocaleName(l.HTMLLang)
}

func ExportUnifiedGlossaryBundle(root, locale, role string) ([]byte, *UnifiedGlossaryBundle, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, nil, err
	}
	if role != "generation" && role != "reviewer" {
		return nil, nil, fmt.Errorf("invalid glossary role")
	}
	corpus, err := LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, nil, err
	}
	if role == "reviewer" {
		if _, err := LoadGlossary(root, locale); err != nil {
			return nil, nil, err
		}
	}
	m := &UnifiedGlossaryBundle{Schema: "go-learning/unified-glossary-bundle/v1", Role: role, Locale: locale, CorpusSHA: corpus.Identity, ExpectedOutputs: []string{}, Files: []TransportBundleFile{}}
	if role == "generation" {
		m.ExpectedOutputs = []string{glossaryReviewGlossaryPath(locale)}
		m.ProvenanceContract = "chatgpt|codex:gpt-5.6-sol-high:unified-locale-session/v1"
	} else {
		m.ProvenanceContract = "chatgpt:gpt-5.6-sol-high:independent-locale-session:findings-only/v1"
	}
	paths := append([]string{GlossaryCorpusPath, glossaryReviewGlossaryPath(locale), "locales/" + locale + "/locale.json"}, glossaryReviewerAuthorityPaths...)
	paths = append(paths, "docs/SITE_V2_WORKFLOW.md", "docs/SITE_V2_ARCHITECTURE.md", "docs/GLOSSARY_COMPATIBILITY.md", "docs/TRANSLATION_WORKFLOW.md")
	paths = append(paths, "docs/CODEX_TRANSLATION.md", "docs/CHATGPT_LANGUAGE_GENERATION.md")
	entries := []TransportBundleEntry{}
	for _, p := range paths {
		b, err := readCompatibilityFile(root, p)
		if err != nil {
			return nil, nil, err
		}
		if p == "locales/"+locale+"/locale.json" {
			if err := ValidateLocaleIdentityBytes(locale, b); err != nil {
				return nil, nil, err
			}
		}
		name := "inputs/" + p
		entries = append(entries, TransportBundleEntry{Path: name, Data: b})
		m.Files = append(m.Files, NewTransportBundleFile(name, p, b))
		if p == glossaryReviewGlossaryPath(locale) {
			m.GlossarySHA = sum(b)
		}
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].BundlePath < m.Files[j].BundlePath })
	m.Identity = sum(mustJSON(m))
	b, err := WriteDeterministicTransportBundle(append(mustJSON(m), '\n'), entries)
	return b, m, err
}
func CheckUnifiedGlossaryBundle(root string, b []byte) (*UnifiedGlossaryBundle, error) {
	fs, err := ReadTransportBundle(b, 128, 32<<20)
	if err != nil {
		return nil, err
	}
	var m UnifiedGlossaryBundle
	if err := contentidentity.StrictJSON(fs["manifest.json"], &m); err != nil {
		return nil, err
	}
	if err := ValidateTransportBundleInventory(fs, m.Files, true); err != nil {
		return nil, err
	}
	want, _, err := ExportUnifiedGlossaryBundle(root, m.Locale, m.Role)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(want, b) {
		return nil, fmt.Errorf("unified glossary bundle stale/noncanonical")
	}
	return &m, nil
}
func RecordUnifiedGlossaryReview(root string, bundle []byte, id, reviewer, generationSession, decision string, findings []string) (*UnifiedGlossaryReview, string, error) {
	m, err := CheckUnifiedGlossaryBundle(root, bundle)
	if err != nil {
		return nil, "", err
	}
	if m.Role != "reviewer" {
		return nil, "", fmt.Errorf("requires independent Reviewer bundle")
	}
	r := &UnifiedGlossaryReview{Schema: UnifiedGlossaryReviewSchema, Locale: m.Locale, GlossaryPath: glossaryReviewGlossaryPath(m.Locale), GlossarySHA: m.GlossarySHA, CorpusSHA: m.CorpusSHA, InputSHA: m.Identity, BundleSHA: sum(bundle), ReviewID: id, Reviewer: reviewer, GenerationSession: generationSession, Decision: decision, Rubric: "unified-glossary-review/v1", Findings: findings}
	if r.Findings == nil {
		r.Findings = []string{}
	}
	r.Identity = unifiedReviewIdentity(*r)
	if err := validateUnifiedReview(*r, m.Locale); err != nil {
		return nil, "", err
	}
	if err := CheckUnifiedLanguageRole(root, m.Locale, reviewer, "reviewer"); err != nil {
		return nil, "", err
	}
	if err := CheckUnifiedLanguageRole(root, m.Locale, generationSession, "generation"); err != nil {
		return nil, "", err
	}
	if err := writeCompatibilityImmutable(root, unifiedReviewerArchive(m.Locale, r.BundleSHA), bundle); err != nil {
		return nil, "", err
	}
	if _, err := archiveGlossaryBytes(root, m.Locale, []byte(func() string {
		fs, _ := ReadTransportBundle(bundle, 128, 32<<20)
		return string(fs["inputs/"+r.GlossaryPath])
	}())); err != nil {
		return nil, "", err
	}
	p := UnifiedGlossaryReviewPath(m.Locale, id)
	err = writeCompatibilityImmutable(root, p, append(mustJSON(r), '\n'))
	return r, p, err
}

// Coverage coexistence does not merge language roles. Existing unified receipt
// binds the long-term Generation and independent Reviewer sessions.
func CheckUnifiedLanguageRole(root, locale, session, role string) error {
	entries, err := os.ReadDir(root + "/data/unified-glossary-reviews/" + locale)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		b, err := readCompatibilityFile(root, "data/unified-glossary-reviews/"+locale+"/"+entry.Name())
		if err != nil {
			return err
		}
		var r UnifiedGlossaryReview
		if err := contentidentity.StrictJSON(b, &r); err != nil {
			return err
		}
		if err := validateUnifiedReview(r, locale); err != nil {
			return err
		}
		if (role == "generation" && session == r.Reviewer) || (role == "reviewer" && session == r.GenerationSession) {
			return fmt.Errorf("locale language role/session collision")
		}
	}
	return nil
}

func unifiedReviewerArchive(locale, sha string) string {
	return "data/unified-glossary-reviews/" + locale + "-bundles/" + sha + ".zip"
}
func validateUnifiedReviewBundle(root string, r UnifiedGlossaryReview) error {
	b, err := readCompatibilityFile(root, unifiedReviewerArchive(r.Locale, r.BundleSHA))
	if err != nil {
		return err
	}
	if sum(b) != r.BundleSHA {
		return fmt.Errorf("Reviewer archive hash mismatch")
	}
	fs, err := ReadTransportBundle(b, 128, 32<<20)
	if err != nil {
		return err
	}
	var m UnifiedGlossaryBundle
	if err := contentidentity.StrictJSON(fs["manifest.json"], &m); err != nil {
		return err
	}
	if err := ValidateTransportBundleInventory(fs, m.Files, true); err != nil {
		return err
	}
	copy := m
	copy.Identity = ""
	if m.Identity != sum(mustJSON(copy)) || m.Schema != "go-learning/unified-glossary-bundle/v1" || m.Role != "reviewer" || m.Locale != r.Locale || m.CorpusSHA != r.CorpusSHA || m.GlossarySHA != r.GlossarySHA || m.Identity != r.InputSHA || len(m.ExpectedOutputs) != 0 {
		return fmt.Errorf("Reviewer archived input identity mismatch")
	}
	var c GlossarySourceCorpus
	if err := contentidentity.StrictJSON(fs["inputs/"+GlossaryCorpusPath], &c); err != nil {
		return err
	}
	if err := ValidateGlossaryCorpus(&c); err != nil {
		return err
	}
	if c.Identity != r.CorpusSHA || sum(fs["inputs/"+r.GlossaryPath]) != r.GlossarySHA {
		return fmt.Errorf("Reviewer archived corpus/glossary mismatch")
	}
	return nil
}
func RequireNewLanguageGlossaryReview(root, locale string) error {
	_, err := RequireUnifiedGlossaryReview(root, locale)
	return err
}
