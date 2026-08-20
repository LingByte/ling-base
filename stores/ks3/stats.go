// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// ks3 StorageStatsProvider implementation.
//
// Kingsoft KS3 uses an S3-compatible API; GetBucketStats paginates
// ListObjects to compute total size and object count. CDN, API request,
// and origin-fetch statistics require the Kingsoft Cloud CDN / Monitor
// APIs which are separate services. They return ErrStatsUnsupported here.

package ks3

import (
	"fmt"
	"time"

	"github.com/LingByte/ling-base/stores"

	ks3aws "github.com/ks3sdklib/aws-sdk-go/aws"
	ks3s3 "github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// GetBucketStats computes the total storage size and object count for a
// KS3 bucket by paginating ListObjects.
func (s *Store) GetBucketStats(bucket string) (*stores.BucketStats, error) {
	bucket = s.resolveBucket(bucket)
	client := s.client()

	var totalSize int64
	var objectCount int64
	marker := ""

	for {
		input := &ks3s3.ListObjectsInput{
			Bucket:  ks3aws.String(bucket),
			MaxKeys: ks3aws.Long(1000),
			Marker:  ks3aws.String(marker),
		}
		out, err := client.ListObjects(input)
		if err != nil {
			return nil, fmt.Errorf("failed to list KS3 objects: %w", err)
		}

		for _, obj := range out.Contents {
			if obj.Size != nil {
				totalSize += *obj.Size
			}
			objectCount++
		}

		if !ks3aws.ToBoolean(out.IsTruncated) {
			break
		}
		if out.NextMarker != nil {
			marker = ks3aws.ToString(out.NextMarker)
		} else if len(out.Contents) > 0 {
			marker = ks3aws.ToString(out.Contents[len(out.Contents)-1].Key)
		} else {
			break
		}
	}

	return &stores.BucketStats{
		Bucket:      bucket,
		Region:      s.cfg.Region,
		Size:        totalSize,
		ObjectCount: objectCount,
		UpdatedAt:   time.Now(),
		StorageClasses: []stores.StorageClassUsage{
			{Class: "STANDARD", Size: totalSize, ObjectCount: objectCount},
		},
	}, nil
}

// GetCDNStats is not directly supported by the KS3 backend. CDN statistics
// require the Kingsoft Cloud CDN API (a separate service).
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetAPIRequestStats is not directly supported by the KS3 backend.
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats is not directly supported by the KS3 backend.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}
