package sitecontent

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The lower-level transaction receives an already verified source Global.
// The public API always obtains that input from loadSourceCurrent; this tiny
// fixture isolates the new authority contract without copying a whole site.
func shellRepresentationFixture(t *testing.T) (string, *Global, []byte) {
	t.Helper()
	root := t.TempDir()
	raw := []byte("---\ntitle: Synthetic Homepage\nsummary: Synthetic source.\ntemplate: true\n---\n<h1>Go</h1><textarea>\n// You can edit this code!\n// Click here and start typing.\npackage main\n</textarea>")
	writeFixture(t, root, "_content/index.md", raw)
	locale, err := readRegular(repositoryRoot(t), "locales/zh-CN/locale.json")
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "locales/zh-CN/locale.json", locale)
	for _, name := range shellRepresentationParserPaths {
		b, err := readRegular(repositoryRoot(t), name)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, name, b)
	}
	for _, name := range []string{GlobalPath, SourceManifestPath, SnapshotPath} {
		writeFixture(t, root, name, []byte("synthetic frozen authority "+name))
	}
	s, err := ExtractShellLanguageSource("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	f := ShellSuccessorAuthority{Schema: ShellSuccessorSchema, Package: "site-v2-shell", State: "pending-source-authority", ExtractionContract: ShellExtractionContract, Sources: []ShellLanguageSource{*s}}
	f.Identity = identity(f)
	fr := Reference{Path: ShellSuccessorRoot + "sources/" + f.Identity + ".json"}
	if err := saveImmutable(root, fr.Path, f); err != nil {
		t.Fatal(err)
	}
	fr, _ = ReferenceFile(root, fr.Path)
	p := Package{ID: "site-v2-shell", Activation: "independent-shell-gates", ParserContract: ShellExtractionContract, Routes: []string{"/"}, Surfaces: []string{"homepage"}, Dependencies: []Dependency{}, SourceIdentity: digest(raw)}
	p.Identity = identity(p)
	a := ShellAdoption{Schema: ShellAdoptionSchema, Foundation: fr, Package: p, Sources: []Source{{Path: s.Path, Kind: "page", Origin: "frozen-upstream", Package: p.ID, Surface: "homepage", Route: "/", SHA256: digest(raw)}}}
	a.Identity = identity(a)
	if err := saveImmutable(root, shellAdoptionPath(&a), a); err != nil {
		t.Fatal(err)
	}
	ar, _ := ReferenceFile(root, shellAdoptionPath(&a))
	if err := saveImmutable(root, ShellAdoptionCurrent, ar); err != nil {
		t.Fatal(err)
	}
	packages := []Package{{ID: "tour-v1", Activation: "legacy-evidence", Dependencies: []Dependency{}, SourceIdentity: digest([]byte("tour"))}, p, {ID: "learn-docs-v1", Activation: "atomic-locale-package", ParserContract: UnitContract, Dependencies: []Dependency{}, SourceIdentity: digest([]byte("learn"))}}
	for i := range packages {
		packages[i].Identity = ""
		packages[i].Identity = identity(packages[i])
	}
	g := &Global{Schema: GlobalSchema, PublicName: PublicName, UpstreamRepository: "https://github.com/golang/website.git", UpstreamCommit: UpstreamCommit, SnapshotSHA256: digest([]byte("snapshot")), ShellAdoption: &ar, Packages: packages, Surfaces: []Surface{{ID: "homepage", Package: p.ID}}, Versions: []Version{{ID: "site-v1", Packages: []string{"tour-v1"}}, {ID: "site-v2", Packages: []string{"tour-v1", "site-v2-shell", "learn-docs-v1"}}}, Sources: a.Sources}
	g.Identity = identity(*g)
	if err := ValidateGlobal(*g); err != nil {
		t.Fatal(err)
	}
	return root, g, raw
}
func representationFixturePlan(t *testing.T, root string, g *Global) (*ShellRepresentationPlan, Reference) {
	t.Helper()
	var p *ShellRepresentationPlan
	err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
		var err error
		p, err = planShellRepresentationAt(root, g, ShellTeachingRepresentation, v, false)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := saveImmutable(root, "data/representation-plan.json", p); err != nil {
		t.Fatal(err)
	}
	ref, _ := ReferenceFile(root, "data/representation-plan.json")
	return p, ref
}
func TestShellRepresentationCanonicalAuthority(t *testing.T) {
	root, g, raw := shellRepresentationFixture(t)
	p, input := representationFixturePlan(t, root, g)
	if p.RawSourceChanged || p.Authority.Old.Source.SourceSHA != p.Authority.New.Source.SourceSHA || p.Authority.Old.Source.Page.Units[0].ID == p.Authority.New.Source.Page.Units[0].ID || p.Authority.Old.ProtectedInputSHA == p.Authority.New.ProtectedInputSHA || p.State != "generation-required" || p.AffectedPageCount != 1 {
		t.Fatal("invalid bounded transition", p)
	}
	if len(p.Authority.Released) != 1 || len(p.Authority.New.Source.Page.Units) != 1 {
		t.Fatal("physical Page or released exact set changed")
	}
	v := successorValidation(root, nil)
	inspection, err := inspectShellRepresentationAt(root, g, input, v, false)
	if err != nil || inspection.Plan == nil || inspection.Applied || inspection.AlreadyCurrent || inspection.Mutation != "none" {
		t.Fatal("plan inspection must be readonly", inspection, err)
	}
	if _, err := checkShellRepresentation(root, g, input, v, map[Reference]bool{}, 0); err == nil {
		t.Fatal("receipt verification accepted a plan")
	}
	before, _, err := selectShellRepresentationAt(root, g, raw, v)
	if err != nil || before.Page.Units[0].ID != p.Authority.Old.Source.Page.Units[0].ID {
		t.Fatal("prospective parser adopted without authority", err)
	}
	load := func(*structuredSuccessorValidation) (*Global, error) { return g, nil }
	if _, err := applyShellRepresentationSuccessor(root, input, "wrong", p.PriorPointerSHA, load, nil); err == nil {
		t.Fatal("wrong prior accepted")
	}
	r, err := applyShellRepresentationSuccessor(root, input, p.PriorIdentity, p.PriorPointerSHA, load, nil)
	if err != nil || !r.Applied {
		t.Fatal(r, err)
	}
	if _, err := os.Stat(pathJoin(root, ShellRepresentationPending)); !os.IsNotExist(err) {
		t.Fatal("pending marker leaked")
	}
	if _, err := os.Stat(pathJoin(root, "data/.glossary-corpus.lock")); !os.IsNotExist(err) {
		t.Fatal("owned lock leaked")
	}
	replay, err := applyShellRepresentationSuccessor(root, input, p.PriorIdentity, p.PriorPointerSHA, load, nil)
	if err != nil || !replay.AlreadyCurrent || replay.Applied || replay.Mutation != "none" {
		t.Fatal("exact replay mutated authority", replay, err)
	}
	var prior *structuredSuccessorValidation
	for i := 0; i < 2; i++ {
		err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
			if v == prior {
				return fmt.Errorf("invocation state shared")
			}
			prior = v
			current, authority, err := selectShellRepresentationAt(root, g, raw, v)
			if err != nil {
				return err
			}
			if authority.Identity != p.Authority.Identity || current.Page.Units[0].ID != p.Authority.New.Source.Page.Units[0].ID {
				return fmt.Errorf("new representation not selected")
			}
			oldPkg, _ := packageByID(g, "site-v2-shell")
			projected, err := applyShellRepresentationAuthority(root, g, v)
			if err != nil {
				return err
			}
			newPkg, _ := packageByID(projected, "site-v2-shell")
			if oldPkg.Identity == newPkg.Identity || projected.Sources[0].SHA256 != g.Sources[0].SHA256 {
				return fmt.Errorf("source/representation authority confused")
			}
			oldSource := p.Authority.Old.Source
			newSource := p.Authority.New.Source
			unit := func(s ShellLanguageSource) i18n.AdaptedTranslationUnit {
				return i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: s.Page.Units[0].ID, Kind: i18n.UnitKindPage, SourcePath: s.Path, Source: bytes.Clone(raw), SourceSHA256: digest(raw)}, Adapter: ShellPageAdapter{s, raw}, StableIndex: 1, Route: "/"}
			}
			c := &i18n.Catalog{}
			if err := c.RegisterTranslationPackage(i18n.TranslationPackageDescriptor{ID: newPkg.ID, AuthoritySHA256: newPkg.Identity, Units: []i18n.AdaptedTranslationUnit{unit(newSource)}}); err != nil {
				return err
			}
			c.HistoricalTranslationPackages = []i18n.TranslationPackageDescriptor{{ID: oldPkg.ID, AuthoritySHA256: oldPkg.Identity, Units: []i18n.AdaptedTranslationUnit{unit(oldSource)}}}
			if err := c.UseTranslationPackage(newPkg.ID); err != nil {
				return err
			}
			if !reflect.DeepEqual(c.TranslationScope, []string{newSource.Page.Units[0].ID}) {
				return fmt.Errorf("old Unit/A leaked into current scope")
			}
			// Even an independently A-rated predecessor cannot satisfy the new
			// current Unit: its first body is still English and now editable.
			oldOutput := strings.Replace(oldSource.Page.Units[0].Source, "Click here and start typing.", "点击此处开始输入。", 1)
			oldCandidate, err := (ShellPageAdapter{oldSource, raw}).Restore(oldOutput)
			if err != nil {
				return err
			}
			currentUnit, err := c.Unit(newSource.Page.Units[0].ID)
			if err != nil {
				return err
			}
			if _, err := i18n.ValidateTranslationUnitCommentCoverage(root, c, currentUnit, "zh-CN", oldCandidate); err == nil || !strings.Contains(err.Error(), "non-English teaching-comment exact source carry") {
				return fmt.Errorf("old A silently accepted by new Unit: %v", err)
			}
			newOutput := strings.NewReplacer("You can edit this code!", "你可以编辑这段代码！", "Click here and start typing.", "点击此处开始输入。").Replace(newSource.Page.Units[0].Source)
			newCandidate, err := (ShellPageAdapter{newSource, raw}).Restore(newOutput)
			if err != nil {
				return err
			}
			if _, err := i18n.ValidateTranslationUnitCommentCoverage(root, c, currentUnit, "zh-CN", newCandidate); err != nil {
				return err
			}
			h, err := i18n.HistoricalTranslationPackageCatalog(c, oldPkg.ID, oldPkg.Identity)
			if err != nil {
				return err
			}
			if !reflect.DeepEqual(h.TranslationScope, []string{oldSource.Page.Units[0].ID}) {
				return fmt.Errorf("historical predecessor lost")
			}
			if _, err := h.Unit(oldSource.Page.Units[0].ID); err != nil {
				return err
			}
			result, err := inspectShellRepresentationAt(root, g, input, v, false)
			if err != nil || !result.AlreadyCurrent || result.Applied || result.Mutation != "none" {
				return fmt.Errorf("readonly replay not recognized: %v", err)
			}
			// A returned object cannot mutate the next invocation's authority.
			current.Slots[0].Unit.Source = "caller mutation"
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	// Ordinary semantic approval data cannot create or replace this pointer.
	original, _ := readRegular(root, ShellRepresentationCurrent)
	writeFixture(t, root, "data/semantic-approval.json", []byte(`{"decision":"APPROVE_RECONCILIATION"}`))
	if after, _ := readRegular(root, ShellRepresentationCurrent); !bytes.Equal(original, after) {
		t.Fatal("semantic approval changed canonical authority")
	}
}

