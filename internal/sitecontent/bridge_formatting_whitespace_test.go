package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func formattingBridgePage(t *testing.T, source string) PageAdapter {
	t.Helper()
	b := []byte(source)
	d, err := ParseWorkflowDocument(Source{Path: "_content/doc/formatting.html", Kind: "page", Route: "/doc/formatting", SHA256: digest(b)}, b, 1)
	if err != nil {
		t.Fatal(err)
	}
	return PageAdapter{*d, b}
}

func assertFormattingBridge(t *testing.T, a PageAdapter, raw, reviewed []byte) {
	t.Helper()
	before, _ := json.Marshal(a)
	decoder, template, err := a.BridgeResponseFormattingWhitespace(raw)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decoder.Restore(string(template))
	if err != nil || !bytes.Equal(got, reviewed) {
		t.Fatal("baseline must restore byte-exact reviewed candidate", err)
	}
	formatting := []byte("\n        ")
	if !bytes.Contains(template, formatting) {
		t.Fatal("source-owned formatting remains opaque")
	}
	edited := bytes.Replace(template, formatting, nil, 1)
	got, err = decoder.Restore(string(edited))
	if err != nil || !bytes.Equal(got, bytes.Replace(reviewed, formatting, nil, 1)) {
		t.Fatal("Generation-style formatting deletion failed", err)
	}
	for _, line := range bytes.Split(got, []byte("\n")) {
		if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
			t.Fatal("restored candidate retains trailing whitespace")
		}
	}
	// A formatting sequence cannot be invented, duplicated, or moved into a
	// different Page slot. Normal HTML/protected binding checks remain active.
	for _, bad := range [][]byte{bytes.Replace(template, formatting, []byte("\n    "), 1), bytes.Replace(template, formatting, append(bytes.Clone(formatting), formatting...), 1)} {
		if _, err := decoder.Restore(string(bad)); err == nil {
			t.Fatal("unproved formatting boundary accepted")
		}
	}
	private := decoder.(PageAdapter)
	for _, p := range private.Document.Units[0].Protected {
		if p.Role == "boundary" && strings.Contains(p.Raw, "</li>") {
			crossed := bytes.Replace(edited, []byte(p.Token), append([]byte(p.Token), formatting...), 1)
			if _, err := decoder.Restore(string(crossed)); err == nil {
				t.Fatal("formatting crossed Page slot ownership")
			}
			break
		}
	}
	_, tokens, err := decoder.ProtectedInput(nil)
	if err != nil || len(tokens) == 0 {
		t.Fatal("protected identities lost", err)
	}
	for _, token := range tokens {
		for _, bad := range []string{strings.Replace(string(template), token, "", 1), string(template) + token, strings.Replace(string(template), token, "changed", 1)} {
			if _, err := decoder.Restore(bad); err == nil {
				t.Fatal("protected exact/unique check weakened", token)
			}
		}
	}
	after, _ := json.Marshal(a)
	if !bytes.Equal(before, after) {
		t.Fatal("caller formal Page/source identity mutated")
	}
}

func TestBridgeReleasesProseFormattingWhitespace(t *testing.T) {
	a := formattingBridgePage(t, "<ul>\n<li>Use <code>Cmd</code> for reporting\n        and logging from your test.\n</li>\n<li>Keep <a href=\"https://example.test\">the manual</a> and <code>Msg</code>.</li>\n</ul>\n")
	raw := a.Document.Units[0].Source
	raw = strings.Replace(raw, "for reporting", "用于报告和记录日志。", 1)
	raw = strings.Replace(raw, "and logging from your test.", "", 1)
	reviewed, err := a.Restore(raw)
	if err != nil || !bytes.Contains(reviewed, []byte("。\n        \n</li>")) {
		t.Fatal("fixture must reproduce formatting stranded after merged prose", err)
	}
	// Canonical Page syntax still rejects unprotected LF; the exception belongs
	// only to the new private response with source-proved exposed formatting.
	var whitespaceToken string
	for _, p := range a.Document.Units[0].Protected {
		if p.Raw == "\n        " {
			whitespaceToken = p.Token
		}
	}
	if whitespaceToken == "" {
		t.Fatal("fixture has no historical formatting atom")
	}
	if _, err := a.Restore(strings.Replace(raw, whitespaceToken, "\n        ", 1)); err == nil {
		t.Fatal("canonical representation was relaxed")
	}
	assertFormattingBridge(t, a, []byte(raw), reviewed)
}

