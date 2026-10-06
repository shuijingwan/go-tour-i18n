package sitecontent

import (
	"bytes"
	"reflect"
	"testing"
)

func TestSiteFoundationAuthorityUpgradeIdempotent(t *testing.T) {
	root := siteFixture(t)
	before, _ := readRegular(root, GlobalPath)
	plan, err := UpgradeFoundation(root, true)
	if err != nil || !plan.AlreadyCurrent {
		t.Fatal("current authority upgrade", err)
	}
	after, _ := readRegular(root, GlobalPath)
	if !bytes.Equal(before, after) {
		t.Fatal("idempotent check mutated authority")
	}
	global, err := LoadCurrent(root)
	if err != nil {
		t.Fatal(err)
	}
	if global.UpstreamCommit != "db076098077c07d3cef1b85a2cf56ff52777f587" {
		t.Fatal("frozen commit changed")
	}
	pkg, err := packageByID(global, "site-v2-shell")
	if err != nil || pkg.ParserContract != UnitContract {
		t.Fatal("missing source/parser authority", err)
	}
	docs, _, err := PackageDocuments(root, global, "site-v2-shell")
	if err != nil || len(docs) != 2 {
		t.Fatal("shell source closure", err)
	}
	tour, _ := packageByID(global, "tour-v1")
	if tour.ParserContract != "" {
		t.Fatal("Tour parser migration")
	}
	broken := *global
	broken.PublicName = "different"
	b, _ := Encode(broken)
	writeFixture(t, root, GlobalPath, b)
	if _, err := UpgradeFoundation(root, true); err == nil {
		t.Fatal("unrelated source authority overwritten")
	}
	same, _ := readRegular(root, GlobalPath)
	if !reflect.DeepEqual(b, same) {
		t.Fatal("failed upgrade mutated")
	}
}
