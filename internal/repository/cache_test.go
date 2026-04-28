package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Song Cache Consistency
// ---------------------------------------------------------------------------

func TestSongCacheConsistency(t *testing.T) {
	t.Run("GetAllSongs is cached and invalidated on CreateSong", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		// Seed a song
		song := &model.Song{
			SongBase: model.SongBase{Title: "Song1", Artist: "Artist1", Version: "1.0", WikiID: "song1"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyDetected, Level: 5.0}},
		}
		_, err := repo.CreateSong(song)
		assert.NoError(t, err)

		// First read → cache miss → DB query → populate cache
		songs, err := repo.GetAllSongs()
		assert.NoError(t, err)
		assert.Len(t, songs, 1)

		// Verify entry is in cache
		assert.True(t, repo.cache.Has(allSongsCacheKey()), "all_songs should be cached after first read")

		// Second read → cache hit → same data
		songs2, err := repo.GetAllSongs()
		assert.NoError(t, err)
		assert.Len(t, songs2, 1)
		assert.Equal(t, songs[0].Title, songs2[0].Title)

		// Create a new song → cache should be flushed
		song2 := &model.Song{
			SongBase: model.SongBase{Title: "Song2", Artist: "Artist2", Version: "1.0", WikiID: "song2"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 10.0}},
		}
		_, err = repo.CreateSong(song2)
		assert.NoError(t, err)

		// Verify cache was flushed
		assert.False(t, repo.cache.Has(allSongsCacheKey()), "all_songs cache should be flushed after CreateSong")

		// Re-read → should return 2 songs from DB
		songs3, err := repo.GetAllSongs()
		assert.NoError(t, err)
		assert.Len(t, songs3, 2)
	})

	t.Run("GetSongByID cached and invalidated on UpdateSong", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		song := &model.Song{
			SongBase: model.SongBase{Title: "Original", Artist: "A", Version: "1.0", WikiID: "sid"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyDetected, Level: 5.0}},
		}
		created, err := repo.CreateSong(song)
		assert.NoError(t, err)

		// Read by ID → cached
		got, err := repo.GetSongByID(created.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Original", got.Title)
		assert.True(t, repo.cache.Has(songIDCacheKey(1)))

		// Update song title
		updatedSong := &model.Song{
			SongBase: model.SongBase{Title: "Updated", Artist: "A", Version: "1.0", WikiID: "sid"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyDetected, Level: 5.0}},
		}
		_, err = repo.UpdateSong(created.ID, updatedSong)
		assert.NoError(t, err)

		// Cache should be flushed
		assert.False(t, repo.cache.Has(songIDCacheKey(1)), "song cache should be flushed after UpdateSong")

		// Re-read → should return updated data
		got2, err := repo.GetSongByID(created.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Updated", got2.Title)
	})

	t.Run("GetSongByWikiID cached and flushed on CreateSong", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		song := &model.Song{
			SongBase: model.SongBase{Title: "Wiki", Artist: "A", Version: "1.0", WikiID: "mywiki"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 8.0}},
		}
		_, err := repo.CreateSong(song)
		assert.NoError(t, err)

		// Read by wiki ID → cached
		got, err := repo.GetSongByWikiID("mywiki")
		assert.NoError(t, err)
		assert.NotNil(t, got)
		assert.True(t, repo.cache.Has(songWikiCacheKey("mywiki")))

		// Create another song → cache flushed
		song2 := &model.Song{
			SongBase: model.SongBase{Title: "Other", Artist: "B", Version: "1.0", WikiID: "other"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyDetected, Level: 3.0}},
		}
		_, err = repo.CreateSong(song2)
		assert.NoError(t, err)

		assert.False(t, repo.cache.Has(songWikiCacheKey("mywiki")), "wiki cache should be flushed after CreateSong")
	})

	t.Run("GetChartByID cached and flushed on UpdateSong", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		song := &model.Song{
			SongBase: model.SongBase{Title: "ChartTest", Artist: "A", Version: "1.0", WikiID: "ct"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyInvaded, Level: 7.0}},
		}
		created, err := repo.CreateSong(song)
		assert.NoError(t, err)
		chartID := created.Charts[0].ID

		// Read chart by ID → cached
		chart, err := repo.GetChartByID(chartID)
		assert.NoError(t, err)
		assert.NotNil(t, chart)
		assert.Equal(t, 7.0, chart.Level)

		// Update song with new chart level
		updatedSong := &model.Song{
			SongBase: model.SongBase{Title: "ChartTest", Artist: "A", Version: "1.0", WikiID: "ct"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyInvaded, Level: 9.0}},
		}
		_, err = repo.UpdateSong(created.ID, updatedSong)
		assert.NoError(t, err)

		// Re-read chart → should have updated level
		chart2, err := repo.GetChartByID(chartID)
		assert.NoError(t, err)
		assert.NotNil(t, chart2)
		assert.Equal(t, 9.0, chart2.Level)
	})

	t.Run("GetChartByWikiIDAndDifficulty cached", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		song := &model.Song{
			SongBase: model.SongBase{Title: "WikiDiff", Artist: "A", Version: "1.0", WikiID: "wd"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 12.0}},
		}
		_, err := repo.CreateSong(song)
		assert.NoError(t, err)

		// Read → cached
		chart, err := repo.GetChartByWikiIDAndDifficulty("wd", model.DifficultyMassive)
		assert.NoError(t, err)
		assert.NotNil(t, chart)

		key := chartWikiDiffCacheKey("wd", model.DifficultyMassive)
		assert.True(t, repo.cache.Has(key), "chart wiki_diff lookup should be cached")

		// Read again → cache hit
		chart2, err := repo.GetChartByWikiIDAndDifficulty("wd", model.DifficultyMassive)
		assert.NoError(t, err)
		assert.Equal(t, chart.ID, chart2.ID)
	})
}

