package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/LingByte/ling-base/agent/agent"
	"github.com/LingByte/ling-base/agent/session"
)

// seedSession writes a transcript with one real message so MostRecent treats it
// as a resumable conversation (an empty file is now skipped as contentless).
func seedSession(t *testing.T, cwd, id string) {
	t.Helper()
	w, err := session.NewWriter(cwd, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Append(session.Entry{Type: "user", Message: []byte("{}")}); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestResolveResumeIDAutoResumesMostRecentSessionForCWD(t *testing.T) {
	t.Setenv("LING_AGENT_CONFIG_DIR", t.TempDir())
	cwd := "/work/proj"
	otherCWD := "/work/other"

	seedSession(t, otherCWD, "other-session")
	seedSession(t, cwd, "project-session")

	got, err := resolveResumeID(cwd, Options{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "project-session" {
		t.Fatalf("resumeID = %q, want project-session", got)
	}
}

func TestResolveResumeIDStartsNewWhenNoSessionExists(t *testing.T) {
	t.Setenv("LING_AGENT_CONFIG_DIR", t.TempDir())

	got, err := resolveResumeID("/work/proj", Options{}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("resumeID = %q, want empty", got)
	}
}

func TestResolveResumeIDNewSessionSkipsAutoResume(t *testing.T) {
	t.Setenv("LING_AGENT_CONFIG_DIR", t.TempDir())
	cwd := "/work/proj"
	seedSession(t, cwd, "project-session")

	got, err := resolveResumeID(cwd, Options{NewSession: true}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("resumeID = %q, want empty", got)
	}
}

func TestResolveResumeIDExplicitResumeWinsOverAutoResume(t *testing.T) {
	t.Setenv("LING_AGENT_CONFIG_DIR", t.TempDir())
	cwd := "/work/proj"
	seedSession(t, cwd, "project-session")

	got, err := resolveResumeID(cwd, Options{Resume: "explicit-session"}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got != "explicit-session" {
		t.Fatalf("resumeID = %q, want explicit-session", got)
	}
}

func TestResolveResumeIDNewSessionConflictsWithExplicitResume(t *testing.T) {
	_, err := resolveResumeID("/work/proj", Options{NewSession: true, Resume: "old"}, true)
	if err == nil || !strings.Contains(err.Error(), "--new-session cannot be combined") {
		t.Fatalf("err = %v, want --new-session conflict", err)
	}
}

func TestResolveResumeIDHeadlessDoesNotAutoResume(t *testing.T) {
	t.Setenv("LING_AGENT_CONFIG_DIR", t.TempDir())
	cwd := "/work/proj"
	seedSession(t, cwd, "project-session")

	// Headless (interactive=false): a prior session is NOT auto-resumed.
	if got, err := resolveResumeID(cwd, Options{}, false); err != nil || got != "" {
		t.Fatalf("headless auto-resume = (%q, %v), want empty", got, err)
	}
	// …but an explicit --continue still resumes it, even headless.
	if got, err := resolveResumeID(cwd, Options{ContinueSession: true}, false); err != nil || got != "project-session" {
		t.Fatalf("headless --continue = (%q, %v), want project-session", got, err)
	}
}
func TestCompactAndPersistWritesSummaryOnSuccess(t *testing.T) {
	called := false
	_, summary, err := compactAndPersist(context.Background(), nil, func(context.Context, []agent.Message) ([]agent.Message, string, error) {
		return []agent.Message{agent.NewUserMessage("summary")}, "summary", nil
	}, func(got string) {
		called = true
		if got != "summary" {
			t.Errorf("summary = %q, want summary", got)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary != "summary" {
		t.Errorf("summary = %q, want summary", summary)
	}
	if !called {
		t.Fatal("onSummary was not called")
	}
}

func TestCompactAndPersistSkipsSummaryOnError(t *testing.T) {
	boom := errors.New("boom")
	_, _, err := compactAndPersist(context.Background(), nil, func(context.Context, []agent.Message) ([]agent.Message, string, error) {
		return nil, "summary", boom
	}, func(string) {
		t.Fatal("onSummary should not be called")
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}
