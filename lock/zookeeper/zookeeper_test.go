package zookeeper_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-zookeeper/zk"

	"github.com/LingByte/ling-base/lock"
	lockzk "github.com/LingByte/ling-base/lock/zookeeper"
)

func TestNewMutexValidation(t *testing.T) {
	conn := newFakeConn()
	if _, err := lockzk.NewMutex(nil, "/locks"); err == nil {
		t.Fatal("nil conn")
	}
	if _, err := lockzk.NewMutex(conn, ""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := lockzk.NewMutex(conn, "/"); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("root: %v", err)
	}
}

func TestTryLockLowestSequence(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, err := lockzk.NewMutex(conn, "/locks/job", lock.WithValue("a"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}

	m2, _ := lockzk.NewMutex(conn, "/locks/job", lock.WithValue("b"))
	if err := m2.TryLock(ctx); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("second TryLock = %v", err)
	}

	if err := m.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m2.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	_ = m2.Unlock(ctx)
}

func TestLockWaitPredecessor(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()

	first, _ := lockzk.NewMutex(conn, "/locks/w", lock.WithValue("1"))
	if err := first.LockWait(ctx); err != nil {
		t.Fatal(err)
	}

	second, _ := lockzk.NewMutex(conn, "/locks/w", lock.WithValue("2"))
	done := make(chan error, 1)
	go func() { done <- second.LockWait(ctx) }()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected early acquire: %v", err)
		}
	case <-time.After(50 * time.Millisecond):
	}

	if err := first.Unlock(ctx); err != nil {
		t.Fatal(err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("waiter: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for lock")
	}
	_ = second.Unlock(ctx)
}

func TestUnlockRefreshNotHeld(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/x", lock.WithValue("v"))
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh = %v", err)
	}
}

func TestRefreshHeld(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/r", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	_ = m.Unlock(ctx)
}

func TestEnsurePathExistsError(t *testing.T) {
	conn := newFakeConn()
	conn.existsErr["/bad"] = errors.New("exists fail")
	m, _ := lockzk.NewMutex(conn, "/bad/path", lock.WithValue("v"))
	if err := m.TryLock(context.Background()); err == nil {
		t.Fatal("expected exists error")
	}
}

func TestEnsurePathCreateError(t *testing.T) {
	conn := newFakeConn()
	conn.createErr["/new"] = errors.New("create fail")
	m, _ := lockzk.NewMutex(conn, "/new/path", lock.WithValue("v"))
	if err := m.TryLock(context.Background()); err == nil {
		t.Fatal("expected create error")
	}
}

