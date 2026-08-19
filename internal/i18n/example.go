package i18n

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Example struct {
	ID           string
	SourcePath   string
	SourceSHA256 string
	Source       []byte
	ReferencedBy []string
}

type UnitKind string

const (
	UnitKindPage    UnitKind = "page"
	UnitKindExample UnitKind = "example"
)

type TranslationUnit struct {
	ID           string
	Kind         UnitKind
	SourcePath   string
	Source       []byte
	SourceSHA256 string
}

func (c *Catalog) Unit(id string) (*TranslationUnit, error) {
	if page, err := c.Page(id); err == nil {
		return &TranslationUnit{
			ID: page.ID, Kind: UnitKindPage,
			SourcePath: filepath.ToSlash(filepath.Join("_content", "tour", page.Article)),
			Source:     page.Source, SourceSHA256: page.SourceSHA256,
		}, nil
	}
	for i := range c.Examples {
		if c.Examples[i].ID == id {
			example := &c.Examples[i]
			return &TranslationUnit{ID: example.ID, Kind: UnitKindExample, SourcePath: example.SourcePath, Source: example.Source, SourceSHA256: example.SourceSHA256}, nil
		}
	}
	return nil, fmt.Errorf("unknown translation unit %q", id)
}

func discoverExamples(root string, pages []Page) ([]Example, error) {
	references := map[string][]string{}
	for _, page := range pages {
		if err := parseSinglePage(root, page.Article, page.Source); err != nil {
			return nil, fmt.Errorf("%s: discover .play examples: %w", page.ID, err)
		}
		scanner := bufio.NewScanner(bytes.NewReader(page.Source))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 2 && fields[0] == ".play" {
				references[fields[1]] = appendUnique(references[fields[1]], page.ID)
			}
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("%s: scan .play directives: %w", page.ID, err)
		}
	}
	manifest, hasManifest, err := readUpstreamLocalHashes(filepath.Join(root, "UPSTREAM_MANIFEST.tsv"))
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(references))
	for path := range references {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	examples := make([]Example, 0, len(paths))
	for _, referencedPath := range paths {
		clean, err := cleanPlayPath(referencedPath)
		if err != nil {
			return nil, err
		}
		sourcePath := filepath.ToSlash(filepath.Join("_content", "tour", clean))
		source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
		if err != nil {
			return nil, fmt.Errorf(".play example %q: %w", referencedPath, err)
		}
		hash := sum(source)
		if hasManifest {
			want, ok := manifest[sourcePath]
			if !ok {
				return nil, fmt.Errorf(".play example %q is missing from UPSTREAM_MANIFEST.tsv", referencedPath)
			}
			if hash != want {
				return nil, fmt.Errorf(".play example %q SHA-256 %s does not match UPSTREAM_MANIFEST.tsv local_sha256 %s", referencedPath, hash, want)
			}
		}
		by := append([]string(nil), references[referencedPath]...)
		sort.Strings(by)
		examples = append(examples, Example{ID: "example:" + clean, SourcePath: sourcePath, SourceSHA256: hash, Source: source, ReferencedBy: by})
	}
	return examples, nil
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func cleanPlayPath(path string) (string, error) {
	if path == "" || strings.Contains(path, "\\") {
		return "", fmt.Errorf("unsafe .play path %q", path)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if filepath.IsAbs(filepath.FromSlash(path)) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != path {
		return "", fmt.Errorf("unsafe or non-canonical .play path %q", path)
	}
	if filepath.Ext(clean) != ".go" {
		return "", fmt.Errorf(".play path %q is not a .go file", path)
	}
	return clean, nil
}

func readUpstreamLocalHashes(path string) (map[string]string, bool, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = 6
	header, err := r.Read()
	if err != nil {
		return nil, false, err
	}
	want := []string{"upstream_path", "local_path", "mode", "upstream_sha256", "local_sha256", "note"}
	for i := range want {
		if header[i] != want[i] {
			return nil, false, fmt.Errorf("%s: header column %d=%q, want %q", path, i+1, header[i], want[i])
		}
	}
	hashes := map[string]string{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, false, err
		}
		hashes[record[1]] = record[4]
	}
	return hashes, true, nil
}
