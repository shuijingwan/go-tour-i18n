package website_test

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"golang.org/x/tools/present"
)

var expectedArticles = []string{
	"basics.article",
	"concurrency.article",
	"flowcontrol.article",
	"generics.article",
	"methods.article",
	"moretypes.article",
	"welcome.article",
}

func TestTourStructure(t *testing.T) {
	articles, err := filepath.Glob("_content/tour/*.article")
	if err != nil {
		t.Fatal(err)
	}
	for i := range articles {
		articles[i] = filepath.Base(articles[i])
	}
	if !reflect.DeepEqual(articles, expectedArticles) {
		t.Fatalf("top-level articles = %v, want %v", articles, expectedArticles)
	}

	present.PlayEnabled = true
	var pages int
	plays := make(map[string]bool)
	var images []string
	for _, name := range articles {
		path := filepath.Join("_content", "tour", name)
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		ctx := &present.Context{ReadFile: func(name string) ([]byte, error) {
			return os.ReadFile(filepath.Join("_content", "tour", filepath.FromSlash(name)))
		}}
		doc, err := ctx.Parse(f, name, 0)
		f.Close()
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		pages += len(doc.Sections)
		lessonDir := strings.TrimSuffix(name, ".article")
		for _, section := range doc.Sections {
			collectDirectives(t, section, lessonDir, plays, &images)
		}
	}
	if pages != 101 {
		t.Fatalf("standalone pages = %d, want 101", pages)
	}
	if len(plays) != 92 {
		t.Fatalf("unique .play paths = %d, want 92", len(plays))
	}
	if len(images) != 1 {
		t.Fatalf(".image references = %d, want 1", len(images))
	}
	imageURL := strings.TrimPrefix(images[0], "/tour/")
	if filepath.IsAbs(imageURL) || imageURL == ".." || strings.HasPrefix(imageURL, "../") || strings.Contains(imageURL, "/../") {
		t.Fatalf("unsafe .image path %q", images[0])
	}
	imagePath := filepath.Join("_content", "tour", filepath.FromSlash(imageURL))
	if _, err := os.Stat(imagePath); err != nil {
		t.Fatalf("image %q: %v", images[0], err)
	}

	welcome, err := os.ReadFile("_content/tour/welcome.article")
	if err != nil {
		t.Fatal(err)
	}
	conditional := regexp.MustCompile(`(?m)^#appengine: \*`).FindAll(welcome, -1)
	if len(conditional) != 2 {
		t.Fatalf("#appengine conditional pages = %d, want 2", len(conditional))
	}

	values, err := os.ReadFile("_content/tour/static/js/values.js")
	if err != nil {
		t.Fatal(err)
	}
	lessonRE := regexp.MustCompile(`'lessons': \[([^]]+)\]`)
	nameRE := regexp.MustCompile(`'([^']+)'`)
	var order []string
	for _, group := range lessonRE.FindAllSubmatch(values, -1) {
		for _, match := range nameRE.FindAllSubmatch(group[1], -1) {
			order = append(order, string(match[1]))
		}
	}
	wantOrder := []string{"welcome", "basics", "flowcontrol", "moretypes", "methods", "generics", "concurrency"}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("values.js lesson order = %v, want %v", order, wantOrder)
	}
}

func collectDirectives(t *testing.T, section present.Section, lessonDir string, plays map[string]bool, images *[]string) {
	t.Helper()
	for _, elem := range section.Elem {
		switch elem := elem.(type) {
		case present.Code:
			if !elem.Play {
				continue
			}
			name := filepath.ToSlash(filepath.Join(lessonDir, elem.FileName))
			if filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") || strings.Contains(name, "/../") {
				t.Errorf("unsafe .play path %q", name)
				continue
			}
			if plays[name] {
				t.Errorf("duplicate .play path %q", name)
			}
			plays[name] = true
			if _, err := os.Stat(filepath.Join("_content", "tour", filepath.FromSlash(name))); err != nil {
				t.Errorf(".play %q: %v", name, err)
			}
		case present.Image:
			*images = append(*images, elem.URL)
		case present.Section:
			collectDirectives(t, elem, lessonDir, plays, images)
		}
	}
}
