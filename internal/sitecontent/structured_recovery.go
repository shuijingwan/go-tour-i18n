package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const StructuredAuthorityProfile = "structured-language/v1"

// Execution instructions remain transported verbatim but cannot override these
// normative language inputs. Source, glossary, Review and locale remain separate
// exact semantic dependencies in the existing transport inventory and Plan.
var StructuredSemanticAuthorities = []string{
	"docs/STRUCTURED_GENERATION_SEMANTICS.md",
	"docs/TRANSLATION_TERMINOLOGY.md",
	"docs/TERMINOLOGY_GUIDE.md",
}

// Explicitly classified execution context for this profile. Do not derive this
// from all future PackageAuthorities: a newly added, unclassified document is
// an exact semantic dependency until deliberately assigned an execution role.
var structuredExecutionAuthorities = []string{
	"AGENTS.md", "docs/WORKFLOW_HANDOFF.md", "docs/SITE_V2_ARCHITECTURE.md",
	"docs/SITE_V2_WORKFLOW.md", "docs/TRANSLATION_WORKFLOW.md", "docs/TRANSLATION_TASK_SPEC.md",
	"docs/CODEX_TRANSLATION.md", "docs/CHATGPT_LANGUAGE_GENERATION.md",
	"docs/GLOSSARY_REVIEW.md", "docs/GLOSSARY_COMPATIBILITY.md",
	structuredTransportContract,
}

// An optional successor edge on existing StructuredGeneration, not another
// Generation invocation, language acceptance, lifecycle or Surface receipt.
type StructuredCarryForward struct {
	CurrentBundle   Reference   `json:"current_bundle"`
	CurrentIdentity string      `json:"current_bundle_identity_sha256"`
	OriginalOutput  Reference   `json:"original_output"`
	AuthorityEdge   string      `json:"authority_edge"`
	ValidatedFiles  []Reference `json:"validated_files"`
}

type StructuredRecoveryOptions struct {
	BundleSHA, OutputSHA string
	Provenance           Provenance
	Apply                bool
}

type StructuredRecoveryResult struct {
	Status           string                 `json:"status"`
	Applied          bool                   `json:"applied"`
	Locale           string                 `json:"locale"`
	OriginalBundle   Reference              `json:"original_bundle"`
	OriginalIdentity string                 `json:"original_bundle_identity_sha256"`
	OriginalOutput   Reference              `json:"original_output"`
	PlanIdentity     string                 `json:"plan_identity_sha256"`
	Provenance       Provenance             `json:"generation"`
	CarryForward     StructuredCarryForward `json:"carry_forward"`
	Generation       Reference              `json:"generation_evidence"`
	ValidationPassed int                    `json:"validation_passed"`
	ValidationFailed int                    `json:"validation_failed"`
}

func readStructuredBundle(b []byte) (*StructuredBundle, map[string][]byte, error) {
	fs, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return nil, nil, err
	}
	var m StructuredBundle
	if err := StrictJSON(fs["manifest.json"], &m); err != nil {
		return nil, nil, err
	}
	if err := i18n.ValidateTransportBundleInventory(fs, m.Files, true); err != nil {
		return nil, nil, err
	}
	if err := validateStructuredTransportProjection(&m, fs); err != nil {
		return nil, nil, err
	}
	copy, plan := m, m.Plan
	copy.Identity, plan.Identity = "", ""
	if m.Schema != StructuredSchema || m.Plan.Schema != StructuredSchema || !component(m.Plan.Locale) || m.Identity != identity(copy) || m.Plan.Identity != identity(plan) || (m.AuthorityProfile != "" && m.AuthorityProfile != StructuredAuthorityProfile) {
		return nil, nil, fmt.Errorf("invalid structured Bundle/Plan self identity or authority profile")
	}
	if m.AuthorityProfile != "" {
		for _, p := range StructuredSemanticAuthorities {
			if len(fs["inputs/"+p]) == 0 {
				return nil, nil, fmt.Errorf("missing structured semantic authority: %s", p)
			}
		}
	}
	return &m, fs, nil
}

func structuredExecutionAuthority(p string) bool {
	for _, name := range structuredExecutionAuthorities {
		if name == p {
			return true
		}
	}
	return false
}

