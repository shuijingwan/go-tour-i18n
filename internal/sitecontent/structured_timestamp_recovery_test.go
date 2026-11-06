package sitecontent

import (
	"archive/zip"
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func timestampTestZIP(t *testing.T, output []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	z := zip.NewWriter(&b)
	f, err := z.Create("structured-output.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(output); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func timestampTestRequest(t *testing.T, root string, bundle, output []byte, p Provenance) (Reference, StructuredTimestampRequest, StructuredTimestampEvidence) {
	t.Helper()
	m, _, err := readStructuredBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	p.GeneratedAt = ""
	zipBytes := timestampTestZIP(t, output)
	e := StructuredTimestampEvidence{Schema: StructuredTimestampRecoverySchema + "/generation-evidence", Status: "unavailable", Provenance: p, BundleSHA: digest(bundle), BundleIdentity: m.Identity, OutputSHA: digest(output), OutputZIPSHA: digest(zipBytes), Declaration: timestampUnavailableDeclaration}
	q := StructuredTimestampRequest{Schema: StructuredTimestampRecoverySchema + "/request", AttestedBy: "maintainer-local"}
	writeFixture(t, root, "tmp/original-bundle.zip", bundle)
	q.Bundle, _ = ReferenceFile(root, "tmp/original-bundle.zip")
	writeFixture(t, root, "tmp/original-output.json", output)
	q.Output, _ = ReferenceFile(root, "tmp/original-output.json")
	writeFixture(t, root, "tmp/original-outputs.zip", zipBytes)
	q.OutputZIP, _ = ReferenceFile(root, "tmp/original-outputs.zip")
	q.Evidence = landSharedReattestationFixture(t, root, "tmp/original-session-evidence.json", e)
	ref := landSharedReattestationFixture(t, root, "tmp/timestamp-request.json", q)
	return ref, q, e
}

func TestStructuredTimestampRecovery(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, b, o, opts, native := structuredRecoveryFixture(t)
	nativeBefore := mustSharedFixtureBytes(t, root, native.Path)
	// Preserve original lexical JSON bytes, including insignificant whitespace.
	o = append([]byte(" \n"), o...)
	ref, q, e := timestampTestRequest(t, root, b, o, opts.Provenance)
	for _, at := range []string{"", "UNKNOWN", "not-a-timestamp", "2026-10-06T01:00:00+01:00"} {
		p := opts.Provenance
		p.GeneratedAt = at
		if _, _, err := ImportStructuredAssets(root, b, o, p); err == nil || !strings.Contains(err.Error(), "invalid formal Generation provenance") {
			t.Fatal("normal import relaxed", at, err)
		}
	}
	if _, _, err := ImportStructuredAssets(root, b, o, opts.Provenance); err != nil {
		t.Fatal("normal exact timestamp regressed", err)
	}
	before, err := RecoverStructuredTimestamp(root, ref, false)
	if err != nil {
		t.Fatal(err)
	}
	if before.Applied || before.Mutation != "none" || before.ValidationPassed == 0 {
		t.Fatal("invalid readonly preflight", before)
	}
	if _, err := os.Stat(pathJoin(root, before.Authority.Path)); !os.IsNotExist(err) {
		t.Fatal("preflight published authority", err)
	}
	applied, err := RecoverStructuredTimestamp(root, ref, true)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.Mutation != "known" || applied.Generation != before.Generation {
		t.Fatal("unexpected publication", applied)
	}
	r, files, err := CheckStructuredGeneration(root, applied.Generation)
	if err != nil {
		t.Fatal("current downstream reader", err)
	}
	if r.TimestampRecovery == nil || r.Provenance.GeneratedAt != "" || r.Provenance != e.Provenance || r.CarryForward != nil || len(files) != before.ValidationPassed {
		t.Fatal("timestamp invented or current validation bypassed")
	}
	var a StructuredTimestampAuthority
	if err := readReference(root, applied.Authority, &a); err != nil {
		t.Fatal(err)
	}
	if a.Statement.Status != "unavailable" || !bytes.Equal(mustSharedFixtureBytes(t, root, a.Output.Path), o) || !bytes.Equal(mustSharedFixtureBytes(t, root, a.OutputZIP.Path), mustSharedFixtureBytes(t, root, q.OutputZIP.Path)) {
		t.Fatal("original output evidence changed")
	}
	if _, _, _, err := historicalStructuredGeneration(root, applied.Generation); err != nil {
		t.Fatal("historical downstream reader", err)
	}
	if _, _, err := BuildPackageClosure(root, "zh-CN", "site-v2-shell", nil, []Reference{applied.Generation}); err != nil {
		t.Fatal("closure reader rejected metadata lineage", err)
	}
	old, _, _, err := readStructuredGeneration(root, native)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := Encode(old)
	if !bytes.Equal(encoded, nativeBefore) || !bytes.Equal(nativeBefore, mustSharedFixtureBytes(t, root, native.Path)) {
		t.Fatal("old timestamp receipt bytes changed")
	}
	again, err := RecoverStructuredTimestamp(root, ref, true)
	if err != nil || !again.AlreadyCurrent || again.Applied || again.Mutation != "none" {
		t.Fatal("non-idempotent recovery", again, err)
	}

	for _, name := range []string{"bundle-sha", "output-sha", "output-bytes", "bundle-identity", "provider", "model", "session", "timestamp", "unavailable-status", "output-path", "slot-id", "protected-token", "zip-bytes", "finding", "source", "glossary"} {
		t.Run(name, func(t *testing.T) {
			badQ, badE := q, e
			var undo []func()
			mutate := func(path string, value []byte) Reference {
				old := mustSharedFixtureBytes(t, root, path)
				undo = append(undo, func() { writeFixture(t, root, path, old) })
				writeFixture(t, root, path, value)
				r, _ := ReferenceFile(root, path)
				return r
			}
			defer func() {
				for i := len(undo) - 1; i >= 0; i-- {
					undo[i]()
				}
			}()
			switch name {
			case "bundle-sha":
				badQ.Bundle.SHA256 = strings.Repeat("a", 64)
			case "output-sha":
				badQ.Output.SHA256 = strings.Repeat("a", 64)
			case "output-bytes":
				mutate(q.Output.Path, append(bytes.Clone(o), ' '))
			case "bundle-identity":
				badE.BundleIdentity = strings.Repeat("a", 64)
			case "provider":
				badE.Provenance.Provider = "other"
			case "model":
				badE.Provenance.Model = "other"
			case "session":
				badE.Provenance.Session = "synthetic-independent-reviewer"
			case "timestamp":
				badE.Provenance.GeneratedAt = "2026-11-01T00:00:00Z"
			case "unavailable-status":
				badE.Status = "estimated"
			case "zip-bytes":
				badQ.OutputZIP = mutate(q.OutputZIP.Path, timestampTestZIP(t, []byte("{}\n")))
				badE.OutputZIPSHA = badQ.OutputZIP.SHA256
			case "finding":
				bad := rewriteRecoveryBundle(t, b, func(m *StructuredBundle, fs map[string][]byte) {
					m.RevisionFinding = &Reference{"data/unknown-surface.json", strings.Repeat("a", 64)}
				})
				badQ.Bundle = mutate(q.Bundle.Path, bad)
				badE.BundleSHA = badQ.Bundle.SHA256
			case "source":
				mutate("data/site-shell/home.html", []byte("<h1>changed source</h1>\n"))
			case "glossary":
				mutate("locales/zh-CN/glossary.yaml", append(mustSharedFixtureBytes(t, root, "locales/zh-CN/glossary.yaml"), []byte("keep:\n  - Go\n")...))
			default:
				var out StructuredOutput
				_ = StrictJSON(o, &out)
				if name == "output-path" {
					out.Outputs[0].Path = "locales/zh-CN/structured/unrelated.html"
				} else {
					changed := false
					for i := range out.Outputs {
						for j := range out.Outputs[i].Slots {
							if !changed {
								if name == "slot-id" {
									out.Outputs[i].Slots[j].ID = "forged-slot"
								} else {
									out.Outputs[i].Slots[j].Text += " ⟦P999⟧"
								}
								changed = true
							}
						}
					}
					if !changed {
						t.Fatal("fixture lacks structured slots")
					}
				}
				bad, _ := Encode(out)
				badQ.Output = mutate(q.Output.Path, bad)
				badE.OutputSHA = badQ.Output.SHA256
				badQ.OutputZIP = mutate(q.OutputZIP.Path, timestampTestZIP(t, bad))
				badE.OutputZIPSHA = badQ.OutputZIP.SHA256
			}
			eb, _ := Encode(badE)
			badQ.Evidence = mutate(q.Evidence.Path, eb)
			qb, _ := Encode(badQ)
			request := mutate(ref.Path, qb)
			if _, err := RecoverStructuredTimestamp(root, request, false); err == nil {
				t.Fatal("forged metadata/input accepted")
			}
		})
	}
	for _, name := range []string{"missing-authority", "invented-time", "authority-tamper", "evidence-tamper", "mixed-carry", "provider-substitution", "session-substitution"} {
		t.Run(name, func(t *testing.T) {
			copy := *r
			switch name {
			case "missing-authority":
				copy.TimestampRecovery = nil
			case "invented-time":
				copy.Provenance.GeneratedAt = "2026-11-01T00:00:00Z"
			case "mixed-carry":
				copy.CarryForward = &StructuredCarryForward{}
			case "provider-substitution":
				copy.Provenance.Provider = "chatgpt"
				if r.Provenance.Provider == "chatgpt" {
					copy.Provenance.Provider = "codex"
				}
			case "session-substitution":
				copy.Provenance.Session = "other-generation-session"
			default:
				path := applied.Authority.Path
				if name == "evidence-tamper" {
					path = a.Evidence.Path
				}
				saved := mustSharedFixtureBytes(t, root, path)
				defer writeFixture(t, root, path, saved)
				writeFixture(t, root, path, append(saved, ' '))
			}
			if _, err := validateStructuredProvenance(root, &copy); err == nil {
				t.Fatal("historical recovery provenance forged")
			}
		})
	}
	if !reflect.DeepEqual(r.Plan.Expected, old.Plan.Expected) {
		t.Fatal("metadata recovery broadened output scope")
	}
	t.Log("NORMAL_IMPORT_STRICT=yes TIMESTAMP_UNAVAILABLE=yes ORIGINAL_BYTES_EXACT=yes HISTORICAL_READER=PASS CLOSURE_READER=PASS")
}

func TestStructuredTimestampRecoverySharedRevision(t *testing.T) {
	done := contentidentity.BeginMemoization()
	defer done()
	root, closure, generation, _ := sharedRepresentationFixtureWithHomepage(t, true)
	scope, err := BuildIntegratedSurfaceScope(root, sharedFixtureLocale, []Reference{closure})
	if err != nil {
		t.Fatal(err)
	}
	md := []byte("# Synthetic independent structured finding\nFrozen natural language atom.\n")
	writeFixture(t, root, "data/timestamp-shared-finding.md", md)
	r := IntegratedSurfaceReceipt{Schema: IntegratedSurfaceSchema, Scope: *scope, ReviewID: "timestamp-shared-finding", Reviewer: "synthetic-independent-reviewer", Decision: "failed", Findings: []string{"frozen atom"}, ExactFindings: []SurfaceFinding{{ID: "_content/site.tmpl", Class: "structured-surface-language", Finding: "frozen atom"}}, Evidence: Reference{"data/timestamp-shared-finding.md", digest(md)}}
	rb, _ := Encode(r)
	_, rp, err := RecordIntegratedSurface(root, rb)
	if err != nil {
		t.Fatal(err)
	}
	finding, _ := ReferenceFile(root, rp)
	// Everything below is a disposable repo. No real authority is published.
	raw := mustSharedFixtureBytes(t, root, "_content/site.tmpl")
	old, err := ExtractShellLanguageSource("_content/site.tmpl", raw)
	if err != nil {
		t.Fatal(err)
	}
	q := SharedRepresentationRequest{Schema: SharedRepresentationSchema + "/request", Package: "site-v2-shell", AssetID: "source:_content/site.tmpl", Path: "_content/site.tmpl", SourceSHA: digest(raw), PreviousIdentity: identity(*old), Generation: generation, Rule: SharedRepresentationRule, Slots: []string{"0719e14f5f889a9c0dccb5eb291c9fe1de9604b6f5423b3c1497204c4276a85d"}}
	request := landSharedReattestationFixture(t, root, "data/timestamp-shared-request.json", q)
	p, err := PlanSharedStructuredRepresentation(root, request)
	if err != nil {
		t.Fatal(err)
	}
	plan := landSharedReattestationFixture(t, root, "data/timestamp-shared-plan.json", p)
	if _, err := ApplySharedStructuredRepresentation(root, plan, p.PriorIdentity, p.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	parser := sharedRepresentationParserPaths[0]
	code := mustSharedFixtureBytes(t, root, parser)
	writeFixture(t, root, parser, append(bytes.Clone(code), []byte("\n// synthetic dependency audit epoch\n")...))
	rea, err := PlanSharedRepresentationReattestation(root, p.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	reviewed := landSharedReattestationFixture(t, root, "data/timestamp-dependency-plan.json", rea)
	if _, err := InspectSharedRepresentationReattestation(root, reviewed); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplySharedRepresentationReattestation(root, reviewed, rea.PriorPointerSHA); err != nil {
		t.Fatal(err)
	}
	bundle, m, err := ExportStructuredAssetsRevision(root, sharedFixtureLocale, finding)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"locales/" + sharedFixtureLocale + "/structured/_content/site.tmpl"}
	if !reflect.DeepEqual(m.Plan.Expected, want) {
		t.Fatal(m.Plan.Expected)
	}
	o := StructuredOutput{Schema: StructuredSchema, BundleSHA: m.Identity, Outputs: []StructuredTarget{{Path: want[0], Slots: []Target{}}}}
	translationFound := false
	for _, a := range m.Plan.Assets {
		if a.Document != nil && a.Document.Path == "data/site-shell/translation.html" {
			translationFound = true
			if a.State != "context-only" {
				t.Fatal("unrelated source work entered revision")
			}
		}
		if a.TargetPath == want[0] {
			for _, u := range a.Document.Units {
				o.Outputs[0].Slots = append(o.Outputs[0].Slots, Target{u.ID, u.Source})
			}
		}
	}
	if !translationFound {
		t.Fatal("missing independent source context")
	}
	out, _ := Encode(o)
	ref, loss, _ := timestampTestRequest(t, root, bundle, out, Provenance{Provider: "chatgpt", Model: "gpt-5.6-sol-high", Session: "synthetic-generation"})
	// The model consumed A. Only machine dependencies evolve afterward. The
	// ordinary checker and generic recovery must continue rejecting old context.
	advance := func(epoch string) *SharedRepresentationReattestationPlan {
		t.Helper()
		writeFixture(t, root, parser, append(mustSharedFixtureBytes(t, root, parser), []byte("\n// dependency epoch "+epoch+"\n")...))
		next, err := PlanSharedRepresentationReattestation(root, p.Receipt)
		if err != nil {
			t.Fatal(err)
		}
		r := landSharedReattestationFixture(t, root, "data/timestamp-epoch-"+epoch+".json", next)
		if _, err := InspectSharedRepresentationReattestation(root, r); err != nil {
			t.Fatal(err)
		}
		if _, err := ApplySharedRepresentationReattestation(root, r, next.PriorPointerSHA); err != nil {
			t.Fatal(err)
		}
		return next
	}
	bEpoch := advance("B")
	if bEpoch.Authority.Previous == nil || *bEpoch.Authority.Previous != rea.Receipt {
		t.Fatal("B does not extend exact A")
	}
	if _, err := CheckStructuredAssetsBundle(root, bundle); err == nil {
		t.Fatal("normal current checker accepted stale A")
	}
	if _, err := RecoverStructuredAssets(root, bundle, out, StructuredRecoveryOptions{BundleSHA: digest(bundle), OutputSHA: digest(out), Provenance: Provenance{Provider: "chatgpt", Model: "gpt-5.6-sol-high", Session: "synthetic-generation", GeneratedAt: "2026-10-06T00:00:00Z"}}); err == nil {
		t.Fatal("generic recovery acquired re-attestation exception")
	}
	if _, err := RecoverStructuredTimestamp(root, ref, false); err != nil {
		t.Fatal("exact ancestor preflight", err)
	}
	localeBefore := mustSharedFixtureBytes(t, root, LocalePath(sharedFixtureLocale))
	result, err := RecoverStructuredTimestamp(root, ref, true)
	if err != nil {
		t.Fatal("exact shared-representation revision metadata recovery", err)
	}
	got, _, err := CheckStructuredGeneration(root, result.Generation)
	if err != nil {
		t.Fatal(err)
	}
	if got.Provenance.GeneratedAt != "" || !reflect.DeepEqual(got.Plan.Expected, want) || !bytes.Equal(localeBefore, mustSharedFixtureBytes(t, root, LocalePath(sharedFixtureLocale))) {
		t.Fatal("metadata recovery changed language/completion state")
	}
	var recovered StructuredTimestampAuthority
	if err := readReference(root, result.Authority, &recovered); err != nil {
		t.Fatal(err)
	}
	if recovered.Bridge == nil || len(recovered.Bridge.Reattestations) != 1 || recovered.Bridge.Reattestations[0].Old != rea.Receipt || recovered.Bridge.Reattestations[0].Current != bEpoch.Receipt || got.Output.BundleSHA != m.Identity || !bytes.Equal(out, mustSharedFixtureBytes(t, root, recovered.Output.Path)) {
		t.Fatal("original input/output identity or ancestry changed")
	}
	bBytes := mustSharedFixtureBytes(t, root, recovered.Bridge.CurrentBundle.Path)
	bManifest, bFiles, err := readStructuredBundle(bBytes)
	if err != nil {
		t.Fatal(err)
	}
	cEpoch := advance("C")
	preflightC, err := RecoverStructuredTimestamp(root, ref, false)
	if err != nil || preflightC.Authority == result.Authority {
		t.Fatal("multi-descendant currentness proof", err)
	}
	if _, _, _, err := historicalStructuredGeneration(root, result.Generation); err != nil {
		t.Fatal("frozen bridge changed with live head", err)
	}
	if _, _, err := CheckStructuredGeneration(root, result.Generation); err != nil {
		t.Fatal("current descendant reader", err)
	}
	currentBytes, current, err := ExportStructuredAssetsRevision(root, sharedFixtureLocale, finding)
	if err != nil {
		t.Fatal(err)
	}
	_, currentFiles, err := readStructuredBundle(currentBytes)
	if err != nil {
		t.Fatal(err)
	}
	oldManifest, oldFiles, err := readStructuredBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"source", "document", "slot-id", "protected-token", "finding", "expected", "glossary", "review", "semantic-0", "semantic-1", "semantic-2", "unknown-context"} {
		t.Run("lineage-semantic-"+name, func(t *testing.T) {
			bad := rewriteRecoveryBundle(t, bundle, func(m *StructuredBundle, fs map[string][]byte) {
				switch name {
				case "source":
					fs["source/_content/site.tmpl"] = append(bytes.Clone(fs["source/_content/site.tmpl"]), ' ')
				case "finding":
					m.RevisionFinding = &Reference{finding.Path, strings.Repeat("a", 64)}
				case "expected":
					m.Plan.Expected = append(m.Plan.Expected, "unrelated.html")
				case "glossary":
					fs["inputs/locales/"+sharedFixtureLocale+"/glossary.yaml"] = append(bytes.Clone(fs["inputs/locales/"+sharedFixtureLocale+"/glossary.yaml"]), ' ')
				case "review":
					m.Plan.Review.SHA256 = strings.Repeat("a", 64)
				case "semantic-0", "semantic-1", "semantic-2":
					i := int(name[len(name)-1] - '0')
					path := "inputs/" + StructuredSemanticAuthorities[i]
					fs[path] = append(bytes.Clone(fs[path]), ' ')
				case "unknown-context":
					m.Files = append(m.Files, i18n.NewTransportBundleFile("inputs/arbitrary-context.json", "arbitrary-context.json", []byte("{}\n")))
					fs["inputs/arbitrary-context.json"] = []byte("{}\n")
				default:
					for i := range m.Plan.Assets {
						a := &m.Plan.Assets[i]
						if a.ID == q.AssetID {
							switch name {
							case "document":
								a.Document.SourceSHA = strings.Repeat("a", 64)
							case "slot-id":
								a.Document.Units[0].ID = "changed"
							case "protected-token":
								a.Document.Units[0].Protected = append(a.Document.Units[0].Protected, ProtectedToken{Token: "⟦P999⟧", Raw: "forged", Role: "atom"})
							}
						}
					}
				}
			})
			bm, bf, err := readStructuredBundle(bad)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := compatibleTimestampLineage(root, *bm, bf, *current, currentFiles, true); err == nil {
				t.Fatal("semantic drift accepted")
			}
		})
	}
	t.Run("wrong-sibling", func(t *testing.T) {
		// D and B both extend A, with D fresh for today's dependency bytes.
		d := cEpoch.Authority
		d.Previous = &rea.Receipt
		d.Identity = ""
		d.Identity = identity(d)
		dRef := sharedReattestationReceipt(&d)
		landSharedReattestationFixture(t, root, dRef.Path, d)
		head := mustSharedFixtureBytes(t, root, cEpoch.CurrentPath)
		defer writeFixture(t, root, cEpoch.CurrentPath, head)
		landSharedReattestationFixture(t, root, cEpoch.CurrentPath, dRef)
		db, dm, err := ExportStructuredAssetsRevision(root, sharedFixtureLocale, finding)
		if err != nil {
			t.Fatal("sibling must otherwise be family-current", err)
		}
		_, df, err := readStructuredBundle(db)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := compatibleTimestampLineage(root, *bManifest, bFiles, *dm, df, true); err == nil {
			t.Fatal("sibling accepted as ancestor")
		}
	})
	t.Run("different-base", func(t *testing.T) {
		bad := cEpoch.Authority
		bad.BaseReceipt.SHA256 = strings.Repeat("a", 64)
		path := cEpoch.Receipt.Path
		before := mustSharedFixtureBytes(t, root, path)
		defer writeFixture(t, root, path, before)
		encoded, _ := Encode(bad)
		writeFixture(t, root, path, encoded)
		if _, _, err := compatibleTimestampLineage(root, *oldManifest, oldFiles, *current, currentFiles, true); err == nil {
			t.Fatal("different base accepted")
		}
	})
	for _, path := range []string{result.Authority.Path, cEpoch.Receipt.Path, cEpoch.CurrentPath} {
		t.Run("tamper-"+path, func(t *testing.T) {
			before := mustSharedFixtureBytes(t, root, path)
			defer writeFixture(t, root, path, before)
			value := append(bytes.Clone(before), ' ')
			if path == cEpoch.CurrentPath {
				value = []byte("{}\n")
			}
			writeFixture(t, root, path, value)
			if _, _, err := CheckStructuredGeneration(root, result.Generation); err == nil {
				t.Fatal("tampered authority/head accepted")
			}
		})
	}
	t.Log("ANCESTOR_FIXTURE=PASS MULTI_DESCENDANT_FIXTURE=PASS HISTORICAL_READER=PASS CURRENT_DOWNSTREAM_READER=PASS")
	for _, name := range []string{"representation-head", "dependency-authority", "finding", "unrelated-output"} {
		t.Run(name, func(t *testing.T) {
			path := p.CurrentPath
			value := []byte("{}\n")
			switch name {
			case "dependency-authority":
				path = rea.CurrentPath
			case "finding":
				path = rp
			case "unrelated-output":
				for _, path := range []string{loss.Bundle.Path, loss.Output.Path, loss.OutputZIP.Path, loss.Evidence.Path, ref.Path} {
					path := path
					saved := mustSharedFixtureBytes(t, root, path)
					defer writeFixture(t, root, path, saved)
				}
				bad := o
				bad.Outputs = append(append([]StructuredTarget{}, o.Outputs...), StructuredTarget{Path: "locales/" + sharedFixtureLocale + "/structured/data/site-shell/translation.html", Slots: []Target{}})
				data, _ := Encode(bad)
				request, _, _ := timestampTestRequest(t, root, bundle, data, got.Provenance)
				if _, err := RecoverStructuredTimestamp(root, request, false); err == nil {
					t.Fatal("unrelated output accepted")
				}
				return
			}
			before := mustSharedFixtureBytes(t, root, path)
			defer writeFixture(t, root, path, before)
			writeFixture(t, root, path, value)
			if _, err := RecoverStructuredTimestamp(root, ref, false); err == nil {
				t.Fatal("missing current authority accepted")
			}
		})
	}
	// The receipt archives original output bytes, not a normalized replacement.
	if digest(mustSharedFixtureBytes(t, root, loss.Output.Path)) != loss.Output.SHA256 {
		t.Fatal("original output changed")
	}
	t.Log("SHARED_REVISION_RECOVERY=PASS EXPECTED_COUNT=1 TRANSLATION_HTML_CONTEXT_ONLY=yes OLD_A_CARRIED=no")
}
