package i18n

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

type LanguageRegistryReviewEntry struct {
	Locale      string `json:"locale"`
	EnglishName string `json:"english_name"`
	Autonym     string `json:"autonym"`
	URL         string `json:"url"`
	Official    bool   `json:"official"`
}

type LanguageRegistryReviewProfileBaseline struct {
	Locale                 string   `json:"locale"`
	Expression             string   `json:"expression"`
	ReferencedDeclarations []string `json:"referenced_declarations"`
}

type LanguageRegistryReviewVariableBaseline struct {
	Name        string `json:"name"`
	Declaration string `json:"declaration"`
}

type LanguageRegistryReviewBaseline struct {
	PackageName         string                                   `json:"package_name"`
	RegistryType        string                                   `json:"registry_type"`
	ProfilesType        string                                   `json:"profiles_type"`
	Directives          []string                                 `json:"directives"`
	Entries             []LanguageRegistryReviewEntry            `json:"entries"`
	Profiles            []LanguageRegistryReviewProfileBaseline  `json:"profiles"`
	Variables           []LanguageRegistryReviewVariableBaseline `json:"variables"`
	RuntimeDeclarations []string                                 `json:"runtime_declarations"`
}

type surfaceReviewLanguageEntry struct {
	Locale      string `json:"locale"`
	EnglishName string `json:"english_name"`
	Autonym     string `json:"autonym"`
	URL         string `json:"url"`
	Official    bool   `json:"official"`
}

type surfaceReviewLanguageProjection struct {
	Locale                  string                     `json:"locale"`
	Target                  surfaceReviewLanguageEntry `json:"target"`
	English                 surfaceReviewLanguageEntry `json:"english"`
	TargetProfileExpression string                     `json:"target_profile_expression"`
	ReferencedDeclarations  []string                   `json:"referenced_declarations"`
	RuntimeDeclarations     []string                   `json:"runtime_declarations"`
}

type parsedSurfaceReviewLanguages struct {
	packageName  string
	registryType ast.Expr
	profilesType ast.Expr
	directives   []string
	entries      []surfaceReviewLanguageEntry
	profiles     map[string]ast.Expr
	variables    map[string]*ast.ValueSpec
	runtimeDecls []ast.Decl
	fileSet      *token.FileSet
}

func currentLanguageRegistryReviewBaseline(root string) (LanguageRegistryReviewBaseline, error) {
	parsed, err := parseSurfaceReviewLanguages(filepath.Join(root, "internal", "tour", "languages.go"))
	if err != nil {
		return LanguageRegistryReviewBaseline{}, err
	}
	return buildLanguageRegistryReviewBaseline(parsed)
}

