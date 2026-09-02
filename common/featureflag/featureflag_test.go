// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package featureflag

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManager_EnableDisable(t *testing.T) {
	m := NewManager()

	m.SetFlag(&Flag{Name: "f1", Enabled: false})
	assert.False(t, m.IsEnabled("f1", "u1"))

	m.Enable("f1")
	assert.True(t, m.IsEnabled("f1", "u1"))
	assert.True(t, m.IsEnabled("f1", "u2"))

	m.Disable("f1")
	assert.False(t, m.IsEnabled("f1", "u1"))

	// Enable on a non-existent flag creates it.
	m.Enable("f2")
	assert.True(t, m.IsEnabled("f2", "anyone"))

	// Disable on a non-existent flag creates it disabled.
	m.Disable("f3")
	assert.False(t, m.IsEnabled("f3", "anyone"))
}

func TestManager_PercentageConsistency(t *testing.T) {
	m := NewManager()
	m.SetFlag(&Flag{Name: "rollout", Enabled: true, Percentage: 50})

	// The same user must always get the same answer.
	for i := 0; i < 10; i++ {
		first := m.IsEnabled("rollout", "user-123")
		second := m.IsEnabled("rollout", "user-123")
		require.Equal(t, first, second)
	}

	// Across many users roughly half should be enabled.
	enabled := 0
	total := 1000
	for i := 0; i < total; i++ {
		uid := "user-" + itoa(i)
		if m.IsEnabled("rollout", uid) {
			enabled++
		}
	}
	// Allow a generous band to avoid flakiness.
	assert.InDelta(t, 0.5, float64(enabled)/float64(total), 0.1)

	// 0% disables everyone (except whitelist, none here).
	m.SetPercentage("rollout", 0)
	for i := 0; i < 10; i++ {
		assert.False(t, m.IsEnabled("rollout", "user-"+itoa(i)))
	}

	// 100% enables everyone.
	m.SetPercentage("rollout", 100)
	for i := 0; i < 10; i++ {
		assert.True(t, m.IsEnabled("rollout", "user-"+itoa(i)))
	}
}

func TestManager_WhitelistPrecedence(t *testing.T) {
	m := NewManager()
	// 0% rollout but a whitelisted user should still be enabled.
	m.SetFlag(&Flag{
		Name:       "wl",
		Enabled:    true,
		Percentage: 0,
		Whitelist:  []string{"vip"},
	})
	assert.True(t, m.IsEnabled("wl", "vip"))
	assert.False(t, m.IsEnabled("wl", "normal"))

	// AddWhitelist enables and adds.
	m.AddWhitelist("wl", "vip2")
	assert.True(t, m.IsEnabled("wl", "vip2"))
	// Adding twice does not duplicate.
	m.AddWhitelist("wl", "vip2")
	f := m.GetFlag("wl")
	assert.ElementsMatch(t, []string{"vip", "vip2"}, f.Whitelist)
}

func TestManager_Blacklist(t *testing.T) {
	m := NewManager()
	m.SetFlag(&Flag{
		Name:       "bl",
		Enabled:    true,
		Percentage: 100,
		Whitelist:  []string{"vip"},
		Blacklist:  []string{"bad"},
	})
	// Blacklist overrides whitelist and 100% rollout.
	assert.False(t, m.IsEnabled("bl", "bad"))
	// Whitelisted user still enabled.
	assert.True(t, m.IsEnabled("bl", "vip"))
	// Normal user enabled because 100%.
	assert.True(t, m.IsEnabled("bl", "normal"))
}

func TestManager_GetFlagAndAllFlags(t *testing.T) {
	m := NewManager()
	m.SetFlag(&Flag{Name: "a", Enabled: true, Percentage: 10, Metadata: map[string]any{"k": "v"}})
	m.SetFlag(&Flag{Name: "b", Enabled: false})

	f := m.GetFlag("a")
	require.NotNil(t, f)
	assert.Equal(t, "a", f.Name)
	assert.Equal(t, "v", f.Metadata["k"])

	// Mutating the returned copy must not affect the manager.
	f.Metadata["k"] = "changed"
	assert.Equal(t, "v", m.GetFlag("a").Metadata["k"])

	assert.Nil(t, m.GetFlag("missing"))

	all := m.AllFlags()
	require.Len(t, all, 2)
	assert.Equal(t, "a", all[0].Name)
	assert.Equal(t, "b", all[1].Name)
}

func TestManager_RemoveFlag(t *testing.T) {
	m := NewManager()
	m.SetFlag(&Flag{Name: "temp", Enabled: true, Percentage: 100})
	assert.True(t, m.IsEnabled("temp", "u"))
	m.RemoveFlag("temp")
	assert.False(t, m.IsEnabled("temp", "u"))
	assert.Nil(t, m.GetFlag("temp"))
	// Removing a missing flag is a no-op.
	m.RemoveFlag("nope")
}

func TestManager_NonExistentFlag(t *testing.T) {
	m := NewManager()
	assert.False(t, m.IsEnabled("ghost", "u"))
}

