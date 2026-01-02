package util

import (
	"fmt"
	"log"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	var dialector gorm.Dialector

	dbConfig := config.GlobalConfig.Database

	switch dbConfig.Type {
	case "postgres":
		dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
			dbConfig.Host, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.Port, dbConfig.SSLMode)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(dbConfig.DSN)
	default:
		log.Fatalf("Unsupported database type: %s", dbConfig.Type)
	}

	DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models
	err = DB.AutoMigrate(
		&model.User{},
		&model.Song{},
		&model.SongLevel{},
		&model.PlayRecord{},
		&model.BestPlayRecord{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
}
