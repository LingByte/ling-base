// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package listeners wires event bus signals to their handlers (DB persistence,
// cache invalidation, etc.). This decouples event producers (HTTP handlers)
// from event consumers — following the same pattern as LingEchoX's
// internal/listeners.
//
// Listeners are registered once at startup via InitXxx functions and then
// react to events published anywhere in the application.
package listeners

import (
	"context"

	"github.com/LingByte/ling-base/eventbus"
	"github.com/LingByte/ling-base/example/internal/models"
	"github.com/LingByte/ling-base/logger"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Signal names — the "contract" between producers and consumers.
const (
	// SigRequestCompleted is emitted when an HTTP request finishes.
	// Payload: *models.RequestLog
	SigRequestCompleted = "request.completed"
)

var (
	requestDB *gorm.DB
)

// InitRequestListeners wires request-related signals to GORM handlers.
// Call this after DB init and before the app starts.
// If db is nil, the listeners are no-ops (env-only mode).
func InitRequestListeners(db *gorm.DB, bus eventbus.Bus) {
	requestDB = db

	bus.Subscribe(SigRequestCompleted, onRequestCompleted)
}

// onRequestCompleted persists the request log to the database.
// This runs asynchronously in the event bus worker pool, so the HTTP
// handler that emitted the signal is never blocked by DB I/O.
//
// We use context.Background() instead of the event's context because the
// original HTTP request context may already be cancelled by the time the
// async worker picks up the event (the response has been sent).
func onRequestCompleted(_ context.Context, e *eventbus.Event) error {
	if requestDB == nil {
		return nil
	}

	rec, ok := e.Payload.(*models.RequestLog)
	if !ok || rec == nil {
		logger.Warn("request.completed: invalid payload type",
			zap.Any("payload", e.Payload),
		)
		return nil
	}

	if err := requestDB.WithContext(context.Background()).Create(rec).Error; err != nil {
		logger.Warn("request.completed: persist failed",
			zap.String("id", rec.ID),
			zap.Int("seq", rec.Seq),
			zap.Error(err),
		)
		return err
	}

	logger.Info("request.completed: persisted",
		zap.String("id", rec.ID),
		zap.Int("seq", rec.Seq),
		zap.String("endpoint", rec.Endpoint),
	)
	return nil
}
