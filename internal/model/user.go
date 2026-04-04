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
	UploadToken    string  `gorm:"not null;uniqueIndex" json:"upload_token" example:"token_xyz"`
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

// UserPublic represents user information safe for public responses (e.g., registration)
type UserPublic struct {
	UserID         int     `json:"user_id" example:"1"`
	Username       string  `json:"username" example:"user123"`
	Email          string  `json:"email" example:"user@example.com"`
	Nickname       string  `json:"nickname" example:"小明"`
	QQNumber       *int    `json:"qq_number" example:"12345678"`
	Account        *string `json:"account" example:"act_001"`
	AccountNumber  *int    `json:"account_number" example:"1001"`
	UUID           *string `json:"uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AnonymousProbe bool    `json:"anonymous_probe" example:"false"`
}

// ToPublic converts a User to a UserPublic response (excludes upload_token, is_admin, etc.)
func (u *User) ToPublic() UserPublic {
	return UserPublic{
		UserID:         u.UserID,
		Username:       u.Username,
		Email:          u.Email,
		Nickname:       u.Nickname,
		QQNumber:       u.QQNumber,
		Account:        u.Account,
		AccountNumber:  u.AccountNumber,
		UUID:           u.UUID,
		AnonymousProbe: u.AnonymousProbe,
	}
}

// UserInDB represents the user model stored in the database
type UserInDB struct {
	UserBase
	EncodedPassword string `json:"-"`
}
