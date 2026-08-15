package version

import (
	"strings"
	"testing"
)

func TestGetVersion(t *testing.T) {
	if GetVersion() == "" {
		t.Error("GetVersion() returned empty string")
	}
}

func TestGetVersionInfo(t *testing.T) {
	info := GetVersionInfo()
	if info == "" {
		t.Error("GetVersionInfo() returned empty string")
	}
	if !strings.Contains(info, "commit:") {
		t.Error("GetVersionInfo() should contain commit info")
	}
}

func TestGetGitCommit(t *testing.T) {
	commit := GetGitCommit()
	if commit == "" {
		t.Error("GetGitCommit() returned empty string")
	}
}

func TestGetBuildTime(t *testing.T) {
	bt := GetBuildTime()
	if bt == "" {
		t.Error("GetBuildTime() returned empty string")
	}
}

func TestGetGoVersion(t *testing.T) {
	gv := GetGoVersion()
	if gv == "" {
		t.Error("GetGoVersion() returned empty string")
	}
}
