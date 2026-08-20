// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// tos StorageStatsProvider implementation.
//
// Volcengine TOS does not expose a single "bucket stat" RPC; GetBucketStats
// paginates ListObjectsV2 to compute total size and object count. CDN, API
// request, and origin-fetch statistics require the Volcengine CDN /
// CloudMonitor APIs which are separate services. They return
// ErrStatsUnsupported here.

package tos

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/ling-base/stores"

	"github.com/volcengine/ve-tos-golang-sdk/v2/tos"
)

// GetBucketStats computes the total storage size and object count for a
// TOS bucket by paginating ListObjectsV2.
func (s *Store) GetBucketStats(bucket string) (*stores.BucketStats, error) {
	ctx := context.Background()
	client, err := s.client(ctx)
	if err != nil {
		return nil, err
	}
	bucket = s.resolveBucket(bucket)

	var totalSize int64
	var objectCount int64
	var classes map[string]*stores.StorageClassUsage

	marker := ""
	for {
		input := &tos.ListObjectsV2Input{
			Bucket: bucket,
			ListObjectsInput: tos.ListObjectsInput{
				MaxKeys: 1000,
				Marker:  marker,
			},
		}
		out, err := client.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("TOS list objects: %w", err)
		}
		for _, obj := range out.Contents {
			totalSize += obj.Size
			objectCount++
			class := string(obj.StorageClass)
			if class == "" {
				class = "STANDARD"
			}
			if classes == nil {
				classes = make(map[string]*stores.StorageClassUsage)
			}
			if sc, ok := classes[class]; ok {
				sc.Size += obj.Size
				sc.ObjectCount++
			} else {
				classes[class] = &stores.StorageClassUsage{Class: class, Size: obj.Size, ObjectCount: 1}
			}
		}
		if !out.IsTruncated {
			break
		}
		marker = out.NextMarker
	}

	var scList []stores.StorageClassUsage
	for _, sc := range classes {
		scList = append(scList, *sc)
	}

	return &stores.BucketStats{
		Bucket:         bucket,
		Region:         s.cfg.Region,
		Size:           totalSize,
		ObjectCount:    objectCount,
		UpdatedAt:      time.Now(),
		StorageClasses: scList,
	}, nil
}

// GetCDNStats is not directly supported by the TOS backend. CDN statistics
// require the Volcengine CDN API (a separate service).
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetAPIRequestStats is not directly supported by the TOS backend.
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats is not directly supported by the TOS backend.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}
