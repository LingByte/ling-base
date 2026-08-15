package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVCFParser_Basic(t *testing.T) {
	p := &VCFParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeVCF,
		FileName: "test.vcf",
		Content: []byte(`BEGIN:VCARD
VERSION:3.0
FN:Alice Smith
N:Smith;Alice;;;
TEL;TYPE=WORK:+1-555-0100
EMAIL:alice@example.com
ORG:Example Corp
END:VCARD`),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeVCF, res.FileType)
	assert.Contains(t, res.Text, "Alice Smith")
	assert.Contains(t, res.Text, "+1-555-0100")
	assert.Contains(t, res.Text, "alice@example.com")
	assert.Contains(t, res.Text, "Example Corp")
}

func TestVCFParser_MultipleContacts(t *testing.T) {
	p := &VCFParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeVCF,
		FileName: "test.vcf",
		Content: []byte(`BEGIN:VCARD
FN:Alice
END:VCARD
BEGIN:VCARD
FN:Bob
END:VCARD`),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Contains(t, res.Text, "Contact 1")
	assert.Contains(t, res.Text, "Contact 2")
	assert.Contains(t, res.Text, "Alice")
	assert.Contains(t, res.Text, "Bob")
}

func TestVCFParser_Empty(t *testing.T) {
	p := &VCFParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeVCF,
		FileName: "test.vcf",
		Content:  []byte(""),
	}, nil)
	require.Error(t, err)
	assert.True(t, err == ErrEmptyInput)
}

func TestVCFParser_NilRequest(t *testing.T) {
	p := &VCFParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestVCFParser_SupportedTypes(t *testing.T) {
	p := &VCFParser{}
	assert.Equal(t, []string{FileTypeVCF}, p.SupportedTypes())
	assert.Equal(t, FileTypeVCF, p.Provider())
}
