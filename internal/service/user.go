package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
	"paradigm-reboot-prober-go/pkg/auth"
	"time"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) Login(username, plainPassword string) (string, error) {
	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		return "", errors.New("incorrect username or password")
	}

	if !user.IsActive {
		return "", errors.New("inactivated user")
	}

	if !auth.VerifyPassword(plainPassword, user.EncodedPassword) {
		return "", errors.New("incorrect username or password")
	}

	duration := time.Hour * 24 // Default expiration
	return auth.GenerateAccessJWT(username, &duration)
}

func (s *UserService) GetUser(username string) (*model.User, error) {
	return s.userRepo.GetUserByUsername(username)
}

func generateHexToken(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *UserService) CreateUser(req *request.CreateUserRequest) (*model.User, error) {
	existingUser, _ := s.userRepo.GetUserByUsername(req.Username)
	if existingUser != nil {
		return nil, errors.New("username has already existed")
	}

	encodedPassword, err := auth.EncodePassword(req.Password)
	if err != nil {
		return nil, err
	}

	// Generate a random upload token
	uploadToken, err := generateHexToken(16)
	if err != nil {
		return nil, errors.New("generate token failed")
	}

	user := &model.User{
		UserBase:        req.UserBase,
		EncodedPassword: encodedPassword,
	}
	user.UploadToken = uploadToken
	user.IsActive = true
	user.IsAdmin = false

	if err := s.userRepo.CreateUser(user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) RefreshUploadToken(username string) (string, error) {
	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", errors.New("user not found")
	}

	uploadToken, err := generateHexToken(16)
	if err != nil {
		return "", errors.New("generate token failed")
	}
	user.UploadToken = uploadToken
	if err := s.userRepo.UpdateUser(user); err != nil {
		return "", err
	}

	return uploadToken, nil
}

func (s *UserService) CheckProbeAuthority(username string, currentUser *model.User) error {
	targetUser, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		return errors.New("user not found")
	}

	// !(允许匿名查询 | 认证 & 信息匹配 | 认证 & 是管理员)
	isAuthorized := targetUser.AnonymousProbe ||
		(currentUser != nil && currentUser.Username == username) ||
		(currentUser != nil && currentUser.IsAdmin)

	if !isAuthorized {
		if !targetUser.AnonymousProbe {
			return errors.New("anonymous probes are not allowed")
		}
		if currentUser != nil && currentUser.Username != username {
			return errors.New("authentication info not matched")
		}
		return errors.New("forbidden")
	}

	return nil
}
