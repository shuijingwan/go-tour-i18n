package sitecontent

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The hash-bound Bundle owns historical slot membership and protection. Verify
// its source coordinates and structure instead of rediscovering language with
// today's protector. Current assessment/Generation validation remain separate.
func verifyFrozenStructuredRepresentation(d Document, raw []byte) error {
	if d.Contract != UnitContract {
		parsed, err := parseAuthorityStructuredRepresentation(d, raw)
		if err != nil || !reflect.DeepEqual(parsed, &d) {
			return fmt.Errorf("frozen structured representation/source mismatch")
		}
		return nil
	}
	if err := textBytes(raw); err != nil {
		return err
	}
	if digest(raw) != d.SourceSHA || d.UnitScope != "" || d.StableIndex != 0 || len(d.Slots) != 0 {
		return fmt.Errorf("frozen structured source/scope mismatch")
	}
	for _, u := range d.Units {
		seen := map[string]bool{}
		for _, p := range u.Protected {
			if p.Token == "" || seen[p.Token] {
				return fmt.Errorf("frozen structured duplicate/empty protected token")
			}
			seen[p.Token] = true
		}
	}
	if err := validateDocumentRepresentations([]Document{d}, false); err != nil {
		return err
	}
	var structure bytes.Buffer
	seen := map[string]int{}
	pos := 0
	for _, u := range d.Units {
		restored, err := restoreBlock(u, u.Source)
		if err != nil || u.End > len(raw) || restored != string(raw[u.Start:u.End]) {
			return fmt.Errorf("frozen structured slot/source coordinate mismatch")
		}
		// Parse the persisted locator framing, without inferring slot context
		// from today's parser. Occurrence counts and unit IDs stay exact.
		ordinal := strings.LastIndexByte(u.Locator, ':')
		if ordinal < 0 {
			return fmt.Errorf("frozen structured slot locator mismatch")
		}
		base := u.Locator[:ordinal]
		prefix, suffix := d.Path+"#", ":"+u.SourceSHA
		if !strings.HasPrefix(base, prefix) || !strings.HasSuffix(base, suffix) || len(base) <= len(prefix)+len(suffix) {
			return fmt.Errorf("frozen structured slot context mismatch")
		}
		seen[base]++
		if u.Locator[ordinal+1:] != strconv.Itoa(seen[base]) {
			return fmt.Errorf("frozen structured slot occurrence mismatch")
		}
		context := base[len(prefix) : len(base)-len(suffix)]
		structure.Write(raw[pos:u.Start])
		structure.WriteString("\x00" + context + "\x00" + protectionIdentity(u.Protected))
		pos = u.End
	}
	structure.Write(raw[pos:])
	if d.StructureSHA != identity(struct {
		Structure    string
		Dependencies []Dependency
	}{digest(structure.Bytes()), d.Dependencies}) {
		return fmt.Errorf("frozen structured structure identity mismatch")
	}
	return nil
}

func reconstructFrozenStructuredRepresentation(d Document, raw []byte, targets map[string]string) ([]byte, error) {
	if err := verifyFrozenStructuredRepresentation(d, raw); err != nil {
		return nil, err
	}
	if d.Contract != UnitContract {
		return reconstructAuthorityStructuredRepresentation(d, raw, targets)
	}
	if len(targets) != len(d.Units) {
		return nil, fmt.Errorf("frozen structured target exact-set mismatch")
	}
	var out bytes.Buffer
	pos := 0
	for _, u := range d.Units {
		v, ok := targets[u.ID]
		if !ok || v == "" || !utf8.ValidString(v) || strings.ContainsAny(v, "\x00\r") {
			return nil, fmt.Errorf("invalid/missing frozen structured target")
		}
		restored, err := restoreBlock(u, v)
		if err != nil {
			return nil, err
		}
		out.Write(raw[pos:u.Start])
		out.WriteString(restored)
		pos = u.End
	}
	out.Write(raw[pos:])
	// Retain the existing symmetric syntax/structure gate. Lexical slot
	// membership above comes only from the frozen Document, never this parse.
	graph, _, err := sharedHistoricalStructuredGraph(d, raw, out.Bytes())
	if err != nil {
		return nil, err
	}
	if graph.Target.StructureSHA != graph.Source.StructureSHA || len(graph.Target.Units) != len(graph.Source.Units) {
		return nil, fmt.Errorf("frozen structured protected structure/parser round-trip changed")
	}
	return out.Bytes(), nil
}
