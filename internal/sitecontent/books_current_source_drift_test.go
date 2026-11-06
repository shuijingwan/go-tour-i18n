package sitecontent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/contentidentity"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Opt-in diagnosis of an immutable Generation against both production current
// document builders. This test never exports, recovers or creates a closure.
func TestBooksCurrentSourceDriftReadonlyDiagnostic(t *testing.T) {
	if os.Getenv("GO_TOUR_BOOKS_SOURCE_DRIFT_DIAGNOSTIC") != "1" {
		t.Skip("explicit read-only source/input diagnosis")
	}
	root := repositoryRoot(t)
	fingerprint := func() map[string]string {
		t.Helper()
		out := map[string]string{}
		for _, name := range []string{"data/site-shell", "data/glossary-source-corpus.json", "data/quality-check-snapshots", "data/quality-check-reopens", "data/translation-unit-reconciliations", "data/translation-authorizations", "data/locale-language", "data/site-workflow", "data/unified-glossary-reviews", "locales/zh-CN", "_content/learn/books.yaml"} {
			if err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(name)), func(path string, e fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if e.IsDir() {
					return nil
				}
				b, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				out[path] = digest(b)
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		}
		return out
	}
	before := fingerprint()
	defer func() {
		if !reflect.DeepEqual(before, fingerprint()) {
			t.Error("FORMAL_ARTIFACT_MUTATION detected")
		}
	}()
	done := contentidentity.BeginMemoization()
	defer done()
	ref := Reference{"data/locale-language/zh-CN/structured-generation/7caebba33580782e5837719efa23a8d1cf4d60f51de84fa3e5c56c8816d25cb5.json", "3bdea5f48d19a48c69964d515d87117a095015d9aa6a33e2ad4837b59a10c8c3"}
	r, bundle, inputs, err := readStructuredGeneration(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	var frozen StructuredAsset
	for _, a := range r.Plan.Assets {
		if a.ID == "source:_content/learn/books.yaml" {
			frozen = a
		}
	}
	if frozen.Document == nil {
		t.Fatal("missing frozen books Document")
	}
	var output StructuredTarget
	for _, target := range r.Output.Outputs {
		if target.Path == frozen.TargetPath {
			output = target
		}
	}
	g, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	docs, raw, err := currentStructuredDocuments(root, g, []StructuredAsset{frozen})
	if err != nil {
		t.Fatal(err)
	}
	currentDoc := docs[frozen.Document.Path]
	packageDocs, packageRaw, err := CurrentPackageDocuments(root, g, frozen.Package)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, d := range packageDocs {
		if d.Path == currentDoc.Path {
			found = true
			if !reflect.DeepEqual(d, currentDoc) || !bytes.Equal(raw[d.Path], packageRaw[d.Path]) {
				t.Fatal("current checker and canonical package builder disagree")
			}
		}
	}
	if !found {
		t.Fatal("missing canonical package source")
	}
	corpus, err := i18n.LoadUnifiedGlossaryCorpus(root)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := structuredContributorGroups(corpus.Contributors)
	if err != nil {
		t.Fatal(err)
	}
	cs := groups[frozen.ID]
	if len(cs) == 0 {
		t.Fatal("unregistered structured contributor")
	}
	// Reproduce the source projection in PlanStructuredAssets, without invoking
	// a lifecycle writer or pretending that this is a new assessed Plan.
	current := frozen
	current.Package, current.SourceSHA, current.Document = cs[0].Package, identity(cs), &currentDoc
	current.SourcePaths = []string{}
	for _, c := range cs {
		current.SourcePaths = append(current.SourcePaths, c.Path)
	}
	current.SourcePaths = uniqueStrings(current.SourcePaths)
	current.TargetPath = "locales/" + r.Plan.Locale + "/structured/" + cs[0].Path
	actual := mustDiagnosticRead(t, root, currentDoc.Path)
	// Before closure the target need not be materialized. Its accepted bytes
	// come from the exact baseline Generation, not a guessed filesystem file.
	baseline, _, baselineInputs, err := readStructuredGeneration(root, frozen.Assessment.Generation)
	if err != nil {
		t.Fatal(err)
	}
	baselineGlossaryBytes := baselineInputs["inputs/locales/"+r.Plan.Locale+"/glossary.yaml"]
	if digest(baselineGlossaryBytes) != baseline.Plan.GlossarySHA {
		t.Fatal("baseline glossary SHA mismatch")
	}
	baselineGlossary, err := i18n.ParseGlossaryBytes(r.Plan.Locale, baselineGlossaryBytes)
	if err != nil {
		t.Fatal(err)
	}
	var acceptedTarget []byte
	for _, a := range baseline.Plan.Assets {
		if a.ID != frozen.ID {
			continue
		}
		for _, target := range baseline.Output.Outputs {
			if target.Path != a.TargetPath {
				continue
			}
			values, err := structuredTargetValues(a, target, baselineGlossary)
			if err != nil {
				t.Fatal(err)
			}
			acceptedTarget, err = reconstructFrozenStructuredRepresentation(*a.Document, baselineInputs["source/"+a.Document.Path], values)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if len(acceptedTarget) == 0 || digest(acceptedTarget) != frozen.Assessment.TargetSHA {
		t.Fatal("accepted baseline target SHA mismatch")
	}
	current.TargetSHA = digest(acceptedTarget)
	_, targetFileErr := readRegular(root, current.TargetPath)
	frozenRaw := inputs["source/"+currentDoc.Path]
	if digest(frozenRaw) != frozen.Document.SourceSHA || digest(raw[currentDoc.Path]) != currentDoc.SourceSHA {
		t.Fatal("raw source SHA binding failed")
	}
	fieldDiff := func(a, b any) []assessmentFieldDifference {
		av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
		out := []assessmentFieldDifference{}
		for i := 0; i < av.NumField(); i++ {
			if !reflect.DeepEqual(av.Field(i).Interface(), bv.Field(i).Interface()) {
				x, _ := json.Marshal(av.Field(i).Interface())
				y, _ := json.Marshal(bv.Field(i).Interface())
				out = append(out, assessmentFieldDifference{strings.Split(av.Type().Field(i).Tag.Get("json"), ",")[0], x, y})
			}
		}
		return out
	}
	oldUnits, newUnits := map[string]Unit{}, map[string]Unit{}
	for _, u := range frozen.Document.Units {
		oldUnits[u.ID] = u
	}
	for _, u := range currentDoc.Units {
		newUnits[u.ID] = u
	}
	// One data scalar is one source atom. Locator context excludes the derived
	// slot hash/occurrence; coordinates and complete protection metadata remain
	// exact. No text-only or index-only predecessor guesses are accepted.
	context := func(u Unit) string {
		return strings.SplitN(strings.TrimPrefix(u.Locator, currentDoc.Path+"#"), ":", 2)[0]
	}
	atom := func(u Unit) string {
		b, _ := json.Marshal(struct {
			Path, Field, Source string
			Start, End          int
			Protected           []ProtectedToken
		}{currentDoc.Path, context(u), u.Source, u.Start, u.End, u.Protected})
		return string(b)
	}
	oldAtoms, newAtoms := map[string]Unit{}, map[string]Unit{}
	for _, u := range oldUnits {
		oldAtoms[atom(u)] = u
	}
	for _, u := range newUnits {
		newAtoms[atom(u)] = u
	}
	added, removed, changed := []string{}, []string{}, []string{}
	addedAtoms, removedAtoms := []Unit{}, []Unit{}
	details := []map[string]any{}
	for _, u := range currentDoc.Units {
		old, exists := oldUnits[u.ID]
		if !exists {
			added = append(added, u.ID)
		} else if !reflect.DeepEqual(old, u) {
			changed = append(changed, u.ID)
		}
		if _, ok := oldAtoms[atom(u)]; !ok {
			addedAtoms = append(addedAtoms, u)
		}
		if !exists || !reflect.DeepEqual(old, u) {
			matches := []string{}
			for _, candidate := range frozen.Document.Units {
				if candidate.Start == u.Start && candidate.End == u.End && context(candidate) == context(u) {
					matches = append(matches, candidate.ID)
				}
			}
			legacy, locked, err := protectBlockLegacy(raw[currentDoc.Path][u.Start:u.End], context(u))
			if err != nil {
				t.Fatal(err)
			}
			item := map[string]any{"current_unit": u, "frozen_unit": nil, "frozen_coordinate_matches": matches, "unique_predecessor": len(matches) == 1, "legacy_protected_source": legacy, "legacy_protection": locked, "legacy_has_language": hasLanguage(legacy), "current_has_language": hasLanguage(u.Source)}
			if exists {
				item["frozen_unit"], item["field_differences"] = old, fieldDiff(old, u)
			}
			details = append(details, item)
		}
	}
	for _, u := range frozen.Document.Units {
		if _, ok := newUnits[u.ID]; !ok {
			removed = append(removed, u.ID)
		}
		if _, ok := newAtoms[atom(u)]; !ok {
			removedAtoms = append(removedAtoms, u)
		}
	}
	values := map[string]string{}
	for _, slot := range output.Slots {
		if _, exists := values[slot.ID]; exists {
			t.Fatal("duplicate frozen target slot")
		}
		values[slot.ID] = slot.Text
	}
	missing, extra := []string{}, []string{}
	coverage := 0
	glossary, err := i18n.LoadGlossary(root, r.Plan.Locale)
	if err != nil {
		t.Fatal(err)
	}
	frozenGlossary, err := i18n.ParseGlossaryBytes(r.Plan.Locale, inputs["inputs/locales/"+r.Plan.Locale+"/glossary.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	frozenValues, err := structuredTargetValues(frozen, output, frozenGlossary)
	if err != nil {
		t.Fatal(err)
	}
	frozenTarget, err := reconstructFrozenStructuredRepresentation(*frozen.Document, frozenRaw, frozenValues)
	if err != nil {
		t.Fatal("frozen source/Document/output reconstruction", err)
	}
	glossaryFailures := map[string]string{}
	for _, u := range currentDoc.Units {
		if target, ok := values[u.ID]; ok {
			coverage++
			if err := i18n.ValidateParsedGlossary(u.Source, target, glossary); err != nil {
				glossaryFailures[u.ID] = err.Error()
			}
		} else {
			missing = append(missing, u.ID)
			glossaryFailures[u.ID] = "missing formal Generation target slot; glossary validation cannot complete"
		}
	}
	for id := range values {
		if _, ok := newUnits[id]; !ok {
			extra = append(extra, id)
		}
	}
	sort.Strings(extra)
	_, reconstructionErr := reconstructStructuredRepresentation(currentDoc, raw[currentDoc.Path], values)
	plan := r.Plan
	plan.Assets, plan.Expected = []StructuredAsset{current}, []string{current.TargetPath}
	_, validationErr := validateStructuredOutput(root, plan, StructuredOutput{Schema: r.Output.Schema, BundleSHA: r.Output.BundleSHA, Outputs: []StructuredTarget{output}})
	// Isolate the real Document delta in the full-Plan recovery comparator.
	// No synthetic Bundle is saved or submitted as evidence.
	var currentBundle StructuredBundle
	b, _ := json.Marshal(bundle)
	if err := json.Unmarshal(b, &currentBundle); err != nil {
		t.Fatal(err)
	}
	for i := range currentBundle.Plan.Assets {
		if currentBundle.Plan.Assets[i].ID == current.ID {
			currentBundle.Plan.Assets[i] = current
		}
	}
	recoveryErr := compatibleStructuredBundles(*bundle, inputs, currentBundle, inputs)
	semanticChanged := len(addedAtoms) != 0 || len(removedAtoms) != 0
	report := map[string]any{
		"generation": ref, "frozen_asset": frozen, "current_source_projection": current,
		"asset_field_differences": fieldDiff(frozen, current), "document_field_differences": fieldDiff(*frozen.Document, currentDoc),
		"actual_file_sha256": digest(actual), "frozen_raw_source_sha256": digest(frozenRaw), "current_raw_source_sha256": digest(raw[currentDoc.Path]),
		"raw_source_bytes_changed": !bytes.Equal(frozenRaw, actual), "current_builders_equal": true,
		"frozen_dependency_count": len(frozen.Document.Dependencies), "current_dependency_count": len(currentDoc.Dependencies),
		"frozen_unit_count": len(oldUnits), "current_unit_count": len(newUnits),
		"frozen_semantic_atom_count": len(oldAtoms), "current_semantic_atom_count": len(newAtoms), "semantic_atom_set_changed": semanticChanged,
		"added_unit_ids": added, "removed_unit_ids": removed, "changed_unit_ids": changed, "unit_details": details,
		"added_semantic_atoms": addedAtoms, "removed_semantic_atoms": removedAtoms,
		"frozen_output": output, "current_accepted_target_sha256": digest(acceptedTarget), "filesystem_target_status": fmt.Sprint(targetFileErr),
		"frozen_output_reconstructed_sha256": digest(frozenTarget), "frozen_representation_verified": true,
		"current_unit_target_coverage": fmt.Sprintf("%d/%d", coverage, len(newUnits)), "missing_current_target_unit_ids": missing, "extra_frozen_target_unit_ids": extra,
		"current_glossary_validation_failures": glossaryFailures, "current_reconstruction_error": fmt.Sprint(reconstructionErr), "current_validation_error": fmt.Sprint(validationErr),
		"structured_recover_eligible": recoveryErr == nil, "structured_recover_error": fmt.Sprint(recoveryErr),
		"semantic_atom_definition": "data scalar: exact source text, raw source coordinates, source path, field context, complete protected token role/pair/binding metadata; excludes derived unit ID/hash",
		"generation_input_note":    "Raw source bytes are transported as context; formal localizable inputs and permitted output slots are the frozen Document.Units. Context presence alone does not authorize additional output slots.",
	}
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("/tmp/go-tour-books-current-source-drift.json", append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	t.Logf("raw_changed=%t units=%d/%d semantic_atoms=%d/%d added=%v removed=%v changed=%v coverage=%d/%d reconstruction=%v validation=%v recovery=%v", !bytes.Equal(frozenRaw, actual), len(oldUnits), len(newUnits), len(oldAtoms), len(newAtoms), added, removed, changed, coverage, len(newUnits), reconstructionErr, validationErr, recoveryErr)
	for _, u := range addedAtoms {
		t.Logf("ADDED_SEMANTIC_ATOM %s source=%q locator=%s coordinates=%d:%d", u.ID, u.Source, u.Locator, u.Start, u.End)
	}
	if semanticChanged && (reconstructionErr == nil || validationErr == nil || recoveryErr == nil) {
		t.Error("expanded semantic inputs must remain fail closed for frozen output/recovery")
	}
}
