// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package models defines data structures for the application.
package models

import (
	"time"

	"github.com/LingByte/ling-base/common/idgen"
)

// RequestLog is the GORM model for persisted API request logs.
type RequestLog struct {
	ID        string    `json:"id" gorm:"primaryKey;size:22"`
	Seq       int       `json:"seq"`
	Endpoint  string    `json:"endpoint" gorm:"size:128"`
	Method    string    `json:"method" gorm:"size:10"`
	Status    string    `json:"status" gorm:"size:20"`
	ClientIP  string    `json:"client_ip" gorm:"size:45"`
	Duration  int64     `json:"duration_ms"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName overrides the default table name.
func (RequestLog) TableName() string { return "request_logs" }

// NewRequestLog creates a new RequestLog with a generated ID.
func NewRequestLog(seq int, method, endpoint, status, clientIP string, duration time.Duration) *RequestLog {
	return &RequestLog{
		ID:        idgen.ShortID(),
		Seq:       seq,
		Endpoint:  endpoint,
		Method:    method,
		Status:    status,
		ClientIP:  clientIP,
		Duration:  duration.Milliseconds(),
		CreatedAt: time.Now(),
	}
}

// HealthStatus represents the application health check result.
type HealthStatus struct {
	Status    string            `json:"status"`
	App       string            `json:"app"`
	Version   string            `json:"version"`
	Profile   string            `json:"profile"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]string `json:"checks"`
	Timestamp time.Time         `json:"timestamp"`
}

// ErrorResponse is a standard JSON error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// SearchRequest is the API request body for the /api/v1/search endpoint.
type SearchRequest struct {
	Keyword string `json:"keyword" binding:"required"`
	Size    int    `json:"size"`
}

// SearchResponse is the API response for the /api/v1/search endpoint.
type SearchResponse struct {
	Keyword string `json:"keyword"`
	Total   uint64 `json:"total"`
	Took    string `json:"took"`
	Hits    []Hit  `json:"hits"`
	Cached  bool   `json:"cached"`
}

// Hit is a single search result in the API response.
type Hit struct {
	ID     string         `json:"id"`
	Score  float64        `json:"score"`
	Fields map[string]any `json:"fields"`
}
