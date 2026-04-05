package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"paradigm-reboot-prober-go/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// VerifyPassword checks if the provided plain password matches the encoded password.
func VerifyPassword(plainPassword, encodedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(encodedPassword), []byte(plainPassword))
	return err == nil
}

// EncodePassword hashes the password using bcrypt.
func EncodePassword(password string) (string, error) {
	if len(password) > 72 {
		return "", errors.New("password must not exceed 72 bytes")
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), config.GlobalConfig.Auth.BcryptCost)
	return string(bytes), err
}

// GenerateJWT generates a new JWT token with the given claims and expiration.
func GenerateJWT(claims jwt.MapClaims, expiresDelta *time.Duration) (string, error) {
	var expire time.Time
	if expiresDelta != nil {
		expire = time.Now().Add(*expiresDelta)
	} else {
		expire = time.Now().Add(config.JWTExpirationDuration)
	}

	claims["exp"] = expire.Unix()
	claims["iat"] = time.Now().Unix()

	// Generate a random JWT ID for potential token revocation
	jtiBytes := make([]byte, 16)
	if _, err := rand.Read(jtiBytes); err != nil {
		return "", err
	}
	claims["jti"] = hex.EncodeToString(jtiBytes)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(config.GlobalConfig.Auth.SecretKey))
}

// GenerateAccessJWT generates an access token for the given username.
func GenerateAccessJWT(username string, expiresDelta *time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
	}
	return GenerateJWT(claims, expiresDelta)
}

// ExtractPayloads parses the token and returns the claims.
func ExtractPayloads(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.GlobalConfig.Auth.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ExtractUsername extracts the username from the JWT token.
func ExtractUsername(tokenString string) (string, error) {
	claims, err := ExtractPayloads(tokenString)
	if err != nil {
		return "", err
	}

	if sub, ok := claims["sub"].(string); ok {
		return sub, nil
	}

	return "", errors.New("username not found in token")
}
