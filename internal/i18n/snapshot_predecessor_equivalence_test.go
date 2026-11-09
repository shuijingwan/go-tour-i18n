package i18n

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func snapshotPredecessorEquivalenceFixture(t *testing.T) (string, CurrentValidationPredecessor, CurrentValidationPredecessor, map[string][]byte, map[string][]byte) {
	t.Helper()
	root := t.TempDir()
	old := CurrentValidationPredecessor{Schema: "go-learning/current-validation-predecessor-transport/v1", Locale: "zh-CN", UnitID: "unit", Template: "protected template", TemplateSHA256: sum([]byte("protected template")), ImplementationIdentity: sum([]byte("old producer")), CurrentAuthority: sum([]byte("old authority"))}
	old.CurrentSource = GlossaryArchiveReference{Path: "source.txt", SHA256: sum([]byte("source"))}
	old.Dependencies = []GlossaryArchiveReference{old.CurrentSource}
	if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	current := old
	current.ImplementationIdentity = sum([]byte("current producer"))
	current.CurrentAuthority = sum([]byte("current authority"))
	historical, fresh := map[string][]byte{}, map[string][]byte{}
	for _, path := range currentValidationPredecessorImplementationPaths {
		a, b := []byte("historical "+path), []byte("current "+path)
		key := "formal/current-validation-predecessor-implementation/" + path
		historical[key], fresh[key] = a, b
		old.ImplementationInputs = append(old.ImplementationInputs, GlossaryArchiveReference{Path: path, SHA256: sum(a)})
		current.ImplementationInputs = append(current.ImplementationInputs, GlossaryArchiveReference{Path: path, SHA256: sum(b)})
		if err := os.MkdirAll(filepath.Dir(filepath.Join(root, path)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, path), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root, old, current, historical, fresh
}

func TestSnapshotCurrentValidationPredecessorEquivalence(t *testing.T) {
	root, old, current, historical, fresh := snapshotPredecessorEquivalenceFixture(t)
	if _, err := snapshotCurrentValidationPredecessorEquivalent(root, &old, &current, historical, fresh); err != nil {
		t.Fatal(err)
	}
	// Keep the private semantic body closed as the transport evolves.
	semanticType := reflect.TypeOf(snapshotPredecessorSemanticBody{})
	fullType := reflect.TypeOf(CurrentValidationPredecessor{})
	for i := 0; i < fullType.NumField(); i++ {
		f := fullType.Field(i)
		if !f.IsExported() || f.Name == "ImplementationInputs" || f.Name == "ImplementationIdentity" || f.Name == "CurrentAuthority" {
			continue
		}
		if _, ok := semanticType.FieldByName(f.Name); !ok {
			t.Fatal("unclassified predecessor semantic field", f.Name)
		}
	}
	// Each declared semantic field is independently non-negotiable. The explicit
	// production body prevents producer identity from hiding any one of them.
	typ := reflect.TypeOf(snapshotPredecessorSemanticBody{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		t.Run("semantic/"+name, func(t *testing.T) {
			b := current
			v := reflect.ValueOf(&b).Elem().FieldByName(name)
			switch v.Kind() {
			case reflect.String:
				v.SetString(v.String() + "changed")
			case reflect.Bool:
				v.SetBool(!v.Bool())
			case reflect.Slice:
				v.Set(reflect.Append(v, reflect.New(v.Type().Elem()).Elem()))
			case reflect.Struct:
				for j := 0; j < v.NumField(); j++ {
					if v.Field(j).CanSet() && v.Field(j).Kind() == reflect.String {
						v.Field(j).SetString("changed")
						break
					}
				}
			default:
				t.Fatal("test mutation missing", v.Kind())
			}
			if _, err := snapshotCurrentValidationPredecessorEquivalent(root, &old, &b, historical, fresh); err == nil {
				t.Fatal("semantic change accepted", name)
			}
		})
	}
	mutations := map[string]func(*CurrentValidationPredecessor){
		"path":    func(p *CurrentValidationPredecessor) { p.ImplementationInputs[0].Path = "different.go" },
		"missing": func(p *CurrentValidationPredecessor) { p.ImplementationInputs = p.ImplementationInputs[1:] },
		"extra": func(p *CurrentValidationPredecessor) {
			p.ImplementationInputs = append(p.ImplementationInputs, p.ImplementationInputs[0])
		},
		"duplicate": func(p *CurrentValidationPredecessor) { p.ImplementationInputs[1] = p.ImplementationInputs[0] },
		"order": func(p *CurrentValidationPredecessor) {
			p.ImplementationInputs[0], p.ImplementationInputs[1] = p.ImplementationInputs[1], p.ImplementationInputs[0]
		},
		"malformed-sha":   func(p *CurrentValidationPredecessor) { p.ImplementationInputs[0].SHA256 = "invalid" },
		"producer-sha":    func(p *CurrentValidationPredecessor) { p.ImplementationIdentity = "invalid" },
		"authority-sha":   func(p *CurrentValidationPredecessor) { p.CurrentAuthority = "invalid" },
		"dependency-path": func(p *CurrentValidationPredecessor) { p.Dependencies[0].Path = "wrong.txt" },
		"dependency-sha":  func(p *CurrentValidationPredecessor) { p.Dependencies[0].SHA256 = sum([]byte("different")) },
	}
	for name, change := range mutations {
		for _, side := range []string{"historical", "current"} {
			t.Run(name+"/"+side, func(t *testing.T) {
				a, b := old, current
				a.ImplementationInputs = append([]GlossaryArchiveReference{}, a.ImplementationInputs...)
				b.ImplementationInputs = append([]GlossaryArchiveReference{}, b.ImplementationInputs...)
				a.Dependencies = append([]GlossaryArchiveReference{}, a.Dependencies...)
				b.Dependencies = append([]GlossaryArchiveReference{}, b.Dependencies...)
				if side == "historical" {
					change(&a)
				} else {
					change(&b)
				}
				if _, err := snapshotCurrentValidationPredecessorEquivalent(root, &a, &b, historical, fresh); err == nil {
					t.Fatal("invalid binding accepted")
				}
			})
		}
	}
	t.Run("fresh-dependency-reader", func(t *testing.T) {
		if err := os.WriteFile(filepath.Join(root, "source.txt"), []byte("tampered"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := snapshotCurrentValidationPredecessorEquivalent(root, &old, &current, historical, fresh); err == nil {
			t.Fatal("dependency tamper accepted")
		}
	})
}

func TestSnapshotPredecessorInputReconciliation(t *testing.T) {
	for _, mode := range []string{"pass", "missing-old-member", "missing-current-member", "tampered-old-member", "tampered-current-member", "orphan", "current-orphan", "extra-old-transport", "extra-current-transport", "unknown-field", "conflict", "late-semantic-failure"} {
		t.Run(mode, func(t *testing.T) {
			root, a, b, old, current := snapshotPredecessorEquivalenceFixture(t)
			path := "formal/current-validation-predecessor-transport/unit.txt.json"
			old[path], current[path] = mustJSON(a), mustJSON(b)
			key := "formal/current-validation-predecessor-implementation/" + currentValidationPredecessorImplementationPaths[0]
			switch mode {
			case "missing-old-member":
				delete(old, key)
			case "missing-current-member":
				delete(current, key)
			case "tampered-old-member":
				old[key] = []byte("tampered")
			case "tampered-current-member":
				current[key] = []byte("tampered")
			case "current-orphan":
				current["formal/current-validation-predecessor-implementation/extra.go"] = []byte("extra")
			case "orphan":
				old["formal/current-validation-predecessor-implementation/extra.go"] = []byte("extra")
			case "extra-old-transport":
				old[path+".json"] = mustJSON(a)
			case "extra-current-transport":
				current[path+".json"] = mustJSON(b)
			case "unknown-field":
				current[path] = []byte(strings.TrimSuffix(string(mustJSON(b)), "}") + `,"unknown":true}`)
			case "conflict":
				a.ImplementationInputs[0].SHA256 = sum([]byte("different old bytes"))
				old[path+".json"] = mustJSON(a)
				current[path+".json"] = mustJSON(b)
			case "late-semantic-failure":
				old[path+".json"] = mustJSON(a)
				b.Template = "changed"
				current[path+".json"] = mustJSON(b)
			}
			state := snapshotApprovedPass{}
			for p, data := range current {
				state.inputs = append(state.inputs, generationBundleInput{path: p, data: data})
			}
			before := append([]generationBundleInput{}, state.inputs...)
			err := state.reconcilePredecessors(root, old)
			if mode != "pass" {
				if err == nil {
					t.Fatal("invalid inventory accepted")
				}
				if !reflect.DeepEqual(before, state.inputs) {
					t.Fatal("historical substitution occurred before all equivalence checks")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(state.predecessorMembers) != len(currentValidationPredecessorImplementationPaths) {
				t.Fatal("coverage incomplete")
			}
			for _, in := range state.inputs {
				if !bytes.Equal(in.data, old[in.path]) {
					t.Fatal("original historical bytes not retained", in.path)
				}
			}
			if bytes.Equal(current[path], old[path]) {
				t.Fatal("fixture failed to construct distinct current transport")
			}
		})
	}
}

func TestSnapshotPredecessorExactCurrentBoundary(t *testing.T) {
	path := "formal/current-validation-predecessor-implementation/internal/i18n/current_validation_predecessor.go"
	transport := "formal/current-validation-predecessor-transport/unit.article.json"
	original := map[string][]byte{path: []byte("old producer"), transport: []byte("old transport")}
	inputs := []generationBundleInput{{path: path, data: original[path]}, {path: transport, data: original[transport]}}
	if !snapshotPredecessorInputsExact(inputs, original) {
		t.Fatal("exact-current rejected")
	}
	for _, p := range []string{path, transport} {
		changed := append([]generationBundleInput{}, inputs...)
		for i := range changed {
			if changed[i].path == p {
				changed[i].data = []byte("changed")
			}
		}
		if snapshotPredecessorInputsExact(changed, original) {
			t.Fatal("predecessor drift bypassed typed admission", p)
		}
	}
	if snapshotPredecessorInputsExact(inputs[:1], original) || snapshotPredecessorInputsExact(inputs, map[string][]byte{}) {
		t.Fatal("inventory drift bypassed typed admission")
	}
}
