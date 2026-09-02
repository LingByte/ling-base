// Copyright (c) 2026 LingByte. All rights reserved.
// SPDX-License-Identifier: MIT

package antispam

import (
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ──────────────────────────────────────────────
// KeywordFilter
// ──────────────────────────────────────────────

func TestKeywordFilter_Add_Match(t *testing.T) {
	kf := NewKeywordFilter([]string{"spam", "scam", "bad"})
	assert.Equal(t, 3, kf.Count())

	matched := kf.Match("this is spam and scam")
	assert.ElementsMatch(t, []string{"spam", "scam"}, matched)
}

func TestKeywordFilter_Contains(t *testing.T) {
	kf := NewKeywordFilter([]string{"hello"})
	assert.True(t, kf.Contains("say hello world"))
	assert.False(t, kf.Contains("say hi world"))
}

func TestKeywordFilter_Replace(t *testing.T) {
	kf := NewKeywordFilter([]string{"bad", "word"})
	got := kf.Replace("this is bad word text", "***")
	assert.Equal(t, "this is *** *** text", got)
}

func TestKeywordFilter_Replace_EmptyReplacement(t *testing.T) {
	kf := NewKeywordFilter([]string{"bad"})
	got := kf.Replace("this is bad text", "")
	assert.Equal(t, "this is  text", got)
}

func TestKeywordFilter_AddMany_Dedup(t *testing.T) {
	kf := NewKeywordFilter(nil)
	kf.AddMany([]string{"a", "a", "b", ""})
	assert.Equal(t, 2, kf.Count(), "duplicates and empty strings ignored")
}

func TestKeywordFilter_Add_Existing(t *testing.T) {
	kf := NewKeywordFilter([]string{"key"})
	kf.Add("key")
	assert.Equal(t, 1, kf.Count(), "re-adding existing keyword is no-op")
}

func TestKeywordFilter_Add_Empty(t *testing.T) {
	kf := NewKeywordFilter(nil)
	kf.Add("")
	assert.Equal(t, 0, kf.Count())
}

func TestKeywordFilter_Remove(t *testing.T) {
	kf := NewKeywordFilter([]string{"spam", "scam", "bad"})
	kf.Remove("scam")
	assert.Equal(t, 2, kf.Count())
	assert.False(t, kf.Contains("this is a scam"))
	assert.True(t, kf.Contains("this is spam"))
}

func TestKeywordFilter_Remove_NonExistent(t *testing.T) {
	kf := NewKeywordFilter([]string{"spam"})
	kf.Remove("xyz")
	assert.Equal(t, 1, kf.Count())
	kf.Remove("")
	assert.Equal(t, 1, kf.Count())
}

func TestKeywordFilter_Remove_PrefixNotKeyword(t *testing.T) {
	kf := NewKeywordFilter([]string{"spam"})
	// "spa" is a path prefix but not a registered keyword
	kf.Remove("spa")
	assert.Equal(t, 1, kf.Count(), "removing a non-keyword prefix is a no-op")
	assert.True(t, kf.Contains("spam"))
}

func TestKeywordFilter_Remove_PrunesBranch(t *testing.T) {
	kf := NewKeywordFilter([]string{"abc"})
	kf.Remove("abc")
	assert.Equal(t, 0, kf.Count())
	assert.False(t, kf.Contains("abc"))
}

func TestKeywordFilter_Remove_KeepsSharedBranch(t *testing.T) {
	kf := NewKeywordFilter([]string{"abc", "abd"})
	kf.Remove("abc")
	assert.True(t, kf.Contains("abd"))
}

func TestKeywordFilter_Match_LongestKeyword(t *testing.T) {
	kf := NewKeywordFilter([]string{"a", "ab", "abc"})
	matched := kf.Match("abc")
	assert.Equal(t, []string{"abc"}, matched, "longest keyword should match")
}

func TestKeywordFilter_Match_Dedup(t *testing.T) {
	kf := NewKeywordFilter([]string{"spam"})
	matched := kf.Match("spam spam spam")
	assert.Equal(t, []string{"spam"}, matched, "duplicates removed")
}

func TestKeywordFilter_Match_NoMatch(t *testing.T) {
	kf := NewKeywordFilter([]string{"spam"})
	assert.Empty(t, kf.Match("clean text"))
}

func TestKeywordFilter_Match_EmptyText(t *testing.T) {
	kf := NewKeywordFilter([]string{"spam"})
	assert.Empty(t, kf.Match(""))
}

func TestKeywordFilter_Unicode(t *testing.T) {
	kf := NewKeywordFilter([]string{"赌博", "色情"})
	assert.True(t, kf.Contains("禁止赌博和色情"))
	matched := kf.Match("这里有赌博内容")
	assert.Equal(t, []string{"赌博"}, matched)
	got := kf.Replace("赌博网站", "**")
	assert.Equal(t, "**网站", got)
}

func TestKeywordFilter_Clear(t *testing.T) {
	kf := NewKeywordFilter([]string{"a", "b", "c"})
	kf.Clear()
	assert.Equal(t, 0, kf.Count())
	assert.False(t, kf.Contains("a"))
	// usable after clear
	kf.Add("x")
	assert.True(t, kf.Contains("x"))
}

func TestKeywordFilter_Concurrent(t *testing.T) {
	kf := NewKeywordFilter([]string{"init"})
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			kf.Add("kw")
			_ = kf.Contains("kw")
		}(i)
	}
	wg.Wait()
	assert.Equal(t, 2, kf.Count())
}