// ---------------------------------------------------------------------------
// User Cache Consistency
// ---------------------------------------------------------------------------

func TestUserCacheConsistency(t *testing.T) {
	t.Run("GetUserByUsername cached and invalidated on UpdateUser", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)

		user := &model.User{
			UserBase:        model.UserBase{Username: "cacheuser", Nickname: "OldNick", UploadToken: "tok1", IsActive: true},
			EncodedPassword: "pass",
		}
		_, err := repo.CreateUser(user)
		assert.NoError(t, err)

		// Read → cache miss → stored in cache
		got, err := repo.GetUserByUsername("cacheuser")
		assert.NoError(t, err)
		assert.Equal(t, "OldNick", got.Nickname)
		assert.True(t, repo.cache.Has(userCacheKey("cacheuser")), "user should be cached after GetUserByUsername")

		// Read again → cache hit
		got2, err := repo.GetUserByUsername("cacheuser")
		assert.NoError(t, err)
		assert.Equal(t, "OldNick", got2.Nickname)

		// Update nickname
		got.Nickname = "NewNick"
		_, err = repo.UpdateUser(got)
		assert.NoError(t, err)

		// Cache should be invalidated
		assert.False(t, repo.cache.Has(userCacheKey("cacheuser")), "user cache should be invalidated after UpdateUser")

		// Re-read → fresh from DB
		got3, err := repo.GetUserByUsername("cacheuser")
		assert.NoError(t, err)
		assert.Equal(t, "NewNick", got3.Nickname)
	})

	t.Run("CreateUser does not pollute other users cache", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)

		// Create and cache user1
		user1 := &model.User{
			UserBase:        model.UserBase{Username: "useronexx", Nickname: "One", UploadToken: "t1", IsActive: true},
			EncodedPassword: "p1",
		}
		_, err := repo.CreateUser(user1)
		assert.NoError(t, err)

		_, err = repo.GetUserByUsername("useronexx")
		assert.NoError(t, err)
		assert.True(t, repo.cache.Has(userCacheKey("useronexx")))

		// Create user2 → user1's cache entry should still exist
		user2 := &model.User{
			UserBase:        model.UserBase{Username: "usertwoxx", Nickname: "Two", UploadToken: "t2", IsActive: true},
			EncodedPassword: "p2",
		}
		_, err = repo.CreateUser(user2)
		assert.NoError(t, err)

		assert.True(t, repo.cache.Has(userCacheKey("useronexx")), "user1 cache should not be affected by user2 creation")
	})

	t.Run("WithTransaction shares cache for invalidation", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)

		user := &model.User{
			UserBase:        model.UserBase{Username: "txuserxx", Nickname: "Before", UploadToken: "tx1", IsActive: true},
			EncodedPassword: "pass",
		}
		_, err := repo.CreateUser(user)
		assert.NoError(t, err)

		// Cache the user
		_, err = repo.GetUserByUsername("txuserxx")
		assert.NoError(t, err)
		assert.True(t, repo.cache.Has(userCacheKey("txuserxx")))

		// Update within transaction
		err = repo.WithTransaction(func(txRepo *UserRepository) error {
			u, err := txRepo.GetUserByUsername("txuserxx")
			if err != nil {
				return err
			}
			u.Nickname = "After"
			_, err = txRepo.UpdateUser(u)
			return err
		})
		assert.NoError(t, err)

		// Cache should be invalidated by the TX
		assert.False(t, repo.cache.Has(userCacheKey("txuserxx")), "cache should be invalidated after TX commit with UpdateUser")

		// Read fresh data
		got, err := repo.GetUserByUsername("txuserxx")
		assert.NoError(t, err)
		assert.Equal(t, "After", got.Nickname)
	})

	t.Run("WithTransaction rollback still invalidates cache pessimistically", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)

		user := &model.User{
			UserBase:        model.UserBase{Username: "rolluser", Nickname: "Original", UploadToken: "ru1", IsActive: true},
			EncodedPassword: "pass",
		}
		_, err := repo.CreateUser(user)
		assert.NoError(t, err)

		// Cache the user
		_, err = repo.GetUserByUsername("rolluser")
		assert.NoError(t, err)

		// Transaction that updates then rolls back
		txErr := repo.WithTransaction(func(txRepo *UserRepository) error {
			u, err := txRepo.GetUserByUsername("rolluser")
			if err != nil {
				return err
			}
			u.Nickname = "ShouldNotPersist"
			_, err = txRepo.UpdateUser(u) // invalidates cache
			if err != nil {
				return err
			}
			return errors.New("forced rollback")
		})
		assert.Error(t, txErr)

		// Cache was invalidated pessimistically, re-read gives original data
		got, err := repo.GetUserByUsername("rolluser")
		assert.NoError(t, err)
		assert.Equal(t, "Original", got.Nickname, "after rollback, original data should be returned")
	})

	t.Run("Shallow copy prevents cache mutation", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)

		user := &model.User{
			UserBase:        model.UserBase{Username: "copyuser", Nickname: "Immutable", UploadToken: "cu1", IsActive: true},
			EncodedPassword: "pass",
		}
		_, err := repo.CreateUser(user)
		assert.NoError(t, err)

		// Cache the user
		got1, err := repo.GetUserByUsername("copyuser")
		assert.NoError(t, err)

		// Mutate the returned copy
		got1.Nickname = "Mutated"

		// Read again → should still get original cached value
		got2, err := repo.GetUserByUsername("copyuser")
		assert.NoError(t, err)
		assert.Equal(t, "Immutable", got2.Nickname, "cached value should not be affected by caller mutation")
	})
}

