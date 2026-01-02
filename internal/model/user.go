package model

// UserBase represents the basic information of a user
type UserBase struct {
	Username       string  `gorm:"unique;not null" json:"username" binding:"required" example:"user123"`
	Email          string  `gorm:"not null" json:"email" binding:"required,email" example:"user@example.com"`
	Nickname       string  `gorm:"not null" json:"nickname" example:"小明"`
	QQNumber       *int    `gorm:"column:qq_number" json:"qq_number" example:"12345678"`
	Account        *string `gorm:"column:account" json:"account" example:"act_001"`
	AccountNumber  *int    `gorm:"column:account_number" json:"account_number" example:"1001"`
	UUID           *string `gorm:"column:uuid" json:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AnonymousProbe bool    `gorm:"not null;default:false" json:"anonymous_probe" example:"false"`
	UploadToken    string  `gorm:"not null" json:"upload_token" example:"token_xyz"`
	IsActive       bool    `gorm:"not null;default:true" json:"is_active" example:"true"`
	IsAdmin        bool    `gorm:"not null;default:false" json:"is_admin" example:"false"`
}

// User represents the user entity in the database
type User struct {
	UserID int `gorm:"primaryKey;column:user_id" json:"user_id"`
	UserBase
	EncodedPassword string `gorm:"not null" json:"-"`
}

// TableName specifies the table name for GORM
func (User) TableName() string {
	return "prober_users"
}

// UserInDB represents the user model stored in the database
type UserInDB struct {
	UserBase
	EncodedPassword string `json:"-"`
}
