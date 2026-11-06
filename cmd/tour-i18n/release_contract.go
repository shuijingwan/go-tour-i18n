package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/shuijingwan/go-tour-i18n/internal/assets"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"github.com/shuijingwan/go-tour-i18n/internal/releasecontract"
	"github.com/shuijingwan/go-tour-i18n/internal/site"
	"github.com/shuijingwan/go-tour-i18n/internal/sitecontent"
)

type releaseBuild struct {
	Schema   string `json:"schema"`
	Identity string `json:"identity"`
	Locale   string `json:"locale"`
	Units    int    `json:"units"`
	Pages    int    `json:"pages"`
	Examples int    `json:"examples"`
	Articles int    `json:"articles"`
}

// CompilerInputs uses the existing production target's dependency graph, not a
// release inventory scan. Module-local compiler/embed inputs bind the build.
func releaseCompilerInputs(root string) (map[string]string, error) {
	cmd := exec.Command("go", "list", "-mod=readonly", "-deps", "-json", "./cmd/tour-production", "./cmd/tour-i18n")
	cmd.Dir = root
	r, e := cmd.StdoutPipe()
	if e != nil {
		return nil, e
	}
	cmd.Stderr = os.Stderr
	if e = cmd.Start(); e != nil {
		return nil, e
	}
	inputs := map[string]string{}
	decoder := json.NewDecoder(r)
	var decodeErr error
	for {
		var p struct {
			Dir                           string
			GoFiles, CgoFiles, EmbedFiles []string
		}
		e := decoder.Decode(&p)
		if e == io.EOF {
			break
		}
		if e != nil {
			decodeErr = e
			break
		}
		rel, e := filepath.Rel(root, p.Dir)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		for _, f := range append(append(p.GoFiles, p.CgoFiles...), p.EmbedFiles...) {
			name := filepath.ToSlash(filepath.Join(rel, f))
			b, e := releasecontract.ReadRegular(root, name)
			if e != nil {
				decodeErr = e
				break
			}
			inputs[name] = releasecontract.Digest(b)
		}
	}
	if decodeErr != nil {
		_ = cmd.Process.Kill()
	}
	e = cmd.Wait()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if e != nil {
		return nil, e
	}
	for _, p := range []string{"go.mod", "go.sum"} {
		b, e := releasecontract.ReadRegular(root, p)
		if e != nil {
			return nil, e
		}
		inputs[p] = releasecontract.Digest(b)
	}
	return inputs, nil
}

func buildReleaseContent(root string, catalog *i18n.Catalog, locale, output string) (*i18n.LocaleProjection, *releasecontract.Artifact, error) {
	inputs, e := releaseCompilerInputs(root)
	if e != nil {
		return nil, nil, e
	}
	// These existing locale inputs are copied/read, never mutated or re-reviewed.
	for _, name := range []string{"locale.json", "status.tsv", "glossary.yaml", "article-metadata.json", "course-metadata.json"} {
		p := "locales/" + locale + "/" + name
		b, e := releasecontract.ReadRegular(root, p)
		if os.IsNotExist(e) && name == "article-metadata.json" {
			continue
		}
		if e != nil {
			return nil, nil, e
		}
		inputs[p] = releasecontract.Digest(b)
	}
	// The projection reads all canonical Tour candidates. Bind additions/deletions
	// through status/catalog and the exact bytes actually used by the projection.
	candidates := filepath.Join(root, "locales", locale, "candidates")
	e = filepath.WalkDir(candidates, func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported candidate input")
		}
		rel, e := filepath.Rel(root, p)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		b, e := releasecontract.ReadRegular(root, rel)
		if e != nil {
			return e
		}
		inputs[rel] = releasecontract.Digest(b)
		return nil
	})
	if e != nil {
		return nil, nil, e
	}
	projection, e := i18n.BuildLocaleProjection(root, catalog, locale, output)
	if e != nil {
		return nil, nil, e
	}
	a, e := site.ExportRelease(root, projection.ContentDir, locale, inputs)
	if e != nil {
		return nil, nil, e
	}
	desc := releaseBuild{Schema: "go-learning/release-build/v1", Identity: a.Identity, Locale: locale, Units: projection.UnitCount, Pages: projection.PageCount, Examples: projection.ExampleCount, Articles: projection.ArticleCount}
	b, e := json.MarshalIndent(desc, "", "  ")
	if e != nil {
		return nil, nil, e
	}
	if e = os.WriteFile(filepath.Join(output, "build.json"), append(b, '\n'), 0644); e != nil {
		return nil, nil, e
	}
	sums, e := releasecontract.Checksums(output)
	if e != nil {
		return nil, nil, e
	}
	if e = os.WriteFile(filepath.Join(output, "SHA256SUMS"), sums, 0644); e != nil {
		return nil, nil, e
	}
	return projection, a, nil
}

