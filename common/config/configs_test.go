// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	lrucache "github.com/LingByte/ling-base/cache/lru"
)

// ===== Env-only mode =====

func TestStore_EnvOnly_GetValue(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	t.Setenv("MY_TEST_KEY", "hello")
	v := s.GetValue("MY_TEST_KEY")
	assert.Equal(t, "hello", v)
}

func TestStore_EnvOnly_GetValue_NotFound(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	v := s.GetValue("NONEXISTENT_KEY_12345")
	assert.Equal(t, "", v)
}

func TestStore_EnvOnly_SetValue(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	err = s.SetValueSimple("MY_SETTING", "testval")
	require.NoError(t, err)

	v := s.GetValue("MY_SETTING")
	assert.Equal(t, "testval", v)
}

func TestStore_EnvOnly_LookupValue(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	t.Setenv("LOOKUP_TEST", "found")
	v, ok := s.LookupValue("LOOKUP_TEST")
	assert.True(t, ok)
	assert.Equal(t, "found", v)

	_, ok = s.LookupValue("MISSING_KEY_XYZ")
	assert.False(t, ok)
}

func TestStore_EnvOnly_TypedGetters(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	t.Setenv("INT_VAL", "42")
	t.Setenv("FLOAT_VAL", "3.14")
	t.Setenv("BOOL_VAL", "true")
	t.Setenv("DUR_VAL", "5m30s")

	assert.Equal(t, 42, s.GetIntValue("INT_VAL", 0))
	assert.Equal(t, int64(42), s.GetInt64Value("INT_VAL", 0))
	assert.Equal(t, 3.14, s.GetFloatValue("FLOAT_VAL", 0))
	assert.True(t, s.GetBoolValue("BOOL_VAL"))
	assert.Equal(t, 5*time.Minute+30*time.Second, s.GetDurationValue("DUR_VAL", 0))
}

func TestStore_EnvOnly_TypedGetters_Defaults(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	assert.Equal(t, 99, s.GetIntValue("MISSING_INT", 99))
	assert.Equal(t, int64(99), s.GetInt64Value("MISSING_INT64", 99))
	assert.Equal(t, 1.5, s.GetFloatValue("MISSING_FLOAT", 1.5))
	assert.False(t, s.GetBoolValue("MISSING_BOOL"))
	assert.True(t, s.GetBoolValueWithDefault("MISSING_BOOL", true))
	assert.Equal(t, 10*time.Second, s.GetDurationValue("MISSING_DUR", 10*time.Second))
	assert.Equal(t, "fallback", s.GetStringWithDefault("MISSING_STR", "fallback"))
}

func TestStore_EnvOnly_CheckValue(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	// Should set default since key doesn't exist.
	err = s.CheckValue("NEW_KEY", "default_val", ConfigFormatText, false, false)
	require.NoError(t, err)
	assert.Equal(t, "default_val", s.GetValue("NEW_KEY"))
}

func TestStore_EnvOnly_DeleteValue(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	s.SetValueSimple("DEL_KEY", "val")
	assert.Equal(t, "val", s.GetValue("DEL_KEY"))

	err = s.DeleteValue("DEL_KEY")
	require.NoError(t, err)
	assert.Equal(t, "", s.GetValue("DEL_KEY"))
}

func TestStore_EnvOnly_LoadAutoloads(t *testing.T) {
	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	// No-op in env-only mode.
	err = s.LoadAutoloads()
	require.NoError(t, err)

	configs, err := s.LoadPublicConfigs()
	require.NoError(t, err)
	assert.Nil(t, configs)

	all, err := s.AllConfigs()
	require.NoError(t, err)
	assert.Nil(t, all)
}

func TestStore_EnvOnly_AutoMigrate(t *testing.T) {
	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	// No-op in env-only mode.
	err = s.AutoMigrate()
	require.NoError(t, err)
}

