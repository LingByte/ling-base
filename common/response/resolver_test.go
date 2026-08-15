// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package response

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStaticResolver_FoundKey(t *testing.T) {
	r := &StaticResolver{Messages: map[string]string{
		"common.not_found": "Resource not found",
	}}
	assert.Equal(t, "Resource not found", r.Resolve("common.not_found"))
}

func TestStaticResolver_MissingKeyReturnsKey(t *testing.T) {
	r := &StaticResolver{Messages: map[string]string{}}
	assert.Equal(t, "missing.key", r.Resolve("missing.key"))
}

func TestStaticResolver_WithArgs(t *testing.T) {
	r := &StaticResolver{Messages: map[string]string{
		"common.not_found": "Resource %s with id %d not found",
	}}
	assert.Equal(t, "Resource user with id 7 not found", r.Resolve("common.not_found", "user", 7))
}

func TestStaticResolver_NilReceiver(t *testing.T) {
	var r *StaticResolver
	assert.Equal(t, "", r.Resolve("any.key"))
}

func TestResolverFunc_Normal(t *testing.T) {
	f := ResolverFunc(func(key string, args ...any) string {
		return "resolved:" + key
	})
	assert.Equal(t, "resolved:common.not_found", f.Resolve("common.not_found"))
}

func TestResolverFunc_WithArgs(t *testing.T) {
	f := ResolverFunc(func(key string, args ...any) string {
		return key + ":" + args[0].(string)
	})
	assert.Equal(t, "common.not_found:user", f.Resolve("common.not_found", "user"))
}

func TestResolverFunc_Nil(t *testing.T) {
	var f ResolverFunc
	assert.Equal(t, "", f.Resolve("any.key"))
}

func TestNoopResolver(t *testing.T) {
	assert.Equal(t, "common.not_found", NoopResolver.Resolve("common.not_found"))
	assert.Equal(t, "any.key", NoopResolver.Resolve("any.key"))
	// with args, still returns key
	assert.Equal(t, "common.not_found", NoopResolver.Resolve("common.not_found", "a", 1))
}

func TestChainResolver_FirstHit(t *testing.T) {
	r1 := &StaticResolver{Messages: map[string]string{"k": "v1"}}
	r2 := &StaticResolver{Messages: map[string]string{"k": "v2"}}
	c := &ChainResolver{Resolvers: []MessageResolver{r1, r2}}
	assert.Equal(t, "v1", c.Resolve("k"))
}

func TestChainResolver_SecondHit(t *testing.T) {
	r1 := &StaticResolver{Messages: map[string]string{}}
	r2 := &StaticResolver{Messages: map[string]string{"k": "v2"}}
	c := &ChainResolver{Resolvers: []MessageResolver{r1, r2}}
	assert.Equal(t, "v2", c.Resolve("k"))
}

func TestChainResolver_AllMissReturnsKey(t *testing.T) {
	r1 := &StaticResolver{Messages: map[string]string{}}
	r2 := &StaticResolver{Messages: map[string]string{}}
	c := &ChainResolver{Resolvers: []MessageResolver{r1, r2}}
	assert.Equal(t, "missing.key", c.Resolve("missing.key"))
}

func TestChainResolver_NilResolversSkipped(t *testing.T) {
	r2 := &StaticResolver{Messages: map[string]string{"k": "v2"}}
	c := &ChainResolver{Resolvers: []MessageResolver{nil, r2}}
	assert.Equal(t, "v2", c.Resolve("k"))
}

func TestChainResolver_AllNilResolvers(t *testing.T) {
	c := &ChainResolver{Resolvers: []MessageResolver{nil, nil}}
	assert.Equal(t, "missing.key", c.Resolve("missing.key"))
}

func TestChainResolver_NilReceiver(t *testing.T) {
	var c *ChainResolver
	assert.Equal(t, "any.key", c.Resolve("any.key"))
}

func TestChainResolver_WithArgs(t *testing.T) {
	r1 := &StaticResolver{Messages: map[string]string{"k": "value %s %d"}}
	c := &ChainResolver{Resolvers: []MessageResolver{r1}}
	assert.Equal(t, "value a 1", c.Resolve("k", "a", 1))
}

func TestChainResolver_EmptyResultSkipped(t *testing.T) {
	// ResolverFunc returning "" should be skipped (empty result)
	f1 := ResolverFunc(func(key string, args ...any) string { return "" })
	r2 := &StaticResolver{Messages: map[string]string{"k": "v2"}}
	c := &ChainResolver{Resolvers: []MessageResolver{f1, r2}}
	assert.Equal(t, "v2", c.Resolve("k"))
}
