package sitecontent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

func TestStructuredHistoricalPackageRegistrationCoexistence(t *testing.T) {
	b, n, c := historicalRegistrationFixture(t)
	membership := b.Packages[0]
	membership.Identity = digest([]byte("unrelated membership predecessor"))
	membership.ParserContract = "different historical Page inventory contract"
	arbitrary := b.Packages[0]
	arbitrary.Identity = digest([]byte("same-contract arbitrary history"))
	ancestor := b.Packages[0]
	ancestor.Identity = digest([]byte("receipt-bound structured predecessor"))
	n.HistoricalPackages = append(n.HistoricalPackages, membership, arbitrary, ancestor)
	if err := registerStructuredHistoricalPackages(b, n, c, ancestor); err != nil {
		t.Fatal(err)
	}
	for _, p := range []Package{b.Packages[0], ancestor} {
		if _, err := i18n.HistoricalTranslationPackageCatalog(c, p.ID, p.Identity); err != nil {
			t.Fatal("exact structured predecessor missing", err)
		}
	}
	for _, p := range []Package{membership, arbitrary} {
		if _, err := i18n.HistoricalTranslationPackageCatalog(c, p.ID, p.Identity); err == nil {
			t.Fatal("unrelated same-ID history acquired structured authority")
		}
	}
}

func TestStructuredHistoricalPackageRegistrationAncestorFailClosed(t *testing.T) {
	for _, label := range []string{"missing", "contract-drift", "ambiguous"} {
		t.Run(label, func(t *testing.T) {
			b, n, c := historicalRegistrationFixture(t)
			ancestor := b.Packages[0]
			ancestor.Identity = digest([]byte("structured ancestor"))
			if label == "contract-drift" {
				ancestor.ParserContract = "illegal parser drift"
			}
			if label != "missing" {
				n.HistoricalPackages = append(n.HistoricalPackages, ancestor)
			}
			if label == "ambiguous" {
				conflict := ancestor
				conflict.ParserContract = "conflicting exact identity"
				n.HistoricalPackages = append(n.HistoricalPackages, conflict)
			}
			if err := registerStructuredHistoricalPackages(b, n, c, ancestor); err == nil {
				t.Fatal("invalid structured ancestry accepted")
			}
			if len(c.HistoricalTranslationPackages) != 0 {
				t.Fatal("failed registration published partial authority")
			}
		})
	}
}

