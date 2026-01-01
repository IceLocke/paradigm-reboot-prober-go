package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestPasswordHashing(t *testing.T) {
	password := "mysecretpassword"
	encoded, err := EncodePassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, encoded)
	assert.NotEqual(t, password, encoded)

	match := VerifyPassword(password, encoded)
	assert.True(t, match)

	match = VerifyPassword("wrongpassword", encoded)
	assert.False(t, match)
}

func TestJWTGenerationAndExtraction(t *testing.T) {
	username := "testuser"
	duration := time.Minute * 15
	token, err := GenerateAccessJWT(username, &duration)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ExtractPayloads(token)
	assert.NoError(t, err)
	assert.Equal(t, username, claims["sub"])

	extractedUsername, err := ExtractUsername(token)
	assert.NoError(t, err)
	assert.Equal(t, username, extractedUsername)
}

func TestExpiredToken(t *testing.T) {
	username := "testuser"
	duration := -time.Minute // Expired
	token, err := GenerateAccessJWT(username, &duration)
	assert.NoError(t, err)

	_, err = ExtractPayloads(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token is expired")
}

func TestCustomClaims(t *testing.T) {
	claims := jwt.MapClaims{
		"foo": "bar",
		"sub": "custom_sub",
	}
	token, err := GenerateJWT(claims, nil)
	assert.NoError(t, err)

	extracted, err := ExtractPayloads(token)
	assert.NoError(t, err)
	assert.Equal(t, "bar", extracted["foo"])
	assert.Equal(t, "custom_sub", extracted["sub"])
}
