# export

A small, uniform API for exporting tabular data (`[]map[string]any`) to CSV, Excel, JSON and Markdown formats.

## Exporter interface

```go
type Exporter interface {
    Export(rows []map[string]any, w io.Writer) error
}
```

## Exporters

- `CSVExporter{ Headers []string; WithBOM *bool }` — CSV export (UTF-8 BOM by default for Excel/Chinese compatibility)
- `ExcelExporter{ SheetName string; Headers []string }` — Excel (.xlsx) export via excelize
- `JSONExporter{}` — JSON array export
- `MarkdownExporter{ Headers []string }` — GitHub-flavored Markdown table export

## Shortcut functions

- `ExportCSV(rows, w)` / `ExportExcel(rows, w)` / `ExportJSON(rows, w)` / `ExportMarkdown(rows, w)`
- `ExportFile(format, rows, filename)` — export to a file; format inferred from extension when `format` is empty

Supported formats: `csv`, `excel`/`xlsx`, `json`, `markdown`/`md`.

## Quick start

```go
import "github.com/LingByte/ling-base/common/export"

rows := []map[string]any{
    {"name": "Alice", "age": 30},
    {"name": "Bob", "age": 25},
}

var buf bytes.Buffer
_ = export.ExportCSV(rows, &buf)

// Or write to a file directly:
_ = export.ExportFile("csv", rows, "users.csv")
_ = export.ExportFile("", rows, "users.xlsx") // format from extension
```

## CSV BOM

`ExportCSV` and `CSVExporter` write a UTF-8 BOM (`EF BB BF`) by default so that Excel opens UTF-8 CSV files containing Chinese characters correctly. Set `WithBOM: &false` to disable.

## Dependencies

- `github.com/xuri/excelize/v2` v2.9.0 (Excel)
- standard library `encoding/csv` (CSV), `encoding/json` (JSON)
