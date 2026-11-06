package sitecontent

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func structuredRecoveryFixture(t *testing.T) (string, []byte, []byte, StructuredRecoveryOptions, Reference) {
	t.Helper()
	root := siteFixture(t, "current-authorization-fresh-fixture")
	r, ref := importStructuredFixture(t, root)
	b, err := readRegularReference(root, r.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	o, err := Encode(r.Output)
	if err != nil {
		t.Fatal(err)
	}
	return root, b, o, StructuredRecoveryOptions{BundleSHA: digest(b), OutputSHA: digest(o), Provenance: r.Provenance}, ref
}

func TestStructuredRecoveryPreservesOriginalInvocationAndConsumers(t *testing.T) {
	root, b, o, options, native := structuredRecoveryFixture(t)
	nativeBefore, _ := readRegularReference(root, native)
	// Execution policy changes raw transport, but not new profiled language
	// currentness. Validator code evolves independently and still runs current.
	writeFixture(t, root, "AGENTS.md", []byte("New execution-only Call-Zero-First instruction\n"))
	writeFixture(t, root, "internal/i18n/parsed_glossary.go", []byte("Synthetic validator implementation successor\n"))
	if _, err := CheckStructuredAssetsBundle(root, b); err != nil {
		t.Fatal("execution-only drift invalidated language input", err)
	}
	preflight, err := RecoverStructuredAssets(root, b, o, options)
	if err != nil {
		t.Fatal(err)
	}
	if preflight.Applied || preflight.ValidationFailed != 0 || preflight.OriginalOutput.SHA256 != digest(o) || preflight.OriginalBundle.SHA256 != digest(b) || preflight.CarryForward.CurrentBundle.SHA256 == digest(b) {
		t.Fatalf("preflight=%+v", preflight)
	}
	if _, err := os.Stat(filepath.Join(root, preflight.Generation.Path)); !os.IsNotExist(err) {
		t.Fatal("preflight mutated formal evidence")
	}
	options.Apply = true
	applied, err := RecoverStructuredAssets(root, b, o, options)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || !reflect.DeepEqual(applied.Provenance, options.Provenance) {
		t.Fatalf("applied=%+v", applied)
	}
	r, files, err := CheckStructuredGeneration(root, applied.Generation)
	if err != nil {
		t.Fatal("existing closure/Surface generation consumer rejected lineage", err)
	}
	if len(files) != applied.ValidationPassed || r.Output.BundleSHA != preflight.OriginalIdentity || r.Bundle != applied.OriginalBundle || !reflect.DeepEqual(r.Provenance, options.Provenance) {
		t.Fatal("original invocation relabeled")
	}
	for _, a := range r.Plan.Assets {
		if a.Document != nil {
			if _, ok := files[a.TargetPath]; ok && !structuredSupersedes(root, r, native, a.Document.Path) {
				t.Fatal("carry-forward cannot supersede its original native record")
			}
		}
	}
	for _, ref := range []Reference{applied.OriginalOutput, applied.OriginalBundle, native} {
		got, err := readRegularReference(root, ref)
		if err != nil {
			t.Fatal(err)
		}
		want := o
		if ref == applied.OriginalBundle {
			want = b
		}
		if ref == native {
			want = nativeBefore
		}
		if !bytes.Equal(got, want) {
			t.Fatal("original evidence overwritten")
		}
	}
	// Replaying the same exact immutable edge is idempotent, not an invocation.
	second, err := RecoverStructuredAssets(root, b, o, options)
	if err != nil || second.Generation != applied.Generation {
		t.Fatalf("second=%+v error=%v", second, err)
	}
	// Historical readers keep the frozen edge; current consumers detect a real
	// semantic authority mutation without relying on Plan identity alone.
	p := StructuredSemanticAuthorities[0]
	before, _ := readRegular(root, p)
	writeFixture(t, root, p, append(before, []byte("New language semantic requirement\n")...))
	if _, _, _, err := historicalStructuredGeneration(root, applied.Generation); err != nil {
		t.Fatal("historical edge rewritten by current docs", err)
	}
	if _, _, err := CheckStructuredGeneration(root, applied.Generation); err == nil {
		t.Fatal("current consumer accepted semantic authority drift")
	}
}

func rewriteRecoveryBundle(t *testing.T, b []byte, change func(*StructuredBundle, map[string][]byte)) []byte {
	t.Helper()
	m, fs, err := readStructuredBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	change(m, fs)
	entries := []i18n.TransportBundleEntry{}
	for i, f := range m.Files {
		m.Files[i] = i18n.NewTransportBundleFile(f.BundlePath, f.RepositoryPath, fs[f.BundlePath])
		entries = append(entries, i18n.TransportBundleEntry{Path: f.BundlePath, Data: fs[f.BundlePath]})
	}
	m.Plan.Identity = ""
	m.Plan.Identity = identity(m.Plan)
	m.Identity = ""
	m.Identity = identity(m)
	manifest, _ := Encode(m)
	result, err := i18n.WriteDeterministicTransportBundle(manifest, entries)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestStructuredRecoveryRejectsDriftAndForgery(t *testing.T) {
	root, original, output, opts, _ := structuredRecoveryFixture(t)
	for _, name := range []string{"source", "semantic-authority", "glossary", "locale", "slot-source", "protected-structure", "expected-set", "unknown-legacy-authority", "missing-execution-context", "forged-bundle", "forged-output", "output-binding", "manual-identity-substitution", "validator-failure", "provenance-model", "provenance-session", "missing-sha"} {
		t.Run(name, func(t *testing.T) {
			b, o, options := bytes.Clone(original), bytes.Clone(output), opts
			var restore func()
			mutate := func(p string, v []byte) {
				before, err := readRegular(root, p)
				if err != nil {
					t.Fatal(err)
				}
				writeFixture(t, root, p, v)
				restore = func() { writeFixture(t, root, p, before) }
			}
			switch name {
			case "source":
				mutate("data/site-shell/home.html", []byte("<h1>Real changed source</h1>\n"))
			case "semantic-authority":
				mutate(StructuredSemanticAuthorities[0], []byte("New actual language rules\n"))
			case "glossary":
				before, _ := readRegular(root, "locales/zh-CN/glossary.yaml")
				mutate("locales/zh-CN/glossary.yaml", append(before, []byte("keep:\n  - Go\n")...))
			case "locale":
				before, _ := readRegular(root, "locales/zh-CN/locale.json")
				mutate("locales/zh-CN/locale.json", append(before, '\n'))
			case "slot-source", "protected-structure", "expected-set", "unknown-legacy-authority", "missing-execution-context":
				b = rewriteRecoveryBundle(t, b, func(m *StructuredBundle, fs map[string][]byte) {
					if name == "missing-execution-context" {
						kept := []i18n.TransportBundleFile{}
						for _, f := range m.Files {
							if f.BundlePath != "inputs/AGENTS.md" {
								kept = append(kept, f)
							}
						}
						m.Files = kept
						return
					}
					if name == "expected-set" {
						m.Plan.Expected = m.Plan.Expected[1:]
						return
					}
					if name == "unknown-legacy-authority" {
						m.AuthorityProfile = ""
						fs["inputs/AGENTS.md"] = []byte("Unreviewed legacy semantic/execution policy\n")
						return
					}
					for i := range m.Plan.Assets {
						a := &m.Plan.Assets[i]
						if a.Document == nil {
							continue
						}
						if name == "slot-source" {
							a.Document.Units[0].Source += " real drift"
						} else {
							a.Document.StructureSHA = strings.Repeat("a", 64)
						}
						break
					}
				})
				options.BundleSHA = digest(b)
				m, _, _ := readStructuredBundle(b)
				var out StructuredOutput
				_ = StrictJSON(o, &out)
				out.BundleSHA = m.Identity
				if name == "expected-set" {
					out.Outputs = out.Outputs[1:]
				}
				o, _ = Encode(out)
				options.OutputSHA = digest(o)
			case "forged-bundle":
				b[len(b)/2] ^= 1
			case "forged-output":
				o = bytes.Replace(o, []byte(`"schema"`), []byte(`"forged"`), 1)
			case "output-binding", "manual-identity-substitution":
				var out StructuredOutput
				_ = StrictJSON(o, &out)
				out.BundleSHA = strings.Repeat("a", 64)
				if name == "manual-identity-substitution" {
					writeFixture(t, root, "AGENTS.md", []byte("Execution drift\n"))
					defer writeFixture(t, root, "AGENTS.md", mustReadRecoveryTest(t, original, "inputs/AGENTS.md"))
					_, m, err := ExportStructuredAssets(root, "zh-CN")
					if err != nil {
						t.Fatal(err)
					}
					out.BundleSHA = m.Identity
				}
				o, _ = Encode(out)
				options.OutputSHA = digest(o)
			case "validator-failure":
				var out StructuredOutput
				_ = StrictJSON(o, &out)
				for i := range out.Outputs {
					if len(out.Outputs[i].Slots) > 0 {
						out.Outputs[i].Slots[0].Text = "<script>illegal markup</script>"
						break
					}
				}
				o, _ = Encode(out)
				options.OutputSHA = digest(o)
			case "provenance-model":
				options.Provenance.Model = "fake-model"
			case "provenance-session":
				options.Provenance.Session = "synthetic-independent-reviewer"
			case "missing-sha":
				options.OutputSHA = ""
			}
			if restore != nil {
				defer restore()
			}
			options.Apply = true
			before, _ := os.ReadDir(filepath.Join(root, "data/locale-language/zh-CN/structured-generation"))
			if _, err := RecoverStructuredAssets(root, b, o, options); err == nil {
				t.Fatal("drift/forgery accepted")
			}
			after, _ := os.ReadDir(filepath.Join(root, "data/locale-language/zh-CN/structured-generation"))
			if len(before) != len(after) {
				t.Fatal("failed recovery wrote evidence")
			}
		})
	}
}

func TestStructuredRecoveryRejectsFrozenLineageTampering(t *testing.T) {
	root, b, o, options, _ := structuredRecoveryFixture(t)
	options.Apply = true
	applied, err := RecoverStructuredAssets(root, b, o, options)
	if err != nil {
		t.Fatal(err)
	}
	r, _, _, err := readStructuredGeneration(root, applied.Generation)
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []Reference{applied.OriginalOutput, applied.CarryForward.CurrentBundle, applied.OriginalBundle} {
		before, err := readRegularReference(root, ref)
		if err != nil {
			t.Fatal(err)
		}
		writeFixture(t, root, ref.Path, append(bytes.Clone(before), 'x'))
		if _, _, _, err := readStructuredGeneration(root, applied.Generation); err == nil {
			t.Fatal("archive tampering accepted", ref.Path)
		}
		writeFixture(t, root, ref.Path, before)
	}
	copy := *r
	carry := *r.CarryForward
	carry.ValidatedFiles = append([]Reference{}, carry.ValidatedFiles...)
	carry.ValidatedFiles[0].SHA256 = strings.Repeat("a", 64)
	copy.CarryForward = &carry
	copy.Identity = ""
	copy.Identity = identity(copy)
	path := "data/locale-language/" + copy.Plan.Locale + "/structured-generation/" + copy.Identity + ".json"
	if err := saveImmutable(root, path, copy); err != nil {
		t.Fatal(err)
	}
	ref, err := ReferenceFile(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := historicalStructuredGeneration(root, ref); err == nil {
		t.Fatal("forged target validation identity accepted")
	}
}

func TestStructuredRecoverySemanticComparisonIsLocaleIndependent(t *testing.T) {
	for _, locale := range []string{"ja-JP", "fr-FR"} {
		fs := map[string][]byte{}
		m := StructuredBundle{Schema: StructuredSchema, AuthorityProfile: StructuredAuthorityProfile, Plan: StructuredPlan{Schema: StructuredSchema, Locale: locale, Expected: []string{"locales/" + locale + "/structured/source.yaml"}}, Files: []i18n.TransportBundleFile{}}
		for _, p := range append(append([]string{}, PackageAuthorities...), StructuredSemanticAuthorities...) {
			b := []byte("Synthetic frozen authority " + p)
			name := "inputs/" + p
			fs[name] = b
			m.Files = append(m.Files, i18n.NewTransportBundleFile(name, p, b))
		}
		currentFiles := map[string][]byte{}
		for p, b := range fs {
			currentFiles[p] = bytes.Clone(b)
		}
		currentFiles["inputs/AGENTS.md"] = []byte("Execution-only mutation")
		if err := compatibleStructuredBundles(m, fs, m, currentFiles); err != nil {
			t.Fatal(locale, err)
		}
		currentFiles["inputs/"+StructuredSemanticAuthorities[0]] = []byte("Real semantic mutation")
		if err := compatibleStructuredBundles(m, fs, m, currentFiles); err == nil {
			t.Fatal("semantic drift accepted", locale)
		}
		unknown := "docs/FUTURE_GENERATION_RULES.md"
		name := "inputs/" + unknown
		fs[name] = []byte("Frozen unclassified authority")
		m.Files = append(m.Files, i18n.NewTransportBundleFile(name, unknown, fs[name]))
		currentFiles["inputs/"+StructuredSemanticAuthorities[0]] = fs["inputs/"+StructuredSemanticAuthorities[0]]
		currentFiles[name] = []byte("Changed actual generation rules")
		if err := compatibleStructuredBundles(m, fs, m, currentFiles); err == nil {
			t.Fatal("unclassified future docs mutation accepted", locale)
		}
		root := t.TempDir()
		for _, p := range StructuredSemanticAuthorities {
			writeFixture(t, root, p, fs["inputs/"+p])
		}
		writeFixture(t, root, unknown, currentFiles[name])
		if err := checkStructuredSemanticAuthority(root, m, fs); err == nil {
			t.Fatal("current consumer ignored unknown semantic authority mutation", locale)
		}
	}
}

func mustReadRecoveryTest(t *testing.T, b []byte, p string) []byte {
	t.Helper()
	_, fs, err := readStructuredBundle(b)
	if err != nil {
		t.Fatal(err)
	}
	return fs[p]
}

func TestStructuredRecoveryLegacyAuthorityIsExactAndLocaleIndependent(t *testing.T) {
	// The migration approves exact reviewed versions, never a locale or a .md
	// suffix. Both registries must cover the full declared authority inventory.
	if len(legacyStructuredAuthoritySHA) != len(PackageAuthorities) || len(legacyStructuredSemanticSHA) != len(StructuredSemanticAuthorities) {
		t.Fatal("non-exact migration authority inventory")
	}
	for _, p := range PackageAuthorities {
		if !validSHA(legacyStructuredAuthoritySHA[p]) {
			t.Fatal(p)
		}
	}
	for _, p := range StructuredSemanticAuthorities {
		b, err := readRegular(repositoryRoot(t), p)
		if err != nil || digest(b) != legacyStructuredSemanticSHA[p] {
			t.Fatal("audited semantic migration baseline changed", p, err)
		}
	}
	for p := range legacyStructuredAuthoritySHA {
		if strings.Contains(p, "zh-CN") || strings.Contains(p, "locale") {
			t.Fatal("locale-specific migration")
		}
	}
}
