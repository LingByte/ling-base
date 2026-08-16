// Package main implements search-demo: a CLI tool that demonstrates
// the ling-base/search module with indexing, searching, and analysis.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/LingByte/ling-base/search"
	"github.com/LingByte/ling-base/search/bleve"
)

const usage = `search-demo: 全文搜索引擎演示工具

用法:
  search-demo index <path>           创建索引并写入示例文档
  search-demo search <path> <keyword> 在索引中搜索关键词
  search-demo bench <path>            批量索引并基准测试搜索性能
  search-demo memory                  内存索引演示（无需磁盘）
  search-demo interactive <path>      交互式搜索（输入关键词即搜索）

选项:
  --json      输出 JSON 格式
  --size N    返回结果数（默认10）
  --batch N   批量索引大小（默认100）
  --count N   基准测试文档数（默认1000）
`

var articles = []struct {
	Title   string
	Content string
	Tags    string
	Author  string
}{
	{"Go 语言入门指南", "Go 是一种静态类型、编译型语言，由 Google 开发。它简洁、高效、并发支持好。", "go,programming,backend", "Alice"},
	{"Python 数据科学教程", "Python 是数据科学领域最流行的语言，拥有 NumPy、Pandas、Scikit-learn 等强大库。", "python,data,ai", "Bob"},
	{"Rust 系统编程", "Rust 是一种内存安全的系统编程语言，无需垃圾回收器即可保证内存安全。", "rust,systems,performance", "Charlie"},
	{"JavaScript 异步编程", "JavaScript 的异步编程模型包括 Promise、async/await 和事件循环。", "javascript,async,web", "Diana"},
	{"Java Spring Boot 开发", "Spring Boot 是 Java 生态中最流行的微服务框架，支持快速开发和自动配置。", "java,spring,backend", "Eve"},
	{"C++ 性能优化技巧", "C++ 性能优化涉及内存布局、缓存友好性、内联和编译器优化等技巧。", "cpp,performance,optimization", "Frank"},
	{"TypeScript 类型系统", "TypeScript 为 JavaScript 添加了静态类型检查，支持泛型、条件类型和映射类型。", "typescript,types,web", "Grace"},
	{"Kotlin 协程指南", "Kotlin 协程提供了一种轻量级的并发方案，简化了异步代码的编写。", "kotlin,coroutines,android", "Henry"},
	{"Swift iOS 开发", "Swift 是 Apple 推出的编程语言，用于 iOS、macOS、watchOS 和 tvOS 开发。", "swift,ios,apple", "Ivy"},
	{"Docker 容器化部署", "Docker 是一种容器化技术，可以将应用及其依赖打包成可移植的容器。", "docker,devops,container", "Jack"},
	{"Kubernetes 集群管理", "Kubernetes 是容器编排平台，支持自动扩缩容、服务发现和滚动更新。", "k8s,devops,orchestration", "Karen"},
	{"React 前端开发", "React 是 Facebook 开发的 UI 库，采用组件化和虚拟 DOM 技术。", "react,frontend,web", "Leo"},
	{"Vue.js 渐进式框架", "Vue.js 是一个渐进式 JavaScript 框架，易于上手，支持响应式数据绑定。", "vue,frontend,web", "Mia"},
	{"数据库索引优化", "数据库索引是提升查询性能的关键技术，包括 B-Tree、Hash 和全文索引。", "database,index,optimization", "Nathan"},
	{"Redis 缓存策略", "Redis 是高性能内存数据库，支持字符串、哈希、列表、集合和有序集合。", "redis,cache,nosql", "Olivia"},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	args := os.Args[1:]
	jsonFlag := false
	size := 10
	batchSize := 100
	docCount := 1000

	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonFlag = true
		case "--size":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &size)
				i++
			}
		case "--batch":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &batchSize)
				i++
			}
		case "--count":
			if i+1 < len(args) {
				fmt.Sscanf(args[i+1], "%d", &docCount)
				i++
			}
		default:
			positional = append(positional, args[i])
		}
	}

	if len(positional) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch positional[0] {
	case "index":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定索引路径")
			os.Exit(1)
		}
		runIndex(positional[1], batchSize, jsonFlag)
	case "search":
		if len(positional) < 3 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定索引路径和关键词")
			os.Exit(1)
		}
		runSearch(positional[1], positional[2], size, jsonFlag)
	case "bench":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定索引路径")
			os.Exit(1)
		}
		runBench(positional[1], docCount, batchSize, size, jsonFlag)
	case "memory":
		runMemoryDemo(jsonFlag)
	case "interactive":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定索引路径")
			os.Exit(1)
		}
		runInteractive(positional[1], size)
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n%s", positional[0], usage)
		os.Exit(1)
	}
}

