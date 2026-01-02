package config

import (
	"log"
	"os"

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
		SecretKey    string `yaml:"secret_key"`
		JWTAlgorithm string `yaml:"jwt_algorithm"`
	} `yaml:"auth"`
	Upload struct {
		CSVPath string `yaml:"csv_path"`
		ImgPath string `yaml:"img_path"`
	} `yaml:"upload"`
}

var GlobalConfig Config

func LoadConfig(configPath string) {
	// Set defaults
	GlobalConfig.Server.Port = ":8080"
	GlobalConfig.Database.Type = "sqlite"
	GlobalConfig.Database.DSN = "prober.db"
	GlobalConfig.Auth.SecretKey = "your_secret_key_here"
	GlobalConfig.Auth.JWTAlgorithm = "HS256"
	GlobalConfig.Upload.CSVPath = "./uploads/csv/"
	GlobalConfig.Upload.ImgPath = "./uploads/img/"

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
	if secret := os.Getenv("SECRET_KEY"); secret != "" {
		GlobalConfig.Auth.SecretKey = secret
	}
}