// ---------------------------------------------------------------------------
// Record Cache Consistency
// ---------------------------------------------------------------------------

func TestRecordCacheConsistency(t *testing.T) {
	// Helper to create test fixtures: user, song, chart
	setupFixtures := func(t *testing.T, db *gorm.DB) (*UserRepository, *SongRepository, *RecordRepository, *model.Song) {
		t.Helper()
		userRepo := NewUserRepository(db)
		songRepo := NewSongRepository(db)
		recordRepo := NewRecordRepository(db)

		user := &model.User{
			UserBase:        model.UserBase{Username: "recuser1", Nickname: "Rec", UploadToken: "rt1", IsActive: true},
			EncodedPassword: "p",
		}
		_, err := userRepo.CreateUser(user)
		assert.NoError(t, err)

		song := &model.Song{
			SongBase: model.SongBase{Title: "RecSong", Artist: "A", Version: "1.0", WikiID: "recsong", B15: false},
			Charts: []model.Chart{
				{Difficulty: model.DifficultyDetected, Level: 5.0},
				{Difficulty: model.DifficultyMassive, Level: 10.0},
			},
		}
		created, err := songRepo.CreateSong(song)
		assert.NoError(t, err)

		return userRepo, songRepo, recordRepo, created
	}

	t.Run("B50 cached and invalidated on CreateRecord", func(t *testing.T) {
		db := setupTestDB(t)
		_, _, recordRepo, song := setupFixtures(t, db)
		chartID := song.Charts[0].ID
		score := 950000

		// Upload a record
		_, err := recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		// Query B50 → cached
		b35, b15, err := recordRepo.GetBest50Records("recuser1", 0, model.RecordFilter{})
		assert.NoError(t, err)
		assert.Len(t, b35, 1) // song is not b15
		assert.Len(t, b15, 0)

		key := b50CacheKey("recuser1", 0, model.RecordFilter{})
		assert.True(t, recordRepo.cache.Has(key), "B50 should be cached")

		// Upload another record → cache invalidated
		score2 := 1000000
		_, err = recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: song.Charts[1].ID, Score: &score2},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		assert.False(t, recordRepo.cache.Has(key), "B50 cache should be invalidated after CreateRecord")

		// Re-query → should have 2 records
		b35v2, _, err := recordRepo.GetBest50Records("recuser1", 0, model.RecordFilter{})
		assert.NoError(t, err)
		assert.Len(t, b35v2, 2)
	})

	t.Run("BestRecordByChart cached and invalidated on higher score", func(t *testing.T) {
		db := setupTestDB(t)
		_, _, recordRepo, song := setupFixtures(t, db)
		chartID := song.Charts[0].ID
		score := 900000

		_, err := recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		// Query best by chart → cached
		best, err := recordRepo.GetBestRecordByChart("recuser1", chartID)
		assert.NoError(t, err)
		assert.NotNil(t, best)
		assert.Equal(t, 900000, *best.Score)

		key := bestChartCacheKey("recuser1", chartID)
		assert.True(t, recordRepo.cache.Has(key))

		// Upload higher score
		score2 := 980000
		_, err = recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score2},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		assert.False(t, recordRepo.cache.Has(key), "best_chart cache should be invalidated after new record")

		// Re-query → should have higher score
		best2, err := recordRepo.GetBestRecordByChart("recuser1", chartID)
		assert.NoError(t, err)
		assert.NotNil(t, best2)
		assert.Equal(t, 980000, *best2.Score)
	})

	t.Run("BestRecordsBySong cached and invalidated", func(t *testing.T) {
		db := setupTestDB(t)
		_, _, recordRepo, song := setupFixtures(t, db)
		chartID1 := song.Charts[0].ID
		score := 950000

		_, err := recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID1, Score: &score},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		// Query best by song → cached
		bests, err := recordRepo.GetBestRecordsBySong("recuser1", song.ID)
		assert.NoError(t, err)
		assert.Len(t, bests, 1)

		// Upload record for another chart of the same song → cache invalidated
		score2 := 880000
		_, err = recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: song.Charts[1].ID, Score: &score2},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		// Re-query → should have 2 best records (one per difficulty)
		bests2, err := recordRepo.GetBestRecordsBySong("recuser1", song.ID)
		assert.NoError(t, err)
		assert.Len(t, bests2, 2)
	})

	t.Run("Cross-user isolation", func(t *testing.T) {
		db := setupTestDB(t)
		userRepo := NewUserRepository(db)
		songRepo := NewSongRepository(db)
		recordRepo := NewRecordRepository(db)

		// Create two users
		for _, u := range []string{"xusraaaa", "xusrbbbb"} {
			_, err := userRepo.CreateUser(&model.User{
				UserBase:        model.UserBase{Username: u, Nickname: u, UploadToken: u + "_tok", IsActive: true},
				EncodedPassword: "p",
			})
			assert.NoError(t, err)
		}

		song := &model.Song{
			SongBase: model.SongBase{Title: "Iso", Artist: "A", Version: "1.0", WikiID: "iso"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyDetected, Level: 6.0}},
		}
		created, err := songRepo.CreateSong(song)
		assert.NoError(t, err)
		chartID := created.Charts[0].ID

		// Both users upload records
		for _, u := range []string{"xusraaaa", "xusrbbbb"} {
			score := 900000
			_, err := recordRepo.CreateRecord(&model.PlayRecord{
				PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score},
				Username:       u,
			}, false)
			assert.NoError(t, err)
		}

		// Cache B50 for both users
		_, _, err = recordRepo.GetBest50Records("xusraaaa", 0, model.RecordFilter{})
		assert.NoError(t, err)
		_, _, err = recordRepo.GetBest50Records("xusrbbbb", 0, model.RecordFilter{})
		assert.NoError(t, err)

		keyA := b50CacheKey("xusraaaa", 0, model.RecordFilter{})
		keyB := b50CacheKey("xusrbbbb", 0, model.RecordFilter{})
		assert.True(t, recordRepo.cache.Has(keyA))
		assert.True(t, recordRepo.cache.Has(keyB))

		// Upload a new record for user A only
		scoreNew := 1000000
		_, err = recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &scoreNew},
			Username:       "xusraaaa",
		}, false)
		assert.NoError(t, err)

		// User A's cache invalidated, user B's intact
		assert.False(t, recordRepo.cache.Has(keyA), "user A's B50 cache should be invalidated")
		assert.True(t, recordRepo.cache.Has(keyB), "user B's B50 cache should remain intact")
	})

	t.Run("BatchCreateRecords invalidates all affected users", func(t *testing.T) {
		db := setupTestDB(t)
		userRepo := NewUserRepository(db)
		songRepo := NewSongRepository(db)
		recordRepo := NewRecordRepository(db)

		for _, u := range []string{"batchusr1", "batchusr2"} {
			_, err := userRepo.CreateUser(&model.User{
				UserBase:        model.UserBase{Username: u, Nickname: u, UploadToken: u + "_tok", IsActive: true},
				EncodedPassword: "p",
			})
			assert.NoError(t, err)
		}

		song := &model.Song{
			SongBase: model.SongBase{Title: "Batch", Artist: "A", Version: "1.0", WikiID: "batch"},
			Charts:   []model.Chart{{Difficulty: model.DifficultyMassive, Level: 11.0}},
		}
		created, err := songRepo.CreateSong(song)
		assert.NoError(t, err)
		chartID := created.Charts[0].ID

		// Seed initial records so B50 has data
		for _, u := range []string{"batchusr1", "batchusr2"} {
			score := 800000
			_, err := recordRepo.CreateRecord(&model.PlayRecord{
				PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score},
				Username:       u,
			}, false)
			assert.NoError(t, err)
		}

		// Cache B50 for both
		_, _, err = recordRepo.GetBest50Records("batchusr1", 0, model.RecordFilter{})
		assert.NoError(t, err)
		_, _, err = recordRepo.GetBest50Records("batchusr2", 0, model.RecordFilter{})
		assert.NoError(t, err)

		// Batch create for user batchusr1 only
		score := 999000
		_, err = recordRepo.BatchCreateRecords([]*model.PlayRecord{
			{PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score}, Username: "batchusr1"},
		}, false)
		assert.NoError(t, err)

		// batchusr1 cache invalidated
		assert.False(t, recordRepo.cache.Has(b50CacheKey("batchusr1", 0, model.RecordFilter{})),
			"batchusr1 B50 cache should be invalidated after batch create")

		// batchusr2 cache intact
		assert.True(t, recordRepo.cache.Has(b50CacheKey("batchusr2", 0, model.RecordFilter{})),
			"batchusr2 B50 cache should remain intact")
	})

	t.Run("AllChartsWithBestScores cached and invalidated", func(t *testing.T) {
		db := setupTestDB(t)
		_, _, recordRepo, song := setupFixtures(t, db)
		chartID := song.Charts[0].ID
		score := 950000

		_, err := recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: chartID, Score: &score},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		// Query all charts with scores → cached
		results, err := recordRepo.GetAllChartsWithBestScores("recuser1", model.RecordFilter{})
		assert.NoError(t, err)
		assert.NotEmpty(t, results)

		key := allChartsCacheKey("recuser1", model.RecordFilter{})
		assert.True(t, recordRepo.cache.Has(key), "all_charts should be cached")

		// Upload new record → cache invalidated
		score2 := 1000000
		_, err = recordRepo.CreateRecord(&model.PlayRecord{
			PlayRecordBase: model.PlayRecordBase{ChartID: song.Charts[1].ID, Score: &score2},
			Username:       "recuser1",
		}, false)
		assert.NoError(t, err)

		assert.False(t, recordRepo.cache.Has(key), "all_charts cache should be invalidated after CreateRecord")
	})
}

