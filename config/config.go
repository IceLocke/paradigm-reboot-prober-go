package config

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	Logging struct {
		Output       string   `yaml:"output"`        // "stdout" (default), "stderr", or "file"
		File         string   `yaml:"file"`          // file path when Output == "file"
		Format       string   `yaml:"format"`        // "text" (default) or "json"
		ExcludePaths []string `yaml:"exclude_paths"` // request paths starting with any of these prefixes are not logged by SlogRequestMiddleware
	} `yaml:"logging"`
	Metrics struct {
		Enabled      bool     `yaml:"enabled"`       // when true, HTTP metrics middleware is installed and a separate metrics server is started
		Addr         string   `yaml:"addr"`          // listen address for the metrics HTTP server, e.g. ":9090" (must not overlap with Server.Port)
		Path         string   `yaml:"path"`          // URL path that serves Prometheus metrics, e.g. "/metrics"
		ExcludePaths []string `yaml:"exclude_paths"` // Gin route templates starting with any of these prefixes are not counted in HTTP metrics
	} `yaml:"metrics"`
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
	GlobalConfig.Logging.Output = "stdout"
	GlobalConfig.Logging.File = ""
	GlobalConfig.Logging.Format = "text"
	GlobalConfig.Logging.ExcludePaths = []string{"/healthz"}
	GlobalConfig.Metrics.Enabled = true
	GlobalConfig.Metrics.Addr = ":9090"
	GlobalConfig.Metrics.Path = "/metrics"
	GlobalConfig.Metrics.ExcludePaths = []string{"/healthz"}

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
	if out := os.Getenv("LOG_OUTPUT"); out != "" {
		GlobalConfig.Logging.Output = out
	}
	if f := os.Getenv("LOG_FILE"); f != "" {
		GlobalConfig.Logging.File = f
	}
	if fmtEnv := os.Getenv("LOG_FORMAT"); fmtEnv != "" {
		GlobalConfig.Logging.Format = fmtEnv
	}
	if paths := os.Getenv("LOG_EXCLUDE_PATHS"); paths != "" {
		parts := strings.Split(paths, ",")
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		GlobalConfig.Logging.ExcludePaths = cleaned
	}
	if v := os.Getenv("METRICS_ENABLED"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			GlobalConfig.Metrics.Enabled = b
		} else {
			log.Fatalf("Invalid METRICS_ENABLED value %q: %v", v, err)
		}
	}
	if v := os.Getenv("METRICS_ADDR"); v != "" {
		GlobalConfig.Metrics.Addr = v
	}
	if v := os.Getenv("METRICS_PATH"); v != "" {
		GlobalConfig.Metrics.Path = v
	}
	if paths := os.Getenv("METRICS_EXCLUDE_PATHS"); paths != "" {
		parts := strings.Split(paths, ",")
		cleaned := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		GlobalConfig.Metrics.ExcludePaths = cleaned
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

	// Validate logging output
	switch GlobalConfig.Logging.Output {
	case "stdout", "stderr":
		// ok
	case "file":
		if strings.TrimSpace(GlobalConfig.Logging.File) == "" {
			log.Fatalf("logging.output=file requires logging.file to be set")
		}
	default:
		log.Fatalf("Invalid logging.output %q: must be one of stdout, stderr, file", GlobalConfig.Logging.Output)
	}

	// Validate logging format
	switch GlobalConfig.Logging.Format {
	case "", "text", "json":
		// ok
	default:
		log.Fatalf("Invalid logging.format %q: must be one of text, json", GlobalConfig.Logging.Format)
	}

	// Validate metrics
	if GlobalConfig.Metrics.Enabled {
		if strings.TrimSpace(GlobalConfig.Metrics.Addr) == "" {
			log.Fatalf("metrics.enabled=true requires metrics.addr to be set (e.g. \":9090\")")
		}
		if !strings.HasPrefix(GlobalConfig.Metrics.Path, "/") {
			log.Fatalf("Invalid metrics.path %q: must start with '/'", GlobalConfig.Metrics.Path)
		}
		if GlobalConfig.Metrics.Addr == GlobalConfig.Server.Port {
			log.Fatalf("metrics.addr must differ from server.port (both are %q)", GlobalConfig.Metrics.Addr)
		}
	}
}
