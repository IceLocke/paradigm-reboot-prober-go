package repository

import (
	"errors"
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
		createdUser, err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.NotNil(t, createdUser)
		assert.Equal(t, "testuser", createdUser.Username)
		assert.Equal(t, "Test User", createdUser.Nickname)
		assert.Equal(t, "token123", createdUser.UploadToken)
		assert.True(t, createdUser.IsActive)
		assert.False(t, createdUser.IsAdmin)
	})

	t.Run("Create User Default Nickname", func(t *testing.T) {
		user := &model.User{
			UserBase: model.UserBase{
				Username:    "user_no_nick",
				Email:       "nonick@example.com",
				UploadToken: "token_nonick",
			},
			EncodedPassword: "pass",
		}
		createdUser, err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.Equal(t, "user_no_nick", createdUser.Nickname)
	})

	t.Run("Create Duplicate User", func(t *testing.T) {
		user1 := &model.User{
			UserBase: model.UserBase{
				Username:    "duplicate",
				Email:       "dup@example.com",
				UploadToken: "token_dup1",
			},
			EncodedPassword: "pass",
		}
		_, err := repo.CreateUser(user1)
		assert.NoError(t, err)

		user2 := &model.User{
			UserBase: model.UserBase{
				Username:    "duplicate",
				Email:       "dup@example.com",
				UploadToken: "token_dup2",
			},
			EncodedPassword: "pass",
		}
		_, err = repo.CreateUser(user2)
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
			UploadToken: "token_findme",
		},
		EncodedPassword: "pass",
	}
	_, err := repo.CreateUser(user)
	assert.NoError(t, err)

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
			UploadToken: "token_update",
		},
		EncodedPassword: "pass",
	}
	_, err := repo.CreateUser(user)
	assert.NoError(t, err)

	t.Run("Update Fields", func(t *testing.T) {
		// Fetch user first
		userToUpdate, _ := repo.GetUserByUsername("update_target")

		newNick := "Updated Nick"
		newQQ := 123456

		userToUpdate.Nickname = newNick
		userToUpdate.QQNumber = &newQQ

		updatedUser, err := repo.UpdateUser(userToUpdate)
		assert.NoError(t, err)
		assert.Equal(t, "Updated Nick", updatedUser.Nickname)

		// Verify
		fetchedUser, _ := repo.GetUserByUsername("update_target")
		assert.Equal(t, "Updated Nick", fetchedUser.Nickname)
		assert.NotNil(t, fetchedUser.QQNumber)
		assert.Equal(t, 123456, *fetchedUser.QQNumber)
	})

	t.Run("Idempotency Check (PUT semantics)", func(t *testing.T) {
		userToUpdate, _ := repo.GetUserByUsername("update_target")
		newAccount := "new_account"
		userToUpdate.Account = &newAccount

		_, err := repo.UpdateUser(userToUpdate)
		assert.NoError(t, err)

		updatedUser, _ := repo.GetUserByUsername("update_target")
		assert.Equal(t, "new_account", *updatedUser.Account)
		assert.Equal(t, "Updated Nick", updatedUser.Nickname) // Should persist from previous update
	})
}

func TestUserRepository_WithTransaction(t *testing.T) {
	db := setupTestDB(t)
	repo := NewUserRepository(db)

	t.Run("Successful transaction commits", func(t *testing.T) {
		err := repo.WithTransaction(func(txRepo *UserRepository) error {
			user := &model.User{
				UserBase: model.UserBase{
					Username:    "tx_commit",
					Email:       "txcommit@example.com",
					UploadToken: "token_tx_commit",
				},
				EncodedPassword: "pass",
			}
			_, err := txRepo.CreateUser(user)
			return err
		})
		assert.NoError(t, err)

		found, err := repo.GetUserByUsername("tx_commit")
		assert.NoError(t, err)
		assert.NotNil(t, found)
		assert.Equal(t, "tx_commit", found.Username)
	})

	t.Run("Error rolls back transaction", func(t *testing.T) {
		err := repo.WithTransaction(func(txRepo *UserRepository) error {
			user := &model.User{
				UserBase: model.UserBase{
					Username:    "tx_rollback",
					Email:       "txrb@example.com",
					UploadToken: "token_tx_rollback",
				},
				EncodedPassword: "pass",
			}
			if _, err := txRepo.CreateUser(user); err != nil {
				return err
			}
			return errors.New("forced rollback")
		})
		assert.Error(t, err)
		assert.Equal(t, "forced rollback", err.Error())

		// User should NOT exist after rollback
		found, err := repo.GetUserByUsername("tx_rollback")
		assert.NoError(t, err)
		assert.Nil(t, found)
	})
}
