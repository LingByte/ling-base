// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

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
// fakeProvider — a controllable MailProvider for Mailer tests
// ──────────────────────────────────────────────

type fakeProvider struct {
	mu          sync.Mutex
	kind        string
	fail        bool
	sendCount   int
	htmlCalls   int
	textCalls   int
	lastSubject string
	lastBody    string
}

func (f *fakeProvider) Kind() string {
	if f.kind == "" {
		return "fake"
	}
	return f.kind
}

func (f *fakeProvider) SendHTMLWith(to, subject, htmlBody string, vars map[string]any) (string, error) {
	f.mu.Lock()
	f.htmlCalls++
	f.sendCount++
	id := f.htmlCalls
	f.lastSubject = ReplacePlaceholders(subject, vars)
	f.lastBody = ReplacePlaceholders(htmlBody, vars)
	fail := f.fail
	f.mu.Unlock()
	if fail {
		return "", errors.New("fake: send failed")
	}
	return fmt.Sprintf("id-html-%d", id), nil
}

func (f *fakeProvider) SendTextWith(to, subject, textBody string, vars map[string]any) (string, error) {
	f.mu.Lock()
	f.textCalls++
	f.sendCount++
	id := f.textCalls
	f.lastSubject = ReplacePlaceholders(subject, vars)
	f.lastBody = ReplacePlaceholders(textBody, vars)
	fail := f.fail
	f.mu.Unlock()
	if fail {
		return "", errors.New("fake: send failed")
	}
	return fmt.Sprintf("id-text-%d", id), nil
}

// ──────────────────────────────────────────────
// fakeTemplateStore — a controllable TemplateStore for Mailer tests
// ──────────────────────────────────────────────

type fakeTemplateStore struct {
	subject    string
	body       string
	err        error
	lastLocale *string
}

func (s fakeTemplateStore) LoadTemplate(code, locale string) (string, string, error) {
	if s.lastLocale != nil {
		*s.lastLocale = locale
	}
	return s.subject, s.body, s.err
}

// ──────────────────────────────────────────────
// Mailer
// ──────────────────────────────────────────────

func TestMailer_FirstFailsSecondSucceeds(t *testing.T) {
	first := &fakeProvider{kind: "first", fail: true}
	second := &fakeProvider{kind: "second"}
	m := NewMailer([]MailProvider{first, second}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    2,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}))

	err := m.Send(context.Background(), "to@x.com", "subj", "<p>hi</p>")
	require.NoError(t, err)
	assert.Equal(t, 2, first.sendCount) // retried MaxAttempts times
	assert.Equal(t, 1, second.sendCount)
	assert.Equal(t, 1, second.htmlCalls)
}

func TestMailer_AllFail(t *testing.T) {
	first := &fakeProvider{kind: "first", fail: true}
	second := &fakeProvider{kind: "second", fail: true}
	m := NewMailer([]MailProvider{first, second}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    2,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}))

	err := m.Send(context.Background(), "to@x.com", "subj", "<p>hi</p>")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all providers failed")
	assert.Equal(t, 2, first.sendCount)
	assert.Equal(t, 2, second.sendCount)
}

func TestMailer_RoundRobinIndexAdvances(t *testing.T) {
	a := &fakeProvider{kind: "a"}
	b := &fakeProvider{kind: "b"}
	m := NewMailer([]MailProvider{a, b}, WithStartingIndex(0))

	require.NoError(t, m.Send(context.Background(), "to@x.com", "s", "<p>1</p>"))
	// After a success starting at index 0, the next start should be 1.
	require.NoError(t, m.Send(context.Background(), "to@x.com", "s", "<p>2</p>"))
	// And then back to 0.
	require.NoError(t, m.Send(context.Background(), "to@x.com", "s", "<p>3</p>"))

	m.mu.Lock()
	idx := m.currentIndex
	m.mu.Unlock()
	assert.Equal(t, 1, idx) // advanced after the 3rd send (started at 0)

	// a handled sends 1 and 3; b handled send 2.
	assert.Equal(t, 2, a.sendCount)
	assert.Equal(t, 1, b.sendCount)
}

