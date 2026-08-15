// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package bootstrap

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProfileManager_Default(t *testing.T) {
	m := NewProfileManager("")
	assert.Equal(t, ProfileDev, m.DefaultProfile())
	assert.True(t, m.IsActive(ProfileDev))
	assert.True(t, m.IsDev())
}

func TestProfileManager_CustomDefault(t *testing.T) {
	m := NewProfileManager(ProfileProd)
	assert.True(t, m.IsProd())
}

func TestProfileManager_Active(t *testing.T) {
	m := NewProfileManager(ProfileDev)
	m.Active(ProfileProd, ProfileStaging)
	assert.True(t, m.IsActive(ProfileProd))
	assert.True(t, m.IsActive(ProfileStaging))
	assert.False(t, m.IsActive(ProfileDev))
}

func TestProfileManager_AddRemove(t *testing.T) {
	m := NewProfileManager(ProfileDev)
	m.AddActive(ProfileProd)
	assert.True(t, m.IsActive(ProfileProd))
	m.RemoveActive(ProfileProd)
	assert.False(t, m.IsActive(ProfileProd))
}

func TestProfileManager_ActiveProfiles(t *testing.T) {
	m := NewProfileManager(ProfileDev)
	m.AddActive(ProfileProd)
	profiles := m.ActiveProfiles()
	assert.Contains(t, profiles, ProfileDev)
	assert.Contains(t, profiles, ProfileProd)
}

func TestProfileManager_IsHelpers(t *testing.T) {
	m := NewProfileManager(ProfileDev)
	assert.True(t, m.IsDev())
	assert.False(t, m.IsTest())
	assert.False(t, m.IsStaging())
	assert.False(t, m.IsProd())
}

func TestProfileManager_LoadFromEnv(t *testing.T) {
	os.Setenv("APP_ENV", "staging")
	defer os.Unsetenv("APP_ENV")

	m := NewProfileManager(ProfileDev)
	m.LoadFromEnv("")
	assert.True(t, m.IsActive(ProfileStaging))
	assert.False(t, m.IsActive(ProfileDev))
}

func TestProfileManager_LoadFromEnvCustomKey(t *testing.T) {
	os.Setenv("MY_PROFILE", "prod")
	defer os.Unsetenv("MY_PROFILE")

	m := NewProfileManager(ProfileDev)
	m.LoadFromEnv("MY_PROFILE")
	assert.True(t, m.IsProd())
}

func TestProfileManager_LoadFromEnvNotSet(t *testing.T) {
	os.Unsetenv("APP_ENV")
	m := NewProfileManager(ProfileDev)
	m.LoadFromEnv("")
	assert.True(t, m.IsDev()) // stays default
}

func TestProfileManager_String(t *testing.T) {
	m := NewProfileManager(ProfileDev)
	s := m.String()
	assert.Contains(t, s, "default=dev")
}
