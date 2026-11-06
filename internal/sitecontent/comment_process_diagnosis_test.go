package sitecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// Explicit readonly diagnosis of imported bytes; no process/recovery is run.
func TestCommentProcessImportedBodyDiagnosis(t *testing.T) {
	out := os.Getenv("GO_TOUR_COMMENT_PROCESS_DIAGNOSE")
	if out == "" {
		t.Skip("explicit diagnosis output required")
	}
	root := repositoryRoot(t)
	const id = "a69694fb9c89d47440249dfd760d315586297f634d199b13532ac95cbced201f"
	const path = "_content/doc/security/vuln/database.md"
	z, err := os.ReadFile("/tmp/zh-CN-chatgpt-zh-CN-051-generation-comment-transport-v2.zip")
	if err != nil {
		t.Fatal(err)
	}
	if digest(z) != "6cd5e647196a1e46df4092073f4cd38716acd7ff426b5106d5f92883e4d05bb6" {
		t.Fatal("Bundle bytes changed")
	}
	files, err := i18n.ReadTransportBundle(z, 8192, 64<<20)
	if err != nil {
		t.Fatal(err)
	}
	var p i18n.CommentRevisionTransport
	if err := json.Unmarshal(files["formal/comment-revision-transport/"+id+".article.json"], &p); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	d, err := ParseWorkflowDocument(Source{Path: path, Kind: "page", Route: "/doc/security/vuln/database", SHA256: digest(source)}, source, 1)
	if err != nil {
		t.Fatal(err)
	}
	if d.Units[0].ID != id || digest(source) != p.SourceSHA256 {
		t.Fatal("source/Unit drift")
	}
	a := PageAdapter{Document: *d, Source: source}
	candidate := files["formal/candidates/"+id+".article"]
	bodies, err := a.CommentRevisionBodies(candidate)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "data/retranslation-runs/zh-CN/chatgpt-zh-CN-051/raw-responses/"+id+".article"))
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for i, s := range p.Slots {
		var adapter i18n.TranslationUnitAdapter
		for _, b := range bodies {
			if b.Start == s.Start && b.End == s.End {
				adapter = b.Adapter
			}
		}
		if adapter == nil {
			t.Fatal("missing body coordinate")
		}
		start := strings.Index(string(raw), p.Boundaries[i]) + len(p.Boundaries[i])
		end := strings.Index(string(raw), p.Boundaries[i+1])
		text := string(raw[start:end])
		for j, token := range s.Tokens {
			text = strings.ReplaceAll(text, token, s.OriginalTokens[j])
		}
		_, restoreErr := adapter.Restore(text)
		plain := text
		for _, token := range s.OriginalTokens {
			plain = strings.ReplaceAll(plain, token, "")
		}
		_, added, err := protectBlockMode([]byte(plain), "teaching-comment", true)
		if err != nil {
			t.Fatal(err)
		}
		row := map[string]any{"slot_id": s.SlotID, "source_body": string(candidate[s.Start:s.End]), "target_protected_body": text, "added_lexical_tokens": added, "restore": "PASS"}
		if restoreErr != nil {
			row["restore"] = restoreErr.Error()
			t.Logf("%s: %v; added=%+v", s.SlotID, restoreErr, added)
		}
		rows = append(rows, row)
	}
	b, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(out, append(b, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
