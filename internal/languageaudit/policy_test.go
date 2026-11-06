package languageaudit

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFormalPolicyMultilingualAuthority(t *testing.T) {
	for _, tc := range []struct{ locale, script, end, preferred string }{{"zh-CN", "Han", ".", "。"}, {"ja-JP", "Han", ".", "。"}, {"ko-KR", "Hangul", "", ""}, {"fr-FR", "Latin", "", ""}, {"de-DE", "Latin", "", ""}, {"ru-RU", "Cyrillic", "", ""}, {"ar", "Arabic", "?", "؟"}, {"fa-IR", "Arabic", "?", "؟"}} {
		p := Policy{Locale: tc.locale, Profile: "reviewed-profile-v1", ProseScripts: []string{tc.script}, SentenceTerminators: ".!?。！？؟", ResidualMinWords: 2}
		if tc.end != "" {
			p.SentenceEnd = map[string]string{tc.end: tc.preferred}
		}
		a := Authority{Schema: PolicySchema, Version: 1, Policy: p}
		b, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		got, err := DecodeAuthority(b, tc.locale)
		if err != nil || got.Policy.ProseScripts[0] != tc.script {
			t.Fatal(tc.locale, err)
		}
		if _, err := DecodeAuthority(b, "other"); err == nil {
			t.Fatal("wrong locale policy")
		}
		if _, err := DecodeAuthority([]byte(strings.Replace(string(b), `"profile":`, `"unknown":`, 1)), tc.locale); err == nil {
			t.Fatal("unknown policy field")
		}
	}
}
func TestFormalPolicyRejectsIncompleteAndForgedAuthority(t *testing.T) {
	a := Authority{Schema: PolicySchema, Version: 1, Policy: Policy{Locale: "fr-FR", Profile: "p1", ProseScripts: []string{"Latin"}, ResidualMinWords: 1}}
	b, _ := json.Marshal(a)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(b, &fields)
	var p map[string]any
	_ = json.Unmarshal(fields["policy"], &p)
	delete(p, "spacing")
	fields["policy"], _ = json.Marshal(p)
	bad, _ := json.Marshal(fields)
	if _, err := DecodeAuthority(bad, "fr-FR"); err == nil {
		t.Fatal("omission silently became empty rules")
	}
	a.Policy.ProtectedIdentities = []string{"ordinary prose"}
	if a.Validate("fr-FR") == nil {
		t.Fatal("policy forged identity bypass")
	}
	a.Policy.ProtectedIdentities = nil
	a.Version = 0
	if a.Validate("fr-FR") == nil {
		t.Fatal("unversioned authority")
	}
}