func TestStore_EnvOnly_HasDB(t *testing.T) {
	s, _ := NewEnvOnlyStore()
	defer s.Close()
	assert.False(t, s.HasDB())
}

// ===== DB mode =====

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestStore_DB_AutoMigrate(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()

	err = s.AutoMigrate()
	require.NoError(t, err)

	// Verify table exists by inserting.
	c := &ConfigItem{Key: "TEST", Value: "v"}
	require.NoError(t, db.Create(c).Error)

	var got ConfigItem
	require.NoError(t, db.Where("key = ?", "TEST").First(&got).Error)
	assert.Equal(t, "v", got.Value)
}

func TestStore_DB_SetGetValue(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	err = s.SetValue("DB_KEY", "db_value", ConfigFormatText, true, false)
	require.NoError(t, err)

	v := s.GetValue("DB_KEY")
	assert.Equal(t, "db_value", v)
}

func TestStore_DB_SetValueUpsert(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	// Insert.
	require.NoError(t, s.SetValue("UPSERT_KEY", "v1", ConfigFormatText, false, false))
	assert.Equal(t, "v1", s.GetValue("UPSERT_KEY"))

	// Update (upsert).
	require.NoError(t, s.SetValue("UPSERT_KEY", "v2", ConfigFormatJSON, true, true))
	assert.Equal(t, "v2", s.GetValue("UPSERT_KEY"))

	// Verify DB state.
	var c ConfigItem
	require.NoError(t, db.Where("key = ?", "UPSERT_KEY").First(&c).Error)
	assert.Equal(t, "v2", c.Value)
	assert.Equal(t, ConfigFormatJSON, c.Format)
	assert.True(t, c.Autoload)
	assert.True(t, c.Public)
}

func TestStore_DB_GetValue_NotInDB_FallsToEnv(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	t.Setenv("ENV_FALLBACK_KEY", "from_env")

	// Key not in DB, should fall back to env.
	v := s.GetValue("ENV_FALLBACK_KEY")
	assert.Equal(t, "from_env", v)
}

func TestStore_DB_TypedGetters(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	require.NoError(t, s.SetValue("PORT", "8080", ConfigFormatInt, false, false))
	require.NoError(t, s.SetValue("RATIO", "0.85", ConfigFormatFloat, false, false))
	require.NoError(t, s.SetValue("ENABLED", "true", ConfigFormatBool, false, false))
	require.NoError(t, s.SetValue("TIMEOUT", "30s", ConfigFormatText, false, false))

	assert.Equal(t, 8080, s.GetIntValue("PORT", 0))
	assert.Equal(t, 0.85, s.GetFloatValue("RATIO", 0))
	assert.True(t, s.GetBoolValue("ENABLED"))
	assert.Equal(t, 30*time.Second, s.GetDurationValue("TIMEOUT", 0))
}

func TestStore_DB_CheckValue(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	// First call inserts.
	require.NoError(t, s.CheckValue("SEED_KEY", "seed_val", ConfigFormatText, false, false))
	assert.Equal(t, "seed_val", s.GetValue("SEED_KEY"))

	// Second call should NOT overwrite.
	require.NoError(t, s.SetValue("SEED_KEY", "changed", ConfigFormatText, false, false))
	require.NoError(t, s.CheckValue("SEED_KEY", "seed_val", ConfigFormatText, false, false))
	assert.Equal(t, "changed", s.GetValue("SEED_KEY"))
}

func TestStore_DB_DeleteValue(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	require.NoError(t, s.SetValueSimple("DEL_ME", "val"))
	assert.Equal(t, "val", s.GetValue("DEL_ME"))

	require.NoError(t, s.DeleteValue("DEL_ME"))
	assert.Equal(t, "", s.GetValue("DEL_ME"))
}

