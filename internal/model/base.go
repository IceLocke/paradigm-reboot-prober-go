package model

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel provides audit fields for all GORM entities.
// Embed this struct to get automatic CreatedAt/UpdatedAt timestamps
// and soft-delete support via DeletedAt.
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
