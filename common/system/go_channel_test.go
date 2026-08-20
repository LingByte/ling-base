package system

import (
	"testing"
	"time"
)

func TestSafeSendBool(t *testing.T) {
	ch := make(chan bool, 1)
	if closed := SafeSendBool(ch, true); closed {
		t.Fatal("expected not closed")
	}
	if v := <-ch; v != true {
		t.Fatalf("got %v want true", v)
	}

	close(ch)
	if closed := SafeSendBool(ch, true); !closed {
		t.Fatal("expected closed")
	}
}

func TestSafeSendString(t *testing.T) {
	ch := make(chan string, 1)
	if closed := SafeSendString(ch, "hello"); closed {
		t.Fatal("expected not closed")
	}
	if v := <-ch; v != "hello" {
		t.Fatalf("got %q want hello", v)
	}

	close(ch)
	if closed := SafeSendString(ch, "hello"); !closed {
		t.Fatal("expected closed")
	}
}

func TestSafeSendStringTimeout(t *testing.T) {
	// unbuffered channel, no receiver → should timeout
	ch := make(chan string)
	start := time.Now()
	if sent := SafeSendStringTimeout(ch, "test", 1); sent {
		t.Fatal("expected timeout (sent=false)")
	}
	elapsed := time.Since(start)
	if elapsed < 900*time.Millisecond {
		t.Fatalf("timeout too fast: %v", elapsed)
	}

	// buffered channel with space → should send immediately
	ch2 := make(chan string, 1)
	if sent := SafeSendStringTimeout(ch2, "test", 1); !sent {
		t.Fatal("expected sent=true for buffered channel")
	}
	if v := <-ch2; v != "test" {
		t.Fatalf("got %q want test", v)
	}

	// closed channel → recover, return false
	close(ch)
	if sent := SafeSendStringTimeout(ch, "test", 1); sent {
		t.Fatal("expected sent=false for closed channel")
	}
}
