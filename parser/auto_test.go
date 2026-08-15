package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectFileType_ByExt(t *testing.T) {
	assert.Equal(t, FileTypeTXT, DetectFileType(&ParseRequest{FileName: "a.TXT"}))
	assert.Equal(t, FileTypeYML, DetectFileType(&ParseRequest{FileName: "a.yml"}))
	assert.Equal(t, FileTypeDOCX, DetectFileType(&ParseRequest{FileName: "a.docx"}))
}

func TestRouter_Parse_RejectsDoc(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{FileName: "a.doc", Content: []byte("x")}, &ParseOptions{})
	assert.Error(t, err)
	assert.NotContains(t, err.Error(), "convert to .docx")
}

func TestRouter_Parse_Unsupported(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{FileName: "a.bin", Content: []byte("x")}, &ParseOptions{})
	assert.ErrorIs(t, err, ErrUnsupportedFileType)
}

func TestDetectFileType_VariousExtensions(t *testing.T) {
	tests := []struct {
		name     string
		fileName string
		want     string
	}{
		{"txt", "file.txt", FileTypeTXT},
		{"md", "file.md", FileTypeMD},
		{"markdown", "file.markdown", FileTypeMD},
		{"mdx", "file.mdx", FileTypeMDX},
		{"csv", "file.csv", FileTypeCSV},
		{"html", "file.html", FileTypeHTML},
		{"htm", "file.htm", FileTypeHTML},
		{"json", "file.json", FileTypeJSON},
		{"yaml", "file.yaml", FileTypeYAML},
		{"yml", "file.yml", FileTypeYML},
		{"eml", "file.eml", FileTypeEML},
		{"rtf", "file.rtf", FileTypeRTF},
		{"pdf", "file.pdf", FileTypePDF},
		{"png", "file.png", FileTypePNG},
		{"jpg", "file.jpg", FileTypeJPG},
		{"jpeg", "file.jpeg", FileTypeJPEG},
		{"webp", "file.webp", FileTypeWEBP},
		{"gif", "file.gif", FileTypeGIF},
		{"bmp", "file.bmp", FileTypeBMP},
		{"tif", "file.tif", FileTypeTIF},
		{"tiff", "file.tiff", FileTypeTIFF},
		{"doc", "file.doc", FileTypeDOC},
		{"docx", "file.docx", FileTypeDOCX},
		{"pptx", "file.pptx", FileTypePPTX},
		{"xlsx", "file.xlsx", FileTypeXLSX},
		{"epub", "file.epub", FileTypeEPUB},
		{"mhtml", "file.mhtml", FileTypeMHTML},
		{"mht", "file.mht", FileTypeMHT},
		{"wav", "file.wav", FileTypeWAV},
		{"mp3", "file.mp3", FileTypeMP3},
		{"ogg", "file.ogg", FileTypeOGG},
		{"flac", "file.flac", FileTypeFLAC},
		{"m4a", "file.m4a", FileTypeM4A},
		{"aac", "file.aac", FileTypeAAC},
		{"xml", "file.xml", FileTypeXML},
		{"toml", "file.toml", FileTypeTOML},
		{"tsv", "file.tsv", FileTypeTSV},
		{"ics", "file.ics", FileTypeICS},
		{"vcf", "file.vcf", FileTypeVCF},
		{"log", "file.log", FileTypeLOG},
		{"odt", "file.odt", FileTypeODT},
		{"ods", "file.ods", FileTypeODS},
		{"odp", "file.odp", FileTypeODP},
		{"ppt", "file.ppt", FileTypePPT},
		{"xls", "file.xls", FileTypeXLS},
		{"svg", "file.svg", FileTypeSVG},
		{"unknown", "file.xyz", FileTypeUnknown},
		{"no extension", "Makefile", FileTypeUnknown},
		{"uppercase ext", "FILE.TXT", FileTypeTXT},
		{"mixed case", "File.TxT", FileTypeTXT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectFileType(&ParseRequest{FileName: tt.fileName})
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDetectFileType_NilRequest(t *testing.T) {
	assert.Equal(t, FileTypeUnknown, DetectFileType(nil))
}

func TestDetectFileType_EmptyFileName_UsesPath(t *testing.T) {
	assert.Equal(t, FileTypeTXT, DetectFileType(&ParseRequest{Path: "/some/dir/file.txt"}))
}

func TestDetectFileType_EmptyFileNameAndPath(t *testing.T) {
	assert.Equal(t, FileTypeUnknown, DetectFileType(&ParseRequest{}))
}

func TestIsOCRFileType(t *testing.T) {
	tests := []struct {
		name string
		ft   string
		want bool
	}{
		{"png", FileTypePNG, true},
		{"jpg", FileTypeJPG, true},
		{"jpeg", FileTypeJPEG, true},
		{"webp", FileTypeWEBP, true},
		{"gif", FileTypeGIF, true},
		{"bmp", FileTypeBMP, true},
		{"tiff", FileTypeTIFF, true},
		{"tif", FileTypeTIF, true},
		{"txt", FileTypeTXT, false},
		{"pdf", FileTypePDF, false},
		{"mp3", FileTypeMP3, false},
		{"unknown", FileTypeUnknown, false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isOCRFileType(tt.ft))
		})
	}
}

func TestIsASRFileType(t *testing.T) {
	tests := []struct {
		name string
		ft   string
		want bool
	}{
		{"wav", FileTypeWAV, true},
		{"mp3", FileTypeMP3, true},
		{"ogg", FileTypeOGG, true},
		{"flac", FileTypeFLAC, true},
		{"m4a", FileTypeM4A, true},
		{"aac", FileTypeAAC, true},
		{"txt", FileTypeTXT, false},
		{"pdf", FileTypePDF, false},
		{"png", FileTypePNG, false},
		{"unknown", FileTypeUnknown, false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isASRFileType(tt.ft))
		})
	}
}

