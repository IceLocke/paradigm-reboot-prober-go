package repository

import (
	"errors"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"

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
func (r *UserRepository) CreateUser(req *request.CreateUserRequest, encodedPassword string, uploadToken string) (*model.User, error) {
	user := model.User{
		UserBase:        req.UserBase,
		EncodedPassword: encodedPassword,
	}

	user.UploadToken = uploadToken
	user.IsActive = true
	user.IsAdmin = false

	// Set default nickname if not provided
	if user.Nickname == "" {
		user.Nickname = user.Username
	}

	if err := r.db.Create(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateUser updates an existing user's information (PUT semantics)
func (r *UserRepository) UpdateUser(username string, req *request.UpdateUserRequest) (*model.User, error) {
	var user model.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}

	if req.Nickname != nil {
		user.Nickname = *req.Nickname
	}
	user.QQNumber = req.QQNumber
	user.Account = req.Account
	user.AccountNumber = req.AccountNumber
	user.UUID = req.UUID
	if req.AnonymousProbe != nil {
		user.AnonymousProbe = *req.AnonymousProbe
	}

	if err := r.db.Save(&user).Error; err != nil {
		return nil, err
	}

	return &user, nil
}
