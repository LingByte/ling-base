package parser

import (
	"archive/zip"
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestODTParser_Basic(t *testing.T) {
	// Build a minimal ODT (zip with content.xml).
	contentXML := `<?xml version="1.0" encoding="UTF-8"?>
<office:document-content xmlns:office="urn:oasis:names:tc:opendocument:xmlns:office:1.0" xmlns:text="urn:oasis:names:tc:opendocument:xmlns:text:1.0">
<office:body><office:text>
<text:p>Hello ODT World!</text:p>
<text:p>This is the second paragraph.</text:p>
</office:text></office:body>
</office:document-content>`

	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	f, _ := w.Create("content.xml")
	f.Write([]byte(contentXML))
	w.Close()

	p := &ODTParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeODT,
		FileName: "test.odt",
		Content:  buf.Bytes(),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeODT, res.FileType)
	assert.Contains(t, res.Text, "Hello ODT World!")
	assert.Contains(t, res.Text, "second paragraph")
}

func TestODTParser_NoContentXML(t *testing.T) {
	buf := &bytes.Buffer{}
	w := zip.NewWriter(buf)
	f, _ := w.Create("mimetype")
	f.Write([]byte("application/vnd.oasis.opendocument.text"))
	w.Close()

	p := &ODTParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeODT,
		FileName: "test.odt",
		Content:  buf.Bytes(),
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "content.xml")
}

func TestODTParser_NilRequest(t *testing.T) {
	p := &ODTParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestODTParser_SupportedTypes(t *testing.T) {
	p := &ODTParser{}
	types := p.SupportedTypes()
	assert.Contains(t, types, FileTypeODT)
	assert.Contains(t, types, FileTypeODS)
	assert.Contains(t, types, FileTypeODP)
	assert.Equal(t, FileTypeODT, p.Provider())
}