func buildLanguageRegistryReviewBaseline(parsed *parsedSurfaceReviewLanguages) (LanguageRegistryReviewBaseline, error) {
	registryType, err := formatASTNode(parsed.fileSet, parsed.registryType)
	if err != nil {
		return LanguageRegistryReviewBaseline{}, err
	}
	profilesType, err := formatASTNode(parsed.fileSet, parsed.profilesType)
	if err != nil {
		return LanguageRegistryReviewBaseline{}, err
	}
	entries := make([]LanguageRegistryReviewEntry, 0, len(parsed.entries))
	for _, entry := range parsed.entries {
		entries = append(entries, LanguageRegistryReviewEntry{
			Locale: entry.Locale, EnglishName: entry.EnglishName, Autonym: entry.Autonym, URL: entry.URL, Official: entry.Official,
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Locale < entries[j].Locale })

	profileLocales := make([]string, 0, len(parsed.profiles))
	for locale := range parsed.profiles {
		profileLocales = append(profileLocales, locale)
	}
	sort.Strings(profileLocales)
	profiles := make([]LanguageRegistryReviewProfileBaseline, 0, len(profileLocales))
	for _, locale := range profileLocales {
		expression := parsed.profiles[locale]
		text, err := formatASTNode(parsed.fileSet, expression)
		if err != nil {
			return LanguageRegistryReviewBaseline{}, err
		}
		profiles = append(profiles, LanguageRegistryReviewProfileBaseline{
			Locale: locale, Expression: text, ReferencedDeclarations: referencedLanguageDeclarations(parsed, expression),
		})
	}

	variableNames := make([]string, 0, len(parsed.variables))
	for name := range parsed.variables {
		if name != "languageRegistry" && name != "localeProfiles" {
			variableNames = append(variableNames, name)
		}
	}
	sort.Strings(variableNames)
	variables := make([]LanguageRegistryReviewVariableBaseline, 0, len(variableNames))
	for _, name := range variableNames {
		text, err := formatASTNode(parsed.fileSet, parsed.variables[name])
		if err != nil {
			return LanguageRegistryReviewBaseline{}, err
		}
		variables = append(variables, LanguageRegistryReviewVariableBaseline{Name: name, Declaration: text})
	}

	runtime := make([]string, 0, len(parsed.runtimeDecls))
	for _, decl := range parsed.runtimeDecls {
		text, err := formatASTNode(parsed.fileSet, decl)
		if err != nil {
			return LanguageRegistryReviewBaseline{}, err
		}
		runtime = append(runtime, text)
	}
	return LanguageRegistryReviewBaseline{PackageName: parsed.packageName, RegistryType: registryType, ProfilesType: profilesType, Directives: append([]string(nil), parsed.directives...), Entries: entries, Profiles: profiles, Variables: variables, RuntimeDeclarations: runtime}, nil
}

func languageRegistryReviewBaselineCompatible(recorded, current LanguageRegistryReviewBaseline) bool {
	if !validLanguageRegistryReviewBaseline(recorded) || !validLanguageRegistryReviewBaseline(current) ||
		recorded.PackageName != current.PackageName || recorded.RegistryType != current.RegistryType || recorded.ProfilesType != current.ProfilesType ||
		!reflect.DeepEqual(recorded.Directives, current.Directives) ||
		!reflect.DeepEqual(recorded.RuntimeDeclarations, current.RuntimeDeclarations) {
		return false
	}
	currentEntries := make(map[string]LanguageRegistryReviewEntry, len(current.Entries))
	for _, entry := range current.Entries {
		currentEntries[entry.Locale] = entry
	}
	for _, entry := range recorded.Entries {
		if currentEntries[entry.Locale] != entry {
			return false
		}
	}
	recordedEntries := make(map[string]bool, len(recorded.Entries))
	for _, entry := range recorded.Entries {
		recordedEntries[entry.Locale] = true
	}
	currentProfiles := make(map[string]LanguageRegistryReviewProfileBaseline, len(current.Profiles))
	for _, profile := range current.Profiles {
		currentProfiles[profile.Locale] = profile
	}
	for _, profile := range recorded.Profiles {
		currentProfile, ok := currentProfiles[profile.Locale]
		if !ok || !reflect.DeepEqual(currentProfile, profile) {
			return false
		}
	}
	recordedVariables := make(map[string]LanguageRegistryReviewVariableBaseline, len(recorded.Variables))
	for _, variable := range recorded.Variables {
		recordedVariables[variable.Name] = variable
	}
	allowedNewVariables := map[string]bool{}
	for _, profile := range current.Profiles {
		if recordedEntries[profile.Locale] {
			continue
		}
		for _, declaration := range profile.ReferencedDeclarations {
			if name, _, ok := strings.Cut(declaration, "="); ok {
				allowedNewVariables[name] = true
			}
		}
	}
	for _, variable := range current.Variables {
		if old, ok := recordedVariables[variable.Name]; ok {
			if old != variable {
				return false
			}
			delete(recordedVariables, variable.Name)
			continue
		}
		if !allowedNewVariables[variable.Name] {
			return false
		}
	}
	return len(recordedVariables) == 0
}

func validLanguageRegistryReviewBaseline(baseline LanguageRegistryReviewBaseline) bool {
	if baseline.PackageName == "" || baseline.RegistryType == "" || baseline.ProfilesType == "" || len(baseline.Entries) == 0 || len(baseline.RuntimeDeclarations) == 0 {
		return false
	}
	entryLocales := make(map[string]bool, len(baseline.Entries))
	englishCount := 0
	lastLocale := ""
	for _, entry := range baseline.Entries {
		if entry.Locale == "" || entry.EnglishName == "" || entry.Autonym == "" || entry.URL == "" ||
			entryLocales[entry.Locale] || (lastLocale != "" && entry.Locale <= lastLocale) {
			return false
		}
		entryLocales[entry.Locale] = true
		lastLocale = entry.Locale
		if entry.Locale == "en" {
			if !entry.Official || entry.URL != "https://go.dev/tour/" {
				return false
			}
			englishCount++
		}
	}
	if englishCount != 1 || len(baseline.Profiles) < len(baseline.Entries)-1 || len(baseline.Profiles) > len(baseline.Entries) {
		return false
	}
	profileLocales := make(map[string]bool, len(baseline.Profiles))
	lastLocale = ""
	for _, profile := range baseline.Profiles {
		if profile.Locale == "" || profile.Expression == "" || !entryLocales[profile.Locale] || profileLocales[profile.Locale] ||
			(lastLocale != "" && profile.Locale <= lastLocale) {
			return false
		}
		profileLocales[profile.Locale] = true
		lastLocale = profile.Locale
	}
	for locale := range entryLocales {
		if locale != "en" && !profileLocales[locale] {
			return false
		}
	}
	lastVariable := ""
	variables := make(map[string]string, len(baseline.Variables))
	for _, variable := range baseline.Variables {
		if variable.Name == "" || variable.Declaration == "" || variable.Name == "languageRegistry" || variable.Name == "localeProfiles" ||
			(lastVariable != "" && variable.Name <= lastVariable) {
			return false
		}
		lastVariable = variable.Name
		variables[variable.Name] = variable.Declaration
	}
	for _, profile := range baseline.Profiles {
		lastReference := ""
		for _, reference := range profile.ReferencedDeclarations {
			name, declaration, ok := strings.Cut(reference, "=")
			if !ok || name == "" || declaration == "" || variables[name] != declaration ||
				(lastReference != "" && name <= lastReference) {
				return false
			}
			lastReference = name
		}
	}
	for _, directive := range baseline.Directives {
		if directive == "" {
			return false
		}
	}
	for _, declaration := range baseline.RuntimeDeclarations {
		if declaration == "" {
			return false
		}
	}
	return true
}

func currentLanguageReviewProjectionSHA256(root, locale string) (string, error) {
	parsed, err := parseSurfaceReviewLanguages(filepath.Join(root, "internal", "tour", "languages.go"))
	if err != nil {
		return "", err
	}
	var target, english *surfaceReviewLanguageEntry
	for i := range parsed.entries {
		entry := &parsed.entries[i]
		if entry.Locale == locale {
			target = entry
		}
		if entry.Locale == "en" {
			english = entry
		}
	}
	profile, ok := parsed.profiles[locale]
	if target == nil || english == nil || !ok {
		return "", fmt.Errorf("language review projection requires registry target, English entry, and locale profile for %s", locale)
	}
	profileText, err := formatASTNode(parsed.fileSet, profile)
	if err != nil {
		return "", err
	}
	runtime := make([]string, 0, len(parsed.runtimeDecls))
	for _, decl := range parsed.runtimeDecls {
		text, err := formatASTNode(parsed.fileSet, decl)
		if err != nil {
			return "", err
		}
		runtime = append(runtime, text)
	}
	projection := surfaceReviewLanguageProjection{
		Locale: locale, Target: *target, English: *english,
		TargetProfileExpression: profileText,
		ReferencedDeclarations:  referencedLanguageDeclarations(parsed, profile),
		RuntimeDeclarations:     runtime,
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		return "", err
	}
	return hashBytes(encoded), nil
}

func validateCurrentLanguageRegistryCompatibility(root string) error {
	parsed, err := parseSurfaceReviewLanguages(filepath.Join(root, "internal", "tour", "languages.go"))
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(root, "production", "identity.json"))
	if err != nil {
		return fmt.Errorf("read language registry production identity: %w", err)
	}
	var identity localeSurfaceReviewProductionIdentity
	if err := decodeSingleJSONValue(data, &identity); err != nil {
		return fmt.Errorf("parse language registry production identity: %w", err)
	}
	byLocale := map[string]localeSurfaceReviewProductionProfile{}
	for _, profile := range identity.Locales {
		if profile.Locale == "" || profile.ProductionHostname == "" || profile.ProductionPublicURL == "" {
			return fmt.Errorf("language registry production identity has an incomplete public profile")
		}
		if _, exists := byLocale[profile.Locale]; exists {
			return fmt.Errorf("language registry production identity has duplicate locale %s", profile.Locale)
		}
		byLocale[profile.Locale] = profile
	}
	seenLocale, seenURL, seenEnglish := map[string]bool{}, map[string]bool{}, map[string]bool{}
	englishCount := 0
	lastEnglishName := ""
	for _, entry := range parsed.entries {
		if entry.Locale == "" || entry.EnglishName == "" || entry.Autonym == "" || entry.URL == "" {
			return fmt.Errorf("language registry has an incomplete entry")
		}
		if seenLocale[entry.Locale] || seenURL[entry.URL] || seenEnglish[entry.EnglishName] {
			return fmt.Errorf("language registry has duplicate locale, URL, or English name at %s", entry.Locale)
		}
		seenLocale[entry.Locale], seenURL[entry.URL], seenEnglish[entry.EnglishName] = true, true, true
		if lastEnglishName != "" && entry.EnglishName < lastEnglishName {
			return fmt.Errorf("language registry is not ordered by English name at %s", entry.Locale)
		}
		lastEnglishName = entry.EnglishName
		parsedURL, err := url.Parse(entry.URL)
		if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() == "" || parsedURL.User != nil || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
			return fmt.Errorf("language registry URL is not canonical HTTPS for %s", entry.Locale)
		}
		if entry.Locale == "en" {
			englishCount++
			if !entry.Official || entry.URL != "https://go.dev/tour/" {
				return fmt.Errorf("language registry English entry must be the official Tour URL")
			}
			continue
		}
		if entry.Official || parsedURL.Path != "/" {
			return fmt.Errorf("language registry community URL is invalid for %s", entry.Locale)
		}
		profile, ok := byLocale[entry.Locale]
		if !ok || profile.ProductionPublicURL != entry.URL || profile.ProductionHostname != parsedURL.Hostname() {
			return fmt.Errorf("language registry entry %s does not exactly match production public identity", entry.Locale)
		}
		if _, ok := parsed.profiles[entry.Locale]; !ok {
			return fmt.Errorf("language registry entry %s has no runtime locale profile", entry.Locale)
		}
	}
	if englishCount != 1 {
		return fmt.Errorf("language registry requires exactly one official English entry")
	}
	for locale := range parsed.profiles {
		if !seenLocale[locale] {
			return fmt.Errorf("runtime locale profile %s has no community registry entry", locale)
		}
	}
	for locale := range byLocale {
		if !seenLocale[locale] {
			return fmt.Errorf("production public identity locale %s is missing from language registry", locale)
		}
	}
	return nil
}