func TestShellRepresentationPlanFailClosed(t *testing.T) {
	root, g, _ := shellRepresentationFixture(t)
	plan, _ := representationFixturePlan(t, root, g)
	for _, label := range []string{"raw-sha", "old-identity", "new-identity", "wrong-package", "wrong-path", "foundation", "missing-predecessor", "discontinuous-lineage", "semantic-review", "ambiguous-pointer"} {
		t.Run(label, func(t *testing.T) {
			b, _ := Encode(plan)
			var p ShellRepresentationPlan
			_ = StrictJSON(b, &p)
			switch label {
			case "raw-sha":
				p.Authority.Basis.Source.SHA256 = digest([]byte("changed"))
			case "old-identity":
				p.Authority.Old.Identity = digest([]byte("wrong"))
			case "new-identity":
				p.Authority.New.Identity = digest([]byte("wrong"))
			case "wrong-package":
				p.Authority.Basis.Package.ID = "learn-docs-v1"
			case "wrong-path":
				p.Authority.Basis.Source.Path = "_content/other.md"
			case "foundation":
				p.Authority.Basis.Foundation.SHA256 = digest([]byte("wrong"))
			case "missing-predecessor":
				p.Authority.Predecessor = &Reference{"data/missing.json", digest(nil)}
			case "discontinuous-lineage":
				p.Authority.PredecessorIdentity = digest([]byte("unrelated authority"))
			}
			p.Authority.Identity = ""
			p.Authority.Identity = identity(p.Authority)
			p.Receipt = shellRepresentationReceipt(&p.Authority)
			p.Identity = ""
			p.Identity = identity(p)
			name := "data/negative-" + label + ".json"
			raw, _ := Encode(p)
			if label == "semantic-review" {
				raw = []byte(`{"schema":"go-learning/representation-semantic-review/v1","decision":"APPROVE_RECONCILIATION"}`)
			}
			if label == "ambiguous-pointer" {
				writeFixture(t, root, ShellRepresentationCurrent, []byte(`[{},{}]`))
				defer os.Remove(pathJoin(root, ShellRepresentationCurrent))
			}
			writeFixture(t, root, name, raw)
			ref, _ := ReferenceFile(root, name)
			err := withReadonlyStructuredValidation(root, func(v *structuredSuccessorValidation) error {
				_, err := inspectShellRepresentationAt(root, g, ref, v, false)
				return err
			})
			if err == nil {
				t.Fatal("invalid authority accepted")
			}
		})
	}
}

