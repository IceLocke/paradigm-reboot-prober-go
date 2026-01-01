package repository

import (
	"paradigm-reboot-prober-go/internal/model"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserRepository_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("Create User Success", func(t *testing.T) {
		user := &model.User{
			UserBase: model.UserBase{
				Username:    "testuser",
				Email:       "test@example.com",
				Nickname:    "Test User",
				UploadToken: "token123",
				IsActive:    true,
				IsAdmin:     false,
			},
			EncodedPassword: "encoded_password",
		}
		err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "Test User", user.Nickname)
		assert.Equal(t, "token123", user.UploadToken)
		assert.True(t, user.IsActive)
		assert.False(t, user.IsAdmin)
	})

	t.Run("Create User Default Nickname", func(t *testing.T) {
		user := &model.User{
			UserBase: model.UserBase{
				Username:    "user_no_nick",
				Email:       "nonick@example.com",
				UploadToken: "token",
			},
			EncodedPassword: "pass",
		}
		err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.Equal(t, "user_no_nick", user.Nickname)
	})

	t.Run("Create Duplicate User", func(t *testing.T) {
		user1 := &model.User{
			UserBase: model.UserBase{
				Username:    "duplicate",
				Email:       "dup@example.com",
				UploadToken: "token",
			},
			EncodedPassword: "pass",
		}
		err := repo.CreateUser(user1)
		assert.NoError(t, err)

		user2 := &model.User{
			UserBase: model.UserBase{
				Username:    "duplicate",
				Email:       "dup@example.com",
				UploadToken: "token",
			},
			EncodedPassword: "pass",
		}
		err = repo.CreateUser(user2)
		assert.Error(t, err) // Should fail due to unique constraint
	})
}

func TestUserRepository_GetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	// Setup data
	user := &model.User{
		UserBase: model.UserBase{
			Username:    "findme",
			Email:       "find@example.com",
			UploadToken: "token",
		},
		EncodedPassword: "pass",
	}
	repo.CreateUser(user)

	t.Run("User Found", func(t *testing.T) {
		foundUser, err := repo.GetUserByUsername("findme")
		assert.NoError(t, err)
		assert.NotNil(t, foundUser)
		assert.Equal(t, "findme", foundUser.Username)
	})

	t.Run("User Not Found", func(t *testing.T) {
		foundUser, err := repo.GetUserByUsername("ghost")
		assert.NoError(t, err)
		assert.Nil(t, foundUser)
	})
}

func TestUserRepository_UpdateUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	// Setup data
	user := &model.User{
		UserBase: model.UserBase{
			Username:    "update_target",
			Email:       "update@example.com",
			Nickname:    "Original",
			UploadToken: "token",
		},
		EncodedPassword: "pass",
	}
	repo.CreateUser(user)

	t.Run("Update Fields", func(t *testing.T) {
		// Fetch user first
		userToUpdate, _ := repo.GetUserByUsername("update_target")

		newNick := "Updated Nick"
		newQQ := 123456

		userToUpdate.Nickname = newNick
		userToUpdate.QQNumber = &newQQ

		err := repo.UpdateUser(userToUpdate)
		assert.NoError(t, err)

		// Verify
		updatedUser, _ := repo.GetUserByUsername("update_target")
		assert.Equal(t, "Updated Nick", updatedUser.Nickname)
		assert.NotNil(t, updatedUser.QQNumber)
		assert.Equal(t, 123456, *updatedUser.QQNumber)
	})

	t.Run("Idempotency Check (PUT semantics)", func(t *testing.T) {
		userToUpdate, _ := repo.GetUserByUsername("update_target")
		newAccount := "new_account"
		userToUpdate.Account = &newAccount

		err := repo.UpdateUser(userToUpdate)
		assert.NoError(t, err)

		updatedUser, _ := repo.GetUserByUsername("update_target")
		assert.Equal(t, "new_account", *updatedUser.Account)
		assert.Equal(t, "Updated Nick", updatedUser.Nickname) // Should persist from previous update
	})
}
