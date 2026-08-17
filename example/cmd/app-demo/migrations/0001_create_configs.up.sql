-- 0001_create_configs.up.sql
-- Creates the configs table for the DB-backed config store.
-- This mirrors the ConfigItem GORM model from common/config/configs.go.

CREATE TABLE IF NOT EXISTS configs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    key        VARCHAR(128) NOT NULL,
    `desc`     VARCHAR(200),
    autoload   BOOLEAN DEFAULT FALSE,
    public     BOOLEAN DEFAULT FALSE,
    format     VARCHAR(20) DEFAULT 'text',
    value      TEXT,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_configs_key ON configs(key);
CREATE INDEX IF NOT EXISTS idx_configs_autoload ON configs(autoload);
CREATE INDEX IF NOT EXISTS idx_configs_public ON configs(public);
