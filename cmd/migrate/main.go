package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"paradigm-reboot-prober-go/config"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	configPath := flag.String("config", "config/config.yaml", "Path to config file")
	sqlFile := flag.String("sql-file", "", "Path to migration SQL file (default: legacy/migration.sql relative to executable)")
	dryRun := flag.Bool("dry-run", false, "Print the SQL without executing")
	flag.Parse()

	// Load config (reuse project's config system for DB connection info)
	config.LoadConfig(*configPath)

	dbConfig := config.GlobalConfig.Database
	if dbConfig.Type != "postgres" {
		log.Fatalf("Migration only supports PostgreSQL. Current database type: %s", dbConfig.Type)
	}

	// Resolve SQL file path
	sqlPath := *sqlFile
	if sqlPath == "" {
		// Default: legacy/migration.sql relative to working directory
		sqlPath = filepath.Join("legacy", "migration.sql")
	}

	// Read migration SQL
	sqlBytes, err := os.ReadFile(sqlPath)
	if err != nil {
		log.Fatalf("Failed to read migration SQL file %s: %v", sqlPath, err)
	}
	migrationSQL := string(sqlBytes)

	// Strip psql meta-commands (\restrict, \unrestrict) that are not valid in SQL
	var cleanedLines []string
	for _, line := range strings.Split(migrationSQL, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "\\restrict") || strings.HasPrefix(trimmed, "\\unrestrict") {
			continue
		}
		cleanedLines = append(cleanedLines, line)
	}
	migrationSQL = strings.Join(cleanedLines, "\n")

	if *dryRun {
		fmt.Println("=== DRY RUN MODE — SQL to be executed ===")
		fmt.Println(migrationSQL)
		fmt.Println("=== END OF SQL ===")
		return
	}

	// Build DSN
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		dbConfig.Host, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.Port, dbConfig.SSLMode)

	// Connect to database
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("Failed to open database connection: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Verify connection
	if err := db.PingContext(context.Background()); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to PostgreSQL database successfully.")

	// Execute migration
	log.Println("Starting migration...")
	_, err = db.ExecContext(context.Background(), migrationSQL)
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Migration completed successfully!")
	log.Println("")
	log.Println("Next steps:")
	log.Println("  1. Start the Go server to let GORM AutoMigrate apply any remaining minor adjustments.")
	log.Println("  2. Verify the application works correctly with the migrated data.")
	log.Println("  3. Optionally drop the _legacy_best50_trend table after confirming migration success.")
}
