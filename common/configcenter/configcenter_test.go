// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package configcenter

import (
	"testing"
	"time"
)

func setupCC(t *testing.T) *ConfigCenter {
	t.Helper()
	cc := New()
	cc.AddApp("payment", "prod")
	if err := cc.AddNamespace("payment", "prod", "default", "application", false); err != nil {
		t.Fatalf("add namespace: %v", err)
	}
	return cc
}

// ─── Basic CRUD tests ───

func TestConfigCenter_AddNamespace(t *testing.T) {
	cc := New()
	cc.AddApp("app1", "dev")
	err := cc.AddNamespace("app1", "dev", "default", "application", false)
	if err != nil {
		t.Fatalf("add namespace: %v", err)
	}
	// Duplicate should fail.
	err = cc.AddNamespace("app1", "dev", "default", "application", false)
	if err == nil {
		t.Error("expected error for duplicate namespace")
	}
}

func TestConfigCenter_SetGetItem(t *testing.T) {
	cc := setupCC(t)
	if err := cc.SetItem("payment", "prod", "default", "application", "timeout", "5000"); err != nil {
		t.Fatalf("set item: %v", err)
	}

	// Verify via namespace directly.
	cc.mu.RLock()
	ns := cc.namespaces[namespaceKey("payment", "prod", "default", "application")]
	cc.mu.RUnlock()
	item, ok := ns.GetItem("timeout")
	if !ok || item.Value != "5000" {
		t.Errorf("expected timeout=5000, got %v ok=%v", item, ok)
	}
}

func TestConfigCenter_DeleteItem(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "key1", "val1")
	cc.DeleteItem("payment", "prod", "default", "application", "key1")

	cc.mu.RLock()
	ns := cc.namespaces[namespaceKey("payment", "prod", "default", "application")]
	cc.mu.RUnlock()
	if _, ok := ns.GetItem("key1"); ok {
		t.Error("expected item to be deleted")
	}
}

func TestConfigCenter_SetItem_NotFound(t *testing.T) {
	cc := New()
	err := cc.SetItem("x", "y", "z", "w", "k", "v")
	if err == nil {
		t.Error("expected error for missing namespace")
	}
}

// ─── Publish tests ───

func TestConfigCenter_Publish(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "timeout", "5000")
	cc.SetItem("payment", "prod", "default", "application", "retries", "3")

	release, err := cc.Publish("payment", "prod", "default", "application", "admin")
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if release.ReleaseKey == "" {
		t.Error("expected non-empty release key")
	}
	if release.Config["timeout"] != "5000" {
		t.Errorf("expected timeout=5000, got %v", release.Config["timeout"])
	}
	if release.Config["retries"] != "3" {
		t.Errorf("expected retries=3, got %v", release.Config["retries"])
	}
	if release.Operator != "admin" {
		t.Errorf("expected operator=admin, got %q", release.Operator)
	}
}

func TestConfigCenter_Publish_NotFound(t *testing.T) {
	cc := New()
	_, err := cc.Publish("x", "y", "z", "w", "op")
	if err == nil {
		t.Error("expected error for missing namespace")
	}
}

func TestConfigCenter_Publish_MultipleReleases(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "v", "1")
	r1, _ := cc.Publish("payment", "prod", "default", "application", "op")

	cc.SetItem("payment", "prod", "default", "application", "v", "2")
	r2, _ := cc.Publish("payment", "prod", "default", "application", "op")

	if r1.ID == r2.ID {
		t.Error("expected different release IDs")
	}
	if r1.ReleaseKey == r2.ReleaseKey {
		t.Error("expected different release keys")
	}
}

// ─── GetConfig tests ───

func TestConfigCenter_GetConfig(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "timeout", "5000")
	cc.Publish("payment", "prod", "default", "application", "op")

	result, err := cc.GetConfig("payment", "prod", "default", "application", "")
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if result.Config["timeout"] != "5000" {
		t.Errorf("expected timeout=5000, got %v", result.Config["timeout"])
	}
	if result.NotModified {
		t.Error("expected NotModified=false for first fetch")
	}
}

func TestConfigCenter_GetConfig_NotModified(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "timeout", "5000")
	release, _ := cc.Publish("payment", "prod", "default", "application", "op")

	result, err := cc.GetConfig("payment", "prod", "default", "application", release.ReleaseKey)
	if err != nil {
		t.Fatalf("get config: %v", err)
	}
	if !result.NotModified {
		t.Error("expected NotModified=true when releaseKey matches")
	}
}

func TestConfigCenter_GetConfig_NotFound(t *testing.T) {
	cc := New()
	_, err := cc.GetConfig("x", "y", "z", "w", "")
	if err == nil {
		t.Error("expected error for missing config")
	}
}

// ─── Cluster fallback tests ───

func TestConfigCenter_ClusterFallback(t *testing.T) {
	cc := New()
	cc.AddApp("app1", "prod")
	// Only publish to default cluster.
	cc.AddNamespace("app1", "prod", "default", "application", false)
	cc.SetItem("app1", "prod", "default", "application", "key", "default-val")
	cc.Publish("app1", "prod", "default", "application", "op")

	// Query from a custom cluster that doesn't have its own release.
	// Should fall back to default cluster.
	result, err := cc.GetConfig("app1", "prod", "dc-east", "application", "")
	if err != nil {
		t.Fatalf("get config with fallback: %v", err)
	}
	if result.Config["key"] != "default-val" {
		t.Errorf("expected fallback to default cluster, got %v", result.Config["key"])
	}
}