func TestMailer_SendText(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 1 * time.Millisecond,
		MaxBackoff:     5 * time.Millisecond,
	}))
	require.NoError(t, m.SendText(context.Background(), "to@x.com", "subj", "body"))
	assert.Equal(t, 1, ok.textCalls)
	assert.Equal(t, 0, ok.htmlCalls)
	assert.Equal(t, "subj", ok.lastSubject)
	assert.Equal(t, "body", ok.lastBody)
}

func TestMailer_NoProviders(t *testing.T) {
	m := NewMailer(nil)
	err := m.Send(context.Background(), "to@x.com", "subj", "<p>hi</p>")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no providers")
}

func TestMailer_ContextCancelled(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := m.Send(ctx, "to@x.com", "subj", "<p>hi</p>")
	require.Error(t, err)
	assert.Equal(t, 0, ok.sendCount)
}

// ──────────────────────────────────────────────
// SendWithTemplate
// ──────────────────────────────────────────────

func TestMailer_SendWithTemplate(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok})

	store := fakeTemplateStore{
		subject: "Welcome {{.Name}}",
		body:    "<p>Hello {{.Name}}</p>",
	}
	err := m.SendWithTemplate(context.Background(), "to@x.com", "", "welcome", map[string]any{"Name": "Alice"}, store)
	require.NoError(t, err)
	assert.Equal(t, "Welcome Alice", ok.lastSubject)
	assert.Equal(t, "<p>Hello Alice</p>", ok.lastBody)
}

func TestMailer_SendWithTemplate_NilStore(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok})
	err := m.SendWithTemplate(context.Background(), "to@x.com", "", "welcome", nil, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "template store is nil")
}

func TestMailer_SendWithTemplate_LoadError(t *testing.T) {
	ok := &fakeProvider{kind: "ok"}
	m := NewMailer([]MailProvider{ok})
	store := fakeTemplateStore{err: errors.New("not found")}
	err := m.SendWithTemplate(context.Background(), "to@x.com", "", "missing", nil, store)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "load template")
}

func TestSendWithTemplate_LocaleFromVars(t *testing.T) {
	var locale string
	store := &fakeTemplateStore{subject: "Subj", body: "Body {{.Name}}", lastLocale: &locale}
	fp := &fakeProvider{kind: "test"}
	m := NewMailer([]MailProvider{fp})
	err := m.SendWithTemplate(context.Background(), "to@x.com", "", "tpl", map[string]any{"Name": "Bob", "locale": "zh-CN"}, store)
	require.NoError(t, err)
	assert.Equal(t, "zh-CN", locale)
	assert.Equal(t, "Body Bob", fp.lastBody)
}

func TestSendWithTemplate_LocaleNonString(t *testing.T) {
	var locale string
	store := &fakeTemplateStore{subject: "Subj", body: "Body", lastLocale: &locale}
	fp := &fakeProvider{kind: "test"}
	m := NewMailer([]MailProvider{fp})
	err := m.SendWithTemplate(context.Background(), "to@x.com", "", "tpl", map[string]any{"locale": 123}, store)
	require.NoError(t, err)
	assert.Equal(t, "", locale)
}

func TestSendWithTemplate_EmptySubjectUsesTemplate(t *testing.T) {
	store := &fakeTemplateStore{subject: "Template Subject", body: "Body"}
	fp := &fakeProvider{kind: "test"}
	m := NewMailer([]MailProvider{fp})
	err := m.SendWithTemplate(context.Background(), "to@x.com", "", "tpl", nil, store)
	require.NoError(t, err)
	assert.Equal(t, "Template Subject", fp.lastSubject)
}

func TestSendWithTemplate_ProvidedSubjectOverrides(t *testing.T) {
	store := &fakeTemplateStore{subject: "Template Subject", body: "Body"}
	fp := &fakeProvider{kind: "test"}
	m := NewMailer([]MailProvider{fp})
	err := m.SendWithTemplate(context.Background(), "to@x.com", "Custom Subject", "tpl", nil, store)
	require.NoError(t, err)
	assert.Equal(t, "Custom Subject", fp.lastSubject)
}

