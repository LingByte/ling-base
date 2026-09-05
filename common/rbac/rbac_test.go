// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package rbac_test

import (
	"testing"

	"github.com/LingByte/ling-base/common/rbac"
)

func TestManager_AddPolicy_And_Enforce(t *testing.T) {
	mgr, err := rbac.NewMemory()
	if err != nil {
		t.Fatalf("NewMemory: %v", err)
	}
	defer mgr.Close()

	ok, err := mgr.AddPolicy("admin", "/api/users", "GET")
	if err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}
	if !ok {
		t.Error("AddPolicy returned false")
	}

	// Admin should be allowed.
	allowed, err := mgr.Enforce("admin", "/api/users", "GET")
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if !allowed {
		t.Error("admin should be allowed GET /api/users")
	}

	// Viewer should not be allowed.
	allowed, _ = mgr.Enforce("viewer", "/api/users", "GET")
	if allowed {
		t.Error("viewer should not be allowed")
	}
}

func TestManager_RemovePolicy(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")
	ok, err := mgr.RemovePolicy("admin", "/api/users", "GET")
	if err != nil {
		t.Fatalf("RemovePolicy: %v", err)
	}
	if !ok {
		t.Error("RemovePolicy returned false")
	}

	allowed, _ := mgr.Enforce("admin", "/api/users", "GET")
	if allowed {
		t.Error("admin should not be allowed after policy removal")
	}
}

func TestManager_HasPolicy(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")
	has, _ := mgr.HasPolicy("admin", "/api/users", "GET")
	if !has {
		t.Error("HasPolicy should return true")
	}

	has, _ = mgr.HasPolicy("admin", "/api/users", "POST")
	if has {
		t.Error("HasPolicy should return false for non-existent policy")
	}
}

func TestManager_ListPolicies(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")
	mgr.AddPolicy("admin", "/api/users", "POST")
	mgr.AddPolicy("viewer", "/api/users", "GET")

	all := mgr.ListPolicies()
	if len(all) != 3 {
		t.Errorf("ListPolicies count = %d, want 3", len(all))
	}

	adminPolicies := mgr.ListPoliciesForSubject("admin")
	if len(adminPolicies) != 2 {
		t.Errorf("ListPoliciesForSubject(admin) = %d, want 2", len(adminPolicies))
	}
}

func TestManager_ClearPolicies(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")
	mgr.AddPolicy("admin", "/api/users", "POST")

	err := mgr.ClearPolicies("admin")
	if err != nil {
		t.Fatalf("ClearPolicies: %v", err)
	}

	policies := mgr.ListPoliciesForSubject("admin")
	if len(policies) != 0 {
		t.Errorf("Policies after clear = %d, want 0", len(policies))
	}
}

func TestManager_RoleInheritance(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	// Grant admin access to /api/users.
	mgr.AddPolicy("admin", "/api/users", "GET")

	// Assign alice to admin role.
	ok, err := mgr.AssignRole("alice", "admin")
	if err != nil {
		t.Fatalf("AssignRole: %v", err)
	}
	if !ok {
		t.Error("AssignRole returned false")
	}

	// Alice should inherit admin's permissions.
	allowed, _ := mgr.Enforce("alice", "/api/users", "GET")
	if !allowed {
		t.Error("alice should inherit admin's permissions")
	}

	// HasRole check.
	has, _ := mgr.HasRole("alice", "admin")
	if !has {
		t.Error("alice should have admin role")
	}

	// RolesForUser.
	roles, _ := mgr.RolesForUser("alice")
	if len(roles) != 1 || roles[0] != "admin" {
		t.Errorf("RolesForUser = %v, want [admin]", roles)
	}

	// UsersForRole.
	users, _ := mgr.UsersForRole("admin")
	if len(users) != 1 || users[0] != "alice" {
		t.Errorf("UsersForRole = %v, want [alice]", users)
	}
}

func TestManager_RevokeRole(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")
	mgr.AssignRole("alice", "admin")

	ok, err := mgr.RevokeRole("alice", "admin")
	if err != nil {
		t.Fatalf("RevokeRole: %v", err)
	}
	if !ok {
		t.Error("RevokeRole returned false")
	}

	allowed, _ := mgr.Enforce("alice", "/api/users", "GET")
	if allowed {
		t.Error("alice should not have access after role revocation")
	}
}

