package i18n

import (
	"bufio"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

var literalTreePath = regexp.MustCompile(`^[A-Za-z0-9_.<>/-]+$`)
var literalVersion = regexp.MustCompile(`^v([0-9]+|x)\.([0-9]+|x)\.([0-9]+|x)(-[A-Za-z0-9.-]+)?$`)
var literalPlaceholderPath = regexp.MustCompile(`^<[A-Za-z][A-Za-z0-9-]*>(/<[A-Za-z][A-Za-z0-9-]*>)+$`)
var literalTuple = regexp.MustCompile(`^\(([A-Za-z_][A-Za-z_0-9]*)(,\s*[A-Za-z_][A-Za-z_0-9]*)+\)$`)

func SourceLiteralProof(code, context string) (string, bool) {
	s := strings.TrimSpace(code)
	context = strings.Join(strings.Fields(context), " ")
	if s == "" {
		return "", false
	}
	// These are explicit source declarations, not a keyword search for code
	// language. The entire block is quoted output under that declaration.
	for _, declaration := range []string{"You should see this output:", "You will see this output:", "You will now see this output:"} {
		if context == declaration {
			return "adjacent source paragraph explicitly declares terminal output: " + context, true
		}
	}
	if strings.HasSuffix(context, "the output will end with something like this:") {
		return "adjacent source paragraph explicitly declares diagnostic output: " + context, true
	}
	if strings.HasPrefix(context, "First, consider the docs for ") {
		name := strings.Fields(s)[0]
		if strings.HasSuffix(context, "."+name+".") {
			return "source explicitly introduces quoted API documentation: " + context, true
		}
	}
	if strings.HasPrefix(s, "HTTP/") {
		head, body, ok := strings.Cut(strings.TrimLeft(code, "\n"), "\n\n")
		if ok && json.Valid([]byte(body)) {
			response, err := http.ReadResponse(bufio.NewReader(strings.NewReader(head+"\n\n")), nil)
			if err == nil {
				response.Body.Close()
				return "net/http response header grammar plus complete JSON body; displayed Content-Length retained as source literal", true
			}
		}
	}
	if literalPlaceholderPath.MatchString(s) {
		return "angle-delimited path metavariable grammar", true
	}
	if literalTuple.MatchString(s) && strings.Contains(context, "a tuple of ") {
		ok := true
		for _, name := range strings.FieldsFunc(s, func(r rune) bool { return r == '(' || r == ')' || r == ',' || r == ' ' }) {
			if !regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`).MatchString(context) {
				ok = false
			}
		}
		if ok {
			return "source-declared tuple notation with source-mentioned members: " + context, true
		}
	}
	versions := true
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if !literalVersion.MatchString(line) || !semver.IsValid(strings.ReplaceAll(line, "x", "0")) {
			versions = false
		}
	}
	if versions {
		return "complete semantic-version / version-metavariable listing grammar", true
	}
	words := strings.Fields(s)
	if len(words) == 3 && words[0] == "module" && module.Check(words[1], words[2]) == nil {
		return "module path/version pair validated by x/mod/module.Check", true
	}
	if literalFileTree(s) {
		return "complete rooted file-tree indentation/branch grammar", true
	}
	if literalFuzzCorpus(s) {
		return "go test fuzz v1 data header and Go AST typed-literal records", true
	}
	if strings.Contains(context, "mutation strategy may look like the following:") {
		lines := strings.Split(s, "\n")
		ok := len(lines) > 2 && lines[len(lines)-1] == "..."
		for i, line := range lines[:len(lines)-1] {
			prefix := strconv.Itoa(i) + ". "
			if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, "]") || !strings.Contains(line, " [") {
				ok = false
			}
		}
		if ok {
			return "source-declared sequential numbered mutation trace with bracketed annotations", true
		}
	}
	return "", false
}

func literalFileTree(s string) bool {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 || !strings.HasSuffix(lines[0], "/") || !literalTreePath.MatchString(lines[0]) {
		return false
	}
	last := 0
	for _, line := range lines[1:] {
		n := len(line) - len(strings.TrimLeft(line, " "))
		entry := line[n:]
		branch := strings.HasPrefix(entry, "|-- ")
		if branch {
			entry = strings.TrimPrefix(entry, "|-- ")
		}
		if n == 0 || n > last+2 || (!branch && n%2 != 0) {
			return false
		}
		if !literalTreePath.MatchString(entry) && !strings.HasPrefix(entry, "... ") {
			return false
		}
		last = n
	}
	return true
}

func literalFuzzCorpus(s string) bool {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 || lines[0] != "go test fuzz v1" {
		return false
	}
	for _, line := range lines[1:] {
		e, err := parser.ParseExpr(line)
		if err != nil {
			return false
		}
		call, ok := e.(*ast.CallExpr)
		if !ok || len(call.Args) != 1 {
			return false
		}
		validType := false
		switch typ := call.Fun.(type) {
		case *ast.Ident:
			switch typ.Name {
			case "string", "bool", "byte", "rune", "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
				validType = true
			}
		case *ast.ArrayType:
			id, ok := typ.Elt.(*ast.Ident)
			validType = ok && typ.Len == nil && id.Name == "byte"
		}
		arg := call.Args[0]
		if unary, ok := arg.(*ast.UnaryExpr); ok && unary.Op == token.SUB {
			arg = unary.X
		}
		_, literal := arg.(*ast.BasicLit)
		if id, ok := arg.(*ast.Ident); ok {
			literal = id.Name == "true" || id.Name == "false"
		}
		events, err := scanGoExampleLexicalEvents([]byte(line))
		comments := false
		for _, event := range events {
			if event.token == token.COMMENT {
				comments = true
			}
		}
		if !validType || !literal || comments || err != nil {
			return false
		}
	}
	return true
}
