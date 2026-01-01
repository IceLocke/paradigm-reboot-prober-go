package config

import "os"

var (
	// SecretKey is used to sign JWT tokens.
	// In production, this should be loaded from environment variables.
	SecretKey = getEnv("SECRET_KEY", "your_secret_key_here")

	// JWTAlgorithm is the signing algorithm.
	// Go's jwt package handles this via the SigningMethod type, but we keep the config here.
	JWTAlgorithm = "HS256"
)

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