func checkReleaseCurrent(root string, a *releasecontract.Artifact) error {
	if e := releasecontract.Current(root, a); e != nil {
		return e
	}
	if err := checkCompilerInputs(root, a.Inputs); err != nil {
		return err
	}
	if err := checkSharedProductionTarget(root, a.AssetOrigin); err != nil {
		return err
	}
	_, l, e := sitecontent.LocalCoverage(root, a.Locale)
	if e != nil {
		return e
	}
	if releasecontract.Identity(l) != a.CoverageIdentity {
		return fmt.Errorf("activated release coverage STALE")
	}
	origin, e := sitecontent.LocaleOrigin(root, a.Locale)
	if e != nil {
		return e
	}
	if a.Origin != strings.TrimSuffix(origin, "/") || a.AssetOrigin != assets.BaseURL {
		return fmt.Errorf("release origin/asset policy STALE")
	}
	routes := []string{}
	families := map[string]bool{}
	for _, p := range l.Packages {
		routes = append(routes, p.Routes...)
		for _, f := range p.RouteFamilies {
			families[f] = true
		}
	}
	sort.Strings(routes)
	fs := []string{}
	for f := range families {
		fs = append(fs, f)
	}
	sort.Strings(fs)
	if !reflect.DeepEqual(routes, a.Routes) || !reflect.DeepEqual(fs, a.Families) {
		return fmt.Errorf("non-exact activated release route universe")
	}
	paths, e := assets.Paths(root)
	if e != nil {
		return e
	}
	got := map[string]string{}
	for _, p := range paths {
		b, e := releasecontract.ReadRegular(root, "_content/"+p)
		if e != nil {
			return e
		}
		got[p] = releasecontract.Digest(b)
	}
	if !reflect.DeepEqual(got, a.SharedAssets) {
		return fmt.Errorf("shared-assets inventory/bytes STALE")
	}
	return nil
}

func checkSharedProductionTarget(root, origin string) error {
	b, e := releasecontract.ReadRegular(root, "production/identity.json")
	if e != nil {
		return e
	}
	var profile struct {
		Shared struct {
			Origin string `json:"shared_assets_public_origin"`
		} `json:"shared"`
	}
	if e = json.Unmarshal(b, &profile); e != nil {
		return e
	}
	if strings.TrimSuffix(profile.Shared.Origin, "/") != origin || origin != assets.BaseURL {
		return fmt.Errorf("shared-assets hostname differs from current production identity")
	}
	return nil
}

func checkCompilerInputs(root string, inputs map[string]string) error {
	compiler, err := releaseCompilerInputs(root)
	if err != nil {
		return err
	}
	for p, h := range compiler {
		if inputs[p] != h {
			return fmt.Errorf("compiler input exact-set/bytes STALE: %s", p)
		}
	}
	return nil
}

type releaseInfo struct {
	Result        string                    `json:"result"`
	Kind          string                    `json:"kind"`
	Locale        string                    `json:"locale"`
	Origin        string                    `json:"origin"`
	InputIdentity string                    `json:"input_identity"`
	Content       *releasecontract.Artifact `json:"content"`
	BrowserPlan   json.RawMessage           `json:"browser_plan"`
}

func inspectRelease(input string) (*releaseInfo, error) {
	kind := "production"
	id, e := releasecontract.VerifyChecksums(input)
	if e != nil {
		return nil, e
	}
	content := filepath.Join(input, "_content")
	a, e := releasecontract.Read(os.DirFS(content))
	if e != nil {
		return nil, e
	}
	manifest, e := os.ReadFile(filepath.Join(input, "release.json"))
	if e == nil {
		if _, _, e = releasecontract.VerifyBundle(input); e != nil {
			return nil, e
		}
		var m releaseManifest
		if e = sitecontent.StrictJSON(manifest, &m); e != nil {
			return nil, e
		}
		if m.SchemaVersion != releaseManifestSchemaVersion || m.Locale != a.Locale || m.SiteIdentity != a.Identity || m.SharedAssetsIdentity != a.SharedAssetsIdentity || m.BuildIdentity != a.Identity || m.LocalSocketEnabled || m.ExecutionTransport != "http-playground-proxy" {
			return nil, fmt.Errorf("release manifest identity/transport mismatch")
		}
	} else if os.IsNotExist(e) {
		kind = "build"
		entries, err := os.ReadDir(input)
		if err != nil {
			return nil, err
		}
		want := map[string]bool{"_content": true, "build.json": true, "SHA256SUMS": true}
		if len(entries) != len(want) {
			return nil, fmt.Errorf("non-exact build root")
		}
		for _, entry := range entries {
			if !want[entry.Name()] {
				return nil, fmt.Errorf("unexpected build root entry")
			}
		}
		b, e := os.ReadFile(filepath.Join(input, "build.json"))
		if e != nil {
			return nil, e
		}
		var m releaseBuild
		if e = sitecontent.StrictJSON(b, &m); e != nil {
			return nil, e
		}
		if m.Schema != "go-learning/release-build/v1" || m.Locale != a.Locale || m.Identity != a.Identity {
			return nil, fmt.Errorf("build descriptor mismatch")
		}
	} else {
		return nil, e
	}
	b, e := os.ReadFile(filepath.Join(content, filepath.FromSlash(releasecontract.BrowserPlanPath)))
	if e != nil {
		return nil, e
	}
	return &releaseInfo{Result: "PASS", Kind: kind, Locale: a.Locale, Origin: a.Origin, InputIdentity: id, Content: a, BrowserPlan: b}, nil
}

func releaseContractCommand(root string, args []string) error {
	if len(args) == 0 || args[0] != "check" {
		return fmt.Errorf("usage: release-contract check --input <build-or-release> [--current] [--json]")
	}
	f := flag.NewFlagSet("release-contract check", flag.ContinueOnError)
	input := f.String("input", "", "sealed build or bundle")
	current := f.Bool("current", false, "verify current activated and source inputs")
	jsonOut := f.Bool("json", false, "JSON proof")
	if e := f.Parse(args[1:]); e != nil {
		return e
	}
	if *input == "" || f.NArg() != 0 {
		return fmt.Errorf("--input required")
	}
	info, e := inspectRelease(*input)
	if e != nil {
		return e
	}
	if *current {
		if e = checkReleaseCurrent(root, info.Content); e != nil {
			return e
		}
	}
	if *jsonOut {
		return json.NewEncoder(os.Stdout).Encode(info)
	}
	fmt.Printf("release contract: PASS locale=%s input_identity=%s coverage=%s\n", info.Locale, info.InputIdentity, info.Content.CoverageIdentity)
	return nil
}
