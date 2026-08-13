package captcha

import (
	"errors"
	"testing"
	"time"
)

// failingStore is a test Store that always returns errors.
type failingStore struct{}

func (failingStore) Set(string, interface{}, time.Time) error {
	return errors.New("store unavailable")
}
func (failingStore) Get(string) (interface{}, error) {
	return nil, errors.New("store unavailable")
}
func (failingStore) Delete(string) error {
	return errors.New("store unavailable")
}
func (failingStore) VerifyWithFunc(string, interface{}, func(stored, input interface{}) bool) (bool, error) {
	return false, errors.New("store unavailable")
}
func (failingStore) VerifyWithFuncWithoutDelete(string, interface{}, func(stored, input interface{}) bool) (bool, error) {
	return false, errors.New("store unavailable")
}

// Test error paths in Generate for each captcha type.
func TestImageCaptcha_Generate_StoreError(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, failingStore{})
	_, err := c.Generate()
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestClickCaptcha_Generate_StoreError(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, failingStore{})
	_, err := cc.Generate()
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestSliderCaptcha_Generate_StoreError(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, failingStore{})
	_, err := sc.Generate()
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestMathCaptcha_Generate_StoreError(t *testing.T) {
	mc := NewMathCaptcha(5*time.Minute, failingStore{})
	_, err := mc.Generate()
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestJigsawCaptcha_Generate_StoreError(t *testing.T) {
	jc := NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, failingStore{})
	_, err := jc.Generate()
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestRotateCaptcha_Generate_StoreError(t *testing.T) {
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, failingStore{})
	_, err := rc.Generate()
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

// Test Verify type assertion failures (passing wrong value type).
func TestMathCaptcha_Verify_TypeMismatch(t *testing.T) {
	mc := NewMathCaptcha(5*time.Minute, nil)
	// compare with wrong types should return false, not panic.
	if mc.compare("not-mathStored", 0) {
		t.Fatal("type mismatch should return false")
	}
	if mc.compare(mathStored{Answer: 5}, "not-an-int") {
		t.Fatal("type mismatch should return false")
	}
}

func TestJigsawCaptcha_Verify_TypeMismatch(t *testing.T) {
	jc := NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, nil)
	// compare with wrong types should return false, not panic.
	if jc.compare("not-jigsawStored", 100) {
		t.Fatal("type mismatch should return false")
	}
	if jc.compare(jigsawStored{TargetX: 100}, "not-an-int") {
		t.Fatal("type mismatch should return false")
	}
}

func TestRotateCaptcha_Verify_TypeMismatch(t *testing.T) {
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, nil)
	ok := rc.compare(rotateStored{Angle: 90}, "not-an-int")
	if ok {
		t.Fatal("type mismatch should return false")
	}
}

func TestSliderCaptcha_Verify_StoreError(t *testing.T) {
	sc := NewSliderCaptcha(300, 0.9, 5*time.Minute, failingStore{})
	_, err := sc.Verify("x", 100)
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestClickCaptcha_Verify_StoreError(t *testing.T) {
	cc := NewClickCaptcha(300, 200, 3, 20, 5*time.Minute, failingStore{})
	_, err := cc.Verify("x", []Point{{X: 1, Y: 2}})
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestImageCaptcha_Verify_StoreError(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, failingStore{})
	_, err := c.Verify("x", "code")
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestImageCaptcha_VerifyWithoutDelete_StoreError(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, failingStore{})
	_, err := c.VerifyWithoutDelete("x", "code")
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

func TestMathCaptcha_VerifyString_StoreError(t *testing.T) {
	mc := NewMathCaptcha(5*time.Minute, failingStore{})
	_, err := mc.VerifyString("x", "42")
	if err == nil {
		t.Fatal("expected error from failing store")
	}
}

// Test RandomType with empty LoginCaptchaTypes.
func TestRandomType_Empty(t *testing.T) {
	orig := LoginCaptchaTypes
	LoginCaptchaTypes = nil
	defer func() { LoginCaptchaTypes = orig }()

	got := RandomType()
	if got != TypeSlider {
		t.Fatalf("expected TypeSlider for empty types, got %s", got)
	}
}

// Test RandomType with single element.
func TestRandomType_Single(t *testing.T) {
	orig := LoginCaptchaTypes
	LoginCaptchaTypes = []Type{TypeMath}
	defer func() { LoginCaptchaTypes = orig }()

	for i := 0; i < 10; i++ {
		if got := RandomType(); got != TypeMath {
			t.Fatalf("expected TypeMath, got %s", got)
		}
	}
}
