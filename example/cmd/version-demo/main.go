// Package main implements version-demo: a CLI tool that demonstrates
// the ling-base/version module.
package main

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/LingByte/ling-base/version"
)

func main() {
	jsonFlag := flag.Bool("json", false, "output as JSON")
	infoFlag := flag.Bool("info", false, "show full version info")
	setFlag := flag.String("set", "", "set version (demo only, in-memory)")
	flag.Parse()

	if *setFlag != "" {
		version.Version = *setFlag
		fmt.Printf("Version set to: %s\n", version.Version)
		return
	}

	if *jsonFlag {
		out := map[string]string{
			"version":   version.GetVersion(),
			"gitCommit": version.GetGitCommit(),
			"buildTime": version.GetBuildTime(),
			"goVersion": version.GetGoVersion(),
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	if *infoFlag {
		fmt.Println(version.GetVersionInfo())
		return
	}

	// Default: print all fields
	fmt.Println("=== ling-base Version Info ===")
	fmt.Printf("  Version:   %s\n", version.GetVersion())
	fmt.Printf("  GitCommit: %s\n", version.GetGitCommit())
	fmt.Printf("  BuildTime: %s\n", version.GetBuildTime())
	fmt.Printf("  GoVersion: %s\n", version.GetGoVersion())
	fmt.Println()
	fmt.Printf("  Full:      %s\n", version.GetVersionInfo())
}
