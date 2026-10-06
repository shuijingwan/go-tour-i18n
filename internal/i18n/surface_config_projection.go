package i18n

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

// TourConfigProjection freezes declarations that influence the reviewed Tour
// surface, including values, struct fields and function bodies. Additional
// unreferenced pure functions/constants can coexist without changing it.
// Initializers, methods, new variables/types/imports and Go directives cannot
// be mechanically proved unrelated and are rejected on compatibility paths.
type TourConfigProjection struct {
	Version      string                  `json:"version"`
	Package      string                  `json:"package"`
	Declarations []TourConfigDeclaration `json:"declarations"`
	SHA256       string                  `json:"sha256"`
}

type TourConfigDeclaration struct {
	ID           string   `json:"id"`
	Text         string   `json:"text"`
	References   []string `json:"references"`
	AdditiveSafe bool     `json:"additive_safe"`
}

type TourSurfaceConfigProjections struct {
	Project TourConfigProjection `json:"project"`
	SEO     TourConfigProjection `json:"seo"`
}

type TourSurfaceConfigBaseline struct {
	Schema     string                       `json:"schema"`
	Locale     string                       `json:"locale"`
	ReviewID   string                       `json:"review_id"`
	Gate       GlossaryArchiveReference     `json:"gate"`
	Project    GlossaryArchiveReference     `json:"project"`
	SEO        GlossaryArchiveReference     `json:"seo"`
	Projection TourSurfaceConfigProjections `json:"projection"`
	Identity   string                       `json:"evidence_identity_sha256"`
}

func projectTourConfig(data []byte) (TourConfigProjection, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "config.go", data, parser.SkipObjectResolution)
	if err != nil {
		return TourConfigProjection{}, err
	}
	if file.Name.Name != "tour" {
		return TourConfigProjection{}, fmt.Errorf("unexpected Tour config package")
	}
	// Compiler directives can alter behavior independently of declaration AST.
	withComments, err := parser.ParseFile(token.NewFileSet(), "config.go", data, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return TourConfigProjection{}, err
	}
	for _, group := range withComments.Comments {
		for _, c := range group.List {
			if strings.HasPrefix(c.Text, "//go:") || strings.HasPrefix(c.Text, "//line ") || strings.HasPrefix(c.Text, "// +build ") || strings.HasPrefix(c.Text, "/*line ") {
				return TourConfigProjection{}, fmt.Errorf("unknown config compiler directive")
			}
		}
	}
	projection := TourConfigProjection{Version: "tour-surface-config-projection/v1", Package: file.Name.Name, Declarations: []TourConfigDeclaration{}}
	if len(file.Decls) == 0 {
		return projection, fmt.Errorf("missing Tour config declarations")
	}
	seen := map[string]bool{}
	for _, decl := range file.Decls {
		id := ""
		safe := false
		switch d := decl.(type) {
		case *ast.FuncDecl:
			id = "func:" + d.Name.Name
			if d.Recv != nil {
				if len(d.Recv.List) != 1 {
					return projection, fmt.Errorf("ambiguous config method receiver")
				}
				var b bytes.Buffer
				if err := format.Node(&b, fset, d.Recv.List[0].Type); err != nil {
					return projection, err
				}
				id = "method:" + b.String() + ":" + d.Name.Name
			}
			safe = d.Recv == nil && d.Name.Name != "init"
		case *ast.GenDecl:
			names := []string{}
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range s.Names {
						names = append(names, name.Name)
					}
				case *ast.TypeSpec:
					names = append(names, s.Name.Name)
				case *ast.ImportSpec:
					name := ""
					if s.Name != nil {
						name = s.Name.Name
					}
					names = append(names, name+s.Path.Value)
				default:
					return projection, fmt.Errorf("unknown config specification")
				}
			}
			id = d.Tok.String() + ":" + strings.Join(names, ",")
			safe = d.Tok == token.CONST
			if safe {
				for _, spec := range d.Specs {
					s := spec.(*ast.ValueSpec)
					if len(s.Values) == 0 {
						safe = false
					}
					for _, v := range s.Values {
						if _, ok := v.(*ast.BasicLit); !ok {
							safe = false
						}
					}
				}
			}
		default:
			return projection, fmt.Errorf("unknown config declaration")
		}
		if seen[id] {
			return projection, fmt.Errorf("ambiguous config declaration %s", id)
		}
		seen[id] = true
		var b bytes.Buffer
		if err := format.Node(&b, fset, decl); err != nil {
			return projection, err
		}
		refs := map[string]bool{}
		ast.Inspect(decl, func(n ast.Node) bool {
			if ident, ok := n.(*ast.Ident); ok {
				refs[ident.Name] = true
			}
			return true
		})
		ordered := []string{}
		for ref := range refs {
			ordered = append(ordered, ref)
		}
		sort.Strings(ordered)
		projection.Declarations = append(projection.Declarations, TourConfigDeclaration{id, b.String(), ordered, safe})
	}
	sort.Slice(projection.Declarations, func(i, j int) bool { return projection.Declarations[i].ID < projection.Declarations[j].ID })
	projection.SHA256 = sum(mustJSON(projection))
	return projection, nil
}