func TestRouter_Register_CustomParser(t *testing.T) {
	r := NewRouter()
	require.NotNil(t, r)

	// Register a custom parser
	custom := &mockParser{
		types: []string{"custom"},
	}
	err := r.Register(custom)
	require.NoError(t, err)

	// Verify it's used
	result, err := r.Parse(context.Background(), &ParseRequest{
		FileType: "custom",
		Content:  []byte("test"),
	}, &ParseOptions{})
	require.NoError(t, err)
	assert.Equal(t, "custom result", result.Text)
}

func TestRouter_Register_NilParser(t *testing.T) {
	r := NewRouter()
	err := r.Register(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil parser")
}

func TestRouter_Register_NilRouter(t *testing.T) {
	var r *Router
	err := r.Register(&TXTParser{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil router")
}

func TestRouter_Register_EmptyType(t *testing.T) {
	r := NewRouter()
	custom := &mockParser{
		types: []string{"", "  ", "valid"},
	}
	err := r.Register(custom)
	require.NoError(t, err)

	// "valid" should be registered, empty strings should be skipped
	result, err := r.Parse(context.Background(), &ParseRequest{
		FileType: "valid",
		Content:  []byte("test"),
	}, &ParseOptions{})
	require.NoError(t, err)
	assert.Equal(t, "custom result", result.Text)
}

func TestRouter_Register_OverwriteExisting(t *testing.T) {
	r := NewRouter()

	// Register first parser for "txt"
	first := &mockParser{types: []string{"txt"}, text: "first"}
	require.NoError(t, r.Register(first))

	// Overwrite with second parser
	second := &mockParser{types: []string{"txt"}, text: "second"}
	require.NoError(t, r.Register(second))

	result, err := r.Parse(context.Background(), &ParseRequest{
		FileType: "txt",
		Content:  []byte("test"),
	}, &ParseOptions{})
	require.NoError(t, err)
	assert.Equal(t, "second", result.Text)
}

func TestRouter_Parse_NilRequest(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), nil, &ParseOptions{})
	assert.ErrorIs(t, err, ErrEmptyInput)
}

func TestRouter_Parse_OCRFileType(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{
		FileName: "image.png",
		Content:  []byte("x"),
	}, &ParseOptions{})
	// OCR parser is registered but will fail because no provider is set
	// and the content is not a valid image
	assert.Error(t, err)
}

func TestRouter_Parse_ASRFileType(t *testing.T) {
	r := DefaultRouter()
	_, err := r.Parse(context.Background(), &ParseRequest{
		FileName: "audio.mp3",
		Content:  []byte("x"),
	}, &ParseOptions{})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnsupportedFileType)
	assert.Contains(t, err.Error(), "ASR")
}

func TestRouter_Parse_AutoDetectsFileType(t *testing.T) {
	r := DefaultRouter()
	result, err := r.Parse(context.Background(), &ParseRequest{
		FileName: "test.txt",
		Content:  []byte("hello world"),
	}, &ParseOptions{})
	require.NoError(t, err)
	assert.Equal(t, FileTypeTXT, result.FileType)
	assert.Contains(t, result.Text, "hello world")
}

func TestParseBytesWithMeta(t *testing.T) {
	text, meta, err := ParseBytesWithMeta(context.Background(), "test.txt", []byte("hello world"), &ParseOptions{})
	require.NoError(t, err)
	assert.Equal(t, "hello world", text)
	require.NotNil(t, meta)
	assert.Equal(t, "txt", meta.Format)
	assert.Equal(t, 11, meta.CharCount)
	assert.NotEmpty(t, meta.Preview)
}

func TestParseBytesWithMeta_NoExtension(t *testing.T) {
	// A file with no extension resolves to unknown type and should error
	_, _, err := ParseBytesWithMeta(context.Background(), "Makefile", []byte("hello"), &ParseOptions{})
	assert.Error(t, err)
}

func TestParseBytesWithMeta_Error(t *testing.T) {
	_, _, err := ParseBytesWithMeta(context.Background(), "file.bin", []byte("x"), &ParseOptions{})
	assert.Error(t, err)
}

func TestParseBytes(t *testing.T) {
	result, err := ParseBytes(context.Background(), "test.txt", []byte("hello"), &ParseOptions{})
	require.NoError(t, err)
	assert.Equal(t, FileTypeTXT, result.FileType)
	assert.Contains(t, result.Text, "hello")
}

func TestParsePath(t *testing.T) {
	// ParsePath with a non-existent file should return an error
	_, err := ParsePath(context.Background(), "/nonexistent/file.txt", &ParseOptions{})
	assert.Error(t, err)
}

func TestNewRouter(t *testing.T) {
	r := NewRouter(&TXTParser{}, &CSVParser{})
	require.NotNil(t, r)

	// TXT parser should be registered
	result, err := r.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeTXT,
		Content:  []byte("hello"),
	}, &ParseOptions{})
	require.NoError(t, err)
	assert.Contains(t, result.Text, "hello")
}

// mockParser is a simple parser for testing Router.Register
type mockParser struct {
	types []string
	text  string
}

func (m *mockParser) Provider() string {
	return "mock"
}

func (m *mockParser) SupportedTypes() []string {
	return m.types
}

func (m *mockParser) Parse(ctx context.Context, req *ParseRequest, opts *ParseOptions) (*ParseResult, error) {
	text := m.text
	if text == "" {
		text = "custom result"
	}
	return &ParseResult{
		FileType: "mock",
		FileName: req.FileName,
		Text:     text,
	}, nil
}