// Opt-in readonly real-authority evidence; never creates language artifacts.
func TestStructuredHistoricalPackageRegistrationCurrentAuthority(t *testing.T) {
	if os.Getenv("GO_TOUR_STRUCTURED_HISTORICAL_DIAG") != "1" {
		t.Skip("readonly real-authority diagnosis is opt-in")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	b, err := adoptionBase(root)
	if err != nil {
		t.Fatal(err)
	}
	n, err := loadSourceCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, old := range n.HistoricalPackages {
		after, err := packageByID(n, old.ID)
		if err != nil {
			continue
		}
		before, err := packageByID(b, old.ID)
		if err != nil || before.Identity == after.Identity {
			continue
		}
		contract := old
		contract.Identity, contract.SourceIdentity = after.Identity, after.SourceIdentity
		if reflect.DeepEqual(contract, after) {
			continue
		}
		x, _ := json.Marshal(contract)
		y, _ := json.Marshal(after)
		var left, right map[string]json.RawMessage
		json.Unmarshal(x, &left)
		json.Unmarshal(y, &right)
		diff := map[string][2]json.RawMessage{}
		for key, value := range right {
			if !reflect.DeepEqual(left[key], value) {
				diff[key] = [2]json.RawMessage{left[key], value}
			}
		}
		out, _ := json.Marshal(map[string]any{"package": old.ID, "historical_identity": old.Identity, "structured_base_identity": before.Identity, "current_identity": after.Identity, "differing_fields": diff})
		t.Log("TRIGGER=" + string(out))
	}
}

func historicalRegistrationFixture(t *testing.T) (*Global, *Global, *i18n.Catalog) {
	t.Helper()
	raw := []byte("Synthetic formal Page source")
	p := Package{ID: "future-page-package", Identity: digest([]byte("old package")), SourceIdentity: digest([]byte("old source graph"))}
	next := p
	next.Identity, next.SourceIdentity = digest([]byte("current package")), digest([]byte("structured expansion"))
	s := Source{Path: "_content/future.html", Package: p.ID, Kind: "page", SHA256: digest(raw)}
	base := &Global{Packages: []Package{p}, Sources: []Source{s}}
	current := &Global{Packages: []Package{next}, Sources: []Source{s}, HistoricalPackages: []Package{p}}
	id := digest([]byte("formal Page ID"))
	u := i18n.AdaptedTranslationUnit{Unit: i18n.TranslationUnit{ID: id, Kind: i18n.UnitKindPage, SourcePath: s.Path, Source: raw, SourceSHA256: s.SHA256}, Adapter: PageAdapter{}, StableIndex: 1, Route: "/future/"}
	c := &i18n.Catalog{}
	if err := c.RegisterTranslationPackage(i18n.TranslationPackageDescriptor{ID: p.ID, AuthoritySHA256: next.Identity, Units: []i18n.AdaptedTranslationUnit{u}}); err != nil {
		t.Fatal(err)
	}
	return base, current, c
}

func TestStructuredHistoricalPackageRegistration(t *testing.T) {
	base, current, c := historicalRegistrationFixture(t)
	before := c.TranslationPackages[0]
	if _, err := i18n.HistoricalTranslationPackageCatalog(c, before.ID, base.Packages[0].Identity); err == nil {
		t.Fatal("unregistered authority accepted")
	}
	if err := registerStructuredHistoricalPackages(base, current, c); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, c.TranslationPackages[0]) {
		t.Fatal("current descriptor changed")
	}
	h, err := i18n.HistoricalTranslationPackageCatalog(c, before.ID, base.Packages[0].Identity)
	if err != nil {
		t.Fatal(err)
	}
	want := before
	want.AuthoritySHA256 = base.Packages[0].Identity
	if !reflect.DeepEqual(h.TranslationPackages, []i18n.TranslationPackageDescriptor{want}) || !reflect.DeepEqual(h.TranslationScope, []string{before.Units[0].Unit.ID}) {
		t.Fatal("exact historical descriptor/source/ordered membership changed")
	}
	if h.TranslationPackages[0].AuthoritySHA256 == c.TranslationPackages[0].AuthoritySHA256 {
		t.Fatal("historical authority became current")
	}
	for _, key := range [][2]string{{before.ID, digest([]byte("wrong SHA"))}, {"unknown-package", before.AuthoritySHA256}} {
		if _, err := i18n.HistoricalTranslationPackageCatalog(c, key[0], key[1]); err == nil {
			t.Fatal("wrong authority accepted")
		}
	}
}

func TestStructuredHistoricalPackageRegistrationRejectsMismatch(t *testing.T) {
	for label, mutate := range map[string]func(*Global, *Global, *i18n.Catalog){
		"missing-frozen-authority":              func(b, n *Global, c *i18n.Catalog) { n.HistoricalPackages = nil },
		"wrong-frozen-authority":                func(b, n *Global, c *i18n.Catalog) { n.HistoricalPackages[0].Identity = digest([]byte("wrong")) },
		"current-cannot-impersonate-historical": func(b, n *Global, c *i18n.Catalog) { c.TranslationPackages[0].AuthoritySHA256 = b.Packages[0].Identity },
		"changed-Page-source":                   func(b, n *Global, c *i18n.Catalog) { n.Sources[0].SHA256 = digest([]byte("changed source")) },
		"wrong-unit-source": func(b, n *Global, c *i18n.Catalog) {
			c.TranslationPackages[0].Units[0].Unit.Source = []byte("wrong bytes")
		},
		"changed-package-contract": func(b, n *Global, c *i18n.Catalog) { n.Packages[0].ParserContract = "different" },
	} {
		t.Run(label, func(t *testing.T) {
			b, n, c := historicalRegistrationFixture(t)
			mutate(b, n, c)
			if err := registerStructuredHistoricalPackages(b, n, c); err == nil {
				t.Fatal("unproven historical binding accepted")
			}
			if len(c.HistoricalTranslationPackages) != 0 {
				t.Fatal("failed registration published descriptor")
			}
		})
	}
}
