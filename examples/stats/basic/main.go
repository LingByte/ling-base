// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

// Example: 使用内存版 stats 收集网站指标
package main

import (
	"fmt"
	"time"

	"github.com/LingByte/ling-base/common/stats"
	"github.com/LingByte/ling-base/common/stats/memory"
)

func main() {
	// 1. 创建内存版 collector（零依赖）
	c := memory.New()
	defer c.Close()

	// 2. 创建 WebsiteMetrics 便捷层
	wm := stats.NewWebsiteMetrics(c)
	date := time.Now().Format("2006-01-02")

	// 3. 记录基础流量指标
	// PV
	wm.RecordPV(date, "/home")
	wm.RecordPV(date, "/home")
	wm.RecordPV(date, "/about")
	wm.RecordPVTotal(date)
	wm.RecordPVTotal(date)
	wm.RecordPVTotal(date)

	// UV（HyperLogLog 去重，~12KB）
	wm.RecordUV(date, "user-001")
	wm.RecordUV(date, "user-002")
	wm.RecordUV(date, "user-001") // 重复，不计

	// IP
	wm.RecordIP(date, "1.2.3.4")
	wm.RecordIP(date, "1.2.3.4")
	wm.RecordIP(date, "5.6.7.8")

	// VV（会话数）
	wm.RecordVV(date)
	wm.RecordVV(date)

	// 4. 会话指标
	wm.RecordBounce(date) // 一次跳出
	wm.RecordSessionDuration(date, 30)  // 30 秒
	wm.RecordSessionDuration(date, 120) // 2 分钟

	// 5. 用户行为指标
	wm.RecordImpression(date, "banner")
	wm.RecordImpression(date, "banner")
	wm.RecordClick(date, "banner")

	wm.RecordConversion(date, "signup")

	wm.RecordDAU(date, "user-001")
	wm.RecordDAU(date, "user-002")

	wm.RecordMAU("2026-08", "user-001")
	wm.RecordMAU("2026-08", "user-002")

	// 留存（精确 Set，用于交集计算）
	wm.RecordDailyUserSet("2026-08-17", "user-001")
	wm.RecordDailyUserSet("2026-08-17", "user-002")
	wm.RecordDailyUserSet("2026-08-18", "user-001")
	wm.RecordDailyUserSet("2026-08-18", "user-003")

	// 6. 性能指标
	wm.RecordResponseTimeMs(date, 50)
	wm.RecordResponseTimeMs(date, 100)
	wm.RecordResponseTimeMs(date, 200)
	wm.RecordResponseTimeMs(date, 500)

	wm.RecordRequest(date)
	wm.RecordRequest(date)
	wm.RecordRequest(date)
	wm.RecordError(date)

	wm.RecordFirstScreen(date, 800)
	wm.RecordFirstScreen(date, 1200)

	// 7. 查询并输出
	fmt.Println("════════════════════════════════════════")
	fmt.Println("  网站指标统计（内存版）")
	fmt.Println("════════════════════════════════════════")
	fmt.Printf("  PV (/home):     %d\n", wm.GetPV(date, "/home"))
	fmt.Printf("  PV (/about):    %d\n", wm.GetPV(date, "/about"))
	fmt.Printf("  UV (estimated): %d\n", wm.GetUV(date))
	fmt.Printf("  IP (estimated): %d\n", wm.GetIP(date))
	fmt.Printf("  VV:             %d\n", wm.GetVV(date))
	fmt.Printf("  跳出率:          %.1f%%\n", wm.GetBounceRate(date)*100)
	fmt.Printf("  平均停留时长:    %.1f 秒\n", wm.GetAvgSessionDuration(date))
	fmt.Printf("  平均访问深度:    %.1f 页/次\n", wm.GetPagesPerVisit(date))
	fmt.Println("──────────────────────────────────────────")
	fmt.Printf("  CTR (banner):   %.1f%%\n", wm.GetCTR(date, "banner")*100)
	fmt.Printf("  CVR (signup):   %.1f%%\n", wm.GetCVR(date, "signup")*100)
	fmt.Printf("  DAU:            %d\n", wm.GetDAU(date))
	fmt.Printf("  MAU:            %d\n", wm.GetMAU("2026-08"))
	fmt.Printf("  次日留存率:      %.1f%%\n", wm.GetRetention("2026-08-17", "2026-08-18")*100)
	fmt.Println("──────────────────────────────────────────")
	fmt.Printf("  响应时间 P50:   %.1f ms\n", wm.GetResponseTimeP50(date))
	fmt.Printf("  响应时间 P95:   %.1f ms\n", wm.GetResponseTimeP95(date))
	fmt.Printf("  响应时间 P99:   %.1f ms\n", wm.GetResponseTimeP99(date))
	fmt.Printf("  QPS:            %.4f\n", wm.GetQPS(date))
	fmt.Printf("  错误率:          %.1f%%\n", wm.GetErrorRate(date)*100)
	fmt.Printf("  首屏 P50:       %.1f ms\n", wm.GetFirstScreenP50(date))
	fmt.Printf("  首屏 P95:       %.1f ms\n", wm.GetFirstScreenP95(date))
	fmt.Println("════════════════════════════════════════")

	// 8. 直接使用底层原语（自定义指标）
	customCounter := c.Counter("custom:my_metric")
	customCounter.IncrBy(42)
	fmt.Printf("\n  自定义指标: %d\n", customCounter.Get())
}
