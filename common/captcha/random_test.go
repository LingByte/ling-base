package captcha

import "testing"

func TestRandomType_InRange(t *testing.T) {
	seen := map[Type]bool{}
	for i := 0; i < 500; i++ {
		got := RandomType()
		switch got {
		case TypeImage, TypeMath:
			seen[got] = true
		default:
			t.Fatalf("unexpected type %q", got)
		}
	}
	if len(seen) != 2 {
		t.Fatalf("expected both login types over 500 draws, got %d: %v", len(seen), seen)
	}
}

func TestGenerateRandom(t *testing.T) {
	m := NewManager(DefaultConfig())
	for i := 0; i < 50; i++ {
		result, err := m.GenerateRandom()
		if err != nil {
			t.Fatalf("GenerateRandom failed: %v", err)
		}
		switch result.Type {
		case TypeImage, TypeMath:
		default:
			t.Fatalf("unexpected login random type %q", result.Type)
		}
	}
}
