package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLOGParser_Basic(t *testing.T) {
	p := &LOGParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeLOG,
		FileName: "test.log",
		Content: []byte(`2026-01-15 10:00:00 INFO Application started
2026-01-15 10:00:01 ERROR Database connection failed
2026-01-15 10:00:02 WARN Retrying connection
2026-01-15 10:00:03 DEBUG Connection retry attempt 1`),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeLOG, res.FileType)
	assert.Contains(t, res.Text, "Application started")
	assert.Contains(t, res.Text, "Database connection failed")

	// Should have sections for different log levels.
	assert.True(t, len(res.Sections) > 1)
}

func TestLOGParser_LevelSections(t *testing.T) {
	p := &LOGParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeLOG,
		FileName: "test.log",
		Content:  []byte("INFO message one\nERROR message two\nFATAL message three"),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)

	// Find FATAL section
	var fatalSection *Section
	for i := range res.Sections {
		if res.Sections[i].Title == "FATAL" {
			fatalSection = &res.Sections[i]
			break
		}
	}
	require.NotNil(t, fatalSection)
	assert.Contains(t, fatalSection.Text, "message three")
}

func TestLOGParser_Empty(t *testing.T) {
	p := &LOGParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeLOG,
		FileName: "test.log",
		Content:  []byte(""),
	}, nil)
	require.Error(t, err)
	assert.True(t, err == ErrEmptyInput)
}

func TestLOGParser_NilRequest(t *testing.T) {
	p := &LOGParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestLOGParser_SupportedTypes(t *testing.T) {
	p := &LOGParser{}
	assert.Equal(t, []string{FileTypeLOG}, p.SupportedTypes())
	assert.Equal(t, FileTypeLOG, p.Provider())
}
