// Package swarm implements LingAgent's multi-agent supervisor.
//
// A Swarm manages a set of headless ling-agent subprocesses ("agents")
// that share the host's working directory. Each agent runs as a
// `ling-agent -p <task> --dangerously-skip-permissions` subprocess,
// reusing ling-agent's own model resolution and tooling without
// re-implementing the agent loop.
//
// Each Agent has:
//   - a unique id (short slug + nanoseconds)
//   - a Runner (the thing that actually executes the task)
//   - a Status string + Activity string that the dashboard reads
//
// The Runner abstraction means tests can swap a fake in instead of
// really spawning a subprocess.
package swarm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Status is the high-level lifecycle state of an Agent.
type Status string

const (
	StatusPending  Status = "pending"  // created, not started yet
	StatusRunning  Status = "running"  // Runner.Run is in flight
	StatusDone     Status = "done"     // Runner.Run returned nil
	StatusFailed   Status = "failed"   // Runner.Run returned an error
	StatusKilled   Status = "killed"   // Stop() called before completion
	StatusDetached Status = "detached" // reloaded from disk; no live runner
)

// Config configures a Swarm.
type Config struct {
	// Root is the directory under which per-agent state files live.
	// Typically ~/.ling-agent/swarm, but tests pass a tempdir.
	Root string

	// RepoRoot is the working directory every spawned agent runs in.
	RepoRoot string

	// NewRunner produces the Runner for an Agent. If nil, the default
	// exec runner is used (spawns `ling-agent -p <task>`).
	NewRunner func(a *Agent) Runner

	// Now is a clock seam for tests; defaults to time.Now.
	Now func() time.Time
}

// Runner executes one agent task. Run blocks until the task finishes,
// is cancelled via ctx, or hits an unrecoverable error.
type Runner interface {
	Run(ctx context.Context, sink Sink) error
}

// Sink is how a Runner reports activity and transcript back to the
// supervisor. All methods are safe to call from any goroutine and
// never block.
type Sink interface {
	Activity(msg string)
	Transcript(chunk string)
}

// Swarm supervises a set of Agents.
type Swarm struct {
	cfg Config

	mu     sync.Mutex
	agents map[string]*Agent
	order  []string // creation order for stable listing
}

// New constructs a Swarm from cfg.
func New(cfg Config) *Swarm {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.NewRunner == nil {
		cfg.NewRunner = func(a *Agent) Runner {
			return &execRunner{agent: a}
		}
	}
	return &Swarm{
		cfg:    cfg,
		agents: map[string]*Agent{},
	}
}

// Agent is one supervised task.
type Agent struct {
	ID        string
	Task      string
	Model     string
	Provider  string
	Dir       string
	Status    Status
	Activity  string
	Transcript string
	SessionID string

	startedAt time.Time
	endedAt   time.Time

	mu     sync.Mutex
	cancel context.CancelFunc

	swarm *Swarm
}

// StartedAt returns when the agent was spawned.
func (a *Agent) StartedAt() time.Time { return a.startedAt }

// EndedAt returns when the agent reached a terminal state.
func (a *Agent) EndedAt() time.Time { return a.endedAt }

// SpawnRequest configures a Spawn.
type SpawnRequest struct {
	Task     string
	Model    string
	Provider string
}

// Spawn creates a new Agent for the given task and starts it on a
// background goroutine.
func (f *Swarm) Spawn(ctx context.Context, req SpawnRequest) (*Agent, error) {
	id := newID()
	stateDir := filepath.Join(f.cfg.Root, "agents", id)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("swarm: create state dir: %w", err)
	}

	a := &Agent{
		ID:        id,
		Task:      req.Task,
		Model:     req.Model,
		Provider:  req.Provider,
		Dir:       f.cfg.RepoRoot,
		Status:    StatusPending,
		startedAt: f.cfg.Now(),
		swarm:     f,
	}

	f.mu.Lock()
	f.agents[id] = a
	f.order = append(f.order, id)
	f.mu.Unlock()

	runner := f.cfg.NewRunner(a)
	runCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()

	a.Status = StatusRunning
	go func() {
		sink := &agentSink{a: a}
		err := runner.Run(runCtx, sink)
		a.mu.Lock()
		a.endedAt = f.cfg.Now()
		if err != nil && a.Status != StatusKilled {
			a.Status = StatusFailed
		} else if a.Status != StatusKilled {
			a.Status = StatusDone
		}
		a.mu.Unlock()
	}()

	return a, nil
}