// ---------------------------------------------------------------------------
// Cache Miss/Hit & Nil Handling
// ---------------------------------------------------------------------------

func TestCacheMissAndHit(t *testing.T) {
	t.Run("Nil result for non-existent user is not cached", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewUserRepository(db)

		got, err := repo.GetUserByUsername("ghost")
		assert.NoError(t, err)
		assert.Nil(t, got)

		// nil results should NOT be stored in cache
		assert.False(t, repo.cache.Has(userCacheKey("ghost")), "nil result should not be cached")
	})

	t.Run("Nil result for non-existent song is not cached", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		got, err := repo.GetSongByID(9999)
		assert.NoError(t, err)
		assert.Nil(t, got)

		assert.False(t, repo.cache.Has(songIDCacheKey(9999)), "nil song result should not be cached")
	})

	t.Run("Nil result for non-existent chart is not cached", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		got, err := repo.GetChartByID(9999)
		assert.NoError(t, err)
		assert.Nil(t, got)

		assert.False(t, repo.cache.Has(chartIDCacheKey(9999)), "nil chart result should not be cached")
	})

	t.Run("Nil result for non-existent chart by wiki_diff is not cached", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewSongRepository(db)

		got, err := repo.GetChartByWikiIDAndDifficulty("nope", model.DifficultyMassive)
		assert.NoError(t, err)
		assert.Nil(t, got)

		assert.False(t, repo.cache.Has(chartWikiDiffCacheKey("nope", model.DifficultyMassive)),
			"nil chart wiki_diff result should not be cached")
	})

	t.Run("Nil result for non-existent best record by chart is not cached", func(t *testing.T) {
		db := setupTestDB(t)
		repo := NewRecordRepository(db)

		got, err := repo.GetBestRecordByChart("nobody", 9999)
		assert.NoError(t, err)
		assert.Nil(t, got)

		assert.False(t, repo.cache.Has(bestChartCacheKey("nobody", 9999)),
			"nil best record should not be cached")
	})
}

