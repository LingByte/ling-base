// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package codegen_test

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/LingByte/ling-base/common/codegen"
)

const testSource = `package main

import (
	"fmt"
	"strings"
)

type User struct {
	Name string
	Age  int
}

func main() {
	fmt.Println("hello")
}

func helper() string {
	return strings.ToUpper("test")
}
`

func parseTest(t *testing.T) (*token.FileSet, *ast.File) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := codegen.ParseFileFromSource(fset, "test.go", testSource)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return fset, file
}

func TestParseFile(t *testing.T) {
	fset, file := parseTest(t)
	if file.Name.Name != "main" {
		t.Errorf("Package name = %q, want main", file.Name.Name)
	}
	_ = fset
}

func TestHasImport(t *testing.T) {
	_, file := parseTest(t)

	if !codegen.HasImport(file, "fmt") {
		t.Error("HasImport(fmt) should be true")
	}
	if !codegen.HasImport(file, "strings") {
		t.Error("HasImport(strings) should be true")
	}
	if codegen.HasImport(file, "os") {
		t.Error("HasImport(os) should be false")
	}
}

func TestListImports(t *testing.T) {
	_, file := parseTest(t)
	imports := codegen.ListImports(file)
	if len(imports) != 2 {
		t.Fatalf("ListImports count = %d, want 2", len(imports))
	}
}

func TestAddImport(t *testing.T) {
	_, file := parseTest(t)

	codegen.AddImport(file, "os")
	if !codegen.HasImport(file, "os") {
		t.Error("HasImport(os) should be true after AddImport")
	}

	// Adding existing import should not duplicate.
	codegen.AddImport(file, "fmt")
	count := 0
	for _, imp := range file.Imports {
		if imp.Path.Value == `"fmt"` {
			count++
		}
	}
	if count != 1 {
		t.Errorf("fmt import count = %d, want 1", count)
	}
}

func TestAddImportWithAlias(t *testing.T) {
	_, file := parseTest(t)

	codegen.AddImportWithAlias(file, "github.com/x/y", "y")
	if !codegen.HasImport(file, "github.com/x/y") {
		t.Error("HasImport should be true after AddImportWithAlias")
	}
}

func TestRemoveImport(t *testing.T) {
	_, file := parseTest(t)

	removed := codegen.RemoveImport(file, "strings")
	if !removed {
		t.Error("RemoveImport should return true for existing import")
	}
	if codegen.HasImport(file, "strings") {
		t.Error("HasImport(strings) should be false after removal")
	}

	removed = codegen.RemoveImport(file, "nonexistent")
	if removed {
		t.Error("RemoveImport should return false for non-existent import")
	}
}

func TestFindFunction(t *testing.T) {
	_, file := parseTest(t)

	fn := codegen.FindFunction(file, "main")
	if fn == nil {
		t.Fatal("FindFunction(main) returned nil")
	}
	if fn.Name.Name != "main" {
		t.Errorf("Function name = %q, want main", fn.Name.Name)
	}

	fn = codegen.FindFunction(file, "helper")
	if fn == nil {
		t.Fatal("FindFunction(helper) returned nil")
	}

	fn = codegen.FindFunction(file, "nonexistent")
	if fn != nil {
		t.Error("FindFunction(nonexistent) should return nil")
	}
}

func TestListFunctions(t *testing.T) {
	_, file := parseTest(t)
	fns := codegen.ListFunctions(file)
	if len(fns) != 2 {
		t.Fatalf("ListFunctions count = %d, want 2", len(fns))
	}
}

func TestFindTypeDecl(t *testing.T) {
	_, file := parseTest(t)

	ts := codegen.FindTypeDecl(file, "User")
	if ts == nil {
		t.Fatal("FindTypeDecl(User) returned nil")
	}

	ts = codegen.FindTypeDecl(file, "NonExistent")
	if ts != nil {
		t.Error("FindTypeDecl(NonExistent) should return nil")
	}
}

func TestCreateStmt(t *testing.T) {
	stmt, err := codegen.CreateStmt("fmt.Println(\"test\")")
	if err != nil {
		t.Fatalf("CreateStmt: %v", err)
	}
	if stmt == nil {
		t.Fatal("CreateStmt returned nil")
	}
}