// Stop cancels a running agent.
func (a *Agent) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.Status = StatusKilled
	}
}

// Snapshot returns a snapshot of the agent's current state.
func (a *Agent) Snapshot() AgentSnapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	return AgentSnapshot{
		ID:        a.ID,
		Task:      a.Task,
		Status:    string(a.Status),
		Activity:  a.Activity,
		Transcript: a.Transcript,
		Model:     a.Model,
	}
}

// AgentSnapshot is a point-in-time copy of an Agent's state.
type AgentSnapshot struct {
	ID        string `json:"id"`
	Task      string `json:"task"`
	Status    string `json:"status"`
	Activity  string `json:"activity"`
	Transcript string `json:"transcript"`
	Model     string `json:"model"`
}

// SnapshotAll returns snapshots of all agents, sorted by creation order.
func (f *Swarm) SnapshotAll() []AgentSnapshot {
	f.mu.Lock()
	ids := make([]string, len(f.order))
	copy(ids, f.order)
	f.mu.Unlock()

	out := make([]AgentSnapshot, 0, len(ids))
	for _, id := range ids {
		f.mu.Lock()
		a := f.agents[id]
		f.mu.Unlock()
		if a != nil {
			out = append(out, a.Snapshot())
		}
	}
	return out
}

// Get returns the agent with the given ID, or nil.
func (f *Swarm) Get(id string) *Agent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.agents[id]
}

// Agents returns all agents sorted by creation order.
func (f *Swarm) Agents() []*Agent {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*Agent, 0, len(f.order))
	for _, id := range f.order {
		out = append(out, f.agents[id])
	}
	return out
}

// agentSink implements Sink for an Agent.
type agentSink struct {
	a *Agent
}

func (s *agentSink) Activity(msg string) {
	s.a.mu.Lock()
	s.a.Activity = msg
	s.a.mu.Unlock()
}

func (s *agentSink) Transcript(chunk string) {
	s.a.mu.Lock()
	s.a.Transcript += chunk
	s.a.mu.Unlock()
}

// newID generates a short unique agent ID.
func newID() string {
	return fmt.Sprintf("agent-%d", time.Now().UnixNano())
}

// ---- exec runner ----

// execRunner spawns `ling-agent -p <task> --dangerously-skip-permissions`
// as a subprocess and captures its stdout as transcript.
type execRunner struct {
	agent *Agent

	// Command overrides the default invocation. Tests set this to a
	// fake binary. Production code leaves it nil.
	Command []string
}

func (r *execRunner) buildArgs() []string {
	a := r.agent
	args := []string{"-p", a.Task, "--dangerously-skip-permissions"}
	if a.Model != "" {
		args = append(args, "--model", a.Model)
	}
	return args
}

// SortSnapshots sorts snapshots by creation order (using ID as a proxy).
func SortSnapshots(snaps []AgentSnapshot) {
	sort.Slice(snaps, func(i, j int) bool {
		return snaps[i].ID < snaps[j].ID
	})
}

// FormatStatus returns a human-readable one-liner for an agent.
func FormatStatus(s AgentSnapshot) string {
	task := s.Task
	if len(task) > 50 {
		task = task[:47] + "..."
	}
	return fmt.Sprintf("[%s] %s — %s", s.ID, strings.ToUpper(s.Status), task)
}
