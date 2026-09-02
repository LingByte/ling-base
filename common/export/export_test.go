// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package export

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleRows() []map[string]any {
	return []map[string]any{
		{"name": "Alice", "age": 30, "city": "北京"},
		{"name": "Bob", "age": 25, "city": "上海"},
	}
}

func TestExportCSV(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportCSV(sampleRows(), &buf))
	out := buf.String()
	// BOM present
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, buf.Bytes()[:3])
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "Bob")
	assert.Contains(t, out, "北京")
}

func TestExportCSV_NoBOM(t *testing.T) {
	no := false
	exp := CSVExporter{WithBOM: &no}
	var buf bytes.Buffer
	require.NoError(t, exp.Export(sampleRows(), &buf))
	assert.NotEqual(t, []byte{0xEF, 0xBB, 0xBF}, buf.Bytes()[:3])
}

func TestExportCSV_CustomHeaders(t *testing.T) {
	exp := CSVExporter{Headers: []string{"name", "age"}}
	var buf bytes.Buffer
	require.NoError(t, exp.Export(sampleRows(), &buf))
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Contains(t, lines[0], "name")
	assert.Contains(t, lines[0], "age")
	assert.NotContains(t, lines[0], "city")
}

func TestExportCSV_Empty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportCSV(nil, &buf))
	out := buf.String()
	// BOM + empty header line
	assert.True(t, strings.HasSuffix(out, "\n"))
}

func TestExportExcel(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportExcel(sampleRows(), &buf))
	assert.NotEmpty(t, buf.Bytes())
	// xlsx files start with the ZIP local file header PK\x03\x04
	assert.Equal(t, "PK", string(buf.Bytes()[:2]))
}

func TestExportExcel_CustomSheet(t *testing.T) {
	exp := ExcelExporter{SheetName: "Data", Headers: []string{"name", "age"}}
	var buf bytes.Buffer
	require.NoError(t, exp.Export(sampleRows(), &buf))
	assert.NotEmpty(t, buf.Bytes())
}

func TestExportExcel_Empty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportExcel(nil, &buf))
	assert.Equal(t, "PK", string(buf.Bytes()[:2]))
}

func TestExportJSON(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportJSON(sampleRows(), &buf))
	out := buf.String()
	assert.Contains(t, out, `"name"`)
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "北京")
	assert.True(t, strings.HasPrefix(strings.TrimSpace(out), "["))
}

func TestExportJSON_Empty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportJSON(nil, &buf))
	assert.Equal(t, "[]\n", buf.String())
}

func TestExportMarkdown(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(sampleRows(), &buf))
	out := buf.String()
	assert.Contains(t, out, "| name |")
	assert.Contains(t, out, "| --- |")
	assert.Contains(t, out, "Alice")
	assert.Contains(t, out, "北京")
}

func TestExportMarkdown_CustomHeaders(t *testing.T) {
	exp := MarkdownExporter{Headers: []string{"name"}}
	var buf bytes.Buffer
	require.NoError(t, exp.Export(sampleRows(), &buf))
	out := buf.String()
	assert.Contains(t, out, "name")
	assert.NotContains(t, out, "city")
}

func TestExportMarkdown_Empty(t *testing.T) {
	var buf bytes.Buffer
	require.NoError(t, ExportMarkdown(nil, &buf))
	out := buf.String()
	// header + separator only
	lines := strings.Split(strings.TrimSpace(out), "\n")
	assert.Len(t, lines, 2)
}

func TestExporterInterface(t *testing.T) {
	var _ Exporter = CSVExporter{}
	var _ Exporter = ExcelExporter{}
	var _ Exporter = JSONExporter{}
	var _ Exporter = MarkdownExporter{}
}

func TestExportFile_CSV(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "out.csv")
	require.NoError(t, ExportFile("csv", sampleRows(), fn))
	data, err := os.ReadFile(fn)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Alice")
}

func TestExportFile_ByExtension(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		ext   string
		check string
	}{
		{"csv", "Alice"},
		{"json", "Alice"},
		{"md", "Alice"},
	}
	for _, c := range cases {
		fn := filepath.Join(dir, "out."+c.ext)
		require.NoError(t, ExportFile("", sampleRows(), fn), c.ext)
		data, err := os.ReadFile(fn)
		require.NoError(t, err, c.ext)
		assert.Contains(t, string(data), c.check, c.ext)
	}
}

func TestExportFile_ExcelByExtension(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "out.xlsx")
	require.NoError(t, ExportFile("", sampleRows(), fn))
	data, err := os.ReadFile(fn)
	require.NoError(t, err)
	assert.Equal(t, "PK", string(data[:2]))
}

func TestExportFile_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "out.xyz")
	err := ExportFile("xyz", sampleRows(), fn)
	assert.Error(t, err)
}

func TestExportFile_UnknownExtension(t *testing.T) {
	dir := t.TempDir()
	fn := filepath.Join(dir, "out.unknown")
	err := ExportFile("", sampleRows(), fn)
	assert.Error(t, err)
}

func TestResolveHeaders_Configured(t *testing.T) {
	h := resolveHeaders([]string{"a", "b"}, sampleRows())
	assert.Equal(t, []string{"a", "b"}, h)
}

func TestResolveHeaders_Derived(t *testing.T) {
	h := resolveHeaders(nil, sampleRows())
	assert.Contains(t, h, "name")
	assert.Contains(t, h, "age")
	assert.Contains(t, h, "city")
	// sorted
	assert.True(t, strings.Join(h, ",") == strings.Join(sortedCopy(h), ","))
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	// sort
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}

func TestToStr(t *testing.T) {
	assert.Equal(t, "", toStr(nil))
	assert.Equal(t, "hello", toStr("hello"))
	assert.Equal(t, "42", toStr(42))
	assert.Equal(t, "3.14", toStr(3.14))
}
