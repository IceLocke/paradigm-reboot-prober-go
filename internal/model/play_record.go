package model

import (
	"time"
)

// PlayRecord represents a play record entity
type PlayRecord struct {
	PlayRecordBase
	PlayRecordID int       `gorm:"primaryKey;column:play_record_id" json:"play_record_id"`
	RecordTime   time.Time `gorm:"not null" json:"record_time"`
	Username     string    `gorm:"not null" json:"username"`
	Rating       int       `gorm:"not null" json:"rating"`
	Chart        *Chart    `gorm:"foreignKey:ChartID;references:ChartID" json:"chart,omitempty"`
}

// TableName specifies the table name for GORM
func (PlayRecord) TableName() string {
	return "play_records"
}

// BestPlayRecord represents the best record for a specific chart
type BestPlayRecord struct {
	BestRecordID int         `gorm:"primaryKey;column:best_record_id" json:"best_record_id"`
	PlayRecordID int         `gorm:"column:play_record_id;not null" json:"play_record_id"`
	PlayRecord   *PlayRecord `gorm:"foreignKey:PlayRecordID;references:PlayRecordID" json:"play_record,omitempty"`
}

// TableName specifies the table name for GORM
func (BestPlayRecord) TableName() string {
	return "best_play_records"
}

// PlayRecordBase represents the basic information of a play record
type PlayRecordBase struct {
	ChartID int `json:"chart_id" binding:"required" example:"1"`
	Score   int `json:"score" binding:"min=0" example:"1000000"`
}

// PlayRecordInfo represents play record details including chart information
type PlayRecordInfo struct {
	PlayRecordID int             `json:"play_record_id"`
	RecordTime   time.Time       `json:"record_time"`
	Score        int             `json:"score"`
	Rating       int             `json:"rating"`
	Chart        ChartInfoSimple `json:"chart"`
}

// ToPlayRecordInfo converts a PlayRecord (with preloaded Chart.Song) to PlayRecordInfo
func ToPlayRecordInfo(record *PlayRecord) PlayRecordInfo {
	info := PlayRecordInfo{
		PlayRecordID: record.PlayRecordID,
		RecordTime:   record.RecordTime,
		Score:        record.Score,
		Rating:       record.Rating,
	}
	if record.Chart != nil {
		info.Chart = ChartInfoSimple{
			ChartID:      record.Chart.ChartID,
			Difficulty:   record.Chart.Difficulty,
			Level:        record.Chart.Level,
			FittingLevel: record.Chart.FittingLevel,
		}
		if record.Chart.Song != nil {
			info.Chart.WikiID = record.Chart.Song.WikiID
			info.Chart.Title = record.Chart.Song.Title
			info.Chart.Version = record.Chart.Song.Version
			info.Chart.B15 = record.Chart.Song.B15
			info.Chart.SongID = record.Chart.Song.SongID
			info.Chart.Cover = record.Chart.Song.Cover
		}
	}
	return info
}

// PlayRecordResponse represents the response for play records
type PlayRecordResponse struct {
	Username string           `json:"username"`
	Total    int              `json:"total"`
	Records  []PlayRecordInfo `json:"records"`
}