func parseSurfaceReviewLanguages(path string) (*parsedSurfaceReviewLanguages, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse language registry: %w", err)
	}
	result := &parsedSurfaceReviewLanguages{packageName: file.Name.Name, profiles: map[string]ast.Expr{}, variables: map[string]*ast.ValueSpec{}, fileSet: fset}
	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") || strings.HasPrefix(comment.Text, "// +build") {
				result.directives = append(result.directives, comment.Text)
			}
		}
	}
	for _, decl := range file.Decls {
		switch value := decl.(type) {
		case *ast.FuncDecl:
			result.runtimeDecls = append(result.runtimeDecls, decl)
		case *ast.GenDecl:
			if value.Tok == token.IMPORT || value.Tok == token.CONST || value.Tok == token.TYPE {
				result.runtimeDecls = append(result.runtimeDecls, decl)
			}
			if value.Tok != token.VAR {
				continue
			}
			for _, spec := range value.Specs {
				vs, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vs.Names {
					result.variables[name.Name] = vs
				}
				for i, name := range vs.Names {
					if i >= len(vs.Values) {
						continue
					}
					literal, ok := vs.Values[i].(*ast.CompositeLit)
					if !ok {
						continue
					}
					switch name.Name {
					case "languageRegistry":
						result.registryType = literal.Type
						entries, err := parseLanguageRegistryLiteral(literal)
						if err != nil {
							return nil, err
						}
						result.entries = entries
					case "localeProfiles":
						result.profilesType = literal.Type
						for _, element := range literal.Elts {
							pair, ok := element.(*ast.KeyValueExpr)
							if !ok {
								return nil, fmt.Errorf("localeProfiles contains a non-keyed entry")
							}
							key, err := stringLiteral(pair.Key)
							if err != nil || result.profiles[key] != nil {
								return nil, fmt.Errorf("localeProfiles has invalid or duplicate locale key")
							}
							result.profiles[key] = pair.Value
						}
					}
				}
			}
		}
	}
	if len(result.entries) == 0 || len(result.profiles) == 0 || result.registryType == nil || result.profilesType == nil {
		return nil, fmt.Errorf("language registry source is missing languageRegistry or localeProfiles")
	}
	return result, nil
}

