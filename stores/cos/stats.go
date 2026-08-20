// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// cos StorageStatsProvider implementation.
//
// COS does not expose a single "bucket stat" RPC; GetBucketStats paginates
// the Bucket.Get (ListObjects) API to compute total size and object count.
// CDN, API request, and origin-fetch statistics require the Tencent Cloud
// CDN / CloudMonitor APIs which are separate services. They return
// ErrStatsUnsupported here.

package cos

import (
	"context"
	"fmt"
	"time"

	"github.com/LingByte/ling-base/stores"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// GetBucketStats computes the total storage size and object count for a
// COS bucket by paginating the Bucket.Get API.
func (s *Store) GetBucketStats(bucket string) (*stores.BucketStats, error) {
	ctx := context.Background()
	bucket = s.resolveBucket(bucket)
	c, err := s.clientForBucket(bucket, "")
	if err != nil {
		return nil, err
	}

	var totalSize int64
	var objectCount int64
	var classes map[string]*stores.StorageClassUsage

	marker := ""
	for {
		opt := &cos.BucketGetOptions{Marker: marker, MaxKeys: 1000}
		out, _, err := c.Bucket.Get(ctx, opt)
		if err != nil {
			return nil, fmt.Errorf("COS list objects: %w", err)
		}
		for _, obj := range out.Contents {
			totalSize += obj.Size
			objectCount++
			class := obj.StorageClass
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

// GetCDNStats is not directly supported by the COS backend. CDN statistics
// require the Tencent Cloud CDN API (a separate service).
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetAPIRequestStats is not directly supported by the COS backend. API
// request metrics are available via CloudMonitor (a separate service).
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats is not directly supported by the COS backend.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}
