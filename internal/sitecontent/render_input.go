package sitecontent

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// PageRenderInput uses the same metadata boundary as the formal Page parser.
// The title and template flag come from completed bytes, never source fallback.
func PageRenderInput(raw []byte) (body []byte, title string, templated bool, err error) {
	offset, err := parseMetadata(raw, func(int, int, string) {})
	if err != nil {
		return nil, "", false, err
	}
	if bytes.HasPrefix(raw, []byte("<!--{")) {
		var meta struct {
			Title    string
			Template bool
		}
		err = json.Unmarshal(raw[4:bytes.Index(raw, []byte("}-->"))+1], &meta)
		title, templated = meta.Title, meta.Template
	} else if bytes.HasPrefix(raw, []byte("---\n")) {
		for _, line := range strings.Split(string(raw[4:offset-5]), "\n") {
			key, value, _ := strings.Cut(line, ":")
			value = strings.TrimSpace(value)
			switch key {
			case "title":
				title = value
				if strings.HasPrefix(value, `"`) {
					title, err = strconv.Unquote(value)
				}
			case "template":
				templated = value == "true"
			}
		}
	}
	return raw[offset:], title, templated, err
}

// LearnRenderData decodes the formally supported sequence of scalar mappings.
// Plain scalar continuation lines fold to spaces, as in the frozen YAML data.
func LearnRenderData(raw []byte) ([]map[string]string, error) {
	if err := parseLearnData(raw, func(int, int, string) {}); err != nil {
		return nil, err
	}
	var rows []map[string]string
	key := ""
	baseIndent := -1
	for _, line := range strings.Split(string(raw), "\n") {
		value := strings.TrimSpace(line)
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "- ") {
			rows = append(rows, map[string]string{})
			baseIndent = indent + 2
			value = strings.TrimPrefix(value, "- ")
			indent = baseIndent
		}
		if k, v, ok := strings.Cut(value, ":"); ok && indent == baseIndent {
			key = k
			rows[len(rows)-1][key] = strings.TrimSpace(v)
		} else {
			rows[len(rows)-1][key] += " " + value
		}
	}
	return rows, nil
}
