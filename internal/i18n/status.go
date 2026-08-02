package i18n

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Locale struct {
	Locale          string `json:"locale"`
	LanguageName    string `json:"language_name"`
	EnglishName     string `json:"english_name"`
	HTMLLang        string `json:"html_lang"`
	Phase           string `json:"phase"`
	TranslationUnit string `json:"translation_unit"`
	DefaultLanguage bool   `json:"default_language"`
}

type Status struct {
	PageID        string
	State         string
	Attempts      int
	SourceSHA256  string
	CandidatePath string
	UpdatedAt     string
	Note          string
}

var allowedStates = map[string]bool{"pending": true, "candidate": true, "ready": true, "blocked": true, "published": true}

func CheckStatus(root, localeName string, catalog *Catalog) error {
	localeDir := filepath.Join(root, "locales", localeName)
	b, err := os.ReadFile(filepath.Join(localeDir, "locale.json"))
	if err != nil {
		return err
	}
	var locale Locale
	if err := json.Unmarshal(b, &locale); err != nil {
		return fmt.Errorf("locale.json: %w", err)
	}
	if locale.Locale != localeName || locale.Phase != "scaffold" || locale.TranslationUnit != "present.Section" {
		return fmt.Errorf("locale metadata mismatch: locale=%q phase=%q translation_unit=%q", locale.Locale, locale.Phase, locale.TranslationUnit)
	}
	statuses, err := ReadStatuses(filepath.Join(localeDir, "status.tsv"))
	if err != nil {
		return err
	}
	if len(statuses) != len(catalog.Pages) {
		return fmt.Errorf("status entries = %d, want %d", len(statuses), len(catalog.Pages))
	}
	pages := make(map[string]Page, len(catalog.Pages))
	for _, page := range catalog.Pages {
		pages[page.ID] = page
	}
	seen := map[string]bool{}
	for _, s := range statuses {
		if seen[s.PageID] {
			return fmt.Errorf("duplicate page_id %q", s.PageID)
		}
		seen[s.PageID] = true
		page, ok := pages[s.PageID]
		if !ok {
			return fmt.Errorf("status has unknown persistent page_id %q", s.PageID)
		}
		if !allowedStates[s.State] {
			return fmt.Errorf("%s: invalid status %q", s.PageID, s.State)
		}
		if s.Attempts < 0 {
			return fmt.Errorf("%s: attempts must be non-negative", s.PageID)
		}
		if s.State == "pending" && s.CandidatePath != "" {
			return fmt.Errorf("%s: pending requires empty candidate_path", s.PageID)
		}
		if s.SourceSHA256 != page.SourceSHA256 {
			return fmt.Errorf("%s: stale source_sha256", s.PageID)
		}
		if s.UpdatedAt != "" {
			tm, err := time.Parse(time.RFC3339, s.UpdatedAt)
			if err != nil || tm.Location() != time.UTC {
				return fmt.Errorf("%s: updated_at must be UTC RFC3339", s.PageID)
			}
		}
	}
	return nil
}

func ReadStatuses(path string) ([]Status, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = 7
	header, err := r.Read()
	if err != nil {
		return nil, err
	}
	want := []string{"page_id", "status", "attempts", "source_sha256", "candidate_path", "updated_at", "note"}
	for i := range want {
		if header[i] != want[i] {
			return nil, fmt.Errorf("status header column %d=%q, want %q", i+1, header[i], want[i])
		}
	}
	var out []Status
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		for i, field := range record {
			if strings.ContainsAny(field, "\t\r\n") {
				return nil, fmt.Errorf("%s: field %d contains a tab or newline", record[0], i+1)
			}
		}
		attempts, err := strconv.Atoi(record[2])
		if err != nil {
			return nil, fmt.Errorf("%s: invalid attempts %q", record[0], record[2])
		}
		out = append(out, Status{record[0], record[1], attempts, record[3], record[4], record[5], record[6]})
	}
	return out, nil
}