// A Plan hash is insufficient: it omits raw source, locale and authority bytes.
// Compare the entire Plan and every non-execution transport member, including
// raw source, semantic authority, glossary, corpus, Review, baseline and finding.
func compatibleStructuredBundles(old StructuredBundle, oldFiles map[string][]byte, current StructuredBundle, currentFiles map[string][]byte) error {
	projected, err := compatibleStructuredTransportProjection(old, current)
	if err != nil {
		return err
	}
	if current.AuthorityProfile != StructuredAuthorityProfile || !reflect.DeepEqual(old.Plan, current.Plan) || !reflect.DeepEqual(old.RevisionFinding, current.RevisionFinding) {
		return fmt.Errorf("structured semantic Plan/Expected/Document/finding drift; regeneration required")
	}
	legacy := old.AuthorityProfile == ""
	if legacy {
		if err := compatibleLegacyStructuredAuthority(oldFiles, currentFiles); err != nil {
			return err
		}
	} else if old.AuthorityProfile != current.AuthorityProfile {
		return fmt.Errorf("unknown structured authority transition")
	}
	// Execution bytes may evolve, but the declared transport/context inventory
	// and repository bindings cannot be deleted or forged by that exception.
	for _, m := range []StructuredBundle{old, current} {
		for _, p := range PackageAuthorities {
			found := false
			for _, f := range m.Files {
				if f.BundlePath == "inputs/"+p && f.RepositoryPath == p {
					found = true
				}
			}
			if !found {
				return fmt.Errorf("missing structured execution context inventory: %s", p)
			}
		}
	}
	index := func(m StructuredBundle, files map[string][]byte) (map[string]i18n.TransportBundleFile, error) {
		out := map[string]i18n.TransportBundleFile{}
		for _, f := range m.Files {
			if projected[f.BundlePath] {
				continue
			}
			if f.BundlePath != "inputs/"+f.RepositoryPath && f.BundlePath != "source/"+f.RepositoryPath && f.RepositoryPath != "" && !structuredSourcePayloadBinding(m, files, f) {
				return nil, fmt.Errorf("noncanonical structured repository binding: %s", f.BundlePath)
			}
			if f.BundlePath == "inputs/"+f.RepositoryPath && structuredExecutionAuthority(f.RepositoryPath) {
				continue
			}
			if legacy {
				added := false
				for _, p := range StructuredSemanticAuthorities {
					if f.BundlePath == "inputs/"+p {
						added = true
					}
				}
				if added {
					continue
				}
			}
			out[f.BundlePath] = f
		}
		return out, nil
	}
	a, err := index(old, oldFiles)
	if err != nil {
		return err
	}
	b, err := index(current, currentFiles)
	if err != nil {
		return err
	}
	if len(a) != len(b) {
		return fmt.Errorf("structured semantic transport inventory drift; regeneration required")
	}
	for p, f := range a {
		n, ok := b[p]
		if !ok || f.RepositoryPath != n.RepositoryPath || !bytes.Equal(oldFiles[p], currentFiles[p]) {
			return fmt.Errorf("structured semantic input drift: %s; regeneration required", p)
		}
	}
	return nil
}

func checkStructuredSemanticAuthority(root string, m StructuredBundle, files map[string][]byte) error {
	if m.AuthorityProfile == "" {
		return nil
	} // Immutable legacy consumers retain their existing gates.
	paths := append([]string{}, StructuredSemanticAuthorities...)
	for _, f := range m.Files {
		if strings.HasPrefix(f.RepositoryPath, "docs/") && !structuredExecutionAuthority(f.RepositoryPath) {
			paths = append(paths, f.RepositoryPath)
		}
	}
	for _, p := range uniqueStrings(paths) {
		b, err := readRegular(root, p)
		if err != nil || !bytes.Equal(b, files["inputs/"+p]) {
			return fmt.Errorf("structured language-semantic authority stale: %s", p)
		}
	}
	return nil
}

