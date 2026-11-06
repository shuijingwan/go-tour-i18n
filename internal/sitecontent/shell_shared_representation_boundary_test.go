package sitecontent

import (
	"bytes"
	"strings"
	"testing"
)

// This is source/transport diagnosis, not an authorized representation or
// language result. The current successor contract supports Homepage teaching
// bodies only; a shared structured asset cannot silently adopt the Page rule.
func TestShellSharedRepresentationAuthorityBoundary(t *testing.T) {
	const path = "_content/site.tmpl"
	const slotID = "0719e14f5f889a9c0dccb5eb291c9fe1de9604b6f5423b3c1497204c4276a85d"
	root := repositoryRoot(t)
	raw, err := readRegular(root, path)
	if err != nil {
		t.Fatal(err)
	}
	old, err := ExtractShellLanguageSource(path, raw)
	if err != nil {
		t.Fatal(err)
	}
	var affected *ShellLanguageSlot
	for i := range old.Slots {
		if old.Slots[i].Unit.ID == slotID {
			affected = &old.Slots[i]
		}
	}
	if affected == nil || string(raw[affected.Unit.Start:affected.Unit.End]) != "Press Enter to activate/deactivate dropdown" {
		t.Fatal("exact canonical source/slot changed")
	}
	if len(affected.Unit.Protected) != 1 || affected.Unit.Protected[0].Raw != "activate/deactivate" || affected.Unit.Protected[0].Role != "atom" || affected.Unit.Source != "Press Enter to ⟦P0⟧ dropdown" {
		t.Fatalf("frozen protected topology changed: %+v", affected.Unit)
	}
	if !legacyMachineWord("activate/deactivate", "activate/deactivate") || classifyProseToken("activate/deactivate", "Press Enter to ", " dropdown") != naturalLanguage {
		t.Fatal("legacy punctuation rule / semantic role diagnosis changed")
	}
	oldValues := map[string]string{}
	for _, s := range old.Slots {
		oldValues[s.Unit.ID] = s.Unit.Source
	}
	identity, err := reconstructShellSlots(*old, raw, oldValues)
	if err != nil || !bytes.Equal(identity, raw) {
		t.Fatal("frozen source round-trip", err)
	}
	// Literal transport fixture, never a Generation output or accepted target.
	const replacement = "按 Enter 键可激活或停用下拉菜单"
	oldValues[slotID] = replacement
	if _, err := reconstructShellSlots(*old, raw, oldValues); err == nil {
		t.Fatal("frozen atom was bypassed")
	}

	// Existing semantic protection already uses source-owned URL/path/API/code
	// authority rather than a slash spelling heuristic. Compute it in memory;
	// no canonical selector, plan or receipt adopts this prospective structure.
	next, err := extractShellLanguageSource(path, raw, true)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]string{}
	found := false
	for _, s := range next.Slots {
		values[s.Unit.ID] = s.Unit.Source
		if s.Unit.Start != affected.Unit.Start || s.Unit.End != affected.Unit.End {
			continue
		}
		found = true
		if s.Unit.Source != "Press Enter to activate/deactivate dropdown" {
			t.Fatalf("prospective prose is not editable: %+v", s.Unit)
		}
		for _, p := range s.Unit.Protected {
			if strings.Contains(p.Raw, "activate/deactivate") {
				t.Fatal("prospective natural alternatives remain frozen")
			}
		}
		values[s.Unit.ID] = replacement
	}
	if !found || next.SourceSHA != old.SourceSHA || next.Page != nil {
		t.Fatal("structured source boundary changed")
	}
	target, err := reconstructShellSlots(*next, raw, values)
	if err != nil || !bytes.Equal(target, bytes.Replace(raw, raw[affected.Unit.Start:affected.Unit.End], []byte(replacement), 1)) {
		t.Fatal("prospective exact reconstruction / Enter / HTML/template bytes", err)
	}
	if err := validateShellStructure(*next, raw, target); err != nil {
		t.Fatal("prospective source structure", err)
	}
	if _, err := reconstructStructuredRepresentation(shellDocument(*next), raw, values); err == nil || !strings.Contains(err.Error(), "structured shell representation/source mismatch") {
		t.Fatal("unadopted structured topology accepted", err)
	}
	if _, err := prospectiveShellRepresentation(path, raw, ShellTeachingRepresentation); err == nil || !strings.Contains(err.Error(), "formal Homepage representation required") {
		t.Fatal("Homepage-only successor accepted shared structured source", err)
	}
	after, err := readRegular(root, path)
	if err != nil || !bytes.Equal(raw, after) {
		t.Fatal("raw source mutated", err)
	}
	t.Log("OLD_PROTECTED_RAW=activate/deactivate OLD_PROTECTED_ROLE=atom")
	t.Log("EXPERIMENTAL_SEMANTIC_PROTECTED_RAW_CONTAINS_ACTIVATE_DEACTIVATE=no EXPERIMENTAL_RECONSTRUCTION=PASS")
	t.Log("CANONICAL_STRUCTURED_ADOPTION=REJECTED SUCCESSOR_SHARED_ASSET_SUPPORT=missing")
}

// Use machine controls already exercised by TestSemanticRoleMachineControls;
// this checks the same source-role classifier in the shared shell parser.
func TestShellSharedSemanticMachineControls(t *testing.T) {
	for _, tc := range []struct{ source, atom string }{
		{"Read ./testdata/fuzz here.", "./testdata/fuzz"},
		{"Read https://go.dev/doc here.", "https://go.dev/doc"},
		{"Use encoding/json here.", "encoding/json"},
		{"Use json.Marshal here.", "json.Marshal"},
		{"Use HTTP/2 here.", "HTTP/2"},
	} {
		t.Run(tc.atom, func(t *testing.T) {
			raw := []byte(`<a href="/fixed" aria-label="Open {{.name}} menu">` + tc.source + `</a>`)
			s, err := extractShellLanguageSource("_content/site.tmpl", raw, true)
			if err != nil {
				t.Fatal(err)
			}
			values := map[string]string{}
			var owner *ShellLanguageSlot
			var token ProtectedToken
			for i := range s.Slots {
				slot := &s.Slots[i]
				values[slot.Unit.ID] = slot.Unit.Source
				for _, p := range slot.Unit.Protected {
					if p.Raw == tc.atom {
						owner, token = slot, p
					}
				}
			}
			if owner == nil {
				t.Fatal("machine identity released")
			}
			got, err := reconstructShellSlots(*s, raw, values)
			if err != nil || !bytes.Equal(got, raw) {
				t.Fatal("machine round-trip", err)
			}
			for _, replacement := range []string{"modified", token.Token + token.Token} {
				values[owner.Unit.ID] = strings.Replace(owner.Unit.Source, token.Token, replacement, 1)
				if _, err := reconstructShellSlots(*s, raw, values); err == nil {
					t.Fatal("protected token membership drift accepted")
				}
			}
			if validateShellStructure(*s, raw, bytes.Replace(raw, []byte("/fixed"), []byte("/other"), 1)) == nil || validateShellStructure(*s, raw, bytes.Replace(raw, []byte("{{.name}}"), []byte("{{.other}}"), 1)) == nil {
				t.Fatal("HTML/template authority changed")
			}
		})
	}
}
