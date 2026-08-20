// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// obs StorageStatsProvider implementation.
//
// Huawei OBS uses the S3-compatible API; GetBucketStats paginates
// ListObjectsV2 to compute total size and object count. CDN, API request,
// and origin-fetch statistics require the Huawei Cloud CDN / CloudMonitor
// APIs which are separate services. They return ErrStatsUnsupported here.

package obs

import (
	"context"
	"time"

	"github.com/LingByte/ling-base/stores"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// GetBucketStats computes the total storage size and object count for an
// OBS bucket by paginating ListObjectsV2.
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

// GetCDNStats is not directly supported by the OBS backend. CDN statistics
// require the Huawei Cloud CDN API (a separate service).
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetAPIRequestStats is not directly supported by the OBS backend.
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats is not directly supported by the OBS backend.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}
