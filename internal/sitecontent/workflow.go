package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const WorkflowSchema = "go-learning/package-workflow/v2"
const MaxReviewUnits = 60
const MaxReviewTextBytes = 16 << 20
const MaxReviewContextBytes = 16 << 20

var PackageAuthorities = []string{"AGENTS.md", "docs/WORKFLOW_HANDOFF.md", "docs/SITE_V2_ARCHITECTURE.md", "docs/SITE_V2_WORKFLOW.md", "docs/TRANSLATION_WORKFLOW.md", "docs/TRANSLATION_TASK_SPEC.md", "docs/CODEX_TRANSLATION.md", "docs/CHATGPT_LANGUAGE_GENERATION.md", "docs/GLOSSARY_REVIEW.md", "docs/GLOSSARY_COMPATIBILITY.md"}

type Batch struct {
	Schema             string                     `json:"schema"`
	Locale             string                     `json:"locale"`
	Package            string                     `json:"package"`
	Task               string                     `json:"task"`
	Batch              string                     `json:"batch"`
	Contract           string                     `json:"parser_contract"`
	PackageSHA         string                     `json:"package_sha256"`
	SourceSHA          string                     `json:"source_sha256"`
	FrozenCommit       string                     `json:"frozen_commit"`
	GlossarySHA        string                     `json:"glossary_sha256"`
	GlossaryReview     Reference                  `json:"full_glossary_review"`
	Documents          []Document                 `json:"documents"`
	Selected           []string                   `json:"selected_units"`
	Finding            *Reference                 `json:"revision_finding,omitempty"`
	ProvenanceContract string                     `json:"provenance_contract"`
	Files              []i18n.TransportBundleFile `json:"files"`
	Identity           string                     `json:"identity_sha256"`
}
type Target struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}
type Output struct {
	Schema   string   `json:"schema"`
	BatchSHA string   `json:"batch_sha256"`
	Targets  []Target `json:"targets"`
}
type Provenance struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	Session     string `json:"session"`
	GeneratedAt string `json:"generated_at"`
}
type Generation struct {
	Schema     string     `json:"schema"`
	Batch      Batch      `json:"batch"`
	Output     Output     `json:"output"`
	Provenance Provenance `json:"generation"`
	Bundle     Reference  `json:"bundle"`
	Identity   string     `json:"identity_sha256"`
}
type Rating struct {
	ID        string   `json:"id"`
	TargetSHA string   `json:"target_sha256"`
	Rating    string   `json:"rating"`
	Findings  []string `json:"findings"`
}
type Review struct {
	GlossarySHA         string      `json:"glossary_sha256"`
	CorpusSHA           string      `json:"corpus_identity_sha256"`
	GlossaryReview      Reference   `json:"unified_glossary_review"`
	Schema              string      `json:"schema"`
	Locale              string      `json:"locale"`
	Package             string      `json:"package"`
	ReviewID            string      `json:"review_id"`
	Invocation          string      `json:"invocation"`
	Session             string      `json:"session"`
	Model               string      `json:"model"`
	ReviewedAt          string      `json:"reviewed_at"`
	Generations         []Reference `json:"generations"`
	Selected            []string    `json:"selected_units"`
	SelectedGenerations []Reference `json:"selected_generations"`
	ContextSHA          string      `json:"context_sha256"`
	Ratings             []Rating    `json:"ratings"`
	Identity            string      `json:"identity_sha256"`
}
type Finalization struct {
	Schema        string      `json:"schema"`
	Locale        string      `json:"locale"`
	Package       string      `json:"package"`
	PackageSHA    string      `json:"package_sha256"`
	SourceSHA     string      `json:"source_sha256"`
	GlossarySHA   string      `json:"glossary_sha256"`
	Generations   []Reference `json:"generations"`
	Reviews       []Reference `json:"reviews"`
	Compatibility []Reference `json:"compatibility"`
	Documents     []Document  `json:"documents"`
	Targets       []Target    `json:"targets"`
	Identity      string      `json:"identity_sha256"`
}