func TestShellRepresentationDependencyMutation(t *testing.T) {
	for _, mode := range []bool{false, true} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			root, g, _ := shellRepresentationFixture(t)
			p, input := representationFixturePlan(t, root, g)
			name := shellRepresentationParserPaths[0]
			old, _ := readRegular(root, name)
			_, err := applyShellRepresentationSuccessor(root, input, p.PriorIdentity, p.PriorPointerSHA, func(*structuredSuccessorValidation) (*Global, error) { return g, nil }, func() {
				if mode {
					_ = os.Chmod(pathJoin(root, name), 0600)
				} else {
					writeFixture(t, root, name, append(old, ' '))
				}
			})
			if err == nil {
				t.Fatal("post-preflight dependency mutation accepted")
			}
			if _, err := os.Stat(pathJoin(root, ShellRepresentationCurrent)); !os.IsNotExist(err) {
				t.Fatal("failed CAS installed pointer")
			}
		})
	}
	root, g, _ := shellRepresentationFixture(t)
	p, input := representationFixturePlan(t, root, g)
	if _, err := applyShellRepresentationSuccessor(root, input, p.PriorIdentity, p.PriorPointerSHA, func(*structuredSuccessorValidation) (*Global, error) { return g, nil }, nil); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, p.Receipt.Path, []byte(`{"tampered":true}`))
	if _, _, err := currentShellRepresentation(root, g, successorValidation(root, nil), false); err == nil {
		t.Fatal("tampered receipt accepted")
	}
}

