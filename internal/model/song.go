package model

// SongBase represents the basic information of a song
type SongBase struct {
	WikiID      string `gorm:"unique;not null" json:"wiki_id" binding:"required" example:"w123"`
	Title       string `gorm:"not null" json:"title" binding:"required" example:"Song Title"`
	Artist      string `gorm:"not null" json:"artist" binding:"required" example:"Artist Name"`
	Genre       string `gorm:"not null" json:"genre" example:"Pop"`
	Cover       string `gorm:"not null" json:"cover" example:"Cover_d3d3d3.jpg"`
	Illustrator string `gorm:"not null" json:"illustrator" example:"Artist"`
	Version     string `gorm:"not null" json:"version" example:"1.0.0"`
	B15         bool   `gorm:"not null;index" json:"b15" example:"false"`
	Album       string `gorm:"not null" json:"album" example:"First Album"`
	BPM         string `gorm:"not null" json:"bpm" example:"180"`
	Length      string `gorm:"not null" json:"length" example:"2:30"`
}

// Song represents the song entity
type Song struct {
	ID int `gorm:"primaryKey" json:"id"`
	SongBase
	Charts []Chart `gorm:"foreignKey:SongID;references:ID" json:"charts"`
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

// Order returns the sort weight for a Difficulty (higher = harder).
func (d Difficulty) Order() int {
	switch d {
	case DifficultyReboot:
		return 4
	case DifficultyMassive:
		return 3
	case DifficultyInvaded:
		return 2
	case DifficultyDetected:
		return 1
	default:
		return 0
	}
}

// SongBaseOverride holds optional per-chart overrides for SongBase fields.
// A nil pointer means "use the song's original value".
type SongBaseOverride struct {
	OverrideTitle   *string `gorm:"column:override_title"   json:"override_title,omitempty"   example:"Alt Title"`
	OverrideArtist  *string `gorm:"column:override_artist"  json:"override_artist,omitempty"  example:"Alt Artist"`
	OverrideVersion *string `gorm:"column:override_version" json:"override_version,omitempty" example:"2.0.0"`
	OverrideCover   *string `gorm:"column:override_cover"   json:"override_cover,omitempty"   example:"Cover_alt.jpg"`
}

// WithOverride returns a copy of SongBase with non-nil override fields applied.
func (s SongBase) WithOverride(o SongBaseOverride) SongBase {
	if o.OverrideTitle != nil {
		s.Title = *o.OverrideTitle
	}
	if o.OverrideArtist != nil {
		s.Artist = *o.OverrideArtist
	}
	if o.OverrideVersion != nil {
		s.Version = *o.OverrideVersion
	}
	if o.OverrideCover != nil {
		s.Cover = *o.OverrideCover
	}
	return s
}

// Chart represents a specific difficulty chart (谱面) of a song
type Chart struct {
	ID           int        `gorm:"primaryKey" json:"id"`
	SongID       int        `gorm:"not null;uniqueIndex:idx_song_difficulty" json:"song_id"`
	Difficulty   Difficulty `gorm:"type:varchar(20);not null;uniqueIndex:idx_song_difficulty" json:"difficulty" example:"massive"`
	Level        float64    `gorm:"not null" json:"level"`
	FittingLevel *float64   `gorm:"column:fitting_level" json:"fitting_level"`
	LevelDesign  *string    `gorm:"column:level_design" json:"level_design"`
	Notes        int        `gorm:"not null" json:"notes"`
	SongBaseOverride
	Song *Song `gorm:"foreignKey:SongID;references:ID" json:"song,omitempty"`
}

// TableName specifies the table name for GORM
func (Chart) TableName() string {
	return "charts"
}

// ChartInput represents the details of a chart for create/update requests
type ChartInput struct {
	Difficulty  Difficulty `json:"difficulty" binding:"required,oneof=detected invaded massive reboot" example:"massive"`
	Level       float64    `json:"level" binding:"required,gt=0" example:"14.5"`
	LevelDesign string     `json:"level_design" example:"Designer"`
	Notes       int        `json:"notes" binding:"required,min=0" example:"1000"`
	SongBaseOverride
}

// ChartInfo represents the detailed information of a song's chart (flattened view)
type ChartInfo struct {
	SongBase
	SongID       int        `json:"song_id" example:"1"`
	ID           int        `json:"id" example:"10"`
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
	ID           int        `json:"id"`
	Difficulty   Difficulty `json:"difficulty"`
	Level        float64    `json:"level"`
	Cover        string     `json:"cover"`
	FittingLevel *float64   `json:"fitting_level"`
}

// ChartWithScore represents a chart with the user's best score
type ChartWithScore struct {
	ID         int        `json:"id"`
	Title      string     `json:"title"`
	Version    string     `json:"version"`
	Difficulty Difficulty `json:"difficulty"`
	Level      float64    `json:"level"`
	Score      int        `json:"score"`
}