func runIndex(indexPath string, batchSize int, jsonOut bool) {
	ctx := context.Background()

	// Remove old index if exists
	os.RemoveAll(indexPath)

	eng, err := bleve.NewDefault(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建索引失败: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close()

	// Index sample articles
	docs := make([]search.Doc, len(articles))
	for i, a := range articles {
		docs[i] = search.Doc{
			ID:   fmt.Sprintf("article-%d", i+1),
			Type: "article",
			Fields: map[string]any{
				"title":   a.Title,
				"content": a.Content,
				"tags":    a.Tags,
				"author":  a.Author,
			},
		}
	}

	start := time.Now()
	if err := eng.IndexBatch(ctx, docs); err != nil {
		fmt.Fprintf(os.Stderr, "索引失败: %v\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	count, _ := eng.DocCount(ctx)

	if jsonOut {
		out := map[string]any{
			"indexPath": indexPath,
			"docCount":  count,
			"indexTime": elapsed.String(),
			"batchSize": batchSize,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 索引创建完成 ===")
	fmt.Printf("  索引路径:   %s\n", indexPath)
	fmt.Printf("  文档数:     %d\n", count)
	fmt.Printf("  索引耗时:   %s\n", elapsed)
	fmt.Printf("  批量大小:   %d\n", batchSize)

	stats := eng.Stats()
	fmt.Println()
	fmt.Println("=== 引擎统计 ===")
	fmt.Printf("  索引操作:   %d\n", stats["indexOps"])
	fmt.Printf("  批量操作:   %d\n", stats["batchOps"])
}

func runSearch(indexPath, keyword string, size int, jsonOut bool) {
	ctx := context.Background()

	eng, err := bleve.NewDefault(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开索引失败: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close()

	req := search.SearchRequest{
		Keyword:      keyword,
		SearchFields: []string{"title", "content"},
		Size:         size,
		Highlight:    true,
	}

	start := time.Now()
	res, err := eng.Search(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "搜索失败: %v\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	if jsonOut {
		out := map[string]any{
			"keyword":    keyword,
			"total":      res.Total,
			"took":       res.Took.String(),
			"searchTime": elapsed.String(),
			"hits":       res.Hits,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 搜索结果 ===")
	fmt.Printf("  关键词:     %s\n", keyword)
	fmt.Printf("  总命中:     %d\n", res.Total)
	fmt.Printf("  搜索耗时:   %s (bleve: %s)\n", elapsed, res.Took)
	fmt.Printf("  返回数:     %d\n", len(res.Hits))
	fmt.Println()

	for i, hit := range res.Hits {
		fmt.Printf("--- 结果 %d (score: %.4f) ---\n", i+1, hit.Score)
		fmt.Printf("  ID: %s\n", hit.ID)
		if title, ok := hit.Fields["title"].(string); ok {
			fmt.Printf("  标题: %s\n", title)
		}
		if author, ok := hit.Fields["author"].(string); ok {
			fmt.Printf("  作者: %s\n", author)
		}
		if tags, ok := hit.Fields["tags"].(string); ok {
			fmt.Printf("  标签: %s\n", tags)
		}
		if len(hit.Fragments) > 0 {
			for field, frags := range hit.Fragments {
				for _, frag := range frags {
					fmt.Printf("  高亮[%s]: %s\n", field, frag)
				}
			}
		}
		fmt.Println()
	}
}

func runBench(indexPath string, docCount, batchSize, size int, jsonOut bool) {
	ctx := context.Background()
	os.RemoveAll(indexPath)

	eng, err := bleve.NewDefault(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建索引失败: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close()

	// Generate random documents
	rng := rand.New(rand.NewSource(42))
	docs := make([]search.Doc, docCount)
	for i := 0; i < docCount; i++ {
		a := articles[rng.Intn(len(articles))]
		docs[i] = search.Doc{
			ID:   fmt.Sprintf("doc-%04d", i+1),
			Type: "article",
			Fields: map[string]any{
				"title":   fmt.Sprintf("%s #%d", a.Title, i+1),
				"content": a.Content,
				"tags":    a.Tags,
				"author":  a.Author,
				"views":   float64(rng.Intn(10000)),
			},
		}
	}

	// Index benchmark
	fmt.Printf("基准测试: %d 文档, 批量大小 %d\n\n", docCount, batchSize)

	start := time.Now()
	if err := eng.IndexBatch(ctx, docs); err != nil {
		fmt.Fprintf(os.Stderr, "索引失败: %v\n", err)
		os.Exit(1)
	}
	indexTime := time.Since(start)

	count, _ := eng.DocCount(ctx)

	// Search benchmark
	keywords := []string{"Go", "Python", "Rust", "JavaScript", "Docker", "database", "performance", "前端", "后端"}
	searchTimes := make([]time.Duration, len(keywords))
	totalHits := uint64(0)

	for i, kw := range keywords {
		s := time.Now()
		res, err := eng.Search(ctx, search.NewKeywordSearch(kw, size))
		searchTimes[i] = time.Since(s)
		if err == nil {
			totalHits += res.Total
		}
	}

	avgSearch := time.Duration(0)
	maxSearch := time.Duration(0)
	for _, t := range searchTimes {
		avgSearch += t
		if t > maxSearch {
			maxSearch = t
		}
	}
	avgSearch /= time.Duration(len(searchTimes))

	if jsonOut {
		out := map[string]any{
			"docCount":      count,
			"indexTime":     indexTime.String(),
			"avgSearchTime": avgSearch.String(),
			"maxSearchTime": maxSearch.String(),
			"totalHits":     totalHits,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 索引基准 ===")
	fmt.Printf("  文档数:       %d\n", count)
	fmt.Printf("  索引耗时:     %s\n", indexTime)
	fmt.Printf("  吞吐量:       %.0f docs/s\n", float64(count)/indexTime.Seconds())

	fmt.Println()
	fmt.Println("=== 搜索基准 ===")
	fmt.Printf("  搜索关键词:   %d\n", len(keywords))
	fmt.Printf("  平均耗时:     %s\n", avgSearch)
	fmt.Printf("  最大耗时:     %s\n", maxSearch)
	fmt.Printf("  总命中:       %d\n", totalHits)

	fmt.Println()
	fmt.Println("=== 逐关键词详情 ===")
	for i, kw := range keywords {
		fmt.Printf("  %-15s  %s\n", kw, searchTimes[i])
	}

	// Auto-complete demo
	fmt.Println()
	fmt.Println("=== 自动补全 ===")
	suggestions, _ := eng.GetAutoCompleteSuggestions(ctx, "go")
	fmt.Printf("  'go' → %v\n", suggestions)

	// Stats
	stats := eng.Stats()
	fmt.Println()
	fmt.Println("=== 引擎统计 ===")
	fmt.Printf("  索引操作:     %d\n", stats["indexOps"])
	fmt.Printf("  搜索操作:     %d\n", stats["searchOps"])
	fmt.Printf("  批量操作:     %d\n", stats["batchOps"])
}

func runMemoryDemo(jsonOut bool) {
	ctx := context.Background()

	eng, err := bleve.NewMemory()
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建内存索引失败: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close()

	// Index sample docs
	docs := make([]search.Doc, len(articles))
	for i, a := range articles {
		docs[i] = search.Doc{
			ID:   fmt.Sprintf("mem-%d", i+1),
			Type: "article",
			Fields: map[string]any{
				"title":   a.Title,
				"content": a.Content,
				"tags":    a.Tags,
				"author":  a.Author,
			},
		}
	}
	eng.IndexBatch(ctx, docs)

	count, _ := eng.DocCount(ctx)

	if jsonOut {
		out := map[string]any{
			"mode":     "memory",
			"docCount": count,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 内存索引演示 ===")
	fmt.Printf("  模式:       内存索引 (无磁盘持久化)\n")
	fmt.Printf("  文档数:     %d\n", count)
	fmt.Println()

	// Run various search types
	demoSearches := []struct {
		name string
		req  search.SearchRequest
	}{
		{"关键词搜索", search.NewKeywordSearch("Go", 5)},
		{"字段匹配", search.NewMatchSearch("title", "Python", 5)},
		{"短语搜索", search.NewPhraseSearch("content", "内存安全", 5)},
		{"Term过滤", search.NewTermSearch("author", "Alice", 5)},
		{"前缀搜索", search.SearchRequest{Prefixes: []search.ClausePrefix{{Field: "title", Prefix: "Go"}}, Size: 5}},
		{"模糊搜索", search.SearchRequest{Fuzzies: []search.ClauseFuzzy{{Field: "title", Term: "Pyton", Fuzziness: 2}}, Size: 5}},
	}

	for _, ds := range demoSearches {
		res, err := eng.Search(ctx, ds.req)
		if err != nil {
			fmt.Printf("  ✗ %s: %v\n", ds.name, err)
			continue
		}
		fmt.Printf("  %-12s  命中: %d  耗时: %s\n", ds.name, res.Total, res.Took)
		for _, hit := range res.Hits {
			title := ""
			if t, ok := hit.Fields["title"].(string); ok {
				title = t
			}
			fmt.Printf("    → [%s] %s (score: %.3f)\n", hit.ID, title, hit.Score)
		}
		fmt.Println()
	}
}

func runInteractive(indexPath string, size int) {
	ctx := context.Background()

	eng, err := bleve.NewDefault(indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开索引失败: %v\n", err)
		os.Exit(1)
	}
	defer eng.Close()

	count, _ := eng.DocCount(ctx)
	fmt.Printf("交互式搜索 (索引: %s, 文档数: %d)\n", indexPath, count)
	fmt.Println("输入关键词搜索，输入 'quit' 退出，输入 'stats' 查看统计")
	fmt.Println()

	reader := os.Stdin
	buf := make([]byte, 1024)

	for {
		fmt.Print("search> ")
		n, err := reader.Read(buf)
		if err != nil {
			break
		}
		keyword := strings.TrimSpace(string(buf[:n]))
		if keyword == "quit" || keyword == "exit" {
			break
		}
		if keyword == "stats" {
			stats := eng.Stats()
			fmt.Printf("  索引操作: %d  搜索操作: %d\n", stats["indexOps"], stats["searchOps"])
			continue
		}
		if keyword == "" {
			continue
		}

		res, err := eng.Search(ctx, search.NewKeywordSearch(keyword, size))
		if err != nil {
			fmt.Printf("  搜索错误: %v\n", err)
			continue
		}

		fmt.Printf("  命中 %d 条 (耗时 %s):\n", res.Total, res.Took)
		for _, hit := range res.Hits {
			title := hit.ID
			if t, ok := hit.Fields["title"].(string); ok {
				title = t
			}
			fmt.Printf("    [%.3f] %s\n", hit.Score, title)
		}
		fmt.Println()
	}
}
