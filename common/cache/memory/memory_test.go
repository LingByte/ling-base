package memory_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/cache"
	"github.com/LingByte/ling-base/common/cache/memory"
)

func TestMemoryBasic(t *testing.T) {
	c := memory.New[string, []byte](memory.WithCleanupInterval(0))
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "a", []byte("1"), 0); err != nil {
		t.Fatal(err)
	}
	got, err := c.Get(ctx, "a")
	if err != nil || string(got) != "1" {
		t.Fatalf("Get = %q, %v", got, err)
	}
	ok, err := c.Exists(ctx, "a")
	if err != nil || !ok {
		t.Fatalf("Exists = %v, %v", ok, err)
	}
}

func TestMemoryTTLAndCleanup(t *testing.T) {
	c := memory.New[string, []byte](memory.WithCleanupInterval(20 * time.Millisecond))
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 30*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(80 * time.Millisecond)
	if c.Len() != 0 {
		t.Fatalf("Len after purge = %d", c.Len())
	}
}

func TestMemoryPrefixDefaultTTL(t *testing.T) {
	c := memory.New[string, []byte](
		memory.WithPrefix("x:"),
		memory.WithDefaultTTL(time.Hour),
		memory.WithCleanupInterval(0),
	)
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "k", []byte("v"), 0); err != nil {
		t.Fatal(err)
	}
	ok, _ := c.Exists(ctx, "k")
	if !ok {
		t.Fatal("expected key with default TTL")
	}
}

func TestMemoryDeleteClearClose(t *testing.T) {
	c := memory.New[string, []byte](memory.WithCleanupInterval(0))
	ctx := context.Background()
	_ = c.Set(ctx, "a", []byte("1"), 0)
	_ = c.Delete(ctx, "a")
	if _, err := c.Get(ctx, "a"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("after delete: %v", err)
	}
	_ = c.Set(ctx, "b", []byte("2"), 0)
	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if c.Len() != 0 {
		t.Fatalf("Len = %d", c.Len())
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Get(ctx, "x"); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("expected ErrClosed, got %v", err)
	}
}

func TestMemoryErrors(t *testing.T) {
	c := memory.New[string, []byte](memory.WithCleanupInterval(0))
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Get(ctx, "k"); err != context.Canceled {
		t.Fatalf("Get cancelled = %v", err)
	}
	if _, err := c.Get(context.Background(), "missing"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("not found = %v", err)
	}

	_ = c.Set(context.Background(), "exp", []byte("x"), 1*time.Nanosecond)
	time.Sleep(5 * time.Millisecond)
	if _, err := c.Get(context.Background(), "exp"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expired get = %v", err)
	}
	ok, err := c.Exists(context.Background(), "exp")
	if err != nil || ok {
		t.Fatalf("expired exists = %v, %v", ok, err)
	}

	c2 := memory.New[string, []byte](memory.WithCleanupInterval(0))
	_ = c2.Close()
	if err := c2.Clear(context.Background()); !errors.Is(err, cache.ErrClosed) {
		t.Fatalf("clear closed = %v", err)
	}
}

func TestMemoryConcurrent(t *testing.T) {
	c := memory.New[string, []byte](memory.WithCleanupInterval(0))
	defer c.Close()
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			k := string(rune('a' + i%26))
			_ = c.Set(ctx, k, []byte{byte(i)}, 0)
			_, _ = c.Get(ctx, k)
		}(i)
	}
	wg.Wait()
}
