package etcd

import (
	"context"
	"errors"
	"testing"

	clientv3 "go.etcd.io/etcd/client/v3"
)

type fakeLease struct {
	grantErr      error
	revokeErr     error
	keepAliveErr  error
	grantID       clientv3.LeaseID
	lastRevoke    clientv3.LeaseID
	lastKeepAlive clientv3.LeaseID
}

func (f *fakeLease) Grant(context.Context, int64) (*clientv3.LeaseGrantResponse, error) {
	if f.grantErr != nil {
		return nil, f.grantErr
	}
	id := f.grantID
	if id == 0 {
		id = 7
	}
	return &clientv3.LeaseGrantResponse{ID: id}, nil
}

func (f *fakeLease) Revoke(_ context.Context, id clientv3.LeaseID) (*clientv3.LeaseRevokeResponse, error) {
	f.lastRevoke = id
	if f.revokeErr != nil {
		return nil, f.revokeErr
	}
	return &clientv3.LeaseRevokeResponse{}, nil
}

func (f *fakeLease) KeepAliveOnce(_ context.Context, id clientv3.LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
	f.lastKeepAlive = id
	if f.keepAliveErr != nil {
		return nil, f.keepAliveErr
	}
	return &clientv3.LeaseKeepAliveResponse{ID: id}, nil
}

type fakeTxn struct {
	succeeded bool
	err       error
}

func (f *fakeTxn) If(...clientv3.Cmp) clientv3.Txn  { return f }
func (f *fakeTxn) Then(...clientv3.Op) clientv3.Txn { return f }
func (f *fakeTxn) Else(...clientv3.Op) clientv3.Txn { return f }
func (f *fakeTxn) Commit() (*clientv3.TxnResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &clientv3.TxnResponse{Succeeded: f.succeeded}, nil
}

type fakeKV struct {
	txn       *fakeTxn
	deleteErr error
	deleted   string
}

func (f *fakeKV) Txn(context.Context) clientv3.Txn {
	if f.txn == nil {
		f.txn = &fakeTxn{succeeded: true}
	}
	return f.txn
}

func (f *fakeKV) Delete(_ context.Context, key string, _ ...clientv3.OpOption) (*clientv3.DeleteResponse, error) {
	f.deleted = key
	if f.deleteErr != nil {
		return nil, f.deleteErr
	}
	return &clientv3.DeleteResponse{}, nil
}

func TestClientBackend(t *testing.T) {
	lease := &fakeLease{grantID: 42}
	kv := &fakeKV{txn: &fakeTxn{succeeded: true}}
	b := clientBackend{lease: lease, kv: kv}
	ctx := context.Background()

	id, err := b.Grant(ctx, 10)
	if err != nil || id != 42 {
		t.Fatalf("Grant = %d, %v", id, err)
	}
	ok, err := b.Acquire(ctx, "/lock", "tok", id)
	if err != nil || !ok {
		t.Fatalf("Acquire = %v, %v", ok, err)
	}
	if err := b.KeepAliveOnce(ctx, id); err != nil {
		t.Fatal(err)
	}
	if lease.lastKeepAlive != 42 {
		t.Fatalf("keepAlive id = %d", lease.lastKeepAlive)
	}
	if err := b.Delete(ctx, "/lock"); err != nil {
		t.Fatal(err)
	}
	if kv.deleted != "/lock" {
		t.Fatalf("deleted = %q", kv.deleted)
	}
	if err := b.Revoke(ctx, id); err != nil {
		t.Fatal(err)
	}
	if lease.lastRevoke != 42 {
		t.Fatalf("revoke id = %d", lease.lastRevoke)
	}
}

func TestClientBackendErrors(t *testing.T) {
	ctx := context.Background()
	b := clientBackend{
		lease: &fakeLease{grantErr: errors.New("grant")},
		kv:    &fakeKV{},
	}
	if _, err := b.Grant(ctx, 1); err == nil {
		t.Fatal("expected grant error")
	}

	b = clientBackend{
		lease: &fakeLease{},
		kv:    &fakeKV{txn: &fakeTxn{err: errors.New("txn")}},
	}
	if _, err := b.Acquire(ctx, "k", "v", 1); err == nil {
		t.Fatal("expected txn error")
	}

	b = clientBackend{
		lease: &fakeLease{},
		kv:    &fakeKV{txn: &fakeTxn{succeeded: false}},
	}
	ok, err := b.Acquire(ctx, "k", "v", 1)
	if err != nil || ok {
		t.Fatalf("Acquire fail = %v, %v", ok, err)
	}

	b = clientBackend{
		lease: &fakeLease{revokeErr: errors.New("revoke")},
		kv:    &fakeKV{},
	}
	if err := b.Revoke(ctx, 1); err == nil {
		t.Fatal("expected revoke error")
	}

	b = clientBackend{
		lease: &fakeLease{keepAliveErr: errors.New("ka")},
		kv:    &fakeKV{},
	}
	if err := b.KeepAliveOnce(ctx, 1); err == nil {
		t.Fatal("expected keepalive error")
	}

	b = clientBackend{
		lease: &fakeLease{},
		kv:    &fakeKV{deleteErr: errors.New("del")},
	}
	if err := b.Delete(ctx, "k"); err == nil {
		t.Fatal("expected delete error")
	}
}
