// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Package qiniu provides Qiniu content moderation implementations for text,
// image, audio, and video. It depends only on the Qiniu Go SDK.
package qiniu

import (
	"fmt"
	"net/http"

	"github.com/qiniu/go-sdk/v7/auth"
)

// Config holds Qiniu content moderation credentials.
type Config struct {
	AccessKey string
	SecretKey string
	Host      string       // defaults to "ai.qiniuapi.com" if empty
	Client    *http.Client // defaults to http.DefaultClient if nil
}

// newMAC builds the Qiniu auth credentials from the config.
func newMAC(cfg Config) (*auth.Credentials, error) {
	if cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, fmt.Errorf("qiniu requires AccessKey and SecretKey")
	}
	return auth.New(cfg.AccessKey, cfg.SecretKey), nil
}

// defaultHost returns cfg.Host or the Qiniu AI default.
func defaultHost(host string) string {
	if host == "" {
		return "ai.qiniuapi.com"
	}
	return host
}

// defaultClient returns cfg.Client or http.DefaultClient.
func defaultClient(c *http.Client) *http.Client {
	if c == nil {
		return http.DefaultClient
	}
	return c
}
