package repository

import (
	"fmt"
	"paradigm-reboot-prober-go/internal/model"
	"sort"
	"strings"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

// Default cache TTLs for each repository domain.
const (
	SongCacheTTL   = 10 * time.Minute
	UserCacheTTL   = 5 * time.Minute
	RecordCacheTTL = 5 * time.Minute
)

// repoCache is a type alias for the concrete cache type used across all repositories.
type repoCache = ttlcache.Cache[string, any]

// newRepoCache creates a new cache instance with the given default TTL
// and starts the automatic expired-item cleanup goroutine.
func newRepoCache(defaultTTL time.Duration) *repoCache {
	c := ttlcache.New[string, any](
		ttlcache.WithTTL[string, any](defaultTTL),
	)
	go c.Start() // non-blocking; runs until Stop() is called
	return c
}

// b50CacheEntry wraps the two-slice return value of GetBest50Records
// so it can be stored as a single value in the cache.
type b50CacheEntry struct {
	B35 []model.PlayRecord
	B15 []model.PlayRecord
}

// invalidateByPrefix collects all keys that start with prefix and deletes them.
func invalidateByPrefix(c *repoCache, prefix string) {
	var keys []string
	c.Range(func(item *ttlcache.Item[string, any]) bool {
		if strings.HasPrefix(item.Key(), prefix) {
			keys = append(keys, item.Key())
		}
		return true
	})
	for _, k := range keys {
		c.Delete(k)
	}
}

// ---------------------------------------------------------------------------
// Cache key builders — centralised so patterns stay consistent and typo-free.
// ---------------------------------------------------------------------------

// User keys
func userCacheKey(username string) string { return "user:" + username }

// Song / chart keys
func allSongsCacheKey() string { return "all_songs" }
func songIDCacheKey(songID int) string {
	return fmt.Sprintf("song:id:%d", songID)
}
func songWikiCacheKey(wikiID string) string { return "song:wiki:" + wikiID }
func chartIDCacheKey(chartID int) string {
	return fmt.Sprintf("chart:id:%d", chartID)
}
func chartWikiDiffCacheKey(wikiID string, diff model.Difficulty) string {
	return fmt.Sprintf("chart:wiki_diff:%s:%s", wikiID, diff)
}

// Record keys (all prefixed with username for per-user invalidation)
func b50CacheKey(username string, underflow int, filter model.RecordFilter) string {
	return fmt.Sprintf("%s:b50:%d:%s", username, underflow, filterCacheKey(filter))
}
func bestSongCacheKey(username string, songID int) string {
	return fmt.Sprintf("%s:best_song:%d", username, songID)
}
func bestChartCacheKey(username string, chartID int) string {
	return fmt.Sprintf("%s:best_chart:%d", username, chartID)
}
func allChartsCacheKey(username string, filter model.RecordFilter) string {
	return fmt.Sprintf("%s:all_charts:%s", username, filterCacheKey(filter))
}

// filterCacheKey returns a deterministic string representation of a RecordFilter
// for use as a cache key segment.
func filterCacheKey(f model.RecordFilter) string {
	if f.IsEmpty() {
		return "nofilter"
	}
	var parts []string
	if f.MinLevel != nil {
		parts = append(parts, fmt.Sprintf("min%.2f", *f.MinLevel))
	}
	if f.MaxLevel != nil {
		parts = append(parts, fmt.Sprintf("max%.2f", *f.MaxLevel))
	}
	if len(f.Difficulties) > 0 {
		// Sort difficulties for deterministic key ordering
		diffs := make([]string, len(f.Difficulties))
		for i, d := range f.Difficulties {
			diffs[i] = string(d)
		}
		sort.Strings(diffs)
		parts = append(parts, "diff:"+strings.Join(diffs, ","))
	}
	return strings.Join(parts, "_")
}
