package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
	"paradigm-reboot-prober-go/pkg/auth"
	"strings"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) Login(username, plainPassword string) (string, error) {
	username = strings.ToLower(username)

	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil || user == nil {
		return "", errors.New("incorrect username or password")
	}

	if !user.IsActive {
		return "", errors.New("inactivated user")
	}

	if !auth.VerifyPassword(plainPassword, user.EncodedPassword) {
		return "", errors.New("incorrect username or password")
	}

	duration := config.JWTExpirationDuration
	return auth.GenerateAccessJWT(username, &duration)
}

func (s *UserService) GetUser(username string) (*model.User, error) {
	return s.userRepo.GetUserByUsername(username)
}

func (s *UserService) GetUserByUploadToken(token string) (*model.User, error) {
	return s.userRepo.GetUserByUploadToken(token)
}

func generateHexToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *UserService) CreateUser(req *request.CreateUserRequest) (*model.User, error) {
	req.Username = strings.ToLower(req.Username)

	if !config.UsernameRegex.MatchString(req.Username) {
		return nil, errors.New("invalid username format")
	}

	existingUser, err := s.userRepo.GetUserByUsername(req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}
	if existingUser != nil {
		return nil, errors.New("username has already existed")
	}

	encodedPassword, err := auth.EncodePassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Generate a random upload token
	uploadToken, err := generateHexToken(config.GlobalConfig.Auth.UploadTokenLength)
	if err != nil {
		return nil, errors.New("generate token failed")
	}

	user := &model.User{
		UserBase: model.UserBase{
			Username:       req.Username,
			Email:          req.Email,
			Nickname:       req.Nickname,
			UploadToken:    uploadToken,
			IsActive:       true,
			IsAdmin:        false,
			AnonymousProbe: false,
		},
		EncodedPassword: encodedPassword,
	}

	return s.userRepo.CreateUser(user)
}

func (s *UserService) RefreshUploadToken(username string) (string, error) {
	var token string
	err := s.userRepo.WithTransaction(func(tx *repository.UserRepository) error {
		user, err := tx.GetUserByUsername(username)
		if err != nil {
			return err
		}
		if user == nil {
			return errors.New("user not found")
		}

		uploadToken, err := generateHexToken(config.GlobalConfig.Auth.UploadTokenLength)
		if err != nil {
			return errors.New("generate token failed")
		}
		user.UploadToken = uploadToken
		if _, err := tx.UpdateUser(user); err != nil {
			return err
		}
		token = uploadToken
		return nil
	})
	return token, err
}

func (s *UserService) UpdateUser(username string, req *request.UpdateUserRequest) (*model.User, error) {
	var result *model.User
	err := s.userRepo.WithTransaction(func(tx *repository.UserRepository) error {
		user, err := tx.GetUserByUsername(username)
		if err != nil {
			return err
		}
		if user == nil {
			return errors.New("user not found")
		}

		if req.Nickname != nil {
			user.Nickname = *req.Nickname
		}
		if req.QQNumber != nil {
			user.QQNumber = req.QQNumber
		}
		if req.Account != nil {
			user.Account = req.Account
		}
		if req.AccountNumber != nil {
			user.AccountNumber = req.AccountNumber
		}
		if req.UUID != nil {
			user.UUID = req.UUID
		}
		if req.AnonymousProbe != nil {
			user.AnonymousProbe = *req.AnonymousProbe
		}

		result, err = tx.UpdateUser(user)
		return err
	})
	return result, err
}

func (s *UserService) ChangePassword(username string, req *request.ChangePasswordRequest) error {
	return s.userRepo.WithTransaction(func(tx *repository.UserRepository) error {
		user, err := tx.GetUserByUsername(username)
		if err != nil {
			return err
		}
		if user == nil {
			return errors.New("user not found")
		}

		if !auth.VerifyPassword(req.OldPassword, user.EncodedPassword) {
			return errors.New("incorrect old password")
		}

		encodedPassword, err := auth.EncodePassword(req.NewPassword)
		if err != nil {
			return err
		}

		user.EncodedPassword = encodedPassword
		_, err = tx.UpdateUser(user)
		return err
	})
}

func (s *UserService) ResetPassword(req *request.ResetPasswordRequest) error {
	req.Username = strings.ToLower(req.Username)

	return s.userRepo.WithTransaction(func(tx *repository.UserRepository) error {
		user, err := tx.GetUserByUsername(req.Username)
		if err != nil {
			return err
		}
		if user == nil {
			return errors.New("user not found")
		}

		encodedPassword, err := auth.EncodePassword(req.NewPassword)
		if err != nil {
			return err
		}

		user.EncodedPassword = encodedPassword
		_, err = tx.UpdateUser(user)
		return err
	})
}

func (s *UserService) CheckProbeAuthority(username string, currentUser *model.User) error {
	targetUser, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		return fmt.Errorf("failed to query user: %w", err)
	}
	if targetUser == nil {
		return errors.New("user not found")
	}

	// Authorized if: anonymous probe enabled, or authenticated as the target user, or admin
	isAuthorized := targetUser.AnonymousProbe ||
		(currentUser != nil && currentUser.Username == username) ||
		(currentUser != nil && currentUser.IsAdmin)

	if !isAuthorized {
		if currentUser == nil {
			return errors.New("anonymous probes are not allowed")
		}
		if currentUser.Username != username {
			return errors.New("authentication info not matched")
		}
		return errors.New("forbidden")
	}

	return nil
}
