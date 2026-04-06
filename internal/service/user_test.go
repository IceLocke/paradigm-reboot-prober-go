package service

import (
	"context"
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
	ctx := context.Background()

	t.Run("CreateUser", func(t *testing.T) {
		req := &request.CreateUserRequest{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "password123",
		}
		user, err := userService.CreateUser(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		assert.NotEmpty(t, user.UploadToken)
		assert.True(t, auth.VerifyPassword("password123", user.EncodedPassword))
	})

	t.Run("CreateDuplicateUser", func(t *testing.T) {
		req := &request.CreateUserRequest{
			Username: "testuser",
			Email:    "test2@example.com",
			Password: "password123",
		}
		user, err := userService.CreateUser(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "username has already existed", err.Error())
	})

	t.Run("LoginSuccess", func(t *testing.T) {
		token, err := userService.Login(ctx, "testuser", "password123")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("LoginFailure", func(t *testing.T) {
		token, err := userService.Login(ctx, "testuser", "wrongpassword")
		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Equal(t, "incorrect username or password", err.Error())
	})

	t.Run("RefreshUploadToken", func(t *testing.T) {
		user, _ := userService.GetUser("testuser")
		oldToken := user.UploadToken
		newToken, err := userService.RefreshUploadToken(ctx, "testuser")
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, oldToken, newToken)

		updatedUser, _ := userService.GetUser("testuser")
		assert.Equal(t, newToken, updatedUser.UploadToken)
	})
}

func TestUserService_ChangePassword(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	userService := NewUserService(userRepo)
	ctx := context.Background()

	_, err := userService.CreateUser(ctx, &request.CreateUserRequest{
		Username: "chgpwd_user",
		Email:    "chgpwd@example.com",
		Password: "oldpassword",
	})
	assert.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		err := userService.ChangePassword(ctx, "chgpwd_user", &request.ChangePasswordRequest{
			OldPassword: "oldpassword",
			NewPassword: "newpassword",
		})
		assert.NoError(t, err)

		// Verify login with new password works
		token, err := userService.Login(ctx, "chgpwd_user", "newpassword")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify login with old password fails
		_, err = userService.Login(ctx, "chgpwd_user", "oldpassword")
		assert.Error(t, err)
	})

	t.Run("Wrong old password", func(t *testing.T) {
		err := userService.ChangePassword(ctx, "chgpwd_user", &request.ChangePasswordRequest{
			OldPassword: "wrongpassword",
			NewPassword: "another",
		})
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("User not found", func(t *testing.T) {
		err := userService.ChangePassword(ctx, "ghost_user", &request.ChangePasswordRequest{
			OldPassword: "a",
			NewPassword: "bbbbbb",
		})
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
	})
}

func TestUserService_ResetPassword(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	userService := NewUserService(userRepo)
	ctx := context.Background()

	_, err := userService.CreateUser(ctx, &request.CreateUserRequest{
		Username: "rstpwd_user",
		Email:    "rstpwd@example.com",
		Password: "original123",
	})
	assert.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		err := userService.ResetPassword(ctx, &request.ResetPasswordRequest{
			Username:    "rstpwd_user",
			NewPassword: "resetpwd123",
		})
		assert.NoError(t, err)

		// Verify login with new password
		token, err := userService.Login(ctx, "rstpwd_user", "resetpwd123")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("User not found", func(t *testing.T) {
		err := userService.ResetPassword(ctx, &request.ResetPasswordRequest{
			Username:    "nobody_here",
			NewPassword: "xxxxxx",
		})
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	db := setupTestDB(t)
	userRepo := repository.NewUserRepository(db)
	userService := NewUserService(userRepo)
	ctx := context.Background()

	_, err := userService.CreateUser(ctx, &request.CreateUserRequest{
		Username: "upduser_svc",
		Email:    "updsvc@example.com",
		Nickname: "OldNick",
		Password: "password123",
	})
	assert.NoError(t, err)

	t.Run("Success", func(t *testing.T) {
		newNick := "NewNick"
		qq := "12345"
		updated, err := userService.UpdateUser(ctx, "upduser_svc", &request.UpdateUserRequest{
			Nickname:  &newNick,
			QQAccount: &qq,
		})
		assert.NoError(t, err)
		assert.Equal(t, "NewNick", updated.Nickname)
		assert.Equal(t, "12345", *updated.QQAccount)

		// Verify persisted
		fetched, _ := userService.GetUser("upduser_svc")
		assert.Equal(t, "NewNick", fetched.Nickname)
	})

	t.Run("User not found", func(t *testing.T) {
		nick := "x"
		_, err := userService.UpdateUser(ctx, "nobody_upd", &request.UpdateUserRequest{
			Nickname: &nick,
		})
		assert.Error(t, err)
		assert.Equal(t, "user not found", err.Error())
	})
}