// ──────────────────────────────────────────────
// RateLimiter
// ──────────────────────────────────────────────

func TestRateLimiter_Allow_WithinLimit(t *testing.T) {
	rl := NewRateLimiter(time.Second, 3)
	for i := 0; i < 3; i++ {
		assert.True(t, rl.Allow("user1"))
	}
	assert.False(t, rl.Allow("user1"))
}

func TestRateLimiter_DifferentKeys(t *testing.T) {
	rl := NewRateLimiter(time.Second, 2)
	assert.True(t, rl.Allow("a"))
	assert.True(t, rl.Allow("a"))
	assert.False(t, rl.Allow("a"))
	// different key independent
	assert.True(t, rl.Allow("b"))
}

func TestRateLimiter_Count(t *testing.T) {
	rl := NewRateLimiter(time.Second, 5)
	rl.Allow("u")
	rl.Allow("u")
	assert.Equal(t, 2, rl.Count("u"))
	assert.Equal(t, 0, rl.Count("unknown"))
}

func TestRateLimiter_Reset(t *testing.T) {
	rl := NewRateLimiter(time.Second, 5)
	rl.Allow("u")
	rl.Allow("u")
	rl.Reset("u")
	assert.Equal(t, 0, rl.Count("u"))
	assert.True(t, rl.Allow("u"))
}

func TestRateLimiter_WindowExpiry(t *testing.T) {
	rl := NewRateLimiter(50*time.Millisecond, 2)
	assert.True(t, rl.Allow("u"))
	assert.True(t, rl.Allow("u"))
	assert.False(t, rl.Allow("u"))
	time.Sleep(60 * time.Millisecond)
	assert.True(t, rl.Allow("u"), "after window expires, requests allowed again")
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter(50*time.Millisecond, 5)
	rl.Allow("u")
	rl.Allow("u2")
	time.Sleep(60 * time.Millisecond)
	rl.Cleanup()
	assert.Equal(t, 0, rl.Count("u"))
	assert.Equal(t, 0, rl.Count("u2"))
}

func TestRateLimiter_Disabled(t *testing.T) {
	rl := NewRateLimiter(time.Second, 0)
	for i := 0; i < 100; i++ {
		assert.True(t, rl.Allow("u"), "maxCount<=0 disables the limiter")
	}
}

func TestRateLimiter_NegativeMaxCount(t *testing.T) {
	rl := NewRateLimiter(time.Second, -1)
	assert.True(t, rl.Allow("u"))
}

