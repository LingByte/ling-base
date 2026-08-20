// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package email

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// mailStatusRank
// ──────────────────────────────────────────────

func TestMailStatusRank(t *testing.T) {
	assert.Equal(t, 1, mailStatusRank(StatusSent))
	assert.Equal(t, 2, mailStatusRank(StatusDelivered))
	assert.Equal(t, 3, mailStatusRank(StatusOpened))
	assert.Equal(t, 4, mailStatusRank(StatusClicked))
	assert.Equal(t, 5, mailStatusRank(StatusUnsubscribed))
	assert.Equal(t, 0, mailStatusRank(StatusUnknown))
	assert.Equal(t, 0, mailStatusRank("nonexistent"))
	assert.Equal(t, 1, mailStatusRank("SENT"))          // case insensitive
	assert.Equal(t, 2, mailStatusRank("  Delivered  ")) // trimmed
}

// ──────────────────────────────────────────────
// isTerminalFailureStatus
// ──────────────────────────────────────────────

func TestIsTerminalFailureStatus(t *testing.T) {
	assert.True(t, isTerminalFailureStatus(StatusFailed))
	assert.True(t, isTerminalFailureStatus(StatusSoftBounce))
	assert.True(t, isTerminalFailureStatus(StatusInvalid))
	assert.True(t, isTerminalFailureStatus(StatusSpam))
	assert.False(t, isTerminalFailureStatus(StatusSent))
	assert.False(t, isTerminalFailureStatus(StatusDelivered))
	assert.False(t, isTerminalFailureStatus(StatusOpened))
	assert.False(t, isTerminalFailureStatus(""))
	assert.True(t, isTerminalFailureStatus("FAILED")) // case insensitive
}

// ──────────────────────────────────────────────
// ResolveMailLogStatusTransition
// ──────────────────────────────────────────────

func TestResolveMailLogStatusTransition_EmptyIncoming(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusSent, "")
	assert.False(t, apply)
	assert.Equal(t, StatusSent, next)
}

func TestResolveMailLogStatusTransition_SameStatus(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusSent, StatusSent)
	assert.False(t, apply)
	assert.Equal(t, StatusSent, next)
}

func TestResolveMailLogStatusTransition_TerminalFailureIncoming(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusDelivered, StatusFailed)
	assert.True(t, apply)
	assert.Equal(t, StatusFailed, next)
}

func TestResolveMailLogStatusTransition_TerminalFailureIncomingBounce(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusOpened, StatusSoftBounce)
	assert.True(t, apply)
	assert.Equal(t, StatusSoftBounce, next)
}

func TestResolveMailLogStatusTransition_TerminalFailureCurrent(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusFailed, StatusDelivered)
	assert.False(t, apply)
	assert.Equal(t, StatusFailed, next)
}

func TestResolveMailLogStatusTransition_HigherRank(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusSent, StatusDelivered)
	assert.True(t, apply)
	assert.Equal(t, StatusDelivered, next)
}

func TestResolveMailLogStatusTransition_LowerRank(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusOpened, StatusSent)
	assert.False(t, apply)
	assert.Equal(t, StatusOpened, next)
}

func TestResolveMailLogStatusTransition_EqualRank(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition(StatusSent, StatusSent)
	assert.False(t, apply)
	assert.Equal(t, StatusSent, next)
}

func TestResolveMailLogStatusTransition_CaseInsensitive(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition("sent", "DELIVERED")
	assert.True(t, apply)
	assert.Equal(t, "DELIVERED", next)
}

func TestResolveMailLogStatusTransition_WhitespaceTrimmed(t *testing.T) {
	next, apply := ResolveMailLogStatusTransition("  sent  ", "  delivered  ")
	assert.True(t, apply)
	assert.Equal(t, "delivered", next) // incoming is trimmed inside the function
}

// ──────────────────────────────────────────────
// InitialMailStatus
// ──────────────────────────────────────────────

func TestInitialMailStatus(t *testing.T) {
	assert.Equal(t, StatusDelivered, InitialMailStatus("smtp"))
	assert.Equal(t, StatusSent, InitialMailStatus("sendcloud"))
	assert.Equal(t, StatusSent, InitialMailStatus("unknown"))
	assert.Equal(t, StatusSent, InitialMailStatus(""))
}

// ──────────────────────────────────────────────
// MemoryMailLogStore — CreateMailLog
// ──────────────────────────────────────────────

func TestMemoryMailLogStore_CreateMailLog(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{
		MessageID: "msg-1",
		Provider:  "sendcloud",
		ToEmail:   "to@x.com",
		Subject:   "Test",
		Status:    StatusSent,
	}
	err := store.CreateMailLog(log)
	require.NoError(t, err)
	assert.False(t, log.CreatedAt.IsZero())
	assert.False(t, log.UpdatedAt.IsZero())
}