func validTourConfigProjection(p TourConfigProjection) bool {
	sha := p.SHA256
	p.SHA256 = ""
	return p.Version == "tour-surface-config-projection/v1" && p.Package == "tour" && len(p.Declarations) > 0 && sha == sum(mustJSON(p))
}

func tourConfigProjectionCompatible(old, current TourConfigProjection) bool {
	if !validTourConfigProjection(old) || !validTourConfigProjection(current) {
		return false
	}
	byID := map[string]TourConfigDeclaration{}
	for _, d := range current.Declarations {
		if _, ok := byID[d.ID]; ok {
			return false
		}
		byID[d.ID] = d
	}
	oldRefs := map[string]bool{}
	for _, d := range old.Declarations {
		got, ok := byID[d.ID]
		if !ok || !reflect.DeepEqual(d, got) {
			return false
		}
		delete(byID, d.ID)
		for _, ref := range d.References {
			oldRefs[ref] = true
		}
	}
	for _, d := range byID {
		if !d.AdditiveSafe {
			return false
		}
		_, name, _ := strings.Cut(d.ID, ":")
		for _, part := range strings.Split(name, ",") {
			if oldRefs[part] {
				return false
			}
		}
	}
	return true
}

func currentTourSurfaceConfigProjections(root string) (*TourSurfaceConfigProjections, error) {
	a, err := readCompatibilityFile(root, "internal/tour/project.go")
	if err != nil {
		return nil, err
	}
	b, err := readCompatibilityFile(root, "internal/tour/seo.go")
	if err != nil {
		return nil, err
	}
	p, err := projectTourConfig(a)
	if err != nil {
		return nil, err
	}
	s, err := projectTourConfig(b)
	if err != nil {
		return nil, err
	}
	return &TourSurfaceConfigProjections{p, s}, nil
}

func archiveRecordedTourSurfaceConfig(root string, inputs LocaleSurfaceReviewAInputs) error {
	project, err := readCompatibilityFile(root, "internal/tour/project.go")
	if err != nil {
		return err
	}
	seo, err := readCompatibilityFile(root, "internal/tour/seo.go")
	if err != nil {
		return err
	}
	if sum(project) != inputs.ProjectConfigSHA256 || sum(seo) != inputs.SEOConfigSHA256 {
		return fmt.Errorf("Tour config changed before record")
	}
	for _, data := range [][]byte{project, seo} {
		if err := writeCompatibilityImmutable(root, "data/surface-config-history/"+sum(data)+".go", data); err != nil {
			return err
		}
	}
	return nil
}

