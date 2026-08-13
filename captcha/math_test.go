package captcha

import (
	"strings"
	"testing"
	"time"
)

func TestMathCaptcha_Generate(t *testing.T) {
	mc := NewMathCaptcha(5*time.Minute, nil)
	result, err := mc.Generate()
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if result.ID == "" || result.Type != TypeMath {
		t.Fatalf("bad result: %+v", result)
	}
	question, _ := result.Data["question"].(string)
	if !strings.Contains(question, "=") {
		t.Fatalf("question should contain '=': %q", question)
	}
	if !strings.Contains(question, "?") {
		t.Fatalf("question should contain '?': %q", question)
	}
}

func TestMathCaptcha_Verify_Correct(t *testing.T) {
	store := NewMemoryStore()
	mc := NewMathCaptcha(5*time.Minute, store)
	result, _ := mc.Generate()

	stored, _ := store.Get(result.ID)
	ms := stored.(mathStored)

	ok, err := mc.Verify(result.ID, ms.Answer)
	if err != nil || !ok {
		t.Fatalf("expected pass with correct answer %d, got %v, %v", ms.Answer, ok, err)
	}
}

func TestMathCaptcha_Verify_Wrong(t *testing.T) {
	store := NewMemoryStore()
	mc := NewMathCaptcha(5*time.Minute, store)
	result, _ := mc.Generate()

	ok, err := mc.Verify(result.ID, -999)
	if err != nil || ok {
		t.Fatalf("expected fail with wrong answer, got %v, %v", ok, err)
	}
}

func TestMathCaptcha_Verify_NotFound(t *testing.T) {
	mc := NewMathCaptcha(5*time.Minute, nil)
	ok, err := mc.Verify("missing", 42)
	if err == nil || ok {
		t.Fatalf("expected error, got %v, %v", ok, err)
	}
}

func TestMathCaptcha_VerifyString_Correct(t *testing.T) {
	store := NewMemoryStore()
	mc := NewMathCaptcha(5*time.Minute, store)
	result, _ := mc.Generate()

	stored, _ := store.Get(result.ID)
	ms := stored.(mathStored)

	ok, err := mc.VerifyString(result.ID, intToString(ms.Answer))
	if err != nil || !ok {
		t.Fatalf("expected pass, got %v, %v", ok, err)
	}
}

func TestMathCaptcha_VerifyString_Invalid(t *testing.T) {
	store := NewMemoryStore()
	mc := NewMathCaptcha(5*time.Minute, store)
	result, _ := mc.Generate()

	ok, err := mc.VerifyString(result.ID, "not-a-number")
	if err != nil || ok {
		t.Fatalf("expected fail with non-numeric, got %v, %v", ok, err)
	}
}

func TestMathCaptcha_VerifyString_Empty(t *testing.T) {
	store := NewMemoryStore()
	mc := NewMathCaptcha(5*time.Minute, store)
	result, _ := mc.Generate()

	ok, err := mc.VerifyString(result.ID, "  ")
	if err != nil || ok {
		t.Fatalf("expected fail with empty, got %v, %v", ok, err)
	}
}

func TestMathCaptcha_generateProblem(t *testing.T) {
	mc := NewMathCaptcha(5*time.Minute, nil)
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		a, b, op, answer := mc.generateProblem()
		if a <= 0 || b <= 0 {
			t.Fatalf("operands should be positive: %d, %d", a, b)
		}
		if answer < 0 {
			t.Fatalf("answer should be non-negative: %d", answer)
		}
		seen[op] = true
		// Verify the answer is correct.
		switch op {
		case "+":
			if a+b != answer {
				t.Fatalf("%d + %d != %d", a, b, answer)
			}
		case "-":
			if a-b != answer {
				t.Fatalf("%d - %d != %d", a, b, answer)
			}
		case "x":
			if a*b != answer {
				t.Fatalf("%d x %d != %d", a, b, answer)
			}
		default:
			t.Fatalf("unexpected operator: %s", op)
		}
	}
	if !seen["+"] || !seen["-"] || !seen["x"] {
		t.Fatalf("expected all three operators, got %v", seen)
	}
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
