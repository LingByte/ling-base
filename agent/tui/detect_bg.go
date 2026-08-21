package tui

import (
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/term"
)

// detectedThemeID holds the result of DetectThemeFromBackground. It is
// read by currentThemeID / chromePaletteFor / glamourThemeOption when
// the user has selected the "auto" theme. Default is "dark" until
// detection runs.
var detectedThemeID = "dark"

// DetectThemeFromBackground queries the controlling tty for its
// current background colour using the OSC 11 escape sequence and
// returns "dark" or "light" based on the response's perceived
// luminance. Falls back to "dark" when the terminal does not reply
// within timeout, which is the expected behaviour for terminals that
// do not implement OSC 11 (Linux console, VS Code's integrated
// terminal in some configurations, tmux without pass-through, very
// old emulators).
//
// The query / parse runs synchronously before the TUI is initialised
// so the returned theme can drive the entire session. We briefly put
// stdin into raw mode and disable echo so the OSC reply doesn't leak
// onto the user's screen as visible bytes.
func DetectThemeFromBackground(timeout time.Duration) string {
	// Honour explicit override env var first.
	switch strings.ToLower(strings.TrimSpace(os.Getenv("LING_AGENT_THEME"))) {
	case "dark":
		return "dark"
	case "light":
		return "light"
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) || !term.IsTerminal(int(os.Stdout.Fd())) {
		return "dark"
	}

	fd := int(os.Stdin.Fd())
	st, err := term.MakeRaw(fd)
	if err != nil {
		return "dark"
	}
	defer term.Restore(fd, st)

	// Send the OSC 11 query. BEL (\x07) is the most widely supported
	// terminator.
	if _, err := os.Stdout.Write([]byte("\x1b]11;?\x07")); err != nil {
		return "dark"
	}

	deadline := time.Now().Add(timeout)
	resp := readOSCResponse(deadline)
	if resp == "" {
		return "dark"
	}

	r, g, b, ok := parseOSC11Reply(resp)
	if !ok {
		return "dark"
	}

	// Rec. 709 luma. Threshold at 0.5: anything brighter than mid-grey
	// gets the light theme.
	luma := 0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(b)
	if luma >= 0.5 {
		return "light"
	}
	return "dark"
}

// InitDetectedTheme runs DetectThemeFromBackground and stores the
// result in detectedThemeID. Called once at TUI startup.
func InitDetectedTheme() {
	detectedThemeID = DetectThemeFromBackground(200 * time.Millisecond)
}

// readOSCResponse drains stdin into a small buffer until either a
// terminator (BEL or ST) is seen, the deadline expires, or stdin hits
// EOF.
func readOSCResponse(deadline time.Time) string {
	var buf [128]byte
	n := 0
	for time.Now().Before(deadline) {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		b, ok, err := peekStdin(os.Stdin, remaining)
		if err != nil || !ok {
			return string(buf[:n])
		}
		if n < len(buf) {
			buf[n] = b
			n++
		}
		// BEL terminator
		if b == 0x07 {
			return string(buf[:n])
		}
		// ST terminator: ESC then '\\'.
		if n >= 2 && buf[n-2] == 0x1b && buf[n-1] == '\\' {
			return string(buf[:n])
		}
	}
	return string(buf[:n])
}

// peekStdin reads a single byte from r with a deadline. Returns the
// byte, whether a byte was read, and any error.
func peekStdin(r *os.File, timeout time.Duration) (byte, bool, error) {
	// Set a read deadline via SetReadDeadline on the underlying fd.
	// os.File doesn't expose this directly, but on Unix we can use
	// syscall. For simplicity and portability, we use a tiny channel+
	// goroutine with a timer.
	type result struct {
		b   byte
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var buf [1]byte
		_, err := r.Read(buf[:])
		ch <- result{buf[0], err}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			return 0, false, res.err
		}
		return res.b, true, nil
	case <-time.After(timeout):
		return 0, false, nil
	}
}

// parseOSC11Reply extracts the (r, g, b) colour components from an
// OSC 11 reply of the form "\x1b]11;rgb:RRRR/GGGG/BBBB\x07" (or with
// ST terminator). The component widths can be 1, 2, 3, or 4 hex
// digits per channel; we normalise them into the 0..1 range.
func parseOSC11Reply(s string) (float64, float64, float64, bool) {
	i := strings.Index(s, "rgb:")
	if i < 0 {
		return 0, 0, 0, false
	}
	body := s[i+len("rgb:"):]
	body = strings.TrimRight(body, "\x07")
	if strings.HasSuffix(body, "\x1b\\") {
		body = strings.TrimSuffix(body, "\x1b\\")
	}
	parts := strings.Split(body, "/")
	if len(parts) != 3 {
		return 0, 0, 0, false
	}
	parse := func(hexstr string) (float64, bool) {
		if len(hexstr) == 0 || len(hexstr) > 4 {
			return 0, false
		}
		v, err := strconv.ParseUint(hexstr, 16, 32)
		if err != nil {
			return 0, false
		}
		max := uint64(1)
		for j := 0; j < len(hexstr); j++ {
			max *= 16
		}
		max--
		if max == 0 {
			return 0, false
		}
		return float64(v) / float64(max), true
	}
	r, ok1 := parse(parts[0])
	g, ok2 := parse(parts[1])
	b, ok3 := parse(parts[2])
	if !ok1 || !ok2 || !ok3 {
		return 0, 0, 0, false
	}
	return r, g, b, true
}