func recordedTourSurfaceConfigProjectionValid(root string, inputs LocaleSurfaceReviewAInputs) bool {
	if inputs.TourConfigProjection == nil {
		return false
	}
	project, err := readCompatibilityFile(root, "data/surface-config-history/"+inputs.ProjectConfigSHA256+".go")
	if err != nil || sum(project) != inputs.ProjectConfigSHA256 {
		return false
	}
	seo, err := readCompatibilityFile(root, "data/surface-config-history/"+inputs.SEOConfigSHA256+".go")
	if err != nil || sum(seo) != inputs.SEOConfigSHA256 {
		return false
	}
	p, err := projectTourConfig(project)
	if err != nil {
		return false
	}
	s, err := projectTourConfig(seo)
	return err == nil && reflect.DeepEqual(*inputs.TourConfigProjection, TourSurfaceConfigProjections{p, s})
}

func surfaceConfigBaselinePath(locale, reviewID string) string {
	return "data/locale-surface-reviews/" + locale + "/" + reviewID + ".config-baseline.json"
}

func RecordTourSurfaceConfigBaseline(root, locale, reviewID string) (string, error) {
	gatePath, err := LocaleSurfaceReviewAGatePath(root, locale, reviewID)
	if err != nil {
		return "", err
	}
	gateRefPath, err := repositoryRelativePath(root, gatePath)
	if err != nil {
		return "", err
	}
	gateData, err := readCompatibilityFile(root, gateRefPath)
	if err != nil {
		return "", err
	}
	var gate LocaleSurfaceReviewAGate
	if err := decodeStrictCourseSourceDescriptionReviewJSON(gateData, &gate); err != nil {
		return "", err
	}
	if err := validateLocaleSurfaceReviewAGate(gate, locale); err != nil {
		return "", err
	}
	if gate.ReviewID != reviewID {
		return "", fmt.Errorf("config baseline gate identity mismatch")
	}
	project, err := readCompatibilityFile(root, "internal/tour/project.go")
	if err != nil {
		return "", err
	}
	seo, err := readCompatibilityFile(root, "internal/tour/seo.go")
	if err != nil {
		return "", err
	}
	if sum(project) != gate.Inputs.ProjectConfigSHA256 || sum(seo) != gate.Inputs.SEOConfigSHA256 {
		return "", fmt.Errorf("cannot prove historical config projection: whole-file identity differs")
	}
	projection, err := currentTourSurfaceConfigProjections(root)
	if err != nil {
		return "", err
	}
	p := GlossaryArchiveReference{"data/surface-config-history/" + sum(project) + ".go", sum(project)}
	s := GlossaryArchiveReference{"data/surface-config-history/" + sum(seo) + ".go", sum(seo)}
	baseline := TourSurfaceConfigBaseline{Schema: "go-learning/tour-surface-config-baseline/v1", Locale: locale, ReviewID: reviewID, Gate: GlossaryArchiveReference{gateRefPath, sum(gateData)}, Project: p, SEO: s, Projection: *projection}
	baseline.Identity = sum(mustJSON(baseline))
	data, err := marshalGlossaryReviewJSON(baseline)
	if err != nil {
		return "", err
	}
	if err := writeCompatibilityImmutable(root, p.Path, project); err != nil {
		return "", err
	}
	if err := writeCompatibilityImmutable(root, s.Path, seo); err != nil {
		return "", err
	}
	path := surfaceConfigBaselinePath(locale, reviewID)
	if err := writeCompatibilityImmutable(root, path, data); err != nil {
		return "", err
	}
	return path, nil
}

