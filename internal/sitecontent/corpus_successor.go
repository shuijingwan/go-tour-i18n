package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

const CorpusSuccessorSchema = "go-learning/unified-glossary-corpus-successor/v1"
const CorpusSuccessorRoot = "data/unified-glossary-corpus-successors/"

type CorpusSuccessor struct {
	ProjectionAuthorities []Reference                `json:"source_projection_authorities,omitempty"`
	Schema                string                     `json:"schema"`
	State                 string                     `json:"state"`
	Changed               bool                       `json:"changed"`
	Predecessor           Reference                  `json:"predecessor_corpus"`
	PredecessorIdentity   string                     `json:"predecessor_corpus_identity"`
	Source                Reference                  `json:"shell_source_authority"`
	SourceIdentity        string                     `json:"shell_source_identity"`
	Affected              Reference                  `json:"affected_scope_authority"`
	AffectedIdentity      string                     `json:"affected_scope_identity"`
	Additions             []i18n.LanguageContributor `json:"contributor_additions"`
	Retirements           []i18n.LanguageContributor `json:"contributor_retirements"`
	NotPresent            []string                   `json:"retirement_not_present"`
	UnchangedCount        int                        `json:"unchanged_contributor_count"`
	UnchangedIdentity     string                     `json:"unchanged_contributors_identity"`
	ProposedContexts      int                        `json:"proposed_context_count"`
	NetContexts           int                        `json:"net_context_delta"`
	Result                i18n.GlossarySourceCorpus  `json:"resulting_corpus"`
	ResultSHA             string                     `json:"resulting_corpus_file_sha256"`
	Identity              string                     `json:"identity_sha256"`
}

func readPendingShellCorpusInputs(root, input string, installed ...bool) (*ShellSuccessorAuthority, *ShellAffectedScope, Reference, error) {
	b, err := readRegular(root, input)
	if err != nil {
		return nil, nil, Reference{}, err
	}
	var s ShellAffectedScope
	if err = StrictJSON(b, &s); err != nil {
		return nil, nil, Reference{}, err
	}
	copy := s
	copy.Identity = ""
	canonical, _ := Encode(s)
	if s.Schema != ShellAffectedSchema || s.State != "pending-language-work" || s.Package != "site-v2-shell" || !component(s.Locale) || s.Identity != identity(copy) || input != ShellSuccessorRoot+"affected/"+s.Locale+"/"+s.Identity+".json" || !bytes.Equal(b, canonical) {
		return nil, nil, Reference{}, fmt.Errorf("invalid affected scope identity/path/canonical bytes")
	}
	var a ShellSuccessorAuthority
	if err = readReference(root, s.SourceAuthority, &a); err != nil {
		return nil, nil, Reference{}, err
	}
	ac := a
	ac.Identity = ""
	ab, err := readRegular(root, s.SourceAuthority.Path)
	if err != nil {
		return nil, nil, Reference{}, err
	}
	canonical, _ = Encode(a)
	if a.Schema != ShellSuccessorSchema || a.Package != "site-v2-shell" || a.State != "pending-source-authority" || a.ExtractionContract != ShellExtractionContract || a.Identity != identity(ac) || s.SourceIdentity != a.Identity || s.SourceAuthority.Path != ShellSuccessorRoot+"sources/"+a.Identity+".json" || !bytes.Equal(ab, canonical) {
		return nil, nil, Reference{}, fmt.Errorf("invalid pending shell source authority")
	}
	m, err := loadSourceManifest(root)
	if err != nil || m.Identity != a.FrozenManifestIdentity {
		return nil, nil, Reference{}, fmt.Errorf("shell source manifest stale")
	}
	if err := verifyShellParserDependenciesThroughAdoption(root, s.SourceAuthority, a); err != nil {
		return nil, nil, Reference{}, err
	}
	expected := []string{"_content/index.md", "_content/site.tmpl", "_content/menus.yaml", "_content/resources.yaml", "_content/testimonials.yaml"}
	if len(a.Sources) != len(expected) {
		return nil, nil, Reference{}, fmt.Errorf("shell source exact-set mismatch")
	}
	cs := []i18n.LanguageContributor{}
	for i, source := range a.Sources {
		if source.Path != expected[i] {
			return nil, nil, Reference{}, fmt.Errorf("shell source membership/order mismatch")
		}
		raw, err := readRegular(root, source.Path)
		if err != nil || digest(raw) != source.SourceSHA {
			return nil, nil, Reference{}, fmt.Errorf("shell frozen source stale")
		}
		c, err := ProjectShellSuccessorContributor(source)
		if err != nil {
			return nil, nil, Reference{}, err
		}
		cs = append(cs, c)
	}
	sort.Slice(cs, func(i, j int) bool { return cs[i].ID < cs[j].ID })
	if !reflect.DeepEqual(cs, s.Terminology.Contributors) || !reflect.DeepEqual(s.Terminology.RetiredSources, []string{a.HistoricalHome.Path}) || a.HistoricalHome.Path != "data/site-shell/home.html" {
		return nil, nil, Reference{}, fmt.Errorf("affected terminology projection differs from shell authority")
	}
	for _, ref := range []Reference{a.HistoricalHome, {Path: a.TranslationBody.Path, SHA256: a.TranslationBody.SourceSHA}, s.CurrentCompletion} {
		raw, err := readRegular(root, ref.Path)
		if err != nil || digest(raw) != ref.SHA256 {
			// Only the checked installed result may consume an immutable
			// predecessor via the exact reviewed activation lineage. Pending
			// creation/apply still requires byte-current completion authority.
			if ref == s.CurrentCompletion && len(installed) > 0 && installed[0] {
				if err := verifyHistoricalActivationContext(root, s.Locale, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256}); err != nil {
					return nil, nil, Reference{}, err
				}
				continue
			}
			return nil, nil, Reference{}, fmt.Errorf("shell predecessor/completion stale: %s", ref.Path)
		}
	}
	return &a, &s, Reference{input, digest(b)}, nil
}

