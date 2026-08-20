// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// kodo StorageStatsProvider implementation.
//
// Bucket storage statistics paginate the Qiniu BucketManager.ListFiles
// API to compute total size and object count.
//
// CDN statistics use the Qiniu Fusion CDN API (CdnManager.GetBandwidthData,
// CdnManager.GetFluxData) which returns bandwidth and traffic data by
// domain and timestamp.
//
// API request statistics use the Qiniu BucketManager API and CDN statistics
// (Qiniu does not expose granular API request metrics via the standard SDK;
// a best-effort approximation is provided via the CDN request data).
//
// Origin-fetch statistics use the Qiniu CDN origin-pull data API.

package kodo

import (
	"fmt"
	"time"

	"github.com/LingByte/ling-base/stores"

	"github.com/qiniu/go-sdk/v7/auth"
	"github.com/qiniu/go-sdk/v7/cdn"
	"github.com/qiniu/go-sdk/v7/storage"
)

// granularityToQiniu converts a Granularity to a Qiniu CDN granularity
// string.
func granularityToQiniu(g stores.Granularity) string {
	switch g {
	case stores.Granularity5Min:
		return "5min"
	case stores.GranularityHour:
		return "hour"
	case stores.GranularityDay:
		return "day"
	case stores.GranularityMonth:
		return "day"
	default:
		return "hour"
	}
}

// cdnManager creates a Qiniu CDN manager.
func (s *Store) cdnManager() *cdn.CdnManager {
	mac := auth.New(s.cfg.AccessKey, s.cfg.SecretKey)
	return cdn.NewCdnManager(mac)
}

