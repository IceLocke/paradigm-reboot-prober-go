package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUser_ToPublic(t *testing.T) {
	qq := "12345678"
	account := "act_001"
	accountNumber := 1001
	uuid := "550e8400-e29b-41d4-a716-446655440000"

	user := &User{
		ID: 42,
		UserBase: UserBase{
			Username:       "testuser",
			Email:          "test@example.com",
			Nickname:       "TestNick",
			QQAccount:      &qq,
			Account:        &account,
			AccountNumber:  &accountNumber,
			UUID:           &uuid,
			AnonymousProbe: true,
			UploadToken:    "secret_token",
			IsActive:       true,
			IsAdmin:        true,
		},
		EncodedPassword: "hashed_password",
	}

	pub := user.ToPublic()

	assert.Equal(t, 42, pub.ID)
	assert.Equal(t, "testuser", pub.Username)
	assert.Equal(t, "test@example.com", pub.Email)
	assert.Equal(t, "TestNick", pub.Nickname)
	assert.Equal(t, &qq, pub.QQAccount)
	assert.Equal(t, &account, pub.Account)
	assert.Equal(t, &accountNumber, pub.AccountNumber)
	assert.Equal(t, &uuid, pub.UUID)
	assert.Equal(t, true, pub.AnonymousProbe)
}

func TestUser_ToPublic_NilOptionalFields(t *testing.T) {
	user := &User{
		ID: 1,
		UserBase: UserBase{
			Username:       "minuser",
			Email:          "min@example.com",
			Nickname:       "Min",
			AnonymousProbe: false,
			UploadToken:    "tok",
			IsActive:       true,
			IsAdmin:        false,
		},
		EncodedPassword: "enc",
	}

	pub := user.ToPublic()

	assert.Equal(t, 1, pub.ID)
	assert.Equal(t, "minuser", pub.Username)
	assert.Nil(t, pub.QQAccount)
	assert.Nil(t, pub.Account)
	assert.Nil(t, pub.AccountNumber)
	assert.Nil(t, pub.UUID)
	assert.Equal(t, false, pub.AnonymousProbe)
}
