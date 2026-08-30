package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

type localeInitOptions struct {
	Locale       string
	LanguageName string
	EnglishName  string
	HTMLLang     string
}

type localeInitResult struct {
	Locale       string
	LocaleDir    string
	UICatalog    string
	UnitCount    int
	PageCount    int
	ExampleCount int
}

type localeInitUICatalog struct {
	Locale   string                         `json:"locale"`
	HTMLLang string                         `json:"html_lang"`
	Messages map[string]localeInitUIMessage `json:"messages"`
}

type localeInitUIMessage struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

type localeInitArticleMetadata struct {
	Locale   string                           `json:"locale"`
	Articles []localeInitArticleMetadataEntry `json:"articles"`
}

type localeInitArticleMetadataEntry struct {
	Article  string `json:"article"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

type localeInitCourseMetadataInventory struct {
	Locale string                                   `json:"locale"`
	Status string                                   `json:"status"`
	Pages  []localeInitCourseMetadataInventoryEntry `json:"pages"`
}

type localeInitCourseMetadataInventoryEntry struct {
	PageID string `json:"page_id"`
	Route  string `json:"route"`
}

const localeInitIncompleteMarker = ".locale-init-incomplete"

var localeInitPlaceholderRE = regexp.MustCompile(`\{[A-Za-z][A-Za-z0-9_]*\}`)
var localeInitRichTagRE = regexp.MustCompile(`</?[^>]+>`)

func initializeLocaleCommand(root string, catalog *i18n.Catalog, args []string) (*localeInitResult, error) {
	fs := flag.NewFlagSet("locale init", flag.ContinueOnError)
	locale := fs.String("locale", "", "locale 标识，例如 it-IT")
	languageName := fs.String("language-name", "", "该语言的本地名称（autonym）")
	englishName := fs.String("english-name", "", "该语言的英文名称")
	htmlLang := fs.String("html-lang", "", "HTML lang；默认与 locale 相同")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() != 0 {
		return nil, fmt.Errorf("locale init 存在未识别参数：%s", strings.Join(fs.Args(), " "))
	}
	if *locale == "" || *languageName == "" || *englishName == "" {
		return nil, errors.New("--locale、--language-name 和 --english-name 为必填")
	}
	if *htmlLang == "" {
		*htmlLang = *locale
	}
	options := localeInitOptions{Locale: *locale, LanguageName: *languageName, EnglishName: *englishName, HTMLLang: *htmlLang}
	return initializeLocale(root, catalog, options)
}

func initializeLocale(root string, catalog *i18n.Catalog, options localeInitOptions) (*localeInitResult, error) {
	if catalog == nil {
		return nil, errors.New("locale init 需要正式 catalog")
	}
	if err := i18n.ValidateLocaleName(options.Locale); err != nil {
		return nil, err
	}
	if err := i18n.ValidateLocaleName(options.HTMLLang); err != nil {
		return nil, fmt.Errorf("invalid html_lang: %w", err)
	}
	if options.LanguageName == "" || options.EnglishName == "" {
		return nil, errors.New("language_name 和 english_name 不能为空")
	}

	localeDir := filepath.Join(root, "locales", options.Locale)
	uiPath := filepath.Join(root, "internal", "tour", "ui", options.Locale+".json")
	for _, path := range []string{localeDir, uiPath} {
		if _, err := os.Lstat(path); err == nil {
			return nil, fmt.Errorf("locale init 拒绝覆盖已存在路径：%s", filepath.ToSlash(path))
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("检查 locale init 目标 %s：%w", filepath.ToSlash(path), err)
		}
	}

	staging, err := os.MkdirTemp(filepath.Join(root, "locales"), "."+options.Locale+".init-")
	if err != nil {
		return nil, fmt.Errorf("创建 locale staging：%w", err)
	}
	defer os.RemoveAll(staging)
	if err := writeLocaleSkeletonFiles(root, staging, catalog, options); err != nil {
		return nil, err
	}

	uiData, err := buildLocaleInitUICatalog(root, options)
	if err != nil {
		return nil, err
	}
	uiTemporary, err := os.CreateTemp(filepath.Dir(uiPath), "."+options.Locale+".json.init-")
	if err != nil {
		return nil, fmt.Errorf("创建 UI catalog staging：%w", err)
	}
	uiTemporaryPath := uiTemporary.Name()
	defer os.Remove(uiTemporaryPath)
	if _, err := uiTemporary.Write(uiData); err != nil {
		_ = uiTemporary.Close()
		return nil, fmt.Errorf("写入 UI catalog staging：%w", err)
	}
	if err := uiTemporary.Close(); err != nil {
		return nil, fmt.Errorf("关闭 UI catalog staging：%w", err)
	}

	if err := commitLocaleInitDirectory(staging, localeDir); err != nil {
		return nil, err
	}
	committedLocale := true
	defer func() {
		if committedLocale {
			_ = os.RemoveAll(localeDir)
		}
	}()
	status, err := i18n.InitializeLocaleStatus(root, options.Locale, catalog)
	if err != nil {
		return nil, fmt.Errorf("初始化 status.tsv：%w", err)
	}
	if err := os.Link(uiTemporaryPath, uiPath); err != nil {
		return nil, fmt.Errorf("提交 UI catalog：%w", err)
	}
	committedLocale = false
	return &localeInitResult{
		Locale: options.Locale, LocaleDir: repositoryPath(root, localeDir), UICatalog: repositoryPath(root, uiPath),
		UnitCount: status.Total, PageCount: status.Pages, ExampleCount: status.Examples,
	}, nil
}

func writeLocaleSkeletonFiles(root, directory string, catalog *i18n.Catalog, options localeInitOptions) error {
	metadata := i18n.Locale{
		Locale: options.Locale, LanguageName: options.LanguageName, EnglishName: options.EnglishName,
		HTMLLang: options.HTMLLang, Phase: "scaffold", TranslationUnit: "present.Section", DefaultLanguage: false,
	}
	if err := writeLocaleInitJSON(filepath.Join(directory, "locale.json"), metadata); err != nil {
		return err
	}
	glossary := fmt.Sprintf("locale: %s\nmandatory:\n  TODO_SOURCE_TERM: TODO_TARGET_TERM\npreferred:\nforbidden:\nkeep:\n", options.Locale)
	if err := os.WriteFile(filepath.Join(directory, "glossary.yaml"), []byte(glossary), 0644); err != nil {
		return fmt.Errorf("写入 glossary 骨架：%w", err)
	}
	articles := make([]localeInitArticleMetadataEntry, 0, len(i18n.ArticleOrder))
	seen := map[string]bool{}
	for _, page := range catalog.Pages {
		seen[page.Article] = true
	}
	for _, article := range i18n.ArticleOrder {
		if seen[article] {
			articles = append(articles, localeInitArticleMetadataEntry{
				Article: article, Title: "TODO: " + article + " title", Subtitle: "TODO: " + article + " subtitle",
			})
		}
	}
	if err := writeLocaleInitJSON(filepath.Join(directory, "article-metadata.json"), localeInitArticleMetadata{Locale: options.Locale, Articles: articles}); err != nil {
		return err
	}
	course := localeInitCourseMetadataInventory{Locale: options.Locale, Status: "TODO"}
	for _, page := range catalog.Pages {
		course.Pages = append(course.Pages, localeInitCourseMetadataInventoryEntry{PageID: page.ID, Route: page.Route})
	}
	if err := writeLocaleInitJSON(filepath.Join(directory, "course-metadata.todo.json"), course); err != nil {
		return err
	}
	marker := "TODO: complete glossary, UI, article metadata, and post-promotion course metadata; then remove this marker before complete build, preview, or publish.\n"
	if err := os.WriteFile(filepath.Join(directory, localeInitIncompleteMarker), []byte(marker), 0644); err != nil {
		return fmt.Errorf("写入 locale init 未完成标记：%w", err)
	}
	_ = root
	return nil
}

func commitLocaleInitDirectory(staging, localeDir string) error {
	if err := os.Mkdir(localeDir, 0755); err != nil {
		return fmt.Errorf("提交 locale 目录：%w", err)
	}
	completed := false
	defer func() {
		if !completed {
			_ = os.RemoveAll(localeDir)
		}
	}()
	entries, err := os.ReadDir(staging)
	if err != nil {
		return fmt.Errorf("读取 locale staging：%w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return fmt.Errorf("locale staging 含意外目录：%s", entry.Name())
		}
		if err := os.Rename(filepath.Join(staging, entry.Name()), filepath.Join(localeDir, entry.Name())); err != nil {
			return fmt.Errorf("提交 locale 文件 %s：%w", entry.Name(), err)
		}
	}
	completed = true
	return nil
}

func requireLocaleInitializationComplete(root, locale string) error {
	marker := filepath.Join(root, "locales", locale, localeInitIncompleteMarker)
	if _, err := os.Lstat(marker); err == nil {
		return fmt.Errorf("locale %s 仍有 %s：完成 glossary、UI、article metadata 和 promotion 后的正式 course metadata，再删除该标记", locale, localeInitIncompleteMarker)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("检查 locale init 完成标记：%w", err)
	}
	return nil
}

func buildLocaleInitUICatalog(root string, options localeInitOptions) ([]byte, error) {
	data, err := os.ReadFile(filepath.Join(root, "internal", "tour", "ui", "en.json"))
	if err != nil {
		return nil, fmt.Errorf("读取英文 UI source：%w", err)
	}
	var source localeInitUICatalog
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&source); err != nil {
		return nil, fmt.Errorf("解析英文 UI source：%w", err)
	}
	target := localeInitUICatalog{Locale: options.Locale, HTMLLang: options.HTMLLang, Messages: make(map[string]localeInitUIMessage, len(source.Messages))}
	keys := make([]string, 0, len(source.Messages))
	for key := range source.Messages {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		message := source.Messages[key]
		placeholder := "TODO: " + key
		for _, identity := range localeInitPlaceholderRE.FindAllString(message.Text, -1) {
			if !strings.Contains(placeholder, identity) {
				placeholder += " " + identity
			}
		}
		if message.Kind == "rich" {
			placeholder = localeInitRichPlaceholder(message.Text, placeholder)
		}
		target.Messages[key] = localeInitUIMessage{Kind: message.Kind, Text: placeholder}
	}
	return json.MarshalIndent(target, "", "  ")
}

func localeInitRichPlaceholder(source, placeholder string) string {
	var out strings.Builder
	position := 0
	wroteText := false
	for _, match := range localeInitRichTagRE.FindAllStringIndex(source, -1) {
		if strings.TrimSpace(source[position:match[0]]) != "" {
			out.WriteString(placeholder)
			wroteText = true
		}
		out.WriteString(source[match[0]:match[1]])
		position = match[1]
	}
	if strings.TrimSpace(source[position:]) != "" || !wroteText {
		out.WriteString(placeholder)
	}
	return out.String()
}

func writeLocaleInitJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("写入 %s：%w", filepath.Base(path), err)
	}
	return nil
}

func repositoryPath(root, path string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(relative)
}
