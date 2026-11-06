package i18n

import (
	"strconv"
	"strings"
)

// BareDocumentationCommand is a finite command grammar for the actual frozen
// documentation, not a growing executable-name heuristic or a shell parser.
// Unsupported operators/comments/quoting stay unknown. The resulting command
// records are protected in full and are never executed by this package.
func BareDocumentationCommand(code string) ([]TranscriptRecord, bool) {
	records := []TranscriptRecord{}
	pos := 0
	seen := false
	for _, line := range strings.SplitAfter(code, "\n") {
		s := strings.TrimSpace(line)
		if s == "" {
			if len(line) > 0 {
				records = append(records, TranscriptRecord{Kind: "output", Start: pos, End: pos + len(line), Text: line})
			}
			pos += len(line)
			continue
		}
		if strings.HasPrefix(s, "export ") {
			rest := strings.TrimPrefix(s, "export ")
			_, value, ok := strings.Cut(rest, "=")
			if ok && strings.HasPrefix(value, "\"") && commandAssignment(rest) {
				_, err := strconv.Unquote(value)
				complete, valid := commandEnvelope(s, false)
				if err != nil || !complete || !valid {
					return nil, false
				}
				records = append(records, TranscriptRecord{Kind: "command", Start: pos, End: pos + len(line), Text: line})
				pos += len(line)
				seen = true
				continue
			}
		}
		if strings.ContainsAny(s, "#'\"`;|&(){}\\\x00\r") {
			return nil, false
		}
		words := strings.Fields(s)
		ok := false
		switch words[0] {
		case "cd":
			ok = len(words) <= 2
		case "mkdir":
			ok = len(words) == 2
		case "export":
			ok = len(words) == 2 && commandAssignment(words[1])
		case "go":
			if len(words) >= 2 {
				switch words[1] {
				case "install", "get", "vet":
					ok = len(words) >= 3
				case "mod":
					ok = len(words) >= 4 && (words[2] == "init" || words[2] == "edit")
				}
			}
		default:
			ok = len(words) == 1 && commandAssignment(words[0])
		}
		if !ok {
			return nil, false
		}
		for _, w := range words {
			for _, r := range w {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_./-@=:%*,$<>", r)) {
					return nil, false
				}
			}
		}
		records = append(records, TranscriptRecord{Kind: "command", Start: pos, End: pos + len(line), Text: line})
		pos += len(line)
		seen = true
	}
	return records, seen
}

func commandAssignment(s string) bool {
	key, value, ok := strings.Cut(s, "=")
	if !ok || key == "" || value == "" {
		return false
	}
	for i, r := range key {
		if !(r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' || i > 0 && r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