// ──────────────────────────────────────────────
// retryProvider edge cases
// ──────────────────────────────────────────────

func TestRetryProvider_ContextCancelledDuringBackoff(t *testing.T) {
	fp := &fakeProvider{kind: "test", fail: true}
	m := NewMailer([]MailProvider{fp}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    3,
		InitialBackoff: 10 * time.Second,
		MaxBackoff:     30 * time.Second,
	}))

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel after a short delay to trigger the ctx.Done() branch in backoff.
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := m.Send(ctx, "to@x.com", "subj", "<p>body</p>")
	require.Error(t, err)
}

func TestRetryProvider_ZeroBackoff(t *testing.T) {
	fp := &fakeProvider{kind: "test", fail: true}
	m := NewMailer([]MailProvider{fp}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 0,
		MaxBackoff:     1,
	}))
	err := m.Send(context.Background(), "to@x.com", "subj", "<p>body</p>")
	require.Error(t, err)
}

// ──────────────────────────────────────────────
// nextIndex edge cases
// ──────────────────────────────────────────────

func TestNextIndex_NegativeIndex(t *testing.T) {
	m := NewMailer([]MailProvider{&fakeProvider{kind: "a"}, &fakeProvider{kind: "b"}})
	m.currentIndex = -5
	idx := m.nextIndex()
	assert.Equal(t, 0, idx)
}

func TestNextIndex_ModuloWrap(t *testing.T) {
	m := NewMailer([]MailProvider{&fakeProvider{kind: "a"}, &fakeProvider{kind: "b"}})
	m.currentIndex = 5
	idx := m.nextIndex()
	assert.Equal(t, 1, idx)
}

// ──────────────────────────────────────────────
// send edge cases
// ──────────────────────────────────────────────

func TestSend_AllProvidersFail(t *testing.T) {
	fp1 := &fakeProvider{kind: "a", fail: true}
	fp2 := &fakeProvider{kind: "b", fail: true}
	m := NewMailer([]MailProvider{fp1, fp2}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 1,
		MaxBackoff:     1,
	}))
	err := m.Send(context.Background(), "to@x.com", "subj", "<p>body</p>")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all providers failed")
}

func TestSendText_AllProvidersFail(t *testing.T) {
	fp1 := &fakeProvider{kind: "a", fail: true}
	m := NewMailer([]MailProvider{fp1}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 1,
		MaxBackoff:     1,
	}))
	err := m.SendText(context.Background(), "to@x.com", "subj", "body")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all providers failed")
}

// ──────────────────────────────────────────────
// Concurrent access (race detection)
// ──────────────────────────────────────────────

func TestMailer_ConcurrentSend(t *testing.T) {
	fp := &fakeProvider{kind: "test"}
	m := NewMailer([]MailProvider{fp}, WithRetryPolicy(RetryPolicy{
		MaxAttempts:    1,
		InitialBackoff: 1,
		MaxBackoff:     1,
	}))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = m.Send(context.Background(), "to@x.com", "subj", "<p>body</p>")
		}()
	}
	wg.Wait()
}

// ──────────────────────────────────────────────
// WithRetryPolicy clamping
// ──────────────────────────────────────────────

func TestWithRetryPolicy_ClampsAttempts(t *testing.T) {
	m := NewMailer([]MailProvider{&fakeProvider{}}, WithRetryPolicy(RetryPolicy{MaxAttempts: 0}))
	assert.Equal(t, 1, m.retryPolicy.MaxAttempts)
}

func TestWithStartingIndex(t *testing.T) {
	a := &fakeProvider{kind: "a"}
	b := &fakeProvider{kind: "b"}
	m := NewMailer([]MailProvider{a, b}, WithStartingIndex(1), WithRetryPolicy(RetryPolicy{MaxAttempts: 1, InitialBackoff: time.Millisecond, MaxBackoff: time.Millisecond}))
	require.NoError(t, m.Send(context.Background(), "to@x.com", "s", "<p>1</p>"))
	assert.Equal(t, 0, a.sendCount)
	assert.Equal(t, 1, b.sendCount)
}