// ─── Public namespace fallback tests ───

func TestConfigCenter_PublicNamespaceFallback(t *testing.T) {
	cc := New()
	cc.AddApp("shared", "prod")
	cc.AddNamespace("shared", "prod", "default", "common-config", true) // public
	cc.SetItem("shared", "prod", "default", "common-config", "global-key", "global-val")
	cc.Publish("shared", "prod", "default", "common-config", "op")

	// App "app1" doesn't have its own "common-config" namespace.
	// Should fall back to the public namespace from "shared".
	result, err := cc.GetConfig("app1", "prod", "default", "common-config", "")
	if err != nil {
		t.Fatalf("get public config: %v", err)
	}
	if result.Config["global-key"] != "global-val" {
		t.Errorf("expected public namespace fallback, got %v", result.Config["global-key"])
	}
}

// ─── Rollback tests ───

func TestConfigCenter_Rollback(t *testing.T) {
	cc := setupCC(t)

	cc.SetItem("payment", "prod", "default", "application", "v", "1")
	cc.Publish("payment", "prod", "default", "application", "op")

	cc.SetItem("payment", "prod", "default", "application", "v", "2")
	cc.Publish("payment", "prod", "default", "application", "op")

	// Latest should be v=2.
	result, _ := cc.GetConfig("payment", "prod", "default", "application", "")
	if result.Config["v"] != "2" {
		t.Fatalf("expected v=2 before rollback, got %v", result.Config["v"])
	}

	// Rollback.
	err := cc.Rollback("payment", "prod", "default", "application")
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}

	// After rollback, latest active should be v=1.
	result, _ = cc.GetConfig("payment", "prod", "default", "application", "")
	if result.Config["v"] != "1" {
		t.Errorf("expected v=1 after rollback, got %v", result.Config["v"])
	}
}

func TestConfigCenter_Rollback_NoReleases(t *testing.T) {
	cc := setupCC(t)
	err := cc.Rollback("payment", "prod", "default", "application")
	if err == nil {
		t.Error("expected error for no releases")
	}
}

func TestConfigCenter_Rollback_OnlyOneRelease(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "v", "1")
	cc.Publish("payment", "prod", "default", "application", "op")

	err := cc.Rollback("payment", "prod", "default", "application")
	if err == nil {
		t.Error("expected error for single release")
	}
}

// ─── Release history tests ───

func TestConfigCenter_GetReleaseHistory(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "v", "1")
	cc.Publish("payment", "prod", "default", "application", "op")
	cc.SetItem("payment", "prod", "default", "application", "v", "2")
	cc.Publish("payment", "prod", "default", "application", "op")

	history, err := cc.GetReleaseHistory("payment", "prod", "default", "application")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(history) != 2 {
		t.Errorf("expected 2 releases, got %d", len(history))
	}
}

// ─── Watch / long-polling tests ───

func TestConfigCenter_Watch(t *testing.T) {
	cc := setupCC(t)

	watcher := cc.Watch("payment", "prod", "default", "application")
	defer watcher.Close()

	// Publish should trigger the watcher.
	cc.SetItem("payment", "prod", "default", "application", "key", "val")
	cc.Publish("payment", "prod", "default", "application", "op")

	select {
	case <-watcher.Ch():
		// Good — notification received.
	case <-time.After(1 * time.Second):
		t.Error("watcher did not receive notification")
	}
}

func TestConfigCenter_Watch_Rollback(t *testing.T) {
	cc := setupCC(t)
	cc.SetItem("payment", "prod", "default", "application", "v", "1")
	cc.Publish("payment", "prod", "default", "application", "op")
	cc.SetItem("payment", "prod", "default", "application", "v", "2")
	cc.Publish("payment", "prod", "default", "application", "op")

	watcher := cc.Watch("payment", "prod", "default", "application")
	defer watcher.Close()

	cc.Rollback("payment", "prod", "default", "application")

	select {
	case <-watcher.Ch():
		// Good.
	case <-time.After(1 * time.Second):
		t.Error("watcher did not receive rollback notification")
	}
}

func TestConfigCenter_Watch_Close(t *testing.T) {
	cc := setupCC(t)
	watcher := cc.Watch("payment", "prod", "default", "application")
	watcher.Close()

	// Double close should not panic.
	watcher.Close()
}

func TestConfigCenter_Watch_Multiple(t *testing.T) {
	cc := setupCC(t)

	w1 := cc.Watch("payment", "prod", "default", "application")
	defer w1.Close()
	w2 := cc.Watch("payment", "prod", "default", "application")
	defer w2.Close()

	cc.SetItem("payment", "prod", "default", "application", "k", "v")
	cc.Publish("payment", "prod", "default", "application", "op")

	// Both watchers should be notified.
	for i, w := range []*Watcher{w1, w2} {
		select {
		case <-w.Ch():
		case <-time.After(500 * time.Millisecond):
			t.Errorf("watcher %d did not receive notification", i)
		}
	}
}

// ─── Concurrency tests ───

func TestConfigCenter_ConcurrentPublish(t *testing.T) {
	cc := setupCC(t)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 20; j++ {
				cc.SetItem("payment", "prod", "default", "application",
					"key", "val")
				cc.Publish("payment", "prod", "default", "application", "op")
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < 10; i++ {
		<-done
	}

	history, _ := cc.GetReleaseHistory("payment", "prod", "default", "application")
	if len(history) != 200 {
		t.Errorf("expected 200 releases, got %d", len(history))
	}
}
