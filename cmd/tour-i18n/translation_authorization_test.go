package main

import (
	"encoding/json"
	"github.com/shuijingwan/go-tour-i18n/internal/i18n"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// Exact first-real-use set (excludes shell family and opt-in real QC proof):
// ^(TestTranslationReattestationCLIFlagsFailClosed|TestTranslationReattestationCLIReadonly|TestTranslationReattestationCLIDispatch|TestReattestCanonicalCLIUpgrade)$
func TestTranslationReattestationCLIFlagsFailClosed(t *testing.T) {
	for _, args := range [][]string{nil, {"--locale", "zh-CN"}, {"--snapshot-id", "accepted"}, {"--locale", "zh-CN", "--snapshot-id", "accepted", "extra"}} {
		if err := translationReattestationCommand(t.TempDir(), nil, args); err == nil || !strings.Contains(err.Error(), "required") {
			t.Fatalf("invalid flags accepted: %v: %v", args, err)
		}
	}
	if err := translationReattestationCommand(t.TempDir(), nil, []string{"--locale", "zh-CN", "--snapshot-id", "accepted", "--canonical"}); err == nil || !strings.Contains(err.Error(), "requires --apply") {
		t.Fatalf("canonical flag: %v", err)
	}
}

// This command's default mode must assess the exact fixture scope without
// publishing a checkpoint or touching unrelated repository inputs.
func TestTranslationReattestationCLIReadonly(t *testing.T) {
	root, c, snapshot := authorizationDiagnosisFixture(t, 2)
	if err := c.UseTranslationPackage(c.TranslationPackages[0].ID); err != nil {
		t.Fatal(err)
	}
	before := authorizationDiagnosisInventory(t, root)
	output, err := authorizationDiagnosisOutput(t, func() error {
		return translationReattestationCommand(root, c, []string{"--locale", "zh-CN", "--snapshot-id", snapshot})
	})
	if err != nil {
		t.Fatal(err)
	}
	var assessment i18n.CurrentTranslationAuthorization
	if err := json.Unmarshal(output, &assessment); err != nil {
		t.Fatal(err)
	}
	if !assessment.Ready() || assessment.Compatible != 2 || assessment.Affected+assessment.Stale+assessment.Unknown != 0 || assessment.Historical.Finalization.SnapshotID != snapshot {
		t.Fatalf("readonly exact scope assessment: %+v", assessment)
	}
	if !reflect.DeepEqual(before, authorizationDiagnosisInventory(t, root)) {
		t.Fatal("readonly reattest changed fixture")
	}
	if _, err := os.Stat(filepath.Join(root, i18n.CurrentTranslationAuthorizationPath(&assessment))); !os.IsNotExist(err) {
		t.Fatal("readonly reattest published checkpoint", err)
	}
}

// main.run requires full source discovery before dispatch. This bounded AST
// contract checks the exact switch/callee/forwarded arguments without replaying
// a repository lifecycle merely to prove command wiring.
func TestTranslationReattestationCLIDispatch(t *testing.T) {
	source, err := parser.ParseFile(token.NewFileSet(), "main.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	ast.Inspect(source, func(node ast.Node) bool {
		clause, ok := node.(*ast.CaseClause)
		if !ok {
			return true
		}
		for _, expr := range clause.List {
			lit, ok := expr.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil || value != "quality-check reattest" {
				continue
			}
			found++
			if len(clause.Body) != 1 {
				t.Fatal("reattest dispatch acquired unrelated behavior")
			}
			ret, ok := clause.Body[0].(*ast.ReturnStmt)
			if !ok || len(ret.Results) != 1 {
				t.Fatal("reattest dispatch must return command result")
			}
			call, ok := ret.Results[0].(*ast.CallExpr)
			if !ok || len(call.Args) != 3 {
				t.Fatal("reattest dispatch shape changed")
			}
			fn, ok := call.Fun.(*ast.Ident)
			if !ok || fn.Name != "translationReattestationCommand" {
				t.Fatal("reattest dispatched to another command")
			}
			for i, name := range []string{"root", "catalog"} {
				id, ok := call.Args[i].(*ast.Ident)
				if !ok || id.Name != name {
					t.Fatal("reattest context changed")
				}
			}
			tail, ok := call.Args[2].(*ast.SliceExpr)
			if !ok || tail.High != nil || tail.Slice3 {
				t.Fatal("reattest flags not forwarded exactly")
			}
			id, ok := tail.X.(*ast.Ident)
			if !ok || id.Name != "args" {
				t.Fatal("wrong argument source")
			}
			low, ok := tail.Low.(*ast.BasicLit)
			if !ok || low.Value != "2" {
				t.Fatal("wrong argument boundary")
			}
		}
		return true
	})
	if found != 1 {
		t.Fatalf("expected one exact quality-check reattest dispatch, got %d", found)
	}
}