func TestRestrictedBridgeReleasesFormattingWhitespace(t *testing.T) {
	a := formattingBridgePage(t, "<p>Prose before\n        prose after.\n</p>\n<pre>\n// Explain the return value.\nfunc f() string { return \"literal\" }\n</pre>\n")
	raw := strings.Replace(a.Document.Units[0].Source, "Prose before", "合并后的正文。", 1)
	raw = strings.Replace(raw, "prose after.", "", 1)
	old, err := a.Restore(raw)
	if err != nil {
		t.Fatal(err)
	}
	bodies, err := a.CommentRevisionBodies(old)
	if err != nil || len(bodies) != 1 {
		t.Fatal("source-owned teaching fixture", err)
	}
	b := bodies[0]
	reviewed := append(append(append([]byte{}, old[:b.Start]...), []byte("说明返回内容。")...), old[b.End:]...)
	p := &i18n.CommentRevisionTransport{Schema: "go-learning/comment-revision-transport/v1", UnitID: a.Document.Units[0].ID, SourceSHA256: digest(a.Source), Slots: []i18n.CommentRevisionTransportSlot{{Start: b.Start, End: b.End}}, Boundaries: []string{"left-boundary", "right-boundary"}}
	decoder, template, err := a.BridgeRestrictedRevision([]byte(raw), old, reviewed, []byte("left-boundary说明返回内容。right-boundary"), p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := decoder.Restore(string(template))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(reviewed)) {
		t.Fatal("restricted baseline mismatch", err)
	}
	got, err = decoder.Restore(string(bytes.Replace(template, []byte("\n        "), nil, 1)))
	if err != nil || !bytes.Equal(bridgeEOF(got), bridgeEOF(bytes.Replace(reviewed, []byte("\n        "), nil, 1))) {
		t.Fatal("restricted formatting edit failed", err)
	}
}

func TestBridgeFormattingWhitespaceSafety(t *testing.T) {
	for _, source := range []string{
		"<p>Keep <code>code\n        identity</code> unchanged.</p>\n",
		"<p>Keep prose.</p>\n<pre>code\n        identity</pre>\n",
		"<p>Keep {{printf `prose\n        value`}} unchanged.</p>\n",
		"<p title=\"prose before\n        prose after\">Keep prose.</p>\n",
		"<p>Keep <a href=\"https://example.test\n        /path\">the manual</a>.</p>\n",
		"<p>Keep prose.</p>\n        <p>Keep next prose.</p>\n",
		"<p style=\"white-space:pre\">Prose before\n        prose after.</p>\n",
	} {
		t.Run(source, func(t *testing.T) {
			a := formattingBridgePage(t, source)
			raw := []byte(a.Document.Units[0].Source)
			next, projected, err := a.bridgeReleaseNaturalAtoms(a, raw)
			if err == nil && (!bytes.Equal(projected, raw) || !reflectFormattingDocumentEqual(a.Document, next.Document)) {
				t.Fatal("non-prose formatting protection released")
			}
		})
	}
	a := formattingBridgePage(t, "<p>Prose before\n        prose after.</p>\n")
	for _, kind := range []string{"bound", "pair", "duplicate-owner", "cross-slot"} {
		t.Run(kind, func(t *testing.T) {
			b, _ := json.Marshal(a)
			var ambiguous PageAdapter
			json.Unmarshal(b, &ambiguous)
			s := &ambiguous.Document.Slots[0]
			switch kind {
			case "bound":
				s.Protected[0].Binding = &StructuralBinding{Version: 2, Kind: "markdown-title"}
			case "pair":
				s.Protected[0].Pair = "technical-binding"
			case "duplicate-owner":
				ambiguous.Document.Slots = append(ambiguous.Document.Slots, *s)
			case "cross-slot":
				s.End = s.Start + strings.Index(string(a.Source[s.Start:s.End]), "\n") + 1
			}
			next, projected, err := ambiguous.bridgeReleaseNaturalAtoms(ambiguous, []byte(a.Document.Units[0].Source))
			if err == nil && !bytes.Equal(projected, []byte(a.Document.Units[0].Source)) {
				t.Fatal("ambiguous or bound formatting released", next.Document)
			}
		})
	}
}

func reflectFormattingDocumentEqual(a, b Document) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return bytes.Equal(x, y)
}