func component(s string) bool        { return validPath(s) && path.Base(s) == s }
func packageWorkflow(id string) bool { return id == "learn-docs-v1" || id == "site-v2-shell" }
func workflowPath(locale, pkg, kind, id string) string {
	return "data/site-workflow/" + locale + "/" + pkg + "/" + kind + "/" + id + ".json"
}
func documentIdentity(d []Document) string { return identity(d) }
func unitsByID(docs []Document) map[string]Unit {
	m := map[string]Unit{}
	for _, d := range docs {
		for _, u := range d.Units {
			m[u.ID] = u
		}
	}
	return m
}
func currentPackage(root, locale, pkg string) (*Global, Package, []Document, map[string][]byte, []byte, error) {
	if !component(locale) || pkg != "learn-docs-v1" {
		return nil, Package{}, nil, nil, nil, fmt.Errorf("invalid locale/package")
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return nil, Package{}, nil, nil, nil, err
	}
	p, err := packageByID(g, pkg)
	if err != nil {
		return nil, p, nil, nil, nil, err
	}
	docs, raw, err := WorkflowDocuments(root, g, pkg)
	if err != nil {
		return nil, p, nil, nil, nil, err
	}
	glossary, err := readRegular(root, "locales/"+locale+"/glossary.yaml")
	if err != nil {
		return nil, p, nil, nil, nil, err
	}
	if err := requireUnifiedReview(root, locale); err != nil {
		return nil, p, nil, nil, nil, err
	}
	return g, p, docs, raw, glossary, nil
}
func exactIDs(ids []string, available map[string]Unit) error {
	if len(ids) == 0 {
		return fmt.Errorf("empty working set")
	}
	last := ""
	for _, id := range ids {
		if id <= last || available[id].ID == "" {
			return fmt.Errorf("unknown/duplicate/unsorted unit %s", id)
		}
		last = id
	}
	return nil
}
func selectAll(docs []Document) []string {
	ids := []string{}
	for id := range unitsByID(docs) {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func generationEligible(root, locale, pkg, task string, finding *Reference, glossary []byte, docs []Document) ([]string, error) {
	selected := selectAll(docs)
	if task != "initial" && task != "revision" && task != "glossary-revision" && task != "surface-revision" {
		return nil, fmt.Errorf("unknown Generation task")
	}
	if task == "surface-revision" {
		if finding == nil {
			return nil, fmt.Errorf("Surface revision requires independent exact finding")
		}
		var err error
		selected, err = surfaceRevisionPages(root, locale, pkg, *finding, docs)
		if err != nil {
			return nil, err
		}
	} else if task == "revision" {
		if finding == nil {
			return nil, fmt.Errorf("revision requires independent finding")
		}
		review, err := loadReview(root, *finding)
		if err != nil {
			return nil, err
		}
		if review.Locale != locale || review.Package != pkg {
			return nil, fmt.Errorf("revision finding scope mismatch")
		}
		allowed := map[string]bool{}
		for _, r := range review.Ratings {
			if r.Rating != "A" && len(r.Findings) > 0 {
				allowed[r.ID] = true
			}
		}
		selected = []string{}
		for id := range allowed {
			selected = append(selected, id)
		}
	} else if task == "glossary-revision" {
		if finding == nil {
			return nil, fmt.Errorf("glossary revision requires affected-scope evidence")
		}
		var e PackageCompatibility
		if err := readReference(root, *finding, &e); err != nil {
			return nil, err
		}
		if e.Locale != locale || e.Package != pkg || e.New.SHA256 != digest(glossary) {
			return nil, fmt.Errorf("glossary revision evidence not current")
		}
		want, err := buildPackageCompatibility(root, locale, pkg, e.Old.SHA256, e.New.SHA256, e.Review, e.Contexts, e.Prior)
		if err != nil || !reflect.DeepEqual(e, *want) {
			return nil, fmt.Errorf("invalid glossary revision evidence")
		}
		allowed := map[string]bool{}
		for _, id := range e.Affected {
			allowed[id] = true
		}
		selected = []string{}
		for id := range allowed {
			selected = append(selected, id)
		}
	} else if finding != nil {
		return nil, fmt.Errorf("initial batch cannot carry finding")
	}
	sort.Strings(selected)
	if err := exactIDs(selected, unitsByID(docs)); err != nil {
		return nil, err
	}
	return selected, nil
}

func ExportPackageGeneration(root, locale, pkg, task, batch string, selected []string, finding *Reference) ([]byte, *Batch, error) {
	g, p, docs, raw, glossary, err := currentPackage(root, locale, pkg)
	if err != nil {
		return nil, nil, err
	}
	if !component(batch) || (task != "initial" && task != "revision" && task != "glossary-revision" && task != "surface-revision") {
		return nil, nil, fmt.Errorf("invalid batch/task")
	}
	eligible, err := generationEligible(root, locale, pkg, task, finding, glossary, docs)
	if err != nil {
		return nil, nil, err
	}
	if selected == nil {
		plan, err := PlanPageWorkingSets(docs, raw, eligible, nil)
		if err != nil {
			return nil, nil, err
		}
		selected = plan.Sets[0].Selected
	}
	allowed := map[string]bool{}
	for _, id := range eligible {
		allowed[id] = true
	}
	for _, id := range selected {
		if !allowed[id] {
			return nil, nil, fmt.Errorf("Generation unit outside authorized task scope")
		}
	}
	sort.Strings(selected)
	if err := requireWorkingSet(docs, raw, selected, nil); err != nil {
		return nil, nil, err
	}
	docs = selectedDocuments(docs, selected)
	review, err := i18n.CurrentParsedGlossaryReview(root, locale, digest(glossary))
	if err != nil {
		return nil, nil, err
	}
	m := &Batch{Schema: WorkflowSchema, Locale: locale, Package: pkg, Task: task, Batch: batch, Contract: UnitContract, PackageSHA: p.Identity, SourceSHA: documentIdentity(docs), FrozenCommit: g.UpstreamCommit, GlossarySHA: digest(glossary), GlossaryReview: Reference{Path: review.Path, SHA256: review.SHA256}, Documents: docs, Selected: selected, Finding: finding, Files: []i18n.TransportBundleFile{}}
	m.ProvenanceContract = "chatgpt|codex:gpt-5.6-sol-high:independent-session/v1"
	entries := []i18n.TransportBundleEntry{}
	add := func(name, repo string, b []byte) {
		entries = append(entries, i18n.TransportBundleEntry{Path: name, Data: b})
		m.Files = append(m.Files, i18n.NewTransportBundleFile(name, repo, b))
	}
	add("formal/glossary.yaml", "locales/"+locale+"/glossary.yaml", glossary)
	for _, doc := range docs {
		add("source/"+doc.Path, doc.Path, raw[doc.Path])
	}
	if finding != nil {
		prior, err := revisionContextTargets(root, task, *finding, selected)
		if err != nil {
			return nil, nil, err
		}
		b, err := Encode(prior)
		if err != nil {
			return nil, nil, err
		}
		add("context/prior-targets.json", "", b)
	}
	authority := append(append([]string{}, PackageAuthorities...), review.Path, "locales/"+locale+"/locale.json", GlobalPath)
	if finding != nil {
		authority = append(authority, finding.Path)
	}
	for _, name := range authority {
		b, err := readRegular(root, name)
		if err != nil {
			return nil, nil, err
		}
		add("authority/"+name, name, b)
	}
	sort.Slice(m.Files, func(i, j int) bool { return m.Files[i].BundlePath < m.Files[j].BundlePath })
	m.Identity = identity(*m)
	manifest, err := Encode(m)
	if err != nil {
		return nil, nil, err
	}
	b, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
	return b, m, err
}

// Revision transport contains only the authorized prior candidates. Unaffected
// A results are context/evidence, never additional replacement outputs.
func revisionContextTargets(root, task string, finding Reference, selected []string) ([]Target, error) {
	values := map[string]string{}
	switch task {
	case "revision":
		r, err := loadReview(root, finding)
		if err != nil {
			return nil, err
		}
		for i, id := range r.Selected {
			g, err := loadGeneration(root, r.SelectedGenerations[i])
			if err != nil {
				return nil, err
			}
			for _, t := range g.Output.Targets {
				if t.ID == id {
					values[id] = t.Text
				}
			}
		}
	case "glossary-revision":
		var e PackageCompatibility
		if err := readReference(root, finding, &e); err != nil {
			return nil, err
		}
		for _, c := range e.Contexts {
			values[c.ID] = c.Target
		}
	case "surface-revision":
		var r IntegratedSurfaceReceipt
		if err := readReference(root, finding, &r); err != nil {
			return nil, err
		}
		for _, ref := range r.Scope.Closures {
			c, _, err := CheckPackageClosure(root, ref)
			if err != nil {
				return nil, err
			}
			if c.PageFinalization != nil {
				f, err := CheckFinalization(root, *c.PageFinalization)
				if err != nil {
					return nil, err
				}
				for _, t := range f.Targets {
					values[t.ID] = t.Text
				}
			}
		}
	default:
		return nil, fmt.Errorf("unknown revision context")
	}
	out := []Target{}
	for _, id := range selected {
		value, ok := values[id]
		if !ok {
			return nil, fmt.Errorf("authorized prior Page candidate missing: %s", id)
		}
		out = append(out, Target{ID: id, Text: value})
	}
	return out, nil
}
func CheckPackageGenerationBundle(root string, b []byte) (*Batch, error) {
	files, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return nil, err
	}
	var m Batch
	if err := StrictJSON(files["manifest.json"], &m); err != nil {
		return nil, err
	}
	if err := i18n.ValidateTransportBundleInventory(files, m.Files, true); err != nil {
		return nil, err
	}
	want, _, err := ExportPackageGeneration(root, m.Locale, m.Package, m.Task, m.Batch, m.Selected, m.Finding)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(b, want) {
		return nil, fmt.Errorf("Generation bundle non-current or contract mismatch")
	}
	return &m, nil
}
func validateProvenance(p Provenance) error {
	t, err := time.Parse(time.RFC3339, p.GeneratedAt)
	if err != nil || t.Location() != time.UTC || !component(p.Session) || (p.Provider != "chatgpt" && p.Provider != "codex") || p.Model != "gpt-5.6-sol-high" {
		return fmt.Errorf("invalid formal Generation provenance")
	}
	return nil
}

func ImportPackageGeneration(root string, bundle, output []byte, p Provenance) (*Generation, string, error) {
	m, err := CheckPackageGenerationBundle(root, bundle)
	if err != nil {
		return nil, "", err
	}
	if err := validateProvenance(p); err != nil {
		return nil, "", err
	}
	if err := requireIndependentSession(root, m.Locale, p.Session, "generation"); err != nil {
		return nil, "", err
	}
	var out Output
	if err := StrictJSON(output, &out); err != nil {
		return nil, "", err
	}
	if out.Schema != WorkflowSchema || out.BatchSHA != m.Identity || len(out.Targets) != len(m.Selected) {
		return nil, "", fmt.Errorf("Generation output exact-set mismatch")
	}
	outputSize := 0
	for i, t := range out.Targets {
		outputSize += len(t.Text)
		if t.ID != m.Selected[i] {
			return nil, "", fmt.Errorf("Generation output order/exact-set mismatch")
		}
	}
	if outputSize > MaxGenerationOutputBytes {
		return nil, "", fmt.Errorf("Generation output exceeds bounded UTF-8 text limit")
	}
	if err := validateTargets(root, m.Locale, m.Documents, out.Targets); err != nil {
		return nil, "", err
	}
	ref := Reference{Path: workflowPath(m.Locale, m.Package, "bundles", digest(bundle)), SHA256: digest(bundle)}
	result := &Generation{Schema: WorkflowSchema, Batch: *m, Output: out, Provenance: p, Bundle: ref}
	result.Identity = identity(*result)
	name := workflowPath(m.Locale, m.Package, "generation", m.Batch)
	encoded, err := Encode(result)
	if err != nil {
		return nil, "", err
	}
	if existing, err := readRegular(root, name); err == nil && !bytes.Equal(existing, encoded) {
		return nil, "", fmt.Errorf("refuse generation overwrite")
	}
	if err := SaveArtifact(root, ref.Path, bundle); err != nil {
		return nil, "", err
	}
	if err := saveImmutable(root, name, result); err != nil {
		return nil, "", err
	}
	return result, name, nil
}
func validateTargets(root, locale string, docs []Document, targets []Target) error {
	glossary, err := i18n.LoadGlossary(root, locale)
	if err != nil {
		return err
	}
	all := unitsByID(docs)
	seen := map[string]bool{}
	for _, t := range targets {
		u := all[t.ID]
		if u.ID == "" || seen[t.ID] {
			return fmt.Errorf("unknown/duplicate target")
		}
		seen[t.ID] = true
		if err := i18n.ValidateParsedGlossary(u.Source, t.Text, glossary); err != nil {
			return err
		}
	}
	g, err := LoadCurrent(root)
	if err != nil {
		return err
	}
	_, raw, err := WorkflowDocuments(root, g, func() string {
		if strings.HasPrefix(docs[0].Path, "data/site-shell/") {
			return "site-v2-shell"
		}
		return "learn-docs-v1"
	}())
	if err != nil {
		return err
	}
	for _, d := range docs {
		has := false
		values := map[string]string{}
		for _, u := range d.Units {
			values[u.ID] = u.Source
		}
		for _, t := range targets {
			if _, ok := values[t.ID]; ok {
				values[t.ID] = t.Text
				has = true
			}
		}
		if !has {
			continue
		}
		if _, err := ReconstructWorkflow(d, raw[d.Path], values); err != nil {
			return err
		}
	}
	return nil
}

func readReference(root string, ref Reference, v any) error {
	if !validSHA(ref.SHA256) {
		return fmt.Errorf("invalid evidence hash")
	}
	b, err := readRegular(root, ref.Path)
	if err != nil {
		return err
	}
	if digest(b) != ref.SHA256 {
		return fmt.Errorf("evidence hash mismatch")
	}
	return StrictJSON(b, v)
}
func ReferenceFile(root, name string) (Reference, error) {
	b, err := readRegular(root, name)
	return Reference{Path: name, SHA256: digest(b)}, err
}

// saveImmutable publishes a fully written file with no-replace link semantics;
// partial files cannot become evidence. Existing identical bytes are idempotent.
func saveImmutable(root, name string, v any) error {
	b, err := Encode(v)
	if err != nil {
		return err
	}
	return SaveArtifact(root, name, b)
}
func ReadArtifact(root, name string) ([]byte, error) { return readRegular(root, name) }
func SaveArtifact(root, name string, b []byte) error {
	if !validPath(name) {
		return fmt.Errorf("unsafe evidence path")
	}
	parent := path.Dir(name)
	current := root
	for _, part := range strings.Split(parent, "/") {
		current = pathJoin(current, part)
		st, err := os.Lstat(current)
		if os.IsNotExist(err) {
			if err := os.Mkdir(current, 0755); err != nil {
				return err
			}
			st, err = os.Lstat(current)
		}
		if err != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe evidence parent")
		}
	}
	if existing, err := readRegular(root, name); err == nil {
		if bytes.Equal(existing, b) {
			return nil
		}
		return fmt.Errorf("refuse evidence overwrite: %s", name)
	} else if !os.IsNotExist(err) {
		return err
	}
	f, err := os.CreateTemp(current, ".package-evidence-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Link(f.Name(), pathJoin(root, name))
}
func pathJoin(root, p string) string { return root + "/" + p }

func loadGeneration(root string, ref Reference) (*Generation, error) {
	var g Generation
	if err := readReference(root, ref, &g); err != nil {
		return nil, err
	}
	copy := g
	copy.Identity = ""
	m := g.Batch
	mi := m.Identity
	m.Identity = ""
	if g.Schema != WorkflowSchema || g.Identity != identity(copy) || mi != identity(m) || g.Batch.Schema != WorkflowSchema || g.Batch.Contract != UnitContract || g.Output.Schema != WorkflowSchema || g.Output.BatchSHA != mi || len(g.Batch.FrozenCommit) != 40 || strings.Trim(g.Batch.FrozenCommit, "0123456789abcdef") != "" || !component(g.Batch.Locale) || !packageWorkflow(g.Batch.Package) || !component(g.Batch.Batch) || ref.Path != workflowPath(g.Batch.Locale, g.Batch.Package, "generation", g.Batch.Batch) {
		return nil, fmt.Errorf("invalid Generation artifact identity")
	}
	if err := validateProvenance(g.Provenance); err != nil {
		return nil, err
	}
	if err := validateDocuments(g.Batch.Documents); err != nil {
		return nil, err
	}
	if g.Batch.ProvenanceContract != "chatgpt|codex:gpt-5.6-sol-high:independent-session/v1" || (g.Batch.Task != "initial" && g.Batch.Task != "revision" && g.Batch.Task != "glossary-revision" && g.Batch.Task != "surface-revision") || len(g.Batch.Selected) == 0 {
		return nil, fmt.Errorf("unknown Generation contract/task")
	}
	if g.Batch.SourceSHA != documentIdentity(g.Batch.Documents) || !validSHA(g.Batch.GlossarySHA) {
		return nil, fmt.Errorf("Generation source/glossary identity mismatch")
	}
	b, err := readRegular(root, g.Bundle.Path)
	if err != nil {
		return nil, err
	}
	if digest(b) != g.Bundle.SHA256 || g.Bundle.Path != workflowPath(g.Batch.Locale, g.Batch.Package, "bundles", digest(b)) {
		return nil, fmt.Errorf("archived bundle identity mismatch")
	}
	files, err := i18n.ReadTransportBundle(b, 256, 32<<20)
	if err != nil {
		return nil, err
	}
	if err := i18n.ValidateTransportBundleInventory(files, g.Batch.Files, true); err != nil {
		return nil, err
	}
	var manifest Batch
	if err := StrictJSON(files["manifest.json"], &manifest); err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(manifest, g.Batch) {
		return nil, fmt.Errorf("Generation/archived manifest mismatch")
	}
	archivedRaw := map[string][]byte{}
	for _, d := range g.Batch.Documents {
		source := files["source/"+d.Path]
		archivedRaw[d.Path] = source
		parsed, err := ParseWorkflowDocument(documentSource(d), source, d.StableIndex)
		if err != nil || !reflect.DeepEqual(d, *parsed) {
			return nil, fmt.Errorf("Generation parser/source context mismatch")
		}
	}
	if err := requireWorkingSet(g.Batch.Documents, archivedRaw, g.Batch.Selected, nil); err != nil {
		return nil, err
	}
	if len(g.Output.Targets) != len(g.Batch.Selected) {
		return nil, fmt.Errorf("Generation exact-set mismatch")
	}
	outputSize := 0
	for i, t := range g.Output.Targets {
		outputSize += len(t.Text)
		if t.ID != g.Batch.Selected[i] {
			return nil, fmt.Errorf("Generation exact-set mismatch")
		}
	}
	if outputSize > MaxGenerationOutputBytes {
		return nil, fmt.Errorf("historical Generation output exceeds bounded text limit")
	}
	return &g, nil
}

type selection struct {
	Unit       Unit
	Document   Document
	Target     Target
	Generation *Generation
	Ref        Reference
}

func selections(root, locale, pkg string, refs []Reference, docs []Document) (map[string]selection, error) {
	available := map[string]Document{}
	for _, d := range docs {
		for _, u := range d.Units {
			available[u.ID] = d
		}
	}
	values := map[string]selection{}
	choices := map[string][]selection{}
	if len(refs) == 0 {
		return nil, fmt.Errorf("no generation evidence")
	}
	last := ""
	for _, ref := range refs {
		if ref.Path <= last {
			return nil, fmt.Errorf("generation refs must be unique sorted")
		}
		last = ref.Path
		g, err := loadGeneration(root, ref)
		if err != nil {
			return nil, err
		}
		if g.Batch.Locale != locale || g.Batch.Package != pkg {
			return nil, fmt.Errorf("generation scope mismatch")
		}
		old := unitsByID(g.Batch.Documents)
		for _, t := range g.Output.Targets {
			d, ok := available[t.ID]
			if !ok {
				continue
			}
			u := unitsByID([]Document{d})[t.ID]
			od := Document{}
			for _, v := range g.Batch.Documents {
				if v.Path == d.Path {
					od = v
				}
			}
			if old[t.ID].SourceSHA != u.SourceSHA || od.StructureSHA != d.StructureSHA || od.Route != d.Route {
				continue
			}
			choices[t.ID] = append(choices[t.ID], selection{u, d, t, g, ref})
		}
	}
	for id, candidates := range choices {
		if len(candidates) == 1 {
			values[id] = candidates[0]
			continue
		}
		parents := map[int][]int{}
		superseded := map[int]bool{}
		for child, c := range candidates {
			for parent, p := range candidates {
				if child == parent {
					continue
				}
				ok, err := revisionSupersedes(root, c, p)
				if err != nil {
					return nil, err
				}
				if ok {
					parents[child] = append(parents[child], parent)
					superseded[parent] = true
				}
			}
		}
		head := -1
		for i := range candidates {
			if !superseded[i] {
				if head >= 0 {
					return nil, fmt.Errorf("ambiguous generation branches for %s", id)
				}
				head = i
			}
		}
		if head < 0 {
			return nil, fmt.Errorf("revision lineage cycle")
		}
		visited := map[int]bool{}
		var walk func(int)
		walk = func(i int) {
			if visited[i] {
				return
			}
			visited[i] = true
			for _, p := range parents[i] {
				walk(p)
			}
		}
		walk(head)
		if len(visited) != len(candidates) {
			return nil, fmt.Errorf("disconnected revision lineage")
		}
		values[id] = candidates[head]
	}
	return values, nil
}

func revisionSupersedes(root string, child, parent selection) (bool, error) {
	batch := child.Generation.Batch
	if batch.Finding == nil {
		return false, nil
	}
	if batch.Task == "surface-revision" {
		var r IntegratedSurfaceReceipt
		if batch.Finding == nil {
			return false, fmt.Errorf("missing Surface authorization")
		}
		if err := readReference(root, *batch.Finding, &r); err != nil {
			return false, err
		}
		if r.Decision != "failed" || r.Scope.Locale != parent.Generation.Batch.Locale {
			return false, fmt.Errorf("Surface authorization mismatch")
		}
		for _, f := range r.ExactFindings {
			if f.ID == parent.Document.Path {
				for _, c := range r.Scope.Closures {
					closure, _, err := CheckPackageClosure(root, c)
					if err != nil {
						return false, err
					}
					if closure.PageFinalization == nil {
						continue
					}
					prior, err := CheckFinalization(root, *closure.PageFinalization)
					if err != nil {
						return false, err
					}
					for _, t := range prior.Targets {
						if t.ID == parent.Unit.ID && t.Text == parent.Target.Text {
							return true, nil
						}
					}
				}
			}
		}
		return false, nil
	}
	if batch.Task == "revision" {
		r, err := loadReview(root, *batch.Finding)
		if err != nil {
			return false, err
		}
		bound := false
		for _, ref := range r.Generations {
			if ref == parent.Ref {
				bound = true
			}
		}
		if !bound {
			return false, nil
		}
		for _, rating := range r.Ratings {
			if rating.ID == parent.Unit.ID && rating.Rating != "A" && rating.TargetSHA == digest([]byte(parent.Target.Text)) {
				return true, nil
			}
		}
		return false, nil
	}
	if batch.Task == "glossary-revision" {
		var e PackageCompatibility
		if err := readReference(root, *batch.Finding, &e); err != nil {
			return false, err
		}
		want, err := buildPackageCompatibility(root, e.Locale, e.Package, e.Old.SHA256, e.New.SHA256, e.Review, e.Contexts, e.Prior)
		if err != nil || !reflect.DeepEqual(e, *want) {
			return false, fmt.Errorf("invalid glossary-revision lineage")
		}
		for _, c := range e.Contexts {
			if c.ID == parent.Unit.ID && c.Target == parent.Target.Text && c.Source == parent.Unit.Source && c.StructureSHA == parent.Document.StructureSHA {
				for _, id := range e.Affected {
					if id == c.ID {
						return true, nil
					}
				}
			}
		}
	}
	return false, nil
}

func ReviewScope(root, locale, pkg string, refs []Reference, selected []string) (*Review, error) {
	_, _, docs, raw, glossary, err := currentPackage(root, locale, pkg)
	if err != nil {
		return nil, err
	}
	if len(selected) > MaxReviewUnits {
		return nil, fmt.Errorf("review invocation exceeds 60 units")
	}
	if err := exactIDs(selected, unitsByID(docs)); err != nil {
		return nil, err
	}
	values, err := selections(root, locale, pkg, refs, docs)
	if err != nil {
		return nil, err
	}
	textSize := 0
	contexts := map[string][]byte{}
	targets := []Target{}
	for _, id := range selected {
		s, ok := values[id]
		if !ok {
			return nil, fmt.Errorf("missing/current-source target %s", id)
		}
		if s.Generation.Batch.GlossarySHA != digest(glossary) {
			return nil, fmt.Errorf("new review scope requires current glossary; use verified carry-forward only for historical A")
		}
		textSize += len(s.Unit.Source) + len(s.Target.Text)
		contexts[s.Document.Path] = raw[s.Document.Path]
		targets = append(targets, s.Target)
	}
	contextSize := 0
	for _, b := range contexts {
		contextSize += len(b)
	}
	if textSize > MaxReviewTextBytes || contextSize > MaxReviewContextBytes {
		return nil, fmt.Errorf("review working set exceeds deterministic text/context byte limit")
	}
	actualTargets := map[string]string{}
	for _, t := range targets {
		actualTargets[t.ID] = t.Text
	}
	if err := requireWorkingSet(docs, raw, selected, actualTargets); err != nil {
		return nil, err
	}
	docs = selectedDocuments(docs, selected)
	r := &Review{Schema: WorkflowSchema, Locale: locale, Package: pkg, Generations: refs, Selected: selected, Ratings: []Rating{}}
	fullReview, err := i18n.RequireUnifiedGlossaryReview(root, locale)
	if err != nil {
		return nil, err
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, err
	}
	r.GlossarySHA, r.CorpusSHA = digest(glossary), corpus.Identity
	r.GlossaryReview = Reference{Path: fullReview.Path, SHA256: fullReview.SHA256}
	used := map[Reference]bool{}
	for _, id := range selected {
		r.SelectedGenerations = append(r.SelectedGenerations, values[id].Ref)
		used[values[id].Ref] = true
	}
	r.Generations = []Reference{}
	for ref := range used {
		r.Generations = append(r.Generations, ref)
	}
	sort.Slice(r.Generations, func(i, j int) bool { return r.Generations[i].Path < r.Generations[j].Path })
	r.ContextSHA = identity(struct {
		Docs     []Document
		Targets  []Target
		Glossary string
	}{docs, targets, digest(glossary)})
	return r, nil
}

func RecordPackageReview(root string, input []byte) (*Review, string, error) {
	var r Review
	if err := StrictJSON(input, &r); err != nil {
		return nil, "", err
	}
	if !component(r.ReviewID) || !component(r.Invocation) || !component(r.Session) || r.Model != "gpt-5.6-sol-high" || r.Identity != "" {
		return nil, "", fmt.Errorf("invalid independent Reviewer identity")
	}
	if err := requireIndependentSession(root, r.Locale, r.Session, "review"); err != nil {
		return nil, "", err
	}
	if _, err := time.Parse(time.RFC3339, r.ReviewedAt); err != nil {
		return nil, "", err
	}
	want, err := ReviewScope(root, r.Locale, r.Package, r.Generations, r.Selected)
	if err != nil {
		return nil, "", err
	}
	if r.Schema != WorkflowSchema || r.ContextSHA != want.ContextSHA || r.GlossarySHA != want.GlossarySHA || r.CorpusSHA != want.CorpusSHA || r.GlossaryReview != want.GlossaryReview || !reflect.DeepEqual(r.Generations, want.Generations) || !reflect.DeepEqual(r.SelectedGenerations, want.SelectedGenerations) || len(r.Ratings) != len(r.Selected) {
		return nil, "", fmt.Errorf("review exact-set/context mismatch")
	}
	for _, ref := range r.Generations {
		g, err := loadGeneration(root, ref)
		if err != nil {
			return nil, "", err
		}
		if g.Provenance.Session == r.Session {
			return nil, "", fmt.Errorf("Reviewer participated in Generation")
		}
	}
	values, err := selections(root, r.Locale, r.Package, r.Generations, func() []Document {
		g, _ := LoadCurrent(root)
		d, _, _ := WorkflowDocuments(root, g, r.Package)
		return d
	}())
	if err != nil {
		return nil, "", err
	}
	for i, rating := range r.Ratings {
		if rating.ID != r.Selected[i] || rating.TargetSHA != digest([]byte(values[rating.ID].Target.Text)) || !strings.Contains("ABCD", rating.Rating) || len(rating.Rating) != 1 || rating.Findings == nil || (rating.Rating == "A" && len(rating.Findings) > 0) || (rating.Rating != "A" && len(rating.Findings) == 0) {
			return nil, "", fmt.Errorf("invalid rating/finding")
		}
	}
	// One invocation cannot be recorded under multiple IDs to evade size limits.
	dir := "data/site-workflow/" + r.Locale + "/" + r.Package + "/review"
	entries, err := os.ReadDir(pathJoin(root, dir))
	if err != nil && !os.IsNotExist(err) {
		return nil, "", err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		b, err := readRegular(root, dir+"/"+entry.Name())
		if err != nil {
			return nil, "", err
		}
		var old Review
		if err := StrictJSON(b, &old); err != nil {
			return nil, "", err
		}
		if old.Invocation == r.Invocation && old.ReviewID != r.ReviewID {
			return nil, "", fmt.Errorf("invocation already recorded")
		}
	}
	r.Identity = identity(r)
	name := workflowPath(r.Locale, r.Package, "review", r.ReviewID)
	if err := saveImmutable(root, name, r); err != nil {
		return nil, "", err
	}
	return &r, name, nil
}

// Session separation is locale-wide across both new packages, not just the
// working set. Historical Tour roles still require the maintainer's truthful
// session assignment, because legacy artifacts have no session field to invent.
func requireIndependentSession(root, locale, session, role string) error {
	if role == "review" {
		role = "reviewer"
	}
	if err := i18n.CheckUnifiedLanguageRole(root, locale, session, role); err != nil {
		return err
	}
	if role == "generation" {
		ref, err := i18n.RequireUnifiedGlossaryReview(root, locale)
		if err != nil {
			return err
		}
		var r i18n.UnifiedGlossaryReview
		if err := readReference(root, Reference{ref.Path, ref.SHA256}, &r); err != nil {
			return err
		}
		if r.GenerationSession != session {
			return fmt.Errorf("Generation session must match the declared unified locale role")
		}
	}
	kind := "generation"
	if role == "generation" {
		kind = "review"
	}
	for _, pkg := range []string{"site-v2-shell", "learn-docs-v1"} {
		dir := "data/site-workflow/" + locale + "/" + pkg + "/" + kind
		entries, err := os.ReadDir(pathJoin(root, dir))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".package-evidence-") {
				continue
			}
			b, err := readRegular(root, dir+"/"+entry.Name())
			if err != nil {
				return err
			}
			other := ""
			if kind == "generation" {
				var g Generation
				if err := StrictJSON(b, &g); err != nil {
					return err
				}
				other = g.Provenance.Session
			} else {
				var r Review
				if err := StrictJSON(b, &r); err != nil {
					return err
				}
				other = r.Session
			}
			if other == session {
				return fmt.Errorf("session participated in the other locale language role")
			}
		}
	}
	return nil
}

