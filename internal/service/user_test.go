package service

import (
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"paradigm-reboot-prober-go/internal/repository"
	"paradigm-reboot-prober-go/pkg/auth"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserService(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	userService := NewUserService(userRepo)

	t.Run("CreateUser", func(t *testing.T) {
		req := &request.CreateUserRequest{
			UserBase: model.UserBase{
				Username: "testuser",
				Email:    "test@example.com",
			},
			Password: "password123",
		}
		user, err := userService.CreateUser(req)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		assert.NotEmpty(t, user.UploadToken)
		assert.True(t, auth.VerifyPassword("password123", user.EncodedPassword))
	})

	t.Run("CreateDuplicateUser", func(t *testing.T) {
		req := &request.CreateUserRequest{
			UserBase: model.UserBase{
				Username: "testuser",
				Email:    "test2@example.com",
			},
			Password: "password123",
		}
		user, err := userService.CreateUser(req)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "username has already existed", err.Error())
	})

	t.Run("LoginSuccess", func(t *testing.T) {
		token, err := userService.Login("testuser", "password123")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("LoginFailure", func(t *testing.T) {
		token, err := userService.Login("testuser", "wrongpassword")
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Equal(t, "incorrect username or password", err.Error())
	})

	t.Run("RefreshUploadToken", func(t *testing.T) {
		user, _ := userService.GetUser("testuser")
		oldToken := user.UploadToken
		newToken, err := userService.RefreshUploadToken("testuser")
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, oldToken, newToken)

		updatedUser, _ := userService.GetUser("testuser")
		assert.Equal(t, newToken, updatedUser.UploadToken)
	})
}
