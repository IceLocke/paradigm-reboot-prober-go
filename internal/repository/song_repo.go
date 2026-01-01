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
	if err := r.db.Find(&songs).Error; err != nil {
		return nil, err
	}
	return songs, nil
}

// GetSongByID retrieves a song by its ID
func (r *SongRepository) GetSongByID(songID int) (*model.Song, error) {
	var song model.Song
	if err := r.db.Where("song_id = ?", songID).First(&song).Error; err != nil {
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
	if err := r.db.Where("wiki_id = ?", wikiID).First(&song).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &song, nil
}

// CreateSong creates a new song with its levels
func (r *SongRepository) CreateSong(song *model.Song) (*model.Song, error) {
	// GORM handles association creation automatically if configured correctly
	if err := r.db.Create(song).Error; err != nil {
		return nil, err
	}
	return song, nil
}

// UpdateSong updates an existing song and its levels
func (r *SongRepository) UpdateSong(songID int, updatedSong *model.Song) (*model.Song, error) {
	var result *model.Song
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existingSong model.Song
		if err := tx.Preload("SongLevels").First(&existingSong, songID).Error; err != nil {
			return err
		}

		// Update basic attributes
		existingSong.Title = updatedSong.Title
		existingSong.Artist = updatedSong.Artist
		existingSong.Cover = updatedSong.Cover
		existingSong.Illustrator = updatedSong.Illustrator
		existingSong.BPM = updatedSong.BPM
		existingSong.B15 = updatedSong.B15
		existingSong.Album = updatedSong.Album
		existingSong.WikiID = updatedSong.WikiID
		// Add other fields if necessary

		if err := tx.Save(&existingSong).Error; err != nil {
			return err
		}

		// Update Song Levels
		// Strategy: Map existing levels by Difficulty, update if exists, create if new
		existingLevelsMap := make(map[model.Difficulty]*model.SongLevel)
		for i := range existingSong.SongLevels {
			level := &existingSong.SongLevels[i]
			existingLevelsMap[level.Difficulty] = level
		}

		for _, newLevel := range updatedSong.SongLevels {
			if existingLevel, exists := existingLevelsMap[newLevel.Difficulty]; exists {
				// Update existing level
				existingLevel.Level = newLevel.Level
				existingLevel.LevelDesign = newLevel.LevelDesign
				existingLevel.Notes = newLevel.Notes
				existingLevel.FittingLevel = newLevel.FittingLevel
				if err := tx.Save(existingLevel).Error; err != nil {
					return err
				}
			} else {
				// Create new level
				newLevel.SongID = existingSong.SongID
				if err := tx.Create(&newLevel).Error; err != nil {
					return err
				}
			}
		}
		result = &existingSong
		return nil
	})

	return result, err
}
