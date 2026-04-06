package model

import (
	"time"
)

// PlayRecord represents a play record entity
type PlayRecord struct {
	PlayRecordBase
	ID         int       `gorm:"primaryKey" json:"id"`
	RecordTime time.Time `gorm:"type:timestamptz;not null" json:"record_time"`
	Username   string    `gorm:"not null;index;index:idx_pr_user_chart,priority:1" json:"username"`
	Rating     int       `gorm:"not null" json:"rating"`
	Chart      *Chart    `gorm:"foreignKey:ChartID;references:ID" json:"chart,omitempty"`
}

// TableName specifies the table name for GORM
func (PlayRecord) TableName() string {
	return "play_records"
}

// BestPlayRecord represents the best record for a specific chart
type BestPlayRecord struct {
	ID           int         `gorm:"primaryKey" json:"id"`
	Username     string      `gorm:"not null;uniqueIndex:idx_best_user_chart" json:"username"`
	ChartID      int         `gorm:"not null;uniqueIndex:idx_best_user_chart" json:"chart_id"`
	PlayRecordID int         `gorm:"column:play_record_id;not null;index" json:"play_record_id"`
	PlayRecord   *PlayRecord `gorm:"foreignKey:PlayRecordID;references:ID" json:"play_record,omitempty"`
}

// TableName specifies the table name for GORM
func (BestPlayRecord) TableName() string {
	return "best_play_records"
}

// PlayRecordBase represents the basic information of a play record
type PlayRecordBase struct {
	ChartID int  `json:"chart_id" gorm:"index:idx_pr_user_chart,priority:2" binding:"required" example:"1"`
	Score   *int `json:"score" gorm:"not null" binding:"required,min=0,max=1010000" example:"1000000"`
}

// PlayRecordInfo represents play record details including chart information
type PlayRecordInfo struct {
	ID         int             `json:"id"`
	RecordTime time.Time       `json:"record_time"`
	Score      int             `json:"score"`
	Rating     int             `json:"rating"`
	Chart      ChartInfoSimple `json:"chart"`
}

// ToPlayRecordInfo converts a PlayRecord (with preloaded Chart.Song) to PlayRecordInfo
func ToPlayRecordInfo(record *PlayRecord) PlayRecordInfo {
	info := PlayRecordInfo{
		ID:         record.ID,
		RecordTime: record.RecordTime,
		Score:      *record.Score,
		Rating:     record.Rating,
	}
	if record.Chart != nil {
		info.Chart = ChartInfoSimple{
			ID:           record.Chart.ID,
			Difficulty:   record.Chart.Difficulty,
			Level:        record.Chart.Level,
			FittingLevel: record.Chart.FittingLevel,
		}
		if record.Chart.Song != nil {
			effective := record.Chart.Song.WithOverride(record.Chart.SongBaseOverride)
			info.Chart.WikiID = effective.WikiID
			info.Chart.Title = effective.Title
			info.Chart.Version = effective.Version
			info.Chart.B15 = effective.B15
			info.Chart.SongID = record.Chart.Song.ID
			info.Chart.Cover = effective.Cover
		}
	}
	return info
}

// AllChartsResponse represents the response for the all-charts scope
type AllChartsResponse struct {
	Username string           `json:"username"`
	Nickname string           `json:"nickname"`
	Charts   []ChartWithScore `json:"charts"`
}

// PlayRecordResponse represents the response for play records
type PlayRecordResponse struct {
	Username string           `json:"username"`
	Nickname string           `json:"nickname"`
	Total    int              `json:"total"`
	Records  []PlayRecordInfo `json:"records"`
}
