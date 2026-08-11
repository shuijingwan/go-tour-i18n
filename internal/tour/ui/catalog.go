// Package ui loads the build-time Tour UI message catalogs.
package ui

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

//go:embed *.json
var catalogFiles embed.FS

// localePattern accepts a lowercase two- or three-letter language followed by
// zero or more simple alphanumeric subtags. It is deliberately narrower than
// full BCP 47 while allowing common tags such as zh-Hant and pt-BR safely.
var localePattern = regexp.MustCompile(`^[a-z]{2,3}(?:-[A-Za-z0-9]{2,8})*$`)
var messageKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:\.[a-z][a-z0-9_]*)+$`)

// Message is a locale-specific UI message. Rich messages have constrained,
// catalog-validated markup; callers must not treat plain messages as HTML.
type Message struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// Catalog is a complete build-time UI catalog for one locale.
type Catalog struct {
	Locale   string             `json:"locale"`
	HTMLLang string             `json:"html_lang"`
	Messages map[string]Message `json:"messages"`
}

// Plain returns a plain-text message for safe use by html/template. Missing
// keys and rich messages are errors so template consumers cannot silently
// produce an incomplete or unsafe UI.
func (c Catalog) Plain(key string) (string, error) {
	message, ok := c.Messages[key]
	if !ok {
		return "", fmt.Errorf("UI message %q is missing", key)
	}
	if message.Kind != "plain" {
		return "", fmt.Errorf("UI message %q has kind %q, not plain", key, message.Kind)
	}
	return message.Text, nil
}

// Rich returns a catalog-validated rich message for a consumer with an
// explicitly constrained rendering boundary. It is not template.HTML.
func (c Catalog) Rich(key string) (string, error) {
	message, ok := c.Messages[key]
	if !ok {
		return "", fmt.Errorf("UI message %q is missing", key)
	}
	if message.Kind != "rich" {
		return "", fmt.Errorf("UI message %q has kind %q, not rich", key, message.Kind)
	}
	return message.Text, nil
}

// Load returns a complete catalog for a supported build locale. It never
// falls back to English: a release locale must cover every English source key.
func Load(locale string) (Catalog, error) {
	return load(locale, catalogFiles)
}

func load(locale string, files fs.FS) (Catalog, error) {
	if !localePattern.MatchString(locale) {
		return Catalog{}, fmt.Errorf("invalid UI locale %q", locale)
	}
	data, err := fs.ReadFile(files, locale+".json")
	if err != nil {
		return Catalog{}, fmt.Errorf("unknown UI locale %q: %w", locale, err)
	}
	catalog, err := parseCatalog(data)
	if err != nil {
		return Catalog{}, fmt.Errorf("parse UI catalog %q: %w", locale, err)
	}
	if catalog.Locale != locale {
		return Catalog{}, fmt.Errorf("UI catalog %q declares locale %q", locale, catalog.Locale)
	}

	if locale != "en" {
		source, err := load("en", files)
		if err != nil {
			return Catalog{}, fmt.Errorf("load English UI source catalog: %w", err)
		}
		if err := validateCoverage(source, catalog); err != nil {
			return Catalog{}, fmt.Errorf("validate UI catalog %q: %w", locale, err)
		}
	}
	return cloneCatalog(catalog), nil
}

func parseCatalog(data []byte) (Catalog, error) {
	d := json.NewDecoder(bytes.NewReader(data))
	if err := expectObjectStart(d); err != nil {
		return Catalog{}, err
	}
	var catalog Catalog
	seen := map[string]bool{}
	for d.More() {
		key, err := objectKey(d)
		if err != nil {
			return Catalog{}, err
		}
		if seen[key] {
			return Catalog{}, fmt.Errorf("duplicate catalog field %q", key)
		}
		seen[key] = true
		switch key {
		case "locale":
			err = d.Decode(&catalog.Locale)
		case "html_lang":
			err = d.Decode(&catalog.HTMLLang)
		case "messages":
			catalog.Messages, err = parseMessages(d)
		default:
			return Catalog{}, fmt.Errorf("unknown catalog field %q", key)
		}
		if err != nil {
			return Catalog{}, err
		}
	}
	if err := expectObjectEnd(d); err != nil {
		return Catalog{}, err
	}
	if err := requireEOF(d); err != nil {
		return Catalog{}, err
	}
	if !seen["locale"] || !seen["html_lang"] || !seen["messages"] {
		return Catalog{}, fmt.Errorf("catalog requires locale, html_lang, and messages")
	}
	if err := validateCatalog(catalog); err != nil {
		return Catalog{}, err
	}
	return catalog, nil
}

func parseMessages(d *json.Decoder) (map[string]Message, error) {
	if err := expectObjectStart(d); err != nil {
		return nil, err
	}
	messages := make(map[string]Message)
	for d.More() {
		key, err := objectKey(d)
		if err != nil {
			return nil, err
		}
		if _, exists := messages[key]; exists {
			return nil, fmt.Errorf("duplicate message key %q", key)
		}
		var raw json.RawMessage
		if err := d.Decode(&raw); err != nil {
			return nil, err
		}
		message, err := parseMessage(raw)
		if err != nil {
			return nil, fmt.Errorf("message %q: %w", key, err)
		}
		messages[key] = message
	}
	if err := expectObjectEnd(d); err != nil {
		return nil, err
	}
	return messages, nil
}

func parseMessage(raw json.RawMessage) (Message, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var message Message
	if err := decoder.Decode(&message); err != nil {
		return Message{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return Message{}, err
	}
	return message, nil
}

func validateCatalog(catalog Catalog) error {
	if !localePattern.MatchString(catalog.Locale) {
		return fmt.Errorf("invalid locale %q", catalog.Locale)
	}
	if !localePattern.MatchString(catalog.HTMLLang) {
		return fmt.Errorf("invalid html_lang %q", catalog.HTMLLang)
	}
	if catalog.HTMLLang != catalog.Locale {
		return fmt.Errorf("html_lang %q does not match locale %q", catalog.HTMLLang, catalog.Locale)
	}
	if len(catalog.Messages) == 0 {
		return fmt.Errorf("catalog has no messages")
	}
	for key, message := range catalog.Messages {
		if !messageKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid message key %q", key)
		}
		if message.Text == "" {
			return fmt.Errorf("message %q is empty", key)
		}
		switch message.Kind {
		case "plain":
		case "rich":
			if err := validateRichMarkup(message.Text); err != nil {
				return fmt.Errorf("rich message %q: %w", key, err)
			}
		default:
			return fmt.Errorf("message %q has invalid kind %q", key, message.Kind)
		}
	}
	return nil
}

// Rich messages are constrained to the frozen Tour module-description markup.
func validateRichMarkup(text string) error {
	tag := regexp.MustCompile(`(?:<p>|</p>|<a href="https://go.dev">|</a>)`)
	stack := []string{}
	position := 0
	for _, match := range tag.FindAllStringIndex(text, -1) {
		if strings.ContainsAny(text[position:match[0]], "<>") {
			return fmt.Errorf("contains unsupported markup")
		}
		value := text[match[0]:match[1]]
		switch value {
		case "<p>", "<a href=\"https://go.dev\">":
			stack = append(stack, value[1:2])
		case "</p>", "</a>":
			want := value[2:3]
			if len(stack) == 0 || stack[len(stack)-1] != want {
				return fmt.Errorf("has unbalanced markup")
			}
			stack = stack[:len(stack)-1]
		}
		position = match[1]
	}
	if strings.ContainsAny(text[position:], "<>") || len(stack) != 0 {
		return fmt.Errorf("has unbalanced or unsupported markup")
	}
	return nil
}

func validateCoverage(source, locale Catalog) error {
	var missing, extra, wrongKind []string
	for key, sourceMessage := range source.Messages {
		message, ok := locale.Messages[key]
		if !ok {
			missing = append(missing, key)
			continue
		}
		if message.Kind != sourceMessage.Kind {
			wrongKind = append(wrongKind, key)
		}
	}
	for key := range locale.Messages {
		if _, ok := source.Messages[key]; !ok {
			extra = append(extra, key)
		}
	}
	if len(missing)+len(extra)+len(wrongKind) == 0 {
		return nil
	}
	sort.Strings(missing)
	sort.Strings(extra)
	sort.Strings(wrongKind)
	parts := make([]string, 0, 3)
	if len(missing) > 0 {
		parts = append(parts, "missing keys: "+strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		parts = append(parts, "unknown keys: "+strings.Join(extra, ", "))
	}
	if len(wrongKind) > 0 {
		parts = append(parts, "message kind mismatch: "+strings.Join(wrongKind, ", "))
	}
	return errors.New(strings.Join(parts, "; "))
}

func expectObjectStart(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '{' {
		return fmt.Errorf("expected JSON object")
	}
	return nil
}

func expectObjectEnd(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	if delimiter, ok := token.(json.Delim); !ok || delimiter != '}' {
		return fmt.Errorf("expected end of JSON object")
	}
	return nil
}

func objectKey(d *json.Decoder) (string, error) {
	token, err := d.Token()
	if err != nil {
		return "", err
	}
	key, ok := token.(string)
	if !ok {
		return "", fmt.Errorf("expected JSON object key")
	}
	return key, nil
}

func requireEOF(d *json.Decoder) error {
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("unexpected data after JSON value")
		}
		return err
	}
	return nil
}

func cloneCatalog(catalog Catalog) Catalog {
	copy := Catalog{Locale: catalog.Locale, HTMLLang: catalog.HTMLLang, Messages: make(map[string]Message, len(catalog.Messages))}
	for key, message := range catalog.Messages {
		copy.Messages[key] = message
	}
	return copy
}
