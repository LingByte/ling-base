package mysql_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/LingByte/ling-base/lock"
	lockmysql "github.com/LingByte/ling-base/lock/mysql"
)

type fakeRow struct {
	err   error
	valid bool
	val   int64
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) < 1 {
		return errors.New("no dest")
	}
	switch d := dest[0].(type) {
	case *sql.NullInt64:
		d.Valid = r.valid
		d.Int64 = r.val
	default:
		return errors.New("unexpected scan type")
	}
	return nil
}

type fakeDB struct {
	responses map[string]*fakeRow
	lastQuery string
}

func (f *fakeDB) QueryRowContext(_ context.Context, query string, _ ...any) lockmysql.Row {
	f.lastQuery = query
	if r, ok := f.responses[query]; ok {
		return r
	}
	return &fakeRow{}
}

func TestNewMutexValidation(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{}}
	if _, err := lockmysql.NewMutex(nil, "k"); err == nil {
		t.Fatal("nil db")
	}
	if _, err := lockmysql.NewMutex(db, ""); !errors.Is(err, lock.ErrEmptyKey) {
		t.Fatalf("empty key: %v", err)
	}
}

func TestTryLockUnlockRefresh(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT GET_LOCK(?, 0)":    {valid: true, val: 1},
		"SELECT RELEASE_LOCK(?)":   {valid: true, val: 1},
	}}
	ctx := context.Background()
	m, err := lockmysql.NewMutex(db, "my-lock")
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
		"SELECT GET_LOCK(?, 0)": {valid: true, val: 0},
	}}
	m, _ := lockmysql.NewMutex(db, "k")
	if err := m.TryLock(context.Background()); !errors.Is(err, lock.ErrNotObtained) {
		t.Fatalf("TryLock = %v", err)
	}
}

func TestTryLockScanError(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT GET_LOCK(?, 0)": {err: errors.New("scan fail")},
	}}
	m, _ := lockmysql.NewMutex(db, "k")
	if err := m.TryLock(context.Background()); err == nil {
		t.Fatal("expected scan error")
	}
}

func TestUnlockNotHeld(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{}}
	ctx := context.Background()
	m, _ := lockmysql.NewMutex(db, "k")
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
	if err := m.Refresh(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Refresh = %v", err)
	}
}

func TestUnlockReleaseFail(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT GET_LOCK(?, 0)":  {valid: true, val: 1},
		"SELECT RELEASE_LOCK(?)": {valid: true, val: 0},
	}}
	ctx := context.Background()
	m, _ := lockmysql.NewMutex(db, "k")
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); !errors.Is(err, lock.ErrNotHeld) {
		t.Fatalf("Unlock = %v", err)
	}
}

func TestLockSuccessAfterRetry(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{
		"SELECT GET_LOCK(?, 0)": {valid: true, val: 0},
	}}
	ctx := context.Background()
	m, _ := lockmysql.NewMutex(db, "k", lock.WithRetryDelay(5*time.Millisecond))

	go func() {
		time.Sleep(15 * time.Millisecond)
		db.responses["SELECT GET_LOCK(?, 0)"] = &fakeRow{valid: true, val: 1}
		db.responses["SELECT RELEASE_LOCK(?)"] = &fakeRow{valid: true, val: 1}
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
		"SELECT GET_LOCK(?, 0)":  {valid: true, val: 1},
		"SELECT RELEASE_LOCK(?)": {err: errors.New("release scan fail")},
	}}
	ctx := context.Background()
	m, _ := lockmysql.NewMutex(db, "k")
	if err := m.TryLock(ctx); err != nil {
		t.Fatal(err)
	}
	if err := m.Unlock(ctx); err == nil {
		t.Fatal("expected unlock scan error")
	}
}

func TestLockRetryCancel(t *testing.T) {
	call := 0
	db := &fakeDB{responses: map[string]*fakeRow{}}
	orig := db.responses
	db.responses = map[string]*fakeRow{
		"SELECT GET_LOCK(?, 0)": {valid: true, val: 0},
	}
	_ = orig
	ctx := context.Background()
	m, _ := lockmysql.NewMutex(db, "k", lock.WithRetryDelay(5*time.Millisecond))

	// Flip to success after first failure
	go func() {
		time.Sleep(15 * time.Millisecond)
		db.responses["SELECT GET_LOCK(?, 0)"] = &fakeRow{valid: true, val: 1}
		call++
	}()

	cancelCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- m.Lock(cancelCtx) }()
	time.Sleep(8 * time.Millisecond)
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel = %v", err)
	}
}

func TestContextCancelled(t *testing.T) {
	db := &fakeDB{responses: map[string]*fakeRow{}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m, _ := lockmysql.NewMutex(db, "k")
	if err := m.TryLock(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("TryLock = %v", err)
	}
}

func TestStdDBAdapter(t *testing.T) {
	// Compile-time adapter smoke: nil DB is fine for interface check.
	var _ lockmysql.DB = lockmysql.StdDB{}
}
