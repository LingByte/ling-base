package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXMLParser_Basic(t *testing.T) {
	p := &XMLParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeXML,
		FileName: "test.xml",
		Content:  []byte(`<?xml version="1.0"?><root><name>hello</name><desc>world</desc></root>`),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeXML, res.FileType)
	assert.Contains(t, res.Text, "hello")
	assert.Contains(t, res.Text, "world")
}

func TestXMLParser_Malformed(t *testing.T) {
	p := &XMLParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeXML,
		FileName: "test.xml",
		Content:  []byte(`<root><name>hello</desc></root>`),
	}, nil)
	require.NoError(t, err)
	assert.Contains(t, res.Text, "hello")
}

func TestXMLParser_Empty(t *testing.T) {
	p := &XMLParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeXML,
		FileName: "test.xml",
		Content:  []byte("   "),
	}, nil)
	require.Error(t, err)
	assert.True(t, err == ErrEmptyInput)
}

func TestXMLParser_NilRequest(t *testing.T) {
	p := &XMLParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestXMLParser_SupportedTypes(t *testing.T) {
	p := &XMLParser{}
	assert.Equal(t, []string{FileTypeXML}, p.SupportedTypes())
	assert.Equal(t, FileTypeXML, p.Provider())
}
