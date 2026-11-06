package languageview

import (
	"archive/zip"
	"database/sql"
	"encoding/binary"
	"go/token"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// These are evidence roles, not guesses about a word's appearance. Explicit
// markup/code/targets are owned by the parser before the prose classifier runs.
// No role is inferred from capitalization, camel case or punctuation alone.
type semanticRole string

const (
	naturalLanguage     semanticRole = "NATURAL_LANGUAGE"
	technicalIdentifier semanticRole = "TECHNICAL_IDENTIFIER"
	codeOrLiteral       semanticRole = "CODE_OR_LITERAL"
	structuralToken     semanticRole = "STRUCTURAL_TOKEN"
	urlOrTarget         semanticRole = "URL_OR_TARGET"
	knownTechnicalAtom  semanticRole = "KNOWN_TECHNICAL_ATOM"
	ambiguousFailClosed semanticRole = "AMBIGUOUS_FAIL_CLOSED"
)

// Closed identity namespaces, with semantic authorities, rather than a list of
// source words to exempt. Ordinary terminology/brands remain the glossary's
// responsibility (TRANSLATION_TERMINOLOGY.md §§2–3).
// IETF/HTTP methods and protocol names; Unicode encodings; Go tool environment;
// and the project's shared identity policy (§6: API/CLI/URL/ASCII/CPU/UTC).
var technicalIdentities = func() map[string]bool {
	m := map[string]bool{}
	for _, s := range strings.Fields("API CLI URL ASCII CPU UTC HTTP HTTPS HTML XML JSON SQL TCP UDP TLS UTF-8 UTF-16 NUL GOPATH GOROOT GOMOD GOWORK GOPROXY GOSUMDB GONOSUMDB GOPRIVATE GOOS GOARCH GODEBUG CGO_ENABLED") {
		m[s] = true
	}
	// Named technical identities used by the registered security/database Pages:
	// NIST algorithm/validation programs, CVE/CVSS/OSV vulnerability namespaces,
	// IETF/WHATWG formats/protocols and database interface standards. These are
	// positive namespace authority, not exemptions for English discourse words.
	for _, s := range strings.Fields("AES-GCM CAVP CMVP CSPRNG DRBG FIPS ML-DSA NIST P-224 CVE CVSS CNA GHSA NVD OSV WHATWG GOAWAY HTTP/2 RPC gRPC REST XSS DoS DSN DBMS DDL ORM ID IDE OS YAML RNG ARM64 AMD64 AIX UTF-32 SP") {
		m[s] = true
	}
	return m
}()

// Bare references lack a code delimiter or package qualifier. Their authority
// is an actual Go API binding, not a spelling rule or an English exception
// list. The compiler resolves these symbols; names are derived from that
// binding. Do not register ambiguous ordinary-English method names globally.
var bareGoIdentities = func() map[string]bool {
	m := map[string]bool{}
	for _, symbol := range []any{binary.ReadVarint, binary.ReadUvarint, (*sql.Tx).Stmt, (*zip.Reader).Open} {
		name := runtime.FuncForPC(reflect.ValueOf(symbol).Pointer()).Name()
		parts := strings.Split(name, ".")
		id := parts[len(parts)-1]
		if len(parts) > 1 && strings.HasPrefix(parts[len(parts)-2], "(*") {
			id = strings.Trim(parts[len(parts)-2], "(*)") + "." + id
		}
		m[id] = true
	}
	return m
}()

// Go's documentation parser owns the standard-library package namespace.
// Merely looking like a selector or a slash-separated word is insufficient.
func standardGoIdentity(word string) bool {
	if standardGoPackages[word] {
		return true
	}
	parts := strings.Split(word, ".")
	if len(parts) < 2 || len(parts) > 3 || !standardGoPackageNames[parts[0]] {
		return false
	}
	for _, part := range parts[1:] {
		if !token.IsIdentifier(part) || part[0] < 'A' || part[0] > 'Z' {
			return false
		}
	}
	return true
}

var (
	cliOption             = regexp.MustCompile(`^--[A-Za-z][A-Za-z0-9-]*(=[^\s]+)?$`)
	shortOption           = regexp.MustCompile(`^-[A-Za-z][A-Za-z0-9-]*(=[^\s]+)?$`)
	optionPosition        = regexp.MustCompile(`(?i)\b(?:flag|option)\s+$`)
	versionIdentity       = regexp.MustCompile(`^v\.?[0-9Xx]+\.[0-9Yyx]+\.[0-9Zzx]+(?:-[A-Za-z0-9.-]+)?$`)
	versionPosition       = regexp.MustCompile(`(?:\bGo|(?i:\bversion|\brelease))\s+$`)
	releaseIdentity       = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+){1,2}\+?$`)
	registeredIdentity    = regexp.MustCompile(`^(?:CVE-[0-9]{4}-[0-9]+|GO-[0-9]{4}-[0-9]+|GHSA-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4})$`)
	certificatePosition   = regexp.MustCompile(`(?i)\bcertificate\s+$`)
	certificateIdentifier = regexp.MustCompile(`^[A-Z][0-9]+$`)
	identifierPosition    = regexp.MustCompile(`(?i)(?:\b(?:function|method|field|type|identifier|symbol|package|constant|variable|calling|called|custom|command)|environment variable)\s+$`)
	enumPosition          = regexp.MustCompile(`(?i)\b(?:state|status|enum|track)\s+$`)
	enumFollowing         = regexp.MustCompile(`(?i)^\s+(?:state|status|track)\b`)
)

func classifyProseToken(word, before, after string) semanticRole {
	return classifyProseTokenInRole(word, before, after, true)
}

// Heading navigation labels are language, not declarations of named symbols.
// Explicit code/URL/API namespaces still take precedence in either role.
func classifyProseTokenInRole(word, before, after string, namedPosition bool) semanticRole {
	if word == "" {
		return naturalLanguage
	}
	if technicalIdentities[word] {
		return knownTechnicalAtom
	}
	if bareGoIdentities[word] {
		return technicalIdentifier
	}
	if registeredIdentity.MatchString(word) {
		return knownTechnicalAtom
	}
	// Test the bounded token before scanning its potentially long prose prefix.
	// These pure predicates commute; role precedence and patterns stay exact.
	if (releaseIdentity.MatchString(word) && versionPosition.MatchString(before)) || (certificateIdentifier.MatchString(word) && certificatePosition.MatchString(before)) {
		return technicalIdentifier
	}
	if u, err := url.Parse(word); err == nil && u.Scheme != "" && (u.Host != "" || u.Scheme == "mailto") {
		return urlOrTarget
	}
	// The public suffix registry supplies domain authority. A dot alone does
	// not: prose abbreviations are not treated as host names.
	host := strings.SplitN(word, "/", 2)[0]
	if suffix, icann := publicsuffix.PublicSuffix(strings.ToLower(host)); icann && strings.Contains(host, ".") && host != suffix {
		return urlOrTarget
	}
	if cliOption.MatchString(word) || (shortOption.MatchString(word) && optionPosition.MatchString(before)) || versionIdentity.MatchString(word) {
		return codeOrLiteral
	}
	// A filename's extension is a format identity, not merely a dot. This list
	// describes supported code/config/media formats, never abbreviations.
	switch path.Ext(word) {
	case ".go", ".mod", ".sum", ".work", ".sql", ".json", ".yaml", ".yml", ".toml", ".xml", ".html", ".md", ".txt", ".zip", ".gz", ".mp4":
		return urlOrTarget
	}
	if (strings.HasPrefix(word, "/") && strings.Trim(word, "/") != "") || strings.HasPrefix(word, "./") || strings.HasPrefix(word, "../") {
		return urlOrTarget
	}
	if strings.ContainsAny(word, "/.") && standardGoIdentity(word) {
		return technicalIdentifier
	}
	identifierContext := namedPosition && identifierPosition.MatchString(before)
	enumContext := strings.ToUpper(word) == word && (enumPosition.MatchString(before) || enumFollowing.MatchString(after))
	if identifierContext || enumContext {
		// The role-bearing context is necessary. Capitalization alone never
		// establishes identity; ordinary following prose ("package can ...")
		// must remain editable.
		if token.IsIdentifier(word) && (word[0] >= 'A' && word[0] <= 'Z' || strings.Contains(word, "_")) {
			return technicalIdentifier
		}
		parts := strings.Split(word, ".")
		if len(parts) == 2 && token.IsIdentifier(parts[0]) && token.IsIdentifier(parts[1]) {
			return technicalIdentifier
		}
		// Only an explicit identity position can be ambiguous. Preserve its
		// bytes; never apply this fallback to an unknown prose abbreviation.
		if strings.ContainsAny(word, "_=()") {
			return ambiguousFailClosed
		}
	}
	// HTTP status literals have a protocol role, not an all-caps heuristic.
	if strings.HasSuffix(strings.TrimSpace(before), "HTTP") {
		switch word {
		case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace, http.MethodPatch:
			return codeOrLiteral
		}
		if n, err := strconv.Atoi(word); err == nil && http.StatusText(n) != "" {
			return codeOrLiteral
		}
	}
	return naturalLanguage
}

// Immutable source-foundation reader only. Do not use for new Page exports.
func legacyMachineWord(word, trim string) bool {
	tech := (len(trim) > 1 && strings.ToUpper(trim) == trim && strings.IndexFunc(trim, func(r rune) bool { return r >= 'A' && r <= 'Z' }) >= 0) || strings.ContainsAny(trim, "/\\_") || strings.Contains(trim, "://") || strings.Contains(trim, ".") || strings.Contains(trim, "=") || strings.HasPrefix(trim, "--") || strings.Contains(word, "()")
	for j := 1; j < len(trim); j++ {
		if trim[j] >= 'A' && trim[j] <= 'Z' && trim[j-1] >= 'a' && trim[j-1] <= 'z' {
			tech = true
		}
	}
	return tech
}

type Role = semanticRole

const (
	NaturalLanguage     = naturalLanguage
	TechnicalIdentifier = technicalIdentifier
	CodeOrLiteral       = codeOrLiteral
	StructuralToken     = structuralToken
	URLOrTarget         = urlOrTarget
	KnownTechnicalAtom  = knownTechnicalAtom
	AmbiguousFailClosed = ambiguousFailClosed
)

func ClassifyProseTokenInRole(w, b, a string, named bool) Role {
	return classifyProseTokenInRole(w, b, a, named)
}
func StandardGoIdentity(w string) bool   { return standardGoIdentity(w) }
func LegacyMachineWord(w, t string) bool { return legacyMachineWord(w, t) }

var IdentifierPosition = identifierPosition
var EnumPosition = enumPosition
var EnumFollowing = enumFollowing
