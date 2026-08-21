package tui

// truncate shortens s to at most n characters, appending "..." if it
// was truncated. Used for one-line summaries in pickers and lists.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 3 {
		return s[:n]
	}
	return s[:n-3] + "..."
}
