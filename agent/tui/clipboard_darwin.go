//go:build darwin

package tui

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/tiff"
)

// ReadClipboardImagePNG reads an image from the system clipboard and
// returns it as PNG bytes. Returns ok=false if the clipboard has no
// image. On macOS, uses AppleScript to extract PNG/TIFF data.
func ReadClipboardImagePNG() ([]byte, bool, error) {
	dir := clipboardImageDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, false, err
	}
	path := filepath.Join(dir, "clipboard-"+time.Now().Format("20060102-150405")+"-"+randomHex(4)+".png")
	rawPath := path + ".raw"
	defer os.Remove(rawPath)

	kind, err := writeClipboardImageData(rawPath)
	if err != nil {
		if strings.Contains(err.Error(), "NO_IMAGE") {
			return nil, false, nil
		}
		return nil, false, err
	}
	defer os.Remove(path)

	if kind == "tiff" {
		// Convert TIFF to PNG.
		f, err := os.Open(rawPath)
		if err != nil {
			return nil, false, err
		}
		defer f.Close()
		img, err := tiff.Decode(f)
		if err != nil {
			return nil, false, fmt.Errorf("decode tiff: %w", err)
		}
		var buf []byte
		pngBuf := new(bytes.Buffer)
		if err := png.Encode(pngBuf, img); err != nil {
			return nil, false, fmt.Errorf("encode png: %w", err)
		}
		buf = pngBuf.Bytes()
		return buf, true, nil
	}

	// Already PNG.
	data, err := os.ReadFile(rawPath)
	if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func clipboardImageDir() string {
	return filepath.Join(os.TempDir(), "ling-agent-clipboard")
}

func writeClipboardImageData(path string) (string, error) {
	cmd := exec.Command("osascript", "-e", readClipboardImageScript, path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", string(out), err)
	}
	kind := strings.TrimSpace(string(out))
	if kind == "NO_IMAGE" {
		return "", fmt.Errorf("NO_IMAGE")
	}
	return kind, nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

const readClipboardImageScript = `
on run argv
	set outPath to item 1 of argv
	try
		set imgData to the clipboard as «class PNGf»
		set imgKind to "png"
	on error
		try
			set imgData to the clipboard as «class TIFF»
			set imgKind to "tiff"
		on error
			return "NO_IMAGE"
		end try
	end try

	set outFile to POSIX file outPath
	set fileRef to open for access outFile with write permission
	try
		set eof of fileRef to 0
		write imgData to fileRef
		close access fileRef
	on error errMsg number errNum
		try
			close access fileRef
		end try
		error errMsg number errNum
	end try
	return imgKind
end run
`