func TestCreateStmt_Invalid(t *testing.T) {
	_, err := codegen.CreateStmt("not valid go !!!")
	if err == nil {
		t.Error("CreateStmt should fail for invalid expression")
	}
}

func TestCreateCallExpr(t *testing.T) {
	call, err := codegen.CreateCallExpr("fmt.Println(\"hello\")")
	if err != nil {
		t.Fatalf("CreateCallExpr: %v", err)
	}
	if call == nil {
		t.Fatal("CreateCallExpr returned nil")
	}
}

func TestCreateCallExpr_NotACall(t *testing.T) {
	_, err := codegen.CreateCallExpr("x")
	if err == nil {
		t.Error("CreateCallExpr should fail for non-call expression")
	}
}

func TestIsBlockStmt(t *testing.T) {
	_, file := parseTest(t)
	fn := codegen.FindFunction(file, "main")
	if fn == nil || fn.Body == nil {
		t.Fatal("main function or body is nil")
	}
	if !codegen.IsBlockStmt(fn.Body) {
		t.Error("Function body should be a BlockStmt")
	}
}

func TestVariableExistsInBlock(t *testing.T) {
	fset := token.NewFileSet()
	file, err := codegen.ParseFileFromSource(fset, "test.go", `package main
func test() {
	x := 1
	y := 2
	_ = x
	_ = y
}
`)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	fn := codegen.FindFunction(file, "test")
	if fn == nil {
		t.Fatal("test function not found")
	}

	if !codegen.VariableExistsInBlock(fn.Body, "x") {
		t.Error("Variable x should exist in block")
	}
	if !codegen.VariableExistsInBlock(fn.Body, "y") {
		t.Error("Variable y should exist in block")
	}
	if codegen.VariableExistsInBlock(fn.Body, "z") {
		t.Error("Variable z should not exist in block")
	}
}

func TestPrintNode(t *testing.T) {
	fset, file := parseTest(t)
	output, err := codegen.PrintNode(fset, file)
	if err != nil {
		t.Fatalf("PrintNode: %v", err)
	}
	if output == "" {
		t.Error("PrintNode returned empty string")
	}
}

// ──────────────────────────────────────────────
// Literal builders
// ──────────────────────────────────────────────

func TestStringLit(t *testing.T) {
	lit := codegen.StringLit("hello")
	if lit.Value != `"hello"` {
		t.Errorf("StringLit value = %q, want %q", lit.Value, `"hello"`)
	}
}

func TestIntLit(t *testing.T) {
	lit := codegen.IntLit(42)
	if lit.Value != "42" {
		t.Errorf("IntLit value = %q, want 42", lit.Value)
	}
}

func TestBoolLit(t *testing.T) {
	lit := codegen.BoolLit(true)
	if lit.Name != "true" {
		t.Errorf("BoolLit(true).Name = %q, want true", lit.Name)
	}
	lit = codegen.BoolLit(false)
	if lit.Name != "false" {
		t.Errorf("BoolLit(false).Name = %q, want false", lit.Name)
	}
}

func TestSelectorExpr(t *testing.T) {
	sel := codegen.SelectorExpr("model", "User")
	if sel.X.(*ast.Ident).Name != "model" {
		t.Error("SelectorExpr X name wrong")
	}
	if sel.Sel.Name != "User" {
		t.Error("SelectorExpr Sel name wrong")
	}
}

func TestKeyValueExpr(t *testing.T) {
	kv := codegen.KeyValueExpr("Name", codegen.StringLit("test"))
	if kv.Key.(*ast.Ident).Name != "Name" {
		t.Error("KeyValueExpr key wrong")
	}
}

func TestCompositeLit(t *testing.T) {
	lit := codegen.CompositeLit(
		codegen.SelectorExpr("model", "User"),
		[]ast.Expr{
			codegen.KeyValueExpr("Name", codegen.StringLit("alice")),
		},
	)
	if lit.Type == nil {
		t.Error("CompositeLit type is nil")
	}
	if len(lit.Elts) != 1 {
		t.Errorf("CompositeLit elts count = %d, want 1", len(lit.Elts))
	}
}
