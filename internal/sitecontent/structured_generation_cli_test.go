package sitecontent

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
)

// The production CLI is executed only with cwd=t.TempDir(). All publication
// below constructs disposable fixtures using the existing canonical helpers.
func TestStructuredGenerationCheckCLIIsolated(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	binary := filepath.Join(t.TempDir(), "tour-i18n")
	build := exec.Command("go", "build", "-mod=readonly", "-o", binary, "./cmd/tour-i18n")
	build.Dir = repositoryRoot(t)
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatal("CLI compile", err, string(b))
	}
	root, closure, normal, _ := sharedRepresentationFixtureWithHomepage(t, true)
	inventory := func() map[string]string {
		t.Helper()
		m := map[string]string{}
		if err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			m[filepath.ToSlash(rel)] = digest(b)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		return m
	}
	check := func(ref Reference, pass bool) []byte {
		t.Helper()
		before := inventory()
		cmd := exec.Command(binary, "site-content", "structured-generation-check", "--input", ref.Path, "--input-sha256", ref.SHA256)
		cmd.Dir = root
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if !reflect.DeepEqual(before, inventory()) {
			t.Fatal("readonly CLI changed fixture")
		}
		if pass && err != nil {
			t.Fatal("CLI rejected fixture", err, stderr.String())
		}
		if !pass && (err == nil || len(out) != 0) {
			t.Fatal("CLI accepted invalid authority or emitted success", err, string(out))
		}
		return out
	}
	var summary struct {
		Schema         string     `json:"schema"`
		Status         string     `json:"status"`
		Generation     Reference  `json:"generation"`
		Locale         string     `json:"locale"`
		Plan           string     `json:"plan_identity_sha256"`
		Outputs        int        `json:"output_count"`
		Files          int        `json:"validated_file_count"`
		Recovery       *Reference `json:"timestamp_recovery,omitempty"`
		OriginalStatus string     `json:"original_generated_at_status,omitempty"`
		Mutation       string     `json:"mutation"`
	}
	first := check(normal, true)
	if err := StrictJSON(first, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Schema != "go-learning/structured-generation-check/v1" || summary.Status != "PASS" || summary.Mutation != "none" || summary.Generation != normal || summary.Files == 0 || summary.Recovery != nil || summary.OriginalStatus != "" {
		t.Fatal("normal summary", summary)
	}
	if again := check(normal, true); !bytes.Equal(first, again) {
		t.Fatal("nondeterministic summary")
	}

	// Build a genuine FAILED finding and shared representation A -> B
	// dependency lineage; no checker or currentness gate is mocked.
	scope, err := BuildIntegratedSurfaceScope(root, sharedFixtureLocale, []Reference{closure})
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Synthetic independent exact structured finding\nFrozen ordinary action.\n")
	writeFixture(t, root, "data/check-cli-finding.md", md)
	review := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "check-cli-finding", Reviewer: "synthetic-independent-reviewer", Decision: "failed", Findings: []string{"frozen atom"}, ExactFindings: []SurfaceFinding{{ID: "_content/site.tmpl", Class: "structured-surface-language", Finding: "frozen atom"}}, Evidence: Reference{"data/check-cli-finding.md", digest(md)}}
	rb, _ := Encode(review)
	_, rp, err := RecordIntegratedSurface(root, rb)
	if err != nil {
		t.Fatal(err)
	}
	finding, _ := ReferenceFile(root, rp)
	raw := mustSharedFixtureBytes(t, root, "_content/site.tmpl")
	old, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	q := SharedRepresentationRequest{Schema: SharedRepresentationSchema + "/request", Package: "site-v2-shell", AssetID: "source:_content/site.tmpl", Path: "_content/site.tmpl", SourceSHA: digest(raw), PreviousIdentity: identity(*old), Generation: normal, Rule: SharedRepresentationRule, Slots: []string{"0719e14f5f889a9c0dccb5eb291c9fe1de9604b6f5423b3c1497204c4276a85d"}}
	request := landSharedReattestationFixture(t, root, "data/check-cli-request.json", q)
	p, err := PlanSharedStructuredRepresentation(root, request)
	if err != nil {
		t.Fatal(err)
	}
	plan := landSharedReattestationFixture(t, root, "data/check-cli-plan.json", p)
	if _, err := ApplySharedStructuredRepresentation(root, plan, p.PriorIdentity, p.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	parser := sharedRepresentationParserPaths[0]
	advance := func(epoch string) *SharedRepresentationReattestationPlan {
		t.Helper()
		writeFixture(t, root, parser, append(mustSharedFixtureBytes(t, root, parser), []byte("\n// CLI fixture epoch "+epoch+"\n")...))
		x, err := PlanSharedRepresentationReattestation(root, p.Receipt)
		if err != nil {
			t.Fatal(err)
		}
		ref := landSharedReattestationFixture(t, root, "data/check-cli-epoch-"+epoch+".json", x)
		if _, err := ApplySharedRepresentationReattestation(root, ref, x.PriorPointerSHA); err != nil {
			t.Fatal(err)
		}
		return x
	}
	advance("A")
	bundle, m, err := ExportStructuredAssetsRevision(root, sharedFixtureLocale, finding)
	if err != nil {
		t.Fatal(err)
	}
	o := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{}}
	for _, a := range m.Plan.Assets {
		for _, path := range m.Plan.Expected {
			if a.TargetPath == path {
				target := StructuredTarget{Path: path, Slots: []Target{}}
				for _, u := range a.Document.Units {
					target.Slots = append(target.Slots, Target{u.ID, u.Source})
				}
				o.Outputs = append(o.Outputs, target)
			}
		}
	}
	out, _ := Encode(o)
	loss, _, _ := timestampTestRequest(t, root, bundle, out, Provenance{Provider: "chatgpt", Model: "gpt-5.6-sol-high", Session: "synthetic-generation"})
	bEpoch := advance("B")
	applied, err := RecoverStructuredTimestamp(root, loss, true)
	if err != nil {
		t.Fatal(err)
	}
	var generation StructuredGeneration
	if err := readReference(root, applied.Generation, &generation); err != nil {
		t.Fatal(err)
	}
	var authority StructuredTimestampAuthority
	if err := readReference(root, applied.Authority, &authority); err != nil {
		t.Fatal(err)
	}
	if authority.Bridge == nil || generation.Provenance.GeneratedAt != "" {
		t.Fatal("fixture lacks exact currentness bridge")
	}
	result := check(applied.Generation, true)
	if err := StrictJSON(result, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Generation != applied.Generation || summary.Recovery == nil || *summary.Recovery != applied.Authority || summary.OriginalStatus != "unavailable" || summary.Files != 1 || summary.Outputs != 1 || summary.Plan != generation.Plan.Identity || summary.Mutation != "none" {
		t.Fatal("recovery summary", summary)
	}

	for _, name := range []string{"generation-bytes", "generation-identity", "recovery", "bundle", "output", "output-zip", "evidence", "current-bundle", "lineage", "current-dependency"} {
		t.Run(name, func(t *testing.T) {
			path := applied.Generation.Path
			ref := applied.Generation
			value := []byte("{}\n")
			switch name {
			case "generation-identity":
				bad := generation
				bad.Identity = strings.Repeat("a", 64)
				value, _ = Encode(bad)
			case "recovery":
				path = applied.Authority.Path
			case "bundle":
				path = authority.Bundle.Path
			case "output":
				path = authority.Output.Path
			case "output-zip":
				path = authority.OutputZIP.Path
			case "evidence":
				path = authority.Evidence.Path
			case "current-bundle":
				path = authority.Bridge.CurrentBundle.Path
			case "lineage":
				path = bEpoch.Receipt.Path
			case "current-dependency":
				path = bEpoch.CurrentPath
			}
			before := mustSharedFixtureBytes(t, root, path)
			defer writeFixture(t, root, path, before)
			writeFixture(t, root, path, value)
			if name == "generation-identity" {
				ref, _ = ReferenceFile(root, path)
			}
			check(ref, false)
		})
	}
	for _, name := range []string{"empty-without-authority", "UNKNOWN"} {
		t.Run(name, func(t *testing.T) {
			bad := generation
			if name == "UNKNOWN" {
				bad.Provenance.GeneratedAt = "UNKNOWN"
			} else {
				bad.TimestampRecovery = nil
			}
			bad.Identity = ""
			bad.Identity = identity(bad)
			ref := landSharedReattestationFixture(t, root, "data/locale-language/"+sharedFixtureLocale+"/structured-generation/"+bad.Identity+".json", bad)
			check(ref, false)
		})
	}
	t.Log("NORMAL_GENERATION_CHECK=PASS TIMESTAMP_RECOVERY_GENERATION_CHECK=PASS READONLY_MUTATION=none")
}
