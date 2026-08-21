package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LingByte/ling-base/agent/permission"
	"github.com/LingByte/ling-base/agent/schema"
	"github.com/LingByte/ling-base/agent/swarm"
)

// SwarmSpawnInput is the SwarmSpawn tool's input.
type SwarmSpawnInput struct {
	Task  string `json:"task" jsonschema:"description=The task for the background agent to work on"`
	Model string `json:"model,omitempty" jsonschema:"description=Optional model override for the spawned agent"`
}

// SwarmSpawn spawns a background headless agent via the swarm supervisor.
type SwarmSpawn struct {
	schema *schema.Schema
	swarm  *swarm.Swarm
}

// NewSwarmSpawn constructs the SwarmSpawn tool backed by a swarm supervisor.
func NewSwarmSpawn(s *swarm.Swarm) (*SwarmSpawn, error) {
	sch, err := schema.For[SwarmSpawnInput]()
	if err != nil {
		return nil, fmt.Errorf("swarmspawn: build schema: %w", err)
	}
	return &SwarmSpawn{schema: sch, swarm: s}, nil
}

func (t *SwarmSpawn) Name() string { return "SwarmSpawn" }

func (t *SwarmSpawn) Description(context.Context) (string, error) {
	return "Spawn a background agent to work on a task in parallel. The agent runs headless in the same working directory. Returns the agent ID and initial status.", nil
}

func (t *SwarmSpawn) InputSchema() json.RawMessage { return t.schema.Raw }

func (t *SwarmSpawn) ValidateInput(raw json.RawMessage) error { return t.schema.Validate(raw) }

func (t *SwarmSpawn) PermissionRequest(json.RawMessage) permission.PermissionRequest {
	return permission.PermissionRequest{}
}

func (t *SwarmSpawn) CheckPermissions(pctx permission.Context, _ permission.PermissionRequest) permission.Decision {
	return allowAlways(pctx)
}

func (t *SwarmSpawn) Execute(ctx context.Context, tctx Context, raw json.RawMessage) ([]Result, error) {
	var in SwarmSpawnInput
	if err := json.Unmarshal(raw, &in); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Task) == "" {
		return []Result{{Content: "Error: task is required", IsError: true}}, nil
	}
	agent, err := t.swarm.Spawn(ctx, swarm.SpawnRequest{Task: in.Task, Model: in.Model})
	if err != nil {
		return []Result{{Content: fmt.Sprintf("Error: %v", err), IsError: true}}, nil
	}
	snap := agent.Snapshot()
	return []Result{{Content: fmt.Sprintf("Spawned agent %s: %s\nStatus: %s", snap.ID, snap.Task, snap.Status)}}, nil
}

// SwarmListInput is the SwarmList tool's input (no parameters).
type SwarmListInput struct{}

// SwarmList lists all swarm agents and their statuses.
type SwarmList struct {
	schema *schema.Schema
	swarm  *swarm.Swarm
}

// NewSwarmList constructs the SwarmList tool.
func NewSwarmList(s *swarm.Swarm) (*SwarmList, error) {
	sch, err := schema.For[SwarmListInput]()
	if err != nil {
		return nil, fmt.Errorf("swarmlist: build schema: %w", err)
	}
	return &SwarmList{schema: sch, swarm: s}, nil
}

func (t *SwarmList) Name() string { return "SwarmList" }

func (t *SwarmList) Description(context.Context) (string, error) {
	return "List all background swarm agents and their current statuses.", nil
}

func (t *SwarmList) InputSchema() json.RawMessage { return t.schema.Raw }

func (t *SwarmList) ValidateInput(raw json.RawMessage) error { return t.schema.Validate(raw) }

func (t *SwarmList) PermissionRequest(json.RawMessage) permission.PermissionRequest {
	return permission.PermissionRequest{}
}

func (t *SwarmList) CheckPermissions(pctx permission.Context, _ permission.PermissionRequest) permission.Decision {
	return allowAlways(pctx)
}

func (t *SwarmList) Execute(_ context.Context, _ Context, _ json.RawMessage) ([]Result, error) {
	snaps := t.swarm.SnapshotAll()
	if len(snaps) == 0 {
		return []Result{{Content: "No swarm agents running"}}, nil
	}
	var b strings.Builder
	for _, s := range snaps {
		b.WriteString(swarm.FormatStatus(s))
		b.WriteString("\n")
	}
	return []Result{{Content: strings.TrimSpace(b.String())}}, nil
}