func projectCorpusSuccessor(old *i18n.GlossarySourceCorpus, a *ShellSuccessorAuthority, s *ShellAffectedScope, refs []i18n.GlossaryArchiveReference) (*CorpusSuccessor, error) {
	p := &CorpusSuccessor{Schema: CorpusSuccessorSchema, State: "pending", Predecessor: s.Terminology.CurrentCorpus, PredecessorIdentity: old.Identity, Source: s.SourceAuthority, SourceIdentity: a.Identity, AffectedIdentity: s.Identity, Additions: s.Terminology.Contributors, Retirements: []i18n.LanguageContributor{}, NotPresent: []string{}, Result: i18n.GlossarySourceCorpus{Schema: old.Schema, Contributors: []i18n.LanguageContributor{}, Sources: refs}}
	retire := map[string]bool{}
	for _, path := range s.Terminology.RetiredSources {
		retire[path] = true
	}
	seen := map[string]bool{}
	retiredSeen := map[string]bool{}
	unchanged := []i18n.LanguageContributor{}
	for _, c := range old.Contributors {
		if retire[c.Path] || retiredSeen[c.Path] {
			if retiredSeen[c.Path] {
				return nil, fmt.Errorf("retirement has another independent source consumer")
			}
			if c.ID != "source:"+c.Path || c.Package != "site-v2-shell" || i18n.EffectiveContributorPurpose(c) != i18n.ContributorPurposeActiveContent {
				return nil, fmt.Errorf("retirement has another independent surface consumer")
			}
			p.Retirements = append(p.Retirements, c)
			delete(retire, c.Path)
			retiredSeen[c.Path] = true
			continue
		}
		seen[c.ID] = true
		unchanged = append(unchanged, c)
		p.Result.Contributors = append(p.Result.Contributors, c)
	}
	for path := range retire {
		p.NotPresent = append(p.NotPresent, path)
	}
	sort.Strings(p.NotPresent)
	if len(p.Additions) != 5 {
		return nil, fmt.Errorf("successor addition exact-set differs")
	}
	for _, c := range p.Additions {
		if seen[c.ID] {
			return nil, fmt.Errorf("addition conflicts with existing contributor")
		}
		seen[c.ID] = true
		p.ProposedContexts += len(c.Contexts)
		p.Result.Contributors = append(p.Result.Contributors, c)
	}
	if p.ProposedContexts != 235 {
		return nil, fmt.Errorf("shell proposed context projection changed")
	}
	p.UnchangedCount = len(unchanged)
	p.UnchangedIdentity = identity(unchanged)
	sort.Slice(p.Result.Contributors, func(i, j int) bool { return p.Result.Contributors[i].ID < p.Result.Contributors[j].ID })
	p.Result.Identity = i18n.CorpusIdentity(p.Result)
	p.Changed = p.Result.Identity != p.PredecessorIdentity
	if err := i18n.ValidateGlossaryCorpus(&p.Result); err != nil {
		return nil, err
	}
	for _, c := range p.Result.Contributors {
		p.NetContexts += len(c.Contexts)
	}
	for _, c := range old.Contributors {
		p.NetContexts -= len(c.Contexts)
	}
	b, err := Encode(p.Result)
	if err != nil {
		return nil, err
	}
	p.ResultSHA = digest(b)
	return p, nil
}

