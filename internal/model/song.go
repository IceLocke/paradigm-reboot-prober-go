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
	Charts []Chart `gorm:"foreignKey:SongID" json:"charts"`
}

// TableName specifies the table name for GORM
func (Song) TableName() string {
	return "songs"
}

// Difficulty represents the difficulty level of a chart
type Difficulty string

const (
	DifficultyDetected Difficulty = "detected"
	DifficultyInvaded  Difficulty = "invaded"
	DifficultyMassive  Difficulty = "massive"
	DifficultyReboot   Difficulty = "reboot"
)

// ValidDifficulty checks if a string is a valid Difficulty value
func ValidDifficulty(s string) bool {
	switch Difficulty(s) {
	case DifficultyDetected, DifficultyInvaded, DifficultyMassive, DifficultyReboot:
		return true
	}
	return false
}

// Chart represents a specific difficulty chart (谱面) of a song
type Chart struct {
	ChartID      int        `gorm:"primaryKey;column:chart_id" json:"chart_id"`
	SongID       int        `gorm:"not null" json:"song_id"`
	Difficulty   Difficulty `gorm:"type:varchar(20);not null" json:"difficulty" example:"massive"`
	Level        float64    `gorm:"not null" json:"level"`
	FittingLevel *float64   `gorm:"column:fitting_level" json:"fitting_level"`
	LevelDesign  *string    `gorm:"column:level_design" json:"level_design"`
	Notes        int        `gorm:"not null" json:"notes"`
	Song         *Song      `gorm:"foreignKey:SongID;references:SongID" json:"song,omitempty"`
}

// TableName specifies the table name for GORM
func (Chart) TableName() string {
	return "charts"
}

// ChartInput represents the details of a chart for create/update requests
type ChartInput struct {
	Difficulty  Difficulty `json:"difficulty" example:"massive"`
	Level       float64    `json:"level" example:"14.5"`
	LevelDesign string     `json:"level_design" example:"Designer"`
	Notes       int        `json:"notes" example:"1000"`
}

// ChartInfo represents the detailed information of a song's chart (flattened view)
type ChartInfo struct {
	SongBase
	SongID       int        `json:"song_id" example:"1"`
	ChartID      int        `json:"chart_id" example:"10"`
	Difficulty   Difficulty `json:"difficulty" example:"massive"`
	Level        float64    `json:"level" example:"13.2"`
	FittingLevel *float64   `json:"fitting_level" example:"13.4"`
	LevelDesign  *string    `json:"level_design" example:"Designer"`
	Notes        int        `json:"notes" example:"850"`
}

// ChartInfoSimple represents a simplified version of chart information
type ChartInfoSimple struct {
	WikiID       string     `json:"wiki_id"`
	Title        string     `json:"title"`
	Version      string     `json:"version"`
	B15          bool       `json:"b15"`
	SongID       int        `json:"song_id"`
	ChartID      int        `json:"chart_id"`
	Difficulty   Difficulty `json:"difficulty"`
	Level        float64    `json:"level"`
	Cover        string     `json:"cover"`
	FittingLevel *float64   `json:"fitting_level"`
}

// ChartCSV represents the model for CSV import
type ChartCSV struct {
	ChartID    int        `json:"chart_id"`
	Title      string     `json:"title"`
	Version    string     `json:"version"`
	Difficulty Difficulty `json:"difficulty"`
	Level      float64    `json:"level"`
	Score      *int       `json:"score"`
}

// ChartWithScore represents a chart with the user's best score
type ChartWithScore struct {
	ChartID    int        `json:"chart_id"`
	Title      string     `json:"title"`
	Version    string     `json:"version"`
	Difficulty Difficulty `json:"difficulty"`
	Level      float64    `json:"level"`
	Score      int        `json:"score"`
}
