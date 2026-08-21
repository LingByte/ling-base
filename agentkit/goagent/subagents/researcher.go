package subagents

import (
	"context"
	"fmt"
	"strings"

	"github.com/LingByte/ling-base/agentkit/goagent"
	"github.com/LingByte/ling-base/agentkit/model/gomodel"
)

// Researcher is a lightweight sub-agent that focuses on synthesizing background information.
type Researcher struct {
	model   gomodel.Agent
	persona string
}

func NewResearcher(model gomodel.Agent) *Researcher {
	return &Researcher{
		model:   model,
		persona: "You are a diligent research assistant. Provide structured findings and cite sources when available.",
	}
}

func (r *Researcher) Name() string { return "researcher" }
func (r *Researcher) Description() string {
	return "Synthesizes background information and drafts research summaries."
}

func (r *Researcher) Run(ctx context.Context, input string) (string, error) {
	if r.model == nil {
		return "", fmt.Errorf("researcher subagent missing model")
	}

	prompt := strings.Builder{}
	prompt.WriteString(r.persona)
	prompt.WriteString("\n\nTask:\n")
	prompt.WriteString(strings.TrimSpace(input))
	prompt.WriteString("\n\nDeliverable: Provide a concise research brief with bullet points and next steps.\n")

	resp, err := r.model.Generate(ctx, prompt.String())
	if err != nil {
		return "", err
	}
	return fmt.Sprint(resp), nil
}

var _ goagent.SubAgent = (*Researcher)(nil)
