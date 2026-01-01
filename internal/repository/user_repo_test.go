package repository

import (
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/internal/model/request"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("Create User Success", func(t *testing.T) {
		req := &request.CreateUserRequest{
			UserBase: model.UserBase{
				Username: "testuser",
				Email:    "test@example.com",
				Nickname: "Test User",
			},
		}
		user, err := repo.CreateUser(req, "encoded_password", "token123")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "Test User", user.Nickname)
		assert.Equal(t, "token123", user.UploadToken)
		assert.True(t, user.IsActive)
		assert.False(t, user.IsAdmin)
	})

	t.Run("Create User Default Nickname", func(t *testing.T) {
		req := &request.CreateUserRequest{
			UserBase: model.UserBase{
				Username: "user_no_nick",
				Email:    "nonick@example.com",
			},
		}
		user, err := repo.CreateUser(req, "pass", "token")
		assert.NoError(t, err)
		assert.Equal(t, "user_no_nick", user.Nickname)
	})

	t.Run("Create Duplicate User", func(t *testing.T) {
		req := &request.CreateUserRequest{
			UserBase: model.UserBase{
				Username: "duplicate",
				Email:    "dup@example.com",
			},
		}
		_, err := repo.CreateUser(req, "pass", "token")
		assert.NoError(t, err)

		_, err = repo.CreateUser(req, "pass", "token")
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

func TestUserRepository_GetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	// Setup data
	req := &request.CreateUserRequest{
		UserBase: model.UserBase{
			Username: "findme",
			Email:    "find@example.com",
		},
	}
	repo.CreateUser(req, "pass", "token")

	t.Run("User Found", func(t *testing.T) {
		user, err := repo.GetUserByUsername("findme")
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "findme", user.Username)
	})

	t.Run("User Not Found", func(t *testing.T) {
		user, err := repo.GetUserByUsername("ghost")
		assert.NoError(t, err)
		assert.Nil(t, user)
	})
}

func TestUserRepository_UpdateUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	// Setup data
	req := &request.CreateUserRequest{
		UserBase: model.UserBase{
			Username: "update_target",
			Email:    "update@example.com",
			Nickname: "Original",
		},
	}
	repo.CreateUser(req, "pass", "token")

	t.Run("Update Fields", func(t *testing.T) {
		newNick := "Updated Nick"
		newQQ := 123456
		updateReq := &request.UpdateUserRequest{
			Nickname: &newNick,
			QQNumber: &newQQ,
		}

		user, err := repo.UpdateUser("update_target", updateReq)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Nick", user.Nickname)
		assert.NotNil(t, user.QQNumber)
		assert.Equal(t, 123456, *user.QQNumber)
	})

	t.Run("Idempotency Check (PUT semantics)", func(t *testing.T) {
		newAccount := "new_account"
		updateReq := &request.UpdateUserRequest{
			Account: &newAccount,
		}
		user, err := repo.UpdateUser("update_target", updateReq)
		assert.NoError(t, err)
		assert.Equal(t, "new_account", *user.Account)
		assert.Equal(t, "Updated Nick", user.Nickname) // Should persist from previous update
	})
}
