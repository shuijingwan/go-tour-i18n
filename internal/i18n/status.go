package i18n

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
	UnitID        string
	State         string
	Attempts      int
	SourceSHA256  string
	CandidatePath string
	UpdatedAt     string
	Note          string
}

// StatusInitializationResult reports the stable workflow inventory written by
// InitializeLocaleStatus.
type StatusInitializationResult struct {
	Total    int
	Pages    int
	Examples int
}

var allowedStates = map[string]bool{"pending": true, "candidate": true, "ready": true, "blocked": true, "published": true}

var localeNameRE = regexp.MustCompile(`^[a-z]{2,8}(?:-[A-Za-z0-9]{2,8})*$`)

// ValidateLocaleName accepts the locale tags that are safe to use as one
// locale directory name. Configuration and workflow completeness are checked
// separately by callers that require them.
func ValidateLocaleName(locale string) error {
	if !localeNameRE.MatchString(locale) {
		return fmt.Errorf("invalid locale %q", locale)
	}
	return nil
}

func localeWorkflowDirectory(root, localeName string) (string, error) {
	if err := ValidateLocaleName(localeName); err != nil {
		return "", err
	}
	localeDir := filepath.Join(root, "locales", localeName)
	b, err := os.ReadFile(filepath.Join(localeDir, "locale.json"))
	if err != nil {
		return "", err
	}
	var locale Locale
	if err := json.Unmarshal(b, &locale); err != nil {
		return "", fmt.Errorf("locale.json: %w", err)
	}
	if locale.Locale != localeName || locale.Phase != "scaffold" || locale.TranslationUnit != "present.Section" {
		return "", fmt.Errorf("locale metadata mismatch: locale=%q phase=%q translation_unit=%q", locale.Locale, locale.Phase, locale.TranslationUnit)
	}
	return localeDir, nil
}

func CheckStatus(root, localeName string, catalog *Catalog) error {
	localeDir, err := localeWorkflowDirectory(root, localeName)
	if err != nil {
		return err
	}
	statuses, err := ReadStatuses(filepath.Join(localeDir, "status.tsv"))
	if err != nil {
		return err
	}
	return checkStatuses(localeName, catalog, statuses)
}

func checkStatuses(localeName string, catalog *Catalog, statuses []Status) error {
	expected, _, _, err := localeWorkflowUnits(catalog)
	if err != nil {
		return err
	}
	if len(statuses) != len(expected) {
		return fmt.Errorf("status entries = %d, want %d translation units", len(statuses), len(expected))
	}
	seen := map[string]bool{}
	for _, s := range statuses {
		unitID := s.UnitID
		if seen[unitID] {
			return fmt.Errorf("duplicate translation unit ID %q", unitID)
		}
		seen[unitID] = true
		unit, ok := expected[unitID]
		if !ok {
			if catalogUnit, unitErr := catalog.Unit(unitID); unitErr == nil && catalogUnit.Kind == UnitKindExample {
				return fmt.Errorf("status has non-eligible example translation unit %q", unitID)
			}
			return fmt.Errorf("status has unknown translation unit ID %q", unitID)
		}
		if !allowedStates[s.State] {
			return fmt.Errorf("%s: invalid status %q", unitID, s.State)
		}
		if s.Attempts < 0 {
			return fmt.Errorf("%s: attempts must be non-negative", unitID)
		}
		if s.State == "pending" && s.CandidatePath != "" {
			return fmt.Errorf("%s: pending requires empty candidate_path", unitID)
		}
		if s.SourceSHA256 != unit.SourceSHA256 {
			return fmt.Errorf("%s: stale source_sha256", unitID)
		}
		if s.CandidatePath != "" {
			wantPath, pathErr := canonicalTranslationUnitCandidatePath(localeName, unit)
			if pathErr != nil {
				return pathErr
			}
			if s.CandidatePath != wantPath {
				return fmt.Errorf("%s: candidate_path %q is not canonical %q", unitID, s.CandidatePath, wantPath)
			}
		}
		if s.UpdatedAt != "" {
			tm, err := time.Parse(time.RFC3339, s.UpdatedAt)
			if err != nil || tm.Location() != time.UTC {
				return fmt.Errorf("%s: updated_at must be UTC RFC3339", unitID)
			}
		}
	}
	for unitID := range expected {
		if !seen[unitID] {
			return fmt.Errorf("status is missing translation unit %q", unitID)
		}
	}
	return nil
}