func TestMemoryMailLogStore_CreateMailLog_Nil(t *testing.T) {
	store := NewMemoryMailLogStore()
	err := store.CreateMailLog(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestMemoryMailLogStore_CreateMailLog_EmptyMessageID(t *testing.T) {
	store := NewMemoryMailLogStore()
	err := store.CreateMailLog(&MailLog{Provider: "smtp"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message ID")
}

func TestMemoryMailLogStore_CreateMailLog_PreservesCreatedAt(t *testing.T) {
	store := NewMemoryMailLogStore()
	ts := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	log := &MailLog{
		MessageID: "msg-ts",
		CreatedAt: ts,
	}
	err := store.CreateMailLog(log)
	require.NoError(t, err)
	assert.Equal(t, ts, log.CreatedAt)
}

// ──────────────────────────────────────────────
// MemoryMailLogStore — CreateFailedMailLog
// ──────────────────────────────────────────────

func TestMemoryMailLogStore_CreateFailedMailLog(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{
		MessageID: "fail-1",
		Provider:  "smtp",
		ToEmail:   "to@x.com",
		ErrorMsg:  "connection refused",
	}
	err := store.CreateFailedMailLog(log)
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, log.Status)
}

func TestMemoryMailLogStore_CreateFailedMailLog_Nil(t *testing.T) {
	store := NewMemoryMailLogStore()
	err := store.CreateFailedMailLog(nil)
	require.Error(t, err)
}

func TestMemoryMailLogStore_CreateFailedMailLog_EmptyMessageID(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{Provider: "smtp", ErrorMsg: "fail"}
	err := store.CreateFailedMailLog(log)
	require.NoError(t, err)
	assert.NotEmpty(t, log.MessageID)
	assert.Contains(t, log.MessageID, "failed-")
}

// ──────────────────────────────────────────────
// MemoryMailLogStore — UpdateMailLogStatusByMessageID
// ──────────────────────────────────────────────

func TestMemoryMailLogStore_UpdateStatus_Success(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{MessageID: "upd-1", Provider: "sendcloud", Status: StatusSent}
	require.NoError(t, store.CreateMailLog(log))

	err := store.UpdateMailLogStatusByMessageID("upd-1", "sendcloud", StatusDelivered, "")
	require.NoError(t, err)

	updated, err := store.GetMailLogByMessageID("upd-1")
	require.NoError(t, err)
	assert.Equal(t, StatusDelivered, updated.Status)
}

func TestMemoryMailLogStore_UpdateStatus_EmptyMessageID(t *testing.T) {
	store := NewMemoryMailLogStore()
	err := store.UpdateMailLogStatusByMessageID("", "sendcloud", StatusDelivered, "")
	require.NoError(t, err) // no-op
}

func TestMemoryMailLogStore_UpdateStatus_NotFound(t *testing.T) {
	store := NewMemoryMailLogStore()
	err := store.UpdateMailLogStatusByMessageID("nonexistent", "sendcloud", StatusDelivered, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemoryMailLogStore_UpdateStatus_ProviderMismatch(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{MessageID: "mismatch-1", Provider: "sendcloud", Status: StatusSent}
	require.NoError(t, store.CreateMailLog(log))

	err := store.UpdateMailLogStatusByMessageID("mismatch-1", "smtp", StatusDelivered, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider mismatch")
}

func TestMemoryMailLogStore_UpdateStatus_ProviderEmpty_SkipsCheck(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{MessageID: "no-check-1", Provider: "sendcloud", Status: StatusSent}
	require.NoError(t, store.CreateMailLog(log))

	err := store.UpdateMailLogStatusByMessageID("no-check-1", "", StatusDelivered, "")
	require.NoError(t, err)
}

func TestMemoryMailLogStore_UpdateStatus_NoTransition(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{MessageID: "no-trans-1", Provider: "sendcloud", Status: StatusOpened}
	require.NoError(t, store.CreateMailLog(log))

	// Opened → Sent is a downgrade, should not apply.
	err := store.UpdateMailLogStatusByMessageID("no-trans-1", "sendcloud", StatusSent, "")
	require.NoError(t, err)

	updated, _ := store.GetMailLogByMessageID("no-trans-1")
	assert.Equal(t, StatusOpened, updated.Status)
}

func TestMemoryMailLogStore_UpdateStatus_WithErrorMessage(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{MessageID: "err-1", Provider: "sendcloud", Status: StatusSent}
	require.NoError(t, store.CreateMailLog(log))

	err := store.UpdateMailLogStatusByMessageID("err-1", "sendcloud", StatusFailed, "bounce")
	require.NoError(t, err)

	updated, _ := store.GetMailLogByMessageID("err-1")
	assert.Equal(t, StatusFailed, updated.Status)
	assert.Equal(t, "bounce", updated.ErrorMsg)
}

// ──────────────────────────────────────────────
// MemoryMailLogStore — GetMailLogByMessageID
// ──────────────────────────────────────────────

func TestMemoryMailLogStore_GetByMessageID_Success(t *testing.T) {
	store := NewMemoryMailLogStore()
	log := &MailLog{MessageID: "get-1", Provider: "smtp", Subject: "Hello"}
	require.NoError(t, store.CreateMailLog(log))

	result, err := store.GetMailLogByMessageID("get-1")
	require.NoError(t, err)
	assert.Equal(t, "Hello", result.Subject)
}

func TestMemoryMailLogStore_GetByMessageID_NotFound(t *testing.T) {
	store := NewMemoryMailLogStore()
	_, err := store.GetMailLogByMessageID("nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// ──────────────────────────────────────────────
// MemoryMailLogStore — GetMailLogs (pagination)
// ──────────────────────────────────────────────

func TestMemoryMailLogStore_GetMailLogs(t *testing.T) {
	store := NewMemoryMailLogStore()
	for i := 0; i < 5; i++ {
		require.NoError(t, store.CreateMailLog(&MailLog{
			MessageID: fmt.Sprintf("page-%d", i),
			Subject:   fmt.Sprintf("Subject %d", i),
			Status:    StatusSent,
		}))
	}

	logs, total, err := store.GetMailLogs(1, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 3)
	// Most recent first — last inserted is page-4.
	assert.Equal(t, "page-4", logs[0].MessageID)
}

func TestMemoryMailLogStore_GetMailLogs_PageBeyondRange(t *testing.T) {
	store := NewMemoryMailLogStore()
	require.NoError(t, store.CreateMailLog(&MailLog{MessageID: "only-1", Status: StatusSent}))

	logs, total, err := store.GetMailLogs(5, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Empty(t, logs)
}

func TestMemoryMailLogStore_GetMailLogs_Empty(t *testing.T) {
	store := NewMemoryMailLogStore()
	logs, total, err := store.GetMailLogs(1, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.Empty(t, logs)
}

func TestMemoryMailLogStore_GetMailLogs_DefaultPageSize(t *testing.T) {
	store := NewMemoryMailLogStore()
	require.NoError(t, store.CreateMailLog(&MailLog{MessageID: "def-1", Status: StatusSent}))

	// page=0 and pageSize=0 should use defaults.
	logs, total, err := store.GetMailLogs(0, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, logs, 1)
}

func TestMemoryMailLogStore_GetMailLogs_SecondPage(t *testing.T) {
	store := NewMemoryMailLogStore()
	for i := 0; i < 5; i++ {
		require.NoError(t, store.CreateMailLog(&MailLog{
			MessageID: fmt.Sprintf("p2-%d", i),
			Status:    StatusSent,
		}))
	}

	logs, total, err := store.GetMailLogs(2, 3)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Len(t, logs, 2) // 5 total, 3 on page 1, 2 on page 2
}

// ──────────────────────────────────────────────
// MemoryMailLogStore — GetMailLogStats
// ──────────────────────────────────────────────

func TestMemoryMailLogStore_GetMailLogStats(t *testing.T) {
	store := NewMemoryMailLogStore()
	require.NoError(t, store.CreateMailLog(&MailLog{MessageID: "stat-1", Status: StatusSent}))
	require.NoError(t, store.CreateMailLog(&MailLog{MessageID: "stat-2", Status: StatusSent}))
	require.NoError(t, store.CreateMailLog(&MailLog{MessageID: "stat-3", Status: StatusDelivered}))
	require.NoError(t, store.CreateFailedMailLog(&MailLog{MessageID: "stat-4", Provider: "smtp"}))

	stats, err := store.GetMailLogStats()
	require.NoError(t, err)
	assert.Equal(t, int64(4), stats["total"])
	assert.Equal(t, int64(2), stats[StatusSent])
	assert.Equal(t, int64(1), stats[StatusDelivered])
	assert.Equal(t, int64(1), stats[StatusFailed])
}

func TestMemoryMailLogStore_GetMailLogStats_Empty(t *testing.T) {
	store := NewMemoryMailLogStore()
	stats, err := store.GetMailLogStats()
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats["total"])
}

// ──────────────────────────────────────────────
// Concurrent access (race detection)
// ──────────────────────────────────────────────

func TestMemoryMailLogStore_Concurrent(t *testing.T) {
	store := NewMemoryMailLogStore()
	done := make(chan struct{})
	// Writers.
	for i := 0; i < 5; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 20; j++ {
				_ = store.CreateMailLog(&MailLog{
					MessageID: fmt.Sprintf("race-%d-%d", n, j),
					Status:    StatusSent,
				})
			}
		}(i)
	}
	// Readers.
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 20; j++ {
				_, _, _ = store.GetMailLogs(1, 10)
				_, _ = store.GetMailLogStats()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
