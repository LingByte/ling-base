package captcha

import (
	"testing"
	"time"
)

func TestSliderCaptcha_Generate(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, nil)
	result, err := sc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.ID == "" || result.Type != TypeSlider {
		t.Fatalf("bad result: %+v", result)
	}
	if result.Data["trackWidth"].(int) != 300 {
		t.Fatal("trackWidth should be 300")
	}
}

func TestSliderCaptcha_Verify_Pass(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, nil)
	result, _ := sc.Generate()
	ok, err := sc.Verify(result.ID, 280)
	if err != nil || !ok {
		t.Fatalf("expected pass at 280, got %v, %v", ok, err)
	}
}

func TestSliderCaptcha_Verify_Fail(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, nil)
	result, _ := sc.Generate()
	ok, err := sc.Verify(result.ID, 50)
	if err != nil || ok {
		t.Fatalf("expected fail at 50, got %v, %v", ok, err)
	}
}

func TestSliderCaptcha_Verify_NotFound(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, nil)
	ok, err := sc.Verify("missing", 280)
	if err == nil || ok {
		t.Fatalf("expected error, got %v, %v", ok, err)
	}
}

func TestSliderCaptcha_Defaults(t *testing.T) {
	// Zero / invalid values should fall back to defaults.
	sc := NewSliderCaptcha(0, 0, 5*time.Minute, nil)
	if sc.trackWidth != defaultSliderTrackWidth {
		t.Fatalf("expected default trackWidth, got %d", sc.trackWidth)
	}
	if sc.passRatio != 0.92 {
		t.Fatalf("expected default passRatio 0.92, got %v", sc.passRatio)
	}
}

func TestSliderCaptcha_Verify_OverTrack(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, nil)
	result, _ := sc.Generate()
	// x > trackWidth should fail.
	ok, err := sc.Verify(result.ID, 400)
	if err != nil || ok {
		t.Fatalf("expected fail at 400, got %v, %v", ok, err)
	}
}

func TestSliderCaptcha_compare_ZeroTrackWidth(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, nil)
	if sc.compare(0, 100) {
		t.Fatal("compare with zero trackWidth should fail")
	}
}

func TestIntValue(t *testing.T) {
	tests := []struct {
		input  interface{}
		expect int
	}{
		{int(42), 42},
		{int32(42), 42},
		{int64(42), 42},
		{float64(42.7), 42},
		{float32(42.7), 42},
		{"hello", 0},
		{nil, 0},
	}
	for _, tc := range tests {
		if got := intValue(tc.input); got != tc.expect {
			t.Fatalf("intValue(%v) = %d, want %d", tc.input, got, tc.expect)
		}
	}
}