// InitializeLocaleStatus creates the first status.tsv for a locale from the
// current formal TranslationUnit catalog. It never overwrites an existing
// status file and does not repair or migrate established workflow state.
func InitializeLocaleStatus(root, localeName string, catalog *Catalog) (*StatusInitializationResult, error) {
	localeDir, err := localeWorkflowDirectory(root, localeName)
	if err != nil {
		return nil, err
	}
	statusPath := filepath.Join(localeDir, "status.tsv")
	if _, err := os.Lstat(statusPath); err == nil {
		return nil, fmt.Errorf("status.tsv already exists for locale %q", localeName)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("check status.tsv for locale %q: %w", localeName, err)
	}

	units, pages, examples, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return nil, err
	}
	statuses := make([]Status, 0, len(units))
	for _, unit := range units {
		statuses = append(statuses, Status{
			UnitID:       unit.ID,
			State:        "pending",
			Attempts:     0,
			SourceSHA256: unit.SourceSHA256,
		})
	}
	if err := checkStatuses(localeName, catalog, statuses); err != nil {
		return nil, fmt.Errorf("validate initial status: %w", err)
	}

	temporary, err := os.CreateTemp(localeDir, ".status.tsv.init-")
	if err != nil {
		return nil, fmt.Errorf("create temporary status.tsv: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if err := temporary.Chmod(0644); err != nil {
		return nil, fmt.Errorf("set temporary status.tsv permissions: %w", err)
	}
	data := statusBytes(statuses)
	if written, err := temporary.Write(data); err != nil {
		return nil, fmt.Errorf("write temporary status.tsv: %w", err)
	} else if written != len(data) {
		return nil, fmt.Errorf("write temporary status.tsv: %w", io.ErrShortWrite)
	}
	if err := temporary.Sync(); err != nil {
		return nil, fmt.Errorf("sync temporary status.tsv: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return nil, fmt.Errorf("close temporary status.tsv: %w", err)
	}
	decoded, err := ReadStatuses(temporaryPath)
	if err != nil {
		return nil, fmt.Errorf("verify temporary status.tsv encoding: %w", err)
	}
	if err := checkStatuses(localeName, catalog, decoded); err != nil {
		return nil, fmt.Errorf("verify temporary status.tsv: %w", err)
	}
	if err := os.Link(temporaryPath, statusPath); err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("status.tsv already exists for locale %q", localeName)
		}
		return nil, fmt.Errorf("commit status.tsv for locale %q: %w", localeName, err)
	}

	return &StatusInitializationResult{Total: len(units), Pages: pages, Examples: examples}, nil
}

func localeWorkflowUnits(catalog *Catalog) (map[string]*TranslationUnit, int, int, error) {
	ordered, pageCount, exampleCount, err := localeWorkflowUnitList(catalog)
	if err != nil {
		return nil, 0, 0, err
	}
	units := make(map[string]*TranslationUnit, len(ordered))
	for _, unit := range ordered {
		units[unit.ID] = unit
	}
	return units, pageCount, exampleCount, nil
}

// localeWorkflowUnitList is the stable workflow order used by artifacts that
// need an index: Catalog Page order first, then eligible Example inventory
// order. localeWorkflowUnits retains the map view used by status/projection.
func localeWorkflowUnitList(catalog *Catalog) ([]*TranslationUnit, int, int, error) {
	if catalog == nil {
		return nil, 0, 0, fmt.Errorf("catalog is required")
	}
	units := make([]*TranslationUnit, 0, len(catalog.Pages)+len(catalog.Examples))
	seen := make(map[string]bool, cap(units))
	for i := range catalog.Pages {
		unit, err := catalog.Unit(catalog.Pages[i].ID)
		if err != nil {
			return nil, 0, 0, err
		}
		if seen[unit.ID] {
			return nil, 0, 0, fmt.Errorf("duplicate catalog translation unit %q", unit.ID)
		}
		seen[unit.ID] = true
		units = append(units, unit)
	}
	exampleCount := 0
	for i := range catalog.Examples {
		example := &catalog.Examples[i]
		eligible, err := hasTranslatableGoExampleComment(example.Source)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("%s: determine locale workflow eligibility: %w", example.ID, err)
		}
		if !eligible {
			continue
		}
		unit, err := catalog.Unit(example.ID)
		if err != nil {
			return nil, 0, 0, err
		}
		if seen[unit.ID] {
			return nil, 0, 0, fmt.Errorf("duplicate catalog translation unit %q", unit.ID)
		}
		seen[unit.ID] = true
		units = append(units, unit)
		exampleCount++
	}
	return units, len(catalog.Pages), exampleCount, nil
}

// LocaleWorkflowUnitCounts reports the Page and eligible Example units that
// must have rows in a locale's unified status table.
func LocaleWorkflowUnitCounts(catalog *Catalog) (total, pages, examples int, err error) {
	units, pages, examples, err := localeWorkflowUnits(catalog)
	if err != nil {
		return 0, 0, 0, err
	}
	return len(units), pages, examples, nil
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
	want := []string{"unit_id", "status", "attempts", "source_sha256", "candidate_path", "updated_at", "note"}
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

func writeStatuses(path string, statuses []Status) error {
	return os.WriteFile(path, statusBytes(statuses), 0644)
}

func statusBytes(statuses []Status) []byte {
	records := make([][]string, 0, len(statuses)+1)
	records = append(records, []string{"unit_id", "status", "attempts", "source_sha256", "candidate_path", "updated_at", "note"})
	for _, status := range statuses {
		records = append(records, []string{
			status.UnitID,
			status.State,
			strconv.Itoa(status.Attempts),
			status.SourceSHA256,
			status.CandidatePath,
			status.UpdatedAt,
			status.Note,
		})
	}

	var b strings.Builder
	for _, record := range records {
		for i, field := range record {
			if i > 0 {
				b.WriteByte('\t')
			}
			if field == "" {
				b.WriteString(`""`)
				continue
			}
			if strings.ContainsAny(field, "\"\t\r\n") {
				b.WriteByte('"')
				b.WriteString(strings.ReplaceAll(field, `"`, `""`))
				b.WriteByte('"')
				continue
			}
			b.WriteString(field)
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
