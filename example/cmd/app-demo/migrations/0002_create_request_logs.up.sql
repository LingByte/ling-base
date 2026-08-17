-- 0002_create_request_logs.up.sql
-- Creates the request_logs table for persisted API request logs.
-- This mirrors the RequestLog GORM model from example/internal/models/models.go.

CREATE TABLE IF NOT EXISTS request_logs (
    id         VARCHAR(22) PRIMARY KEY,
    seq        INTEGER,
    endpoint   VARCHAR(128),
    method     VARCHAR(10),
    status     VARCHAR(20),
    client_ip  VARCHAR(45),
    duration   INTEGER,
    created_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_request_logs_created_at ON request_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_request_logs_endpoint ON request_logs(endpoint);
