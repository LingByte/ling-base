package session

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SessionInfo describes one session transcript on disk.
type SessionInfo struct {
	ID       string    // session ID (filename without .jsonl)
	Path     string    // full path to the transcript file
	Modified time.Time // last modification time
	Size     int64     // file size in bytes
	Summary  string    // first-line summary, if available
}

// ListSessions returns all sessions for the given working directory,
// sorted by modification time (newest first). Includes both the
// current sessions root and the legacy projects root.
func ListSessions(cwd string) []SessionInfo {
	dir := Dir(cwd)
	matches, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	legacyMatches, _ := filepath.Glob(filepath.Join(legacyDir(cwd), "*.jsonl"))
	matches = append(matches, legacyMatches...)

	var out []SessionInfo
	for _, m := range matches {
		st, err := os.Stat(m)
		if err != nil || st.Size() == 0 {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(m), ".jsonl")
		info := SessionInfo{
			ID:       id,
			Path:     m,
			Modified: st.ModTime(),
			Size:     st.Size(),
		}
		// Try to read a summary.
		if summary, ok := ReadSummary(cwd, id); ok {
			info.Summary = summary
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out
}
