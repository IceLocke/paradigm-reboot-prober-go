package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"

	"github.com/jellydator/ttlcache/v3"
	"gorm.io/gorm"
)

type SongRepository struct {
	db    *gorm.DB
	cache *repoCache
}

func NewSongRepository(db *gorm.DB) *SongRepository {
	return &SongRepository{
		db:    db,
		cache: newRepoCache(SongCacheTTL),
	}
}

// GetAllSongs retrieves all songs
func (r *SongRepository) GetAllSongs() ([]model.Song, error) {
	key := allSongsCacheKey()
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().([]model.Song)
			cp := make([]model.Song, len(original))
			copy(cp, original)
			return cp, nil
		}
	}

	var songs []model.Song
	if err := r.db.Preload("Charts").Find(&songs).Error; err != nil {
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(key, songs, ttlcache.DefaultTTL)
	}
	return songs, nil
}

// GetSongByID retrieves a song by its ID
func (r *SongRepository) GetSongByID(songID int) (*model.Song, error) {
	key := songIDCacheKey(songID)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().(*model.Song)
			cp := *original
			return &cp, nil
		}
	}

	var song model.Song
	if err := r.db.Preload("Charts").Where("id = ?", songID).First(&song).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(key, &song, ttlcache.DefaultTTL)
		cp := song
		return &cp, nil
	}
	return &song, nil
}

// GetSongByWikiID retrieves a song by its Wiki ID
func (r *SongRepository) GetSongByWikiID(wikiID string) (*model.Song, error) {
	key := songWikiCacheKey(wikiID)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().(*model.Song)
			cp := *original
			return &cp, nil
		}
	}

	var song model.Song
	if err := r.db.Preload("Charts").Where("wiki_id = ?", wikiID).First(&song).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(key, &song, ttlcache.DefaultTTL)
		cp := song
		return &cp, nil
	}
	return &song, nil
}

// GetChartByID retrieves a chart by its numeric ID with Song preloaded
func (r *SongRepository) GetChartByID(chartID int) (*model.Chart, error) {
	key := chartIDCacheKey(chartID)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().(*model.Chart)
			cp := *original
			return &cp, nil
		}
	}

	var chart model.Chart
	if err := r.db.Preload("Song").Where("id = ?", chartID).First(&chart).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(key, &chart, ttlcache.DefaultTTL)
		cp := chart
		return &cp, nil
	}
	return &chart, nil
}

// GetChartByWikiIDAndDifficulty finds a chart by the song's wiki_id and chart difficulty
func (r *SongRepository) GetChartByWikiIDAndDifficulty(wikiID string, difficulty model.Difficulty) (*model.Chart, error) {
	key := chartWikiDiffCacheKey(wikiID, difficulty)
	if r.cache != nil {
		if item := r.cache.Get(key); item != nil {
			original := item.Value().(*model.Chart)
			cp := *original
			return &cp, nil
		}
	}

	var chart model.Chart
	if err := r.db.Joins("JOIN songs ON songs.id = charts.song_id").
		Preload("Song").
		Where("songs.wiki_id = ? AND charts.difficulty = ?", wikiID, difficulty).
		First(&chart).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if r.cache != nil {
		r.cache.Set(key, &chart, ttlcache.DefaultTTL)
		cp := chart
		return &cp, nil
	}
	return &chart, nil
}

// CreateSong creates a new song with its charts
func (r *SongRepository) CreateSong(song *model.Song) (*model.Song, error) {
	// GORM handles association creation automatically if configured correctly
	if err := r.db.Create(song).Error; err != nil {
		return nil, err
	}
	// Flush all song/chart caches — song creation affects GetAllSongs
	if r.cache != nil {
		r.cache.DeleteAll()
	}
	return song, nil
}

// UpdateSong updates an existing song and its charts
func (r *SongRepository) UpdateSong(songID int, updatedSong *model.Song) (*model.Song, error) {
	var result *model.Song
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existingSong model.Song
		if err := tx.Preload("Charts").First(&existingSong, songID).Error; err != nil {
			return err
		}

		// Update basic attributes
		existingSong.Title = updatedSong.Title
		existingSong.Artist = updatedSong.Artist
		existingSong.Genre = updatedSong.Genre
		existingSong.Cover = updatedSong.Cover
		existingSong.Illustrator = updatedSong.Illustrator
		existingSong.Version = updatedSong.Version
		existingSong.BPM = updatedSong.BPM
		existingSong.B15 = updatedSong.B15
		existingSong.Album = updatedSong.Album
		existingSong.Length = updatedSong.Length
		existingSong.WikiID = updatedSong.WikiID

		if err := tx.Save(&existingSong).Error; err != nil {
			return err
		}

		// Update Charts
		// Strategy: Map existing charts by Difficulty, update if exists, create if new
		existingChartsMap := make(map[model.Difficulty]*model.Chart)
		for i := range existingSong.Charts {
			chart := &existingSong.Charts[i]
			existingChartsMap[chart.Difficulty] = chart
		}

		for _, newChart := range updatedSong.Charts {
			if existingChart, exists := existingChartsMap[newChart.Difficulty]; exists {
				// Update existing chart
				levelChanged := existingChart.Level != newChart.Level
				existingChart.Level = newChart.Level
				existingChart.LevelDesign = newChart.LevelDesign
				existingChart.Notes = newChart.Notes
				existingChart.SongBaseOverride = newChart.SongBaseOverride
				if err := tx.Save(existingChart).Error; err != nil {
					return err
				}
				// Recalculate ratings for all play records when level changes
				if levelChanged {
					if err := RecalculateRatingsByChart(tx, existingChart.ID, newChart.Level); err != nil {
						return err
					}
				}
			} else {
				// Create new chart
				newChart.SongID = existingSong.ID
				if err := tx.Create(&newChart).Error; err != nil {
					return err
				}
				existingSong.Charts = append(existingSong.Charts, newChart)
			}
		}
		// Delete charts not in the update request
		requestedDifficulties := make(map[model.Difficulty]bool)
		for _, c := range updatedSong.Charts {
			requestedDifficulties[c.Difficulty] = true
		}
		remainingCharts := make([]model.Chart, 0, len(existingSong.Charts))
		for i := range existingSong.Charts {
			chart := &existingSong.Charts[i]
			if !requestedDifficulties[chart.Difficulty] {
				if err := tx.Delete(chart).Error; err != nil {
					return err
				}
			} else {
				remainingCharts = append(remainingCharts, *chart)
			}
		}
		existingSong.Charts = remainingCharts
		result = &existingSong
		return nil
	})

	// Flush all song/chart caches after successful TX
	if err == nil && r.cache != nil {
		r.cache.DeleteAll()
	}

	return result, err
}
