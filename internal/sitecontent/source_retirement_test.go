package sitecontent

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/sourcestorage"
)

func TestSourceRetirementIndependentOfShellZIPAbsence(t *testing.T) {
	// Reuse real source/adoption validators in a disposable repository. Include
	// the fixture's Tour files in its manifest so full adoption is checked too.
	m, err := loadSourceManifest(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	extra := []string{}
	for _, f := range m.Files {
		if strings.HasPrefix(f.Path, "_content/tour/") {
			extra = append(extra, f.Path)
		}
	}
	root, _, _, _ := shellAdoptionFixture(t, extra...)
	for _, path := range shellRepresentationParserPaths {
		b, err := readRegular(repositoryRoot(t), path)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, path, b)
	}
	// Install an ordinary source/corpus successor. This exercises the nested
	// installed-corpus projection reader as well as the retirement entry point.
	name := "data/site-shell/translation.html"
	old, err := readRegular(root, name)
	if err != nil {
		t.Fatal(err)
	}
	at := bytes.LastIndex(old, []byte("</main>"))
	if at < 0 {
		t.Fatal("fixture main missing")
	}
	body := append(bytes.Clone(old[:at]), []byte("<section><p>Synthetic retirement source.</p></section>\n")...)
	body = append(body, old[at:]...)
	writeFixture(t, root, "data/retirement-source-body.html", body)
	proposal := StructuredSourceProposal{Schema: StructuredSourceSuccessorSchema, Package: "site-v2-shell", Previous: Reference{name, digest(old)}, Body: Reference{"data/retirement-source-body.html", digest(body)}, InsertAt: at}
	b, err := Encode(proposal)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "data/retirement-source-proposal.json", b)
	ref, err := ReferenceFile(root, "data/retirement-source-proposal.json")
	if err != nil {
		t.Fatal(err)
	}
	successor, err := BuildStructuredSourceSuccessor(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptStructuredSourceSuccessor(root, ref, successor.BaseIdentity); err != nil {
		t.Fatal(err)
	}
	corpus, err := readRegular(root, i18n.GlossaryCorpusPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := AdoptCorpusProjection(root, digest(corpus), true); err != nil {
		t.Fatal(err)
	}
	proof, _, err := CreateSourceStorageCompatibility(root, false)
	if err != nil {
		t.Fatal(err)
	}
	writeProof(t, root, *proof)
	zip, err := readRegular(root, SnapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(pathJoin(root, SnapshotPath)); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanShellRepresentationSuccessor(root, ShellTeachingRepresentation)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, input := range plan.Authority.Inputs {
		if input.Reference.Path == SnapshotPath {
			found = input.Absent
		}
	}
	if !found {
		t.Fatal("regression requires a frozen ZIP-absent representation")
	}
	b, err = Encode(plan)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, root, "data/retirement-representation-plan.json", b)
	input, err := ReferenceFile(root, "data/retirement-representation-plan.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyShellRepresentationSuccessor(root, input, plan.PriorIdentity, plan.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadCurrent(root); err != nil {
		t.Fatal("ZIP-absent fixture representation must be current", err)
	}
	writeFixture(t, root, SnapshotPath, zip)
	if _, err := LoadCurrent(root); err == nil || !strings.Contains(err.Error(), "representation dependency path/mode/absence changed") {
		t.Fatal("ordinary Shell currentness must still reject ZIP presence", err)
	}
	assertReady := func(t *testing.T) {
		t.Helper()
		r, err := CheckSourceZIPRetirement(root)
		if err != nil || r.Status != "READY" || r.CoveredReferences == 0 || r.CoveredEntries == 0 || len(r.Blockers) != 0 {
			t.Fatal("retirement eligibility depends on post-retirement representation", r, err)
		}
	}
	assertRejected := func(t *testing.T) {
		t.Helper()
		r, err := CheckSourceZIPRetirement(root)
		if err == nil && r.Status == "READY" {
			t.Fatal("unsafe ZIP retirement accepted", r)
		}
	}
	t.Run("valid-present", assertReady)
	for label, path := range map[string]string{
		"corrupt-present-zip":    SnapshotPath,
		"consumed-source-drift":  proof.Entries[0].NewPath,
		"manifest-drift":         SourceManifestPath,
		"corpus-bytes-drift":     i18n.GlossaryCorpusPath,
		"global-identity-drift":  GlobalPath,
		"proof-tamper":           sourcestorage.CompatibilityPath(*proof),
		"source-successor-drift": proposal.Body.Path,
	} {
		t.Run(label, func(t *testing.T) {
			prior, err := readRegular(root, path)
			if err != nil {
				t.Fatal(err)
			}
			defer writeFixture(t, root, path, prior)
			writeFixture(t, root, path, append(bytes.Clone(prior), 'x'))
			assertRejected(t)
		})
	}
	t.Run("missing-proof", func(t *testing.T) {
		path := sourcestorage.CompatibilityPath(*proof)
		b, err := readRegular(root, path)
		if err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, path, b)
		if err := os.Remove(pathJoin(root, path)); err != nil {
			t.Fatal(err)
		}
		assertRejected(t)
	})
	t.Run("invalid-additional-proof", func(t *testing.T) {
		bad := *proof
		bad.ManifestIdentity = digest([]byte("stale manifest"))
		sourcestorage.SealCompatibility(&bad)
		writeProof(t, root, bad)
		defer os.Remove(pathJoin(root, sourcestorage.CompatibilityPath(bad)))
		// Invalid old-byte claims cannot be skipped alongside a valid proof.
		bad.Entries = append([]sourcestorage.EquivalentEntry(nil), bad.Entries...)
		bad.Entries[0].OldSHA = digest([]byte("wrong source"))
		bad.Entries[0].NewSHA = bad.Entries[0].OldSHA
		sourcestorage.SealCompatibility(&bad)
		writeProof(t, root, bad)
		defer os.Remove(pathJoin(root, sourcestorage.CompatibilityPath(bad)))
		assertRejected(t)
	})
	t.Run("valid-absent", func(t *testing.T) {
		if err := os.Remove(pathJoin(root, SnapshotPath)); err != nil {
			t.Fatal(err)
		}
		defer writeFixture(t, root, SnapshotPath, zip)
		assertReady(t)
		if _, err := LoadCurrent(root); err != nil {
			t.Fatal("ordinary representation currentness after retirement", err)
		}
	})
	assertReady(t)
	if b, err := readRegular(root, SnapshotPath); err != nil || !bytes.Equal(b, zip) {
		t.Fatal("readonly checker modified the physical ZIP", err)
	}
	if b, err := readRegular(root, plan.Receipt.Path); err != nil || digest(b) != plan.Receipt.SHA256 {
		t.Fatal("historical representation receipt changed", err)
	}
}