// Read only already sealed codec/raw/candidate bytes, with no catalog or
// currentness reconstruction and no lifecycle operation or generated output.
func Test393BridgeFormattingWhitespaceReadonly(t *testing.T) {
	if os.Getenv("GO_TOUR_393_FORMATTING_READONLY") != "1" {
		t.Skip("explicit readonly regression against existing 393 evidence")
	}
	root := repositoryRoot(t)
	const id = "393fb67915a49876749590fd6480c9a3482597a9cebdd61e72c9a5f3d1449e2c"
	read := func(path string) []byte { return mustReadFixture(t, root, path) }
	sealed := func(ref i18n.GlossaryArchiveReference) []byte {
		t.Helper()
		b := read(ref.Path)
		if digest(b) != ref.SHA256 {
			t.Fatal("frozen reference SHA mismatch", ref.Path)
		}
		return b
	}
	var receipt i18n.QualityCheckSurfaceReopen
	if err := json.Unmarshal(read("data/quality-check-reopens/zh-CN/learn-docs-qc024-restricted-bridge-004.json"), &receipt); err != nil {
		t.Fatal(err)
	}
	for _, binding := range receipt.SameUnitBridge.Bindings {
		if binding.UnitID != id {
			continue
		}
		files, err := i18n.ReadTransportBundle(sealed(binding.Archive), i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
		if err != nil || digest(files[binding.TransportMember]) != binding.TransportSHA256 {
			t.Fatal("frozen cvpred archive/member", err)
		}
		var predecessor i18n.CurrentValidationPredecessor
		if err := json.Unmarshal(files[binding.TransportMember], &predecessor); err != nil {
			t.Fatal(err)
		}
		base := "data/retranslation-runs/zh-CN/" + predecessor.Snapshot.SelectedBatchID
		var imported i18n.GenerationResultBundleManifest
		if err := json.Unmarshal(read(base+"/generation-imports/initial-manifest.json"), &imported); err != nil {
			t.Fatal(err)
		}
		archive := sealed(i18n.GlossaryArchiveReference{Path: "data/generation-archives/zh-CN/" + imported.GenerationBundleSHA256 + ".zip", SHA256: imported.GenerationBundleSHA256})
		members, err := i18n.ReadTransportBundle(archive, i18n.TranslationUnitGenerationBundleMaxFiles, i18n.TranslationUnitGenerationBundleMaxBytes)
		if err != nil {
			t.Fatal(err)
		}
		var transport i18n.SurfaceTransportProjection
		if err := json.Unmarshal(members["formal/surface-transport/"+id+".json"], &transport); err != nil {
			t.Fatal(err)
		}
		var codec surfacePageTransport
		if err := json.Unmarshal(transport.Codec, &codec); err != nil {
			t.Fatal(err)
		}
		origin, _, err := codec.ordinaryResponse(string(sealed(predecessor.Raw)))
		if err != nil {
			t.Fatal(err)
		}
		raw := sealed(binding.Raw)
		var reviewed []byte
		for _, q := range receipt.Units {
			if q.UnitID == id {
				reviewed = sealed(i18n.GlossaryArchiveReference{Path: q.Snapshot.CandidatePath, SHA256: q.Snapshot.CandidateSHA256})
			}
		}
		baseline, err := origin.Restore(string(raw))
		if err != nil || !bytes.Equal(bridgeEOF(baseline), reviewed) {
			t.Fatal("sealed ordinary decoder must reproduce reviewed 393", err)
		}
		decoder, template, err := origin.BridgeResponseFormattingWhitespace(raw)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decoder.Restore(string(template))
		if err != nil || !bytes.Equal(bridgeEOF(got), reviewed) {
			t.Fatal("393 private baseline changed", err)
		}
		before, _ := protectionPieces(origin.Document.Units[0])
		after, _ := protectionPieces(decoder.(PageAdapter).Document.Units[0])
		for _, p := range before {
			if p.token == nil || p.token.Token != "⟦P52⟧" || p.token.Raw != "\n        " {
				continue
			}
			for _, q := range after {
				if q.token == nil && q.start <= p.start && q.end >= p.end {
					if !bytes.Contains(template, []byte("\n        ")) {
						t.Fatal("393 formatting was not exposed in response")
					}
					// Locate the response slot using source-owned boundary tokens,
					// never translated word search or inferred target coordinates.
					next := decoder.(PageAdapter)
					boundaries := []ProtectedToken{}
					for _, token := range next.Document.Units[0].Protected {
						if token.Role == "boundary" {
							boundaries = append(boundaries, token)
						}
					}
					for i, slot := range next.Document.Slots {
						if slot.Start <= p.start && slot.End >= p.end {
							start := bytes.Index(template, []byte(boundaries[i].Token)) + len(boundaries[i].Token)
							end := bytes.Index(template, []byte(boundaries[i+1].Token))
							segment := template[start:end]
							oldBoundaries := []ProtectedToken{}
							for _, token := range origin.Document.Units[0].Protected {
								if token.Role == "boundary" {
									oldBoundaries = append(oldBoundaries, token)
								}
							}
							oldStart := bytes.Index(raw, []byte(oldBoundaries[i].Token)) + len(oldBoundaries[i].Token)
							oldEnd := bytes.Index(raw, []byte(oldBoundaries[i+1].Token))
							if !bytes.HasSuffix(raw[oldStart:oldEnd], []byte(p.token.Token)) || !bytes.HasSuffix(segment, []byte(p.token.Raw)) {
								t.Fatal("393 formatting source/response boundary correspondence changed")
							}
							edited := append(append(append([]byte{}, template[:start]...), segment[:len(segment)-len(p.token.Raw)]...), template[end:]...)
							got, err := decoder.Restore(string(edited))
							want := bytes.Replace(reviewed, []byte("\n        \n"), []byte("\n"), 1)
							if err != nil || !bytes.Equal(bridgeEOF(got), want) || bytes.Equal(want, reviewed) {
								t.Fatal("393 formatting-only deletion failed", err)
							}
							return
						}
					}
				}
			}
		}
		t.Fatal("393 source-owned P52 formatting remains protected")
	}
	t.Fatal("missing sealed 393 bridge binding")
}
