// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package notification

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Message constructors
// ──────────────────────────────────────────────

func TestNewEmailMessage(t *testing.T) {
	msg := NewEmailMessage("a@b.com", "Hello", "Body")
	assert.Equal(t, TypeEmail, msg.Type)
	assert.Equal(t, "a@b.com", msg.To)
	assert.Equal(t, "Hello", msg.Subject)
	assert.Equal(t, "Body", msg.Body)
}

func TestNewSMSMessage(t *testing.T) {
	msg := NewSMSMessage("13800138000", "Code: 1234")
	assert.Equal(t, TypeSMS, msg.Type)
	assert.Equal(t, "13800138000", msg.PhoneNumber)
	assert.Equal(t, "Code: 1234", msg.Content)
}

func TestNewIMMessage(t *testing.T) {
	msg := NewIMMessage("Alert", "CPU > 90%")
	assert.Equal(t, TypeIM, msg.Type)
	assert.Equal(t, "Alert", msg.Title)
	assert.Equal(t, "CPU > 90%", msg.Content)
}

func TestNewWebhookMessage(t *testing.T) {
	msg := NewWebhookMessage("user.created", "https://example.com/hook", map[string]any{"id": 1})
	assert.Equal(t, TypeWebhook, msg.Type)
	assert.Equal(t, "user.created", msg.Event)
	assert.Equal(t, "https://example.com/hook", msg.URL)
	assert.Equal(t, 1, msg.Data["id"])
}

func TestNewInboxMessage(t *testing.T) {
	msg := NewInboxMessage("user-123", "Welcome", "Welcome to the platform!")
	assert.Equal(t, TypeInbox, msg.Type)
	assert.Equal(t, "user-123", msg.UserID)
	assert.Equal(t, "Welcome", msg.Title)
}

// ──────────────────────────────────────────────
// ChannelFunc
// ──────────────────────────────────────────────

func TestChannelFunc(t *testing.T) {
	called := false
	ch := NewChannelFunc("test-ch", TypeEmail, true, func(ctx context.Context, msg Message) error {
		called = true
		assert.Equal(t, "test@test.com", msg.To)
		return nil
	})

	assert.Equal(t, "test-ch", ch.Name())
	assert.Equal(t, TypeEmail, ch.Type())
	assert.True(t, ch.Enabled())

	err := ch.Send(context.Background(), NewEmailMessage("test@test.com", "S", "B"))
	require.NoError(t, err)
	assert.True(t, called)
}

func TestChannelFunc_NilSendFunc(t *testing.T) {
	ch := NewChannelFunc("test-ch", TypeEmail, true, nil)
	err := ch.Send(context.Background(), Message{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no send function")
}

func TestChannelFunc_SetEnabled(t *testing.T) {
	ch := NewChannelFunc("test-ch", TypeEmail, true, func(ctx context.Context, msg Message) error { return nil })
	assert.True(t, ch.Enabled())
	ch.SetEnabled(false)
	assert.False(t, ch.Enabled())
	ch.SetEnabled(true)
	assert.True(t, ch.Enabled())
}

func TestChannelFunc_Disabled(t *testing.T) {
	ch := NewChannelFunc("test-ch", TypeEmail, false, func(ctx context.Context, msg Message) error { return nil })
	assert.False(t, ch.Enabled())
}

// ──────────────────────────────────────────────
// BaseChannel
// ──────────────────────────────────────────────

func TestBaseChannel(t *testing.T) {
	bc := &BaseChannel{ChannelName: "base", ChannelType: TypeSMS, IsEnabled: true}
	assert.Equal(t, "base", bc.Name())
	assert.Equal(t, TypeSMS, bc.Type())
	assert.True(t, bc.Enabled())

	bc.IsEnabled = false
	assert.False(t, bc.Enabled())
}

// ──────────────────────────────────────────────
// Dispatcher
// ──────────────────────────────────────────────

func TestDispatcher_AddRemoveChannels(t *testing.T) {
	d := NewDispatcher()
	ch1 := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error { return nil })
	ch2 := NewChannelFunc("ch2", TypeSMS, true, func(ctx context.Context, msg Message) error { return nil })

	d.AddChannel(ch1)
	d.AddChannel(ch2)

	assert.Equal(t, []string{"ch1", "ch2"}, d.ChannelNames())
	assert.Len(t, d.Channels(), 2)

	assert.True(t, d.RemoveChannel("ch1"))
	assert.Equal(t, []string{"ch2"}, d.ChannelNames())

	assert.False(t, d.RemoveChannel("nonexistent"))
}

func TestDispatcher_Send_Success(t *testing.T) {
	d := NewDispatcher()
	called := false
	ch := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error {
		called = true
		return nil
	})
	d.AddChannel(ch)

	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	require.NoError(t, err)
	assert.True(t, called)
}

func TestDispatcher_Send_Failover(t *testing.T) {
	d := NewDispatcher()
	attempts := []string{}
	ch1 := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error {
		attempts = append(attempts, "ch1")
		return errors.New("ch1 failed")
	})
	ch2 := NewChannelFunc("ch2", TypeEmail, true, func(ctx context.Context, msg Message) error {
		attempts = append(attempts, "ch2")
		return nil
	})
	d.AddChannel(ch1)
	d.AddChannel(ch2)

	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	require.NoError(t, err)
	assert.Equal(t, []string{"ch1", "ch2"}, attempts)
}

