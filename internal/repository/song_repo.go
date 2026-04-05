package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"

	"gorm.io/gorm"
)

type SongRepository struct {
	db *gorm.DB
}

func NewSongRepository(db *gorm.DB) *SongRepository {
	return &SongRepository{db: db}
}

// GetAllSongs retrieves all songs
func (r *SongRepository) GetAllSongs() ([]model.Song, error) {
	var songs []model.Song
	if err := r.db.Preload("Charts").Find(&songs).Error; err != nil {
		return nil, err
	}
	return songs, nil
}

// GetSongByID retrieves a song by its ID
func (r *SongRepository) GetSongByID(songID int) (*model.Song, error) {
	var song model.Song
	if err := r.db.Preload("Charts").Where("id = ?", songID).First(&song).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &song, nil
}

// GetSongByWikiID retrieves a song by its Wiki ID
func (r *SongRepository) GetSongByWikiID(wikiID string) (*model.Song, error) {
	var song model.Song
	if err := r.db.Preload("Charts").Where("wiki_id = ?", wikiID).First(&song).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &song, nil
}

// GetChartByID retrieves a chart by its numeric ID with Song preloaded
func (r *SongRepository) GetChartByID(chartID int) (*model.Chart, error) {
	var chart model.Chart
	if err := r.db.Preload("Song").Where("id = ?", chartID).First(&chart).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &chart, nil
}

// GetChartByWikiIDAndDifficulty finds a chart by the song's wiki_id and chart difficulty
func (r *SongRepository) GetChartByWikiIDAndDifficulty(wikiID string, difficulty model.Difficulty) (*model.Chart, error) {
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
	return &chart, nil
}

// CreateSong creates a new song with its charts
func (r *SongRepository) CreateSong(song *model.Song) (*model.Song, error) {
	// GORM handles association creation automatically if configured correctly
	if err := r.db.Create(song).Error; err != nil {
		return nil, err
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
		result = &existingSong
		return nil
	})

	return result, err
}