func parseLanguageRegistryLiteral(literal *ast.CompositeLit) ([]surfaceReviewLanguageEntry, error) {
	entries := make([]surfaceReviewLanguageEntry, 0, len(literal.Elts))
	for _, element := range literal.Elts {
		item, ok := element.(*ast.CompositeLit)
		if !ok {
			return nil, fmt.Errorf("languageRegistry contains a non-literal entry")
		}
		entry := surfaceReviewLanguageEntry{}
		for _, field := range item.Elts {
			pair, ok := field.(*ast.KeyValueExpr)
			if !ok {
				return nil, fmt.Errorf("languageRegistry contains an unkeyed field")
			}
			name, ok := pair.Key.(*ast.Ident)
			if !ok {
				return nil, fmt.Errorf("languageRegistry contains an invalid field")
			}
			var err error
			switch name.Name {
			case "Locale":
				entry.Locale, err = stringLiteral(pair.Value)
			case "EnglishName":
				entry.EnglishName, err = stringLiteral(pair.Value)
			case "Autonym":
				entry.Autonym, err = stringLiteral(pair.Value)
			case "URL":
				entry.URL, err = stringLiteral(pair.Value)
			case "Official":
				ident, ok := pair.Value.(*ast.Ident)
				if !ok || (ident.Name != "true" && ident.Name != "false") {
					err = fmt.Errorf("expected boolean literal")
				} else {
					entry.Official = ident.Name == "true"
				}
			default:
				err = fmt.Errorf("unsupported field %s", name.Name)
			}
			if err != nil {
				return nil, fmt.Errorf("languageRegistry entry %s: %w", entry.Locale, err)
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func stringLiteral(expr ast.Expr) (string, error) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", fmt.Errorf("expected string literal")
	}
	return strconv.Unquote(literal.Value)
}

func referencedLanguageDeclarations(parsed *parsedSurfaceReviewLanguages, expression ast.Expr) []string {
	needed := map[string]bool{}
	var visit func(ast.Node)
	visit = func(node ast.Node) {
		ast.Inspect(node, func(child ast.Node) bool {
			ident, ok := child.(*ast.Ident)
			if !ok || needed[ident.Name] || parsed.variables[ident.Name] == nil {
				return true
			}
			needed[ident.Name] = true
			visit(parsed.variables[ident.Name])
			return true
		})
	}
	visit(expression)
	names := make([]string, 0, len(needed))
	for name := range needed {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]string, 0, len(names))
	for _, name := range names {
		text, err := formatASTNode(parsed.fileSet, parsed.variables[name])
		if err == nil {
			result = append(result, name+"="+text)
		}
	}
	return result
}

func formatASTNode(fset *token.FileSet, node any) (string, error) {
	var buffer bytes.Buffer
	if err := format.Node(&buffer, fset, node); err != nil {
		return "", err
	}
	return strings.TrimSpace(buffer.String()), nil
}
