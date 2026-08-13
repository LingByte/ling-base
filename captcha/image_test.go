package captcha

import (
	"image"
	"strings"
	"testing"
	"time"
)

func TestNewImageCaptcha(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	if c == nil || c.width != 200 || c.height != 60 || c.length != 4 {
		t.Fatalf("bad config: %+v", c)
	}
}

func TestImageCaptcha_Generate(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	result, err := c.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.ID == "" || result.Type != TypeImage {
		t.Fatalf("bad result: %+v", result)
	}
	imgData, _ := result.Data["image"].(string)
	if !strings.HasPrefix(imgData, "data:image/png;base64,") {
		t.Fatal("image should be base64 PNG")
	}
	if result.Data["length"].(int) != 4 {
		t.Fatal("length should be 4")
	}
	if result.Expires.Before(time.Now()) {
		t.Fatal("expires should be in future")
	}
}

func TestImageCaptcha_Verify_Correct(t *testing.T) {
	store := NewMemoryStore()
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, store)
	result, _ := c.Generate()

	stored, _ := store.Get(result.ID)
	code := stored.(string)

	ok, err := c.Verify(result.ID, code)
	if err != nil || !ok {
		t.Fatalf("expected pass, got %v, %v", ok, err)
	}
}

func TestImageCaptcha_Verify_CaseInsensitive(t *testing.T) {
	store := NewMemoryStore()
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, store)
	result, _ := c.Generate()

	stored, _ := store.Get(result.ID)
	code := stored.(string)

	ok, err := c.Verify(result.ID, strings.ToLower(code))
	if err != nil || !ok {
		t.Fatalf("expected case-insensitive pass, got %v, %v", ok, err)
	}
}

func TestImageCaptcha_Verify_Wrong(t *testing.T) {
	store := NewMemoryStore()
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, store)
	result, _ := c.Generate()

	ok, err := c.Verify(result.ID, "WRONG")
	if err != nil || ok {
		t.Fatalf("expected fail, got %v, %v", ok, err)
	}
}

func TestImageCaptcha_Verify_NotFound(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	ok, err := c.Verify("missing", "x")
	if err == nil || ok {
		t.Fatalf("expected error, got %v, %v", ok, err)
	}
}

func TestImageCaptcha_VerifyWithoutDelete(t *testing.T) {
	store := NewMemoryStore()
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, store)
	result, _ := c.Generate()

	stored, _ := store.Get(result.ID)
	code := stored.(string)

	ok, err := c.VerifyWithoutDelete(result.ID, code)
	if err != nil || !ok {
		t.Fatalf("expected pass, got %v, %v", ok, err)
	}
	// Should still exist.
	if _, err := store.Get(result.ID); err != nil {
		t.Fatal("should still exist after verify-without-delete")
	}
}

func TestImageCaptcha_VerifyWithoutDelete_Wrong(t *testing.T) {
	store := NewMemoryStore()
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, store)
	result, _ := c.Generate()

	ok, err := c.VerifyWithoutDelete(result.ID, "WRONG")
	if err != nil || ok {
		t.Fatalf("expected fail, got %v, %v", ok, err)
	}
}

func TestImageCaptcha_generateCode(t *testing.T) {
	c := NewImageCaptcha(200, 60, 6, 5*time.Minute, nil)
	code := c.generateCode()
	if len(code) != 6 {
		t.Fatalf("expected length 6, got %d", len(code))
	}
}

func TestImageCaptcha_generateImage(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	img, err := c.generateImage("TEST")
	if err != nil {
		t.Fatalf("generateImage failed: %v", err)
	}
	bounds := img.Bounds()
	if bounds.Dx() != 200 || bounds.Dy() != 60 {
		t.Fatalf("bad dimensions: %dx%d", bounds.Dx(), bounds.Dy())
	}
}

func TestImageCaptcha_imageToBase64(t *testing.T) {
	c := NewImageCaptcha(200, 60, 4, 5*time.Minute, nil)
	img, _ := c.generateImage("TEST")
	b64, err := c.imageToBase64(img)
	if err != nil || !strings.HasPrefix(b64, "data:image/png;base64,") {
		t.Fatalf("bad base64: %v, %v", b64, err)
	}
}

func TestDrawLine(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	drawLine(img, 0, 0, 50, 50, image.Black)
	drawLine(img, 99, 99, 0, 0, image.White) // reverse direction
}

func TestAbs(t *testing.T) {
	if abs(-5) != 5 || abs(5) != 5 || abs(0) != 0 {
		t.Fatal("abs failed")
	}
}

func TestClampByte(t *testing.T) {
	if clampByte(100, 50) != 150 {
		t.Fatal("clamp 100+50")
	}
	if clampByte(10, -50) != 0 {
		t.Fatal("clamp 10-50 = 0")
	}
	if clampByte(200, 100) != 255 {
		t.Fatal("clamp 200+100 = 255")
	}
	if clampByte(128, 0) != 128 {
		t.Fatal("clamp 128+0 = 128")
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	if id1 == "" || id2 == "" || id1 == id2 {
		t.Fatalf("IDs should be unique: %q, %q", id1, id2)
	}
}

func TestImagePNGDataURL(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	url, err := imagePNGDataURL(img)
	if err != nil || !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("bad data URL: %v, %v", url, err)
	}
}