func structuredOutputBytes(b []byte, m StructuredBundle) (*StructuredOutput, error) {
	if err := textBytes(b); err != nil {
		return nil, err
	}
	if !bytes.HasSuffix(b, []byte("\n")) || bytes.HasSuffix(b, []byte("\n\n")) {
		return nil, fmt.Errorf("structured output requires exactly one EOF LF")
	}
	var o StructuredOutput
	if err := StrictJSON(b, &o); err != nil {
		return nil, err
	}
	if o.Schema != StructuredSchema || o.BundleSHA != m.Identity || len(o.Outputs) != len(m.Plan.Expected) {
		return nil, fmt.Errorf("original structured output does not bind original Bundle/exact set")
	}
	for i, t := range o.Outputs {
		if t.Path != m.Plan.Expected[i] {
			return nil, fmt.Errorf("original structured output exact-set mismatch")
		}
	}
	return &o, nil
}

func structuredBundleRef(locale string, b []byte) Reference {
	return Reference{"data/locale-language/" + locale + "/structured-bundles/" + digest(b) + ".zip", digest(b)}
}
func structuredOutputRef(locale string, b []byte) Reference {
	return Reference{"data/locale-language/" + locale + "/structured-outputs/" + digest(b) + ".json", digest(b)}
}
func structuredValidatedFiles(files map[string][]byte) []Reference {
	refs := []Reference{}
	for p, b := range files {
		refs = append(refs, Reference{p, digest(b)})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	return refs
}

// Default is a read-only preflight. All checks finish before any archive write;
// apply publishes original bytes and a new content-addressed successor record.
func RecoverStructuredAssets(root string, bundle, output []byte, options StructuredRecoveryOptions) (*StructuredRecoveryResult, error) {
	if !validSHA(options.BundleSHA) || !validSHA(options.OutputSHA) || digest(bundle) != options.BundleSHA || digest(output) != options.OutputSHA {
		return nil, fmt.Errorf("trusted original Bundle/output SHA mismatch")
	}
	old, oldFiles, err := readStructuredBundle(bundle)
	if err != nil {
		return nil, err
	}
	o, err := structuredOutputBytes(output, *old)
	if err != nil {
		return nil, err
	}
	if err := validateProvenance(options.Provenance); err != nil {
		return nil, err
	}
	if err := requireIndependentSession(root, old.Plan.Locale, options.Provenance.Session, "generation"); err != nil {
		return nil, err
	}
	var review i18n.UnifiedGlossaryReview
	if err := readReference(root, old.Plan.Review, &review); err != nil {
		return nil, err
	}
	if review.GenerationSession != options.Provenance.Session {
		return nil, fmt.Errorf("original Generation session/Unified Review mismatch")
	}
	currentBytes, current, err := exportStructuredAssets(root, old.Plan.Locale, old.RevisionFinding)
	if err != nil {
		return nil, err
	}
	// Keep the canonical current-check: recovery never relaxes normal import.
	if _, err := CheckStructuredAssetsBundle(root, currentBytes); err != nil {
		return nil, err
	}
	_, currentFiles, err := readStructuredBundle(currentBytes)
	if err != nil {
		return nil, err
	}
	if err := compatibleStructuredBundles(*old, oldFiles, *current, currentFiles); err != nil {
		return nil, err
	}
	files, err := validateStructuredOutput(root, current.Plan, *o)
	if err != nil {
		return nil, fmt.Errorf("structured recovery current validation failed: %w", err)
	}
	edge := StructuredAuthorityProfile
	if old.AuthorityProfile == "" {
		edge = "legacy-execution-authority-to-structured-language/v1"
	}
	carry := StructuredCarryForward{CurrentBundle: structuredBundleRef(old.Plan.Locale, currentBytes), CurrentIdentity: current.Identity, OriginalOutput: structuredOutputRef(old.Plan.Locale, output), AuthorityEdge: edge, ValidatedFiles: structuredValidatedFiles(files)}
	r := &StructuredGeneration{Schema: StructuredSchema, Plan: current.Plan, Output: *o, Provenance: options.Provenance, Bundle: structuredBundleRef(old.Plan.Locale, bundle), CarryForward: &carry}
	r.Identity = identity(*r)
	if _, _, _, err := validateFrozenStructuredGeneration(root, r, current, currentFiles); err != nil {
		return nil, fmt.Errorf("structured recovery frozen consumer integrity failed: %w", err)
	}
	name := "data/locale-language/" + old.Plan.Locale + "/structured-generation/" + r.Identity + ".json"
	b, err := Encode(r)
	if err != nil {
		return nil, err
	}
	result := &StructuredRecoveryResult{Status: "PASS", Locale: old.Plan.Locale, OriginalBundle: r.Bundle, OriginalIdentity: old.Identity, OriginalOutput: carry.OriginalOutput, PlanIdentity: current.Plan.Identity, Provenance: r.Provenance, CarryForward: carry, Generation: Reference{name, digest(b)}, ValidationPassed: len(files)}
	if !options.Apply {
		return result, nil
	}
	for _, archive := range []struct {
		ref  Reference
		data []byte
	}{{r.Bundle, bundle}, {carry.CurrentBundle, currentBytes}, {carry.OriginalOutput, output}, {result.Generation, b}} {
		if err := SaveArtifact(root, archive.ref.Path, archive.data); err != nil {
			return nil, fmt.Errorf("structured recovery archive %s: %w; mutation=known; inspect content-addressed archives before retry", archive.ref.Path, err)
		}
	}
	result.Applied = true
	return result, nil
}

// Historical readers validate the frozen edge without requiring today's Plan
// to remain pending. Current closure readers additionally run current source,
// language authority, glossary and canonical validators through existing gates.
func readStructuredCarryForward(root string, r *StructuredGeneration, old StructuredBundle, oldFiles map[string][]byte) (*StructuredBundle, map[string][]byte, error) {
	c := r.CarryForward
	b, err := readRegularReference(root, c.CurrentBundle)
	if err != nil {
		return nil, nil, err
	}
	current, files, err := readStructuredBundle(b)
	if err != nil {
		return nil, nil, err
	}
	if c.CurrentBundle != structuredBundleRef(r.Plan.Locale, b) || current.Identity != c.CurrentIdentity || !reflect.DeepEqual(r.Plan, current.Plan) {
		return nil, nil, fmt.Errorf("structured carry-forward current binding mismatch")
	}
	if err := compatibleStructuredBundles(old, oldFiles, *current, files); err != nil {
		return nil, nil, err
	}
	edge := StructuredAuthorityProfile
	if old.AuthorityProfile == "" {
		edge = "legacy-execution-authority-to-structured-language/v1"
	}
	if c.AuthorityEdge != edge {
		return nil, nil, fmt.Errorf("invalid structured authority edge")
	}
	original, err := readRegularReference(root, c.OriginalOutput)
	if err != nil {
		return nil, nil, err
	}
	o, err := structuredOutputBytes(original, old)
	if err != nil || c.OriginalOutput != structuredOutputRef(r.Plan.Locale, original) || !reflect.DeepEqual(o, &r.Output) {
		return nil, nil, fmt.Errorf("structured carry-forward original output binding mismatch: %v", err)
	}
	if len(c.ValidatedFiles) != len(r.Output.Outputs) {
		return nil, nil, fmt.Errorf("structured carry-forward validation exact-set mismatch")
	}
	for i, ref := range c.ValidatedFiles {
		if ref.Path != r.Plan.Expected[i] || !validSHA(ref.SHA256) {
			return nil, nil, fmt.Errorf("structured carry-forward validation inventory mismatch")
		}
	}
	return current, files, nil
}

// The legacy bridge is populated with audited immutable authority versions in
// structured_authority_migration.go. Never infer compatibility from a .md suffix.
func compatibleLegacyStructuredAuthority(oldFiles, currentFiles map[string][]byte) error {
	for _, p := range PackageAuthorities {
		if digest(oldFiles["inputs/"+p]) != legacyStructuredAuthoritySHA[p] {
			return fmt.Errorf("unknown legacy language authority: %s; regeneration or audited migration required", p)
		}
	}
	for _, p := range StructuredSemanticAuthorities {
		if digest(currentFiles["inputs/"+p]) != legacyStructuredSemanticSHA[p] {
			return fmt.Errorf("legacy structured semantic authority changed: %s; regeneration required", p)
		}
	}
	return nil
}