func TestRateLimiter_Concurrent(t *testing.T) {
	rl := NewRateLimiter(time.Second, 100)
	var wg sync.WaitGroup
	var allowed int
	var mu sync.Mutex
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if rl.Allow("shared") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	assert.Equal(t, 50, allowed)
}

// ──────────────────────────────────────────────
// ContentScorer
// ──────────────────────────────────────────────

func TestContentScorer_Empty(t *testing.T) {
	cs := NewContentScorer()
	assert.Equal(t, 0, cs.Score(""))
}

func TestContentScorer_CleanText(t *testing.T) {
	cs := NewContentScorer()
	score := cs.Score("Hello, this is a normal message about the weather today.")
	assert.True(t, score < spamThreshold, "clean text should not be spam, got %d", score)
	assert.False(t, cs.IsSpam(score))
}

func TestContentScorer_RepeatedChars(t *testing.T) {
	cs := NewContentScorer()
	score := cs.Score("aaaaaaaaaaaaa")
	assert.True(t, score > 0, "repeated chars should add score")
}

func TestContentScorer_ConsecutivePunct(t *testing.T) {
	cs := NewContentScorer()
	score := cs.Score("wow!!!!!!!?????? amazing")
	assert.True(t, score > 0, "consecutive punctuation should add score")
}

func TestContentScorer_URLs(t *testing.T) {
	cs := NewContentScorer()
	score := cs.Score("buy now http://spam.com http://scam.com http://fake.com http://bad.com")
	assert.True(t, score >= 20, "many URLs should add significant score, got %d", score)
}

func TestContentScorer_UppercaseRatio(t *testing.T) {
	cs := NewContentScorer()
	score := cs.Score("HELLO WORLD THIS IS ALL CAPS SHOUTING")
	assert.True(t, score > 0, "high uppercase ratio should add score")
}

func TestContentScorer_SpecialChars(t *testing.T) {
	cs := NewContentScorer()
	score := cs.Score("@#$%^&*()@#$%^&*()@#$%^&*()")
	assert.True(t, score > 0, "high special char ratio should add score")
}

func TestContentScorer_SpamThreshold(t *testing.T) {
	cs := NewContentScorer()
	// Combine multiple spam signals to exceed threshold.
	score := cs.Score("BUY NOW!!! http://spam.com http://scam.com @#$% AAAAAAAA")
	assert.True(t, cs.IsSpam(score), "combined spam signals should exceed threshold, got %d", score)
}

func TestContentScorer_IsSpam_Boundary(t *testing.T) {
	cs := NewContentScorer()
	assert.True(t, cs.IsSpam(60))
	assert.True(t, cs.IsSpam(100))
	assert.False(t, cs.IsSpam(59))
	assert.False(t, cs.IsSpam(0))
}

func TestContentScorer_CappedAt100(t *testing.T) {
	cs := NewContentScorer()
	score := cs.Score("AAAAAAAAAAAAAAAA!!!!!!!!! @#$%^&*() http://x.com http://y.com http://z.com BUY NOW FREE")
	assert.LessOrEqual(t, score, 100)
}

func TestContentScorer_Deterministic(t *testing.T) {
	cs := NewContentScorer()
	text := "BUY NOW!!! http://spam.com AAAAAAAA"
	assert.Equal(t, cs.Score(text), cs.Score(text))
}

// ──────────────────────────────────────────────
// Checker
// ──────────────────────────────────────────────

func TestChecker_PassesClean(t *testing.T) {
	c := NewChecker()
	res := c.Check("Hello, how are you today?", "user1")
	assert.True(t, res.Passed)
	assert.Empty(t, res.Reasons)
}

func TestChecker_WithKeywords(t *testing.T) {
	c := NewChecker(WithKeywords([]string{"spam", "scam"}))
	res := c.Check("this is spam", "user1")
	assert.False(t, res.Passed)
	assert.Contains(t, res.Reasons, "contains banned keywords")
	assert.Equal(t, []string{"spam"}, res.MatchedKeywords)
}

