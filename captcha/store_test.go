package captcha

import (
	"strings"
	"testing"
	"time"
)

func TestNewMemoryStore(t *testing.T) {
	store := NewMemoryStore()
	if store == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
	if store.data == nil {
		t.Fatal("store.data is nil")
	}
}

func TestMemoryStore_SetGet(t *testing.T) {
	store := NewMemoryStore()
	id := "test-id"
	code := "ABCD"
	expires := time.Now().Add(5 * time.Minute)

	if err := store.Set(id, code, expires); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	retrieved, err := store.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if retrieved.(string) != code {
		t.Fatalf("Expected %s, got %v", code, retrieved)
	}
}

func TestMemoryStore_GetNotFound(t *testing.T) {
	store := NewMemoryStore()
	_, err := store.Get("non-existent")
	if err == nil {
		t.Fatal("Expected error for non-existent captcha")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("Expected 'not found' error, got: %v", err)
	}
}

func TestMemoryStore_GetExpired(t *testing.T) {
	store := NewMemoryStore()
	id := "expired-id"
	_ = store.Set(id, "ABCD", time.Now().Add(-1*time.Minute))
	time.Sleep(50 * time.Millisecond)
	_, err := store.Get(id)
	if err == nil {
		t.Fatal("Expected error for expired captcha")
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("test-id", "ABCD", time.Now().Add(5*time.Minute))
	if err := store.Delete("test-id"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := store.Get("test-id"); err == nil {
		t.Fatal("Expected error after delete")
	}
}

func TestMemoryStore_VerifyWithFunc_Success(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("id1", "ABCD", time.Now().Add(5*time.Minute))
	ok, err := store.VerifyWithFunc("id1", "ABCD", func(stored, input interface{}) bool {
		return stored.(string) == input.(string)
	})
	if err != nil || !ok {
		t.Fatalf("expected ok=true, got %v, %v", ok, err)
	}
	// Should be deleted after success.
	if _, err := store.Get("id1"); err == nil {
		t.Fatal("should be deleted after verify")
	}
}

func TestMemoryStore_VerifyWithFunc_Wrong(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("id2", "ABCD", time.Now().Add(5*time.Minute))
	ok, err := store.VerifyWithFunc("id2", "WRONG", func(stored, input interface{}) bool {
		return stored.(string) == input.(string)
	})
	if err != nil || ok {
		t.Fatalf("expected ok=false, got %v, %v", ok, err)
	}
	// Should still exist after failure.
	if _, err := store.Get("id2"); err != nil {
		t.Fatal("should still exist after failed verify")
	}
}

func TestMemoryStore_VerifyWithFunc_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ok, err := store.VerifyWithFunc("missing", "x", func(stored, input interface{}) bool {
		return true
	})
	if err == nil || ok {
		t.Fatalf("expected error and ok=false, got %v, %v", ok, err)
	}
}

func TestMemoryStore_VerifyWithFunc_NilCompare_DefaultString(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("id3", "Hello", time.Now().Add(5*time.Minute))
	// nil compareFunc -> default case-insensitive string comparison.
	ok, err := store.VerifyWithFunc("id3", "hello", nil)
	if err != nil || !ok {
		t.Fatalf("expected ok=true (case-insensitive default), got %v, %v", ok, err)
	}
}

func TestMemoryStore_VerifyWithFunc_NilCompare_NonString(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("id4", 42, time.Now().Add(5*time.Minute))
	ok, err := store.VerifyWithFunc("id4", 42, nil)
	if err != nil || ok {
		t.Fatalf("expected ok=false (non-string with nil compare), got %v, %v", ok, err)
	}
}

func TestMemoryStore_VerifyWithFuncWithoutDelete_Success(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("id5", "ABCD", time.Now().Add(5*time.Minute))
	ok, err := store.VerifyWithFuncWithoutDelete("id5", "ABCD", func(stored, input interface{}) bool {
		return stored.(string) == input.(string)
	})
	if err != nil || !ok {
		t.Fatalf("expected ok=true, got %v, %v", ok, err)
	}
	// Should still exist (no delete).
	if _, err := store.Get("id5"); err != nil {
		t.Fatal("should still exist after verify-without-delete")
	}
}

func TestMemoryStore_VerifyWithFuncWithoutDelete_NilCompare(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("id6", "Test", time.Now().Add(5*time.Minute))
	ok, err := store.VerifyWithFuncWithoutDelete("id6", "test", nil)
	if err != nil || !ok {
		t.Fatalf("expected ok=true (case-insensitive default), got %v, %v", ok, err)
	}
}

func TestMemoryStore_VerifyWithFuncWithoutDelete_NilCompare_NonString(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("id7", 42, time.Now().Add(5*time.Minute))
	ok, err := store.VerifyWithFuncWithoutDelete("id7", 42, nil)
	if err != nil || ok {
		t.Fatalf("expected ok=false, got %v, %v", ok, err)
	}
}

func TestMemoryStore_VerifyWithFuncWithoutDelete_NotFound(t *testing.T) {
	store := NewMemoryStore()
	ok, err := store.VerifyWithFuncWithoutDelete("missing", "x", nil)
	if err == nil || ok {
		t.Fatalf("expected error, got %v, %v", ok, err)
	}
}

func TestMemoryStore_Cleanup(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Set("expire-me", "x", time.Now().Add(-1*time.Second))
	// Trigger cleanup (runs in goroutine from Set, but also call directly).
	store.cleanup()
	if _, err := store.Get("expire-me"); err == nil {
		t.Fatal("expected cleaned up entry to be gone")
	}
}
