package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTOMLParser_Basic(t *testing.T) {
	p := &TOMLParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeTOML,
		FileName: "test.toml",
		Content: []byte(`
title = "test config"

[server]
host = "localhost"
port = 8080
debug = true
`),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeTOML, res.FileType)
	assert.Contains(t, res.Text, "test config")
	assert.Contains(t, res.Text, "server.host")
	assert.Contains(t, res.Text, "localhost")
	assert.Contains(t, res.Text, "server.port")
	assert.Contains(t, res.Text, "8080")
}

func TestTOMLParser_Invalid(t *testing.T) {
	p := &TOMLParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeTOML,
		FileName: "test.toml",
		Content:  []byte("this is not = valid = toml ="),
	}, nil)
	require.Error(t, err)
}

func TestTOMLParser_Empty(t *testing.T) {
	p := &TOMLParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeTOML,
		FileName: "test.toml",
		Content:  []byte(""),
	}, nil)
	require.Error(t, err)
	assert.True(t, err == ErrEmptyInput)
}

func TestTOMLParser_NilRequest(t *testing.T) {
	p := &TOMLParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestTOMLParser_SupportedTypes(t *testing.T) {
	p := &TOMLParser{}
	assert.Equal(t, []string{FileTypeTOML}, p.SupportedTypes())
	assert.Equal(t, FileTypeTOML, p.Provider())
}