func TestChecker_WithKeywords_NoMatch(t *testing.T) {
	c := NewChecker(WithKeywords([]string{"spam"}))
	res := c.Check("clean text", "user1")
	assert.True(t, res.Passed)
	assert.Empty(t, res.MatchedKeywords)
}

func TestChecker_WithRateLimit(t *testing.T) {
	c := NewChecker(WithRateLimit(time.Second, 2))
	assert.True(t, c.Check("hi", "u").Passed)
	assert.True(t, c.Check("hi", "u").Passed)
	res := c.Check("hi", "u")
	assert.False(t, res.Passed)
	assert.Contains(t, res.Reasons, "rate limit exceeded")
}

func TestChecker_WithMinLength(t *testing.T) {
	c := NewChecker(WithMinLength(5))
	res := c.Check("hi", "u")
	assert.False(t, res.Passed)
	assert.Contains(t, res.Reasons, "content too short")

	res = c.Check("hello there", "u")
	assert.True(t, res.Passed)
}

func TestChecker_WithMaxLength(t *testing.T) {
	c := NewChecker(WithMaxLength(5))
	res := c.Check("hello world this is long", "u")
	assert.False(t, res.Passed)
	assert.Contains(t, res.Reasons, "content too long")

	res = c.Check("hi", "u")
	assert.True(t, res.Passed)
}

func TestChecker_WithBannedPatterns(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)buy\s+now`)
	c := NewChecker(WithBannedPatterns([]*regexp.Regexp{pattern}))
	res := c.Check("BUY NOW cheap", "u")
	assert.False(t, res.Passed)
	found := false
	for _, r := range res.Reasons {
		if len(r) >= len("matches banned pattern") && r[:len("matches banned pattern")] == "matches banned pattern" {
			found = true
		}
	}
	assert.True(t, found, "should contain a 'matches banned pattern' reason, got %v", res.Reasons)
}

func TestChecker_WithBannedPatterns_NoMatch(t *testing.T) {
	pattern := regexp.MustCompile(`(?i)buy\s+now`)
	c := NewChecker(WithBannedPatterns([]*regexp.Regexp{pattern}))
	res := c.Check("hello there", "u")
	assert.True(t, res.Passed)
}

func TestChecker_SpamScore_Fails(t *testing.T) {
	c := NewChecker()
	res := c.Check("BUY NOW!!! http://spam.com http://scam.com @#$% AAAAAAAA", "u")
	assert.False(t, res.Passed)
	assert.Contains(t, res.Reasons, "content flagged as spam")
	assert.True(t, res.Score >= 60)
}

func TestChecker_MultipleReasons(t *testing.T) {
	c := NewChecker(
		WithKeywords([]string{"spam"}),
		WithMinLength(10),
	)
	// short AND contains keyword
	res := c.Check("spam", "u")
	assert.False(t, res.Passed)
	assert.Contains(t, res.Reasons, "content too short")
	assert.Contains(t, res.Reasons, "contains banned keywords")
}

func TestChecker_NoOptions(t *testing.T) {
	c := NewChecker()
	// clean short text passes (no min length, no keywords, no rate limit)
	res := c.Check("ok", "u")
	assert.True(t, res.Passed)
}

func TestChecker_MatchedKeywordsPopulatedEvenWhenPassed(t *testing.T) {
	// When keywords match, it always fails; but verify field is populated.
	c := NewChecker(WithKeywords([]string{"spam"}))
	res := c.Check("spam", "u")
	assert.Equal(t, []string{"spam"}, res.MatchedKeywords)
}

func TestChecker_RateLimitPerUser(t *testing.T) {
	c := NewChecker(WithRateLimit(time.Second, 1))
	assert.True(t, c.Check("hi", "userA").Passed)
	assert.True(t, c.Check("hi", "userB").Passed, "different user has own limit")
	res := c.Check("hi", "userA")
	assert.False(t, res.Passed)
}
