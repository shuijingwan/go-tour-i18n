package i18n

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Glossary struct {
	Mandatory map[string]string
	Preferred map[string]string
	Forbidden []string
}

func LoadGlossary(root, locale string) (*Glossary, error) {
	path := filepath.Join(root, "locales", locale, "glossary.yaml")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read glossary: %w", err)
	}
	defer f.Close()
	g := &Glossary{Mandatory: map[string]string{}, Preferred: map[string]string{}}
	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if !strings.HasPrefix(line, " ") {
			section = strings.TrimSuffix(trimmed, ":")
			continue
		}
		switch section {
		case "mandatory":
			key, value, ok := strings.Cut(trimmed, ":")
			if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("parse glossary mandatory entry %q", trimmed)
			}
			g.Mandatory[strings.TrimSpace(key)] = strings.TrimSpace(value)
		case "preferred":
			key, value, ok := strings.Cut(trimmed, ":")
			if !ok || strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				return nil, fmt.Errorf("parse glossary preferred entry %q", trimmed)
			}
			g.Preferred[strings.TrimSpace(key)] = strings.TrimSpace(value)
		case "forbidden":
			if !strings.HasPrefix(trimmed, "- ") || strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")) == "" {
				return nil, fmt.Errorf("parse glossary forbidden entry %q", trimmed)
			}
			g.Forbidden = append(g.Forbidden, strings.TrimSpace(strings.TrimPrefix(trimmed, "- ")))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(g.Mandatory) == 0 {
		return nil, fmt.Errorf("glossary has no mandatory mappings")
	}
	return g, nil
}

func (g *Glossary) PromptRules(pageID string) string {
	keys := make([]string, 0, len(g.Mandatory))
	for key := range g.Mandatory {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var rules []string
	for _, key := range keys {
		rules = append(rules, fmt.Sprintf("- %s => %s（强制；不得保留对应的英文显示文本）", key, g.Mandatory[key]))
	}
	preferredKeys := make([]string, 0, len(g.Preferred))
	for key := range g.Preferred {
		preferredKeys = append(preferredKeys, key)
	}
	sort.Strings(preferredKeys)
	for _, key := range preferredKeys {
		rules = append(rules, fmt.Sprintf("- 普通正文中的 %s => %s（上下文指导；应结合完整页面自然翻译）", key, g.Preferred[key]))
	}
	for _, value := range g.Forbidden {
		rules = append(rules, "- 禁止使用的 zh-CN 译法："+value)
	}
	if pageID == "welcome/1" {
		rules = append(rules, "- welcome/1 必须将 tour 的含义保留为“之旅”；不得简化或改变该含义")
	}
	return strings.Join(rules, "\n")
}
