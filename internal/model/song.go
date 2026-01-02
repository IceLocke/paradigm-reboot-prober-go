package model

// SongBase represents the basic information of a song
type SongBase struct {
	WikiID      string `gorm:"unique;not null" json:"wiki_id" example:"w123"`
	Title       string `gorm:"not null" json:"title" example:"Song Title"`
	Artist      string `gorm:"not null" json:"artist" example:"Artist Name"`
	Genre       string `gorm:"not null" json:"genre" example:"Pop"`
	Cover       string `gorm:"not null" json:"cover" example:"https://example.com/cover.jpg"`
	Illustrator string `gorm:"not null" json:"illustrator" example:"Artist"`
	Version     string `gorm:"not null" json:"version" example:"1.0.0"`
	B15         bool   `gorm:"not null" json:"b15" example:"false"`
	Album       string `gorm:"not null" json:"album" example:"First Album"`
	BPM         string `gorm:"not null" json:"bpm" example:"180"`
	Length      string `gorm:"not null" json:"length" example:"2:30"`
}

// Song represents the song entity
type Song struct {
	SongID int `gorm:"primaryKey;column:song_id" json:"song_id"`
	SongBase
	SongLevels []SongLevel `gorm:"foreignKey:SongID" json:"song_levels"`
}

// TableName specifies the table name for GORM
func (Song) TableName() string {
	return "songs"
}

// Difficulty represents the difficulty level of a song level
type Difficulty string

const (
	DifficultyDetected Difficulty = "Detected"
	DifficultyInvaded  Difficulty = "Invaded"
	DifficultyMassive  Difficulty = "Massive"
	DifficultyReboot   Difficulty = "Reboot"
)

// SongLevel represents the specific difficulty of a song
type SongLevel struct {
	SongLevelID  int        `gorm:"primaryKey;column:song_level_id" json:"song_level_id"`
	SongID       int        `gorm:"not null" json:"song_id"`
	Difficulty   Difficulty `gorm:"type:varchar(20);not null" json:"difficulty" example:"Massive"`
	Level        float64    `gorm:"not null" json:"level"`
	FittingLevel *float64   `gorm:"column:fitting_level" json:"fitting_level"`
	LevelDesign  *string    `gorm:"column:level_design" json:"level_design"`
	Notes        int        `gorm:"not null" json:"notes"`
	Song         *Song      `gorm:"foreignKey:SongID;references:SongID" json:"song,omitempty"`
}

// TableName specifies the table name for GORM
func (SongLevel) TableName() string {
	return "song_levels"
}

// LevelInfo represents the details of a difficulty level
type LevelInfo struct {
	Difficulty  Difficulty `json:"difficulty" example:"Massive"`
	Level       float64    `json:"level" example:"14.5"`
	LevelDesign string     `json:"level_design" example:"Designer"`
	Notes       int        `json:"notes" example:"1000"`
}

// SongLevelInfo represents the detailed information of a song's difficulty level
type SongLevelInfo struct {
	SongBase
	SongID       int        `json:"song_id" example:"1"`
	SongLevelID  int        `json:"song_level_id" example:"10"`
	Difficulty   Difficulty `json:"difficulty" example:"Massive"`
	Level        float64    `json:"level" example:"13.2"`
	FittingLevel float64    `json:"fitting_level" example:"13.4"`
	LevelDesign  string     `json:"level_design" example:"Designer"`
	Notes        int        `json:"notes" example:"850"`
}

// SongLevelInfoSimple represents a simplified version of song difficulty information
type SongLevelInfoSimple struct {
	WikiID       string     `json:"wiki_id"`
	Title        string     `json:"title"`
	Version      string     `json:"version"`
	B15          bool       `json:"b15"`
	SongID       int        `json:"song_id"`
	SongLevelID  int        `json:"song_level_id"`
	Difficulty   Difficulty `json:"difficulty"`
	Level        float64    `json:"level"`
	Cover        string     `json:"cover"`
	FittingLevel float64    `json:"fitting_level"`
}

// SongLevelCSV represents the model for CSV import
type SongLevelCSV struct {
	SongLevelID int        `json:"song_level_id"`
	Title       string     `json:"title"`
	Version     string     `json:"version"`
	Difficulty  Difficulty `json:"difficulty"`
	Level       float64    `json:"level"`
	Score       *int       `json:"score"`
}

// SongLevelWithScore represents a song level with the user's best score
type SongLevelWithScore struct {
	SongLevelID int        `json:"song_level_id"`
	Title       string     `json:"title"`
	Version     string     `json:"version"`
	Difficulty  Difficulty `json:"difficulty"`
	Level       float64    `json:"level"`
	Score       int        `json:"score"`
}
