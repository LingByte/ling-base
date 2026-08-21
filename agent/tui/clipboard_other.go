//go:build !darwin

package tui

// ReadClipboardImagePNG reads an image from the system clipboard.
// On non-darwin platforms, returns ok=false (not implemented).
func ReadClipboardImagePNG() ([]byte, bool, error) {
	return nil, false, nil
}
