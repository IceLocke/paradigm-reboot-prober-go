package repository

import (
	"fmt"
	"paradigm-reboot-prober-go/internal/model"
	"sort"
	"strings"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

// Default cache TTLs and cleanup intervals for each repository domain.
const (
	SongCacheTTL       = 10 * time.Minute
	SongCacheCleanup   = 15 * time.Minute
	UserCacheTTL       = 5 * time.Minute
	UserCacheCleanup   = 10 * time.Minute
	RecordCacheTTL     = 5 * time.Minute
	RecordCacheCleanup = 10 * time.Minute
)

// b50CacheEntry wraps the two-slice return value of GetBest50Records
// so it can be stored as a single interface{} in go-cache.
type b50CacheEntry struct {
	B35 []model.PlayRecord
	B15 []model.PlayRecord
}

// invalidateByPrefix iterates all cache items and deletes those whose key
// starts with the given prefix. Used for per-user record cache invalidation.
func invalidateByPrefix(c *gocache.Cache, prefix string) {
	for key := range c.Items() {
		if strings.HasPrefix(key, prefix) {
			c.Delete(key)
		}
	}
}

// ---------------------------------------------------------------------------
// Cache key builders — centralised so patterns stay consistent and typo-free.
// ---------------------------------------------------------------------------

// User keys
func userCacheKey(username string) string { return "user:" + username }

// Song / chart keys
func allSongsCacheKey() string              { return "all_songs" }
func songIDCacheKey(songID int) string      { return fmt.Sprintf("song:id:%d", songID) }
func songWikiCacheKey(wikiID string) string { return "song:wiki:" + wikiID }
func chartIDCacheKey(chartID int) string    { return fmt.Sprintf("chart:id:%d", chartID) }
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
