package captcha

import (
	"strings"
	"testing"
	"time"
)

func TestJigsawCaptcha_Generate(t *testing.T) {
	jc := NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, nil)
	result, err := jc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.ID == "" || result.Type != TypeJigsaw {
		t.Fatalf("bad result: %+v", result)
	}
	bg, _ := result.Data["background"].(string)
	if !strings.HasPrefix(bg, "data:image/png;base64,") {
		t.Fatal("background should be base64 PNG")
	}
	piece, _ := result.Data["piece"].(string)
	if !strings.HasPrefix(piece, "data:image/png;base64,") {
		t.Fatal("piece should be base64 PNG")
	}
	if result.Data["width"].(int) != 300 {
		t.Fatal("width should be 300")
	}
	if result.Data["pieceSize"].(int) != 40 {
		t.Fatal("pieceSize should be 40")
	}
}

func TestJigsawCaptcha_Verify_Correct(t *testing.T) {
	store := NewMemoryStore()
	jc := NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, store)
	result, _ := jc.Generate()

	stored, _ := store.Get(result.ID)
	js := stored.(jigsawStored)

	ok, err := jc.Verify(result.ID, js.TargetX)
	if err != nil || !ok {
		t.Fatalf("expected pass at targetX=%d, got %v, %v", js.TargetX, ok, err)
	}
}

func TestJigsawCaptcha_Verify_WithinTolerance(t *testing.T) {
	store := NewMemoryStore()
	jc := NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, store)
	result, _ := jc.Generate()

	stored, _ := store.Get(result.ID)
	js := stored.(jigsawStored)

	ok, err := jc.Verify(result.ID, js.TargetX+3)
	if err != nil || !ok {
		t.Fatalf("expected pass within tolerance (offset 3), got %v, %v", ok, err)
	}
}

func TestJigsawCaptcha_Verify_OutsideTolerance(t *testing.T) {
	store := NewMemoryStore()
	jc := NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, store)
	result, _ := jc.Generate()

	stored, _ := store.Get(result.ID)
	js := stored.(jigsawStored)

	ok, err := jc.Verify(result.ID, js.TargetX+50)
	if err != nil || ok {
		t.Fatalf("expected fail outside tolerance, got %v, %v", ok, err)
	}
}

func TestJigsawCaptcha_Verify_NotFound(t *testing.T) {
	jc := NewJigsawCaptcha(300, 150, 40, 5, 5*time.Minute, nil)
	ok, err := jc.Verify("missing", 100)
	if err == nil || ok {
		t.Fatalf("expected error, got %v, %v", ok, err)
	}
}

func TestJigsawCaptcha_Defaults(t *testing.T) {
	jc := NewJigsawCaptcha(0, 0, 0, -1, 5*time.Minute, nil)
	if jc.width != defaultJigsawWidth {
		t.Fatalf("expected default width, got %d", jc.width)
	}
	if jc.height != defaultJigsawHeight {
		t.Fatalf("expected default height, got %d", jc.height)
	}
	if jc.pieceSize != defaultJigsawPieceSize {
		t.Fatalf("expected default pieceSize, got %d", jc.pieceSize)
	}
	if jc.tolerance != defaultJigsawTolerance {
		t.Fatalf("expected default tolerance, got %d", jc.tolerance)
	}
}

func TestEncodePNGDataURL(t *testing.T) {
	img := generateTestImage(10, 10)
	url, err := encodePNGDataURL(img)
	if err != nil || !strings.HasPrefix(url, "data:image/png;base64,") {
		t.Fatalf("bad data URL: %v, %v", url, err)
	}
}
