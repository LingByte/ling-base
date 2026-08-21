package tui

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// tryPasteClipboardImage reads an image from the clipboard, saves it
// to a temp file under the session's working directory, and returns
// the relative path. Returns ok=false if the clipboard has no image.
// The file is placed in .ling-agent/clipboard/ so it's easy to clean
// up and doesn't pollute the repo root.
func (m *Model) tryPasteClipboardImage() (string, bool) {
	data, ok, err := ReadClipboardImagePNG()
	if err != nil || !ok {
		return "", false
	}

	cwd := "."
	if m.sess != nil && m.sess.CWD != "" {
		cwd = m.sess.CWD
	}
	clipDir := filepath.Join(cwd, ".ling-agent", "clipboard")
	if err := os.MkdirAll(clipDir, 0o755); err != nil {
		return "", false
	}

	filename := fmt.Sprintf("paste-%s.png", time.Now().Format("20060102-150405"))
	path := filepath.Join(clipDir, filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", false
	}

	// Return a path relative to cwd.
	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		rel = path
	}

	// Also stash the base64 for potential multimodal use later.
	_ = base64.StdEncoding.EncodeToString(data) // reserved for image block support

	return rel, true
}
