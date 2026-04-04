package repository

import (
	"errors"
	"paradigm-reboot-prober-go/config"
	"paradigm-reboot-prober-go/internal/model"
	"paradigm-reboot-prober-go/pkg/rating"
	"time"

	"gorm.io/gorm"
)

// allowedSortColumns defines the whitelist of columns that can be used for sorting
var allowedSortColumns = map[string]string{
	"rating":      "rating",
	"score":       "score",
	"record_time": "record_time",
}

// validateSortBy returns a safe column name for ORDER BY, defaulting to "rating" if not allowed
func validateSortBy(sortBy string) string {
	if col, ok := allowedSortColumns[sortBy]; ok {
		return col
	}
	return "rating" // default
}

type RecordRepository struct {
	db *gorm.DB
}

func NewRecordRepository(db *gorm.DB) *RecordRepository {
	return &RecordRepository{db: db}
}

// CreateRecord creates a new play record and updates the best record if necessary
func (r *RecordRepository) CreateRecord(record *model.PlayRecord, isReplaced bool) (*model.PlayRecord, error) {
	var chart model.Chart
	if err := r.db.Where("chart_id = ?", record.ChartID).First(&chart).Error; err != nil {
		return nil, errors.New("chart does not exist")
	}

	// Calculate rating
	calculatedRating := rating.SingleRating(chart.Level, record.Score)
	record.Rating = calculatedRating
	record.RecordTime = time.Now()

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		var bestRecord model.BestPlayRecord
		result := tx.Joins("PlayRecord").
			Where("PlayRecord.chart_id = ? AND PlayRecord.username = ?", record.ChartID, record.Username).
			First(&bestRecord)

		if result.Error == nil {
			// Best record exists
			if isReplaced || record.Score > bestRecord.PlayRecord.Score {
				bestRecord.PlayRecordID = record.PlayRecordID
				bestRecord.PlayRecord = record
				if err := tx.Save(&bestRecord).Error; err != nil {
					return err
				}
			}
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Create new best record
			newBestRecord := model.BestPlayRecord{
				PlayRecordID: record.PlayRecordID,
				PlayRecord:   record,
			}
			if err := tx.Create(&newBestRecord).Error; err != nil {
				return err
			}
		} else {
			return result.Error
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return record, nil
}

// GetBest50Records retrieves the best 35 (old) and 15 (new) records for B50 calculation
func (r *RecordRepository) GetBest50Records(username string, underflow int) ([]model.PlayRecord, []model.PlayRecord, error) {
	var b35 []model.PlayRecord
	var b15 []model.PlayRecord

	// Base query for best records
	baseQuery := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.play_record_id").
		Joins("Chart").
		Joins("Chart.Song").
		Where("play_records.username = ?", username)

	// B35: Not B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where("Chart__Song.b15 = ?", false).
		Order("rating desc").
		Limit(config.GlobalConfig.Game.B35Limit + underflow).
		Find(&b35).Error; err != nil {
		return nil, nil, err
	}

	// B15: B15 songs
	if err := baseQuery.Session(&gorm.Session{}).
		Where("Chart__Song.b15 = ?", true).
		Order("rating desc").
		Limit(config.GlobalConfig.Game.B15Limit + underflow).
		Find(&b15).Error; err != nil {
		return nil, nil, err
	}

	return b35, b15, nil
}

// GetAllRecords retrieves all records for a user with pagination and sorting
func (r *RecordRepository) GetAllRecords(username string, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Where("username = ?", username).
		Joins("Chart").
		Joins("Chart.Song")

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	// Use whitelist validation to prevent SQL injection
	safeSortBy := validateSortBy(sortBy)
	query = query.Order(safeSortBy + " " + orderStr)

	// pageIndex is 0-indexed from the service layer
	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// GetBestRecords retrieves the best records for a user with pagination and sorting
func (r *RecordRepository) GetBestRecords(username string, pageSize, pageIndex int, sortBy string, order bool) ([]model.PlayRecord, error) {
	var records []model.PlayRecord
	query := r.db.Model(&model.PlayRecord{}).
		Joins("JOIN best_play_records ON best_play_records.play_record_id = play_records.play_record_id").
		Joins("Chart").
		Joins("Chart.Song").
		Where("play_records.username = ?", username)

	orderStr := "desc"
	if !order {
		orderStr = "asc"
	}

	// Use whitelist validation to prevent SQL injection
	safeSortBy := validateSortBy(sortBy)
	query = query.Order(safeSortBy + " " + orderStr)

	// pageIndex is 0-indexed from the service layer
	err := query.Offset(pageSize * pageIndex).Limit(pageSize).Find(&records).Error
	return records, err
}

// GetAllChartsWithBestScores retrieves all charts with the user's best score (if any)
func (r *RecordRepository) GetAllChartsWithBestScores(username string) ([]model.ChartWithScore, error) {
	var results []model.ChartWithScore

	err := r.db.Table("charts").
		Select("charts.chart_id, songs.title, songs.version, charts.difficulty, charts.level, COALESCE(play_records.score, 0) as score").
		Joins("JOIN songs ON charts.song_id = songs.song_id").
		Joins("LEFT JOIN play_records ON charts.chart_id = play_records.chart_id AND play_records.username = ?", username).
		Joins("LEFT JOIN best_play_records ON play_records.play_record_id = best_play_records.play_record_id").
		Where("play_records.play_record_id IS NULL OR best_play_records.play_record_id IS NOT NULL").
		Scan(&results).Error

	return results, err
}

// CountBestRecords counts the number of best records for a user
func (r *RecordRepository) CountBestRecords(username string) (int64, error) {
	var count int64
	err := r.db.Model(&model.BestPlayRecord{}).
		Joins("JOIN play_records ON best_play_records.play_record_id = play_records.play_record_id").
		Where("play_records.username = ?", username).
		Count(&count).Error
	return count, err
}

// CountAllRecords counts the total number of records for a user
func (r *RecordRepository) CountAllRecords(username string) (int64, error) {
	var count int64
	err := r.db.Model(&model.PlayRecord{}).
		Where("username = ?", username).
		Count(&count).Error
	return count, err
}
