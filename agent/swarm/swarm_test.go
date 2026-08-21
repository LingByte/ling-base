package swarm

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

// fakeRunner is a test Runner that records activity and can be controlled.
type fakeRunner struct {
	mu       sync.Mutex
	started  bool
	finished bool
	fail     error
	delay    time.Duration
}

func (f *fakeRunner) Run(ctx context.Context, sink Sink) error {
	f.mu.Lock()
	f.started = true
	f.mu.Unlock()

	sink.Activity("working")
	time.Sleep(f.delay)

	if f.fail != nil {
		sink.Activity("failed")
		return f.fail
	}

	sink.Activity("done")
	sink.Transcript("task completed successfully")
	f.mu.Lock()
	f.finished = true
	f.mu.Unlock()
	return nil
}

func TestSpawnAndComplete(t *testing.T) {
	s := New(Config{
		Root:     t.TempDir(),
		RepoRoot: t.TempDir(),
		NewRunner: func(a *Agent) Runner {
			return &fakeRunner{delay: 10 * time.Millisecond}
		},
	})

	a, err := s.Spawn(context.Background(), SpawnRequest{Task: "test task"})
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == "" {
		t.Fatal("agent should have an ID")
	}

	// Wait for completion.
	time.Sleep(50 * time.Millisecond)

	snap := a.Snapshot()
	if snap.Status != "done" {
		t.Errorf("status = %s, want done", snap.Status)
	}
	if snap.Transcript == "" {
		t.Error("transcript should not be empty after completion")
	}
}

func TestSpawnFailure(t *testing.T) {
	s := New(Config{
		Root:     t.TempDir(),
		RepoRoot: t.TempDir(),
		NewRunner: func(a *Agent) Runner {
			return &fakeRunner{fail: fmt.Errorf("boom"), delay: 10 * time.Millisecond}
		},
	})

	a, err := s.Spawn(context.Background(), SpawnRequest{Task: "failing task"})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	snap := a.Snapshot()
	if snap.Status != "failed" {
		t.Errorf("status = %s, want failed", snap.Status)
	}
}

func TestStop(t *testing.T) {
	s := New(Config{
		Root:     t.TempDir(),
		RepoRoot: t.TempDir(),
		NewRunner: func(a *Agent) Runner {
			return &fakeRunner{delay: 5 * time.Second}
		},
	})

	a, err := s.Spawn(context.Background(), SpawnRequest{Task: "long task"})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(10 * time.Millisecond) // let it start
	a.Stop()

	snap := a.Snapshot()
	if snap.Status != "killed" {
		t.Errorf("status = %s, want killed", snap.Status)
	}
}

func TestSnapshotAll(t *testing.T) {
	s := New(Config{
		Root:     t.TempDir(),
		RepoRoot: t.TempDir(),
		NewRunner: func(a *Agent) Runner {
			return &fakeRunner{delay: 100 * time.Millisecond}
		},
	})

	s.Spawn(context.Background(), SpawnRequest{Task: "task 1"})
	s.Spawn(context.Background(), SpawnRequest{Task: "task 2"})

	snaps := s.SnapshotAll()
	if len(snaps) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(snaps))
	}
}

func TestGet(t *testing.T) {
	s := New(Config{
		Root:     t.TempDir(),
		RepoRoot: t.TempDir(),
		NewRunner: func(a *Agent) Runner {
			return &fakeRunner{delay: 100 * time.Millisecond}
		},
	})

	a, _ := s.Spawn(context.Background(), SpawnRequest{Task: "find me"})
	got := s.Get(a.ID)
	if got == nil || got.ID != a.ID {
		t.Error("Get should find the agent by ID")
	}

	if s.Get("nonexistent") != nil {
		t.Error("Get should return nil for missing ID")
	}
}

func TestFormatStatus(t *testing.T) {
	snap := AgentSnapshot{
		ID:     "agent-123",
		Task:   "fix the bug",
		Status: "running",
	}
	got := FormatStatus(snap)
	if got == "" {
		t.Error("FormatStatus should return non-empty string")
	}
}
