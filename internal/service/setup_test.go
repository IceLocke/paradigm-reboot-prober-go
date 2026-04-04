package service

import (
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	config.InitDefaults()
	config.GlobalConfig.Auth.SecretKey = "testsecret"

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(
		&model.User{},
		&model.Song{},
		&model.Chart{},
		&model.PlayRecord{},
		&model.BestPlayRecord{},
	)
	if err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}
