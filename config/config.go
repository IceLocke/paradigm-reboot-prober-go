package config

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		Type     string `yaml:"type"` // sqlite or postgres
		DSN      string `yaml:"dsn"`
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
		SSLMode  string `yaml:"sslmode"`
	} `yaml:"database"`
	Auth struct {
		SecretKey              string `yaml:"secret_key"`
		JWTAlgorithm           string `yaml:"jwt_algorithm"`
		JWTExpiration          string `yaml:"jwt_expiration"`           // duration string, e.g. "24h", "30m"
		RefreshTokenExpiration string `yaml:"refresh_token_expiration"` // duration string, e.g. "168h"
		BcryptCost             int    `yaml:"bcrypt_cost"`
		UploadTokenLength      int    `yaml:"upload_token_length"` // bytes (hex output is 2x)
		UsernamePattern        string `yaml:"username_pattern"`
	} `yaml:"auth"`
	Pagination struct {
		DefaultPageSize int `yaml:"default_page_size"`
		MaxPageSize     int `yaml:"max_page_size"`
	} `yaml:"pagination"`
	Game struct {
		B35Limit int `yaml:"b35_limit"`
		B15Limit int `yaml:"b15_limit"`
	} `yaml:"game"`
}

var GlobalConfig Config

// Parsed values derived from config strings
var (
	JWTExpirationDuration          time.Duration
	RefreshTokenExpirationDuration time.Duration
	UsernameRegex                  *regexp.Regexp
)

// InitDefaults sets all config fields to their default values and parses derived values.
// Useful for testing scenarios where LoadConfig is not called.
func InitDefaults() {
	GlobalConfig.Server.Port = ":8080"
	GlobalConfig.Database.Type = "sqlite"
	GlobalConfig.Database.DSN = "prober.db"
	GlobalConfig.Auth.SecretKey = "your_secret_key_here"
	GlobalConfig.Auth.JWTAlgorithm = "HS256"
	GlobalConfig.Auth.JWTExpiration = "30m"
	GlobalConfig.Auth.RefreshTokenExpiration = "168h"
	GlobalConfig.Auth.BcryptCost = 10
	GlobalConfig.Auth.UploadTokenLength = 16
	GlobalConfig.Auth.UsernamePattern = `^[a-z][a-z0-9_]{5,15}$`
	GlobalConfig.Pagination.DefaultPageSize = 50
	GlobalConfig.Pagination.MaxPageSize = 200
	GlobalConfig.Game.B35Limit = 35
	GlobalConfig.Game.B15Limit = 15

	// Parse derived values (defaults are always valid, no error expected)
	JWTExpirationDuration, _ = time.ParseDuration(GlobalConfig.Auth.JWTExpiration)
	RefreshTokenExpirationDuration, _ = time.ParseDuration(GlobalConfig.Auth.RefreshTokenExpiration)
	UsernameRegex = regexp.MustCompile(GlobalConfig.Auth.UsernamePattern)
}

func LoadConfig(configPath string) {
	// Set defaults
	InitDefaults()

	// Read from file
	file, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("Warning: Config file not found at %s, using defaults and environment variables", configPath)
	} else {
		err = yaml.Unmarshal(file, &GlobalConfig)
		if err != nil {
			log.Fatalf("Error parsing config file: %v", err)
		}
	}

	// Override with environment variables if present
	if port := os.Getenv("SERVER_PORT"); port != "" {
		GlobalConfig.Server.Port = port
	}
	if dbType := os.Getenv("DB_TYPE"); dbType != "" {
		GlobalConfig.Database.Type = dbType
	}
	if dsn := os.Getenv("DB_DSN"); dsn != "" {
		GlobalConfig.Database.DSN = dsn
	}
	if host := os.Getenv("DB_HOST"); host != "" {
		GlobalConfig.Database.Host = host
	}
	if portStr := os.Getenv("DB_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			GlobalConfig.Database.Port = p
		}
	}
	if user := os.Getenv("DB_USER"); user != "" {
		GlobalConfig.Database.User = user
	}
	if pass := os.Getenv("DB_PASSWORD"); pass != "" {
		GlobalConfig.Database.Password = pass
	}
	if name := os.Getenv("DB_NAME"); name != "" {
		GlobalConfig.Database.DBName = name
	}
	if ssl := os.Getenv("DB_SSLMODE"); ssl != "" {
		GlobalConfig.Database.SSLMode = ssl
	}
	if secret := os.Getenv("SECRET_KEY"); secret != "" {
		GlobalConfig.Auth.SecretKey = secret
	}
	// Re-parse derived values after file/env overrides
	JWTExpirationDuration, err = time.ParseDuration(GlobalConfig.Auth.JWTExpiration)
	if err != nil {
		log.Fatalf("Invalid jwt_expiration value %q: %v", GlobalConfig.Auth.JWTExpiration, err)
	}

	RefreshTokenExpirationDuration, err = time.ParseDuration(GlobalConfig.Auth.RefreshTokenExpiration)
	if err != nil {
		log.Fatalf("Invalid refresh_token_expiration value %q: %v", GlobalConfig.Auth.RefreshTokenExpiration, err)
	}

	UsernameRegex, err = regexp.Compile(GlobalConfig.Auth.UsernamePattern)
	if err != nil {
		log.Fatalf("Invalid username_pattern %q: %v", GlobalConfig.Auth.UsernamePattern, err)
	}

	// Validate bcrypt cost
	if GlobalConfig.Auth.BcryptCost < 4 || GlobalConfig.Auth.BcryptCost > 31 {
		log.Fatalf("Invalid bcrypt_cost %d: must be between 4 and 31", GlobalConfig.Auth.BcryptCost)
	}
}
