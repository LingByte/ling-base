package cache_test

import (
	"testing"
	"time"

	"github.com/LingByte/ling-base/cache"
)

func TestApplyOptions(t *testing.T) {
	o := cache.ApplyOptions(
		cache.WithPrefix("p:"),
		cache.WithDefaultTTL(time.Minute),
		nil,
	)
	if o.Prefix != "p:" || o.DefaultTTL != time.Minute {
		t.Fatalf("ApplyOptions = %+v", o)
	}
}

func TestOptionsResolveTTL(t *testing.T) {
	o := cache.Options{DefaultTTL: time.Hour}
	if got := o.ResolveTTL(30 * time.Second); got != 30*time.Second {
		t.Fatalf("ResolveTTL(positive) = %v", got)
	}
	if got := o.ResolveTTL(0); got != time.Hour {
		t.Fatalf("ResolveTTL(zero) = %v", got)
	}
}

func TestOptionsKey(t *testing.T) {
	if got := (cache.Options{}).Key("k"); got != "k" {
		t.Fatalf("Key without prefix = %q", got)
	}
	if got := (cache.Options{Prefix: "app:"}).Key("k"); got != "app:k" {
		t.Fatalf("Key with prefix = %q", got)
	}
}
