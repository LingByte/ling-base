package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSVGParser_Basic(t *testing.T) {
	p := &SVGParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeSVG,
		FileName: "test.svg",
		Content: []byte(`<?xml version="1.0"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100" height="50">
  <text x="10" y="20">Hello SVG</text>
  <text x="10" y="40">Text extraction</text>
</svg>`),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeSVG, res.FileType)
	assert.Contains(t, res.Text, "Hello SVG")
	assert.Contains(t, res.Text, "Text extraction")
}

func TestSVGParser_Malformed(t *testing.T) {
	p := &SVGParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeSVG,
		FileName: "test.svg",
		Content:  []byte(`<svg><text>Hello</text>`),
	}, nil)
	require.NoError(t, err)
	assert.Contains(t, res.Text, "Hello")
}

func TestSVGParser_Empty(t *testing.T) {
	p := &SVGParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeSVG,
		FileName: "test.svg",
		Content:  []byte(""),
	}, nil)
	require.Error(t, err)
	assert.True(t, err == ErrEmptyInput)
}

func TestSVGParser_NilRequest(t *testing.T) {
	p := &SVGParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestSVGParser_SupportedTypes(t *testing.T) {
	p := &SVGParser{}
	assert.Equal(t, []string{FileTypeSVG}, p.SupportedTypes())
	assert.Equal(t, FileTypeSVG, p.Provider())
}
