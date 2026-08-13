package captcha

import (
	"math/rand"
	"testing"
	"time"
)

// newTestRNG creates a seeded RNG for deterministic tests.
func newTestRNG() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

func TestManager_GenerateAllTypes(t *testing.T) {
	m := NewManager(DefaultConfig())
	types := []Type{TypeImage, TypeClick, TypeSlider, TypeMath, TypeJigsaw, TypeRotate}
	for _, typ := range types {
		result, err := m.Generate(typ)
		if err != nil {
			t.Fatalf("Generate(%s) failed: %v", typ, err)
		}
		if result.Type != typ {
			t.Fatalf("expected type %s, got %s", typ, result.Type)
		}
	}
}

func TestManager_GenerateUnsupported(t *testing.T) {
	m := NewManager(DefaultConfig())
	_, err := m.Generate("unknown")
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestManager_VerifyAllTypes(t *testing.T) {
	m := NewManager(DefaultConfig())

	// Image
	imgResult, _ := m.Generate(TypeImage)
	store := m.store
	stored, _ := store.Get(imgResult.ID)
	code := stored.(string)
	ok, err := m.Verify(Payload{ID: imgResult.ID, Type: TypeImage, Value: code})
	if err != nil || !ok {
		t.Fatalf("image verify failed: %v, %v", ok, err)
	}

	// Slider
	sliderResult, _ := m.Generate(TypeSlider)
	ok, err = m.Verify(Payload{ID: sliderResult.ID, Type: TypeSlider, Value: 300})
	if err != nil || !ok {
		t.Fatalf("slider verify failed: %v, %v", ok, err)
	}

	// Math
	mathResult, _ := m.Generate(TypeMath)
	stored, _ = store.Get(mathResult.ID)
	ms := stored.(mathStored)
	ok, err = m.Verify(Payload{ID: mathResult.ID, Type: TypeMath, Value: ms.Answer})
	if err != nil || !ok {
		t.Fatalf("math verify failed: %v, %v", ok, err)
	}

	// Jigsaw
	jigsawResult, _ := m.Generate(TypeJigsaw)
	stored, _ = store.Get(jigsawResult.ID)
	js := stored.(jigsawStored)
	ok, err = m.Verify(Payload{ID: jigsawResult.ID, Type: TypeJigsaw, Value: js.TargetX})
	if err != nil || !ok {
		t.Fatalf("jigsaw verify failed: %v, %v", ok, err)
	}

	// Rotate
	rotateResult, _ := m.Generate(TypeRotate)
	stored, _ = store.Get(rotateResult.ID)
	rs := stored.(rotateStored)
	ok, err = m.Verify(Payload{ID: rotateResult.ID, Type: TypeRotate, Value: rs.Angle})
	if err != nil || !ok {
		t.Fatalf("rotate verify failed: %v, %v", ok, err)
	}

	// Click
	clickResult, _ := m.Generate(TypeClick)
	targets, _ := clickResult.Data["targets"].([]string)
	chars, _ := clickResult.Data["chars"].([]CharMarker)
	byChar := map[string]Point{}
	for _, c := range chars {
		byChar[c.Char] = Point{X: c.X, Y: c.Y}
	}
	ordered := make([]Point, len(targets))
	for i, target := range targets {
		ordered[i] = byChar[target]
	}
	ok, err = m.Verify(Payload{ID: clickResult.ID, Type: TypeClick, Value: ordered})
	if err != nil || !ok {
		t.Fatalf("click verify failed: %v, %v", ok, err)
	}
}

func TestManager_Verify_UnsupportedType(t *testing.T) {
	m := NewManager(DefaultConfig())
	_, err := m.Verify(Payload{ID: "x", Type: "unknown", Value: 1})
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestManager_Verify_ImageBadValue(t *testing.T) {
	m := NewManager(DefaultConfig())
	_, err := m.Verify(Payload{ID: "x", Type: TypeImage, Value: 123})
	if err == nil {
		t.Fatal("expected error for non-string image value")
	}
}

func TestManager_Verify_ClickBadValue(t *testing.T) {
	m := NewManager(DefaultConfig())
	_, err := m.Verify(Payload{ID: "x", Type: TypeClick, Value: "bad"})
	if err == nil {
		t.Fatal("expected error for bad click value")
	}
}

func TestManager_Verify_ClickBadPoint(t *testing.T) {
	m := NewManager(DefaultConfig())
	_, err := m.Verify(Payload{ID: "x", Type: TypeClick, Value: []interface{}{123}})
	if err == nil {
		t.Fatal("expected error for bad click point")
	}
}

func TestManager_Verify_ClickMapPoints(t *testing.T) {
	m := NewManager(DefaultConfig())
	result, _ := m.Generate(TypeClick)
	targets, _ := result.Data["targets"].([]string)
	chars, _ := result.Data["chars"].([]CharMarker)
	byChar := map[string]Point{}
	for _, c := range chars {
		byChar[c.Char] = Point{X: c.X, Y: c.Y}
	}
	// Submit as []interface{} of map[string]interface{} (simulates JSON unmarshal).
	mapPoints := make([]interface{}, len(targets))
	for i, target := range targets {
		p := byChar[target]
		mapPoints[i] = map[string]interface{}{"x": p.X, "y": p.Y}
	}
	ok, err := m.Verify(Payload{ID: result.ID, Type: TypeClick, Value: mapPoints})
	if err != nil || !ok {
		t.Fatalf("click verify with map points failed: %v, %v", ok, err)
	}
}

func TestNewManager_NilConfig(t *testing.T) {
	m := NewManager(nil)
	if m == nil {
		t.Fatal("NewManager(nil) should use defaults")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ImageWidth != 200 || cfg.Expiration == 0 {
		t.Fatalf("bad default config: %+v", cfg)
	}
}

func TestInitGlobalManager(t *testing.T) {
	InitGlobalManager(DefaultConfig())
	if GlobalManager == nil {
		t.Fatal("GlobalManager should be initialized")
	}
}

func TestEnsureGlobalManager(t *testing.T) {
	GlobalManager = nil
	m := EnsureGlobalManager()
	if m == nil {
		t.Fatal("EnsureGlobalManager should create manager")
	}
}

func TestVerifyPayload(t *testing.T) {
	GlobalManager = nil
	// Empty ID/type should fail.
	_, err := VerifyPayload(Payload{ID: "", Type: ""})
	if err == nil {
		t.Fatal("expected error for empty payload")
	}

	// Valid flow.
	m := NewManager(DefaultConfig())
	GlobalManager = m
	result, _ := m.Generate(TypeMath)
	stored, _ := m.store.Get(result.ID)
	ms := stored.(mathStored)
	ok, err := VerifyPayload(Payload{ID: result.ID, Type: TypeMath, Value: ms.Answer})
	if err != nil || !ok {
		t.Fatalf("VerifyPayload failed: %v, %v", ok, err)
	}
}

func TestValidatePayload(t *testing.T) {
	GlobalManager = nil
	// Empty -> ErrPayloadRequired.
	err := ValidatePayload("", "", nil)
	if err != ErrPayloadRequired {
		t.Fatalf("expected ErrPayloadRequired, got %v", err)
	}

	// Valid.
	m := NewManager(DefaultConfig())
	GlobalManager = m
	result, _ := m.Generate(TypeSlider)
	err = ValidatePayload(result.ID, string(TypeSlider), 300)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	// Wrong answer -> ErrPayloadInvalid.
	result2, _ := m.Generate(TypeSlider)
	err = ValidatePayload(result2.ID, string(TypeSlider), 10)
	if err != ErrPayloadInvalid {
		t.Fatalf("expected ErrPayloadInvalid, got %v", err)
	}
}

func TestParsePoints(t *testing.T) {
	// []Point directly.
	pts, err := parsePoints([]Point{{X: 1, Y: 2}})
	if err != nil || len(pts) != 1 || pts[0].X != 1 {
		t.Fatalf("parsePoints []Point failed: %v, %v", pts, err)
	}

	// []interface{} of Point.
	pts, err = parsePoints([]interface{}{Point{X: 3, Y: 4}})
	if err != nil || pts[0].X != 3 {
		t.Fatalf("parsePoints []interface{} Point failed: %v, %v", pts, err)
	}

	// []interface{} of map.
	pts, err = parsePoints([]interface{}{map[string]interface{}{"x": 5, "y": 6}})
	if err != nil || pts[0].X != 5 || pts[0].Y != 6 {
		t.Fatalf("parsePoints map failed: %v, %v", pts, err)
	}

	// []interface{} of bad type.
	_, err = parsePoints([]interface{}{123})
	if err == nil {
		t.Fatal("expected error for bad point type")
	}

	// Invalid top-level type.
	_, err = parsePoints("bad")
	if err == nil {
		t.Fatal("expected error for bad top-level type")
	}
}
