// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package robot

import (
	"testing"
	"time"
)

func TestPageRankCache_Get(t *testing.T) {
	cache := NewPageRankCache(5 * time.Minute)

	// Test cache miss
	scores, ok := cache.Get(1)
	if ok {
		t.Error("Expected cache miss for empty cache")
	}
	if scores != nil {
		t.Error("Expected nil scores for cache miss")
	}

	// Set cache entry
	testScores := map[int64]float64{
		1: 0.5,
		2: 0.3,
		3: 0.2,
	}
	cache.Set(1, testScores)

	// Test cache hit
	scores, ok = cache.Get(1)
	if !ok {
		t.Error("Expected cache hit after Set")
	}
	if scores == nil {
		t.Error("Expected non-nil scores for cache hit")
	}
	if scores[1] != 0.5 {
		t.Errorf("Expected score 0.5, got %f", scores[1])
	}

	// Test cache miss for different repo
	scores, ok = cache.Get(2)
	if ok {
		t.Error("Expected cache miss for different repo")
	}
}

func TestPageRankCache_Expiration(t *testing.T) {
	// Create cache with very short TTL
	cache := NewPageRankCache(1 * time.Millisecond)

	// Set cache entry
	testScores := map[int64]float64{1: 0.5}
	cache.Set(1, testScores)

	// Should be a hit immediately
	_, ok := cache.Get(1)
	if !ok {
		t.Error("Expected cache hit immediately after Set")
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Should be a miss after expiration
	_, ok = cache.Get(1)
	if ok {
		t.Error("Expected cache miss after TTL expiration")
	}
}

func TestPageRankCache_Computing(t *testing.T) {
	cache := NewPageRankCache(5 * time.Minute)

	// Test start computing
	if !cache.StartComputing(1) {
		t.Error("Expected StartComputing to return true for new repo")
	}

	// Test concurrent computing prevention
	if cache.StartComputing(1) {
		t.Error("Expected StartComputing to return false when already computing")
	}

	// Test finish computing
	cache.FinishComputing(1)

	// Should be able to start again
	if !cache.StartComputing(1) {
		t.Error("Expected StartComputing to return true after FinishComputing")
	}
}

func TestPageRankCache_ConcurrentAccess(t *testing.T) {
	cache := NewPageRankCache(5 * time.Minute)

	// Simulate concurrent reads and writes
	done := make(chan bool, 3)

	// Writer
	go func() {
		for i := 0; i < 100; i++ {
			cache.Set(1, map[int64]float64{int64(i): float64(i)})
		}
		done <- true
	}()

	// Reader 1
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get(1)
		}
		done <- true
	}()

	// Reader 2
	go func() {
		for i := 0; i < 100; i++ {
			cache.Get(1)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// If we get here without panic, concurrent access is safe
}

func TestPageRankCache_Invalidate(t *testing.T) {
	cache := NewPageRankCache(5 * time.Minute)

	// Set cache entries
	cache.Set(1, map[int64]float64{1: 0.5})
	cache.Set(2, map[int64]float64{1: 0.3})

	// Verify entries exist
	if _, ok := cache.Get(1); !ok {
		t.Error("Expected repo 1 to be in cache")
	}

	// Invalidate single repo
	cache.Invalidate(1)

	// Verify repo 1 is gone
	if _, ok := cache.Get(1); ok {
		t.Error("Expected repo 1 to be invalidated")
	}

	// Verify repo 2 still exists
	if _, ok := cache.Get(2); !ok {
		t.Error("Expected repo 2 to still be in cache")
	}
}

func TestPageRankCache_InvalidateAll(t *testing.T) {
	cache := NewPageRankCache(5 * time.Minute)

	// Set multiple cache entries
	cache.Set(1, map[int64]float64{1: 0.5})
	cache.Set(2, map[int64]float64{1: 0.3})
	cache.Set(3, map[int64]float64{1: 0.2})

	// Mark one as computing
	cache.StartComputing(4)

	// Invalidate all
	cache.InvalidateAll()

	// Verify all entries are gone
	for i := int64(1); i <= 4; i++ {
		if _, ok := cache.Get(i); ok {
			t.Errorf("Expected repo %d to be invalidated", i)
		}
	}

	// Verify computing map is also cleared
	if cache.IsComputing(4) {
		t.Error("Expected computing state to be cleared")
	}
}

func TestPageRankCache_GetStats(t *testing.T) {
	cache := NewPageRankCache(5 * time.Minute)

	// Initial state
	entries, computing := cache.GetStats()
	if entries != 0 || computing != 0 {
		t.Errorf("Expected 0 entries and 0 computing, got %d and %d", entries, computing)
	}

	// Add entries
	cache.Set(1, map[int64]float64{1: 0.5})
	cache.Set(2, map[int64]float64{1: 0.3})
	cache.StartComputing(3)

	entries, computing = cache.GetStats()
	if entries != 2 {
		t.Errorf("Expected 2 entries, got %d", entries)
	}
	if computing != 1 {
		t.Errorf("Expected 1 computing, got %d", computing)
	}
}

func TestPageRankCache_ScoreIsolation(t *testing.T) {
	cache := NewPageRankCache(5 * time.Minute)

	// Set initial scores
	originalScores := map[int64]float64{1: 0.5, 2: 0.3}
	cache.Set(1, originalScores)

	// Modify the original map
	originalScores[1] = 0.9

	// Retrieve from cache
	cachedScores, _ := cache.Get(1)

	// Verify cache wasn't affected by external modification
	if cachedScores[1] != 0.5 {
		t.Errorf("Expected cached score 0.5, got %f (scores not isolated)", cachedScores[1])
	}
}

func BenchmarkPageRankCache_Get(b *testing.B) {
	cache := NewPageRankCache(5 * time.Minute)
	cache.Set(1, map[int64]float64{1: 0.5, 2: 0.3, 3: 0.2})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get(1)
	}
}

func BenchmarkPageRankCache_Set(b *testing.B) {
	cache := NewPageRankCache(5 * time.Minute)
	scores := map[int64]float64{1: 0.5, 2: 0.3, 3: 0.2}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(int64(i), scores)
	}
}
