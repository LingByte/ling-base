package subagents

import (
	"context"
	"strings"
	"testing"

	compat "github.com/LingByte/ling-base/relay/compat"
)

type fakeModel struct {
	response string
	err      error
	prompts  []string
}

func (f *fakeModel) Info() compat.Info { return compat.Info{Name: "fake"} }

func (f *fakeModel) GenerateContent(ctx context.Context, req *compat.Request) (<-chan *compat.Response, error) {
	prompt := ""
	for _, msg := range req.Messages {
		if msg.Role == compat.RoleUser {
			prompt = msg.Content
			break
		}
	}
	f.prompts = append(f.prompts, prompt)
	if f.err != nil {
		return nil, f.err
	}
	text := f.response
	if text == "" {
		text = "ok"
	}
	ch := make(chan *compat.Response, 1)
	finishReason := "stop"
	ch <- &compat.Response{
		Done:    true,
		Choices: []compat.Choice{{Message: compat.NewAssistantMessage(text), FinishReason: &finishReason}},
	}
	close(ch)
	return ch, nil
}

func TestResearcherRunIncludesPersonaAndTask(t *testing.T) {
	fm := &fakeModel{response: "result"}
	researcher := NewResearcher(fm)

	out, err := researcher.Run(context.Background(), "Investigate topic")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if out != "result" {
		t.Fatalf("expected result, got %q", out)
	}
	if len(fm.prompts) == 0 {
		t.Fatal("expected prompt to be captured")
	}
	if !strings.Contains(fm.prompts[0], "diligent research assistant") {
		t.Fatalf("expected persona in prompt, got %q", fm.prompts[0])
	}
	if !strings.Contains(fm.prompts[0], "Investigate topic") {
		t.Fatalf("expected task in prompt, got %q", fm.prompts[0])
	}
}