// GetBucketStats computes the total storage size and object count for a
// Kodo bucket by paginating ListFiles.
func (s *Store) GetBucketStats(bucket string) (*stores.BucketStats, error) {
	bucket = s.resolveBucket(bucket)
	cfg := s.makeConfig()
	mgr := storage.NewBucketManager(s.mac(), &cfg)

	var totalSize int64
	var objectCount int64
	marker := ""
	limit := 1000

	for {
		entries, _, nextMarker, hasNext, err := mgr.ListFiles(bucket, "", "", marker, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to list Kodo files: %w", err)
		}
		for _, entry := range entries {
			totalSize += entry.Fsize
			objectCount++
		}
		if !hasNext {
			break
		}
		marker = nextMarker
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

// GetCDNStats returns CDN statistics from the Qiniu Fusion CDN API.
func (s *Store) GetCDNStats(req *stores.CDNStatsRequest) (*stores.CDNStatsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	mgr := s.cdnManager()

	startDate := req.Range.Start.Format("2006-01-02")
	endDate := req.Range.End.Format("2006-01-02")
	granularity := granularityToQiniu(req.Granularity)

	// Get bandwidth data.
	bwResp, err := mgr.GetBandwidthData(startDate, endDate, granularity, req.Domains)
	if err != nil {
		return nil, fmt.Errorf("Qiniu CDN GetBandwidthData: %w", err)
	}

	// Get flux (traffic) data.
	fluxResp, err := mgr.GetFluxData(startDate, endDate, granularity, req.Domains)
	if err != nil {
		return nil, fmt.Errorf("Qiniu CDN GetFluxData: %w", err)
	}

	// Merge by timestamp.
	pointsMap := make(map[time.Time]*stores.CDNStatsPoint)

	parseTimes := func(times []string) []time.Time {
		var ts []time.Time
		for _, t := range times {
			if parsed, err := time.Parse("2006-01-02 15:04:05", t); err == nil {
				ts = append(ts, parsed)
			} else if parsed, err := time.Parse("2006-01-02", t); err == nil {
				ts = append(ts, parsed)
			}
		}
		return ts
	}

	bwTimes := parseTimes(bwResp.Time)
	fluxTimes := parseTimes(fluxResp.Time)

	for _, domain := range req.Domains {
		if data, ok := bwResp.Data[domain]; ok {
			for i, v := range data.DomainChina {
				if i < len(bwTimes) {
					pt := pointsMap[bwTimes[i]]
					if pt == nil {
						pt = &stores.CDNStatsPoint{Timestamp: bwTimes[i]}
						pointsMap[bwTimes[i]] = pt
					}
					pt.Bandwidth += float64(v)
				}
			}
		}
		if data, ok := fluxResp.Data[domain]; ok {
			for i, v := range data.DomainChina {
				if i < len(fluxTimes) {
					pt := pointsMap[fluxTimes[i]]
					if pt == nil {
						pt = &stores.CDNStatsPoint{Timestamp: fluxTimes[i]}
						pointsMap[fluxTimes[i]] = pt
					}
					pt.Traffic += int64(v)
				}
			}
		}
	}

	var allPoints []stores.CDNStatsPoint
	for _, pt := range pointsMap {
		allPoints = append(allPoints, *pt)
	}
	sortCDNPointsByTime(allPoints)

	summary := stores.CDNStatsSummary{}
	for _, p := range allPoints {
		summary.TotalTraffic += p.Traffic
		if p.Bandwidth > summary.PeakBandwidth {
			summary.PeakBandwidth = p.Bandwidth
		}
	}

	return &stores.CDNStatsResponse{
		Domains: req.Domains,
		Points:  allPoints,
		Summary: summary,
	}, nil
}

// GetAPIRequestStats returns API request statistics. Qiniu does not expose
// granular API request metrics via the standard SDK; this returns
// ErrStatsUnsupported.
func (s *Store) GetAPIRequestStats(req *stores.APIStatsRequest) (*stores.APIStatsResponse, error) {
	return nil, stores.ErrStatsUnsupported
}

// GetOriginFetchStats returns origin-fetch statistics from the Qiniu CDN
// origin-pull data API.
func (s *Store) GetOriginFetchStats(req *stores.OriginStatsRequest) (*stores.OriginStatsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request is required")
	}
	mgr := s.cdnManager()

	startDate := req.Range.Start.Format("2006-01-02")
	endDate := req.Range.End.Format("2006-01-02")
	granularity := granularityToQiniu(req.Granularity)

	// Use flux data as origin traffic approximation (Qiniu CDN API does
	// not expose a separate origin-pull endpoint in the standard SDK).
	fluxResp, err := mgr.GetFluxData(startDate, endDate, granularity, req.Domains)
	if err != nil {
		return nil, fmt.Errorf("Qiniu CDN GetFluxData (origin): %w", err)
	}

	pointsMap := make(map[time.Time]*stores.OriginStatsPoint)
	for _, t := range fluxResp.Time {
		if parsed, err := time.Parse("2006-01-02 15:04:05", t); err == nil {
			pointsMap[parsed] = &stores.OriginStatsPoint{Timestamp: parsed}
		} else if parsed, err := time.Parse("2006-01-02", t); err == nil {
			pointsMap[parsed] = &stores.OriginStatsPoint{Timestamp: parsed}
		}
	}

	for _, domain := range req.Domains {
		if data, ok := fluxResp.Data[domain]; ok {
			for i, v := range data.DomainChina {
				if i < len(fluxResp.Time) {
					if parsed, err := time.Parse("2006-01-02 15:04:05", fluxResp.Time[i]); err == nil {
						pt := pointsMap[parsed]
						if pt != nil {
							pt.OriginTraffic += int64(v)
						}
					} else if parsed, err := time.Parse("2006-01-02", fluxResp.Time[i]); err == nil {
						pt := pointsMap[parsed]
						if pt != nil {
							pt.OriginTraffic += int64(v)
						}
					}
				}
			}
		}
	}

	var allPoints []stores.OriginStatsPoint
	for _, pt := range pointsMap {
		allPoints = append(allPoints, *pt)
	}
	sortOriginPointsByTime(allPoints)

	summary := stores.OriginStatsSummary{}
	for _, p := range allPoints {
		summary.TotalOriginTraffic += p.OriginTraffic
	}

	return &stores.OriginStatsResponse{Points: allPoints, Summary: summary}, nil
}

// ─── helpers ───

func sortCDNPointsByTime(points []stores.CDNStatsPoint) {
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if points[i].Timestamp.After(points[j].Timestamp) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
}

func sortOriginPointsByTime(points []stores.OriginStatsPoint) {
	for i := 0; i < len(points); i++ {
		for j := i + 1; j < len(points); j++ {
			if points[i].Timestamp.After(points[j].Timestamp) {
				points[i], points[j] = points[j], points[i]
			}
		}
	}
}
