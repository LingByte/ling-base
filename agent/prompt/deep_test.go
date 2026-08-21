package prompt

import (
	"strings"
	"testing"
)

func TestDeepThinkingPrompt(t *testing.T) {
	got := DeepThinkingPrompt("应该用微服务还是单体架构？")
	if !strings.Contains(got, "钢人论证") {
		t.Error("prompt should contain 钢人论证")
	}
	if !strings.Contains(got, "应该用微服务还是单体架构？") {
		t.Error("prompt should contain the user's question")
	}
	if !strings.Contains(got, "只问我一个最关键的问题") {
		t.Error("prompt should instruct the model to ask one key question")
	}
}

func TestDeepThinkingPromptEmpty(t *testing.T) {
	got := DeepThinkingPrompt("")
	if !strings.Contains(got, "我的问题是：") {
		t.Error("prompt should still contain the template with empty question")
	}
}
