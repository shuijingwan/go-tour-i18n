package main

import (
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"testing"
)

func TestCommentResidualReviewScope(t *testing.T) {
	inventory := []commentInventoryUnit{{StableIndex: 1, Package: "tour-v1", Kind: i18n.UnitKindPage, ID: "p1"}, {StableIndex: 2, Package: "tour-v1", Kind: i18n.UnitKindPage, ID: "p2"}, {StableIndex: 3, Package: "tour-v1", Kind: i18n.UnitKindExample, ID: "e1"}}
	row := func(id string, review bool) commentAuditRow {
		return commentAuditRow{ID: id, CommentSlotComparison: i18n.CommentSlotComparison{StructureMatch: true, ProtectedUnchanged: true, ReviewRequired: review}}
	}
	audits := []commentLocaleAudit{{Locale: "zh-CN", Base: "zh", Packages: []commentPackageAudit{{Package: "tour-v1", State: "CURRENT", Rows: []commentAuditRow{row("e1", true), row("p1", true), row("p2", false)}}}}, {Locale: "en-GB", Base: "en", Packages: []commentPackageAudit{{Package: "tour-v1", State: "CURRENT", Rows: []commentAuditRow{row("p1", false)}}}}, {Locale: "unavailable", Packages: []commentPackageAudit{{Package: "tour-v1", State: "CURRENT_TARGET_UNAVAILABLE", Rows: []commentAuditRow{row("p1", true)}}}}}
	scope := commentReviewerScope(audits, inventory)
	if len(scope) != 1 || len(scope[0].Packages[0].Units) != 2 || scope[0].Packages[0].Units[0].ID != "p1" || scope[0].Packages[0].Units[1].ID != "e1" {
		t.Fatal(scope)
	}
	audits[0].Packages[0].Diagnostics = []string{"p2: comment/code structure mismatch"}
	scope = commentReviewerScope(audits, inventory)
	if len(scope[0].Packages[0].Units) != 3 {
		t.Fatal("block-only structural failure omitted", scope)
	}
}
