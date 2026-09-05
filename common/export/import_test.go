// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package export

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestImportExcel_RoundTrip(t *testing.T) {
	// Export then import to verify round-trip.
	rows := []map[string]any{
		{"name": "Alice", "age": "30", "email": "alice@example.com"},
		{"name": "Bob", "age": "25", "email": "bob@example.com"},
	}
	var buf bytes.Buffer
	if err := ExportExcel(rows, &buf); err != nil {
		t.Fatalf("ExportExcel: %v", err)
	}

	imported, err := ImportExcel(&buf)
	if err != nil {
		t.Fatalf("ImportExcel: %v", err)
	}
	if len(imported) != 2 {
		t.Fatalf("imported %d rows, want 2", len(imported))
	}
	if imported[0]["name"] != "Alice" {
		t.Errorf("row 0 name = %v, want Alice", imported[0]["name"])
	}
	if imported[1]["age"] != "25" {
		t.Errorf("row 1 age = %v, want 25", imported[1]["age"])
	}
}

func TestImportExcel_EmptyRows(t *testing.T) {
	rows := []map[string]any{
		{"name": "Alice", "age": "30"},
		{"name": "", "age": ""}, // empty row, should be skipped
		{"name": "Bob", "age": "25"},
	}
	var buf bytes.Buffer
	_ = ExportExcel(rows, &buf)

	imported, err := ImportExcel(&buf)
	if err != nil {
		t.Fatalf("ImportExcel: %v", err)
	}
	if len(imported) != 2 {
		t.Errorf("imported %d rows, want 2 (empty row skipped)", len(imported))
	}
}

func TestImportExcelInto(t *testing.T) {
	type User struct {
		Name  string    `excel:"Name"`
		Email string    `excel:"Email"`
		Age   int       `excel:"Age"`
		Born  time.Time `excel:"Born"`
	}

	rows := []map[string]any{
		{"Name": "Alice", "Email": "alice@example.com", "Age": "30", "Born": "1996-01-15"},
		{"Name": "Bob", "Email": "bob@example.com", "Age": "25", "Born": "2001-06-20"},
	}
	var buf bytes.Buffer
	if err := ExportExcel(rows, &buf); err != nil {
		t.Fatalf("ExportExcel: %v", err)
	}

	var users []User
	if err := ImportExcelInto(&buf, &users); err != nil {
		t.Fatalf("ImportExcelInto: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}
	if users[0].Name != "Alice" {
		t.Errorf("users[0].Name = %q", users[0].Name)
	}
	if users[0].Age != 30 {
		t.Errorf("users[0].Age = %d, want 30", users[0].Age)
	}
	if users[1].Email != "bob@example.com" {
		t.Errorf("users[1].Email = %q", users[1].Email)
	}
	if !users[0].Born.IsZero() && users[0].Born.Year() != 1996 {
		t.Errorf("users[0].Born = %v, want 1996", users[0].Born)
	}
}

func TestImportExcelInto_NotPointer(t *testing.T) {
	var users []struct{ Name string }
	err := ImportExcelInto(bytes.NewReader(nil), users)
	if err == nil {
		t.Error("expected error for non-pointer out")
	}
}

func TestImportExcelInto_NotSlicePointer(t *testing.T) {
	var user struct{ Name string }
	err := ImportExcelInto(bytes.NewReader(nil), &user)
	if err == nil {
		t.Error("expected error for non-slice pointer")
	}
}

func TestImportCSV(t *testing.T) {
	csvData := "name,age,email\nAlice,30,alice@example.com\nBob,25,bob@example.com\n"
	imported, err := ImportCSV(bytes.NewReader([]byte(csvData)))
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if len(imported) != 2 {
		t.Fatalf("imported %d rows, want 2", len(imported))
	}
	if imported[0]["name"] != "Alice" {
		t.Errorf("row 0 name = %v", imported[0]["name"])
	}
	if imported[1]["age"] != "25" {
		t.Errorf("row 1 age = %v", imported[1]["age"])
	}
}

func TestImportCSV_WithBOM(t *testing.T) {
	// CSV with UTF-8 BOM prefix.
	csvData := "\xEF\xBB\xBFname,age\nAlice,30\n"
	imported, err := ImportCSV(bytes.NewReader([]byte(csvData)))
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("imported %d rows, want 1", len(imported))
	}
	if imported[0]["name"] != "Alice" {
		t.Errorf("row 0 name = %q, want Alice (BOM should be stripped)", imported[0]["name"])
	}
}

func TestImportExcelFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.xlsx")

	rows := []map[string]any{
		{"name": "Alice", "age": "30"},
	}
	var buf bytes.Buffer
	_ = ExportExcel(rows, &buf)
	_ = os.WriteFile(path, buf.Bytes(), 0644)

	imported, err := ImportExcelFile(path)
	if err != nil {
		t.Fatalf("ImportExcelFile: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("imported %d rows, want 1", len(imported))
	}
	if imported[0]["name"] != "Alice" {
		t.Errorf("name = %v", imported[0]["name"])
	}
}

func TestImportCSVFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	_ = os.WriteFile(path, []byte("name,age\nAlice,30\n"), 0644)

	imported, err := ImportCSVFile(path)
	if err != nil {
		t.Fatalf("ImportCSVFile: %v", err)
	}
	if len(imported) != 1 {
		t.Fatalf("imported %d rows, want 1", len(imported))
	}
	if imported[0]["name"] != "Alice" {
		t.Errorf("name = %v", imported[0]["name"])
	}
}
