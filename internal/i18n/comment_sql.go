package i18n

import "strings"

// SQLCommentFreeProof accepts only the complete MySQL tutorial productions
// below. It is not a general SQL parser. Quotes own their --/# delimiters;
// actual comments and every unsupported production fail closed.
func SQLCommentFreeProof(code string) bool {
	tokens := []string{}
	for i := 0; i < len(code); {
		c := code[i]
		if strings.ContainsRune(" \t\n\r", rune(c)) {
			i++
			continue
		}
		if c == '#' || (i+1 < len(code) && ((c == '-' && code[i+1] == '-') || (c == '/' && code[i+1] == '*'))) {
			return false
		}
		if c == '\'' || c == '"' || c == '`' {
			start, quote := i, c
			i++
			closed := false
			for i < len(code) {
				if code[i] == '\\' {
					i += 2
					continue
				}
				if code[i] == quote {
					if i+1 < len(code) && code[i+1] == quote {
						i += 2
						continue
					}
					i++
					closed = true
					break
				}
				i++
			}
			if !closed || i > len(code) {
				return false
			}
			tokens = append(tokens, code[start:i])
			continue
		}
		if strings.ContainsRune("(),;*=.+-", rune(c)) {
			tokens = append(tokens, string(c))
			i++
			continue
		}
		start := i
		for i < len(code) && (code[i] >= 'a' && code[i] <= 'z' || code[i] >= 'A' && code[i] <= 'Z' || code[i] >= '0' && code[i] <= '9' || code[i] == '_') {
			i++
		}
		if i == start {
			return false
		}
		tokens = append(tokens, code[start:i])
	}
	p := sqlProofParser{tokens: tokens}
	if len(tokens) == 0 {
		return false
	}
	for p.pos < len(tokens) {
		if !p.statement() {
			return false
		}
	}
	return true
}

type sqlProofParser struct {
	tokens []string
	pos    int
}

func (p *sqlProofParser) take(s string) bool {
	if p.pos < len(p.tokens) && strings.EqualFold(p.tokens[p.pos], s) {
		p.pos++
		return true
	}
	return false
}
func (p *sqlProofParser) identifier() bool {
	if p.pos >= len(p.tokens) {
		return false
	}
	s := p.tokens[p.pos]
	if len(s) > 2 && s[0] == '`' && s[len(s)-1] == '`' {
		p.pos++
		return true
	}
	for i, c := range s {
		if !(c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	if s == "" {
		return false
	}
	p.pos++
	return true
}
func (p *sqlProofParser) number() bool {
	start := p.pos
	p.take("-")
	if p.pos >= len(p.tokens) {
		p.pos = start
		return false
	}
	for _, r := range p.tokens[p.pos] {
		if r < '0' || r > '9' {
			p.pos = start
			return false
		}
	}
	p.pos++
	if p.take(".") {
		if p.pos >= len(p.tokens) {
			return false
		}
		for _, r := range p.tokens[p.pos] {
			if r < '0' || r > '9' {
				return false
			}
		}
		p.pos++
	}
	return true
}
func (p *sqlProofParser) value() bool {
	if p.pos < len(p.tokens) && strings.HasPrefix(p.tokens[p.pos], "'") {
		p.pos++
		return true
	}
	return p.number()
}
func (p *sqlProofParser) identifiers() bool {
	if !p.identifier() {
		return false
	}
	for p.take(",") {
		if !p.identifier() {
			return false
		}
	}
	return true
}
func (p *sqlProofParser) statement() bool {
	switch {
	case p.take("DROP"):
		if !p.take("TABLE") {
			return false
		}
		if p.take("IF") && !p.take("EXISTS") {
			return false
		}
		return p.identifier() && p.take(";")
	case p.take("CREATE"):
		if !p.take("TABLE") || !p.identifier() || !p.take("(") {
			return false
		}
		for {
			if p.take("PRIMARY") {
				if !p.take("KEY") || !p.take("(") || !p.identifiers() || !p.take(")") {
					return false
				}
			} else {
				if !p.identifier() {
					return false
				}
				switch {
				case p.take("INT"):
				case p.take("VARCHAR"):
					if !p.take("(") || !p.number() || !p.take(")") {
						return false
					}
				case p.take("DECIMAL"):
					if !p.take("(") || !p.number() || !p.take(",") || !p.number() || !p.take(")") {
						return false
					}
				default:
					return false
				}
				for {
					if p.take("AUTO_INCREMENT") {
						continue
					}
					if p.take("NOT") {
						if !p.take("NULL") {
							return false
						}
						continue
					}
					break
				}
			}
			if !p.take(",") {
				break
			}
		}
		return p.take(")") && p.take(";")
	case p.take("INSERT"):
		if !p.take("INTO") || !p.identifier() {
			return false
		}
		if p.take("(") && (!p.identifiers() || !p.take(")")) {
			return false
		}
		if !p.take("VALUES") {
			return false
		}
		for {
			if !p.take("(") || !p.value() {
				return false
			}
			for p.take(",") {
				if !p.value() {
					return false
				}
			}
			if !p.take(")") {
				return false
			}
			if !p.take(",") {
				break
			}
		}
		return p.take(";")
	case p.take("SELECT"):
		if !p.take("*") && !p.value() {
			return false
		}
		if !p.take("FROM") || !p.identifier() {
			return false
		}
		if p.take("WHERE") {
			if !p.identifier() || !p.take("=") || !p.value() {
				return false
			}
			for p.take("OR") {
				if !p.value() || !p.take("=") || !p.value() {
					return false
				}
			}
		}
		return p.take(";")
	}
	return false
}
