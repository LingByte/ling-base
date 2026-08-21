package goprovider

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/LingByte/ling-base/agentkit/model/gomodel"
)

// NewLLMProvider returns a concrete Agent for the given provider name.
func NewLLMProvider(ctx context.Context, provider string, model string, promptPrefix string) (gomodel.Agent, error) {
	var agent gomodel.Agent
	var err error

	switch provider {
	case "openai":
		agent = gomodel.NewOpenAILLM(model, promptPrefix)
	case "gemini", "google":
		agent, err = gomodel.NewGeminiLLM(ctx, model, promptPrefix)
	case "vertex", "vertexai", "vertex-ai":
		project := strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT"))
		location := strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_LOCATION"))
		if location == "" {
			location = strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_REGION"))
		}
		agent, err = gomodel.NewVertexLLM(ctx, model, promptPrefix, project, location)
	case "ollama":
		agent, err = gomodel.NewOllamaLLM(model, promptPrefix)
	case "anthropic", "claude":
		agent = gomodel.NewAnthropicLLM(model, promptPrefix)
	case "openrouter":
		agent = gomodel.NewOpenRouterLLM(model, promptPrefix)
	default:
		return nil, fmt.Errorf("unknown provider: %s", provider)
	}

	if err != nil {
		return nil, err
	}

	return gomodel.TryCreateCachedLLM(agent), nil
}
