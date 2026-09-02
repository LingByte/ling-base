// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package consistent

import (
	"fmt"
	"hash/fnv"
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ──────────────────────────────────────────────
// Construction / options
// ──────────────────────────────────────────────

func TestNew_Defaults(t *testing.T) {
	r := New()
	assert.Equal(t, defaultReplicas, r.replicas)
	assert.NotNil(t, r.hash)
	assert.Equal(t, 0, r.Size())
}

func TestNew_WithReplicas(t *testing.T) {
	r := New(WithReplicas(100))
	assert.Equal(t, 100, r.replicas)
}

func TestNew_WithHash(t *testing.T) {
	custom := func(data []byte) uint32 {
		h := fnv.New32a()
		_, _ = h.Write(data)
		return h.Sum32()
	}
	r := New(WithHash(custom))
	// Verify the custom hash is in use by checking that a key routes
	// differently than it would with the default hash (fnv != crc32 for
	// most inputs). We confirm via Distribution that virtual nodes exist.
	r.Add("a", "b")
	assert.Equal(t, 2, r.Size())
	dist := r.Distribution()
	assert.Equal(t, r.replicas, dist["a"])
	assert.Equal(t, r.replicas, dist["b"])
}

func TestNew_WithHash_NilIgnored(t *testing.T) {
	r := New(WithHash(nil))
	// nil hash keeps DefaultHash; verify the ring still works.
	r.Add("a")
	node, ok := r.Get("key")
	require.True(t, ok)
	assert.Equal(t, "a", node)
}

func TestNew_InvalidReplicas_Panics(t *testing.T) {
	assert.Panics(t, func() {
		New(WithReplicas(0))
	})
	assert.Panics(t, func() {
		New(WithReplicas(-1))
	})
}

// ──────────────────────────────────────────────
// Add / Remove / Nodes / Size / Contains
// ──────────────────────────────────────────────

func TestAdd_Size(t *testing.T) {
	r := New()
	r.Add("a", "b", "c")
	assert.Equal(t, 3, r.Size())
}

func TestAdd_Duplicate_NoOp(t *testing.T) {
	r := New()
	r.Add("a")
	r.Add("a")
	assert.Equal(t, 1, r.Size())
}

func TestAdd_EmptyString_Ignored(t *testing.T) {
	r := New()
	r.Add("")
	assert.Equal(t, 0, r.Size())
}

func TestNodes_Sorted(t *testing.T) {
	r := New()
	r.Add("c", "a", "b")
	assert.Equal(t, []string{"a", "b", "c"}, r.Nodes())
}

func TestContains(t *testing.T) {
	r := New()
	r.Add("a", "b")
	assert.True(t, r.Contains("a"))
	assert.False(t, r.Contains("z"))
}

func TestRemove(t *testing.T) {
	r := New()
	r.Add("a", "b", "c")
	r.Remove("b")
	assert.Equal(t, 2, r.Size())
	assert.False(t, r.Contains("b"))
	assert.True(t, r.Contains("a"))
	assert.True(t, r.Contains("c"))
}

func TestRemove_NonExistent_NoOp(t *testing.T) {
	r := New()
	r.Add("a")
	r.Remove("xyz")
	assert.Equal(t, 1, r.Size())
}

func TestRemove_All(t *testing.T) {
	r := New()
	r.Add("a", "b")
	r.Remove("a", "b")
	assert.Equal(t, 0, r.Size())
	_, ok := r.Get("key")
	assert.False(t, ok)
}

// ──────────────────────────────────────────────
// Get consistency
// ──────────────────────────────────────────────

func TestGet_Consistency(t *testing.T) {
	r := New()
	r.Add("node1", "node2", "node3")

	node1, ok1 := r.Get("my-key")
	require.True(t, ok1)
	// Same key must always map to the same node.
	for i := 0; i < 100; i++ {
		node, ok := r.Get("my-key")
		require.True(t, ok)
		assert.Equal(t, node1, node)
	}
}

func TestGet_DistributesAcrossNodes(t *testing.T) {
	r := New(WithReplicas(100))
	r.Add("n1", "n2", "n3", "n4", "n5")

	counts := make(map[string]int)
	for i := 0; i < 10000; i++ {
		node, ok := r.Get(fmt.Sprintf("key-%d", i))
		require.True(t, ok)
		counts[node]++
	}
	// Every node should get some keys.
	for _, node := range r.Nodes() {
		assert.Greater(t, counts[node], 0, "node %s got 0 keys", node)
	}
}

func TestGet_EmptyRing(t *testing.T) {
	r := New()
	node, ok := r.Get("key")
	assert.False(t, ok)
	assert.Empty(t, node)
}

// ──────────────────────────────────────────────
// GetN
// ──────────────────────────────────────────────

func TestGetN_DistinctNodes(t *testing.T) {
	r := New()
	r.Add("a", "b", "c", "d", "e")

	nodes, err := r.GetN("key", 3)
	require.NoError(t, err)
	require.Len(t, nodes, 3)

	seen := make(map[string]bool)
	for _, n := range nodes {
		assert.False(t, seen[n], "node %s duplicated", n)
		seen[n] = true
	}
}

func TestGetN_MoreThanNodes(t *testing.T) {
	r := New()
	r.Add("a", "b")

	nodes, err := r.GetN("key", 5)
	require.NoError(t, err)
	assert.Len(t, nodes, 2, "should return all available nodes")
}

func TestGetN_EmptyRing(t *testing.T) {
	r := New()
	nodes, err := r.GetN("key", 3)
	assert.ErrorIs(t, err, ErrEmptyRing)
	assert.Nil(t, nodes)
}

func TestGetN_ZeroOrNegative(t *testing.T) {
	r := New()
	r.Add("a")

	nodes, err := r.GetN("key", 0)
	require.NoError(t, err)
	assert.Nil(t, nodes)

	nodes, err = r.GetN("key", -1)
	require.NoError(t, err)
	assert.Nil(t, nodes)
}

func TestGetN_Consistency(t *testing.T) {
	r := New()
	r.Add("a", "b", "c", "d")

	first, err := r.GetN("key", 3)
	require.NoError(t, err)
	for i := 0; i < 50; i++ {
		got, err := r.GetN("key", 3)
		require.NoError(t, err)
		assert.Equal(t, first, got)
	}
}

// ──────────────────────────────────────────────
// Minimal movement on topology change
// ──────────────────────────────────────────────

func TestMinimalMovement_OnAdd(t *testing.T) {
	r := New(WithReplicas(100))
	r.Add("n1", "n2", "n3")

	// Record mapping for many keys.
	original := make(map[string]string)
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		k := fmt.Sprintf("key-%d", i)
		keys[i] = k
		node, ok := r.Get(k)
		require.True(t, ok)
		original[k] = node
	}

	// Add a node; most keys should stay on the same node.
	r.Add("n4")
	moved := 0
	for _, k := range keys {
		node, ok := r.Get(k)
		require.True(t, ok)
		if node != original[k] {
			moved++
		}
	}
	// Adding 1 node to 3 should move roughly 1/4 of keys. Allow generous
	// slack for hash variance.
	assert.Less(t, moved, 400, "adding one node should move a minority of keys, moved %d", moved)
}

