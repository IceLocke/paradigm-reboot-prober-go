package main

import (
	"fmt"
	"log"
	"os"

	"paradigm-reboot-prober-go/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dsn := "host=localhost port=5432 user=postgres dbname=prober_migration_test sslmode=disable"
	if envDSN := os.Getenv("VERIFY_DSN"); envDSN != "" {
		dsn = envDSN
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	fmt.Println("✅ Connected to database")

	// Step 1: AutoMigrate — should be a no-op on a correctly migrated DB
	fmt.Println("\n--- Running GORM AutoMigrate ---")
	err = db.AutoMigrate(
		&model.User{},
		&model.Song{},
		&model.Chart{},
		&model.PlayRecord{},
		&model.BestPlayRecord{},
	)
	if err != nil {
		log.Fatalf("❌ AutoMigrate failed: %v", err)
	}
	fmt.Println("✅ AutoMigrate succeeded (no schema conflicts)")

	// Step 2: Row counts
	fmt.Println("\n--- Row Counts ---")
	tables := []struct {
		name  string
		model interface{}
	}{
		{"prober_users", &model.User{}},
		{"songs", &model.Song{}},
		{"charts", &model.Chart{}},
		{"play_records", &model.PlayRecord{}},
		{"best_play_records", &model.BestPlayRecord{}},
	}
	for _, t := range tables {
		var count int64
		db.Model(t.model).Count(&count)
		fmt.Printf("  %-25s %d rows\n", t.name, count)
	}

	// Step 3: Sample data checks
	fmt.Println("\n--- Sample Data Checks ---")

	// Check a user
	var user model.User
	if err := db.First(&user).Error; err != nil {
		fmt.Printf("  ❌ User query failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ User: ID=%d, Username=%s\n", user.ID, user.Username)
	}

	// Check a song with charts
	var song model.Song
	if err := db.Preload("Charts").First(&song).Error; err != nil {
		fmt.Printf("  ❌ Song query failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ Song: ID=%d, Title=%s, Charts=%d\n", song.ID, song.Title, len(song.Charts))
		for _, c := range song.Charts {
			fmt.Printf("     Chart: ID=%d, SongID=%d, Difficulty=%s, Level=%.1f\n", c.ID, c.SongID, c.Difficulty, c.Level)
		}
	}

	// Check a play record with chart preload
	var pr model.PlayRecord
	if err := db.Preload("Chart").First(&pr).Error; err != nil {
		fmt.Printf("  ❌ PlayRecord query failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ PlayRecord: ID=%d, Username=%s, ChartID=%d, Score=%d, Rating=%d\n",
			pr.ID, pr.Username, pr.ChartID, *pr.Score, pr.Rating)
		if pr.Chart != nil {
			fmt.Printf("     Chart: ID=%d, Difficulty=%s\n", pr.Chart.ID, pr.Chart.Difficulty)
		}
	}

	// Check a best play record with play_record preload
	var bpr model.BestPlayRecord
	if err := db.Preload("PlayRecord").First(&bpr).Error; err != nil {
		fmt.Printf("  ❌ BestPlayRecord query failed: %v\n", err)
	} else {
		fmt.Printf("  ✅ BestPlayRecord: ID=%d, Username=%s, ChartID=%d, PlayRecordID=%d\n",
			bpr.ID, bpr.Username, bpr.ChartID, bpr.PlayRecordID)
		if bpr.PlayRecord != nil {
			fmt.Printf("     PlayRecord: ID=%d, Score=%d\n", bpr.PlayRecord.ID, *bpr.PlayRecord.Score)
		}
	}

	fmt.Println("\n🎉 Verification complete!")
}
