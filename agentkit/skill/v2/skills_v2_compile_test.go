package v2

import (
	"testing"

	"github.com/LingByte/ling-base/agentkit/goagent"
)

func TestSkillSystemV2Compiles(t *testing.T) {
	skill := SkillDefinition{Skill: goagent.Skill{Name: "compile", Instructions: "instructions"}, Version: "2", Enabled: true}
	registry := NewSkillRegistry()
	if err := registry.Register(skill); err != nil {
		t.Fatal(err)
	}
	if got, ok := registry.Get("compile"); !ok || got.Version != "2" {
		t.Fatalf("skill not registered: %+v", got)
	}
}