func TestManager_Concurrent(t *testing.T) {
	m := NewManager()
	m.SetFlag(&Flag{Name: "c", Enabled: true, Percentage: 50, Whitelist: []string{"w"}, Blacklist: []string{"b"}})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			uid := "u" + itoa(i%20)
			_ = m.IsEnabled("c", uid)
			m.AddWhitelist("c", "extra-"+itoa(i))
			m.SetPercentage("c", i%101)
			_ = m.GetFlag("c")
			_ = m.AllFlags()
		}(i)
	}
	wg.Wait()
}

func TestManager_SetFlagClampsPercentage(t *testing.T) {
	m := NewManager()
	m.SetFlag(&Flag{Name: "clamp", Enabled: true, Percentage: 200})
	f := m.GetFlag("clamp")
	require.NotNil(t, f)
	assert.Equal(t, 100, f.Percentage)

	m.SetFlag(&Flag{Name: "clamp", Enabled: true, Percentage: -5})
	f = m.GetFlag("clamp")
	require.NotNil(t, f)
	assert.Equal(t, 0, f.Percentage)
}

func TestManager_SetFlagNilAndEmptyName(t *testing.T) {
	m := NewManager()
	// nil flag is a no-op.
	m.SetFlag(nil)
	assert.Empty(t, m.AllFlags())
	// Empty name is a no-op.
	m.SetFlag(&Flag{Name: "", Enabled: true})
	assert.Empty(t, m.AllFlags())
}

func TestManager_SetFlagCopiesSlicesAndMetadata(t *testing.T) {
	m := NewManager()
	wl := []string{"a", "b"}
	bl := []string{"x"}
	md := map[string]any{"k": "v"}
	m.SetFlag(&Flag{Name: "cp", Enabled: true, Percentage: 100, Whitelist: wl, Blacklist: bl, Metadata: md})

	// Mutate the original slices/map; the manager's copy must be unaffected.
	wl[0] = "changed"
	bl[0] = "changed"
	md["k"] = "changed"
	f := m.GetFlag("cp")
	require.NotNil(t, f)
	assert.Equal(t, []string{"a", "b"}, f.Whitelist)
	assert.Equal(t, []string{"x"}, f.Blacklist)
	assert.Equal(t, "v", f.Metadata["k"])
}

func TestManager_SetPercentageClamps(t *testing.T) {
	m := NewManager()
	// Negative clamps to 0 and still enables the flag.
	m.SetPercentage("p", -10)
	f := m.GetFlag("p")
	require.NotNil(t, f)
	assert.True(t, f.Enabled)
	assert.Equal(t, 0, f.Percentage)
	assert.False(t, m.IsEnabled("p", "u"))

	// Over 100 clamps to 100.
	m.SetPercentage("p", 999)
	f = m.GetFlag("p")
	require.NotNil(t, f)
	assert.Equal(t, 100, f.Percentage)
	assert.True(t, m.IsEnabled("p", "u"))
}

func TestManager_AddWhitelistDuplicate(t *testing.T) {
	m := NewManager()
	m.AddWhitelist("w", "u1")
	m.AddWhitelist("w", "u1") // duplicate, should be ignored
	m.AddWhitelist("w", "u2")
	f := m.GetFlag("w")
	require.NotNil(t, f)
	assert.ElementsMatch(t, []string{"u1", "u2"}, f.Whitelist)
	assert.True(t, f.Enabled)
}

func TestManager_AllFlagsEmpty(t *testing.T) {
	m := NewManager()
	all := m.AllFlags()
	assert.Empty(t, all)
}

func TestManager_IsEnabledConcurrent(t *testing.T) {
	m := NewManager()
	m.SetFlag(&Flag{Name: "cc", Enabled: true, Percentage: 50})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			uid := "u" + itoa(i%30)
			_ = m.IsEnabled("cc", uid)
		}(i)
	}
	wg.Wait()
}

func TestManager_PercentageBoundaries(t *testing.T) {
	m := NewManager()
	// 0% with no whitelist: everyone disabled.
	m.SetFlag(&Flag{Name: "zero", Enabled: true, Percentage: 0})
	assert.False(t, m.IsEnabled("zero", "any"))
	// 100% with no blacklist: everyone enabled.
	m.SetFlag(&Flag{Name: "full", Enabled: true, Percentage: 100})
	assert.True(t, m.IsEnabled("full", "any"))
}

func TestManager_WhitelistAndBlacklistSameUser(t *testing.T) {
	m := NewManager()
	// When the same user is in both whitelist and blacklist, blacklist wins.
	m.SetFlag(&Flag{
		Name:       "wb",
		Enabled:    true,
		Percentage: 100,
		Whitelist:  []string{"u"},
		Blacklist:  []string{"u"},
	})
	assert.False(t, m.IsEnabled("wb", "u"))
}

// itoa is a tiny dependency-free int->string helper to keep the test file
// free of strconv imports (purely stylistic).
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	pos := len(buf)
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
