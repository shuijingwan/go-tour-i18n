package sitecontent

import (
	"reflect"
	"testing"
)

func TestPageRenderInputFormalMetadata(t *testing.T) {
	for _, raw := range []string{"<!--{\"Title\":\"正式标题\",\"Template\":true}-->body", "---\ntitle: \"正式标题\"\ntemplate: true\n---\nbody"} {
		body, title, templated, err := PageRenderInput([]byte(raw))
		if err != nil || title != "正式标题" || !templated || string(body) != "body" {
			t.Fatalf("metadata: %q/%q/%v/%v", body, title, templated, err)
		}
	}
	for _, raw := range []string{"<!--{\"Title\":\"a\",\"Unknown\":true}-->body", "---\ntitle: one\ntitle: two\n---\nbody", "<!--{", "---\ntitle: incomplete"} {
		if _, _, _, err := PageRenderInput([]byte(raw)); err == nil {
			t.Fatalf("invalid formal metadata accepted: %s", raw)
		}
	}
}

func TestLearnRenderDataFormalScalars(t *testing.T) {
	want := []map[string]string{{"title": "标题", "content": "第一行 <i>HTML</i> 第二行", "url": "/tour/"}, {"title": "Other", "content": "Body", "url": "https://go.dev/"}}
	for _, raw := range []string{
		"  - title: 标题\n    content: 第一行\n      <i>HTML</i> 第二行\n    url: /tour/\n  - title: Other\n    content: Body\n    url: https://go.dev/\n",
		"- title: 标题\n  content: 第一行\n    <i>HTML</i> 第二行\n  url: /tour/\n- title: Other\n  content: Body\n  url: https://go.dev/\n",
	} {
		got, err := LearnRenderData([]byte(raw))
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Fatalf("folded data: %v/%v", got, err)
		}
	}
	if _, err := LearnRenderData([]byte("- title: Invalid\n  unknown: value\n  url: /tour/\n")); err == nil {
		t.Fatal("unknown data field accepted")
	}
	if _, err := LearnRenderData([]byte("- title: \"Title\"\n  content: 'It''s text'\n  url: /tour/\n")); err == nil {
		t.Fatal("quoted visible scalar must follow formal parser rejection")
	}
}