func TestManager_DeleteRole(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")
	mgr.AssignRole("alice", "admin")

	err := mgr.DeleteRole("admin")
	if err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	// Alice should no longer have access.
	allowed, _ := mgr.Enforce("alice", "/api/users", "GET")
	if allowed {
		t.Error("alice should not have access after role deletion")
	}
}

func TestManager_SetPolicies(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/old", "GET")

	err := mgr.SetPolicies("admin", [][2]string{
		{"/api/users", "GET"},
		{"/api/users", "POST"},
		{"/api/users", "GET"}, // duplicate
	})
	if err != nil {
		t.Fatalf("SetPolicies: %v", err)
	}

	policies := mgr.ListPoliciesForSubject("admin")
	if len(policies) != 2 {
		t.Errorf("Policies after SetPolicies = %d, want 2 (deduplicated)", len(policies))
	}

	// Old policy should be gone.
	has, _ := mgr.HasPolicy("admin", "/api/old", "GET")
	if has {
		t.Error("old policy should have been cleared")
	}
}

func TestManager_Enforce_EmptyFields(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	_, err := mgr.Enforce("", "/api", "GET")
	if err != rbac.ErrEmptySubject {
		t.Errorf("Enforce with empty subject: err = %v, want ErrEmptySubject", err)
	}

	_, err = mgr.Enforce("admin", "", "GET")
	if err != rbac.ErrEmptyObject {
		t.Errorf("Enforce with empty object: err = %v, want ErrEmptyObject", err)
	}

	_, err = mgr.Enforce("admin", "/api", "")
	if err != rbac.ErrEmptyAction {
		t.Errorf("Enforce with empty action: err = %v, want ErrEmptyAction", err)
	}
}

func TestManager_AddPolicies_Batch(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	rules := [][]string{
		{"admin", "/api/users", "GET"},
		{"admin", "/api/users", "POST"},
		{"admin", "/api/orders", "GET"},
	}
	ok, err := mgr.AddPolicies(rules)
	if err != nil {
		t.Fatalf("AddPolicies: %v", err)
	}
	if !ok {
		t.Error("AddPolicies returned false")
	}

	policies := mgr.ListPoliciesForSubject("admin")
	if len(policies) != 3 {
		t.Errorf("Policies count = %d, want 3", len(policies))
	}
}

// ──────────────────────────────────────────────
// Checker tests
// ──────────────────────────────────────────────

func TestChecker_Check(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")

	checker := rbac.NewChecker(mgr)

	ctx := rbac.SimpleCheckContext{Sub: "admin", Obj: "/api/users", Act: "GET"}
	ok, err := checker.Check(ctx)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !ok {
		t.Error("Check should allow admin GET /api/users")
	}

	ctx = rbac.SimpleCheckContext{Sub: "viewer", Obj: "/api/users", Act: "GET"}
	ok, _ = checker.Check(ctx)
	if ok {
		t.Error("Check should deny viewer")
	}
}

func TestChecker_CustomExtractors(t *testing.T) {
	mgr, _ := rbac.NewMemory()
	defer mgr.Close()

	mgr.AddPolicy("admin", "/api/users", "GET")

	checker := rbac.NewChecker(mgr,
		rbac.WithSubjectFunc(func(ctx rbac.CheckContext) string {
			return "admin" // always admin for this test
		}),
	)

	ctx := rbac.SimpleCheckContext{Obj: "/api/users", Act: "GET"}
	ok, _ := checker.Check(ctx)
	if !ok {
		t.Error("Check with custom subject extractor should allow")
	}
}

// ──────────────────────────────────────────────
// NormalizePath tests
// ──────────────────────────────────────────────

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		path   string
		prefix string
		want   string
	}{
		{"/api/v1/users", "/api", "/v1/users"},
		{"/api/v1/users/", "/api", "/v1/users"},
		{"/users", "", "/users"},
		{"/", "", "/"},
		{"/api", "/api", "/"},
	}
	for _, tt := range tests {
		got := rbac.NormalizePath(tt.path, tt.prefix)
		if got != tt.want {
			t.Errorf("NormalizePath(%q, %q) = %q, want %q", tt.path, tt.prefix, got, tt.want)
		}
	}
}
