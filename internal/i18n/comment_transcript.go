package i18n

import (
	"strings"
	"unicode/utf8"
)

// TranscriptRecord preserves the original order and bytes. Commands are opaque
// machine input, never shell source inferred from output. This projection does
// not claim to parse general shell, cmd.exe or SQL syntax.
type TranscriptRecord struct {
	Kind   string `json:"kind"`
	Prompt string `json:"prompt,omitempty"`
	Start  int    `json:"start"`
	End    int    `json:"end"`
	Text   string `json:"text"`
}

func transcriptPrompt(line string) (string, string, bool) {
	for _, p := range []string{"$ ", "~ ", "mysql>"} {
		if strings.HasPrefix(line, p) {
			body := strings.TrimPrefix(line, p)
			if p == "mysql>" && body != "" && !strings.HasPrefix(body, " ") {
				return "", "", false
			}
			return p, strings.TrimLeft(body, " "), true
		}
	}
	if len(line) >= 4 && line[0] >= 'A' && line[0] <= 'Z' && line[1:3] == ":\\" {
		if n := strings.Index(line, "> "); n >= 3 {
			path := line[3:n]
			if !strings.ContainsAny(path, "<>|\"\t\r\n") {
				return line[:n+2], line[n+2:], true
			}
		}
	}
	return "", "", false
}

// commandEnvelope is a deliberately bounded lexical envelope, not a shell
// parser. Comment-capable or heredoc input outside quotes is refused rather
// than silently treated as comment-free. Word-internal # (URL/parameter value)
// and quoted # are literal. Unsupported constructs remain unknown.
func commandEnvelope(code string, mysql bool) (complete, valid bool) {
	var quote byte
	escaped, word := false, false
	for i := 0; i < len(code); i++ {
		c := code[i]
		if escaped {
			escaped = false
			word = true
			continue
		}
		if quote == '\'' {
			if c == '\'' {
				quote = 0
			}
			continue
		}
		if c == '`' {
			return false, false
		}
		if c == '$' && i+1 < len(code) && code[i+1] == '(' {
			// Only a single plain-word substitution is in this corpus. Do not
			// hide nested shell/comment syntax inside a double-quoted word.
			end := strings.IndexByte(code[i+2:], ')')
			if end < 0 {
				return false, false
			}
			body := code[i+2 : i+2+end]
			if strings.TrimSpace(body) == "" {
				return false, false
			}
			for _, r := range body {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune(" _./-", r)) {
					return false, false
				}
			}
			i += 2 + end
			word = true
			continue
		}
		if quote == '"' {
			if c == '"' {
				quote = 0
			} else if c == '\\' && i+1 < len(code) && strings.ContainsRune("\"$`\\\n", rune(code[i+1])) {
				escaped = true
			}
			continue
		}
		switch c {
		case '\'', '"':
			quote = c
			word = true
		case '\\':
			// cmd.exe path separators are not shell escapes. Escapes only affect
			// syntax characters/newlines; ordinary path bytes remain literal.
			if i+1 == len(code) || strings.ContainsRune(" \t\n'\"#$\\", rune(code[i+1])) {
				escaped = true
			}
			word = true
		case '#':
			if !word || mysql {
				return false, false
			}
		case '`':
			return false, false
		case '<':
			if i+1 < len(code) && code[i+1] == '<' {
				return false, false
			}
			word = false
		case '-', '/':
			if mysql && i+1 < len(code) && ((c == '-' && code[i+1] == '-') || (c == '/' && code[i+1] == '*')) {
				return false, false
			}
			word = true
		case ' ', '\t', '\n', ';', '|', '&', '>':
			word = false
		default:
			if c < 32 || c == 127 {
				return false, false
			}
			word = true
		}
	}
	return quote == 0 && !escaped, true
}

func TerminalTranscript(code string) ([]TranscriptRecord, bool) {
	if !utf8.ValidString(code) || strings.ContainsAny(code, "\x00\r") {
		return nil, false
	}
	lines := strings.SplitAfter(code, "\n")
	records := []TranscriptRecord{}
	pos := 0
	started := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		text := strings.TrimSuffix(line, "\n")
		prompt, body, isPrompt := transcriptPrompt(strings.TrimLeft(text, " \t"))
		if !started && strings.TrimSpace(text) == "" {
			records = append(records, TranscriptRecord{Kind: "output", Start: pos, End: pos + len(line), Text: line})
			pos += len(line)
			continue
		}
		if !started && !isPrompt {
			return nil, false
		}
		if isPrompt {
			started = true
			start := pos
			command := body
			mysql := prompt == "mysql>"
			for {
				complete, valid := commandEnvelope(command, mysql)
				if !valid {
					return nil, false
				}
				pos += len(lines[i])
				if complete {
					break
				}
				i++
				if i >= len(lines) {
					return nil, false
				}
				command += "\n" + strings.TrimSuffix(lines[i], "\n")
			}
			records = append(records, TranscriptRecord{Kind: "command", Prompt: prompt, Start: start, End: pos, Text: code[start:pos]})
		} else {
			records = append(records, TranscriptRecord{Kind: "output", Start: pos, End: pos + len(line), Text: line})
			pos += len(line)
		}
	}
	return records, started
}