func historicalTourSurfaceConfigCompatible(root, locale string, gate LocaleSurfaceReviewAGate, gateData []byte) bool {
	data, err := readCompatibilityFile(root, surfaceConfigBaselinePath(locale, gate.ReviewID))
	if err != nil {
		return false
	}
	var b TourSurfaceConfigBaseline
	if decodeStrictCourseSourceDescriptionReviewJSON(data, &b) != nil {
		return false
	}
	identity := b.Identity
	b.Identity = ""
	wantGate := "data/locale-surface-reviews/" + locale + "/" + gate.ReviewID + ".a-gate.json"
	if b.Schema != "go-learning/tour-surface-config-baseline/v1" || b.Locale != locale || b.ReviewID != gate.ReviewID || b.Gate.Path != wantGate || b.Gate.SHA256 != sum(gateData) || identity != sum(mustJSON(b)) || b.Project.SHA256 != gate.Inputs.ProjectConfigSHA256 || b.SEO.SHA256 != gate.Inputs.SEOConfigSHA256 {
		return false
	}
	if b.Project.Path != "data/surface-config-history/"+b.Project.SHA256+".go" || b.SEO.Path != "data/surface-config-history/"+b.SEO.SHA256+".go" {
		return false
	}
	a, err := readCompatibilityFile(root, b.Project.Path)
	if err != nil || sum(a) != b.Project.SHA256 {
		return false
	}
	s, err := readCompatibilityFile(root, b.SEO.Path)
	if err != nil || sum(s) != b.SEO.SHA256 {
		return false
	}
	p, err := projectTourConfig(a)
	if err != nil {
		return false
	}
	q, err := projectTourConfig(s)
	if err != nil || !reflect.DeepEqual(b.Projection, TourSurfaceConfigProjections{p, q}) {
		return false
	}
	current, err := currentTourSurfaceConfigProjections(root)
	if err != nil {
		return false
	}
	return tourConfigProjectionCompatible(p, current.Project) && tourConfigProjectionCompatible(q, current.SEO)
}

// BootstrapTourSurfaceConfigBaselines preflights the complete requested cohort.
// It records only gates whose historical full bytes still exactly match; other
// historical gates receive no fabricated projection.
func BootstrapTourSurfaceConfigBaselines(root string, locales []string) ([]string, error) {
	if _, err := currentTourSurfaceConfigProjections(root); err != nil {
		return nil, err
	}
	project, err := readCompatibilityFile(root, "internal/tour/project.go")
	if err != nil {
		return nil, err
	}
	seo, err := readCompatibilityFile(root, "internal/tour/seo.go")
	if err != nil {
		return nil, err
	}
	type item struct{ locale, review string }
	plan := []item{}
	seen := map[string]bool{}
	for _, locale := range locales {
		if ValidateLocaleName(locale) != nil || seen[locale] {
			return nil, fmt.Errorf("invalid/duplicate baseline locale")
		}
		seen[locale] = true
		directory := "data/locale-surface-reviews/" + locale
		sentinel, err := compatibilityPath(root, directory+"/.inventory", false)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(filepath.Dir(sentinel))
		if err != nil {
			return nil, err
		}
		count := 0
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".a-gate.json") {
				continue
			}
			data, err := readCompatibilityFile(root, directory+"/"+entry.Name())
			if err != nil {
				return nil, err
			}
			var gate LocaleSurfaceReviewAGate
			if err := decodeStrictCourseSourceDescriptionReviewJSON(data, &gate); err != nil {
				return nil, err
			}
			if err := validateLocaleSurfaceReviewAGate(gate, locale); err != nil {
				return nil, err
			}
			if gate.ReviewID+".a-gate.json" != entry.Name() {
				return nil, fmt.Errorf("gate path mismatch")
			}
			if gate.Inputs.ProjectConfigSHA256 == sum(project) && gate.Inputs.SEOConfigSHA256 == sum(seo) {
				plan = append(plan, item{locale, gate.ReviewID})
				count++
			}
		}
		if count == 0 {
			return nil, fmt.Errorf("%s: no provable historical config baseline", locale)
		}
	}
	paths := []string{}
	for _, item := range plan {
		path, err := RecordTourSurfaceConfigBaseline(root, item.locale, item.review)
		if err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}
