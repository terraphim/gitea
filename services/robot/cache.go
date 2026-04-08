// Copyright 2026 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package robot

import (
	"sync"
	"time"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/setting"
)

// PageRankCache provides thread-safe caching for PageRank calculations
type PageRankCache struct {
	mu        sync.RWMutex
	entries   map[int64]*cacheEntry
	computing map[int64]bool
	ttl       time.Duration
}

// cacheEntry stores cached PageRank scores with metadata
type cacheEntry struct {
	scores    map[int64]float64
	computed  time.Time
	repoID    int64
}

// Global cache instance
var (
	globalCache     *PageRankCache
	globalCacheOnce sync.Once
)

// GetPageRankCache returns the singleton cache instance
func GetPageRankCache() *PageRankCache {
	globalCacheOnce.Do(func() {
		ttl := time.Duration(setting.IssueGraphSettings.PageRankCacheTTL) * time.Second
		if ttl <= 0 {
			ttl = 5 * time.Minute // Default 5 minutes
		}
		globalCache = NewPageRankCache(ttl)
		log.Info("PageRank cache initialized with TTL: %v", ttl)
	})
	return globalCache
}

// NewPageRankCache creates a new PageRank cache with the specified TTL
func NewPageRankCache(ttl time.Duration) *PageRankCache {
	return &PageRankCache{
		entries:   make(map[int64]*cacheEntry),
		computing: make(map[int64]bool),
		ttl:       ttl,
	}
}

// Get retrieves cached PageRank scores for a repository
// Returns (scores, true) if cache hit, (nil, false) if cache miss or expired
func (c *PageRankCache) Get(repoID int64) (map[int64]float64, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[repoID]
	if !exists {
		log.Trace("PageRank cache miss for repo %d: not in cache", repoID)
		return nil, false
	}

	// Check if expired
	if time.Since(entry.computed) > c.ttl {
		log.Trace("PageRank cache miss for repo %d: expired (computed %v)", repoID, entry.computed)
		return nil, false
	}

	log.Trace("PageRank cache hit for repo %d", repoID)
	return entry.scores, true
}

// Set stores PageRank scores in the cache
func (c *PageRankCache) Set(repoID int64, scores map[int64]float64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Make a copy of scores to prevent external modification
	scoresCopy := make(map[int64]float64, len(scores))
	for k, v := range scores {
		scoresCopy[k] = v
	}

	c.entries[repoID] = &cacheEntry{
		scores:   scoresCopy,
		computed: time.Now(),
		repoID:   repoID,
	}

	log.Trace("PageRank cached for repo %d (%d issues)", repoID, len(scoresCopy))
}

// IsComputing checks if PageRank is currently being computed for a repository
func (c *PageRankCache) IsComputing(repoID int64) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.computing[repoID]
}

// StartComputing marks a repository as being computed
// Returns true if successfully started, false if already computing
func (c *PageRankCache) StartComputing(repoID int64) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.computing[repoID] {
		return false
	}

	c.computing[repoID] = true
	return true
}

// FinishComputing marks computation as complete
func (c *PageRankCache) FinishComputing(repoID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.computing, repoID)
}

// Invalidate removes a repository's cached scores
func (c *PageRankCache) Invalidate(repoID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, repoID)
	delete(c.computing, repoID)
	log.Trace("PageRank cache invalidated for repo %d", repoID)
}

// InvalidateAll clears the entire cache
func (c *PageRankCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[int64]*cacheEntry)
	c.computing = make(map[int64]bool)
	log.Info("PageRank cache fully invalidated")
}

// GetStats returns cache statistics (for monitoring/debugging)
func (c *PageRankCache) GetStats() (entries int, computing int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries), len(c.computing)
}
