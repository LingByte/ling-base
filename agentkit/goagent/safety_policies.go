package goagent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/LingByte/ling-base/agentkit/model/gomodel"
)

// RegexBlocklistPolicy enforces that a configurable list of regular
// expressions are not matched within the LLM output.
type RegexBlocklistPolicy struct {
	patterns []*regexp.Regexp
}

// NewRegexBlocklistPolicy creates a new policy with the given string regex patterns.
// It returns an error if any of the patterns fail to compile.
func NewRegexBlocklistPolicy(patterns []string) (*RegexBlocklistPolicy, error) {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("failed to compile regex %q: %w", p, err)
		}
		compiled = append(compiled, r)
	}
	return &RegexBlocklistPolicy{patterns: compiled}, nil
}

// Validate checks the response against all configured regex patterns.
func (p *RegexBlocklistPolicy) Validate(ctx context.Context, response string) error {
	for _, r := range p.patterns {
		if r.MatchString(response) {
			return fmt.Errorf("safety policy violation: output matches blocked pattern %q", r.String())
		}
	}
	return nil
}

// LLMEvaluatorPolicy uses a secondary language model to evaluate the safety
// of the proposed response.
type LLMEvaluatorPolicy struct {
	model  gomodel.Agent
	prompt string
}

const defaultEvaluatorPrompt = `Evaluate the following text for safety.
If the text contains hate speech, dangerous instructions, PII, or violates general safety guidelines, respond with exactly "UNSAFE".
Otherwise, respond with exactly "SAFE".

TEXT TO EVALUATE:
<text>
%s
</text>`

// NewLLMEvaluatorPolicy creates a new safety policy that uses an LLM to evaluate responses.
// If promptTemplate is empty, a default evaluation prompt is used.
func NewLLMEvaluatorPolicy(model gomodel.Agent, promptTemplate string) *LLMEvaluatorPolicy {
	if promptTemplate == "" {
		promptTemplate = defaultEvaluatorPrompt
	}
	return &LLMEvaluatorPolicy{
		model:  model,
		prompt: promptTemplate,
	}
}

// Validate sends the response to the evaluating LLM and checks its verdict.
func (p *LLMEvaluatorPolicy) Validate(ctx context.Context, response string) error {
	// Sanitize output to prevent prompt injection breaking out of the <text> block
	safeResponse := strings.ReplaceAll(response, "<text>", "(text)")
	safeResponse = strings.ReplaceAll(safeResponse, "</text>", "(/text)")

	evalPrompt := fmt.Sprintf(p.prompt, safeResponse)

	result, err := p.model.Generate(ctx, evalPrompt)
	if err != nil {
		return fmt.Errorf("safety evaluation failed: %w", err)
	}

	verdict := strings.ToUpper(strings.TrimSpace(fmt.Sprintf("%v", result)))

	if strings.Contains(verdict, "UNSAFE") {
		return fmt.Errorf("safety policy violation: output flagged as unsafe by LLM evaluator")
	}

	return nil
}