func TestMinimalMovement_OnRemove(t *testing.T) {
	r := New(WithReplicas(100))
	r.Add("n1", "n2", "n3", "n4")

	original := make(map[string]string)
	keys := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		k := fmt.Sprintf("key-%d", i)
		keys[i] = k
		node, ok := r.Get(k)
		require.True(t, ok)
		original[k] = node
	}

	// Remove a node; only keys that were on the removed node should move.
	r.Remove("n2")
	moved := 0
	for _, k := range keys {
		node, ok := r.Get(k)
		require.True(t, ok)
		if node != original[k] {
			moved++
			// Moved keys must have originally been on the removed node.
			assert.Equal(t, "n2", original[k], "key %s moved from %s (not the removed node)", k, original[k])
		}
	}
	assert.Greater(t, moved, 0, "removing a node should move some keys")
}

// ──────────────────────────────────────────────
// Virtual node distribution
// ──────────────────────────────────────────────

func TestDistribution_Evenness(t *testing.T) {
	r := New(WithReplicas(200))
	r.Add("n1", "n2", "n3", "n4")

	dist := r.Distribution()
	require.Len(t, dist, 4)

	total := 0
	for _, count := range dist {
		total += count
	}
	assert.Equal(t, 200*4, total)

	// Each node should have close to 200 virtual nodes.
	for node, count := range dist {
		assert.InDelta(t, 200, count, 20, "node %s has %d virtual nodes", node, count)
	}
}

