package cache_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/cache"
	"github.com/LingByte/ling-base/cache/lru"
)

func TestHelpers(t *testing.T) {
	c, err := lru.New(10, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := cache.SetString(ctx, c, "name", "ling", time.Minute); err != nil {
		t.Fatal(err)
	}
	got, err := cache.GetString(ctx, c, "name")
	if err != nil || got != "ling" {
		t.Fatalf("GetString = %q, %v", got, err)
	}

	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := cache.SetJSON(ctx, c, "user", user{ID: 1, Name: "a"}, 0); err != nil {
		t.Fatal(err)
	}
	var u user
	if err := cache.GetJSON(ctx, c, "user", &u); err != nil {
		t.Fatal(err)
	}
	if u.ID != 1 || u.Name != "a" {
		t.Fatalf("unexpected user: %+v", u)
	}
}

func TestGetOrSet(t *testing.T) {
	c, err := lru.New(10, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	calls := 0
	fn := func(context.Context) ([]byte, error) {
		calls++
		return []byte("computed"), nil
	}

	v1, err := cache.GetOrSet(ctx, c, "k", time.Minute, fn)
	if err != nil || string(v1) != "computed" || calls != 1 {
		t.Fatalf("first GetOrSet = %q, calls=%d, err=%v", v1, calls, err)
	}
	v2, err := cache.GetOrSet(ctx, c, "k", time.Minute, fn)
	if err != nil || string(v2) != "computed" || calls != 1 {
		t.Fatalf("second GetOrSet = %q, calls=%d, err=%v", v2, calls, err)
	}
}

func TestGetOrSetMissOnly(t *testing.T) {
	c, err := lru.New(1, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	_, err = cache.GetOrSet(context.Background(), c, "k", 0, nil)
	if !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestGetJSONErrors(t *testing.T) {
	c, err := lru.New(10, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()
	if err := c.Set(ctx, "bad", []byte("not-json"), 0); err != nil {
		t.Fatal(err)
	}
	var dest struct{}
	if err := cache.GetJSON(ctx, c, "bad", &dest); err == nil {
		t.Fatal("expected unmarshal error")
	}
	if err := cache.SetJSON(ctx, c, "x", make(chan int), 0); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestGetOrSetFnError(t *testing.T) {
	c, err := lru.New(1, lru.WithCleanupInterval(0))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	want := errors.New("fn failed")
	_, err = cache.GetOrSet(context.Background(), c, "k", 0, func(context.Context) ([]byte, error) {
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("expected fn error, got %v", err)
	}
}

type errCache struct {
	getErr error
	setErr error
}

func (e errCache) Get(context.Context, string) ([]byte, error) { return nil, e.getErr }
func (e errCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return e.setErr
}
func (errCache) Delete(context.Context, string) error           { return nil }
func (errCache) Exists(context.Context, string) (bool, error)   { return false, nil }
func (errCache) Clear(context.Context) error                    { return nil }
func (errCache) Close() error                                   { return nil }

func TestGetOrSetGetError(t *testing.T) {
	boom := errors.New("get failed")
	_, err := cache.GetOrSet(context.Background(), errCache{getErr: boom}, "k", 0, func(context.Context) ([]byte, error) {
		t.Fatal("fn should not run")
		return nil, nil
	})
	if !errors.Is(err, boom) {
		t.Fatalf("get error = %v", err)
	}
}

func TestGetOrSetSetError(t *testing.T) {
	setErr := errors.New("set failed")
	_, err := cache.GetOrSet(context.Background(), errCache{getErr: cache.ErrNotFound, setErr: setErr}, "k", 0, func(context.Context) ([]byte, error) {
		return []byte("v"), nil
	})
	if !errors.Is(err, setErr) {
		t.Fatalf("set error = %v", err)
	}
}