func TestStore_DB_LoadAutoloads(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	require.NoError(t, s.SetValue("AUTO_1", "a1", ConfigFormatText, true, false))
	require.NoError(t, s.SetValue("AUTO_2", "a2", ConfigFormatText, true, false))
	require.NoError(t, s.SetValue("NON_AUTO", "na", ConfigFormatText, false, false))

	// Purge cache to force DB read.
	s.PurgeAllCache()

	require.NoError(t, s.LoadAutoloads())

	// After LoadAutoloads, these should be in cache.
	assert.Equal(t, "a1", s.GetValue("AUTO_1"))
	assert.Equal(t, "a2", s.GetValue("AUTO_2"))
}

func TestStore_DB_LoadPublicConfigs(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	require.NoError(t, s.SetValue("PUB_1", "p1", ConfigFormatText, false, true))
	require.NoError(t, s.SetValue("PUB_2", "p2", ConfigFormatText, false, true))
	require.NoError(t, s.SetValue("PRIV", "pv", ConfigFormatText, false, false))

	s.PurgeAllCache()

	configs, err := s.LoadPublicConfigs()
	require.NoError(t, err)
	assert.Len(t, configs, 2)
}

func TestStore_DB_AllConfigs(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	require.NoError(t, s.SetValueSimple("K1", "v1"))
	require.NoError(t, s.SetValueSimple("K2", "v2"))

	all, err := s.AllConfigs()
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestStore_DB_HasDB(t *testing.T) {
	db := newTestDB(t)
	s, _ := NewStoreWithDB(db)
	defer s.Close()
	assert.True(t, s.HasDB())
}

func TestStore_DB_CachePurge(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	defer s.Close()
	require.NoError(t, s.AutoMigrate())

	require.NoError(t, s.SetValueSimple("CACHED", "val"))
	assert.Equal(t, "val", s.GetValue("CACHED"))

	s.PurgeCache("CACHED")
	// Should still get from DB after cache purge.
	assert.Equal(t, "val", s.GetValue("CACHED"))
}

func TestStore_DB_Close(t *testing.T) {
	db := newTestDB(t)
	s, err := NewStoreWithDB(db)
	require.NoError(t, err)
	assert.NoError(t, s.Close())
}

// ===== ConfigItem model =====

func TestConfigItem_TableName(t *testing.T) {
	assert.Equal(t, "configs", ConfigItem{}.TableName())
}

// ===== Key normalization =====

func TestStore_KeyNormalization(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	s, err := NewEnvOnlyStore()
	require.NoError(t, err)
	defer s.Close()

	t.Setenv("NORMALIZED_KEY", "val")

	// Different casing should match.
	assert.Equal(t, "val", s.GetValue("normalized_key"))
	assert.Equal(t, "val", s.GetValue("NORMALIZED_KEY"))
	assert.Equal(t, "val", s.GetValue("  Normalized_Key  "))
}

// ===== SetEnv helper =====

func TestSetEnv(t *testing.T) {
	err := SetEnv("MY_HELPER_ENV", "helper_val")
	require.NoError(t, err)
	assert.Equal(t, "helper_val", s_getEnvRaw("MY_HELPER_ENV"))
}

func s_getEnvRaw(key string) string {
	// Direct os.Getenv for test verification.
	v, ok := lookupEnvValue(key)
	if !ok {
		return ""
	}
	return v
}

// ===== Custom cache injection =====

func TestStore_CustomCache(t *testing.T) {
	envStore.Purge()
	t.Cleanup(func() { envStore.Purge() })

	// Use a custom cache (LRU with short TTL).
	c, err := lrucache.New(100, lrucache.WithDefaultTTL(5*time.Second))
	require.NoError(t, err)

	s, err := NewStore(StoreOptions{
		Cache: c,
	})
	require.NoError(t, err)
	defer s.Close()

	t.Setenv("CUSTOM_CACHE_KEY", "cached_val")
	assert.Equal(t, "cached_val", s.GetValue("CUSTOM_CACHE_KEY"))
}
