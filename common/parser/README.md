# parser

Document and media parser that extracts plain text from 30+ file formats for ingestion into search and LLM pipelines.

## Supported formats

Text (txt, md, mdx, csv, tsv, html, json, yaml, xml, toml, log), documents (pdf, doc, docx, pptx, xlsx, rtf, odt, epub), email (eml), calendars (ics), contacts (vcf), web archives (mhtml), images via OCR (png, jpg, webp, gif, bmp, tiff), audio via ASR (wav, mp3, ogg, flac, m4a, aac), and SVG.

## Key types

- `Parser` — interface (`Provider`, `SupportedTypes`, `Parse`)
- `ParseRequest` / `ParseResult` / `ParseOptions` — request, result, and options
- `Section` — a structured chunk of the parsed document
- `Router` — routes a request to the matching parser by file type
- `DetectFileType` — detects the file type from filename/extension

## Quick start

```go
import "github.com/LingByte/ling-base/common/parser"

// One-call helpers (auto-detect format).
result, err := parser.ParsePath(ctx, "report.pdf", nil)
result, err = parser.ParseBytes(ctx, "notes.md", data, nil)

// Explicit router with custom parsers.
r := parser.NewRouter(&parser.TXTParser{}, &parser.PDFParser{})
result, err = r.Parse(ctx, &parser.ParseRequest{FileType: "pdf", Content: data}, nil)
```
