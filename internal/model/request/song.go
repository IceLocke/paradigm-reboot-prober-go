package request

import "paradigm-reboot-prober-go/internal/model"

// CreateSongRequest represents the request to create a new song
type CreateSongRequest struct {
	model.SongBase
	Levels []model.LevelInfo `json:"levels"`
}

// UpdateSongRequest represents the request to update an existing song
type UpdateSongRequest struct {
	SongID int `json:"song_id" binding:"required"`
	model.SongBase
	Levels []model.LevelInfo `json:"levels"`
}