func TestDispatcher_Send_AllFail(t *testing.T) {
	d := NewDispatcher()
	ch1 := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error {
		return errors.New("ch1 failed")
	})
	ch2 := NewChannelFunc("ch2", TypeEmail, true, func(ctx context.Context, msg Message) error {
		return errors.New("ch2 failed")
	})
	d.AddChannel(ch1)
	d.AddChannel(ch2)

	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "all channels failed")
	assert.Contains(t, err.Error(), "ch1")
	assert.Contains(t, err.Error(), "ch2")
}

func TestDispatcher_Send_NoChannels(t *testing.T) {
	d := NewDispatcher()
	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no enabled channels")
}

func TestDispatcher_Send_NoEnabledChannels(t *testing.T) {
	d := NewDispatcher()
	ch := NewChannelFunc("ch1", TypeEmail, false, func(ctx context.Context, msg Message) error { return nil })
	d.AddChannel(ch)

	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no enabled channels")
}

func TestDispatcher_Send_TypeFiltering(t *testing.T) {
	d := NewDispatcher()
	emailCalled := false
	smsCalled := false
	emailCh := NewChannelFunc("email", TypeEmail, true, func(ctx context.Context, msg Message) error {
		emailCalled = true
		return nil
	})
	smsCh := NewChannelFunc("sms", TypeSMS, true, func(ctx context.Context, msg Message) error {
		smsCalled = true
		return nil
	})
	d.AddChannel(emailCh)
	d.AddChannel(smsCh)

	err := d.Send(context.Background(), NewSMSMessage("123", "msg"))
	require.NoError(t, err)
	assert.False(t, emailCalled)
	assert.True(t, smsCalled)
}

func TestDispatcher_SendToType(t *testing.T) {
	d := NewDispatcher()
	ch := NewChannelFunc("ch1", TypeIM, true, func(ctx context.Context, msg Message) error {
		assert.Equal(t, "Title", msg.Subject)
		assert.Equal(t, "Body", msg.Body)
		return nil
	})
	d.AddChannel(ch)

	err := d.SendToType(context.Background(), TypeIM, "user1", "Title", "Body")
	require.NoError(t, err)
}

func TestDispatcher_EnabledChannelCount(t *testing.T) {
	d := NewDispatcher()
	d.AddChannel(NewChannelFunc("ch1", TypeEmail, true, nil))
	d.AddChannel(NewChannelFunc("ch2", TypeEmail, true, nil))
	d.AddChannel(NewChannelFunc("ch3", TypeEmail, false, nil))
	d.AddChannel(NewChannelFunc("ch4", TypeSMS, true, nil))

	assert.Equal(t, 2, d.EnabledChannelCount(TypeEmail))
	assert.Equal(t, 1, d.EnabledChannelCount(TypeSMS))
	assert.Equal(t, 0, d.EnabledChannelCount(TypeWebhook))
}

func TestDispatcher_SetLogStore(t *testing.T) {
	d := NewDispatcher()
	store := &mockLogStore{}
	d.SetLogStore(store)

	ch := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error { return nil })
	d.AddChannel(ch)

	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	require.NoError(t, err)

	assert.Len(t, store.created, 1)
	assert.Len(t, store.updated, 1)
	assert.Equal(t, "sent", store.updated[0].status)
}

func TestDispatcher_Send_WithLogStore_Failure(t *testing.T) {
	d := NewDispatcherWithStore(&mockLogStore{})
	ch := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error {
		return errors.New("send failed")
	})
	d.AddChannel(ch)

	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	assert.Error(t, err)
}

func TestNewDispatcherWithStore_Nil(t *testing.T) {
	d := NewDispatcherWithStore(nil)
	// Should use NoopLogStore, not panic.
	ch := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error { return nil })
	d.AddChannel(ch)
	err := d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
	require.NoError(t, err)
}

// ──────────────────────────────────────────────
// NoopLogStore
// ──────────────────────────────────────────────

func TestNoopLogStore(t *testing.T) {
	s := NoopLogStore{}
	assert.NoError(t, s.CreateLog(LogEntry{}))
	assert.NoError(t, s.UpdateStatus("id", "sent", ""))
	log, err := s.GetLog("id")
	assert.NoError(t, err)
	assert.Nil(t, log)
}

// ──────────────────────────────────────────────
// Concurrency
// ──────────────────────────────────────────────

func TestDispatcher_ConcurrentSend(t *testing.T) {
	d := NewDispatcher()
	var mu sync.Mutex
	count := 0
	ch := NewChannelFunc("ch1", TypeEmail, true, func(ctx context.Context, msg Message) error {
		mu.Lock()
		count++
		mu.Unlock()
		return nil
	})
	d.AddChannel(ch)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = d.Send(context.Background(), NewEmailMessage("a@b.com", "S", "B"))
		}()
	}
	wg.Wait()

	assert.Equal(t, 50, count)
}

// ──────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────

type mockLogStore struct {
	mu      sync.Mutex
	created []LogEntry
	updated []mockLogUpdate
}

type mockLogUpdate struct {
	id       string
	status   string
	errorMsg string
}

func (m *mockLogStore) CreateLog(entry LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.created = append(m.created, entry)
	return nil
}

func (m *mockLogStore) UpdateStatus(id, status, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updated = append(m.updated, mockLogUpdate{id: id, status: status, errorMsg: errorMsg})
	return nil
}

func (m *mockLogStore) GetLog(id string) (*LogEntry, error) {
	return nil, nil
}

// Ensure fmt and time are used.
var _ = fmt.Sprintf
var _ = time.Now
