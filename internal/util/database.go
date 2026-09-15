package util

import (
	"fmt"
	"log"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"strings"

	"github.com/glebarez/sqlite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	var dialector gorm.Dialector

	dbConfig := config.GlobalConfig.Database

	switch dbConfig.Type {
	case "postgres":
		// pgx's DSN parser mis-handles an unquoted empty value (e.g. `password= dbname=...`
		// swallows the following keyword as the password), so always single-quote the
		// password and escape embedded quotes/backslashes per libpq rules.
		escapedPassword := strings.ReplaceAll(strings.ReplaceAll(dbConfig.Password, `\`, `\\`), `'`, `\'`)
		dsn := fmt.Sprintf("host=%s user=%s password='%s' dbname=%s port=%d sslmode=%s",
			dbConfig.Host, dbConfig.User, escapedPassword, dbConfig.DBName, dbConfig.Port, dbConfig.SSLMode)
		dialector = postgres.Open(dsn)
	case "sqlite":
		dialector = sqlite.Open(dbConfig.DSN)
	default:
		log.Fatalf("Unsupported database type: %s", dbConfig.Type)
	}

	DB, err = gorm.Open(dialector, &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models
	err = DB.AutoMigrate(
		&model.User{},
		&model.Song{},
		&model.Chart{},
		&model.PlayRecord{},
		&model.BestPlayRecord{},
		// chart_statistics is owned by the fitting-calculator microservice (cmd/fitting);
		// migrating it here ensures the schema exists regardless of which binary starts first.
		&model.ChartStatistic{},
	)
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
}
