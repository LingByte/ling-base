package subagents

import (
	"context"
	"fmt"
	"strings"

	"github.com/LingByte/ling-base/agentkit/goagent"
	compat "github.com/LingByte/ling-base/relay/compat"
)

// Researcher is a lightweight sub-agent that focuses on synthesizing background information.
type Researcher struct {
	model   compat.Model
	persona string
}

func NewResearcher(model compat.Model) *Researcher {
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

	req := compat.NewRequest([]compat.Message{compat.NewUserMessage(prompt.String())})
	respCh, err := r.model.GenerateContent(ctx, req)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for resp := range respCh {
		if resp.Error != nil {
			return "", resp.Error
		}
		for _, choice := range resp.Choices {
			if choice.Message.Content != "" {
				sb.WriteString(choice.Message.Content)
			}
		}
	}
	return sb.String(), nil
}

var _ goagent.SubAgent = (*Researcher)(nil)
