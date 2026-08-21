package jsrt

import (
	"testing"

	"github.com/LingByte/ling-base/agentkit/flowcraft/core/agent"
	"github.com/LingByte/ling-base/agentkit/flowcraft/core/agent/agenttest"
)

// TestRuntime_ScriptRuntimeContract runs the shared agent.ScriptRuntime
// conformance suite against the JS runtime.
func TestRuntime_ScriptRuntimeContract(t *testing.T) {
	agenttest.ScriptRuntimeSuite(
		t,
		func() agent.ScriptRuntime { return New() },
		agenttest.ScriptFixture{Source: "1 + 1;"},
	)
}