// ---------------------------------------------------------------------------
// filterCacheKey determinism
// ---------------------------------------------------------------------------

func TestFilterCacheKey(t *testing.T) {
	t.Run("Empty filter", func(t *testing.T) {
		assert.Equal(t, "nofilter", filterCacheKey(model.RecordFilter{}))
	})

	t.Run("MinLevel only", func(t *testing.T) {
		f := model.RecordFilter{MinLevel: float64Ptr(5.0)}
		assert.Equal(t, "min5.00", filterCacheKey(f))
	})

	t.Run("Full filter with sorted difficulties", func(t *testing.T) {
		f := model.RecordFilter{
			MinLevel:     float64Ptr(3.0),
			MaxLevel:     float64Ptr(12.0),
			Difficulties: []model.Difficulty{model.DifficultyMassive, model.DifficultyDetected},
		}
		// Difficulties should be sorted alphabetically in key
		key := filterCacheKey(f)
		assert.Equal(t, "min3.00_max12.00_diff:detected,massive", key)
	})

	t.Run("B15 filter", func(t *testing.T) {
		fTrue := model.RecordFilter{B15: boolPtr(true)}
		assert.Equal(t, "b15:true", filterCacheKey(fTrue))

		fFalse := model.RecordFilter{B15: boolPtr(false)}
		assert.Equal(t, "b15:false", filterCacheKey(fFalse))
	})

	t.Run("Combined filter with B15", func(t *testing.T) {
		f := model.RecordFilter{
			MinLevel:     float64Ptr(13.0),
			B15:          boolPtr(true),
			Difficulties: []model.Difficulty{model.DifficultyMassive},
		}
		key := filterCacheKey(f)
		assert.Equal(t, "min13.00_diff:massive_b15:true", key)
	})

	t.Run("Same filter produces same key regardless of difficulty order", func(t *testing.T) {
		f1 := model.RecordFilter{
			Difficulties: []model.Difficulty{model.DifficultyMassive, model.DifficultyDetected},
		}
		f2 := model.RecordFilter{
			Difficulties: []model.Difficulty{model.DifficultyDetected, model.DifficultyMassive},
		}
		assert.Equal(t, filterCacheKey(f1), filterCacheKey(f2))
	})
}
