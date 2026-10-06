package i18n

import (
	"bufio"
	"bytes"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const glossaryNormalizationVersion = "tour-glossary-normalization/v1"
const glossaryImpactAlgorithmVersion = "tour-visible-glossary-impact/v1"

type GlossarySemanticEntry struct {
	Category string `json:"category"`
	Term     string `json:"term"`
	Value    string `json:"value"`
}

type GlossarySemanticDelta struct {
	Category string  `json:"category"`
	Term     string  `json:"term"`
	Old      *string `json:"old"`
	New      *string `json:"new"`
}

// The formal loader uses a deliberately narrow line format, not general YAML.
// Preserve every literal value it consumes; reject duplicates and unsupported
// legacy terms rather than silently discard information during normalization.
func normalizeCompatibilityGlossary(locale string, data []byte) ([]GlossarySemanticEntry, error) {
	if err := ValidateLocaleName(locale); err != nil {
		return nil, err
	}
	if !utf8.Valid(data) || bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) || bytes.ContainsRune(data, '\x00') {
		return nil, fmt.Errorf("invalid glossary encoding")
	}
	entries := []GlossarySemanticEntry{}
	sections, keys := map[string]bool{}, map[string]bool{}
	section := ""
	localeSeen := false
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			key, value, ok := strings.Cut(trimmed, ":")
			key, value = strings.TrimSpace(key), strings.TrimSpace(value)
			if !ok || sections[key] {
				return nil, fmt.Errorf("ambiguous glossary section %q", trimmed)
			}
			sections[key] = true
			if key == "locale" {
				if value != locale {
					return nil, fmt.Errorf("glossary locale mismatch")
				}
				localeSeen, section = true, ""
				continue
			}
			if value != "" || (key != "mandatory" && key != "preferred" && key != "forbidden" && key != "keep") {
				return nil, fmt.Errorf("unsupported glossary normalization section %q", key)
			}
			section = key
			continue
		}
		var term, value string
		if section == "mandatory" || section == "preferred" {
			var ok bool
			term, value, ok = strings.Cut(trimmed, ":")
			if !ok {
				return nil, fmt.Errorf("invalid glossary mapping")
			}
			term, value = strings.TrimSpace(term), strings.TrimSpace(value)
		} else if section == "forbidden" || section == "keep" {
			if !strings.HasPrefix(trimmed, "- ") {
				return nil, fmt.Errorf("invalid glossary list")
			}
			term = strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			value = term
		} else {
			return nil, fmt.Errorf("glossary entry outside section")
		}
		identity := section + "\x00" + term
		if term == "" || value == "" || keys[identity] {
			return nil, fmt.Errorf("empty or duplicate glossary term %q", term)
		}
		keys[identity] = true
		entries = append(entries, GlossarySemanticEntry{section, term, value})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !localeSeen || !sections["mandatory"] {
		return nil, fmt.Errorf("glossary locale/mandatory missing")
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Category != entries[j].Category {
			return entries[i].Category < entries[j].Category
		}
		return entries[i].Term < entries[j].Term
	})
	return entries, nil
}

func glossarySemanticDelta(locale string, oldBytes, newBytes []byte) ([]GlossarySemanticDelta, error) {
	old, err := normalizeCompatibilityGlossary(locale, oldBytes)
	if err != nil {
		return nil, err
	}
	newEntries, err := normalizeCompatibilityGlossary(locale, newBytes)
	if err != nil {
		return nil, err
	}
	oldMap, newMap := map[string]GlossarySemanticEntry{}, map[string]GlossarySemanticEntry{}
	keys := map[string]bool{}
	for _, e := range old {
		key := e.Category + "\x00" + e.Term
		oldMap[key] = e
		keys[key] = true
	}
	for _, e := range newEntries {
		key := e.Category + "\x00" + e.Term
		newMap[key] = e
		keys[key] = true
	}
	ordered := []string{}
	for key := range keys {
		ordered = append(ordered, key)
	}
	sort.Strings(ordered)
	delta := []GlossarySemanticDelta{}
	for _, key := range ordered {
		a, aOK := oldMap[key]
		b, bOK := newMap[key]
		if aOK && bOK && a.Value == b.Value {
			continue
		}
		category, term, _ := strings.Cut(key, "\x00")
		d := GlossarySemanticDelta{Category: category, Term: term}
		if aOK {
			value := a.Value
			d.Old = &value
		}
		if bOK {
			value := b.Value
			d.New = &value
		}
		delta = append(delta, d)
	}
	return delta, nil
}
