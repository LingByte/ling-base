package parser

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestICSParser_Basic(t *testing.T) {
	p := &ICSParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeICS,
		FileName: "test.ics",
		Content: []byte(`BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
SUMMARY:Team Meeting
DTSTART:20260115T100000Z
DTEND:20260115T110000Z
DESCRIPTION:Weekly sync
LOCATION:Room A
END:VEVENT
END:VCALENDAR`),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Equal(t, FileTypeICS, res.FileType)
	assert.Contains(t, res.Text, "Team Meeting")
	assert.Contains(t, res.Text, "Weekly sync")
	assert.Contains(t, res.Text, "Room A")
}

func TestICSParser_LineFolding(t *testing.T) {
	p := &ICSParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeICS,
		FileName: "test.ics",
		Content:  []byte("BEGIN:VCALENDAR\nBEGIN:VEVENT\nSUMMARY:This is a long\n  continued title\nEND:VEVENT\nEND:VCALENDAR"),
	}, &ParseOptions{PreserveLineBreaks: true})
	require.NoError(t, err)
	assert.Contains(t, res.Text, "This is a long continued title")
}

func TestICSParser_NoEvents(t *testing.T) {
	p := &ICSParser{}
	res, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeICS,
		FileName: "test.ics",
		Content:  []byte("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:test\nEND:VCALENDAR"),
	}, nil)
	require.NoError(t, err)
	// Should not panic, should return some text or empty.
	assert.NotNil(t, res)
}

func TestICSParser_Empty(t *testing.T) {
	p := &ICSParser{}
	_, err := p.Parse(context.Background(), &ParseRequest{
		FileType: FileTypeICS,
		FileName: "test.ics",
		Content:  []byte(""),
	}, nil)
	require.Error(t, err)
	assert.True(t, err == ErrEmptyInput)
}

func TestICSParser_NilRequest(t *testing.T) {
	p := &ICSParser{}
	_, err := p.Parse(context.Background(), nil, nil)
	require.Error(t, err)
}

func TestICSParser_SupportedTypes(t *testing.T) {
	p := &ICSParser{}
	assert.Equal(t, []string{FileTypeICS}, p.SupportedTypes())
	assert.Equal(t, FileTypeICS, p.Provider())
}
