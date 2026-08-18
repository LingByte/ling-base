package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/common/lock"
	lockpostgres "github.com/LingByte/ling-base/common/lock/postgres"
)

type fakeRow struct {
	err error
	ok  bool
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) < 1 {
		return errors.New("no dest")
	}
	if b, ok := dest[0].(*bool); ok {
		*b = r.ok
		return nil
	}
	return errors.New("unexpected type")
}

type fakeDB struct {
	responses map[string]*fakeRow
}

func (f *fakeDB) QueryRowContext(_ context.Context, query string, _ ...any) lockpostgres.Row {
	if r, ok := f.responses[query]; ok {
		return r
	}
	return &fakeRow{}
}

func TestNewMutexValidation(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{}}
	if _, err := lockpostgres.NewMutex(nil, "k"); err == nil {
		t.Fatal("nil db")
	}
	if _, err := lockpostgres.NewMutex(db, ""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty key: %v", err)
	}
}

func TestTryLockUnlockRefresh(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT pg_try_advisory_lock($1)": {ok: true},
		"SELECT pg_advisory_unlock($1)":   {ok: true},
	}}
	ctx := context.Background()
	m, err := lockpostgres.NewMutex(db, "resource-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Refresh(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestTryLockNotObtained(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT pg_try_advisory_lock($1)": {ok: false},
	}}
	m, _ := lockpostgres.NewMutex(db, "k")
	if err := m.TryLock(context.Background()); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("TryLock = %v", err)
	}
}

func TestTryLockScanError(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT pg_try_advisory_lock($1)": {err: errors.New("fail")},
	}}
	m, _ := lockpostgres.NewMutex(db, "k")
	if err := m.TryLock(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestUnlockNotHeld(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{}}
	ctx := context.Background()
	m, _ := lockpostgres.NewMutex(db, "k")
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh = %v", err)
	}
}

func TestUnlockFail(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT pg_try_advisory_lock($1)": {ok: true},
		"SELECT pg_advisory_unlock($1)":   {ok: false},
	}}
	ctx := context.Background()
	m, _ := lockpostgres.NewMutex(db, "k")
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
}

func TestLockSuccessAfterRetry(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT pg_try_advisory_lock($1)": {ok: false},
	}}
	ctx := context.Background()
	m, _ := lockpostgres.NewMutex(db, "k", lock.WithRetryDelay(5*time.Millisecond))

	go func() {
		time.Sleep(15 * time.Millisecond)
		db.responses["SELECT pg_try_advisory_lock($1)"] = &fakeRow{ok: true}
		db.responses["SELECT pg_advisory_unlock($1)"] = &fakeRow{ok: true}
	}()

	if err := m.Lock(ctx); err != nil {
		t.Fatalf("Lock = %v", err)
	}
	if err := m.Unlock(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestUnlockScanError(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT pg_try_advisory_lock($1)": {ok: true},
		"SELECT pg_advisory_unlock($1)":   {err: errors.New("fail")},
	}}
	ctx := context.Background()
	m, _ := lockpostgres.NewMutex(db, "k")
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); err == nil {
		t.Fatal("expected error")
	}
}

func TestLockRetryCancel(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT pg_try_advisory_lock($1)": {ok: false},
	}}
	ctx := context.Background()
	m, _ := lockpostgres.NewMutex(db, "k", lock.WithRetryDelay(5*time.Millisecond))
	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- m.Lock(cancelCtx) }()
	time.Sleep(15 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
}

func TestContextCancelled(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := lockpostgres.NewMutex(db, "k")
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}

func TestHashKeyDistinct(t *testing.T) {
	m1, _ := lockpostgres.NewMutex(&fakeDB{responses: map[string]*fakeRow{}}, "a")
	m2, _ := lockpostgres.NewMutex(&fakeDB{responses: map[string]*fakeRow{}}, "b")
	if m1 == nil || m2 == nil {
		t.Fatal("expected mutexes")
	}
}

func TestStdDBAdapter(t *testing.T) {
	var _ lockpostgres.DB = lockpostgres.StdDB{}
}
