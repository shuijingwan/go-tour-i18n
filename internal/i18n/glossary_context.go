package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/tour/ui"
)

type GlossaryCompatibilityContext struct {
	Scope      string                     `json:"scope"`
	Parser     string                     `json:"parser"`
	Source     string                     `json:"source"`
	Target     string                     `json:"target"`
	References []GlossaryArchiveReference `json:"references"`
	Identity   string                     `json:"context_identity_sha256"`
}

func compatibilityCatalog(root string) (*Catalog, error) {
	current, err := BuildSourceCatalog(root)
	if err != nil {
		return nil, err
	}
	if err := CheckCatalogFiles(root, current); err != nil {
		return nil, err
	}
	c, err := ReadCatalog(root)
	if err != nil {
		return nil, err
	}
	if err := HydrateCatalogSources(c, current); err != nil {
		return nil, err
	}
	return c, nil
}

// Inventory does not call downstream freshness gates (which consume this
// inventory). It binds real bytes even when an artifact needs revision.
func currentGlossaryCompatibilityContexts(root, locale string, catalog *Catalog) ([]GlossaryCompatibilityContext, error) {
	if catalog == nil {
		var err error
		catalog, err = compatibilityCatalog(root)
		if err != nil {
			return nil, err
		}
	}
	contexts := []GlossaryCompatibilityContext{}
	add := func(scope, parser, source, target string, paths ...string) error {
		refs := []GlossaryArchiveReference{}
		seen := map[string]bool{}
		for _, path := range paths {
			if seen[path] {
				continue
			}
			seen[path] = true
			data, err := readCompatibilityFile(root, path)
			if err != nil {
				return err
			}
			refs = append(refs, GlossaryArchiveReference{path, sum(data)})
		}
		sort.Slice(refs, func(i, j int) bool { return refs[i].Path < refs[j].Path })
		context := GlossaryCompatibilityContext{Scope: scope, Parser: parser, Source: source, Target: target, References: refs}
		context.Identity = sum(mustJSON(context))
		contexts = append(contexts, context)
		return nil
	}
	latest, err := selectLatestRetranslationUnits(root, catalog, locale)
	if err != nil {
		return nil, err
	}
	for _, unit := range latest.ordered {
		choice, ok := latest.selectedByID[unit.ID]
		if !ok || !selectedRetranslationIdentityMatches(unit, choice) || choice.result.Status != "passed" {
			return nil, fmt.Errorf("compatibility inventory missing current-source passed candidate: %s", unit.ID)
		}
		candidatePath := filepath.ToSlash(filepath.Join("data", "retranslation-runs", locale, choice.batchID, choice.result.CandidatePath))
		target, err := readCompatibilityFile(root, candidatePath)
		if err != nil {
			return nil, err
		}
		parser := "present-visible/v1"
		if unit.Kind == UnitKindExample {
			parser = "go-comment-visible/v1"
		}
		sourceBytes, err := readCompatibilityFile(root, unit.SourcePath)
		if err != nil {
			return nil, err
		}
		if err := validateSnapshotSourceIdentity(catalog, unit, sourceBytes); err != nil {
			return nil, err
		}
		if err := add("tu:"+unit.ID, parser, string(unit.Source), string(target), unit.SourcePath, candidatePath); err != nil {
			return nil, err
		}
	}
	uiRoot := filepath.Join(root, "internal", "tour", "ui")
	sourceUI, err := ui.LoadFromFS("en", os.DirFS(uiRoot))
	if err != nil {
		return nil, err
	}
	targetUI, err := ui.LoadFromFS(locale, os.DirFS(uiRoot))
	if err != nil {
		return nil, err
	}
	keys := []string{}
	for key := range sourceUI.Messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		s, t := sourceUI.Messages[key], targetUI.Messages[key]
		parser := "plain-visible/v1"
		source, err := ui.VisibleMessageText(s)
		if err != nil {
			return nil, err
		}
		target, err := ui.VisibleMessageText(t)
		if err != nil {
			return nil, err
		}
		if err := add("ui:"+key, parser, source, target, "internal/tour/ui/en.json", "internal/tour/ui/"+locale+".json"); err != nil {
			return nil, err
		}
	}
	metadata, err := LoadArticleMetadata(root, locale, catalog)
	if err != nil {
		return nil, err
	}
	for name, m := range metadata {
		path := "_content/tour/" + name
		title, subtitle, err := sourceArticleHeader(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		if err := add("article:"+name, "plain-visible/v1", title+"\n"+subtitle, m.Title+"\n"+m.Subtitle, path, "locales/"+locale+"/article-metadata.json"); err != nil {
			return nil, err
		}
	}
	coursePath := "locales/" + locale + "/course-metadata.json"
	courseData, err := readCompatibilityFile(root, coursePath)
	if err != nil {
		return nil, err
	}
	course, err := decodeCourseMetadata(courseData)
	if err != nil {
		return nil, err
	}
	if course.Locale != locale {
		return nil, fmt.Errorf("compatibility course locale mismatch")
	}
	entries, err := courseMetadataBaseIndex(course, catalog)
	if err != nil {
		return nil, err
	}
	var canonical map[string]CourseSourceDescriptionPage
	if course.SchemaVersion == CourseMetadataSchemaVersionV2 {
		source, err := LoadCourseSourceDescriptions(root, catalog)
		if err != nil {
			return nil, err
		}
		if _, err := RequireCurrentCourseSourceDescriptionReview(root, catalog); err != nil {
			return nil, err
		}
		canonical = courseSourceDescriptionsByID(source)
	} else if course.SchemaVersion != CourseMetadataSchemaVersion {
		return nil, fmt.Errorf("unknown Course SEO schema")
	}
	for _, page := range catalog.Pages {
		entry := entries[page.ID]
		// Full generation context is retained for v1 and v2, with the canonical
		// description added only for v2. Distinct parser contexts avoid treating
		// a plain description as a present directive or code block.
		targetPath := canonicalCandidatePath(locale, page.ID)
		target, err := readCompatibilityFile(root, targetPath)
		if err != nil {
			return nil, err
		}
		if err := add("seo-page:"+page.ID, "present-visible/v1", string(page.Source), string(target), "_content/tour/"+page.Article, targetPath); err != nil {
			return nil, err
		}
		paths := []string{coursePath}
		source := ""
		if canonical != nil {
			source = canonical[page.ID].Description
			paths = append(paths, courseSourceDescriptionsRelativePath)
		}
		if err := add("seo:"+page.ID, "plain-visible/v1", source, entry.Description, paths...); err != nil {
			return nil, err
		}
	}
	for directory, extension := range map[string]string{"_content/tour/static/js": ".js", "_content/tour/template": ".tmpl", "_content/tour/static/partials": ".html"} {
		path, err := compatibilityPath(root, directory+"/.inventory", false)
		if err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			if filepath.Ext(entry.Name()) == extension && !entry.Type().IsRegular() {
				return nil, fmt.Errorf("unsafe formal context member %s/%s", directory, entry.Name())
			}
		}
	}
	others, err := exportOtherLocaleSurfaces(root, keys)
	if err != nil {
		return nil, err
	}
	for _, surface := range others {
		// No reliable source-to-target mapping exists for these mixed Go/JS/
		// template contexts. Never infer compatibility from an absent raw term.
		if err := add("surface:"+surface.ID, "unknown", surface.SourceText, "", surface.Path); err != nil {
			return nil, err
		}
	}
	sort.Slice(contexts, func(i, j int) bool { return contexts[i].Scope < contexts[j].Scope })
	for i, c := range contexts {
		if i > 0 && contexts[i-1].Scope == c.Scope {
			return nil, fmt.Errorf("duplicate compatibility context %s", c.Scope)
		}
	}
	return contexts, nil
}

