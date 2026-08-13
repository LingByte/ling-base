package captcha

import (
	"image"
	"strings"
	"testing"
	"time"
)

func TestRotateCaptcha_Generate(t *testing.T) {
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, nil)
	result, err := rc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.ID == "" || result.Type != TypeRotate {
		t.Fatalf("bad result: %+v", result)
	}
	imgURL, _ := result.Data["image"].(string)
	if !strings.HasPrefix(imgURL, "data:image/png;base64,") {
		t.Fatal("image should be base64 PNG")
	}
	if result.Data["size"].(int) != 200 {
		t.Fatal("size should be 200")
	}
	if result.Data["tolerance"].(int) != 15 {
		t.Fatal("tolerance should be 15")
	}
}

func TestRotateCaptcha_Verify_Correct(t *testing.T) {
	store := NewMemoryStore()
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, store)
	result, _ := rc.Generate()

	stored, _ := store.Get(result.ID)
	rs := stored.(rotateStored)

	// User rotates back by the same angle -> residual = 0.
	ok, err := rc.Verify(result.ID, rs.Angle)
	if err != nil || !ok {
		t.Fatalf("expected pass with angle=%d, got %v, %v", rs.Angle, ok, err)
	}
}

func TestRotateCaptcha_Verify_WithinTolerance(t *testing.T) {
	store := NewMemoryStore()
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, store)
	result, _ := rc.Generate()

	stored, _ := store.Get(result.ID)
	rs := stored.(rotateStored)

	// Off by 10 degrees (within tolerance of 15).
	ok, err := rc.Verify(result.ID, rs.Angle-10)
	if err != nil || !ok {
		t.Fatalf("expected pass within tolerance, got %v, %v", ok, err)
	}
}

func TestRotateCaptcha_Verify_OutsideTolerance(t *testing.T) {
	store := NewMemoryStore()
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, store)
	result, _ := rc.Generate()

	stored, _ := store.Get(result.ID)
	rs := stored.(rotateStored)

	// Off by 90 degrees.
	ok, err := rc.Verify(result.ID, rs.Angle-90)
	if err != nil || ok {
		t.Fatalf("expected fail outside tolerance, got %v, %v", ok, err)
	}
}

func TestRotateCaptcha_Verify_WrapAround(t *testing.T) {
	store := NewMemoryStore()
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, store)
	result, _ := rc.Generate()

	stored, _ := store.Get(result.ID)
	rs := stored.(rotateStored)

	// Submit userAngle = storedAngle + 360 (same rotation, wrapped).
	// residual = (storedAngle - (storedAngle + 360)) % 360 = -360 % 360 = 0.
	ok, err := rc.Verify(result.ID, rs.Angle+360)
	if err != nil || !ok {
		t.Fatalf("expected pass with wrapped angle %d, got %v, %v", rs.Angle+360, ok, err)
	}
}

func TestRotateCaptcha_Verify_NotFound(t *testing.T) {
	rc := NewRotateCaptcha(200, 15, 5*time.Minute, nil)
	ok, err := rc.Verify("missing", 0)
	if err == nil || ok {
		t.Fatalf("expected error, got %v, %v", ok, err)
	}
}

func TestRotateCaptcha_Defaults(t *testing.T) {
	rc := NewRotateCaptcha(0, -1, 5*time.Minute, nil)
	if rc.size != defaultRotateSize {
		t.Fatalf("expected default size, got %d", rc.size)
	}
	if rc.tolerance != defaultRotateTolerance {
		t.Fatalf("expected default tolerance, got %d", rc.tolerance)
	}
}

func TestRotateCaptcha_rotate(t *testing.T) {
	rc := NewRotateCaptcha(100, 15, 5*time.Minute, nil)
	img := generateTestImage(100, 100)
	// Rotate by 0 (should be identical).
	rotated := rc.rotate(img, 0)
	bounds := rotated.Bounds()
	if bounds.Dx() != 100 || bounds.Dy() != 100 {
		t.Fatalf("bad dimensions: %dx%d", bounds.Dx(), bounds.Dy())
	}
	// Rotate by 90.
	rotated90 := rc.rotate(img, 90)
	if rotated90.Bounds().Dx() != 100 {
		t.Fatal("rotated image should have same dimensions")
	}
}

func TestRotateCaptcha_generateImage(t *testing.T) {
	rc := NewRotateCaptcha(100, 15, 5*time.Minute, nil)
	rng := newTestRNG()
	img := rc.generateImage(rng)
	if img.Bounds().Dx() != 100 || img.Bounds().Dy() != 100 {
		t.Fatalf("bad dimensions: %dx%d", img.Bounds().Dx(), img.Bounds().Dy())
	}
}

// Helper: create a simple test image.
func generateTestImage(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, image.NewRGBA(image.Rect(0, 0, 1, 1)).At(0, 0))
		}
	}
	return img
}