func TestShellRepresentationProspectiveMinimalPage(t *testing.T) {
	_, _, raw := shellRepresentationFixture(t)
	old, err := ExtractShellPageRepresentation("_content/index.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	next, err := prospectiveShellRepresentation("_content/index.md", raw, ShellTeachingRepresentation)
	if err != nil {
		t.Fatal(err)
	}
	if old.SourceSHA != next.SourceSHA || len(next.Page.Units) != 1 || old.Page.Units[0].ID == next.Page.Units[0].ID {
		t.Fatal("invalid one-Page transition")
	}
	for _, s := range []struct {
		source *ShellLanguageSource
		want   int
	}{{old, 1}, {next, 2}} {
		count := 0
		for _, u := range s.source.Slots {
			if u.Role == "teaching-comment" {
				count++
			}
		}
		if count != s.want {
			t.Fatal(count, s.want)
		}
	}
	if len(next.Page.Units[0].Protected) != len(old.Page.Units[0].Protected)+1 {
		t.Fatal("unexpected token topology")
	}
}

func TestShellRepresentationRecoveryFailClosed(t *testing.T) {
	root, g, _ := shellRepresentationFixture(t)
	_, input := representationFixturePlan(t, root, g)
	for _, name := range []string{ShellRepresentationPending, "data/.glossary-corpus.lock"} {
		writeFixture(t, root, name, []byte("interrupted publication"))
		if _, err := inspectShellRepresentationAt(root, g, input, successorValidation(root, nil), false); err == nil {
			t.Fatal("pending/lock bypassed", name)
		}
		if err := os.Remove(filepath.Join(root, name)); err != nil {
			t.Fatal(err)
		} // isolated fixture cleanup only
	}
}

func TestShellRepresentationLineageAndSourceDrift(t *testing.T) {
	root, g, raw := shellRepresentationFixture(t)
	p, input := representationFixturePlan(t, root, g)
	if _, err := applyShellRepresentationSuccessor(root, input, p.PriorIdentity, p.PriorPointerSHA, func(*structuredSuccessorValidation) (*Global, error) { return g, nil }, nil); err != nil {
		t.Fatal(err)
	}
	v := successorValidation(root, nil)
	if _, err := checkShellRepresentation(root, g, p.Receipt, v, map[Reference]bool{p.Receipt: true}, 0); err == nil {
		t.Fatal("active successor cycle accepted")
	}
	if _, err := checkShellRepresentation(root, g, p.Receipt, v, map[Reference]bool{}, 33); err == nil {
		t.Fatal("ancestry depth limit lost")
	}
	r := p.Authority
	r.Predecessor = &p.Receipt
	r.Identity = ""
	r.Identity = identity(r)
	ref := shellRepresentationReceipt(&r)
	if err := saveImmutable(root, ref.Path, r); err != nil {
		t.Fatal(err)
	}
	if _, err := checkShellRepresentation(root, g, ref, successorValidation(root, nil), map[Reference]bool{}, 0); err == nil || !strings.Contains(err.Error(), "discontinuity") {
		t.Fatal("discontinuous self-addressed lineage accepted", err)
	}
	writeFixture(t, root, "_content/index.md", append(raw, ' '))
	if _, _, err := selectShellRepresentationAt(root, g, raw, successorValidation(root, nil)); err == nil {
		t.Fatal("new invocation accepted source drift")
	}
}

func TestShellRepresentationManifestSnapshotAbsence(t *testing.T) {
	root, g, _ := shellRepresentationFixture(t)
	if err := os.Remove(pathJoin(root, SnapshotPath)); err != nil {
		t.Fatal(err)
	}
	p, input := representationFixturePlan(t, root, g)
	if p.Authority.Basis.HistoricalSnapshotSHA != g.SnapshotSHA256 {
		t.Fatal("virtual historical snapshot identity lost")
	}
	found := false
	for _, dep := range p.Authority.Inputs {
		if dep.Reference.Path == SnapshotPath {
			found = dep.Absent && dep.Mode == 0 && dep.Reference.SHA256 == digest(nil)
		}
	}
	if !found {
		t.Fatal("physical snapshot absence not bound")
	}
	_, err := applyShellRepresentationSuccessor(root, input, p.PriorIdentity, p.PriorPointerSHA, func(*structuredSuccessorValidation) (*Global, error) { return g, nil }, func() { writeFixture(t, root, SnapshotPath, []byte("appeared during invocation")) })
	if err == nil {
		t.Fatal("snapshot absent-to-present mutation accepted")
	}
	if _, err := os.Stat(pathJoin(root, ShellRepresentationCurrent)); !os.IsNotExist(err) {
		t.Fatal("failed preflight installed authority")
	}
}