func compatibilityVisiblePair(context GlossaryCompatibilityContext) (string, string, error) {
	source, target := context.Source, context.Target
	switch context.Parser {
	case "plain-visible/v1":
		// Plain UI/metadata has no present inline-code grammar. Only machine
		// URLs are excluded; backticks and labels remain literal visible text.
		maskURLs := func(s string) string {
			data := []byte(s)
			for _, span := range machineURLRE.FindAllStringIndex(s, -1) {
				for i := span[0]; i < span[1]; i++ {
					data[i] = ' '
				}
			}
			return string(data)
		}
		return maskURLs(source), maskURLs(target), nil
	case "present-visible/v1":
		return maskCompatibilityText(source, source), maskCompatibilityText(source, target), nil
	case "go-comment-visible/v1":
		a, err := scanGoExampleComments([]byte(source))
		if err != nil {
			return "", "", err
		}
		b, err := scanGoExampleComments([]byte(target))
		if err != nil || len(a) != len(b) {
			return "", "", fmt.Errorf("ambiguous Go comments")
		}
		var s, t []string
		for i, c := range a {
			if c.Kind == goExampleCommentNatural {
				s = append(s, maskCompatibilityText(source[c.PayloadStart:c.PayloadEnd], source[c.PayloadStart:c.PayloadEnd]))
				t = append(t, maskCompatibilityText(source[c.PayloadStart:c.PayloadEnd], target[b[i].PayloadStart:b[i].PayloadEnd]))
			}
		}
		return strings.Join(s, "\n"), strings.Join(t, "\n"), nil
	default:
		return "", "", fmt.Errorf("unknown visibility parser")
	}
}

func maskCompatibilityText(source, target string) string {
	visible := visibleCandidateTextBytes(source, target)
	data := []byte(target)
	for i, v := range visible {
		if !v {
			data[i] = ' '
		}
	}
	return string(data)
}
