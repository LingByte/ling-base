package lock_test

import (
	"testing"

	"github.com/LingByte/ling-base/common/lock"
)

func TestNewToken(t *testing.T) {
	t1, err := lock.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if len(t1) != 32 {
		t.Fatalf("token length = %d", len(t1))
	}
	t2, err := lock.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if t1 == t2 {
		t.Fatal("expected distinct tokens")
	}
}

func TestResolveValueExplicit(t *testing.T) {
	o := lock.Options{Value: "fixed"}
	v, err := lock.ResolveValue(&o)
	if err != nil || v != "fixed" {
		t.Fatalf("ResolveValue = %q, %v", v, err)
	}
	if o.Value != "fixed" {
		t.Fatalf("opts.Value mutated to %q", o.Value)
	}
}

func TestResolveValueGenerated(t *testing.T) {
	o := lock.Options{}
	v, err := lock.ResolveValue(&o)
	if err != nil {
		t.Fatal(err)
	}
	if v == "" || o.Value != v {
		t.Fatalf("generated value not stored: %q / %q", v, o.Value)
	}
}
