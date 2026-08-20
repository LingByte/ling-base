package captcha

import (
	"testing"
	"time"
)

func TestClickCaptcha_Generate(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	result, err := cc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.ID == "" || result.Type != TypeClick {
		t.Fatalf("bad result: %+v", result)
	}
	targets, _ := result.Data["targets"].([]string)
	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}
	chars, _ := result.Data["chars"].([]CharMarker)
	if len(chars) != 5 { // 3 targets + 2 decoys
		t.Fatalf("expected 5 chars, got %d", len(chars))
	}
	bg, _ := result.Data["background"].(string)
	if bg == "" {
		t.Fatal("background image missing")
	}
}

func TestClickCaptcha_OrderedVerify_Pass(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	result, _ := cc.Generate()

	targets, _ := result.Data["targets"].([]string)
	chars, _ := result.Data["chars"].([]CharMarker)
	byChar := map[string]Point{}
	for _, c := range chars {
		byChar[c.Char] = Point{X: c.X, Y: c.Y}
	}
	ordered := make([]Point, len(targets))
	for i, target := range targets {
		ordered[i] = byChar[target]
	}

	ok, err := cc.Verify(result.ID, ordered)
	if err != nil || !ok {
		t.Fatalf("ordered verify failed: ok=%v err=%v", ok, err)
	}
}

func TestClickCaptcha_OrderedVerify_WrongOrder(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	result, _ := cc.Generate()

	targets, _ := result.Data["targets"].([]string)
	chars, _ := result.Data["chars"].([]CharMarker)
	byChar := map[string]Point{}
	for _, c := range chars {
		byChar[c.Char] = Point{X: c.X, Y: c.Y}
	}
	reversed := make([]Point, len(targets))
	for i := range targets {
		reversed[i] = byChar[targets[len(targets)-1-i]]
	}

	ok, err := cc.Verify(result.ID, reversed)
	if err != nil || ok {
		t.Fatalf("expected wrong order to fail, ok=%v err=%v", ok, err)
	}
}

func TestClickCaptcha_Verify_WrongLength(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	result, _ := cc.Generate()
	// Submit only 2 points instead of 3.
	ok, err := cc.Verify(result.ID, []Point{{X: 1, Y: 2}, {X: 3, Y: 4}})
	if err != nil || ok {
		t.Fatalf("expected fail for wrong length, got %v, %v", ok, err)
	}
}

func TestClickCaptcha_Verify_NotFound(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	ok, err := cc.Verify("missing", []Point{{X: 1, Y: 2}})
	if err == nil || ok {
		t.Fatalf("expected error, got %v, %v", ok, err)
	}
}

func TestClickCaptcha_DefaultCount(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 0, 20, 5*time.Minute, nil)
	if cc.count != 3 {
		t.Fatalf("expected default count 3, got %d", cc.count)
	}
}

func TestClickCaptcha_compareOrdered_TypeMismatch(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	if cc.compareOrdered("bad", []Point{{X: 1, Y: 2}}) {
		t.Fatal("type mismatch should fail")
	}
	if cc.compareOrdered(clickStored{Positions: []Point{{X: 1, Y: 2}}}, "bad") {
		t.Fatal("type mismatch should fail")
	}
}

func TestClickCaptcha_positionsMatchOrdered_Tolerance(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 2, 10, 5*time.Minute, nil)
	stored := []Point{{X: 50, Y: 50}, {X: 100, Y: 100}}
	// Within tolerance.
	if !cc.positionsMatchOrdered(stored, []Point{{X: 55, Y: 55}, {X: 95, Y: 105}}) {
		t.Fatal("should pass within tolerance")
	}
	// Outside tolerance.
	if cc.positionsMatchOrdered(stored, []Point{{X: 80, Y: 50}, {X: 100, Y: 100}}) {
		t.Fatal("should fail outside tolerance")
	}
}

func TestClickCaptcha_generateWords(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, nil)
	words := cc.generateWords(5)
	if len(words) != 5 {
		t.Fatalf("expected 5 words, got %d", len(words))
	}
	seen := map[string]bool{}
	for _, w := range words {
		if seen[w] {
			t.Fatalf("duplicate word: %s", w)
		}
		seen[w] = true
	}
}

func TestRect_overlaps(t *testing.T) {
	r1 := rect{x1: 0, y1: 0, x2: 10, y2: 10}
	r2 := rect{x1: 5, y1: 5, x2: 15, y2: 15}
	r3 := rect{x1: 20, y1: 20, x2: 30, y2: 30}
	if !r1.overlaps(r2) {
		t.Fatal("r1 should overlap r2")
	}
	if r1.overlaps(r3) {
		t.Fatal("r1 should not overlap r3")
	}
}
