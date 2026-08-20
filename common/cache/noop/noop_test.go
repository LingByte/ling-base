package noop_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/cache"
	"github.com/LingByte/ling-base/common/cache/noop"
)

func TestNoop(t *testing.T) {
	c := noop.New[string, []byte]()
	ctx := context.Background()

	if _, err := c.Get(ctx, "k"); !errors.Is(err, cache.ErrNotFound) {
		t.Fatalf("Get = %v", err)
	}
	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatal(err)
	}
	ok, err := c.Exists(ctx, "k")
	if err != nil || ok {
		t.Fatalf("Exists = %v, %v", ok, err)
	}
	if err := c.Clear(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
