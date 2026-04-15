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
	assert.Equal(t, "access", claims["type"])

	extractedUsername, err := ExtractUsername(token)
	assert.NoError(t, err)
	assert.Equal(t, username, extractedUsername)
}

func TestRefreshJWTGeneration(t *testing.T) {
	username := "testuser"
	duration := time.Hour * 24
	token, err := GenerateRefreshJWT(username, &duration)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := ExtractPayloads(token)
	assert.NoError(t, err)
	assert.Equal(t, username, claims["sub"])
	assert.Equal(t, "refresh", claims["type"])

	// Ensure we can extract username from refresh token
	extractedUsername, err := ExtractUsername(token)
	assert.NoError(t, err)
	assert.Equal(t, username, extractedUsername)
}

func TestExtractTokenType(t *testing.T) {
	t.Run("Access token", func(t *testing.T) {
		duration := time.Minute * 15
		token, err := GenerateAccessJWT("testuser", &duration)
		assert.NoError(t, err)

		tokenType, err := ExtractTokenType(token)
		assert.NoError(t, err)
		assert.Equal(t, "access", tokenType)
	})

	t.Run("Refresh token", func(t *testing.T) {
		duration := time.Hour * 24
		token, err := GenerateRefreshJWT("testuser", &duration)
		assert.NoError(t, err)

		tokenType, err := ExtractTokenType(token)
		assert.NoError(t, err)
		assert.Equal(t, "refresh", tokenType)
	})

	t.Run("Legacy token without type", func(t *testing.T) {
		// Simulate a legacy token without type claim
		claims := jwt.MapClaims{
			"sub": "testuser",
		}
		token, err := GenerateJWT(claims, nil)
		assert.NoError(t, err)

		tokenType, err := ExtractTokenType(token)
		assert.NoError(t, err)
		assert.Equal(t, "", tokenType)
	})

	t.Run("Invalid token", func(t *testing.T) {
		_, err := ExtractTokenType("invalid.token.string")
		assert.Error(t, err)
	})
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
