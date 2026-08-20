// Package main implements parser-demo: a CLI tool for testing and benchmarking
// the ling-base parser module. It supports single-file parsing, directory
// batch parsing, and detailed metrics analysis.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/LingByte/ling-base/common/parser"
)

const usage = `parser-demo: 文件解析演示与基准工具

用法:
  parser-demo single <file>              解析单个文件，输出文本和指标
  parser-demo single <file> --preview    解析单个文件，仅输出预览（前200字符）
  parser-demo single <file> --fidelity   解析单个文件，并分析内容真实度
  parser-demo dir <directory>            批量解析目录下所有支持的文件
  parser-demo dir <directory> --json     批量解析，输出 JSON 格式报告
  parser-demo dir <directory> --summary  批量解析，仅输出汇总统计
  parser-demo dir <directory> --fidelity 批量解析，并分析每个文件的内容真实度
  parser-demo types                      列出所有支持的文件类型
  parser-demo bench <directory>          批量解析并输出详细性能基准
  parser-demo fidelity <directory>       专门进行内容真实度分析

选项:
  --json       输出 JSON 格式
  --preview    仅显示文本预览
  --summary    仅显示汇总统计
  --fidelity   启用内容真实度分析
  --preserve   保留换行符（不折叠空白）
  --maxlen N   最大文本长度（字节）
  --ocr <driver>  指定 OCR 驱动（aliyun/qcloud/baidu/google/azure/aws）
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	args := os.Args[1:]
	flags := parseFlags(args)
	positional := positionalArgs(args)

	switch positional[0] {
	case "single":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定文件路径")
			os.Exit(1)
		}
		runSingle(positional[1], flags)
	case "dir":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定目录路径")
			os.Exit(1)
		}
		runDirectory(positional[1], flags)
	case "bench":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定目录路径")
			os.Exit(1)
		}
		runBenchmark(positional[1], flags)
	case "fidelity":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定文件或目录路径")
			os.Exit(1)
		}
		info, err := os.Stat(positional[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "错误: %v\n", err)
			os.Exit(1)
		}
		if info.IsDir() {
			runFidelityDirectory(positional[1], flags)
		} else {
			runFidelitySingle(positional[1], flags)
		}
	case "types":
		listSupportedTypes()
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n%s", positional[0], usage)
		os.Exit(1)
	}
}

// --- Flags ---

type demoFlags struct {
	json     bool
	preview  bool
	summary  bool
	fidelity bool
	preserve bool
	maxLen   int
	ocr      string
}

func parseFlags(args []string) demoFlags {
	f := demoFlags{maxLen: 0}
	for _, a := range args {
		switch {
		case a == "--json":
			f.json = true
		case a == "--preview":
			f.preview = true
		case a == "--summary":
			f.summary = true
		case a == "--fidelity":
			f.fidelity = true
		case a == "--preserve":
			f.preserve = true
		case strings.HasPrefix(a, "--maxlen="):
			fmt.Sscanf(a[len("--maxlen="):], "%d", &f.maxLen)
		case strings.HasPrefix(a, "--ocr="):
			f.ocr = a[len("--ocr="):]
		}
	}
	return f
}

func positionalArgs(args []string) []string {
	var out []string
	for _, a := range args {
		if !strings.HasPrefix(a, "--") {
			out = append(out, a)
		}
	}
	return out
}

// --- Metrics ---

// FileMetrics records parsing metrics for a single file.
type FileMetrics struct {
	FileName     string        `json:"fileName"`
	FileType     string        `json:"fileType"`
	FileSize     int64         `json:"fileSize"`
	DetectedType string        `json:"detectedType"`
	Success      bool          `json:"success"`
	Error        string        `json:"error,omitempty"`
	ParseTime    time.Duration `json:"parseTime"`
	TextLength   int           `json:"textLength"`
	RuneCount    int           `json:"runeCount"`
	SectionCount int           `json:"sectionCount"`
	LineCount    int           `json:"lineCount"`
	WordCount    int           `json:"wordCount"`
	TextPreview  string        `json:"textPreview,omitempty"`
}

// SummaryReport aggregates metrics across multiple files.
type SummaryReport struct {
	TotalFiles     int                   `json:"totalFiles"`
	SuccessCount   int                   `json:"successCount"`
	FailureCount   int                   `json:"failureCount"`
	SuccessRate    float64               `json:"successRate"`
	TotalTextLen   int                   `json:"totalTextLength"`
	TotalRunes     int                   `json:"totalRunes"`
	TotalSections  int                   `json:"totalSections"`
	TotalParseTime time.Duration         `json:"totalParseTime"`
	AvgParseTime   time.Duration         `json:"avgParseTime"`
	MaxParseTime   time.Duration         `json:"maxParseTime"`
	MinParseTime   time.Duration         `json:"minParseTime"`
	ByFileType     map[string]*TypeStats `json:"byFileType"`
	Files          []FileMetrics         `json:"files,omitempty"`
	Failures       []FileMetrics         `json:"failures,omitempty"`
}

type TypeStats struct {
	Count        int           `json:"count"`
	SuccessCount int           `json:"successCount"`
	TotalText    int           `json:"totalText"`
	TotalTime    time.Duration `json:"totalTime"`
	AvgTime      time.Duration `json:"avgTime"`
}

func newSummaryReport() *SummaryReport {
	return &SummaryReport{
		ByFileType: make(map[string]*TypeStats),
	}
}

func (s *SummaryReport) add(m FileMetrics) {
	s.TotalFiles++
	s.TotalParseTime += m.ParseTime
	if m.Success {
		s.SuccessCount++
		s.TotalTextLen += m.TextLength
		s.TotalRunes += m.RuneCount
		s.TotalSections += m.SectionCount
	} else {
		s.FailureCount++
		s.Failures = append(s.Failures, m)
	}

	ft := m.DetectedType
	if ft == "" {
		ft = m.FileType
	}
	if ft == "" {
		ft = "unknown"
	}
	ts, ok := s.ByFileType[ft]
	if !ok {
		ts = &TypeStats{}
		s.ByFileType[ft] = ts
	}
	ts.Count++
	ts.TotalTime += m.ParseTime
	if m.Success {
		ts.SuccessCount++
		ts.TotalText += m.TextLength
	}
}

func (s *SummaryReport) finalize() {
	if s.TotalFiles > 0 {
		s.SuccessRate = float64(s.SuccessCount) / float64(s.TotalFiles) * 100
		s.AvgParseTime = s.TotalParseTime / time.Duration(s.TotalFiles)
	}
	s.MinParseTime = time.Duration(1<<63 - 1)
	for _, ts := range s.ByFileType {
		if ts.Count > 0 {
			ts.AvgTime = ts.TotalTime / time.Duration(ts.Count)
		}
	}
	// Compute min/max from files if present
	for _, f := range s.Files {
		if f.ParseTime > s.MaxParseTime {
			s.MaxParseTime = f.ParseTime
		}
		if f.ParseTime < s.MinParseTime {
			s.MinParseTime = f.ParseTime
		}
	}
	if s.MinParseTime == time.Duration(1<<63-1) {
		s.MinParseTime = 0
	}
}

// --- Core parsing logic ---

func buildOptions(f demoFlags) *parser.ParseOptions {
	opts := &parser.ParseOptions{
		PreserveLineBreaks: f.preserve,
	}
	if f.maxLen > 0 {
		opts.MaxTextLength = f.maxLen
	}
	return opts
}

func parseFile(ctx context.Context, path string, opts *parser.ParseOptions) (FileMetrics, *parser.ParseResult) {
	info, err := os.Stat(path)
	if err != nil {
		return FileMetrics{
			FileName: filepath.Base(path),
			Success:  false,
			Error:    err.Error(),
		}, nil
	}

	start := time.Now()
	req := &parser.ParseRequest{
		Path:     path,
		FileName: filepath.Base(path),
	}
	res, perr := parser.ParseAuto(ctx, req, opts)
	elapsed := time.Since(start)

	m := FileMetrics{
		FileName:     filepath.Base(path),
		FileSize:     info.Size(),
		DetectedType: parser.DetectFileType(req),
		ParseTime:    elapsed,
	}

	if perr != nil {
		m.Success = false
		m.Error = perr.Error()
		return m, nil
	}

	m.Success = true
	m.FileType = res.FileType
	m.TextLength = len(res.Text)
	m.RuneCount = len([]rune(res.Text))
	m.SectionCount = len(res.Sections)
	m.LineCount = strings.Count(res.Text, "\n") + 1
	m.WordCount = len(strings.Fields(res.Text))
	if len(res.Text) > 200 {
		m.TextPreview = res.Text[:200] + "..."
	} else {
		m.TextPreview = res.Text
	}

	return m, res
}

// --- Commands ---

func runSingle(path string, f demoFlags) {
	ctx := context.Background()
	opts := buildOptions(f)

	m, res := parseFile(ctx, path, opts)

	if f.json && !f.fidelity {
		out, _ := json.MarshalIndent(m, "", "  ")
		fmt.Println(string(out))
		return
	}

	printFileMetrics(m)

	if !f.summary && res != nil {
		fmt.Println()
		fmt.Println("=== 解析文本 ===")
		if f.preview && len(res.Text) > 200 {
			fmt.Println(res.Text[:200] + "...")
		} else {
			fmt.Println(res.Text)
		}

		if len(res.Sections) > 1 {
			fmt.Println()
			fmt.Printf("=== 段落 (%d) ===\n", len(res.Sections))
			for _, s := range res.Sections {
				preview := s.Text
				if len(preview) > 80 {
					preview = preview[:80] + "..."
				}
				fmt.Printf("  [%d] %s: %s\n", s.Index, s.Title, preview)
			}
		}
	}

	if f.fidelity && res != nil {
		raw, _ := os.ReadFile(path)
		report := analyzeFidelity(path, res, raw)
		if f.json {
			out, _ := json.MarshalIndent(report, "", "  ")
			fmt.Println(string(out))
			return
		}
		fmt.Println()
		printFidelityReport(report)
	}
}

func runDirectory(dir string, f demoFlags) {
	ctx := context.Background()
	opts := buildOptions(f)

	files := discoverFiles(dir)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "目录 %s 中没有找到支持的文件\n", dir)
		os.Exit(1)
	}

	report := newSummaryReport()

	for _, path := range files {
		m, _ := parseFile(ctx, path, opts)
		report.Files = append(report.Files, m)
		report.add(m)

		if !f.summary && !f.json {
			status := "OK"
			if !m.Success {
				status = "FAIL"
			}
			fmt.Printf("[%s] %-40s %s  %10s  %d chars  %d sections\n",
				status, m.FileName, m.DetectedType, m.ParseTime.Round(time.Microsecond), m.TextLength, m.SectionCount)
		}
	}

	report.finalize()

	if f.json {
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(out))
		return
	}

	fmt.Println()
	printSummaryReport(report)
}

func runBenchmark(dir string, f demoFlags) {
	ctx := context.Background()
	opts := buildOptions(f)

	files := discoverFiles(dir)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "目录 %s 中没有找到支持的文件\n", dir)
		os.Exit(1)
	}

	report := newSummaryReport()

	fmt.Printf("基准测试: %d 个文件\n\n", len(files))
	fmt.Printf("%-40s %-10s %-12s %10s %10s %8s %8s\n",
		"文件名", "类型", "状态", "耗时", "字节", "字符", "段落")
	fmt.Println(strings.Repeat("-", 105))

	for _, path := range files {
		m, _ := parseFile(ctx, path, opts)
		report.add(m)

		status := "OK"
		if !m.Success {
			status = "FAIL"
		}
		fmt.Printf("%-40s %-10s %-12s %10s %10d %10d %8d\n",
			truncateStr(m.FileName, 40), m.DetectedType, status,
			m.ParseTime.Round(time.Microsecond), m.FileSize, m.TextLength, m.SectionCount)
	}

	report.finalize()

	fmt.Println(strings.Repeat("-", 105))
	fmt.Println()
	printSummaryReport(report)

	// Per-type breakdown
	fmt.Println()
	fmt.Println("=== 按文件类型统计 ===")
	fmt.Printf("%-12s %6s %8s %12s %12s %12s\n",
		"类型", "文件数", "成功数", "总字符", "总耗时", "平均耗时")
	fmt.Println(strings.Repeat("-", 75))

	types := make([]string, 0, len(report.ByFileType))
	for t := range report.ByFileType {
		types = append(types, t)
	}
	sort.Strings(types)

	for _, t := range types {
		ts := report.ByFileType[t]
		fmt.Printf("%-12s %6d %8d %12d %12s %12s\n",
			t, ts.Count, ts.SuccessCount, ts.TotalText,
			ts.TotalTime.Round(time.Microsecond), ts.AvgTime.Round(time.Microsecond))
	}
}

func listSupportedTypes() {
	formats := parser.SupportedDocumentFormats()
	fmt.Printf("支持的文件类型 (%d):\n\n", len(formats))
	for _, f := range formats {
		fmt.Printf("  %-12s %s\n", f.Extension, f.Description)
	}

	fmt.Println()
	fmt.Println("注意事项:")
	for _, n := range parser.SupportedDocumentNotes() {
		fmt.Printf("  • %s\n", n)
	}
}

// --- Helpers ---

func discoverFiles(dir string) []string {
	var files []string
	supportedExts := buildSupportedExtSet()

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if _, ok := supportedExts[ext]; ok {
			files = append(files, path)
		}
		return nil
	})

	sort.Strings(files)
	return files
}

func buildSupportedExtSet() map[string]bool {
	exts := make(map[string]bool)
	for _, f := range parser.SupportedDocumentFormats() {
		exts[strings.ToLower(f.Extension)] = true
	}
	return exts
}

func printFileMetrics(m FileMetrics) {
	fmt.Println("=== 文件指标 ===")
	fmt.Printf("  文件名:     %s\n", m.FileName)
	fmt.Printf("  文件大小:   %d bytes\n", m.FileSize)
	fmt.Printf("  检测类型:   %s\n", m.DetectedType)
	fmt.Printf("  解析状态:   %s\n", boolStr(m.Success, "成功", "失败"))
	if m.Error != "" {
		fmt.Printf("  错误信息:   %s\n", m.Error)
	}
	if m.Success {
		fmt.Printf("  解析耗时:   %s\n", m.ParseTime.Round(time.Microsecond))
		fmt.Printf("  文本长度:   %d bytes / %d runes\n", m.TextLength, m.RuneCount)
		fmt.Printf("  行数:       %d\n", m.LineCount)
		fmt.Printf("  词数:       %d\n", m.WordCount)
		fmt.Printf("  段落数:     %d\n", m.SectionCount)
	}
}

func printSummaryReport(s *SummaryReport) {
	fmt.Println("=== 汇总报告 ===")
	fmt.Printf("  总文件数:       %d\n", s.TotalFiles)
	fmt.Printf("  成功:           %d\n", s.SuccessCount)
	fmt.Printf("  失败:           %d\n", s.FailureCount)
	fmt.Printf("  成功率:         %.1f%%\n", s.SuccessRate)
	fmt.Printf("  总文本长度:     %d bytes / %d runes\n", s.TotalTextLen, s.TotalRunes)
	fmt.Printf("  总段落数:       %d\n", s.TotalSections)
	fmt.Printf("  总解析耗时:     %s\n", s.TotalParseTime.Round(time.Microsecond))
	fmt.Printf("  平均解析耗时:   %s\n", s.AvgParseTime.Round(time.Microsecond))
	fmt.Printf("  最大解析耗时:   %s\n", s.MaxParseTime.Round(time.Microsecond))
	fmt.Printf("  最小解析耗时:   %s\n", s.MinParseTime.Round(time.Microsecond))

	if len(s.Failures) > 0 {
		fmt.Println()
		fmt.Printf("=== 失败文件 (%d) ===\n", len(s.Failures))
		for _, f := range s.Failures {
			fmt.Printf("  %s: %s\n", f.FileName, f.Error)
		}
	}
}

func boolStr(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// --- Fidelity commands ---

func runFidelitySingle(path string, f demoFlags) {
	ctx := context.Background()
	opts := buildOptions(f)

	_, res := parseFile(ctx, path, opts)
	if res == nil {
		fmt.Fprintln(os.Stderr, "解析失败，无法进行真实度分析")
		os.Exit(1)
	}

	raw, _ := os.ReadFile(path)
	report := analyzeFidelity(path, res, raw)

	if f.json {
		out, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(out))
		return
	}

	printFidelityReport(report)
}

func runFidelityDirectory(dir string, f demoFlags) {
	ctx := context.Background()
	opts := buildOptions(f)

	files := discoverFiles(dir)
	if len(files) == 0 {
		fmt.Fprintf(os.Stderr, "目录 %s 中没有找到支持的文件\n", dir)
		os.Exit(1)
	}

	var reports []*FidelityReport
	var totalScore float64
	var gradeCount = map[string]int{}

	fmt.Printf("内容真实度分析: %d 个文件\n\n", len(files))
	fmt.Printf("%-30s %-8s %-8s %8s %8s %8s %s\n",
		"文件名", "类型", "等级", "得分", "提取率", "关键词", "状态")
	fmt.Println(strings.Repeat("-", 95))

	for _, path := range files {
		_, res := parseFile(ctx, path, opts)
		if res == nil {
			fmt.Printf("%-30s %-8s %-8s %8s %8s %8s %s\n",
				truncateStr(baseName(path), 30), "?", "F", "0.0", "0.00", "0/0", "解析失败")
			continue
		}
		raw, _ := os.ReadFile(path)
		report := analyzeFidelity(path, res, raw)
		reports = append(reports, report)
		totalScore += report.FidelityScore
		gradeCount[report.FidelityGrade]++

		status := "PASS"
		if report.FidelityScore < 70 {
			status = "WARN"
		}
		if report.FidelityScore < 50 {
			status = "FAIL"
		}

		fmt.Printf("%-30s %-8s %-8s %7.1f %8.2f %4d/%-3d %s\n",
			truncateStr(report.FileName, 30), report.FileType, report.FidelityGrade,
			report.FidelityScore, report.ExtractionRatio,
			report.KeyTermsFound, report.KeyTermsExpected, status)
	}

	fmt.Println(strings.Repeat("-", 95))

	// Summary
	avgScore := totalScore / float64(len(reports))
	fmt.Println()
	fmt.Println("=== 真实度汇总 ===")
	fmt.Printf("  文件数:         %d\n", len(reports))
	fmt.Printf("  平均得分:       %.1f\n", avgScore)
	fmt.Printf("  等级分布:       A:%d  B:%d  C:%d  D:%d  F:%d\n",
		gradeCount["A"], gradeCount["B"], gradeCount["C"], gradeCount["D"], gradeCount["F"])

	// Show issues
	var totalIssues int
	for _, r := range reports {
		totalIssues += len(r.Issues)
	}
	if totalIssues > 0 {
		fmt.Printf("  总问题数:       %d\n", totalIssues)
		fmt.Println()
		fmt.Println("=== 问题详情 ===")
		for _, r := range reports {
			if len(r.Issues) > 0 {
				fmt.Printf("  %s (得分 %.1f, 等级 %s):\n", r.FileName, r.FidelityScore, r.FidelityGrade)
				for _, issue := range r.Issues {
					fmt.Printf("    ⚠ %s\n", issue)
				}
			}
		}
	}

	if f.json {
		out, _ := json.MarshalIndent(map[string]any{
			"files":       reports,
			"avgScore":    avgScore,
			"gradeCount":  gradeCount,
			"totalIssues": totalIssues,
		}, "", "  ")
		fmt.Println()
		fmt.Println(string(out))
	}
}

func printFidelityReport(r *FidelityReport) {
	fmt.Println("=== 内容真实度分析 ===")
	fmt.Printf("  文件名:         %s\n", r.FileName)
	fmt.Printf("  文件类型:       %s\n", r.FileType)
	fmt.Printf("  文件大小:       %d bytes\n", r.FileSize)
	fmt.Printf("  提取文本:       %d bytes / %d runes\n", r.TextLength, r.RuneCount)
	fmt.Printf("  提取比率:       %.2f (文本/原始)\n", r.ExtractionRatio)
	fmt.Printf("  真实度得分:     %.1f / 100 (等级: %s)\n", r.FidelityScore, r.FidelityGrade)
	fmt.Printf("  关键词覆盖:     %d / %d\n", r.KeyTermsFound, r.KeyTermsExpected)

	if len(r.KeyTermsMissing) > 0 {
		fmt.Printf("  缺失关键词:     %s\n", strings.Join(r.KeyTermsMissing, ", "))
	}

	fmt.Println()
	fmt.Println("=== 检查项 ===")
	for _, c := range r.Checks {
		status := "✓"
		if !c.Passed {
			status = "✗"
		}
		fmt.Printf("  %s %-25s %5.1f  %s\n", status, c.Name, c.Score, c.Detail)
	}

	if len(r.Issues) > 0 {
		fmt.Println()
		fmt.Printf("=== 问题 (%d) ===\n", len(r.Issues))
		for _, issue := range r.Issues {
			fmt.Printf("  ⚠ %s\n", issue)
		}
	}
}
