// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// oss StorageStatsProvider implementation.
//
// Bucket storage statistics use the OSS GetBucketStat API which returns
// total storage, object count, and per-storage-class breakdown.
//
// CDN, API request, and origin-fetch statistics require the Alibaba Cloud
// CDN / CloudMonitor APIs which are separate services. They return
// ErrStatsUnsupported here; callers needing those should use the Alibaba
// Cloud SDK directly.

package oss

import (
	"fmt"
	"time"

	"github.com/LingByte/ling-base/stores"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// GetBucketStats returns the current storage usage snapshot for a bucket
// using the OSS GetBucketStat API.
func (s *Store) GetBucketStats(bucket string) (*stores.BucketStats, error) {
	if s.cfg.AccessKeyID == "" || s.cfg.AccessKeySecret == "" || s.cfg.Endpoint == "" {
		return nil, fmt.Errorf("OSS credentials not configured")
	}
	client, err := oss.New(s.cfg.Endpoint, s.cfg.AccessKeyID, s.cfg.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("failed to create OSS client: %v", err)
	}
	bucket = s.resolveBucket(bucket)

	stat, err := client.GetBucketStat(bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket stat: %w", err)
	}

	var classes []stores.StorageClassUsage
	if stat.StandardStorage > 0 || stat.StandardObjectCount > 0 {
		classes = append(classes, stores.StorageClassUsage{
			Class:       "STANDARD",
			Size:        stat.StandardStorage,
			ObjectCount: stat.StandardObjectCount,
		})
	}
	if stat.InfrequentAccessStorage > 0 || stat.InfrequentAccessObjectCount > 0 {
		classes = append(classes, stores.StorageClassUsage{
			Class:       "IA",
			Size:        stat.InfrequentAccessStorage,
			ObjectCount: stat.InfrequentAccessObjectCount,
		})
	}
	if stat.ArchiveStorage > 0 || stat.ArchiveObjectCount > 0 {
		classes = append(classes, stores.StorageClassUsage{
			Class:       "ARCHIVE",
			Size:        stat.ArchiveStorage,
			ObjectCount: stat.ArchiveObjectCount,
		})
	}
	if stat.ColdArchiveStorage > 0 || stat.ColdArchiveObjectCount > 0 {
		classes = append(classes, stores.StorageClassUsage{
			Class:       "COLD_ARCHIVE",
			Size:        stat.ColdArchiveStorage,
			ObjectCount: stat.ColdArchiveObjectCount,
		})
	}

	return &stores.BucketStats{
		Bucket:         bucket,
		Size:           stat.Storage,
		ObjectCount:    stat.ObjectCount,
		UpdatedAt:      time.Now(),
		StorageClasses: classes,
	}, nil
}

// GetCDNStats is not directly supported by the OSS backend. CDN statistics
// require the Alibaba Cloud CDN API (a separate service).
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetAPIRequestStats is not directly supported by the OSS backend. API
// request metrics are available via CloudMonitor (a separate service).
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats is not directly supported by the OSS backend.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}
