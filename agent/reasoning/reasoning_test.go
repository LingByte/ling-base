package reasoning

import "testing"

func TestNormalize(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"off", ""},
		{"none", ""},
		{"disabled", ""},
		{"min", "minimum"},
		{"minimum", "minimum"},
		{"minimal", "minimum"},
		{"low", "low"},
		{"med", "medium"},
		{"medium", "medium"},
		{"hi", "high"},
		{"high", "high"},
		{"xhigh", "xhigh"},
		{"maximum", "xhigh"},
		{"max", "max"},
		{"HIGH", "high"},
	}
	for _, c := range cases {
		if got := Normalize(c.in); got != c.want {
			t.Errorf("Normalize(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestBudget(t *testing.T) {
	cases := []struct {
		level string
		want  int
	}{
		{"", 0},
		{"minimum", 1024},
		{"low", 2048},
		{"medium", 8192},
		{"high", 16384},
		{"xhigh", 32768},
		{"max", 32768},
	}
	for _, c := range cases {
		if got := Budget(c.level); got != c.want {
			t.Errorf("Budget(%q) = %d, want %d", c.level, got, c.want)
		}
	}
}

func TestAvailableLevelsNonReasoning(t *testing.T) {
	m := Model{Reasoning: false}
	levels := AvailableLevels(m)
	if len(levels) != 1 || levels[0] != "" {
		t.Fatalf("non-reasoning model should only have off, got %v", levels)
	}
}

func TestAvailableLevelsAnthropic(t *testing.T) {
	m := Model{Reasoning: true, Provider: "anthropic", API: "anthropic"}
	levels := AvailableLevels(m)
	// Should include off + minimum through xhigh
	if len(levels) < 5 {
		t.Fatalf("anthropic reasoning model should have multiple levels, got %v", levels)
	}
}

func TestClamp(t *testing.T) {
	m := Model{Reasoning: true, Provider: "anthropic", API: "anthropic"}
	// Clamp "max" to nearest available (xhigh for anthropic)
	got := Clamp(m, "max")
	if got != "xhigh" {
		t.Errorf("Clamp(max) for anthropic = %q, want xhigh", got)
	}
}

func TestOpenAIEffort(t *testing.T) {
	cases := []struct {
		level, want string
	}{
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"xhigh", "high"},
		{"max", "high"},
		{"", ""},
	}
	for _, c := range cases {
		if got := OpenAIEffort(c.level); got != c.want {
			t.Errorf("OpenAIEffort(%q) = %q, want %q", c.level, got, c.want)
		}
	}
}

func TestAnthropicAdaptiveEffort(t *testing.T) {
	cases := []struct {
		level, want string
	}{
		{"low", "low"},
		{"medium", "medium"},
		{"high", "high"},
		{"xhigh", "xhigh"},
		{"max", "max"},
		{"", ""},
	}
	for _, c := range cases {
		if got := AnthropicAdaptiveEffort(c.level); got != c.want {
			t.Errorf("AnthropicAdaptiveEffort(%q) = %q, want %q", c.level, got, c.want)
		}
	}
}
