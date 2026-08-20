package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSupportedDocumentFormats(t *testing.T) {
	formats := SupportedDocumentFormats()
	require.NotEmpty(t, formats)

	// Verify each format has an extension and description
	for _, f := range formats {
		assert.NotEmpty(t, f.Extension, "extension should not be empty")
		assert.NotEmpty(t, f.Description, "description should not be empty for %s", f.Extension)
	}

	// Verify some expected formats are present
	extensions := make(map[string]bool)
	for _, f := range formats {
		extensions[f.Extension] = true
	}

	expected := []string{".txt", ".md", ".csv", ".html", ".json", ".yaml", ".yml", ".pdf", ".docx", ".pptx", ".xlsx", ".png", ".jpg", ".mp3", ".wav"}
	for _, ext := range expected {
		assert.True(t, extensions[ext], "expected %s to be in supported formats", ext)
	}
}

func TestSupportedDocumentNotes(t *testing.T) {
	notes := SupportedDocumentNotes()
	require.NotEmpty(t, notes)

	// Verify each note is non-empty
	for _, note := range notes {
		assert.NotEmpty(t, note)
	}

	// Should mention OCR and ASR
	allNotes := ""
	for _, n := range notes {
		allNotes += n + " "
	}
	assert.Contains(t, allNotes, "OCR")
	assert.Contains(t, allNotes, "ASR")
}
