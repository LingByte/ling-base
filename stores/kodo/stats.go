// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// kodo StorageStatsProvider implementation.
//
// Qiniu Kodo does not expose a single "bucket stat" RPC; GetBucketStats
// paginates the BucketManager.ListFiles API to compute total size and
// object count. CDN, API request, and origin-fetch statistics require
// the Qiniu CDN / CloudMonitor APIs which are separate services. They
// return ErrStatsUnsupported here.

package kodo

import (
	"fmt"
	"time"

	"github.com/LingByte/ling-base/stores"

	"github.com/qiniu/go-sdk/v7/storage"
)

// GetBucketStats computes the total storage size and object count for a
// Kodo bucket by paginating ListFiles.
func (s *Store) GetBucketStats(bucket string) (*stores.BucketStats, error) {
	bucket = s.resolveBucket(bucket)
	cfg := s.makeConfig()
	bm := storage.NewBucketManager(s.mac(), &cfg)

	var totalSize int64
	var objectCount int64
	marker := ""
	limit := 1000

	for {
		entries, _, nextMarker, hasNext, err := bm.ListFiles(bucket, "", "", marker, limit)
		if err != nil {
			return nil, fmt.Errorf("Kodo list files: %w", err)
		}
		for _, item := range entries {
			if item.IsEmpty() {
				continue
			}
			totalSize += item.Fsize
			objectCount++
		}
		if !hasNext {
			break
		}
		marker = nextMarker
	}

	return &stores.BucketStats{
		Bucket:      bucket,
		Size:        totalSize,
		ObjectCount: objectCount,
		UpdatedAt:   time.Now(),
		StorageClasses: []stores.StorageClassUsage{
			{Class: "STANDARD", Size: totalSize, ObjectCount: objectCount},
		},
	}, nil
}

// GetCDNStats is not directly supported by the Kodo backend. CDN statistics
// require the Qiniu CDN API (a separate service).
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetAPIRequestStats is not directly supported by the Kodo backend.
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats is not directly supported by the Kodo backend.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}
