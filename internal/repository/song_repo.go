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
	if err := r.db.Preload("Charts").Where("song_id = ?", songID).First(&song).Error; err != nil {
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
		existingLevelsMap := make(map[model.Difficulty]*model.Chart)
		for i := range existingSong.Charts {
			chart := &existingSong.Charts[i]
			existingLevelsMap[chart.Difficulty] = chart
		}

		for _, newChart := range updatedSong.Charts {
			if existingChart, exists := existingLevelsMap[newChart.Difficulty]; exists {
				// Update existing chart
				existingChart.Level = newChart.Level
				existingChart.LevelDesign = newChart.LevelDesign
				existingChart.Notes = newChart.Notes
				if err := tx.Save(existingChart).Error; err != nil {
					return err
				}
			} else {
				// Create new chart
				newChart.SongID = existingSong.SongID
				if err := tx.Create(&newChart).Error; err != nil {
					return err
				}
			}
		}
		result = &existingSong
		return nil
	})

	return result, err
}