func BuildCorpusSuccessor(root, input string) (*CorpusSuccessor, error) {
	old, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		return nil, err
	}
	oldBytes, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return nil, err
	}
	canonical, _ := Encode(old)
	if !bytes.Equal(oldBytes, canonical) {
		return nil, fmt.Errorf("predecessor corpus non-canonical")
	}
	return buildCorpusSuccessorFrom(root, input, old, oldBytes)
}

func buildCorpusSuccessorFrom(root, input string, old *i18n.GlossarySourceCorpus, oldBytes []byte, installed ...bool) (*CorpusSuccessor, error) {
	a, s, affectedRef, err := readPendingShellCorpusInputs(root, input, installed...)
	if err != nil {
		return nil, err
	}
	if s.Terminology.CurrentCorpus != (Reference{i18n.GlossaryCorpusPath, digest(oldBytes)}) || s.Terminology.CurrentIdentity != old.Identity {
		return nil, fmt.Errorf("shell affected predecessor corpus stale")
	}
	refs := []i18n.GlossaryArchiveReference{}
	// The existing validated storage proof already proves these exact registered
	// bytes. Next corpus uses explicit local sources, not a fabricated ZIP proof.
	for _, ref := range old.Sources {
		if ref.Path != SnapshotPath && ref.Path != a.HistoricalHome.Path {
			refs = append(refs, ref)
		}
	}
	paths, err := i18n.CorpusStorageRequirements(old)
	if err != nil {
		return nil, err
	}
	seen := map[string]string{}
	for _, r := range refs {
		seen[r.Path] = r.SHA256
	}
	add := func(ref Reference) error {
		if h, ok := seen[ref.Path]; ok {
			if h != ref.SHA256 {
				return fmt.Errorf("corpus source reference conflict")
			}
			return nil
		}
		seen[ref.Path] = ref.SHA256
		refs = append(refs, i18n.GlossaryArchiveReference{Path: ref.Path, SHA256: ref.SHA256})
		return nil
	}
	for _, name := range paths {
		ref, err := ReferenceFile(root, name)
		if err != nil {
			return nil, err
		}
		if err = add(ref); err != nil {
			return nil, err
		}
	}
	for _, source := range a.Sources {
		if err = add(Reference{source.Path, source.SourceSHA}); err != nil {
			return nil, err
		}
	}
	for _, ref := range []Reference{s.SourceAuthority, affectedRef} {
		if err = add(ref); err != nil {
			return nil, err
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
	p, err := projectCorpusSuccessor(old, a, s, refs)
	if err != nil {
		return nil, err
	}
	p.Affected = affectedRef
	p.Identity = identity(*p)
	return p, nil
}

// Only an exact installed result backed by immutable successor evidence opts
// into this reader. Merely having a pending artifact never changes authority.
func CheckAppliedCorpusSuccessor(root string) (bool, error) {
	p, err := checkedAppliedCorpusSuccessor(root)
	return p != nil, err
}

// Return the fully reconstructed canonical projection, never a decoded prior PASS.
func checkedAppliedCorpusSuccessor(root string, contexts ...*structuredSuccessorValidation) (*CorpusSuccessor, error) {
	// Both reconstructions consume one exact source-authority observation.
	// Only this read-only subtree owns the aggregate guard; it returns no
	// intermediate proof and restores its caller's policy before returning.
	v := successorValidation(root, contexts)
	if v.guard.root != root {
		return nil, fmt.Errorf("corpus validation root changed")
	}
	return v.installedCorpusObservation(func() (*CorpusSuccessor, error) {
		return reconstructAppliedCorpusSuccessor(root, v)
	})
}

func reconstructAppliedCorpusSuccessor(root string, v *structuredSuccessorValidation) (*CorpusSuccessor, error) {
	contexts := []*structuredSuccessorValidation{v}
	current, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		return nil, err
	}
	if err := v.guard.bind(Reference{i18n.GlossaryCorpusPath, digest(current)}); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(pathJoin(root, CorpusSuccessorRoot))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			return nil, fmt.Errorf("unsafe corpus successor evidence")
		}
	}
	if err := v.guard.directory(CorpusSuccessorRoot); err != nil {
		return nil, err
	}
	if len(v.guard.dirs[CorpusSuccessorRoot]) != len(entries) {
		return nil, fmt.Errorf("corpus successor discovery changed during observation")
	}
	var selected *CorpusSuccessor
	for i, entry := range entries {
		if v.guard.dirs[CorpusSuccessorRoot][i] != entry.Name() {
			return nil, fmt.Errorf("corpus successor discovery changed during observation")
		}
		if !entry.Type().IsRegular() {
			return nil, fmt.Errorf("unsafe corpus successor evidence")
		}
		name := CorpusSuccessorRoot + entry.Name()
		b, err := readRegular(root, name)
		if err != nil {
			return nil, err
		}
		var p CorpusSuccessor
		if err = StrictJSON(b, &p); err != nil {
			return nil, err
		}
		copy := p
		copy.Identity = ""
		canonical, _ := Encode(p)
		if p.Schema != CorpusSuccessorSchema || p.State != "pending" || p.Identity != identity(copy) || name != CorpusSuccessorPath(&p) || !bytes.Equal(b, canonical) {
			return nil, fmt.Errorf("invalid corpus successor evidence")
		}
		if p.ResultSHA == digest(current) {
			if selected != nil {
				return nil, fmt.Errorf("ambiguous installed corpus successor")
			}
			selected = &p
		}
	}
	if selected == nil {
		return nil, nil
	}
	// Corpus projection consumes source adoption, not a runtime representation
	// envelope. Requiring that envelope here creates a retirement cycle when
	// it freezes ZIP absence while the storage checker must inspect a present ZIP.
	// The same source/current projection and exact successor checks below remain
	// mandatory; ordinary LoadCurrent still verifies representation currentness.
	g, err := loadSourceCurrent(root, contexts...)
	if err != nil {
		return nil, err
	}
	if len(selected.ProjectionAuthorities) == 0 && (len(g.PackageAdoptions) > 0 || len(g.StructuredSuccessors) > 0) {
		return nil, nil
	}
	archive := "data/glossary-corpus-history/" + selected.Predecessor.SHA256 + ".json"
	var old *i18n.GlossarySourceCorpus
	if len(selected.ProjectionAuthorities) > 0 {
		old, err = readProjectionCorpus(root, archive)
	} else {
		old, err = i18n.LoadGlossaryCorpusAt(root, archive)
	}
	if err != nil {
		return nil, err
	}
	oldBytes, err := readRegular(root, archive)
	if err != nil {
		return nil, err
	}
	if digest(oldBytes) != selected.Predecessor.SHA256 {
		return nil, fmt.Errorf("historical corpus archive SHA mismatch")
	}
	if err := v.guard.bind(Reference{archive, selected.Predecessor.SHA256}); err != nil {
		return nil, err
	}
	var want *CorpusSuccessor
	if len(selected.ProjectionAuthorities) > 0 {
		want, err = buildAdoptedCorpusSuccessorFrom(root, old, oldBytes, contexts...)
	} else {
		want, err = buildCorpusSuccessorFrom(root, selected.Affected.Path, old, oldBytes, true)
	}
	if err != nil {
		return nil, err
	}
	if !reflect.DeepEqual(selected, want) {
		return nil, fmt.Errorf("installed corpus successor stale")
	}
	b, _ := Encode(want.Result)
	if !bytes.Equal(current, b) {
		return nil, fmt.Errorf("installed corpus non-canonical")
	}
	if _, err = i18n.LoadUnifiedGlossaryCorpus(root); err != nil {
		return nil, err
	}
	if err := v.verifyObserved(); err != nil {
		return nil, err
	}
	return want, nil
}