func TestTryLockNoSequenceChildren(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	if _, err := conn.Create("/locks", []byte{}, 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Create("/locks/ns", []byte{}, 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Create("/locks/ns/note", []byte("x"), 0, nil); err != nil {
		t.Fatal(err)
	}
	m, _ := lockzk.NewMutex(conn, "/locks/ns", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	// Remove lock- children scenario: manually clear lock nodes and leave only non-lock child
	children, _, _ := conn.Children("/locks/ns")
	for _, c := range children {
		if strings.HasPrefix(c, "lock-") {
			_ = conn.Delete("/locks/ns/"+c, -1)
		}
	}
	m2, _ := lockzk.NewMutex(conn, "/locks/ns", lock.WithValue("w"))
	// m2 creates new lock node; if only non-lock- siblings... actually after delete we have "note"
	if err := m2.TryLock(ctx); err != nil {
		t.Fatalf("TryLock = %v", err)
	}
	_ = m.Unlock(ctx)
	_ = m2.Unlock(ctx)
}

func TestCreateErrorOnLockNode(t *testing.T) {
	conn := newFakeConn()
	conn.createErr["/locks/f/lock-"] = errors.New("create fail")
	m, _ := lockzk.NewMutex(conn, "/locks/f", lock.WithValue("v"))
	if err := m.TryLock(context.Background()); err == nil {
		t.Fatal("expected create error")
	}
}

func TestEnsurePathNodeExists(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	// Pre-create parent so ensurePath hits ErrNodeExists branch on concurrent create.
	if _, err := conn.Create("/locks", []byte{}, 0, nil); err != nil && !errors.Is(err, zk.ErrNodeExists) {
		if _, err2 := conn.Create("/locks", []byte{}, 0, nil); err2 != nil && !errors.Is(err2, zk.ErrNodeExists) {
			t.Fatalf("create locks: %v", err)
		}
	}
	m, _ := lockzk.NewMutex(conn, "/locks/exists", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	_ = m.Unlock(ctx)
}

func TestChildrenError(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/c", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	conn.childrenErr["/locks/c"] = errors.New("children fail")
	m2, _ := lockzk.NewMutex(conn, "/locks/c", lock.WithValue("w"))
	if err := m2.TryLock(ctx); err == nil {
		t.Fatal("expected children error")
	}
}

func TestGetWNoNodeRetry(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m1, _ := lockzk.NewMutex(conn, "/locks/g", lock.WithValue("1"))
	if err := m1.LockWait(ctx); err != nil {
		t.Fatal(err)
	}
	m2, _ := lockzk.NewMutex(conn, "/locks/g", lock.WithValue("2"))
	go func() {
		time.Sleep(10 * time.Millisecond)
		_ = m1.Unlock(ctx)
	}()
	if err := m2.LockWait(ctx); err != nil {
		t.Fatalf("LockWait = %v", err)
	}
	_ = m2.Unlock(ctx)
}

func TestLockContextCancel(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	holder, _ := lockzk.NewMutex(conn, "/locks/cancel", lock.WithValue("h"))
	if err := holder.LockWait(ctx); err != nil {
		t.Fatal(err)
	}

	waiter, _ := lockzk.NewMutex(conn, "/locks/cancel", lock.WithValue("w"))
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- waiter.LockWait(cancelCtx) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
	_ = holder.Unlock(ctx)
}

func TestUnlockNoNodeIgnored(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/d", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	children, _, err := conn.Children("/locks/d")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range children {
		_ = conn.Delete("/locks/d/"+c, -1)
	}
	if err := m.Unlock(ctx); err != nil {
		t.Fatalf("Unlock after node gone = %v", err)
	}
}

func TestDeleteError(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/e", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	children, _, err := conn.Children("/locks/e")
	if err != nil || len(children) == 0 {
		t.Fatal("expected lock child")
	}
	nodePath := "/locks/e/" + children[len(children)-1]
	conn.deleteErr[nodePath] = errors.New("delete fail")
	if err := m.Unlock(ctx); err == nil {
		t.Fatal("expected delete error")
	}
}

func TestLockUsesLockWait(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/l", lock.WithValue("v"))
	if err := m.Lock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestContextCancelled(t *testing.T) {
	conn := newFakeConn()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := lockzk.NewMutex(conn, "/locks/z", lock.WithValue("v"))
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}

func TestLockWaitMissingFromChildren(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/miss", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	children, _, err := conn.Children("/locks/miss")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range children {
		_ = conn.Delete("/locks/miss/"+c, -1)
	}
	if err := m.LockWait(ctx); err != nil {
		t.Fatalf("LockWait = %v", err)
	}
}

func TestUnlockContextCancelled(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/u", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Unlock(cancelCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Unlock = %v", err)
	}
}

func TestRefreshContextCancelledWhenHeld(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, _ := lockzk.NewMutex(conn, "/locks/r", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	cancelCtx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := m.Refresh(cancelCtx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Refresh = %v", err)
	}
	_ = m.Unlock(ctx)
}

func TestBasePathTrimTrailingSlash(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m, err := lockzk.NewMutex(conn, "/locks/trim/", lock.WithValue("v"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	_ = m.Unlock(ctx)
}

func TestEnsurePathNodeExistsRace(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	if _, err := conn.Create("/locks", []byte{}, 0, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Create("/locks/race", []byte{}, 0, nil); err != nil {
		t.Fatal(err)
	}
	conn.hideExists["/locks/race"] = true
	m, _ := lockzk.NewMutex(conn, "/locks/race/deep", lock.WithValue("v"))
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	_ = m.Unlock(ctx)
}

func TestGetWError(t *testing.T) {
	conn := newFakeConn()
	ctx := context.Background()
	m1, _ := lockzk.NewMutex(conn, "/locks/ge", lock.WithValue("1"))
	if err := m1.LockWait(ctx); err != nil {
		t.Fatal(err)
	}
	m2, _ := lockzk.NewMutex(conn, "/locks/ge", lock.WithValue("2"))
	if err := m2.TryLock(ctx); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("TryLock = %v", err)
	}
	conn.getWErr["/locks/ge/lock-0000000001"] = errors.New("getw fail")
	cancelCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()
	if err := m2.LockWait(cancelCtx); err == nil {
		t.Fatal("expected getw error or timeout")
	}
	_ = m1.Unlock(ctx)
}

// Ensure zk errors are referenced.
var _ = zk.ErrNoNode
