package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTSVParser_Basic(t *testing.T) {
	p := &TSVParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeTSV,
		FileName: "test.tsv",
		Content:  []byte("name\tage\tcity\nAlice\t30\tNYC\nBob\t25\tLondon"),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeTSV, res.FileType)
	assert.Contains(t, res.Text, "Alice")
	assert.Contains(t, res.Text, "Bob")
}

func TestTSVParser_Empty(t *testing.T) {
	p := &TSVParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeTSV,
		FileName: "test.tsv",
		Content:  []byte(""),
	}, nil)
	require.Error(t, err)
	assert.True(t, err == ErrEmptyInput)
}

func TestTSVParser_NilRequest(t *testing.T) {
	p := &TSVParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestTSVParser_SupportedTypes(t *testing.T) {
	p := &TSVParser{}
	assert.Equal(t, []string{FileTypeTSV}, p.SupportedTypes())
	assert.Equal(t, FileTypeTSV, p.Provider())
}
