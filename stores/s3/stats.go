// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// s3 StorageStatsProvider implementation.
//
// S3 does not expose a single "bucket stat" RPC; GetBucketStats paginates
// ListObjectsV2 to compute total size and object count. CDN, API request,
// and origin-fetch statistics require CloudWatch Metrics which is a separate
// AWS service — callers needing those should use the CloudWatch SDK directly
// or configure a CDN distribution (CloudFront) with its own stats endpoint.
// These dimensions return ErrStatsUnsupported here.

package s3

import (
	"context"
	"time"

	"github.com/LingByte/ling-base/stores"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// GetBucketStats computes the total storage size and object count for a
// bucket by paginating ListObjectsV2. For buckets with many objects this
// can be slow; callers should cache the result.
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

	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, obj := range page.Contents {
			size := aws.ToInt64(obj.Size)
			totalSize += size
			objectCount++
			class := string(obj.StorageClass)
			if class == "" {
				class = "STANDARD"
			}
			if classes == nil {
				classes = make(map[string]*stores.StorageClassUsage)
			}
			if sc, ok := classes[class]; ok {
				sc.Size += size
				sc.ObjectCount++
			} else {
				classes[class] = &stores.StorageClassUsage{Class: class, Size: size, ObjectCount: 1}
			}
		}
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

// GetCDNStats is not directly supported by the S3 backend. S3 does not
// include a CDN; CloudFront statistics require the CloudFront SDK.
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetAPIRequestStats is not directly supported by the S3 backend. API
// request metrics are available via CloudWatch Metrics (a separate AWS
// service).
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats is not directly supported by the S3 backend.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}