func CorpusSuccessorPath(p *CorpusSuccessor) string {
	return CorpusSuccessorRoot + p.Identity + ".json"
}

// Apply is explicit and tested only in isolated fixtures this task. Every
// prerequisite is checked before CAS; a durable pending marker makes any
// interrupted publication fail closed rather than guessing mutation outcome.
func ApplyCorpusSuccessor(root, input, prior string) (*CorpusSuccessor, error) {
	if !validSHA(prior) {
		return nil, fmt.Errorf("exact --prior-sha256 required")
	}
	p, err := BuildCorpusSuccessor(root, input)
	if err != nil {
		return nil, err
	}
	if prior != p.Predecessor.SHA256 {
		return nil, fmt.Errorf("predecessor CAS differs")
	}
	return p, installCorpusSuccessor(root, p, prior)
}

func installCorpusSuccessor(root string, p *CorpusSuccessor, prior string) error {
	if prior != p.Predecessor.SHA256 || !validSHA(prior) {
		return fmt.Errorf("predecessor CAS differs")
	}
	lockPath := pathJoin(root, "data/.glossary-corpus.lock")
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if err = lock.Close(); err != nil {
		return err
	}
	defer os.Remove(lockPath)
	marker := pathJoin(root, "data/.corpus-successor-pending")
	if _, err := os.Lstat(marker); !os.IsNotExist(err) {
		return fmt.Errorf("mutation=unknown: corpus successor publication pending")
	}
	old, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil || digest(old) != prior {
		return fmt.Errorf("concurrent corpus mutation")
	}
	if err = SaveArtifact(root, "data/glossary-corpus-history/"+prior+".json", old); err != nil {
		return err
	}
	if err = saveImmutable(root, CorpusSuccessorPath(p), p); err != nil {
		return err
	}
	next, err := Encode(p.Result)
	if err != nil || digest(next) != p.ResultSHA {
		return fmt.Errorf("successor result identity differs")
	}
	f, err := os.CreateTemp(filepath.Join(root, "data"), ".corpus-successor-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(next); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	old, err = readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil || digest(old) != prior {
		return fmt.Errorf("concurrent corpus mutation")
	}
	if len(p.ProjectionAuthorities) > 0 {
		for _, ref := range p.Result.Sources {
			b, err := readRegular(root, ref.Path)
			if err != nil || digest(b) != ref.SHA256 {
				return fmt.Errorf("concurrent corpus source dependency mutation: %s", ref.Path)
			}
		}
	}
	if err = SaveArtifact(root, "data/.corpus-successor-pending", []byte(p.Identity+"\n")); err != nil {
		return err
	}
	if err = os.Rename(f.Name(), pathJoin(root, i18n.GlossaryCorpusPath)); err != nil {
		return fmt.Errorf("mutation=unknown: %w", err)
	}
	if err = os.Remove(marker); err != nil {
		return fmt.Errorf("mutation=unknown: %w", err)
	}
	return nil
}
