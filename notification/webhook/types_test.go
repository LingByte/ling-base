// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ──────────────────────────────────────────────
// NoopDeliveryStore
// ──────────────────────────────────────────────

func TestNoopDeliveryStore(t *testing.T) {
	s := NoopDeliveryStore{}
	assert.NoError(t, s.CreateDelivery(DeliveryLog{}))
	assert.NoError(t, s.UpdateDelivery(DeliveryLog{}))
	pending, err := s.GetPendingRetries(10)
	assert.NoError(t, err)
	assert.Nil(t, pending)
	log, err := s.GetDelivery("x")
	assert.NoError(t, err)
	assert.Nil(t, log)
}
