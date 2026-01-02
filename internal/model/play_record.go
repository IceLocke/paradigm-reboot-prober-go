package model

import (
	"time"
)

// PlayRecord represents a play record entity
type PlayRecord struct {
	PlayRecordBase
	PlayRecordID int        `gorm:"primaryKey;column:play_record_id" json:"play_record_id"`
	RecordTime   time.Time  `gorm:"not null" json:"record_time"`
	Username     string     `gorm:"not null" json:"username"`
	Rating       int        `gorm:"not null" json:"rating"`
	User         *User      `gorm:"foreignKey:Username;references:Username" json:"user,omitempty"`
	SongLevel    *SongLevel `gorm:"foreignKey:SongLevelID;references:SongLevelID" json:"song_level,omitempty"`
}

// TableName specifies the table name for GORM
func (PlayRecord) TableName() string {
	return "play_records"
}

// BestPlayRecord represents the best record for a specific song level
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
	SongLevelID int `json:"song_level_id" binding:"required" example:"1"`
	Score       int `json:"score" binding:"required" example:"1000000"`
}

// PlayRecordInfo represents play record details including song information
type PlayRecordInfo struct {
	PlayRecordID int                 `json:"play_record_id"`
	RecordTime   time.Time           `json:"record_time"`
	Score        int                 `json:"score"`
	Rating       int                 `json:"rating"`
	SongLevel    SongLevelInfoSimple `json:"song_level"`
}

// PlayRecordResponse represents the response for play records
type PlayRecordResponse struct {
	Username string           `json:"username"`
	Total    int              `json:"total"`
	Records  []PlayRecordInfo `json:"records"`
}
