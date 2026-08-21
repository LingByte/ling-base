//
// Tencent is pleased to support the open source community by making trpc-agent-go available.
//
// Copyright (C) 2025 Tencent.  All rights reserved.
//
// trpc-agent-go is licensed under the Apache License Version 2.0.
//
//

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/LingByte/ling-base/agentkit/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTrackEventsByTrackLists_EmptyAndMismatch(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	s := createTestService(t, db)
	got, err := s.getTrackEventsByTrackLists(context.Background(), nil, nil, 0, time.Time{})
	require.NoError(t, err)
	assert.Nil(t, got)
	_, err = s.getTrackEventsByTrackLists(
		context.Background(),
		[]session.Key{{AppName: "app", UserID: "user", SessionID: "sess"}},
		nil,
		0,
		time.Time{},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "track lists count mismatch")
}
