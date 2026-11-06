package sitecontent

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
)

// The immutable adoption inventory is the actual pre-fix source-owned
// representation, not a guessed mapping reconstructed from target prose.
func frozenEntryPages(t *testing.T) []Document {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repositoryRoot(t), "data/content-package-adoptions/receipts/7eb752cada868bec3cc8299db5976aacb809a773f70ff91764afe08ece680ed4.json"))
	if err != nil {
		t.Fatal(err)
	}
	var a PagePackageAdoption
	if err := StrictJSON(b, &a); err != nil {
		t.Fatal(err)
	}
	return a.Documents
}

func TestAdoptedPageRepresentationSuccessor(t *testing.T) {
	root, proposal := pagePackageAdoptionFixture(t)
	base, err := adoptionBase(root)
	if err != nil {
		t.Fatal(err)
	}
	old := frozenEntryPages(t)
	a, err := buildPagePackageAdoption(root, base, proposal, old)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Encode(a)
	ref := Reference{PagePackageAdoptionPath(a), digest(b)}
	writeFixture(t, root, ref.Path, b)
	pointer, _ := Encode(ref)
	writeFixture(t, root, pagePackageCurrent(a.Package.ID), pointer)
	if _, err := CheckPagePackageAdoption(root, ref); err != nil {
		t.Fatal("frozen source adoption must survive protector evolution", err)
	}
	c := &i18n.Catalog{}
	if _, err := AttachAdoptedPageTranslationUnits(root, c); err != nil {
		t.Fatal(err)
	}
	if err := c.UseTranslationPackage(a.Package.ID); err != nil {
		t.Fatal(err)
	}
	frozen, _ := Encode(HistoricalPageRepresentations{Package: a.Package.ID, PackageSHA256: a.Package.Identity, Documents: old})
	writeFixture(t, root, "frozen-pages.json", frozen)
	r, err := resolvePageReplacements(root, c, i18n.GlossaryArchiveReference{Path: "frozen-pages.json", SHA256: digest(frozen)})
	if err != nil {
		t.Fatal(err)
	}
	for index, d := range old {
		p := r.Units[d.Units[0].ID]
		if (p.OldUnitID != p.CurrentUnitID) != (index == 0) {
			t.Fatal("only navigation Page may require replacement", p)
		}
		a := c.Adapted[index].Adapter.(adoptedPageAdapter)
		before := PageAdapter{d, a.Source}
		baseline, err := a.RebaseRevisionPredecessor(before, []byte(d.Units[0].Source))
		if err != nil {
			t.Fatal(err)
		}
		restored, err := a.Restore(string(baseline))
		if err != nil || !bytes.Equal(restored, a.Source) {
			t.Fatal("source-owned rebase changed predecessor bytes", err)
		}
	}
	for _, kind := range []string{"source", "coordinate", "protected-atom", "membership"} {
		t.Run(kind, func(t *testing.T) {
			var forged PagePackageAdoption
			_ = json.Unmarshal(b, &forged)
			switch kind {
			case "source":
				forged.Documents[0].SourceSHA = digest([]byte("other"))
			case "coordinate":
				forged.Documents[0].Slots[0].Start++
			case "protected-atom":
				forged.Documents[0].Units[0].Protected[0].Raw += "changed"
			case "membership":
				forged.Documents[0], forged.Documents[1] = forged.Documents[1], forged.Documents[0]
			}
			forged.Identity = ""
			forged.Identity = identity(forged)
			bad, _ := Encode(forged)
			ref := Reference{PagePackageAdoptionPath(&forged), digest(bad)}
			writeFixture(t, root, ref.Path, bad)
			if _, err := CheckPagePackageAdoption(root, ref); err == nil {
				t.Fatal("rehash did not excuse corrupted frozen foundation")
			}
		})
	}
	unchanged, err := readRegular(root, PagePackageAdoptionPath(a))
	if err != nil || !reflect.DeepEqual(unchanged, b) {
		t.Fatal("historical adoption changed", err)
	}
}
