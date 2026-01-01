package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetUserByUsername retrieves a user by their username
func (r *UserRepository) GetUserByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(user *model.User) error {
	// Set default nickname if not provided
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := r.db.Create(user).Error; err != nil {
		return err
	}
	return nil
}

// UpdateUser updates an existing user's information (PUT semantics)
func (r *UserRepository) UpdateUser(user *model.User) error {
	if err := r.db.Save(user).Error; err != nil {
		return err
	}
	return nil
}
