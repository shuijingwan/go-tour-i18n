package sitecontent

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"

	yaml "github.com/shuijingwan/go-tour-i18n/internal/sitecontent/shellyaml"
)

func shellNodeValue(n *yaml.Node) any {
	switch n.Kind {
	case yaml.MappingNode:
		m := map[string]any{}
		for i := 0; i < len(n.Content); i += 2 {
			m[n.Content[i].Value] = shellNodeValue(n.Content[i+1])
		}
		return m
	case yaml.SequenceNode:
		v := []any{}
		for _, c := range n.Content {
			v = append(v, shellNodeValue(c))
		}
		return v
	case yaml.ScalarNode:
		if n.Tag == "!!bool" {
			return n.Value == "true"
		}
		if n.Tag == "!!null" {
			return nil
		}
		return n.Value
	}
	return nil
}

// Runtime consumes validated localized data, never source fallback language.
func ShellRuntimeData(name string, raw []byte) (any, error) {
	if _, err := ExtractShellLanguageSource(name, raw); err != nil {
		return nil, err
	}
	n, err := shellYAMLNode(raw)
	if err != nil {
		return nil, err
	}
	return shellNodeValue(n), nil
}

func ParseShellRuntimeYAML(raw []byte) (any, error) {
	n, err := shellYAMLNode(raw)
	if err != nil {
		return nil, err
	}
	return shellNodeValue(n), nil
}

func ShellPageRenderInput(raw []byte) ([]byte, map[string]any, error) {
	if !bytes.HasPrefix(raw, []byte("---\n")) {
		return nil, nil, fmt.Errorf("Homepage metadata required")
	}
	end := bytes.Index(raw[4:], []byte("\n---\n"))
	if end < 0 {
		return nil, nil, fmt.Errorf("Homepage metadata framing")
	}
	n, err := shellYAMLNode(raw[4 : 4+end])
	if err != nil {
		return nil, nil, err
	}
	m, ok := shellNodeValue(n).(map[string]any)
	if !ok {
		return nil, nil, fmt.Errorf("Homepage metadata mapping required")
	}
	return raw[4+end+5:], m, nil
}

// Only frozen machine/brand metadata used by the Homepage's company grid is
// projected. It adds no routes or localized body and requires exact manifest
// bytes. Multiline descriptive prose is not a runtime language fallback.
func FrozenHomepageCompanies(root string) ([]map[string]any, error) {
	m, err := loadSourceManifest(root)
	if err != nil {
		return nil, err
	}
	rows := []map[string]any{}
	for _, f := range m.Files {
		if path.Dir(f.Path) != "_content/solutions" || path.Ext(f.Path) != ".md" {
			continue
		}
		b, err := readAdoptedSource(root, f)
		if err != nil {
			return nil, err
		}
		if !bytes.HasPrefix(b, []byte("---\n")) {
			continue
		}
		end := bytes.Index(b[4:], []byte("\n---\n"))
		if end < 0 {
			return nil, fmt.Errorf("frozen company metadata framing")
		}
		selected := []string{}
		seen := map[string]bool{}
		for _, line := range strings.Split(string(b[4:4+end]), "\n") {
			key, _, ok := strings.Cut(line, ":")
			if !ok {
				continue
			}
			switch key {
			case "series", "company", "logoSrc", "logoSrcDark", "link", "inLandingPageGrid", "date":
				if seen[key] {
					return nil, fmt.Errorf("duplicate frozen company field")
				}
				seen[key] = true
				selected = append(selected, line)
			}
		}
		if len(selected) == 0 {
			continue
		}
		n, err := shellYAMLNode([]byte(strings.Join(selected, "\n") + "\n"))
		if err != nil {
			return nil, err
		}
		row := shellNodeValue(n).(map[string]any)
		if row["series"] != "Case Studies" {
			continue
		}
		if row["company"] == nil || row["logoSrc"] == nil {
			return nil, fmt.Errorf("company grid identity incomplete")
		}
		row["URL"] = strings.TrimSuffix(strings.TrimPrefix(f.Path, "_content"), ".md")
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i]["URL"].(string) < rows[j]["URL"].(string) })
	return rows, nil
}