func TestDistribution_AfterRemove(t *testing.T) {
	r := New(WithReplicas(50))
	r.Add("a", "b", "c")
	r.Remove("b")

	dist := r.Distribution()
	assert.NotContains(t, dist, "b")
	assert.Equal(t, 50, dist["a"])
	assert.Equal(t, 50, dist["c"])
}

// ──────────────────────────────────────────────
// Snapshot
// ──────────────────────────────────────────────

func TestSnapshot_Independent(t *testing.T) {
	r := New()
	r.Add("a", "b", "c")

	snap := r.Snapshot()
	assert.Equal(t, 3, snap.Size())

	// Mutate original; snapshot should be unaffected.
	r.Add("d")
	assert.Equal(t, 4, r.Size())
	assert.Equal(t, 3, snap.Size(), "snapshot should not see later additions")
	assert.False(t, snap.Contains("d"))
}

func TestSnapshot_MutateSnapshot_DoesNotAffectOriginal(t *testing.T) {
	r := New()
	r.Add("a")

	snap := r.Snapshot()
	snap.Add("b")

	assert.True(t, snap.Contains("b"))
	assert.False(t, r.Contains("b"), "mutating snapshot should not affect original")
}

func TestSnapshot_GetConsistency(t *testing.T) {
	r := New()
	r.Add("a", "b", "c")

	snap := r.Snapshot()
	r.Remove("a")

	// Snapshot still has "a", so the key may or may not route there, but the
	// snapshot should remain usable and consistent.
	node, ok := snap.Get("some-key")
	require.True(t, ok)
	assert.Contains(t, []string{"a", "b", "c"}, node)
}

// ──────────────────────────────────────────────
// Concurrency
// ──────────────────────────────────────────────

func TestConcurrent_AddGet(t *testing.T) {
	r := New(WithReplicas(50))
	r.Add("n1", "n2", "n3")

	var wg sync.WaitGroup
	// Writers
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r.Add(fmt.Sprintf("extra-%d", i))
		}(i)
	}
	// Readers
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = r.Get(fmt.Sprintf("key-%d", i))
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 13, r.Size())
}

func TestConcurrent_GetN(t *testing.T) {
	r := New(WithReplicas(50))
	r.Add("a", "b", "c", "d")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, _ = r.GetN(fmt.Sprintf("k%d", i), 2)
		}(i)
	}
	wg.Wait()
}

// ──────────────────────────────────────────────
// Custom hash
// ──────────────────────────────────────────────

func TestCustomHash_Used(t *testing.T) {
	calls := 0
	custom := func(data []byte) uint32 {
		calls++
		return DefaultHash(data)
	}
	r := New(WithHash(custom))
	r.Add("a")
	// Add generates replicas+1 hash calls (replicas for virtual nodes). Reset
	// and check Get uses the custom hash too.
	calls = 0
	_, _ = r.Get("key")
	assert.Greater(t, calls, 0, "Get should use the custom hash")
}

// ──────────────────────────────────────────────
// itoa helper
// ──────────────────────────────────────────────

func TestItoa(t *testing.T) {
	assert.Equal(t, "0", itoa(0))
	assert.Equal(t, "1", itoa(1))
	assert.Equal(t, "42", itoa(42))
	assert.Equal(t, "12345", itoa(12345))
}

// ──────────────────────────────────────────────
// Statistical: standard deviation of key distribution
// ──────────────────────────────────────────────

func TestKeyDistribution_LowStdDev(t *testing.T) {
	r := New(WithReplicas(150))
	r.Add("n1", "n2", "n3", "n4", "n5")

	counts := make(map[string]int)
	const N = 50000
	for i := 0; i < N; i++ {
		node, ok := r.Get(fmt.Sprintf("key-%d", i))
		require.True(t, ok)
		counts[node]++
	}

	var values []float64
	for _, node := range r.Nodes() {
		values = append(values, float64(counts[node]))
	}
	mean := float64(N) / float64(len(values))
	var sumSq float64
	for _, v := range values {
		sumSq += math.Pow(v-mean, 2)
	}
	stddev := math.Sqrt(sumSq / float64(len(values)))
	// Standard deviation should be well under 20% of the mean.
	assert.Less(t, stddev, mean*0.2, "stddev %.1f too high vs mean %.1f", stddev, mean)
}