func loadReview(root string, ref Reference) (*Review, error) {
	var r Review
	if err := readReference(root, ref, &r); err != nil {
		return nil, err
	}
	copy := r
	copy.Identity = ""
	if r.Schema != WorkflowSchema || r.Identity != identity(copy) || !component(r.ReviewID) || !component(r.Invocation) || !component(r.Session) || r.Model != "gpt-5.6-sol-high" || len(r.Selected) > MaxReviewUnits || len(r.Selected) != len(r.Ratings) || len(r.SelectedGenerations) != len(r.Selected) || ref.Path != workflowPath(r.Locale, r.Package, "review", r.ReviewID) {
		return nil, fmt.Errorf("invalid Review artifact")
	}
	if _, err := time.Parse(time.RFC3339, r.ReviewedAt); err != nil {
		return nil, err
	}
	if len(r.Generations) == 0 || len(r.Selected) == 0 {
		return nil, fmt.Errorf("empty review")
	}
	if !validSHA(r.CorpusSHA) || !validSHA(r.GlossarySHA) {
		return nil, fmt.Errorf("review glossary/corpus identity missing")
	}
	if err := i18n.ValidateParsedGlossaryReview(root, r.Locale, r.GlossarySHA, i18n.GlossaryArchiveReference{Path: r.GlossaryReview.Path, SHA256: r.GlossaryReview.SHA256}); err != nil {
		return nil, err
	}
	var fullReview i18n.UnifiedGlossaryReview
	if err := readReference(root, r.GlossaryReview, &fullReview); err != nil || fullReview.CorpusSHA != r.CorpusSHA {
		return nil, fmt.Errorf("review full corpus binding mismatch")
	}
	textSize := 0
	contextSize := 0
	seenDocs := map[string]bool{}
	pageBatch := 0
	last := ""
	loaded := map[Reference]*Generation{}
	filesByRef := map[Reference]map[string][]byte{}
	for _, ref := range r.Generations {
		g, err := loadGeneration(root, ref)
		if err != nil {
			return nil, err
		}
		if g.Batch.Locale != r.Locale || g.Batch.Package != r.Package || g.Batch.GlossarySHA != r.GlossarySHA || g.Provenance.Session == r.Session {
			return nil, fmt.Errorf("review generation/session mismatch")
		}
		loaded[ref] = g
		files, err := archivedGenerationFiles(root, g)
		if err != nil {
			return nil, err
		}
		filesByRef[ref] = files
	}
	for i, id := range r.Selected {
		if id <= last || r.Ratings[i].ID != id {
			return nil, fmt.Errorf("review exact-set mismatch")
		}
		last = id
		rating := r.Ratings[i]
		if rating.Findings == nil || len(rating.Rating) != 1 || !strings.Contains("ABCD", rating.Rating) || (rating.Rating == "A" && len(rating.Findings) > 0) || (rating.Rating != "A" && len(rating.Findings) == 0) {
			return nil, fmt.Errorf("invalid review rating")
		}
		found := false
		ref := r.SelectedGenerations[i]
		g := loaded[ref]
		if g == nil {
			return nil, fmt.Errorf("selected generation not bound to review")
		}
		{
			for _, t := range g.Output.Targets {
				if t.ID != id {
					continue
				}
				if found || digest([]byte(t.Text)) != rating.TargetSHA {
					return nil, fmt.Errorf("ambiguous or changed reviewed target")
				}
				found = true
				u := unitsByID(g.Batch.Documents)[id]
				textSize += len(u.Source) + len(t.Text)
				for _, d := range g.Batch.Documents {
					for _, du := range d.Units {
						if du.ID == id && !seenDocs[d.Path] {
							if d.UnitScope != PageContract || d.StableIndex < 1 || d.StableIndex > 60 || len(r.Selected) > 30 {
								return nil, fmt.Errorf("historical review is outside formal Page quality boundary")
							}
							membership := (d.StableIndex-1)/30 + 1
							if pageBatch != 0 && pageBatch != membership {
								return nil, fmt.Errorf("historical review mixes fixed Page batches")
							}
							pageBatch = membership
							seenDocs[d.Path] = true
							files := filesByRef[ref]
							contextSize += len(files["source/"+d.Path])
						}
					}
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("review target not bound to generation")
		}
	}
	if textSize > MaxReviewTextBytes || contextSize > MaxReviewContextBytes {
		return nil, fmt.Errorf("historical review exceeds quality boundary")
	}
	return &r, nil
}

func archivedGenerationFiles(root string, g *Generation) (map[string][]byte, error) {
	b, err := readRegular(root, g.Bundle.Path)
	if err != nil {
		return nil, err
	}
	return i18n.ReadTransportBundle(b, 256, 32<<20)
}

func FinalizePackage(root, locale, pkg string, generations, reviews, compatibility []Reference) (*Finalization, string, error) {
	return finalizePackage(root, locale, pkg, generations, reviews, compatibility, true)
}
func finalizePackage(root, locale, pkg string, generations, reviews, compatibility []Reference, save bool) (*Finalization, string, error) {
	_, p, docs, _, glossary, err := currentPackage(root, locale, pkg)
	if err != nil {
		return nil, "", err
	}
	values, err := selections(root, locale, pkg, generations, docs)
	if err != nil {
		return nil, "", err
	}
	if len(values) != len(unitsByID(docs)) {
		return nil, "", fmt.Errorf("package incomplete: pending source units")
	}
	accepted := map[string]bool{}
	last := ""
	for _, ref := range reviews {
		if ref.Path <= last {
			return nil, "", fmt.Errorf("review references must be unique sorted")
		}
		last = ref.Path
		r, err := loadReview(root, ref)
		if err != nil {
			return nil, "", err
		}
		if r.Locale != locale || r.Package != pkg {
			return nil, "", fmt.Errorf("review scope mismatch")
		}
		for i, rating := range r.Ratings {
			s, ok := values[rating.ID]
			if !ok {
				continue
			}
			bound := r.SelectedGenerations[i] == s.Ref
			if !bound || rating.TargetSHA != digest([]byte(s.Target.Text)) {
				continue
			}
			if s.Generation.Provenance.Session == r.Session {
				return nil, "", fmt.Errorf("generation/review session collision")
			}
			if rating.Rating != "A" || len(rating.Findings) != 0 {
				return nil, "", fmt.Errorf("unresolved B/C/D finding")
			}
			if accepted[rating.ID] {
				return nil, "", fmt.Errorf("ambiguous duplicate A evidence")
			}
			if err := requirePackageGlossaryCompatibility(root, locale, pkg, s.Generation.Batch.GlossarySHA, digest(glossary), s, compatibility); err != nil {
				return nil, "", err
			}
			accepted[rating.ID] = true
		}
	}
	if len(accepted) != len(values) {
		return nil, "", fmt.Errorf("package incomplete: pending independent A review")
	}
	targets := []Target{}
	for _, id := range selectAll(docs) {
		targets = append(targets, values[id].Target)
	}
	if err := validateTargets(root, locale, docs, targets); err != nil {
		return nil, "", err
	}
	result := &Finalization{Schema: WorkflowSchema, Locale: locale, Package: pkg, PackageSHA: p.Identity, SourceSHA: documentIdentity(docs), GlossarySHA: digest(glossary), Generations: generations, Reviews: reviews, Compatibility: compatibility, Documents: docs, Targets: targets}
	result.Identity = identity(*result)
	name := workflowPath(locale, pkg, "finalization", result.Identity)
	if save {
		if err := saveImmutable(root, name, result); err != nil {
			return nil, "", err
		}
	}
	return result, name, nil
}

// CheckFinalization reconstructs the machine decision, never trusting a saved
// passed flag. It performs no writes and does not replace language review.
func CheckFinalization(root string, ref Reference) (*Finalization, error) {
	var f Finalization
	if err := readReference(root, ref, &f); err != nil {
		return nil, err
	}
	copy := f
	copy.Identity = ""
	if f.Schema != WorkflowSchema || f.Identity != identity(copy) || ref.Path != workflowPath(f.Locale, f.Package, "finalization", f.Identity) {
		return nil, fmt.Errorf("invalid finalization identity")
	}
	want, _, err := finalizePackage(root, f.Locale, f.Package, f.Generations, f.Reviews, f.Compatibility, false)
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(f, *want) {
		return nil, fmt.Errorf("finalization stale")
	}
	return &f, nil
}
