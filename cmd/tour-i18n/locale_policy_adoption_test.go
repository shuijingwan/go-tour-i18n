package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func adoptionTestSHA(b []byte) string { s := sha256.Sum256(b); return hex.EncodeToString(s[:]) }
func adoptionTestWrite(t *testing.T, root, path string, b []byte) {
	t.Helper()
	p := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0640); err != nil {
		t.Fatal(err)
	}
}
func adoptionTestRead(t *testing.T, root, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func adoptionTestFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	adoptionTestWrite(t, root, "locales/fr-FR/locale.json", []byte("{\n  \"locale\": \"fr-FR\", \"language_name\": \"Français\",\n  \"english_name\": \"French\", \"html_lang\": \"fr-FR\",\n  \"phase\": \"scaffold\", \"translation_unit\": \"present.Section\",\n  \"default_language\": false\n}\n"))
	b, err := json.MarshalIndent(testLocaleInitLanguagePolicy("fr-FR"), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	adoptionTestWrite(t, root, "engineering/policy.json", append(b, '\n'))
	for _, p := range []string{"locales/fr-FR/status.tsv", "locales/fr-FR/glossary.yaml", "locales/fr-FR/candidates/example.article", "data/surface/old.json", "production/identity.json"} {
		adoptionTestWrite(t, root, p, []byte("frozen sentinel\n"))
	}
	return root
}
func adoptionTestInventory(t *testing.T, root string) map[string]string {
	t.Helper()
	m := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		r, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		if d.Type()&os.ModeSymlink != 0 {
			dest, e := os.Readlink(p)
			if e != nil {
				return e
			}
			m[filepath.ToSlash(r)] = "symlink:" + dest
			return nil
		}
		b, e := os.ReadFile(p)
		if e == nil {
			m[filepath.ToSlash(r)] = adoptionTestSHA(b)
		}
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func adoptionTestArgs() []string {
	return []string{"--locale", "fr-FR", "--input", "engineering/policy.json"}
}
func adoptionTestCall(t *testing.T, root string, args []string) i18n.LocalePolicyAdoptionResult {
	t.Helper()
	var out bytes.Buffer
	if err := localeLanguagePolicyAdoptionCommand(root, args, &out); err != nil {
		t.Fatal(err)
	}
	var r i18n.LocalePolicyAdoptionResult
	d := json.NewDecoder(bytes.NewReader(out.Bytes()))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	expected := []string{"schema", "locale", "policy_schema", "policy_version", "input_policy_sha256", "prior_sha256", "result_sha256", "already_current", "applied", "mutation"}
	if len(fields) != len(expected) {
		t.Fatal("stdout fields drift", out.String())
	}
	for _, key := range expected {
		if fields[key] == nil {
			t.Fatal("stdout field missing", key)
		}
	}
	if r.Schema != i18n.LocalePolicyAdoptionResultSchema {
		t.Fatal("stdout schema drift", r)
	}
	return r
}
func adoptionTestApplyArgs(p i18n.LocalePolicyAdoptionResult) []string {
	return append(adoptionTestArgs(), "--apply", "--prior-sha256", p.PriorSHA256, "--input-sha256", p.InputPolicySHA256, "--result-sha256", p.ResultSHA256)
}

func TestLocaleLanguagePolicyAdoptionCLIContract(t *testing.T) {
	root := adoptionTestFixture(t)
	before := adoptionTestInventory(t, root)
	prior := adoptionTestRead(t, root, "locales/fr-FR/locale.json")
	p := adoptionTestCall(t, root, adoptionTestArgs())
	if p.Locale != "fr-FR" || p.PolicyVersion != 1 || p.PolicySchema != "go-learning/locale-language-audit-policy/v1" || p.PriorSHA256 != adoptionTestSHA(prior) || p.InputPolicySHA256 != adoptionTestSHA(adoptionTestRead(t, root, "engineering/policy.json")) || p.ResultSHA256 == p.PriorSHA256 || p.AlreadyCurrent || p.Applied || p.Mutation != "none" {
		t.Fatal("preflight contract", p)
	}
	if !reflect.DeepEqual(before, adoptionTestInventory(t, root)) {
		t.Fatal("preflight mutated fixture")
	}
	applied := adoptionTestCall(t, root, adoptionTestApplyArgs(p))
	saved := adoptionTestRead(t, root, "locales/fr-FR/locale.json")
	if !applied.Applied || applied.AlreadyCurrent || applied.Mutation != "known" || applied.ResultSHA256 != p.ResultSHA256 || adoptionTestSHA(saved) != p.ResultSHA256 {
		t.Fatal("apply contract", applied)
	}
	if _, _, err := i18n.LoadLanguageAuditPolicy(root, "fr-FR", nil); err != nil {
		t.Fatalf("adopted temporary authority rejected by current policy loader: %v", err)
	}
	// Remove only the newly inserted member and recover every original byte.
	marker := bytes.LastIndex(saved, []byte(",\"language_audit\":"))
	if marker < 0 {
		t.Fatal("missing inserted member")
	}
	if !bytes.Equal(saved[:marker], prior[:len(prior)-2]) {
		t.Fatal("pre-existing metadata prefix changed")
	}
	if !bytes.Equal(saved[len(saved)-2:], prior[len(prior)-2:]) {
		t.Fatal("pre-existing metadata suffix changed")
	}
	var oldFields, newFields map[string]json.RawMessage
	if err := json.Unmarshal(prior, &oldFields); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(saved, &newFields); err != nil {
		t.Fatal(err)
	}
	delete(newFields, "language_audit")
	if !reflect.DeepEqual(oldFields, newFields) {
		t.Fatal("unrelated metadata fields changed")
	}
	after := adoptionTestInventory(t, root)
	delete(before, "locales/fr-FR/locale.json")
	delete(after, "locales/fr-FR/locale.json")
	if !reflect.DeepEqual(before, after) {
		t.Fatal("unrelated path/bytes mutation", after)
	}
	st, err := os.Stat(filepath.Join(root, "locales/fr-FR/locale.json"))
	if err != nil || st.Mode().Perm() != 0640 {
		t.Fatal("mode changed", st, err)
	}
	current := adoptionTestCall(t, root, adoptionTestArgs())
	if !current.AlreadyCurrent || current.Applied || current.Mutation != "none" || current.PriorSHA256 != p.ResultSHA256 || current.ResultSHA256 != p.ResultSHA256 {
		t.Fatal("observed already-current contract", current)
	}
	replay := adoptionTestCall(t, root, adoptionTestApplyArgs(current))
	if !replay.AlreadyCurrent || replay.Applied || replay.Mutation != "none" {
		t.Fatal("current CAS no-op", replay)
	}
	var out bytes.Buffer
	if err := localeLanguagePolicyAdoptionCommand(root, adoptionTestApplyArgs(p), &out); err == nil || out.Len() != 0 {
		t.Fatal("stale prior accepted on no-op")
	}
	if !bytes.Equal(saved, adoptionTestRead(t, root, "locales/fr-FR/locale.json")) {
		t.Fatal("replay mutated metadata")
	}
	// Whitespace/key formatting in a semantically identical policy is a no-op;
	// its exact input SHA is nevertheless different and still pinned on apply.
	var policy any
	if err := json.Unmarshal(adoptionTestRead(t, root, "engineering/policy.json"), &policy); err != nil {
		t.Fatal(err)
	}
	compact, _ := json.Marshal(policy)
	adoptionTestWrite(t, root, "engineering/policy.json", compact)
	same := adoptionTestCall(t, root, adoptionTestArgs())
	if !same.AlreadyCurrent || same.ResultSHA256 != p.ResultSHA256 {
		t.Fatal("semantic exact policy rejected", same)
	}
}

func TestLocaleLanguagePolicyAdoptionCLIRejectsInvalid(t *testing.T) {
	for _, name := range []string{"missing-locale", "missing-input", "absolute", "traversal", "internal-traversal", "symlink", "parent-symlink", "directory", "malformed", "duplicate-key", "incomplete", "wrong-schema", "wrong-version", "locale-mismatch", "metadata-absent", "metadata-malformed", "metadata-invalid-identity", "metadata-null-policy", "metadata-symlink", "different-policy", "unknown-flag", "positional", "duplicate-flag", "apply-missing-pins", "preflight-pins", "preflight-empty-pins"} {
		t.Run(name, func(t *testing.T) {
			root := adoptionTestFixture(t)
			args := adoptionTestArgs()
			policy := testLocaleInitLanguagePolicy("fr-FR")
			switch name {
			case "missing-locale":
				args = []string{"--input", "engineering/policy.json"}
			case "missing-input":
				args = []string{"--locale", "fr-FR"}
			case "absolute":
				args[3] = filepath.Join(root, "engineering/policy.json")
			case "traversal":
				args[3] = "../policy.json"
			case "internal-traversal":
				args[3] = "engineering/../engineering/policy.json"
			case "symlink":
				if err := os.Symlink("policy.json", filepath.Join(root, "engineering/link.json")); err != nil {
					t.Fatal(err)
				}
				args[3] = "engineering/link.json"
			case "parent-symlink":
				if err := os.Symlink("engineering", filepath.Join(root, "alias")); err != nil {
					t.Fatal(err)
				}
				args[3] = "alias/policy.json"
			case "directory":
				args[3] = "engineering"
			case "malformed":
				adoptionTestWrite(t, root, args[3], []byte("{"))
			case "duplicate-key":
				b := adoptionTestRead(t, root, args[3])
				b = bytes.Replace(b, []byte("\"policy_version\": 1"), []byte("\"policy_version\": 1,\"policy_version\": 1"), 1)
				adoptionTestWrite(t, root, args[3], b)
			case "incomplete":
				adoptionTestWrite(t, root, args[3], []byte(`{"schema":"go-learning/locale-language-audit-policy/v1","policy_version":1,"policy":{"locale":"fr-FR"}}`))
			case "wrong-schema":
				policy.Schema = "wrong"
			case "wrong-version":
				policy.Version = 2
			case "locale-mismatch":
				policy.Policy.Locale = "de-DE"
			case "metadata-absent":
				if err := os.Remove(filepath.Join(root, "locales/fr-FR/locale.json")); err != nil {
					t.Fatal(err)
				}
			case "metadata-malformed":
				adoptionTestWrite(t, root, "locales/fr-FR/locale.json", []byte("{"))
			case "metadata-invalid-identity":
				adoptionTestWrite(t, root, "locales/fr-FR/locale.json", []byte(`{"locale":"de-DE"}`))
			case "metadata-null-policy":
				b := adoptionTestRead(t, root, "locales/fr-FR/locale.json")
				adoptionTestWrite(t, root, "locales/fr-FR/locale.json", bytes.Replace(b, []byte("false"), []byte("false,\"language_audit\":null"), 1))
			case "metadata-symlink":
				path := filepath.Join(root, "locales/fr-FR/locale.json")
				if err := os.Rename(path, path+".original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("locale.json.original", path); err != nil {
					t.Fatal(err)
				}
			case "different-policy":
				p := adoptionTestCall(t, root, args)
				adoptionTestCall(t, root, adoptionTestApplyArgs(p))
				policy.Policy.Profile = "different-profile"
			case "unknown-flag":
				args = append(args, "--force")
			case "positional":
				args = append(args, "extra")
			case "duplicate-flag":
				args = append(args, "--locale", "fr-FR")
			case "apply-missing-pins":
				args = append(args, "--apply")
			case "preflight-pins":
				args = append(args, "--prior-sha256", strings.Repeat("a", 64))
			case "preflight-empty-pins":
				args = append(args, "--input-sha256", "")
			}
			if name == "wrong-schema" || name == "wrong-version" || name == "locale-mismatch" || name == "different-policy" {
				b, _ := json.Marshal(policy)
				adoptionTestWrite(t, root, "engineering/policy.json", b)
			}
			before := adoptionTestInventory(t, root)
			var out bytes.Buffer
			err := localeLanguagePolicyAdoptionCommand(root, args, &out)
			if err == nil || out.Len() != 0 || !strings.Contains(err.Error(), "mutation=none") {
				t.Fatal("invalid contract accepted", err, out.String())
			}
			if !reflect.DeepEqual(before, adoptionTestInventory(t, root)) {
				t.Fatal("failure mutated fixture")
			}
		})
	}
}

func TestLocaleLanguagePolicyAdoptionCLICAS(t *testing.T) {
	for _, name := range []string{"wrong-prior", "wrong-result", "input-changed", "metadata-changed"} {
		t.Run(name, func(t *testing.T) {
			root := adoptionTestFixture(t)
			p := adoptionTestCall(t, root, adoptionTestArgs())
			args := adoptionTestApplyArgs(p)
			switch name {
			case "wrong-prior":
				args[6] = strings.Repeat("a", 64)
			case "wrong-result":
				args[10] = strings.Repeat("a", 64)
			case "input-changed":
				path := "engineering/policy.json"
				adoptionTestWrite(t, root, path, append(adoptionTestRead(t, root, path), '\n'))
			case "metadata-changed":
				path := "locales/fr-FR/locale.json"
				adoptionTestWrite(t, root, path, append(adoptionTestRead(t, root, path), '\n'))
			}
			before := adoptionTestInventory(t, root)
			var out bytes.Buffer
			if err := localeLanguagePolicyAdoptionCommand(root, args, &out); err == nil || out.Len() != 0 {
				t.Fatal("CAS drift accepted", err)
			}
			if !reflect.DeepEqual(before, adoptionTestInventory(t, root)) {
				t.Fatal("CAS failure mutated fixture")
			}
		})
	}
}

func TestLocaleLanguagePolicyAdoptionCLIDispatchBeforeCatalog(t *testing.T) {
	root := adoptionTestFixture(t)
	t.Chdir(root)
	output := captureStdout(t, func() error { return run(append([]string{"locale", "language-policy-adopt"}, adoptionTestArgs()...)) })
	var r i18n.LocalePolicyAdoptionResult
	if err := json.Unmarshal(output, &r); err != nil || r.Applied || r.Mutation != "none" {
		t.Fatal("dispatch requires unrelated source catalog or changed stdout", string(output), err)
	}
}
